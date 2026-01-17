# MissionControl — Architecture Spec

## Overview

MissionControl is a multi-agent orchestration system built on a **layers × domains** architecture. Four vertical domains handle distinct responsibilities, while three horizontal layers handle presentation, API, and core logic.

---

## Architecture Model

```
                         LAYERS
                           │
     ┌─────────────────────┼─────────────────────┐
     │                     │                     │
     ▼                     ▼                     ▼
┌─────────┐          ┌──────────┐          ┌──────────┐
│   UI    │   ────►  │   API    │   ────►  │   CORE   │
│ (React) │          │   (Go)   │          │(Rust/LLM)│
└─────────┘          └──────────┘          └──────────┘
                           │
          ┌────────────────┼────────────────┬────────────────┐
          ▼                ▼                ▼                ▼
     ┌─────────┐     ┌──────────┐    ┌───────────┐    ┌─────────┐
     │STRATEGY │     │ WORKFLOW │    │ KNOWLEDGE │    │ RUNTIME │
     └─────────┘     └──────────┘    └───────────┘    └─────────┘
     
                         DOMAINS
```

---

## Domains

### 1. Strategy

**Responsibility:** High-level decisions requiring judgment.

| Aspect | Detail |
|--------|--------|
| Questions answered | Is this spec ready? Should we split this task? How to handle a stuck worker? |
| Intelligence | LLM-driven (Opus) |
| Owner | King |
| State | Decisions, approvals, conversation history |

**What belongs here:**
- User conversation
- Phase gate approvals
- Conflict resolution
- Judgment calls on ambiguous situations
- Finding synthesis when workers disagree

**What doesn't belong here:**
- Task routing (Workflow)
- Briefing compilation (Knowledge)
- Process management (Runtime)

---

### 2. Workflow

**Responsibility:** Where we are in the process.

| Aspect | Detail |
|--------|--------|
| Questions answered | What phase are we in? Is this gate satisfied? What's the next task? |
| Intelligence | Deterministic state machine |
| Owner | Workflow Engine |
| State | Phase, gate status, task dependencies, progress |

**What belongs here:**
- Phase transitions (Idea → Design → Implement → ...)
- Gate status computation
- Task state management (pending/in_progress/done)
- Dependency graph between tasks
- Progress aggregation from TODOs

**What doesn't belong here:**
- Deciding if work is good enough (Strategy)
- What context to pass (Knowledge)
- Spawning processes (Runtime)

---

### 3. Knowledge

**Responsibility:** What we know and how to share it.

| Aspect | Detail |
|--------|--------|
| Questions answered | What does this worker need to know? What changed since last checkpoint? Are we over token budget? |
| Intelligence | Rust (storage, counting, validation) + LLM (distillation) |
| Owner | Knowledge Manager |
| State | Specs, findings, checkpoints, deltas, token budgets |

**What belongs here:**
- Spec storage and retrieval
- Findings accumulation
- Briefing compilation (spec → task-specific context)
- Handoff content management
- Checkpoint/delta versioning
- Token counting and budget enforcement
- Context pruning (stale, duplicate, expired)

**What doesn't belong here:**
- Deciding what to build (Strategy)
- Tracking task completion (Workflow)
- Spawning workers (Runtime)

---

### 4. Runtime

**Responsibility:** How agents execute.

| Aspect | Detail |
|--------|--------|
| Questions answered | Is this worker healthy? How do I spawn a Developer in the Backend zone? |
| Intelligence | Deterministic process management |
| Owner | Orchestrator |
| State | Processes, connections, resource allocation |

**What belongs here:**
- Process spawning and killing
- Worker health monitoring
- Message routing between agents
- WebSocket connections
- Zone assignment
- Resource allocation

**What doesn't belong here:**
- Deciding when to spawn (Strategy/Workflow)
- What to tell the worker (Knowledge)
- Whether the task is done (Workflow)

---

## Layers

### UI Layer (React)

