use std::collections::HashMap;
use thiserror::Error;
use workflow::{Phase, Task};

use crate::tokens::TokenCounter;
use crate::budget::{TokenBudget, BudgetStatus};
use crate::handoff::{Handoff, Finding};
use crate::checkpoint::Checkpoint;
use crate::delta::Delta;

#[derive(Debug, Error)]
pub enum ValidationError {
    #[error("Missing required field: {0}")]
    MissingField(String),

    #[error("Invalid field value: {field} - {reason}")]
    InvalidValue { field: String, reason: String },

    #[error("Summary too long: {0} chars (max 500)")]
    SummaryTooLong(usize),

    #[error("Blocked status requires blocked_reason")]
    MissingBlockedReason,
}

#[derive(Debug, Clone)]
pub struct BriefingInputs {
    pub task: Task,
    pub checkpoint: Option<Checkpoint>,
    pub deltas: Vec<Delta>,
    pub relevant_findings: Vec<Finding>,
}

pub struct KnowledgeManager {
    counter: TokenCounter,
    budgets: HashMap<String, TokenBudget>,
    checkpoints: Vec<Checkpoint>,
    deltas: Vec<Delta>,
    findings: Vec<Finding>,
}

impl KnowledgeManager {
    pub fn new() -> Self {
        Self {
            counter: TokenCounter::new(),
            budgets: HashMap::new(),
            checkpoints: Vec::new(),
            deltas: Vec::new(),
            findings: Vec::new(),
        }
    }

    // Token management
    pub fn count_tokens(&self, text: &str) -> usize {
        self.counter.count(text)
    }

    pub fn create_budget(&mut self, worker_id: &str, budget: usize) {
        self.budgets.insert(
            worker_id.to_string(),
            TokenBudget::new(worker_id, budget),
        );
    }

    pub fn record_usage(&mut self, worker_id: &str, tokens: usize) {
        if let Some(budget) = self.budgets.get_mut(worker_id) {
            budget.record(tokens);
        }
    }

    pub fn check_budget(&self, worker_id: &str) -> Option<BudgetStatus> {
        self.budgets.get(worker_id).map(|b| b.status())
    }

    pub fn get_budget(&self, worker_id: &str) -> Option<&TokenBudget> {
        self.budgets.get(worker_id)
    }

    // Handoff validation
    pub fn validate_handoff(&self, handoff: &Handoff) -> Result<(), ValidationError> {
        // Validate task_id is present
        if handoff.task_id.is_empty() {
            return Err(ValidationError::MissingField("task_id".to_string()));
        }

        // Validate worker_id is present
        if handoff.worker_id.is_empty() {
            return Err(ValidationError::MissingField("worker_id".to_string()));
        }

        // Validate blocked status has reason
        if let crate::handoff::HandoffStatus::Blocked(reason) = &handoff.status {
            if reason.is_empty() {
                return Err(ValidationError::MissingBlockedReason);
            }
        }

        // Validate finding summaries
        for finding in &handoff.findings {
            if finding.summary.len() > 500 {
                return Err(ValidationError::SummaryTooLong(finding.summary.len()));
            }
            if finding.summary.is_empty() {
                return Err(ValidationError::MissingField("finding.summary".to_string()));
            }
        }

        Ok(())
    }

    // Checkpoint management
    pub fn create_checkpoint(
        &mut self,
        phase: Phase,
        tasks: &[Task],
        findings: &[Finding],
    ) -> String {
        let id = format!("cp-{}-{}", phase.as_str(), self.checkpoints.len());
        let checkpoint = Checkpoint::new(&id, phase)
            .with_tasks(tasks.to_vec())
            .with_findings(findings.to_vec());

        self.checkpoints.push(checkpoint);
        id
    }

    pub fn get_checkpoint(&self, id: &str) -> Option<&Checkpoint> {
        self.checkpoints.iter().find(|cp| cp.id == id)
    }

    pub fn latest_checkpoint(&self) -> Option<&Checkpoint> {
        self.checkpoints.last()
    }

