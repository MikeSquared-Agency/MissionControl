# V4 Rust Contracts

Struct and function signatures for the Rust core. Implementation details only - see `ARCHITECTURE-SPEC.md` for rationale.

---

## Workflow Engine

```rust
// core/workflow/src/phase.rs

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Phase {
    Idea,
    Design,
    Implement,
    Verify,
    Document,
    Release,
}

impl Phase {
    pub fn next(&self) -> Option<Phase>;
    pub fn all() -> &'static [Phase];
}
```

```rust
// core/workflow/src/task.rs

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum TaskStatus {
    Pending,
    Ready,              // Dependencies met
    InProgress,
    Blocked(String),    // Reason
    Done,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Task {
    pub id: String,
    pub name: String,
    pub phase: Phase,
    pub zone: String,
    pub status: TaskStatus,
    pub persona: String,
    pub dependencies: Vec<String>,  // Task IDs
    pub created_at: u64,
    pub updated_at: u64,
}
```

```rust
// core/workflow/src/gate.rs

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum GateStatus {
    Open,
    Closed,
    AwaitingApproval,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GateCriterion {
    pub description: String,
    pub satisfied: bool,
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
```

```rust
// core/workflow/src/engine.rs

pub struct WorkflowEngine {
    current_phase: Phase,
    tasks: HashMap<String, Task>,
    gates: HashMap<String, Gate>,
}

impl WorkflowEngine {
    pub fn new() -> Self;
    
    // Phase
    pub fn current_phase(&self) -> Phase;
    pub fn can_transition(&self, to: Phase) -> bool;
    pub fn transition(&mut self, to: Phase) -> Result<(), WorkflowError>;
    
    // Tasks
    pub fn create_task(&mut self, task: Task) -> String;
    pub fn update_task_status(&mut self, id: &str, status: TaskStatus) -> Result<(), WorkflowError>;
    pub fn get_task(&self, id: &str) -> Option<&Task>;
    pub fn get_ready_tasks(&self) -> Vec<&Task>;
    pub fn get_tasks_for_phase(&self, phase: Phase) -> Vec<&Task>;
    
    // Gates
    pub fn get_gate(&self, phase: Phase) -> Option<&Gate>;
    pub fn check_gate(&self, phase: Phase) -> GateStatus;
    pub fn approve_gate(&mut self, phase: Phase, by: &str) -> Result<(), WorkflowError>;
    
    // Serialization
    pub fn to_json(&self) -> String;
    pub fn from_json(json: &str) -> Result<Self, WorkflowError>;
}
```

---

## Knowledge Manager

```rust
// core/knowledge/src/tokens.rs

pub struct TokenCounter {
    // Uses tiktoken-rs cl100k_base internally
}

impl TokenCounter {
    pub fn new() -> Self;
    pub fn count(&self, text: &str) -> usize;
}
```

```rust
// core/knowledge/src/budget.rs

#[derive(Debug, Clone, PartialEq, Eq)]
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
    pub warning_threshold: f32,   // 0.5
    pub critical_threshold: f32,  // 0.75
}

impl TokenBudget {
    pub fn new(worker_id: &str, budget: usize) -> Self;
    pub fn record(&mut self, tokens: usize);
    pub fn status(&self) -> BudgetStatus;
    pub fn remaining(&self) -> usize;
}
```

```rust
// core/knowledge/src/handoff.rs

#[derive(Debug, Clone, Serialize, Deserialize)]
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
    pub severity: Option<String>,  // low, medium, high
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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
```

```rust
// core/knowledge/src/checkpoint.rs

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Checkpoint {
    pub id: String,
    pub phase: Phase,
    pub created_at: u64,
    pub tasks_snapshot: Vec<Task>,
    pub findings_snapshot: Vec<Finding>,
    pub decisions: Vec<String>,
}
```

```rust
// core/knowledge/src/delta.rs

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Delta {
    pub from_checkpoint: String,
    pub new_findings: Vec<Finding>,
    pub modified_files: Vec<String>,
    pub new_decisions: Vec<String>,
    pub open_questions: Vec<String>,
    pub created_at: u64,
}
```

```rust
// core/knowledge/src/manager.rs

pub struct KnowledgeManager {
    counter: TokenCounter,
    budgets: HashMap<String, TokenBudget>,
    checkpoints: Vec<Checkpoint>,
    deltas: Vec<Delta>,
}

impl KnowledgeManager {
    pub fn new() -> Self;
    
    // Tokens
    pub fn count_tokens(&self, text: &str) -> usize;
    pub fn create_budget(&mut self, worker_id: &str, budget: usize);
    pub fn record_usage(&mut self, worker_id: &str, tokens: usize);
    pub fn check_budget(&self, worker_id: &str) -> Option<BudgetStatus>;
    
    // Handoffs
    pub fn validate_handoff(&self, handoff: &Handoff) -> Result<(), ValidationError>;
    
    // Checkpoints
    pub fn create_checkpoint(&mut self, phase: Phase, tasks: &[Task], findings: &[Finding]) -> String;
    pub fn get_checkpoint(&self, id: &str) -> Option<&Checkpoint>;
    pub fn latest_checkpoint(&self) -> Option<&Checkpoint>;
    
    // Deltas
    pub fn compute_delta(&self, from: &str, findings: &[Finding], files: &[String]) -> Delta;
    pub fn get_deltas_since(&self, checkpoint_id: &str) -> Vec<&Delta>;
    
    // Briefing (returns inputs for LLM compilation)
    pub fn compile_briefing_inputs(&self, task: &Task) -> BriefingInputs;
}

pub struct BriefingInputs {
    pub task: Task,
    pub checkpoint: Option<Checkpoint>,
    pub deltas: Vec<Delta>,
    pub relevant_findings: Vec<Finding>,
}
```