**Tech:** React 18, TypeScript, Tailwind, Zustand, Three.js

**Responsibilities per domain:**

| Domain | UI Components |
|--------|---------------|
| Strategy | King chat panel, approval dialogs, decision prompts |
| Workflow | Phase progress view, gate status indicators, TODO display |
| Knowledge | Spec viewer, findings browser, token usage display, briefing preview |
| Runtime | Agent cards, org view, 3D visualization, spawn dialog, zone manager |

**Key screens:**
- Dashboard (overview of all domains)
- King conversation (Strategy focus)
- Project view (Workflow focus)
- Agent detail (Runtime + Knowledge)
- Settings (personas, MCPs, preferences)

---

### API Layer (Go)

**Tech:** Go, gorilla/mux, gorilla/websocket

**Responsibilities per domain:**

| Domain | Endpoints |
|--------|-----------|
| Strategy | `POST /api/king/message`, `POST /api/gates/:id/approve` |
| Workflow | `GET /api/phases`, `GET /api/tasks`, `PUT /api/tasks/:id/status` |
| Knowledge | `GET /api/specs/:id`, `GET /api/briefings/:worker`, `POST /api/handoffs` |
| Runtime | `POST /api/agents`, `DELETE /api/agents/:id`, `GET /api/agents`, WebSocket `/ws` |

**WebSocket events:**

```typescript
// Strategy
{ type: "king_message", content: string, role: "user" | "assistant" }
{ type: "approval_requested", gate_id: string, message: string }

// Workflow  
{ type: "phase_changed", phase: Phase }
{ type: "task_updated", task_id: string, status: TaskStatus }
{ type: "gate_status", gate_id: string, ready: boolean }

// Knowledge
{ type: "token_warning", worker_id: string, usage: number, budget: number }
{ type: "checkpoint_created", checkpoint_id: string }
{ type: "findings_updated", task_id: string }

// Runtime
{ type: "agent_spawned", agent: Agent }
{ type: "agent_status", agent_id: string, status: AgentStatus }
{ type: "agent_health", agent_id: string, health: HealthStatus }
{ type: "tool_call", agent_id: string, tool: string, args: object }
```

---

### Core Layer (Rust + LLM)

**Tech:** Rust (deterministic logic), Claude API (LLM calls)

**Responsibilities per domain:**

| Domain | Core Components | Tech |
|--------|-----------------|------|
| Strategy | King agent | LLM (Opus) |
| Workflow | State machine, phase transitions, dependency resolver | Rust |
| Knowledge | Token counter, checkpoint manager, delta engine, pruner, briefing compiler | Rust + LLM (Sonnet) |
| Runtime | Health monitor, stream parser | Rust (+ Go for process mgmt) |

---

## Domain Interactions

### Flow: User requests a feature

```
┌─────┐
│ YOU │ "Build a login page"
└──┬──┘
   │
   ▼
┌──────────┐
│ STRATEGY │ King: "Let me understand requirements..."
│  (King)  │ Conversation until spec is clear
└────┬─────┘
     │ "Spec ready, proceed to Design"
     ▼
┌──────────┐
│ WORKFLOW │ Transition: Idea → Design
│ (Engine) │ Create tasks: [design-ui, design-api]
└────┬─────┘
     │ Task: design-ui ready
     ▼
┌───────────┐
│ KNOWLEDGE │ Compile briefing for Designer
│ (Manager) │ Spec → 300 token briefing
└─────┬─────┘
      │ Briefing ready
      ▼
┌─────────┐
│ RUNTIME │ Spawn Designer worker
│ (Orch)  │ Assign to Frontend zone
└────┬────┘
     │
     ▼
┌──────────┐
│ DESIGNER │ Creates mockups
│ (Worker) │ Outputs findings
└────┬─────┘
     │ Findings (structured JSON)
     ▼
┌───────────┐
│ KNOWLEDGE │ Validate handoff, compute delta
│ (Manager) │ Store findings, update checkpoint
└─────┬─────┘
      │ Findings stored
      ▼
┌──────────┐
│ WORKFLOW │ Mark design-ui complete
│ (Engine) │ Check gate: Design phase done?
└────┬─────┘
     │ Gate status
     ▼
┌──────────┐
│ STRATEGY │ King: "Design complete. Ready for Implement?"
│  (King)  │ Await user approval
└──────────┘
```

