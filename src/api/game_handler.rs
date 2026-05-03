use actix_web::{get, post, web, HttpRequest, HttpResponse, Responder};
use actix_ws::{Message, Session};
use crate::models::game::{GameState, Player};
use crate::models::multiplayer_game::{GameServer, Room, PlayerSession};
use handlebars::Handlebars;
use serde_json::json;
use std::sync::Mutex;
use futures_util::StreamExt;
use uuid::Uuid;

pub struct GameData {
    pub state: Mutex<GameState>,
    pub server: web::Data<GameServer>,
}

// --- HOTSEAT HANDLERS (Local) ---

#[get("/tictactoe/hotseat")]
pub async fn hotseat_index(hb: web::Data<Handlebars<'_>>, data: web::Data<GameData>) -> impl Responder {
    let state = data.state.lock().unwrap();
    let body = hb.render("tictactoe", &json!({
        "board": state.board,
        "turn": state.current_turn.symbol(),
        "winner": format_winner(&state.winner),
        "is_online": false
    })).unwrap();
    
    let html = hb.render("base", &json!({
        "title": "Tic Tac Toe Hotseat",
        "body": body
    })).unwrap();
    
    HttpResponse::Ok().body(html)
}

// --- ONLINE HANDLERS (WebSocket) ---

#[get("/tictactoe/online")]
pub async fn online_index(hb: web::Data<Handlebars<'_>>) -> impl Responder {
    let body = hb.render("tictactoe_online_setup", &json!({})).unwrap();
    let html = hb.render("base", &json!({
        "title": "Tic Tac Toe Online",
        "body": body
    })).unwrap();
    HttpResponse::Ok().body(html)
}

