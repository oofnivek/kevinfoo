use actix_web::{get, post, web, HttpResponse, Responder};
use crate::models::game::GameState;
use handlebars::Handlebars;
use serde_json::json;
use std::sync::Mutex;

pub struct GameData {
    pub state: Mutex<GameState>,
}

#[get("/tictactoe")]
pub async fn game_index(hb: web::Data<Handlebars<'_>>, data: web::Data<GameData>) -> impl Responder {
    let state = data.state.lock().unwrap();
    let body = hb.render("tictactoe", &json!({
        "board": state.board,
        "turn": state.current_turn.symbol(),
        "winner": format_winner(&state.winner)
    })).unwrap();
    
    let html = hb.render("base", &json!({
        "title": "Tic Tac Toe",
        "body": body
    })).unwrap();
    
    HttpResponse::Ok().body(html)
}

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