### Flow: Worker context bloating

```
┌───────────┐
│ KNOWLEDGE │ Token counter: Worker at 75% budget
│ (Manager) │ Emit warning
└─────┬─────┘
      │ token_warning event
      ▼
┌──────────┐
│ STRATEGY │ King decides: force handoff
│  (King)  │ (or auto-rule triggers)
└────┬─────┘
     │ handoff decision
     ▼
┌─────────┐
│ RUNTIME │ Signal worker to wrap up
│ (Orch)  │
└────┬────┘
     │
     ▼
┌──────────┐
│ WORKER   │ Outputs findings, dies
└────┬─────┘
     │ findings
     ▼
┌───────────┐
│ KNOWLEDGE │ Validate, compute delta
│ (Manager) │ Compile fresh briefing
└─────┬─────┘
      │ briefing
      ▼
┌─────────┐
│ RUNTIME │ Spawn fresh worker with briefing
│ (Orch)  │
└─────────┘
```

---

## Tech Stack by Layer × Domain

|  | Strategy | Workflow | Knowledge | Runtime |
|--|----------|----------|-----------|---------|
| **UI** | React | React | React | React + Three.js |
| **API** | Go | Go | Go | Go |
| **Core** | LLM (Opus) | Rust | Rust + LLM (Sonnet) | Go + Rust |

---

## Rust Core Components

### Workflow Engine

```rust
pub struct WorkflowEngine {
    phases: Vec<Phase>,
    current_phase: PhaseId,
    tasks: HashMap<TaskId, Task>,
    gates: HashMap<GateId, Gate>,
}

impl WorkflowEngine {
    // Phase management
    fn current_phase(&self) -> &Phase;
    fn can_transition(&self, to: PhaseId) -> bool;
    fn transition(&mut self, to: PhaseId) -> Result<(), WorkflowError>;
    
    // Task management
    fn create_task(&mut self, task: Task) -> TaskId;
    fn update_task_status(&mut self, id: TaskId, status: TaskStatus);
    fn get_ready_tasks(&self) -> Vec<&Task>;
    fn resolve_dependencies(&self, task_id: TaskId) -> Vec<TaskId>;
    
    // Gate management
    fn check_gate(&self, gate_id: GateId) -> GateStatus;
    fn compute_gate_status(&self, gate_id: GateId) -> bool;
}

pub enum Phase {
    Idea,
    Design,
    Implement,
    Verify,
    Document,
    Release,
}

pub enum TaskStatus {
    Pending,
    Ready,        // Dependencies met
    InProgress,
    Blocked(String),
    Done,
}

pub enum GateStatus {
    Open,         // Can proceed
    Closed,       // Requirements not met
    AwaitingApproval, // Ready but needs user OK
}
```

### Knowledge Manager

