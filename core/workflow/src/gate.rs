use serde::{Deserialize, Serialize};
use crate::phase::Phase;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum GateStatus {
    Open,
    Closed,
    AwaitingApproval,
}

impl Default for GateStatus {
    fn default() -> Self {
        GateStatus::Closed
    }
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
    pub phase: Phase,
    pub status: GateStatus,
    pub criteria: Vec<GateCriterion>,
    pub approved_at: Option<u64>,
    pub approved_by: Option<String>,
}

impl Gate {
    pub fn new(phase: Phase) -> Self {
        let id = format!("gate-{}", phase.as_str());
        Self {
            id,
            phase,
            status: GateStatus::Closed,
            criteria: Self::default_criteria_for_phase(phase),
            approved_at: None,
            approved_by: None,
        }
    }

    fn default_criteria_for_phase(phase: Phase) -> Vec<GateCriterion> {
        match phase {
            Phase::Idea => vec![
                GateCriterion::new("Problem statement defined"),
                GateCriterion::new("Feasibility assessed"),
            ],
            Phase::Design => vec![
                GateCriterion::new("Spec document complete"),
                GateCriterion::new("Technical approach approved"),
            ],
            Phase::Implement => vec![
                GateCriterion::new("All tasks complete"),
                GateCriterion::new("Code compiles"),
            ],
            Phase::Verify => vec![
                GateCriterion::new("Tests passing"),
                GateCriterion::new("Review complete"),
            ],
            Phase::Document => vec![
                GateCriterion::new("README updated"),
                GateCriterion::new("API documented"),
            ],
            Phase::Release => vec![
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
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_gate_creation() {
        let gate = Gate::new(Phase::Design);
        assert_eq!(gate.id, "gate-design");
        assert_eq!(gate.phase, Phase::Design);
        assert_eq!(gate.status, GateStatus::Closed);
        assert!(!gate.criteria.is_empty());
    }

    #[test]
    fn test_gate_status_progression() {
        let mut gate = Gate::new(Phase::Idea);
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
        let gate = Gate::new(Phase::Implement);
        let json = serde_json::to_string(&gate).unwrap();
        let parsed: Gate = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.phase, Phase::Implement);
    }
}
