use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone, Copy, PartialEq)]
pub enum Player {
    X,
    O,
}

impl Player {
    pub fn symbol(&self) -> &str {
        match self {
            Player::X => "X",
            Player::O => "O",
        }
    }
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct GameState {
    pub board: [[Option<Player>; 3]; 3],
    pub current_turn: Player,
    pub winner: Option<Option<Player>>, // Some(Some(P)) = Winner, Some(None) = Draw, None = In Progress
}

impl Default for GameState {
    fn default() -> Self {
        Self {
            board: [[None; 3]; 3],
            current_turn: Player::X,
            winner: None,
        }
    }
}

impl GameState {
    pub fn make_move(&mut self, row: usize, col: usize) -> bool {
        if self.winner.is_some() || self.board[row][col].is_some() {
            return false;
        }

        self.board[row][col] = Some(self.current_turn);
        self.check_winner();

        if self.winner.is_none() {
            self.current_turn = match self.current_turn {
                Player::X => Player::O,
                Player::O => Player::X,
            };
        }
        true
    }

    fn check_winner(&mut self) {
        // Rows
        for row in 0..3 {
            if self.board[row][0].is_some() && self.board[row][0] == self.board[row][1] && self.board[row][1] == self.board[row][2] {
                self.winner = Some(self.board[row][0]);
                return;
            }
        }
        // Cols
        for col in 0..3 {
            if self.board[0][col].is_some() && self.board[0][col] == self.board[1][col] && self.board[1][col] == self.board[2][col] {
                self.winner = Some(self.board[0][col]);
                return;
            }
        }
        // Diagonals
        if self.board[0][0].is_some() && self.board[0][0] == self.board[1][1] && self.board[1][1] == self.board[2][2] {
            self.winner = Some(self.board[0][0]);
            return;
        }
        if self.board[0][2].is_some() && self.board[0][2] == self.board[1][1] && self.board[1][1] == self.board[2][0] {
            self.winner = Some(self.board[0][2]);
            return;
        }

        // Check Draw
        let mut full = true;
        for r in 0..3 {
            for c in 0..3 {
                if self.board[r][c].is_none() {
                    full = false;
                }
            }
        }
        if full {
            self.winner = Some(None);
        }
    }
}
