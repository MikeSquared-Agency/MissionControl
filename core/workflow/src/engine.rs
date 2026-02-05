use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::stage::Stage;
use crate::task::{Task, TaskStatus};
use crate::gate::{Gate, GateStatus};

#[derive(Debug, Error)]
pub enum WorkflowError {
    #[error("Task not found: {0}")]
    TaskNotFound(String),

    #[error("Gate not found for stage: {0:?}")]
    GateNotFound(Stage),

    #[error("Cannot transition from {from:?} to {to:?}")]
    InvalidTransition { from: Stage, to: Stage },

    #[error("Gate not open for stage: {0:?}")]
    GateNotOpen(Stage),

    #[error("Serialization error: {0}")]
    SerializationError(String),

    #[error("Invalid task status transition")]
    InvalidStatusTransition,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowEngine {
    current_stage: Stage,
    tasks: HashMap<String, Task>,
    gates: HashMap<String, Gate>,
}

impl WorkflowEngine {
    pub fn new() -> Self {
        let mut gates = HashMap::new();
        for stage in Stage::all() {
            let gate = Gate::new(*stage);
            gates.insert(gate.id.clone(), gate);
        }

        Self {
            current_stage: Stage::Discovery,
            tasks: HashMap::new(),
            gates,
        }
    }

    // Stage management
    pub fn current_stage(&self) -> Stage {
        self.current_stage
    }

    pub fn can_transition(&self, to: Stage) -> bool {
        // Can only transition to the next stage
        if let Some(next) = self.current_stage.next() {
            if next == to {
                // Check if current stage's gate is open
                return self.check_gate(self.current_stage) == GateStatus::Open;
            }
        }
        false
    }

    pub fn transition(&mut self, to: Stage) -> Result<(), WorkflowError> {
        if !self.can_transition(to) {
            if self.check_gate(self.current_stage) != GateStatus::Open {
                return Err(WorkflowError::GateNotOpen(self.current_stage));
            }
            return Err(WorkflowError::InvalidTransition {
                from: self.current_stage,
                to,
            });
        }

        self.current_stage = to;
        Ok(())
    }

    // Task management
    pub fn create_task(&mut self, task: Task) -> String {
        let id = task.id.clone();
        self.tasks.insert(id.clone(), task);
        id
    }

    pub fn update_task_status(&mut self, id: &str, status: TaskStatus) -> Result<(), WorkflowError> {
        let task = self.tasks.get_mut(id)
            .ok_or_else(|| WorkflowError::TaskNotFound(id.to_string()))?;

        task.status = status;
        task.updated_at = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Ok(())
    }

    pub fn get_task(&self, id: &str) -> Option<&Task> {
        self.tasks.get(id)
    }

    pub fn get_ready_tasks(&self) -> Vec<&Task> {
        self.tasks.values()
            .filter(|task| {
                // Task must be in pending status and all dependencies done
                if task.status != TaskStatus::Pending {
                    return false;
                }

                // Check all dependencies are done
                task.dependencies.iter().all(|dep_id| {
                    self.tasks.get(dep_id)
                        .map(|dep| dep.is_done())
                        .unwrap_or(false)
                })
            })
            .collect()
    }

    pub fn get_tasks_for_stage(&self, stage: Stage) -> Vec<&Task> {
        self.tasks.values()
            .filter(|task| task.stage == stage)
            .collect()
    }

    pub fn all_tasks(&self) -> Vec<&Task> {
        self.tasks.values().collect()
    }

    // Gate management
    pub fn get_gate(&self, stage: Stage) -> Option<&Gate> {
        let id = format!("gate-{}", stage.as_str());
        self.gates.get(&id)
    }

    pub fn get_gate_mut(&mut self, stage: Stage) -> Option<&mut Gate> {
        let id = format!("gate-{}", stage.as_str());
        self.gates.get_mut(&id)
    }

    pub fn check_gate(&self, stage: Stage) -> GateStatus {
        self.get_gate(stage)
            .map(|g| g.status.clone())
            .unwrap_or(GateStatus::Closed)
    }

    pub fn approve_gate(&mut self, stage: Stage, by: &str) -> Result<(), WorkflowError> {
        let gate = self.get_gate_mut(stage)
            .ok_or(WorkflowError::GateNotFound(stage))?;

        gate.approve(by);
        Ok(())
    }

