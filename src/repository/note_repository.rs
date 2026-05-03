use mongodb::{
    bson::doc,
    Collection, Database,
};
use crate::models::note::Note;
use futures_util::stream::StreamExt;

pub struct NoteRepository {
    collection: Collection<Note>,
}

impl NoteRepository {
    pub fn new(db: &Database) -> Self {
        let collection = db.collection::<Note>("notes");
        Self { collection }
    }

    pub async fn create_note(&self, note: Note) -> Result<String, mongodb::error::Error> {
        let result = self.collection.insert_one(note, None).await?;
        Ok(result.inserted_id.to_string())
    }

    pub async fn get_notes(&self) -> Result<Vec<Note>, mongodb::error::Error> {
        let mut cursor = self.collection.find(None, None).await?;
        let mut notes = Vec::new();
        while let Some(note) = cursor.next().await {
            notes.push(note?);
        }
        Ok(notes)
    }

    pub async fn get_note(&self, id: &str) -> Result<Option<Note>, mongodb::error::Error> {
        let query = doc! { "_id": id };
        self.collection.find_one(query, None).await
    }

    pub async fn delete_note(&self, id: &str) -> Result<bool, mongodb::error::Error> {
        let query = doc! { "_id": id };
        let result = self.collection.delete_one(query, None).await?;
        Ok(result.deleted_count > 0)
    }

    pub async fn update_note(&self, id: &str, content: &str) -> Result<bool, mongodb::error::Error> {
        let query = doc! { "_id": id };
        let update = doc! { "$set": { "content": content } };
        let result = self.collection.update_one(query, update, None).await?;
        Ok(result.modified_count > 0 || result.matched_count > 0)
    }
}