#[get("/tictactoe/ws/{room_id}")]
pub async fn game_ws(
    req: HttpRequest,
    stream: web::Payload,
    path: web::Path<String>,
    server: web::Data<GameServer>,
    hb: web::Data<Handlebars<'static>>,
) -> Result<HttpResponse, actix_web::Error> {
    let room_id = path.into_inner();
    let player_id = Uuid::new_v4().to_string();
    log::debug!("[WS] New connection request: room={} player_id={}", room_id, player_id);

    let (res, session, mut msg_stream) = actix_ws::handle(&req, stream)?;
    log::debug!("[WS] Handshake complete: room={} player_id={}", room_id, player_id);

    let session_clone = session.clone();
    let server_clone = server.clone();
    let hb_clone = hb.clone();
    let room_id_clone = room_id.clone();
    let player_id_clone = player_id.clone();

    actix_web::rt::spawn(async move {
        // --- JOIN ROOM ---
        log::debug!("[WS] Spawned task: acquiring lock to join room={} player_id={}", room_id_clone, player_id_clone);

        let player_type = {
            let mut rooms = server_clone.rooms.lock().unwrap();
            log::debug!("[WS] Lock acquired for room={} player_id={}", room_id_clone, player_id_clone);

            let room = rooms.entry(room_id_clone.clone()).or_insert_with(|| {
                log::debug!("[WS] Creating new room={}", room_id_clone);
                Room {
                    id: room_id_clone.clone(),
                    state: GameState::default(),
                    players: Vec::new(),
                }
            });

            log::debug!("[WS] Room={} currently has {} player(s)", room_id_clone, room.players.len());

            if room.players.len() >= 2 {
                log::warn!("[WS] Room={} is full! Rejecting player_id={}", room_id_clone, player_id_clone);
                drop(rooms);
                let mut s = session_clone.clone();
                let _ = s.text("Room Full").await;
                return;
            }

            let p_type = if room.players.is_empty() { Player::X } else { Player::O };
            log::debug!("[WS] Assigning player_id={} as {:?} in room={}", player_id_clone, p_type.symbol(), room_id_clone);

            room.players.push(PlayerSession {
                id: player_id_clone.clone(),
                player_type: p_type,
                ws_session: session_clone.clone(),
            });

            if room.players.len() == 2 {
                use rand::Rng;
                let mut rng = rand::thread_rng();
                room.state.current_turn = if rng.gen_bool(0.5) { Player::X } else { Player::O };
                log::debug!("[WS] Second player joined. Randomly selected starting turn: {:?}", room.state.current_turn.symbol());
            }

            log::debug!("[WS] Room={} now has {} player(s). Releasing lock.", room_id_clone, room.players.len());
            p_type
            // lock drops here
        };

        log::debug!("[WS] Lock released. Sending initial state to room={}", room_id_clone);

        // --- SEND INITIAL STATE (lock released, safe to await) ---
        {
            let outgoing = collect_outgoing(&server_clone, &room_id_clone, &hb_clone);
            log::debug!("[WS] Initial broadcast: {} message(s) queued for room={}", outgoing.len(), room_id_clone);
            send_all(outgoing).await;
            log::debug!("[WS] Initial broadcast sent for room={}", room_id_clone);
        }

        // --- HEARTBEAT SETUP ---
        let mut interval = tokio::time::interval(std::time::Duration::from_secs(20));
        log::debug!("[WS] Entering message loop with heartbeat: room={} player={:?}", room_id_clone, player_type.symbol());

        loop {
            tokio::select! {
                _ = interval.tick() => {
                    log::debug!("[WS] Sending heartbeat ping: room={} player_id={}", room_id_clone, player_id_clone);
                    let mut s = session_clone.clone();
                    if let Err(e) = s.ping(b"").await {
                        log::warn!("[WS] Heartbeat ping failed: {:?}. room={} player_id={}", e, room_id_clone, player_id_clone);
                        break;
                    }
                }
                msg = msg_stream.next() => {
                    let msg = match msg {
                        Some(Ok(m)) => m,
                        Some(Err(e)) => {
                            log::warn!("[WS] Stream error: {:?}. room={} player_id={}", e, room_id_clone, player_id_clone);
                            break;
                        }
                        None => {
                            log::debug!("[WS] Stream ended. room={} player_id={}", room_id_clone, player_id_clone);
                            break;
                        }
                    };

                    match msg {
                        Message::Text(text) => {
                            log::debug!("[WS] Message received in room={} from {:?}: {}", room_id_clone, player_type.symbol(), text);
                            if let Ok(data) = serde_json::from_str::<serde_json::Value>(&text) {
                                if let (Some(r), Some(c)) = (
                                    data.get("row").and_then(|v| v.as_u64()),
                                    data.get("col").and_then(|v| v.as_u64()),
                                ) {
                                    log::debug!("[WS] Move attempt: room={} player={:?} row={} col={}", room_id_clone, player_type.symbol(), r, c);

                                    let moved = {
                                        let mut rooms = server_clone.rooms.lock().unwrap();
                                        if let Some(room) = rooms.get_mut(&room_id_clone) {
                                            let player_count = room.players.len();
                                            let current_turn = room.state.current_turn.symbol();
                                            log::debug!("[WS] Move check: room={} players={} current_turn={} mover={:?}", room_id_clone, player_count, current_turn, player_type.symbol());
                                            if player_count == 2 && room.state.current_turn == player_type {
                                                let result = room.state.make_move(r as usize, c as usize);
                                                log::debug!("[WS] make_move result={} for room={}", result, room_id_clone);
                                                result
                                            } else {
                                                log::warn!("[WS] Move rejected: not player's turn or room not full. room={}", room_id_clone);
                                                false
                                            }
                                        } else {
                                            log::warn!("[WS] Move rejected: room={} not found", room_id_clone);
                                            false
                                        }
                                        // lock drops here
                                    };

                                    if moved {
                                        let outgoing = collect_outgoing(&server_clone, &room_id_clone, &hb_clone);
                                        log::debug!("[WS] Post-move broadcast: {} message(s) for room={}", outgoing.len(), room_id_clone);
                                        send_all(outgoing).await;
                                    }
                                } else if let Some(action) = data.get("action").and_then(|v| v.as_str()) {
                                    if action == "restart" {
                                        log::debug!("[WS] Restart request: room={}", room_id_clone);
                                        {
                                            let mut rooms = server_clone.rooms.lock().unwrap();
                                            if let Some(room) = rooms.get_mut(&room_id_clone) {
                                                room.state = GameState::default();
                                                use rand::Rng;
                                                let mut rng = rand::thread_rng();
                                                room.state.current_turn = if rng.gen_bool(0.5) { Player::X } else { Player::O };
                                                log::debug!("[WS] Room={} state reset with random turn: {:?}.", room_id_clone, room.state.current_turn.symbol());
                                            }
                                        }
                                        let outgoing = collect_outgoing(&server_clone, &room_id_clone, &hb_clone);
                                        send_all(outgoing).await;
                                    }
                                } else {
                                    log::warn!("[WS] Could not parse action or row/col from message in room={}: {}", room_id_clone, text);
                                }
                            } else {
                                log::warn!("[WS] Non-JSON message in room={}: {}", room_id_clone, text);
                            }
                        }
                        Message::Close(reason) => {
                            log::debug!("[WS] Close message received: room={} player={:?} reason={:?}", room_id_clone, player_type.symbol(), reason);
                            break;
                        }
                        Message::Ping(bytes) => {
                            let mut s = session_clone.clone();
                            let _ = s.pong(&bytes).await;
                        }
                        Message::Pong(_) => {
                            log::debug!("[WS] Pong received: room={}", room_id_clone);
                        }
                        _ => (),
                    }
                }
            }
        }

        // --- CLEANUP ---
        log::debug!("[WS] Cleaning up: removing player_id={} from room={}", player_id_clone, room_id_clone);
        {
            let mut rooms = server_clone.rooms.lock().unwrap();
            if let Some(room) = rooms.get_mut(&room_id_clone) {
                // If any player leaves, we end the game for everyone in this room.
                // We'll collect remaining sessions to close them outside the lock.
                let mut remaining_sessions = Vec::new();
                for p in &room.players {
                    if p.id != player_id_clone {
                        remaining_sessions.push(p.ws_session.clone());
                    }
                }
                
                log::debug!("[WS] Room={} ending because a player left. Disconnecting {} remaining player(s).", room_id_clone, remaining_sessions.len());
                
                // Remove the room entirely
                rooms.remove(&room_id_clone);
                
                // Close remaining sessions
                for session in remaining_sessions {
                    let _ = session.clone().close(None).await;
                }
            }
        }
    });

    Ok(res)
}