```rust
pub struct KnowledgeManager {
    specs: SpecStore,
    findings: FindingsStore,
    checkpoints: CheckpointStore,
    token_budgets: HashMap<WorkerId, TokenBudget>,
}

impl KnowledgeManager {
    // Token management
    fn count_tokens(&self, text: &str) -> usize;
    fn get_budget(&self, worker_id: &WorkerId) -> &TokenBudget;
    fn check_budget(&self, worker_id: &WorkerId) -> BudgetStatus;
    fn record_usage(&mut self, worker_id: &WorkerId, tokens: usize);
    
    // Handoff management
    fn validate_handoff(&self, handoff: &Handoff) -> Result<(), ValidationError>;
    fn compute_delta(&self, findings: &Findings, checkpoint_id: &CheckpointId) -> Delta;
    fn store_findings(&mut self, task_id: &TaskId, findings: Findings);
    
    // Checkpoint management
    fn create_checkpoint(&mut self, state: &ProjectState) -> CheckpointId;
    fn restore_checkpoint(&self, id: &CheckpointId) -> ProjectState;
    fn get_deltas_since(&self, checkpoint_id: &CheckpointId) -> Vec<Delta>;
    
    // Briefing compilation (returns inputs for LLM)
    fn compile_briefing_inputs(
        &self,
        task: &Task,
        checkpoint: &Checkpoint,
        deltas: &[Delta],
    ) -> BriefingInputs;
    
    // Pruning
    fn prune_stale(&mut self, context: &mut Context, max_age: Duration);
    fn prune_duplicates(&mut self, context: &mut Context);
    fn prune_superseded(&mut self, context: &mut Context, decisions: &[Decision]);
}

pub enum BudgetStatus {
    Healthy,                    // < 50%
    Warning { remaining: usize }, // 50-75%
    Critical { remaining: usize }, // > 75%
    Exceeded,                   // Over budget
}

pub struct Handoff {
    task_id: TaskId,
    status: HandoffStatus,
    findings: Vec<Finding>,
    artifacts: Vec<PathBuf>,
    open_questions: Vec<String>,
    context_for_successor: Option<SuccessorContext>,
}

pub struct Finding {
    finding_type: FindingType,
    summary: String,           // 1-2 sentences
    details_path: Option<PathBuf>,
}

pub enum FindingType {
    Discovery,
    Blocker,
    Decision,
    Concern,
}
```

### Health Monitor

```rust
pub struct HealthMonitor {
    workers: HashMap<WorkerId, WorkerHealth>,
    check_interval: Duration,
}

impl HealthMonitor {
    fn check_worker(&self, worker_id: &WorkerId) -> HealthStatus;
    fn detect_stuck(&self, worker_id: &WorkerId, timeout: Duration) -> bool;
    fn get_last_activity(&self, worker_id: &WorkerId) -> Option<Instant>;
    fn mark_activity(&mut self, worker_id: &WorkerId);
}

pub enum HealthStatus {
    Healthy,
    Idle { since: Instant },
    Stuck { since: Instant },
    Unresponsive,
    Dead,
}
```

---

## Go API Components

### Strategy Routes

```go
// POST /api/king/message
type KingMessageRequest struct {
    Content string `json:"content"`
}

type KingMessageResponse struct {
    Content  string `json:"content"`
    Actions  []KingAction `json:"actions,omitempty"`
}

// POST /api/gates/:id/approve
type GateApprovalRequest struct {
    Approved bool   `json:"approved"`
    Comment  string `json:"comment,omitempty"`
}
```

### Workflow Routes

```go
// GET /api/phases
type PhasesResponse struct {
    Current Phase   `json:"current"`
    Phases  []Phase `json:"phases"`
}

// GET /api/tasks
type TasksResponse struct {
    Tasks []Task `json:"tasks"`
}

// PUT /api/tasks/:id/status
type TaskStatusUpdate struct {
    Status TaskStatus `json:"status"`
}
```

### Knowledge Routes

```go
// GET /api/specs/:id
type SpecResponse struct {
    ID      string `json:"id"`
    Content string `json:"content"`
    Version int    `json:"version"`
}

// GET /api/briefings/:worker_id
type BriefingResponse struct {
    WorkerID string `json:"worker_id"`
    TaskID   string `json:"task_id"`
    Content  string `json:"content"`
    Tokens   int    `json:"tokens"`
}

// POST /api/handoffs
type HandoffRequest struct {
    WorkerID      string    `json:"worker_id"`
    TaskID        string    `json:"task_id"`
    Status        string    `json:"status"`
    Findings      []Finding `json:"findings"`
    Artifacts     []string  `json:"artifacts"`
    OpenQuestions []string  `json:"open_questions,omitempty"`
}
```

