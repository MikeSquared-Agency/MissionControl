use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum BudgetStatus {
    Healthy,
    Warning { remaining: usize },
    Critical { remaining: usize },
    Exceeded,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenBudget {
    pub worker_id: String,
    pub budget: usize,
    pub used: usize,
    pub warning_threshold: f32,
    pub critical_threshold: f32,
}

impl TokenBudget {
    pub fn new(worker_id: &str, budget: usize) -> Self {
        Self {
            worker_id: worker_id.to_string(),
            budget,
            used: 0,
            warning_threshold: 0.5,
            critical_threshold: 0.75,
        }
    }

    pub fn with_thresholds(mut self, warning: f32, critical: f32) -> Self {
        self.warning_threshold = warning;
        self.critical_threshold = critical;
        self
    }

    pub fn record(&mut self, tokens: usize) {
        self.used += tokens;
    }

    pub fn remaining(&self) -> usize {
        self.budget.saturating_sub(self.used)
    }

    pub fn usage_ratio(&self) -> f32 {
        if self.budget == 0 {
            return 1.0;
        }
        self.used as f32 / self.budget as f32
    }

    pub fn status(&self) -> BudgetStatus {
        let ratio = self.usage_ratio();
        let remaining = self.remaining();

        if ratio >= 1.0 {
            BudgetStatus::Exceeded
        } else if ratio >= self.critical_threshold {
            BudgetStatus::Critical { remaining }
        } else if ratio >= self.warning_threshold {
            BudgetStatus::Warning { remaining }
        } else {
            BudgetStatus::Healthy
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_budget_creation() {
        let budget = TokenBudget::new("worker-1", 20000);
        assert_eq!(budget.worker_id, "worker-1");
        assert_eq!(budget.budget, 20000);
        assert_eq!(budget.used, 0);
        assert_eq!(budget.remaining(), 20000);
    }

    #[test]
    fn test_budget_recording() {
        let mut budget = TokenBudget::new("worker-1", 20000);
        budget.record(5000);
        assert_eq!(budget.used, 5000);
        assert_eq!(budget.remaining(), 15000);
    }

    #[test]
    fn test_budget_status_healthy() {
        let mut budget = TokenBudget::new("worker-1", 20000);
        budget.record(8000); // 40%
        assert_eq!(budget.status(), BudgetStatus::Healthy);
    }

    #[test]
    fn test_budget_status_warning() {
        let mut budget = TokenBudget::new("worker-1", 20000);
        budget.record(12000); // 60%
        match budget.status() {
            BudgetStatus::Warning { remaining } => assert_eq!(remaining, 8000),
            _ => panic!("Expected Warning status"),
        }
    }

    #[test]
    fn test_budget_status_critical() {
        let mut budget = TokenBudget::new("worker-1", 20000);
        budget.record(16000); // 80%
        match budget.status() {
            BudgetStatus::Critical { remaining } => assert_eq!(remaining, 4000),
            _ => panic!("Expected Critical status"),
        }
    }

    #[test]
    fn test_budget_status_exceeded() {
        let mut budget = TokenBudget::new("worker-1", 20000);
        budget.record(25000);
        assert_eq!(budget.status(), BudgetStatus::Exceeded);
    }
}
