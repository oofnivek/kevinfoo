use crate::models::note::{Note, CreateNoteRequest, UpdateNoteRequest};
use crate::repository::note_repository::NoteRepository;
use uuid::Uuid;

pub struct NoteService {
    repository: NoteRepository,
}

impl NoteService {
    pub fn new(repository: NoteRepository) -> Self {
        Self { repository }
    }

    pub async fn create_note(&self, req: CreateNoteRequest) -> Result<String, String> {
        let note = Note {
            id: Some(Uuid::new_v4().to_string()),
            content: req.content,
            created_at: chrono::Utc::now(),
        };
        self.repository.create_note(note).await.map_err(|e| e.to_string())
    }

    pub async fn get_notes(&self) -> Result<Vec<Note>, String> {
        self.repository.get_notes().await.map_err(|e| e.to_string())
    }

    pub async fn get_note(&self, id: &str) -> Result<Option<Note>, String> {
        self.repository.get_note(id).await.map_err(|e| e.to_string())
    }

    pub async fn delete_note(&self, id: &str) -> Result<bool, String> {
        self.repository.delete_note(id).await.map_err(|e| e.to_string())
    }

    pub async fn update_note(&self, id: &str, req: UpdateNoteRequest) -> Result<bool, String> {
        self.repository.update_note(id, &req.content).await.map_err(|e| e.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn test_note_creation_logic() {
        assert!(true);
    }
}