/// Collect rendered HTML + sessions while holding the lock. Returns Vec of (Session, html).
fn collect_outgoing(
    server: &GameServer,
    room_id: &str,
    hb: &Handlebars<'static>,
) -> Vec<(Session, String)> {
    let mut rooms = server.rooms.lock().unwrap();
    let room = match rooms.get_mut(room_id) {
        Some(r) => r,
        None => {
            log::warn!("[WS] collect_outgoing: room={} not found", room_id);
            return vec![];
        }
    };

    let mut cells = Vec::new();
    for r in 0..3 {
        for c in 0..3 {
            let val = room.state.board[r][c];
            cells.push(json!({
                "row": r,
                "col": c,
                "value": val.map(|p| p.symbol().to_string()).unwrap_or_default(),
                "is_occupied": val.is_some(),
                "symbol": val.map(|p| p.symbol().to_lowercase()).unwrap_or_default()
            }));
        }
    }

    let waiting_for_opponent = room.players.len() < 2;
    log::debug!("[WS] collect_outgoing: room={} players={} waiting={}", room_id, room.players.len(), waiting_for_opponent);

    let mut outgoing = Vec::new();
    for player in &room.players {
        let is_your_turn = !waiting_for_opponent
            && room.state.current_turn == player.player_type
            && room.state.winner.is_none();

        let body = hb.render("tictactoe_board_online", &json!({
            "cells": cells,
            "turn": room.state.current_turn.symbol(),
            "winner": format_winner(&room.state.winner),
            "you_are": player.player_type.symbol(),
            "is_your_turn": is_your_turn,
            "waiting_for_opponent": waiting_for_opponent,
            "room_id": room.id
        })).unwrap_or_default();

        log::debug!("[WS] Queuing message for player={:?} is_your_turn={} waiting={}", player.player_type.symbol(), is_your_turn, waiting_for_opponent);
        outgoing.push((player.ws_session.clone(), body));
    }
    outgoing
    // lock drops here
}

/// Send all outgoing messages asynchronously AFTER lock has been released.
async fn send_all(outgoing: Vec<(Session, String)>) {
    for (mut session, body) in outgoing {
        log::debug!("[WS] Sending {} bytes via ws", body.len());
        match session.text(body).await {
            Ok(_) => log::debug!("[WS] Send OK"),
            Err(e) => log::warn!("[WS] Send error: {:?}", e),
        }
    }
}

// --- UTILS ---

#[post("/tictactoe/move/{row}/{col}")]
pub async fn make_move(
    hb: web::Data<Handlebars<'_>>,
    data: web::Data<GameData>,
    path: web::Path<(usize, usize)>,
) -> impl Responder {
    let (row, col) = path.into_inner();
    let mut state = data.state.lock().unwrap();
    state.make_move(row, col);
    let body = hb.render("tictactoe_board", &json!({
        "board": state.board,
        "turn": state.current_turn.symbol(),
        "winner": format_winner(&state.winner)
    })).unwrap();
    HttpResponse::Ok().body(body)
}

#[post("/tictactoe/reset")]
pub async fn reset_game(
    hb: web::Data<Handlebars<'_>>,
    data: web::Data<GameData>,
) -> impl Responder {
    let mut state = data.state.lock().unwrap();
    *state = GameState::default();
    let body = hb.render("tictactoe_board", &json!({
        "board": state.board,
        "turn": state.current_turn.symbol(),
        "winner": format_winner(&state.winner)
    })).unwrap();
    HttpResponse::Ok().body(body)
}

fn format_winner(winner: &Option<Option<crate::models::game::Player>>) -> Option<String> {
    match winner {
        Some(Some(p)) => Some(format!("Player {} Wins!", p.symbol())),
        Some(None) => Some("It's a Draw!".to_string()),
        None => None,
    }
}
