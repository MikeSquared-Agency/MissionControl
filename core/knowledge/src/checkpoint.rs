use serde::{Deserialize, Serialize};
use workflow::{Stage, Task};
use crate::handoff::Finding;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Checkpoint {
    pub id: String,
    pub stage: Stage,
    pub created_at: u64,
    pub tasks_snapshot: Vec<Task>,
    pub findings_snapshot: Vec<Finding>,
    pub decisions: Vec<String>,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub blockers: Vec<String>,
}

impl Checkpoint {
    pub fn new(id: impl Into<String>, stage: Stage) -> Self {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            id: id.into(),
            stage,
            created_at: now,
            tasks_snapshot: Vec::new(),
            findings_snapshot: Vec::new(),
            decisions: Vec::new(),
            session_id: None,
            blockers: Vec::new(),
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

    pub fn with_session_id(mut self, session_id: impl Into<String>) -> Self {
        self.session_id = Some(session_id.into());
        self
    }

    pub fn with_blockers(mut self, blockers: Vec<String>) -> Self {
        self.blockers = blockers;
        self
    }

    pub fn add_decision(&mut self, decision: impl Into<String>) {
        self.decisions.push(decision.into());
    }

    pub fn add_blocker(&mut self, blocker: impl Into<String>) {
        self.blockers.push(blocker.into());
    }
}

/// Compiles a checkpoint into a concise markdown briefing (~500 tokens).
pub struct CheckpointCompiler;

impl CheckpointCompiler {
    pub fn compile(checkpoint: &Checkpoint) -> String {
        let mut sections = Vec::new();

        // Stage
        sections.push(format!("## Stage: {}", checkpoint.stage.as_str()));

        // Session
        if let Some(ref session_id) = checkpoint.session_id {
            sections.push(format!("**Session:** {}", session_id));
        }

        // Decisions
        if !checkpoint.decisions.is_empty() {
            let mut s = String::from("## Decisions\n");
            for d in &checkpoint.decisions {
                s.push_str(&format!("- {}\n", d));
            }
            sections.push(s);
        }

        // Tasks Summary
        if !checkpoint.tasks_snapshot.is_empty() {
            let total = checkpoint.tasks_snapshot.len();
            let done = checkpoint.tasks_snapshot.iter()
                .filter(|t| t.is_done())
                .count();
            let blocked = checkpoint.tasks_snapshot.iter()
                .filter(|t| t.is_blocked())
                .count();
            let pending = total - done - blocked;

            let mut s = format!("## Tasks Summary\n- Total: {}\n- Done: {}\n- Pending: {}\n", total, done, pending);
            if blocked > 0 {
                s.push_str(&format!("- Blocked: {}\n", blocked));
            }
            sections.push(s);
        }

        // Blockers
        if !checkpoint.blockers.is_empty() {
            let mut s = String::from("## Blockers\n");
            for b in &checkpoint.blockers {
                s.push_str(&format!("- {}\n", b));
            }
            sections.push(s);
        }

        // Key Findings
        if !checkpoint.findings_snapshot.is_empty() {
            let mut s = String::from("## Key Findings\n");
            for (i, f) in checkpoint.findings_snapshot.iter().enumerate() {
                if i >= 5 { break; } // Limit to keep briefing concise
                s.push_str(&format!("- [{}] {}\n", f.finding_type.as_str(), f.summary));
            }
            if checkpoint.findings_snapshot.len() > 5 {
                s.push_str(&format!("- ... and {} more\n", checkpoint.findings_snapshot.len() - 5));
            }
            sections.push(s);
        }

        sections.join("\n")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::handoff::Finding;

    #[test]
    fn test_checkpoint_creation() {
        let checkpoint = Checkpoint::new("cp-1", Stage::Design);
        assert_eq!(checkpoint.id, "cp-1");
        assert_eq!(checkpoint.stage, Stage::Design);
        assert!(checkpoint.tasks_snapshot.is_empty());
        assert!(checkpoint.findings_snapshot.is_empty());
        assert!(checkpoint.session_id.is_none());
        assert!(checkpoint.blockers.is_empty());
    }

    #[test]
    fn test_checkpoint_with_data() {
        let finding = Finding::decision("Chose REST over GraphQL");
        let checkpoint = Checkpoint::new("cp-2", Stage::Design)
            .with_findings(vec![finding])
            .with_decisions(vec!["Use PostgreSQL".to_string()])
            .with_session_id("session-abc")
            .with_blockers(vec!["Waiting for API key".to_string()]);

        assert_eq!(checkpoint.findings_snapshot.len(), 1);
        assert_eq!(checkpoint.decisions.len(), 1);
        assert_eq!(checkpoint.session_id, Some("session-abc".to_string()));
        assert_eq!(checkpoint.blockers.len(), 1);
    }

    #[test]
    fn test_checkpoint_compile_produces_markdown() {
        let checkpoint = Checkpoint::new("cp-3", Stage::Implement)
            .with_decisions(vec!["Use Rust for core".to_string()])
            .with_session_id("session-001")
            .with_blockers(vec!["CI pipeline failing".to_string()]);

        let briefing = CheckpointCompiler::compile(&checkpoint);
        assert!(briefing.contains("## Stage: implement"));
        assert!(briefing.contains("## Decisions"));
        assert!(briefing.contains("Use Rust for core"));
        assert!(briefing.contains("## Blockers"));
        assert!(briefing.contains("CI pipeline failing"));
        assert!(briefing.contains("session-001"));
    }

    #[test]
    fn test_checkpoint_compile_under_500_tokens() {
        // A checkpoint with reasonable data should produce a concise briefing
        let checkpoint = Checkpoint::new("cp-4", Stage::Verify)
            .with_decisions(vec![
                "Decision 1".to_string(),
                "Decision 2".to_string(),
            ])
            .with_blockers(vec!["Blocker 1".to_string()]);

        let briefing = CheckpointCompiler::compile(&checkpoint);
        // Rough token estimate: ~4 chars per token
        let estimated_tokens = briefing.len() / 4;
        assert!(estimated_tokens < 500, "Briefing too long: ~{} tokens", estimated_tokens);
    }
}
