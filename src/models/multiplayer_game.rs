use std::collections::HashMap;
use std::sync::Mutex;
use actix_ws::Session;
use crate::models::game::{GameState, Player};

#[derive(Clone)]
pub struct PlayerSession {
    pub id: String,
    pub player_type: Player,
    pub ws_session: Session,
}

pub struct Room {
    pub id: String,
    pub state: GameState,
    pub players: Vec<PlayerSession>,
}

pub struct GameServer {
    pub rooms: Mutex<HashMap<String, Room>>,
}

impl GameServer {
    pub fn new() -> Self {
        Self {
            rooms: Mutex::new(HashMap::new()),
        }
    }
}
