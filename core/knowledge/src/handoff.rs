use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FindingType {
    Discovery,
    Blocker,
    Decision,
    Concern,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Finding {
    pub finding_type: FindingType,
    pub summary: String,
    pub details_path: Option<String>,
    pub severity: Option<String>,
}

impl Finding {
    pub fn new(finding_type: FindingType, summary: impl Into<String>) -> Self {
        Self {
            finding_type,
            summary: summary.into(),
            details_path: None,
            severity: None,
        }
    }

    pub fn with_details(mut self, path: impl Into<String>) -> Self {
        self.details_path = Some(path.into());
        self
    }

    pub fn with_severity(mut self, severity: impl Into<String>) -> Self {
        self.severity = Some(severity.into());
        self
    }

    pub fn discovery(summary: impl Into<String>) -> Self {
        Self::new(FindingType::Discovery, summary)
    }

    pub fn blocker(summary: impl Into<String>) -> Self {
        Self::new(FindingType::Blocker, summary)
    }

    pub fn decision(summary: impl Into<String>) -> Self {
        Self::new(FindingType::Decision, summary)
    }

    pub fn concern(summary: impl Into<String>) -> Self {
        Self::new(FindingType::Concern, summary)
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum HandoffStatus {
    Complete,
    Blocked(String),
    Partial,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuccessorContext {
    pub key_decisions: Vec<String>,
    pub gotchas: Vec<String>,
    pub recommended_approach: Option<String>,
}

impl SuccessorContext {
    pub fn new() -> Self {
        Self {
            key_decisions: Vec::new(),
            gotchas: Vec::new(),
            recommended_approach: None,
        }
    }

    pub fn with_decision(mut self, decision: impl Into<String>) -> Self {
        self.key_decisions.push(decision.into());
        self
    }

    pub fn with_gotcha(mut self, gotcha: impl Into<String>) -> Self {
        self.gotchas.push(gotcha.into());
        self
    }

    pub fn with_approach(mut self, approach: impl Into<String>) -> Self {
        self.recommended_approach = Some(approach.into());
        self
    }
}

impl Default for SuccessorContext {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Handoff {
    pub task_id: String,
    pub worker_id: String,
    pub status: HandoffStatus,
    pub findings: Vec<Finding>,
    pub artifacts: Vec<String>,
    pub open_questions: Vec<String>,
    pub context_for_successor: Option<SuccessorContext>,
    pub timestamp: u64,
}

impl Handoff {
    pub fn new(task_id: impl Into<String>, worker_id: impl Into<String>, status: HandoffStatus) -> Self {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            task_id: task_id.into(),
            worker_id: worker_id.into(),
            status,
            findings: Vec::new(),
            artifacts: Vec::new(),
            open_questions: Vec::new(),
            context_for_successor: None,
            timestamp: now,
        }
    }

    pub fn complete(task_id: impl Into<String>, worker_id: impl Into<String>) -> Self {
        Self::new(task_id, worker_id, HandoffStatus::Complete)
    }

    pub fn blocked(task_id: impl Into<String>, worker_id: impl Into<String>, reason: impl Into<String>) -> Self {
        Self::new(task_id, worker_id, HandoffStatus::Blocked(reason.into()))
    }

    pub fn partial(task_id: impl Into<String>, worker_id: impl Into<String>) -> Self {
        Self::new(task_id, worker_id, HandoffStatus::Partial)
    }

    pub fn with_finding(mut self, finding: Finding) -> Self {
        self.findings.push(finding);
        self
    }

    pub fn with_artifact(mut self, path: impl Into<String>) -> Self {
        self.artifacts.push(path.into());
        self
    }

    pub fn with_question(mut self, question: impl Into<String>) -> Self {
        self.open_questions.push(question.into());
        self
    }

    pub fn with_successor_context(mut self, context: SuccessorContext) -> Self {
        self.context_for_successor = Some(context);
        self
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_finding_creation() {
        let finding = Finding::discovery("Found existing auth implementation")
            .with_details(".mission/findings/auth.md")
            .with_severity("low");

        assert_eq!(finding.finding_type, FindingType::Discovery);
        assert!(finding.summary.contains("auth"));
        assert!(finding.details_path.is_some());
    }

    #[test]
    fn test_handoff_creation() {
        let handoff = Handoff::complete("task-1", "worker-1")
            .with_finding(Finding::decision("Chose JWT over sessions"))
            .with_artifact("src/auth.rs")
            .with_question("Should we support refresh tokens?");

        assert_eq!(handoff.task_id, "task-1");
        assert_eq!(handoff.worker_id, "worker-1");
        assert_eq!(handoff.status, HandoffStatus::Complete);
        assert_eq!(handoff.findings.len(), 1);
        assert_eq!(handoff.artifacts.len(), 1);
        assert_eq!(handoff.open_questions.len(), 1);
    }

    #[test]
    fn test_handoff_serialization() {
        let handoff = Handoff::blocked("task-1", "worker-1", "Waiting for API docs");
        let json = serde_json::to_string(&handoff).unwrap();
        assert!(json.contains("blocked"));
        assert!(json.contains("Waiting for API docs"));
    }
}
