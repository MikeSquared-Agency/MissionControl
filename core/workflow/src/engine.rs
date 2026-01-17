use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::phase::Phase;
use crate::task::{Task, TaskStatus};
use crate::gate::{Gate, GateStatus};

#[derive(Debug, Error)]
pub enum WorkflowError {
    #[error("Task not found: {0}")]
    TaskNotFound(String),

    #[error("Gate not found for phase: {0:?}")]
    GateNotFound(Phase),

    #[error("Cannot transition from {from:?} to {to:?}")]
    InvalidTransition { from: Phase, to: Phase },

    #[error("Gate not open for phase: {0:?}")]
    GateNotOpen(Phase),

    #[error("Serialization error: {0}")]
    SerializationError(String),

    #[error("Invalid task status transition")]
    InvalidStatusTransition,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowEngine {
    current_phase: Phase,
    tasks: HashMap<String, Task>,
    gates: HashMap<String, Gate>,
}

impl WorkflowEngine {
    pub fn new() -> Self {
        let mut gates = HashMap::new();
        for phase in Phase::all() {
            let gate = Gate::new(*phase);
            gates.insert(gate.id.clone(), gate);
        }

        Self {
            current_phase: Phase::Idea,
            tasks: HashMap::new(),
            gates,
        }
    }

    // Phase management
    pub fn current_phase(&self) -> Phase {
        self.current_phase
    }

    pub fn can_transition(&self, to: Phase) -> bool {
        // Can only transition to the next phase
        if let Some(next) = self.current_phase.next() {
            if next == to {
                // Check if current phase's gate is open
                return self.check_gate(self.current_phase) == GateStatus::Open;
            }
        }
        false
    }

    pub fn transition(&mut self, to: Phase) -> Result<(), WorkflowError> {
        if !self.can_transition(to) {
            if self.check_gate(self.current_phase) != GateStatus::Open {
                return Err(WorkflowError::GateNotOpen(self.current_phase));
            }
            return Err(WorkflowError::InvalidTransition {
                from: self.current_phase,
                to,
            });
        }

        self.current_phase = to;
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

    pub fn get_tasks_for_phase(&self, phase: Phase) -> Vec<&Task> {
        self.tasks.values()
            .filter(|task| task.phase == phase)
            .collect()
    }

    pub fn all_tasks(&self) -> Vec<&Task> {
        self.tasks.values().collect()
    }

    // Gate management
    pub fn get_gate(&self, phase: Phase) -> Option<&Gate> {
        let id = format!("gate-{}", phase.as_str());
        self.gates.get(&id)
    }

    pub fn get_gate_mut(&mut self, phase: Phase) -> Option<&mut Gate> {
        let id = format!("gate-{}", phase.as_str());
        self.gates.get_mut(&id)
    }

    pub fn check_gate(&self, phase: Phase) -> GateStatus {
        self.get_gate(phase)
            .map(|g| g.status.clone())
            .unwrap_or(GateStatus::Closed)
    }

    pub fn approve_gate(&mut self, phase: Phase, by: &str) -> Result<(), WorkflowError> {
        let gate = self.get_gate_mut(phase)
            .ok_or(WorkflowError::GateNotFound(phase))?;

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
        assert_eq!(engine.current_phase(), Phase::Idea);
        assert!(engine.get_gate(Phase::Idea).is_some());
        assert!(engine.get_gate(Phase::Release).is_some());
    }

    #[test]
    fn test_task_creation_and_retrieval() {
        let mut engine = WorkflowEngine::new();
        let task = Task::new("task-1", "Test task", Phase::Idea, "system", "researcher");

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
        let task1 = Task::new("task-1", "First", Phase::Implement, "backend", "developer");
        engine.create_task(task1);

        // Create task with dependency on task-1
        let task2 = Task::new("task-2", "Second", Phase::Implement, "backend", "developer")
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
    fn test_phase_transition() {
        let mut engine = WorkflowEngine::new();

        // Cannot transition without gate approval
        assert!(!engine.can_transition(Phase::Design));

        // Satisfy gate criteria and approve
        if let Some(gate) = engine.get_gate_mut(Phase::Idea) {
            for i in 0..gate.criteria.len() {
                gate.satisfy_criterion(i);
            }
            gate.approve("user");
        }

        // Now can transition
        assert!(engine.can_transition(Phase::Design));
        engine.transition(Phase::Design).unwrap();
        assert_eq!(engine.current_phase(), Phase::Design);
    }

    #[test]
    fn test_serialization() {
        let mut engine = WorkflowEngine::new();
        let task = Task::new("task-1", "Test", Phase::Idea, "system", "researcher");
        engine.create_task(task);

        let json = engine.to_json();
        let restored = WorkflowEngine::from_json(&json).unwrap();

        assert_eq!(restored.current_phase(), Phase::Idea);
        assert!(restored.get_task("task-1").is_some());
    }
}