---

## Health Monitor

```rust
// core/runtime/src/health.rs

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum HealthStatus {
    Healthy,
    Idle { since_ms: u64 },
    Stuck { since_ms: u64 },
    Unresponsive,
    Dead,
}

#[derive(Debug, Clone)]
pub struct WorkerHealth {
    pub worker_id: String,
    pub status: HealthStatus,
    pub last_activity: u64,
    pub last_tool_call: Option<u64>,
    pub turns_since_progress: usize,
}

pub struct HealthMonitor {
    workers: HashMap<String, WorkerHealth>,
    stuck_threshold_ms: u64,  // Default: 60000
    idle_threshold_ms: u64,   // Default: 30000
}

impl HealthMonitor {
    pub fn new() -> Self;
    pub fn with_thresholds(stuck_ms: u64, idle_ms: u64) -> Self;
    
    pub fn register_worker(&mut self, worker_id: &str);
    pub fn unregister_worker(&mut self, worker_id: &str);
    
    pub fn mark_activity(&mut self, worker_id: &str);
    pub fn mark_tool_call(&mut self, worker_id: &str);
    
    pub fn check_health(&self, worker_id: &str) -> Option<HealthStatus>;
    pub fn get_stuck_workers(&self) -> Vec<&str>;
    pub fn get_all_health(&self) -> Vec<(&str, &HealthStatus)>;
}
```

---

## FFI Bindings

```rust
// core/ffi/src/lib.rs

use std::ffi::{CStr, CString};
use std::os::raw::c_char;

// All functions return JSON strings or null on error
// Caller must free returned strings with the appropriate free function

// --- Workflow Engine ---

#[no_mangle]
pub extern "C" fn workflow_engine_new() -> *mut WorkflowEngine;

#[no_mangle]
pub extern "C" fn workflow_engine_free(ptr: *mut WorkflowEngine);

#[no_mangle]
pub extern "C" fn workflow_engine_current_phase(ptr: *const WorkflowEngine) -> *mut c_char;

#[no_mangle]
pub extern "C" fn workflow_engine_create_task(
    ptr: *mut WorkflowEngine,
    task_json: *const c_char,
) -> *mut c_char;  // Returns task ID or error

#[no_mangle]
pub extern "C" fn workflow_engine_get_ready_tasks(
    ptr: *const WorkflowEngine,
) -> *mut c_char;  // Returns JSON array

#[no_mangle]
pub extern "C" fn workflow_engine_to_json(ptr: *const WorkflowEngine) -> *mut c_char;

// --- Knowledge Manager ---

#[no_mangle]
pub extern "C" fn knowledge_manager_new() -> *mut KnowledgeManager;

#[no_mangle]
pub extern "C" fn knowledge_manager_free(ptr: *mut KnowledgeManager);

#[no_mangle]
pub extern "C" fn knowledge_manager_count_tokens(
    ptr: *const KnowledgeManager,
    text: *const c_char,
) -> usize;

#[no_mangle]
pub extern "C" fn knowledge_manager_validate_handoff(
    ptr: *const KnowledgeManager,
    handoff_json: *const c_char,
) -> *mut c_char;  // Returns null if valid, error string if invalid

// --- Health Monitor ---

#[no_mangle]
pub extern "C" fn health_monitor_new() -> *mut HealthMonitor;

#[no_mangle]
pub extern "C" fn health_monitor_free(ptr: *mut HealthMonitor);

#[no_mangle]
pub extern "C" fn health_monitor_check_health(
    ptr: *const HealthMonitor,
    worker_id: *const c_char,
) -> *mut c_char;  // Returns HealthStatus JSON

// --- String Management ---

#[no_mangle]
pub extern "C" fn missioncontrol_free_string(ptr: *mut c_char);
```

---

## Cargo Dependencies

```toml
# core/workflow/Cargo.toml
[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
thiserror = "1.0"

# core/knowledge/Cargo.toml
[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
thiserror = "1.0"
tiktoken-rs = "0.5"

# core/runtime/Cargo.toml
[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"

# core/ffi/Cargo.toml
[lib]
crate-type = ["cdylib"]

[dependencies]
workflow = { path = "../workflow" }
knowledge = { path = "../knowledge" }
runtime = { path = "../runtime" }
```