    // Serialization
    pub fn to_json(&self) -> String {
        serde_json::to_string(self).unwrap_or_default()
    }

    pub fn from_json(json: &str) -> Result<Self, WorkflowError> {
        serde_json::from_str(json)
            .map_err(|e| WorkflowError::SerializationError(e.to_string()))
    }
}

impl Default for WorkflowEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_engine_creation() {
        let engine = WorkflowEngine::new();
        assert_eq!(engine.current_stage(), Stage::Discovery);
        assert_eq!(engine.gates.len(), 10);
        assert!(engine.get_gate(Stage::Discovery).is_some());
        assert!(engine.get_gate(Stage::Goal).is_some());
        assert!(engine.get_gate(Stage::Requirements).is_some());
        assert!(engine.get_gate(Stage::Planning).is_some());
        assert!(engine.get_gate(Stage::Design).is_some());
        assert!(engine.get_gate(Stage::Implement).is_some());
        assert!(engine.get_gate(Stage::Verify).is_some());
        assert!(engine.get_gate(Stage::Validate).is_some());
        assert!(engine.get_gate(Stage::Document).is_some());
        assert!(engine.get_gate(Stage::Release).is_some());
    }

    #[test]
    fn test_task_creation_and_retrieval() {
        let mut engine = WorkflowEngine::new();
        let task = Task::new("task-1", "Test task", Stage::Discovery, "system", "researcher");

        let id = engine.create_task(task);
        assert_eq!(id, "task-1");

        let retrieved = engine.get_task("task-1");
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().name, "Test task");
    }

    #[test]
    fn test_ready_tasks_with_dependencies() {
        let mut engine = WorkflowEngine::new();

        // Create task with no dependencies
        let task1 = Task::new("task-1", "First", Stage::Implement, "backend", "developer");
        engine.create_task(task1);

        // Create task with dependency on task-1
        let task2 = Task::new("task-2", "Second", Stage::Implement, "backend", "developer")
            .with_dependencies(vec!["task-1".to_string()]);
        engine.create_task(task2);

        // Initially, only task-1 should be ready (pending with no deps)
        let ready = engine.get_ready_tasks();
        assert_eq!(ready.len(), 1);
        assert_eq!(ready[0].id, "task-1");

        // Complete task-1
        engine.update_task_status("task-1", TaskStatus::Done).unwrap();

        // Now task-2 should be ready
        let ready = engine.get_ready_tasks();
        assert_eq!(ready.len(), 1);
        assert_eq!(ready[0].id, "task-2");
    }

    #[test]
    fn test_stage_transition() {
        let mut engine = WorkflowEngine::new();

        // Cannot transition without gate approval
        assert!(!engine.can_transition(Stage::Goal));

        // Satisfy gate criteria and approve
        if let Some(gate) = engine.get_gate_mut(Stage::Discovery) {
            for i in 0..gate.criteria.len() {
                gate.satisfy_criterion(i);
            }
            gate.approve("user");
        }

        // Now can transition
        assert!(engine.can_transition(Stage::Goal));
        engine.transition(Stage::Goal).unwrap();
        assert_eq!(engine.current_stage(), Stage::Goal);
    }

    #[test]
    fn test_serialization() {
        let mut engine = WorkflowEngine::new();
        let task = Task::new("task-1", "Test", Stage::Discovery, "system", "researcher");
        engine.create_task(task);

        let json = engine.to_json();
        let restored = WorkflowEngine::from_json(&json).unwrap();

        assert_eq!(restored.current_stage(), Stage::Discovery);
        assert!(restored.get_task("task-1").is_some());
    }

    #[test]
    fn test_get_tasks_for_stage() {
        let mut engine = WorkflowEngine::new();
        let task1 = Task::new("task-1", "Research", Stage::Discovery, "system", "researcher");
        let task2 = Task::new("task-2", "Build", Stage::Implement, "backend", "developer");
        engine.create_task(task1);
        engine.create_task(task2);

        let discovery_tasks = engine.get_tasks_for_stage(Stage::Discovery);
        assert_eq!(discovery_tasks.len(), 1);
        assert_eq!(discovery_tasks[0].id, "task-1");

        let implement_tasks = engine.get_tasks_for_stage(Stage::Implement);
        assert_eq!(implement_tasks.len(), 1);
        assert_eq!(implement_tasks[0].id, "task-2");
    }
}
