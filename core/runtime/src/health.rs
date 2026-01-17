use std::collections::HashMap;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum HealthStatus {
    Healthy,
    Idle { since_ms: u64 },
    Stuck { since_ms: u64 },
    Unresponsive,
    Dead,
}

impl Default for HealthStatus {
    fn default() -> Self {
        HealthStatus::Healthy
    }
}

#[derive(Debug, Clone)]
pub struct WorkerHealth {
    pub worker_id: String,
    pub status: HealthStatus,
    pub last_activity: u64,
    pub last_tool_call: Option<u64>,
    pub turns_since_progress: usize,
}

impl WorkerHealth {
    pub fn new(worker_id: impl Into<String>) -> Self {
        Self {
            worker_id: worker_id.into(),
            status: HealthStatus::Healthy,
            last_activity: Self::now(),
            last_tool_call: None,
            turns_since_progress: 0,
        }
    }

    fn now() -> u64 {
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_millis() as u64
    }

    pub fn mark_activity(&mut self) {
        self.last_activity = Self::now();
        self.status = HealthStatus::Healthy;
    }

    pub fn mark_tool_call(&mut self) {
        let now = Self::now();
        self.last_activity = now;
        self.last_tool_call = Some(now);
        self.turns_since_progress = 0;
        self.status = HealthStatus::Healthy;
    }

    pub fn mark_turn(&mut self) {
        self.turns_since_progress += 1;
    }

    pub fn time_since_activity(&self) -> u64 {
        Self::now().saturating_sub(self.last_activity)
    }

    pub fn time_since_tool_call(&self) -> Option<u64> {
        self.last_tool_call.map(|t| Self::now().saturating_sub(t))
    }
}

pub struct HealthMonitor {
    workers: HashMap<String, WorkerHealth>,
    stuck_threshold_ms: u64,
    idle_threshold_ms: u64,
}

impl HealthMonitor {
    pub fn new() -> Self {
        Self {
            workers: HashMap::new(),
            stuck_threshold_ms: 60000,  // 60 seconds
            idle_threshold_ms: 30000,   // 30 seconds
        }
    }

    pub fn with_thresholds(stuck_ms: u64, idle_ms: u64) -> Self {
        Self {
            workers: HashMap::new(),
            stuck_threshold_ms: stuck_ms,
            idle_threshold_ms: idle_ms,
        }
    }

    pub fn register_worker(&mut self, worker_id: &str) {
        self.workers.insert(
            worker_id.to_string(),
            WorkerHealth::new(worker_id),
        );
    }

    pub fn unregister_worker(&mut self, worker_id: &str) {
        self.workers.remove(worker_id);
    }

    pub fn mark_activity(&mut self, worker_id: &str) {
        if let Some(health) = self.workers.get_mut(worker_id) {
            health.mark_activity();
        }
    }

    pub fn mark_tool_call(&mut self, worker_id: &str) {
        if let Some(health) = self.workers.get_mut(worker_id) {
            health.mark_tool_call();
        }
    }

    pub fn mark_turn(&mut self, worker_id: &str) {
        if let Some(health) = self.workers.get_mut(worker_id) {
            health.mark_turn();
        }
    }

    pub fn check_health(&self, worker_id: &str) -> Option<HealthStatus> {
        self.workers.get(worker_id).map(|health| {
            self.compute_status(health)
        })
    }

    fn compute_status(&self, health: &WorkerHealth) -> HealthStatus {
        let idle_time = health.time_since_activity();

        if idle_time >= self.stuck_threshold_ms {
            HealthStatus::Stuck { since_ms: idle_time }
        } else if idle_time >= self.idle_threshold_ms {
            HealthStatus::Idle { since_ms: idle_time }
        } else {
            HealthStatus::Healthy
        }
    }

    pub fn get_stuck_workers(&self) -> Vec<&str> {
        self.workers.iter()
            .filter(|(_, health)| {
                health.time_since_activity() >= self.stuck_threshold_ms
            })
            .map(|(id, _)| id.as_str())
            .collect()
    }

    pub fn get_all_health(&self) -> Vec<(&str, HealthStatus)> {
        self.workers.iter()
            .map(|(id, health)| (id.as_str(), self.compute_status(health)))
            .collect()
    }

    pub fn get_worker(&self, worker_id: &str) -> Option<&WorkerHealth> {
        self.workers.get(worker_id)
    }
}

impl Default for HealthMonitor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_monitor_creation() {
        let monitor = HealthMonitor::new();
        assert!(monitor.workers.is_empty());
    }

    #[test]
    fn test_worker_registration() {
        let mut monitor = HealthMonitor::new();
        monitor.register_worker("worker-1");

        assert!(monitor.check_health("worker-1").is_some());
        assert_eq!(monitor.check_health("worker-1"), Some(HealthStatus::Healthy));
    }

    #[test]
    fn test_worker_unregistration() {
        let mut monitor = HealthMonitor::new();
        monitor.register_worker("worker-1");
        monitor.unregister_worker("worker-1");

        assert!(monitor.check_health("worker-1").is_none());
    }

    #[test]
    fn test_activity_marking() {
        let mut monitor = HealthMonitor::new();
        monitor.register_worker("worker-1");
        monitor.mark_activity("worker-1");

        assert_eq!(monitor.check_health("worker-1"), Some(HealthStatus::Healthy));
    }

    #[test]
    fn test_tool_call_marking() {
        let mut monitor = HealthMonitor::new();
        monitor.register_worker("worker-1");
        monitor.mark_tool_call("worker-1");

        let health = monitor.get_worker("worker-1").unwrap();
        assert!(health.last_tool_call.is_some());
        assert_eq!(health.turns_since_progress, 0);
    }

    #[test]
    fn test_custom_thresholds() {
        let monitor = HealthMonitor::with_thresholds(5000, 2000);
        assert_eq!(monitor.stuck_threshold_ms, 5000);
        assert_eq!(monitor.idle_threshold_ms, 2000);
    }

    #[test]
    fn test_get_all_health() {
        let mut monitor = HealthMonitor::new();
        monitor.register_worker("worker-1");
        monitor.register_worker("worker-2");

        let all = monitor.get_all_health();
        assert_eq!(all.len(), 2);
    }
}
