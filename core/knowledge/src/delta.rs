use serde::{Deserialize, Serialize};
use crate::handoff::Finding;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Delta {
    pub from_checkpoint: String,
    pub new_findings: Vec<Finding>,
    pub modified_files: Vec<String>,
    pub new_decisions: Vec<String>,
    pub open_questions: Vec<String>,
    pub created_at: u64,
}

impl Delta {
    pub fn new(from_checkpoint: impl Into<String>) -> Self {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            from_checkpoint: from_checkpoint.into(),
            new_findings: Vec::new(),
            modified_files: Vec::new(),
            new_decisions: Vec::new(),
            open_questions: Vec::new(),
            created_at: now,
        }
    }

    pub fn with_findings(mut self, findings: Vec<Finding>) -> Self {
        self.new_findings = findings;
        self
    }

    pub fn with_files(mut self, files: Vec<String>) -> Self {
        self.modified_files = files;
        self
    }

    pub fn with_decisions(mut self, decisions: Vec<String>) -> Self {
        self.new_decisions = decisions;
        self
    }

    pub fn with_questions(mut self, questions: Vec<String>) -> Self {
        self.open_questions = questions;
        self
    }

    pub fn add_finding(&mut self, finding: Finding) {
        self.new_findings.push(finding);
    }

    pub fn add_file(&mut self, file: impl Into<String>) {
        self.modified_files.push(file.into());
    }

    pub fn add_decision(&mut self, decision: impl Into<String>) {
        self.new_decisions.push(decision.into());
    }

    pub fn add_question(&mut self, question: impl Into<String>) {
        self.open_questions.push(question.into());
    }

    pub fn is_empty(&self) -> bool {
        self.new_findings.is_empty()
            && self.modified_files.is_empty()
            && self.new_decisions.is_empty()
            && self.open_questions.is_empty()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::handoff::Finding;

    #[test]
    fn test_delta_creation() {
        let delta = Delta::new("cp-1");
        assert_eq!(delta.from_checkpoint, "cp-1");
        assert!(delta.is_empty());
    }

    #[test]
    fn test_delta_with_data() {
        let delta = Delta::new("cp-1")
            .with_findings(vec![Finding::discovery("New API endpoint")])
            .with_files(vec!["src/api.rs".to_string()])
            .with_decisions(vec!["Use pagination".to_string()]);

        assert!(!delta.is_empty());
        assert_eq!(delta.new_findings.len(), 1);
        assert_eq!(delta.modified_files.len(), 1);
        assert_eq!(delta.new_decisions.len(), 1);
    }

    #[test]
    fn test_delta_add_methods() {
        let mut delta = Delta::new("cp-1");
        delta.add_finding(Finding::concern("Performance issue"));
        delta.add_file("src/slow.rs");
        delta.add_question("Should we optimize now?");

        assert_eq!(delta.new_findings.len(), 1);
        assert_eq!(delta.modified_files.len(), 1);
        assert_eq!(delta.open_questions.len(), 1);
    }
}