### Runtime Routes

```go
// POST /api/agents
type SpawnRequest struct {
    Persona   string `json:"persona"`
    Task      string `json:"task"`
    Zone      string `json:"zone"`
    WorkDir   string `json:"workdir,omitempty"`
}

// GET /api/agents
type AgentsResponse struct {
    Agents []Agent `json:"agents"`
}

// DELETE /api/agents/:id
// No body, returns 204

// WebSocket /ws
// Bidirectional event stream
```

---

## File Structure

```
mission-control/
├── SPEC.md
├── ARCHITECTURE.md          # This file
│
├── ui/                      # Presentation Layer
│   └── web/
│       ├── package.json
│       ├── src/
│       │   ├── App.tsx
│       │   │
│       │   ├── domains/
│       │   │   ├── strategy/
│       │   │   │   ├── KingChat.tsx
│       │   │   │   ├── ApprovalDialog.tsx
│       │   │   │   └── hooks/useKing.ts
│       │   │   │
│       │   │   ├── workflow/
│       │   │   │   ├── PhaseView.tsx
│       │   │   │   ├── TaskList.tsx
│       │   │   │   ├── GateStatus.tsx
│       │   │   │   └── hooks/useWorkflow.ts
│       │   │   │
│       │   │   ├── knowledge/
│       │   │   │   ├── SpecViewer.tsx
│       │   │   │   ├── FindingsBrowser.tsx
│       │   │   │   ├── TokenUsage.tsx
│       │   │   │   └── hooks/useKnowledge.ts
│       │   │   │
│       │   │   └── runtime/
│       │   │       ├── AgentCard.tsx
│       │   │       ├── OrgView.tsx
│       │   │       ├── Scene3D.tsx
│       │   │       ├── SpawnDialog.tsx
│       │   │       └── hooks/useRuntime.ts
│       │   │
│       │   ├── components/      # Shared components
│       │   ├── stores/          # Zustand stores
│       │   └── types/           # TypeScript types
│       │
│       └── public/
│
├── api/                     # API Layer
│   └── orchestrator/
│       ├── go.mod
│       ├── main.go
│       │
│       ├── strategy/
│       │   ├── routes.go
│       │   └── king.go
│       │
│       ├── workflow/
│       │   └── routes.go
│       │
│       ├── knowledge/
│       │   └── routes.go
│       │
│       ├── runtime/
│       │   ├── routes.go
│       │   ├── manager.go
│       │   └── ws/
│       │       └── hub.go
│       │
│       └── middleware/
│
├── core/                    # Core Layer
│   ├── workflow/            # Rust
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs
│   │       ├── engine.rs
│   │       ├── phase.rs
│   │       ├── task.rs
│   │       └── gate.rs
│   │
│   ├── knowledge/           # Rust
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs
│   │       ├── manager.rs
│   │       ├── tokens.rs
│   │       ├── checkpoint.rs
│   │       ├── delta.rs
│   │       ├── handoff.rs
│   │       ├── pruning.rs
│   │       └── briefing.rs
│   │
│   ├── runtime/             # Rust (monitoring only)
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs
│   │       ├── health.rs
│   │       └── stream.rs    # Existing stream parser
│   │
│   └── ffi/                 # FFI bindings for Go
│       ├── Cargo.toml
│       └── src/
│           └── lib.rs
│
├── agents/                  # Python agents
│   ├── v0_minimal.py
│   ├── v1_basic.py
│   ├── v2_todo.py
│   └── v3_subagent.py
│
└── .mission/                # Project state (per-project)
    ├── config.md
    ├── ideas/
    ├── specs/
    ├── mockups/
    ├── progress/
    ├── reviews/
    ├── checkpoints/
    ├── handoffs/
    └── releases/
```

---

## Token Efficiency Strategy

### Principle: Files are truth, briefings are context

