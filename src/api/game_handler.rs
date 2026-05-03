use actix_web::{get, post, web, HttpRequest, HttpResponse, Responder};
use actix_ws::Message;
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
    let (res, session, mut msg_stream) = actix_ws::handle(&req, stream)?;
    let player_id = Uuid::new_v4().to_string();
    
    let mut session_clone = session.clone();
    let server_clone = server.clone();
    let hb_clone = hb.clone();
    let room_id_clone = room_id.clone();
    let player_id_clone = player_id.clone();

    actix_web::rt::spawn(async move {
        let player_type = {
            let mut rooms = server_clone.rooms.lock().unwrap();
            let room = rooms.entry(room_id_clone.clone()).or_insert_with(|| Room {
                id: room_id_clone.clone(),
                state: GameState::default(),
                players: Vec::new(),
            });

            if room.players.len() >= 2 {
                let _ = session_clone.text("Room Full").await;
                return;
            }

            let p_type = if room.players.is_empty() { Player::X } else { Player::O };
            room.players.push(PlayerSession {
                id: player_id_clone.clone(),
                player_type: p_type,
                ws_session: session_clone.clone(),
            });
            
            broadcast_update(room, &hb_clone).await;
            p_type
        };

        while let Some(Ok(msg)) = msg_stream.next().await {
            match msg {
                Message::Text(text) => {
                    if let Ok(data) = serde_json::from_str::<serde_json::Value>(&text) {
                        if let (Some(r), Some(c)) = (data.get("row").and_then(|v| v.as_u64()), data.get("col").and_then(|v| v.as_u64())) {
                            let mut rooms = server_clone.rooms.lock().unwrap();
                            if let Some(room) = rooms.get_mut(&room_id_clone) {
                                if room.state.current_turn == player_type {
                                    if room.state.make_move(r as usize, c as usize) {
                                        broadcast_update(room, &hb_clone).await;
                                    }
                                }
                            }
                        }
                    }
                }
                Message::Close(_) => break,
                _ => (),
            }
        }

        let mut rooms = server_clone.rooms.lock().unwrap();
        if let Some(room) = rooms.get_mut(&room_id_clone) {
            room.players.retain(|p| p.id != player_id_clone);
            if room.players.is_empty() {
                rooms.remove(&room_id_clone);
            }
        }
    });

    Ok(res)
}

async fn broadcast_update(room: &mut Room, hb: &Handlebars<'static>) {
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

    for player in &mut room.players {
        let body = hb.render("tictactoe_board_online", &json!({
            "cells": cells,
            "turn": room.state.current_turn.symbol(),
            "winner": format_winner(&room.state.winner),
            "you_are": player.player_type.symbol(),
            "is_your_turn": room.state.current_turn == player.player_type && room.state.winner.is_none(),
            "room_id": room.id
        })).unwrap();
        
        let _ = player.ws_session.text(body).await;
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
