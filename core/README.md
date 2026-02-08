# MissionControl Core (Rust)

Rust crates providing deterministic logic for MissionControl.

## Overview

```mermaid
flowchart TB
    subgraph CLI["mc-core CLI"]
        VAL[validate-handoff]
        GATE[check-gate]
        TOK[count-tokens]
    end

    subgraph Crates["Rust Crates"]
        WF[workflow<br/>Stage state machine]
        KN[knowledge<br/>Token management & checkpoints]
        PR[mc-protocol<br/>Data structures]
    end

    subgraph External["External"]
        TK[tiktoken-rs<br/>Token counting]
        SD[serde<br/>JSON serialization]
        NT[notify<br/>File watching]
    end

    VAL --> PR
    GATE --> WF
    TOK --> KN

    WF --> SD
    KN --> TK
    KN --> SD
    PR --> SD
    PR --> NT
```

## Crate Structure

```
core/
├── workflow/       # Stage state machine, gates, tasks
├── knowledge/      # Token counting, budgets, handoffs, checkpoints
├── mc-protocol/    # Shared data structures
├── mc-core/        # CLI binary
├── ffi/            # C-compatible FFI bindings
└── Cargo.toml      # Workspace manifest
```

## mc-core CLI

The `mc-core` binary exposes core functionality to the Go layer:

```bash
# Validate a handoff JSON file
mc-core validate-handoff findings.json

# Check if gate criteria are met
mc-core check-gate design

# Count tokens in a file
mc-core count-tokens spec.md

# Compile checkpoint into markdown briefing
mc-core checkpoint-compile checkpoint.json

# Validate checkpoint JSON schema
mc-core checkpoint-validate checkpoint.json
```

### validate-handoff

```mermaid
flowchart TD
    A[Input: findings.json] --> B[Parse JSON]
    B --> C{Schema valid?}
    C -->|No| D[Return error]
    C -->|Yes| E[Check required fields]
    E --> F{task_id present?}
    F -->|No| D
    F -->|Yes| G{worker_id present?}
    G -->|No| D
    G -->|Yes| H{status valid?}
    H -->|No| D
    H -->|Yes| I{findings array?}
    I -->|No| D
    I -->|Yes| J[Return success]
```

### check-gate

```mermaid
flowchart TD
    A[Input: stage name] --> B[Read gates.json]
    B --> C[Read tasks.json]
    C --> D[Get stage tasks]
    D --> E{All tasks complete?}
    E -->|No| F[Return not_ready]
    E -->|Yes| G[Check criteria]
    G --> H{All criteria met?}
    H -->|No| F
    H -->|Yes| I[Return ready]
```

### count-tokens

```mermaid
flowchart TD
    A[Input: file path] --> B[Read file content]
    B --> C[Select encoding]
    C --> D[tiktoken encode]
    D --> E[Count tokens]
    E --> F[Return count]
```

## workflow Crate

Stage state machine and task management.

```mermaid
stateDiagram-v2
    [*] --> Discovery
    Discovery --> Goal : approve_gate()
    Goal --> Requirements : approve_gate()
    Requirements --> Planning : approve_gate()
    Planning --> Design : approve_gate()
    Design --> Implement : approve_gate()
    Implement --> Verify : approve_gate()
    Verify --> Validate : approve_gate()
    Validate --> Document : approve_gate()
    Document --> Release : approve_gate()
    Release --> [*]
```

### Key Types

```rust
pub enum Stage {
    Discovery,
    Goal,
    Requirements,
    Planning,
    Design,
    Implement,
    Verify,
    Validate,
    Document,
    Release,
}

pub enum TaskStatus {
    Pending,
    Ready,
    InProgress,
    Blocked,
    Done,
}

pub struct Task {
    pub id: String,
    pub description: String,
    pub stage: Stage,
    pub status: TaskStatus,
    pub zone: String,
    pub dependencies: Vec<String>,
}

pub struct Gate {
    pub stage: Stage,
    pub status: GateStatus,
    pub criteria: Vec<GateCriterion>,
}

pub struct WorkflowEngine {
    current_stage: Stage,
    tasks: HashMap<String, Task>,
    gates: HashMap<String, Gate>,
}
```

### API

```rust
impl WorkflowEngine {
    pub fn current_stage(&self) -> Stage;
    pub fn can_transition(&self, to: Stage) -> bool;
    pub fn transition(&mut self, to: Stage) -> Result<()>;

    pub fn create_task(&mut self, task: Task) -> Result<String>;
    pub fn update_task_status(&mut self, id: &str, status: TaskStatus) -> Result<()>;
    pub fn get_ready_tasks(&self) -> Vec<&Task>;

    pub fn check_gate(&self, stage: Stage) -> GateCheckResult;
    pub fn approve_gate(&mut self, stage: Stage) -> Result<()>;
}
```