```
┌─────────────────────────────────────────────────────────────┐
│  SOURCE OF TRUTH (files in .mission/)                       │
│  Complete specs, full history, git-tracked                  │
│  Tokens: 2000+                                              │
└─────────────────────────────────────────────────────────────┘
                      │
                      ▼ Knowledge Manager compiles
┌─────────────────────────────────────────────────────────────┐
│  BRIEFING (what worker receives)                            │
│  - Task description                                         │
│  - Key requirements (3-5 bullets)                           │
│  - Relevant decisions                                       │
│  - File paths for deep-dive                                 │
│  Tokens: ~300                                               │
└─────────────────────────────────────────────────────────────┘
```

### Token Budget Enforcement

| Threshold | Status | Action |
|-----------|--------|--------|
| < 50% | Healthy | Continue |
| 50-75% | Warning | Alert, consider handoff |
| > 75% | Critical | Prepare handoff |
| > 90% | Exceeded | Force handoff |

### Handoff Flow

```
Worker nearing budget
        │
        ▼
Worker outputs structured findings (JSON)
        │
        ▼
Rust validates against schema
        │
        ├── Invalid → Reject, worker retries
        │
        ▼ Valid
Rust computes delta from checkpoint
        │
        ▼
Delta + findings stored in .mission/handoffs/
        │
        ▼
Worker dies
        │
        ▼
New worker spawns
        │
        ▼
Rust compiles: checkpoint + deltas → briefing inputs
        │
        ▼
LLM (Sonnet) generates briefing
        │
        ▼
New worker receives lean briefing
```

### Pruning Rules (Rust)

| Rule | Trigger | Action |
|------|---------|--------|
| Stale | Tool output > N turns old | Remove from context |
| Duplicate | Same file read twice | Keep latest only |
| Superseded | Decision changed | Remove old reasoning |
| Completed | Subtask done | Collapse to summary |

---

## Model Allocation

| Role | Model | Rationale |
|------|-------|-----------|
| King | Claude Opus | Strategic judgment, synthesis |
| Briefing generation | Claude Sonnet | Distillation, structured output |
| Designer | Claude Sonnet | Creative, iterative |
| Architect | Claude Sonnet | System design |
| Developer | Claude Sonnet | Implementation |
| Reviewer | Claude Haiku | Pattern matching, checklists |
| Security | Claude Sonnet | Vulnerability analysis |
| Tester | Claude Haiku | Test generation |
| QA | Claude Haiku | E2E validation |
| Docs | Claude Haiku | Templated writing |
| DevOps | Claude Haiku | Config generation |

---

## Implementation Phases

### Phase 1: Foundation
- [ ] Rust workflow engine (state machine, tasks, gates)
- [ ] Rust knowledge manager (tokens, checkpoints, validation)
- [ ] Go API routes for all four domains
- [ ] React domain structure scaffolding

### Phase 2: Core Loop
- [ ] King integration (Opus)
- [ ] Briefing compilation (Rust + Sonnet)
- [ ] Worker spawning with briefings
- [ ] Handoff validation and storage

### Phase 3: UI
- [ ] King chat panel
- [ ] Phase/workflow view
- [ ] Agent cards and org view
- [ ] Token usage display

### Phase 4: Efficiency
- [ ] Token budget enforcement
- [ ] Auto-handoff triggers
- [ ] Pruning engine
- [ ] Delta computation

### Phase 5: Polish
- [ ] 3D visualization
- [ ] Settings/persona management
- [ ] Error handling and recovery
- [ ] Documentation

---

## Summary

| Aspect | Decision |
|--------|----------|
| Architecture | Layers (UI/API/Core) × Domains (Strategy/Workflow/Knowledge/Runtime) |
| UI | React + TypeScript + Tailwind + Three.js |
| API | Go |
| Core | Rust (deterministic) + LLM (judgment) |
| Token strategy | Files = truth, briefings = context, aggressive pruning |
| Handoffs | Structured JSON schemas, validated by Rust |
| Model allocation | Opus for strategy, Sonnet for complex work, Haiku for simple work |