    // Delta management
    pub fn compute_delta(
        &self,
        from: &str,
        findings: &[Finding],
        files: &[String],
    ) -> Delta {
        Delta::new(from)
            .with_findings(findings.to_vec())
            .with_files(files.to_vec())
    }

    pub fn store_delta(&mut self, delta: Delta) {
        self.deltas.push(delta);
    }

    pub fn get_deltas_since(&self, checkpoint_id: &str) -> Vec<&Delta> {
        self.deltas.iter()
            .filter(|d| d.from_checkpoint == checkpoint_id)
            .collect()
    }

    // Finding management
    pub fn store_finding(&mut self, finding: Finding) {
        self.findings.push(finding);
    }

    pub fn all_findings(&self) -> &[Finding] {
        &self.findings
    }

    // Briefing compilation
    pub fn compile_briefing_inputs(&self, task: &Task) -> BriefingInputs {
        let checkpoint = self.latest_checkpoint().cloned();

        let deltas = if let Some(ref cp) = checkpoint {
            self.get_deltas_since(&cp.id)
                .into_iter()
                .cloned()
                .collect()
        } else {
            Vec::new()
        };

        // Filter findings relevant to this task's zone/phase
        let relevant_findings: Vec<Finding> = self.findings.iter()
            .cloned()
            .collect();

        BriefingInputs {
            task: task.clone(),
            checkpoint,
            deltas,
            relevant_findings,
        }
    }
}

impl Default for KnowledgeManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::handoff::{HandoffStatus, Finding};

    #[test]
    fn test_manager_creation() {
        let manager = KnowledgeManager::new();
        assert!(manager.budgets.is_empty());
        assert!(manager.checkpoints.is_empty());
    }

    #[test]
    fn test_budget_management() {
        let mut manager = KnowledgeManager::new();
        manager.create_budget("worker-1", 20000);

        assert!(manager.check_budget("worker-1").is_some());
        assert_eq!(manager.check_budget("worker-1"), Some(BudgetStatus::Healthy));

        manager.record_usage("worker-1", 15000);
        match manager.check_budget("worker-1") {
            Some(BudgetStatus::Critical { remaining: _ }) => (),
            other => panic!("Expected Critical, got {:?}", other),
        }
    }

    #[test]
    fn test_handoff_validation_success() {
        let manager = KnowledgeManager::new();
        let handoff = Handoff::complete("task-1", "worker-1")
            .with_finding(Finding::decision("Test decision"));

        assert!(manager.validate_handoff(&handoff).is_ok());
    }

    #[test]
    fn test_handoff_validation_missing_task_id() {
        let manager = KnowledgeManager::new();
        let handoff = Handoff::complete("", "worker-1");

        assert!(matches!(
            manager.validate_handoff(&handoff),
            Err(ValidationError::MissingField(_))
        ));
    }

    #[test]
    fn test_handoff_validation_summary_too_long() {
        let manager = KnowledgeManager::new();
        let long_summary = "x".repeat(501);
        let handoff = Handoff::complete("task-1", "worker-1")
            .with_finding(Finding::discovery(long_summary));

        assert!(matches!(
            manager.validate_handoff(&handoff),
            Err(ValidationError::SummaryTooLong(_))
        ));
    }

    #[test]
    fn test_checkpoint_creation() {
        let mut manager = KnowledgeManager::new();
        let id = manager.create_checkpoint(Phase::Design, &[], &[]);

        assert!(id.starts_with("cp-design"));
        assert!(manager.get_checkpoint(&id).is_some());
        assert!(manager.latest_checkpoint().is_some());
    }

    #[test]
    fn test_delta_management() {
        let mut manager = KnowledgeManager::new();
        let cp_id = manager.create_checkpoint(Phase::Design, &[], &[]);

        let delta = manager.compute_delta(
            &cp_id,
            &[Finding::discovery("New finding")],
            &["src/new.rs".to_string()],
        );

        manager.store_delta(delta);

        let deltas = manager.get_deltas_since(&cp_id);
        assert_eq!(deltas.len(), 1);
    }
}
