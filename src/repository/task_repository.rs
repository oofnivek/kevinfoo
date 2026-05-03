use mongodb::{
    bson::doc,
    Collection, Database,
};
use crate::models::task::Task;
use futures_util::stream::StreamExt;

pub struct TaskRepository {
    collection: Collection<Task>,
}

impl TaskRepository {
    pub fn new(db: &Database) -> Self {
        let collection = db.collection::<Task>("tasks");
        Self { collection }
    }

    pub async fn create_task(&self, task: Task) -> Result<String, mongodb::error::Error> {
        let result = self.collection.insert_one(task, None).await?;
        Ok(result.inserted_id.to_string())
    }

    pub async fn get_tasks(&self) -> Result<Vec<Task>, mongodb::error::Error> {
        let mut cursor = self.collection.find(None, None).await?;
        let mut tasks = Vec::new();
        while let Some(task) = cursor.next().await {
            tasks.push(task?);
        }
        Ok(tasks)
    }

    pub async fn delete_task(&self, id: &str) -> Result<bool, mongodb::error::Error> {
        // Handle potential parsing error or direct string ID
        let query = doc! { "_id": id };
        let result = self.collection.delete_one(query, None).await?;
        Ok(result.deleted_count > 0)
    }

    pub async fn toggle_task(&self, id: &str) -> Result<bool, mongodb::error::Error> {
        let query = doc! { "_id": id };
        // This is a bit simplified, in a real app you'd fetch first or use a more complex update
        let task = self.collection.find_one(query.clone(), None).await?;
        if let Some(t) = task {
            let update = doc! { "$set": { "completed": !t.completed } };
            self.collection.update_one(query, update, None).await?;
            return Ok(true);
        }
        Ok(false)
    }
}
