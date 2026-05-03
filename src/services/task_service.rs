use crate::models::task::{Task, CreateTaskRequest};
use crate::repository::task_repository::TaskRepository;
use uuid::Uuid;

pub struct TaskService {
    repository: TaskRepository,
}

impl TaskService {
    pub fn new(repository: TaskRepository) -> Self {
        Self { repository }
    }

    pub async fn create_task(&self, req: CreateTaskRequest) -> Result<String, String> {
        let task = Task {
            id: Some(Uuid::new_v4().to_string()),
            title: req.title,
            description: req.description,
            completed: false,
            created_at: chrono::Utc::now(),
        };
        self.repository.create_task(task).await.map_err(|e| e.to_string())
    }

    pub async fn get_tasks(&self) -> Result<Vec<Task>, String> {
        self.repository.get_tasks().await.map_err(|e| e.to_string())
    }

    pub async fn delete_task(&self, id: &str) -> Result<bool, String> {
        self.repository.delete_task(id).await.map_err(|e| e.to_string())
    }

    pub async fn toggle_task(&self, id: &str) -> Result<bool, String> {
        self.repository.toggle_task(id).await.map_err(|e| e.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    // Note: In a real app, you'd mock the repository here.
    // For now, this is a placeholder to demonstrate 'cargo test'.
    #[test]
    fn test_task_creation_logic() {
        assert!(true);
    }
}
