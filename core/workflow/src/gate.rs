use serde::{Deserialize, Serialize};
use crate::stage::Stage;
use crate::task::Task;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum GateStatus {
    Open,
    #[default]
    Closed,
    AwaitingApproval,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GateCriterion {
    pub description: String,
    pub satisfied: bool,
}

impl GateCriterion {
    pub fn new(description: impl Into<String>) -> Self {
        Self {
            description: description.into(),
            satisfied: false,
        }
    }

    pub fn satisfy(&mut self) {
        self.satisfied = true;
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Gate {
    pub id: String,
    pub stage: Stage,
    pub status: GateStatus,
    pub criteria: Vec<GateCriterion>,
    pub approved_at: Option<u64>,
    pub approved_by: Option<String>,
}

impl Gate {
    pub fn new(stage: Stage) -> Self {
        let id = format!("gate-{}", stage.as_str());
        Self {
            id,
            stage,
            status: GateStatus::Closed,
            criteria: Self::default_criteria_for_stage(stage),
            approved_at: None,
            approved_by: None,
        }
    }

    fn default_criteria_for_stage(stage: Stage) -> Vec<GateCriterion> {
        match stage {
            Stage::Discovery => vec![
                GateCriterion::new("Problem space explored"),
                GateCriterion::new("Stakeholders identified"),
            ],
            Stage::Goal => vec![
                GateCriterion::new("Goal statement defined"),
                GateCriterion::new("Success metrics established"),
            ],
            Stage::Requirements => vec![
                GateCriterion::new("Requirements documented"),
                GateCriterion::new("Acceptance criteria defined"),
            ],
            Stage::Planning => vec![
                GateCriterion::new("Tasks broken down"),
                GateCriterion::new("Dependencies mapped"),
            ],
            Stage::Design => vec![
                GateCriterion::new("Spec document complete"),
                GateCriterion::new("Technical approach approved"),
            ],
            Stage::Implement => vec![
                GateCriterion::new("All unit tests pass"),
                GateCriterion::new("Code compiles cleanly"),
            ],
            Stage::Verify => vec![
                GateCriterion::new("Code review complete"),
                GateCriterion::new("All review issues addressed"),
                GateCriterion::new("Requirements satisfied"),
            ],
            Stage::Validate => vec![
                GateCriterion::new("E2E integration tests pass"),
                GateCriterion::new("Real environment validated"),
            ],
            Stage::Document => vec![
                GateCriterion::new("README updated"),
                GateCriterion::new("API documented"),
            ],
            Stage::Release => vec![
                GateCriterion::new("Deployed successfully"),
                GateCriterion::new("Smoke tests pass"),
            ],
        }
    }

    pub fn all_criteria_satisfied(&self) -> bool {
        self.criteria.iter().all(|c| c.satisfied)
    }

    pub fn update_status(&mut self) {
        if self.all_criteria_satisfied() {
            if self.approved_at.is_some() {
                self.status = GateStatus::Open;
            } else {
                self.status = GateStatus::AwaitingApproval;
            }
        } else {
            self.status = GateStatus::Closed;
        }
    }

    pub fn approve(&mut self, by: impl Into<String>) {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        self.approved_at = Some(now);
        self.approved_by = Some(by.into());
        self.status = GateStatus::Open;
    }

    pub fn satisfy_criterion(&mut self, index: usize) -> bool {
        if let Some(criterion) = self.criteria.get_mut(index) {
            criterion.satisfy();
            self.update_status();
            true
        } else {
            false
        }
    }

    /// Check implement stage gate: if there are multiple implement tasks,
    /// at least one must be an integrator task with status done.
    /// Returns a list of failure messages (empty = pass).
    pub fn check_integrator_requirement(tasks: &[Task]) -> Vec<String> {
        let implement_tasks: Vec<&Task> = tasks
            .iter()
            .filter(|t| t.stage == Stage::Implement)
            .collect();

        if implement_tasks.len() > 1 {
            let has_done_integrator = implement_tasks
                .iter()
                .any(|t| t.persona == "integrator" && t.is_done());

            if !has_done_integrator {
                return vec![
                    "Integration task required: multiple implement tasks but no completed integrator task".to_string(),
                ];
            }
        }

        vec![]
    }

    /// Check verify stage gate: at least one task must have persona "reviewer".
    /// Returns a list of failure messages (empty = pass).
    pub fn check_reviewer_requirement(tasks: &[Task]) -> Vec<String> {
        let verify_tasks: Vec<&Task> = tasks
            .iter()
            .filter(|t| t.stage == Stage::Verify)
            .collect();

        let has_reviewer = verify_tasks
            .iter()
            .any(|t| t.persona == "reviewer" && t.is_done());

        if !has_reviewer {
            return vec![
                "Verify stage requires at least one reviewer task".to_string(),
            ];
        }

        vec![]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_gate_creation() {
        let gate = Gate::new(Stage::Design);
        assert_eq!(gate.id, "gate-design");
        assert_eq!(gate.stage, Stage::Design);
        assert_eq!(gate.status, GateStatus::Closed);
        assert!(!gate.criteria.is_empty());
    }

    #[test]
    fn test_gate_creation_all_stages() {
        for stage in Stage::all() {
            let gate = Gate::new(*stage);
            assert_eq!(gate.id, format!("gate-{}", stage.as_str()));
            assert_eq!(gate.stage, *stage);
            assert!(gate.criteria.len() >= 2);
        }
    }

    #[test]
    fn test_gate_status_progression() {
        let mut gate = Gate::new(Stage::Discovery);
        assert_eq!(gate.status, GateStatus::Closed);

        // Satisfy all criteria
        for i in 0..gate.criteria.len() {
            gate.satisfy_criterion(i);
        }
        assert_eq!(gate.status, GateStatus::AwaitingApproval);

        // Approve
        gate.approve("user");
        assert_eq!(gate.status, GateStatus::Open);
        assert!(gate.approved_at.is_some());
        assert_eq!(gate.approved_by, Some("user".to_string()));
    }

    #[test]
    fn test_gate_serialization() {
        let gate = Gate::new(Stage::Implement);
        let json = serde_json::to_string(&gate).unwrap();
        let parsed: Gate = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.stage, Stage::Implement);
    }

    #[test]
    fn test_reviewer_requirement_fails_without_reviewer() {
        use crate::task::{Task, TaskStatus};

        let mut t1 = Task::new("t1", "Write tests", Stage::Verify, "qa", "developer");
        t1.status = TaskStatus::Done;

        let failures = Gate::check_reviewer_requirement(&[t1]);
        assert_eq!(failures.len(), 1);
        assert!(failures[0].contains("Verify stage requires at least one reviewer task"));
    }

    #[test]
    fn test_reviewer_requirement_passes_with_done_reviewer() {
        use crate::task::{Task, TaskStatus};

        let mut t1 = Task::new("t1", "Write tests", Stage::Verify, "qa", "developer");
        t1.status = TaskStatus::Done;
        let mut t2 = Task::new("t2", "Code review", Stage::Verify, "backend", "reviewer");
        t2.status = TaskStatus::Done;

        let failures = Gate::check_reviewer_requirement(&[t1, t2]);
        assert!(failures.is_empty());
    }

    #[test]
    fn test_reviewer_requirement_single_non_reviewer_task() {
        use crate::task::Task;

        let t1 = Task::new("t1", "Run checks", Stage::Verify, "qa", "developer");

        let failures = Gate::check_reviewer_requirement(&[t1]);
        assert_eq!(failures.len(), 1);
        assert!(failures[0].contains("Verify stage requires at least one reviewer task"));
    }

    #[test]
    fn test_integrator_requirement_fails_without_integrator() {
        use crate::task::{Task, TaskStatus};

        let mut t1 = Task::new("t1", "Build API", Stage::Implement, "backend", "developer");
        t1.status = TaskStatus::Done;
        let mut t2 = Task::new("t2", "Build UI", Stage::Implement, "frontend", "developer");
        t2.status = TaskStatus::Done;
        let t3 = Task::new("t3", "Build DB", Stage::Implement, "backend", "developer");

        let failures = Gate::check_integrator_requirement(&[t1, t2, t3]);
        assert_eq!(failures.len(), 1);
        assert!(failures[0].contains("Integration task required"));
    }

    #[test]
    fn test_integrator_requirement_passes_with_done_integrator() {
        use crate::task::{Task, TaskStatus};

        let mut t1 = Task::new("t1", "Build API", Stage::Implement, "backend", "developer");
        t1.status = TaskStatus::Done;
        let mut t2 = Task::new("t2", "Build UI", Stage::Implement, "frontend", "developer");
        t2.status = TaskStatus::Done;
        let mut t3 = Task::new("t3", "Integrate all", Stage::Implement, "backend", "integrator");
        t3.status = TaskStatus::Done;

        let failures = Gate::check_integrator_requirement(&[t1, t2, t3]);
        assert!(failures.is_empty());
    }

    #[test]
    fn test_integrator_requirement_skipped_for_single_task() {
        use crate::task::Task;

        let t1 = Task::new("t1", "Build API", Stage::Implement, "backend", "developer");
        let failures = Gate::check_integrator_requirement(&[t1]);
        assert!(failures.is_empty());
    }
}
