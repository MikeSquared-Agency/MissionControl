use serde::{Deserialize, Serialize};
use crate::phase::Phase;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum TaskStatus {
    Pending,
    Ready,
    InProgress,
    Blocked(String),
    Done,
}

impl TaskStatus {
    pub fn as_str(&self) -> &str {
        match self {
            TaskStatus::Pending => "pending",
            TaskStatus::Ready => "ready",
            TaskStatus::InProgress => "in_progress",
            TaskStatus::Blocked(_) => "blocked",
            TaskStatus::Done => "done",
        }
    }
}

impl Default for TaskStatus {
    fn default() -> Self {
        TaskStatus::Pending
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Task {
    pub id: String,
    pub name: String,
    pub phase: Phase,
    pub zone: String,
    pub status: TaskStatus,
    pub persona: String,
    pub dependencies: Vec<String>,
    pub created_at: u64,
    pub updated_at: u64,
}

impl Task {
    pub fn new(
        id: impl Into<String>,
        name: impl Into<String>,
        phase: Phase,
        zone: impl Into<String>,
        persona: impl Into<String>,
    ) -> Self {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            id: id.into(),
            name: name.into(),
            phase,
            zone: zone.into(),
            status: TaskStatus::Pending,
            persona: persona.into(),
            dependencies: Vec::new(),
            created_at: now,
            updated_at: now,
        }
    }

    pub fn with_dependencies(mut self, deps: Vec<String>) -> Self {
        self.dependencies = deps;
        self
    }

    pub fn is_blocked(&self) -> bool {
        matches!(self.status, TaskStatus::Blocked(_))
    }

    pub fn is_done(&self) -> bool {
        matches!(self.status, TaskStatus::Done)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_task_creation() {
        let task = Task::new("task-1", "Build login", Phase::Implement, "frontend", "developer");
        assert_eq!(task.id, "task-1");
        assert_eq!(task.name, "Build login");
        assert_eq!(task.phase, Phase::Implement);
        assert_eq!(task.zone, "frontend");
        assert_eq!(task.persona, "developer");
        assert_eq!(task.status, TaskStatus::Pending);
        assert!(task.dependencies.is_empty());
    }

    #[test]
    fn test_task_with_dependencies() {
        let task = Task::new("task-2", "Build auth", Phase::Implement, "backend", "developer")
            .with_dependencies(vec!["task-1".to_string()]);
        assert_eq!(task.dependencies.len(), 1);
        assert_eq!(task.dependencies[0], "task-1");
    }

    #[test]
    fn test_task_status_serialization() {
        let status = TaskStatus::Blocked("Waiting for API".to_string());
        let json = serde_json::to_string(&status).unwrap();
        assert!(json.contains("blocked"));
        assert!(json.contains("Waiting for API"));
    }
}