## knowledge Crate

Token counting and budget management.

```mermaid
flowchart LR
    subgraph Input
        FILE[File]
        TEXT[Text]
    end

    subgraph TokenCounter
        ENC[Encoder<br/>cl100k_base]
        COUNT[Count]
    end

    subgraph Budget
        ALLOC[Allocation]
        TRACK[Tracking]
        CHECK[Check]
    end

    FILE --> ENC
    TEXT --> ENC
    ENC --> COUNT
    COUNT --> TRACK
    ALLOC --> CHECK
    TRACK --> CHECK
```

### Key Types

```rust
pub struct TokenCounter {
    encoding: Encoding,
}

pub struct TokenBudget {
    pub king_context: usize,      // ~8000
    pub worker_briefing: usize,   // ~300
    pub handoff_output: usize,    // ~500
}

pub struct Handoff {
    pub task_id: String,
    pub worker_id: String,
    pub status: String,
    pub findings: Vec<Finding>,
    pub artifacts: Vec<String>,
    pub open_questions: Vec<String>,
}

pub struct Finding {
    pub finding_type: String,
    pub summary: String,
}
```

### API

```rust
impl TokenCounter {
    pub fn count(&self, text: &str) -> usize;
    pub fn count_file(&self, path: &Path) -> Result<usize>;
}

impl TokenBudget {
    pub fn check(&self, usage: &TokenUsage) -> BudgetStatus;
    pub fn remaining(&self, component: Component) -> usize;
}
```

## mc-protocol Crate

Shared data structures and file watching.

```mermaid
classDiagram
    class StageState {
        +String current
        +String updated_at
    }

    class TasksState {
        +Vec~Task~ tasks
    }

    class WorkersState {
        +Vec~Worker~ workers
    }

    class GatesState {
        +HashMap~String,Gate~ gates
    }

    class MissionState {
        +StageState stage
        +TasksState tasks
        +WorkersState workers
        +GatesState gates
    }

    MissionState --> StageState
    MissionState --> TasksState
    MissionState --> WorkersState
    MissionState --> GatesState
```

### File Watcher

```rust
pub fn watch_task(
    task_id: &str,
    mission_dir: &str,
    timeout: Duration,
) -> Result<WatchResult>;

pub enum WatchResult {
    Complete { response_path: String },
    Timeout,
}
```

## Building

```bash
# Build all crates
cargo build --release

# Run tests
cargo test

# Run clippy
cargo clippy

# Format code
cargo fmt
```

## Testing

```mermaid
pie title Test Distribution (79 total)
    "workflow" : 35
    "knowledge" : 30
    "mc-protocol" : 14
```

```bash
# All tests
cargo test

# Specific crate
cargo test -p workflow
cargo test -p knowledge
cargo test -p mc-protocol

# With output
cargo test -- --nocapture
```

## Integration with Go

The Go `mc` CLI calls `mc-core` as a subprocess:

```mermaid
sequenceDiagram
    participant Go as mc CLI
    participant Rust as mc-core
    participant FS as .mission/

    Go->>Rust: mc-core validate-handoff file.json
    Rust->>FS: Read file
    Rust->>Rust: Validate
    Rust-->>Go: JSON result

    Go->>Rust: mc-core check-gate design
    Rust->>FS: Read state files
    Rust->>Rust: Check criteria
    Rust-->>Go: JSON result

    Go->>Rust: mc-core checkpoint-compile checkpoint.json
    Rust->>FS: Read checkpoint
    Rust->>Rust: Compile briefing
    Rust-->>Go: Markdown briefing

    Go->>Rust: mc-core count-tokens file.md
    Rust->>FS: Read file
    Rust->>Rust: Count tokens
    Rust-->>Go: Token count
```

## Error Handling

All errors are returned as JSON for easy parsing:

```json
{
  "error": true,
  "message": "Missing required field: task_id",
  "code": "VALIDATION_ERROR"
}
```

```rust
pub enum CoreError {
    ValidationError(String),
    FileNotFound(PathBuf),
    ParseError(String),
    StateError(String),
}
```

## Performance

Token counting benchmarks:

| File Size | Time |
|-----------|------|
| 1 KB | < 1ms |
| 10 KB | ~2ms |
| 100 KB | ~15ms |
| 1 MB | ~150ms |

The tiktoken-rs library provides fast BPE encoding compatible with Claude's tokenizer.