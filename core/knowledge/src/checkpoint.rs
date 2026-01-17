use serde::{Deserialize, Serialize};
use workflow::{Phase, Task};
use crate::handoff::Finding;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Checkpoint {
    pub id: String,
    pub phase: Phase,
    pub created_at: u64,
    pub tasks_snapshot: Vec<Task>,
    pub findings_snapshot: Vec<Finding>,
    pub decisions: Vec<String>,
}

impl Checkpoint {
    pub fn new(id: impl Into<String>, phase: Phase) -> Self {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            id: id.into(),
            phase,
            created_at: now,
            tasks_snapshot: Vec::new(),
            findings_snapshot: Vec::new(),
            decisions: Vec::new(),
        }
    }

    pub fn with_tasks(mut self, tasks: Vec<Task>) -> Self {
        self.tasks_snapshot = tasks;
        self
    }

    pub fn with_findings(mut self, findings: Vec<Finding>) -> Self {
        self.findings_snapshot = findings;
        self
    }

    pub fn with_decisions(mut self, decisions: Vec<String>) -> Self {
        self.decisions = decisions;
        self
    }

    pub fn add_decision(&mut self, decision: impl Into<String>) {
        self.decisions.push(decision.into());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::handoff::Finding;

    #[test]
    fn test_checkpoint_creation() {
        let checkpoint = Checkpoint::new("cp-1", Phase::Design);
        assert_eq!(checkpoint.id, "cp-1");
        assert_eq!(checkpoint.phase, Phase::Design);
        assert!(checkpoint.tasks_snapshot.is_empty());
        assert!(checkpoint.findings_snapshot.is_empty());
    }

    #[test]
    fn test_checkpoint_with_data() {
        let finding = Finding::decision("Chose REST over GraphQL");
        let checkpoint = Checkpoint::new("cp-2", Phase::Design)
            .with_findings(vec![finding])
            .with_decisions(vec!["Use PostgreSQL".to_string()]);

        assert_eq!(checkpoint.findings_snapshot.len(), 1);
        assert_eq!(checkpoint.decisions.len(), 1);
    }
}
