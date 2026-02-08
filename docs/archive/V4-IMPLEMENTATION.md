# V4 Implementation Guide

> **Superseded by v6**: The 6-phase workflow described here has been replaced by a 10-stage workflow. `Phase` → `Stage`, `phase.json` → `stage.json`. See `ARCHITECTURE.md` and `CHANGELOG.md` for current state.

The actionable roadmap for v4 (Architecture Foundation). References existing specs rather than duplicating them.

---

## Quick Reference

| Topic | Source File |
|-------|-------------|
| Architecture (layers × domains) | `ARCHITECTURE-SPEC.md` |
| Domain responsibilities | `ARCHITECTURE-SPEC.md` → Domains section |
| Rust struct signatures | `V4-RUST-CONTRACTS.md` |
| API routes & WebSocket events | `V4-API-ROUTES.md` |
| Token efficiency strategy | `TOKEN-RESEARCH.md` |
| Workflow phases & gates | `PERSONAS-SPEC.md` → Phases & Gates |
| Worker personas | `PERSONAS-SPEC.md` → Persona Table |
| Handoff schema | `TOKEN-RESEARCH.md` → Layer 3 |
| File structure (.mission/) | `SPEC.md` → File Structure |

---

## Implementation Sequence

### Phase 1: Rust Foundation (No Dependencies)

These three crates can be built in parallel - no interdependencies.

#### 1.1 Workflow Engine
```
core/workflow/
├── Cargo.toml
└── src/
    ├── lib.rs
    ├── phase.rs    # Phase enum
    ├── task.rs     # Task + TaskStatus
    ├── gate.rs     # Gate + GateCriterion
    └── engine.rs   # WorkflowEngine state machine
```

**TODO:**
- [x] `cargo new core/workflow --lib`
- [x] Phase enum with `next()` method
- [x] TaskStatus enum (Pending, Ready, InProgress, Blocked, Done)
- [x] Task struct with dependencies vec
- [x] Gate struct with criteria vec
- [x] WorkflowEngine with HashMap storage
- [x] Implement: `current_phase()`, `can_transition()`, `transition()`
- [x] Implement: `create_task()`, `update_task_status()`, `get_ready_tasks()`
- [x] Implement: `check_gate()`, `approve_gate()`
- [x] JSON serialization via serde
- [x] Unit tests for state transitions

**Reference:** `V4-RUST-CONTRACTS.md` → WorkflowEngine section

---

#### 1.2 Knowledge Manager
```
core/knowledge/
├── Cargo.toml
└── src/
    ├── lib.rs
    ├── tokens.rs      # TokenCounter using tiktoken-rs
    ├── budget.rs      # TokenBudget + BudgetStatus
    ├── handoff.rs     # Handoff + Finding structs
    ├── checkpoint.rs  # Checkpoint serialization
    ├── delta.rs       # Delta computation
    └── manager.rs     # KnowledgeManager orchestration
```

**TODO:**
- [x] `cargo new core/knowledge --lib`
- [x] Add `tiktoken-rs` dependency
- [x] TokenCounter wrapper struct
- [x] TokenBudget with thresholds (50%, 75%, 90%)
- [x] BudgetStatus enum (Healthy, Warning, Critical, Exceeded)
- [x] Finding struct with FindingType enum
- [x] Handoff struct with validation
- [x] `validate_handoff()` - schema enforcement
- [x] Checkpoint struct (full state snapshot)
- [x] Delta struct (changes since checkpoint)
- [x] `compute_delta()` - diff engine
- [x] `compile_briefing_inputs()` - gather context for LLM
- [x] Unit tests

**Reference:** `V4-RUST-CONTRACTS.md` → KnowledgeManager section
**Reference:** `TOKEN-RESEARCH.md` → Handoff schema details

---

#### 1.3 Health Monitor
```
core/runtime/
├── Cargo.toml
└── src/
    ├── lib.rs
    ├── health.rs   # HealthMonitor + HealthStatus
    └── stream.rs   # (existing stream parser, move here)
```

**TODO:**
- [x] `cargo new core/runtime --lib`
- [x] HealthStatus enum (Healthy, Idle, Stuck, Unresponsive, Dead)
- [x] WorkerHealth struct with timestamps
- [x] HealthMonitor with configurable thresholds
- [x] `register_worker()`, `unregister_worker()`
- [x] `mark_activity()`, `mark_tool_call()`
- [x] `check_health()`, `get_stuck_workers()`
- [x] Unit tests
- [x] Move existing stream parser from `stream-parser/` crate

**Reference:** `V4-RUST-CONTRACTS.md` → HealthMonitor section

---

### Phase 2: FFI Layer (Depends on Phase 1)

```
core/ffi/
├── Cargo.toml
└── src/
    └── lib.rs
```

**TODO:**
- [x] `cargo new core/ffi --lib`
- [x] Set `crate-type = ["cdylib"]` in Cargo.toml
- [x] C-compatible exports for WorkflowEngine
- [x] C-compatible exports for KnowledgeManager
- [x] C-compatible exports for HealthMonitor
- [x] JSON string passing (avoid complex FFI types)
- [x] Build script for `.so`/`.dylib` output
- [ ] Test FFI from C (optional sanity check)

**Reference:** `V4-RUST-CONTRACTS.md` → FFI Bindings section

---

### Phase 3: Go Integration (Depends on Phase 2)

**TODO:**
- [ ] CGO bindings to load Rust shared library
- [ ] Wrapper functions in Go for each FFI export
- [x] Add Strategy routes: `POST /api/gates/:id/approve`
- [x] Add Workflow routes: `GET /api/phases`, `GET /api/tasks`, `PUT /api/tasks/:id/status`
- [x] Add Knowledge routes: `POST /api/handoffs`, `GET /api/checkpoints`
- [ ] Integration tests with Rust core

**Note:** Routes implemented with in-memory Go store. CGO integration deferred.

**Reference:** `V4-API-ROUTES.md` → Full endpoint specs

---

### Phase 4: React Updates (Parallel with Phase 3)

**TODO:**
- [x] Create `src/domains/` folder structure:
  - [x] `domains/strategy/`
  - [x] `domains/workflow/`
  - [x] `domains/knowledge/`
  - [x] `domains/runtime/`
- [ ] Move existing components to appropriate domains
- [x] Add TypeScript types for Phase, Task, Gate, etc.
- [x] PhaseView component (workflow domain)
- [x] TokenUsage component (knowledge domain)
- [x] GateApproval component (strategy domain)
- [x] WebSocket handlers for new events

**Reference:** `V4-API-ROUTES.md` → WebSocket Events section

---

### Phase 5: Integration Testing

**TODO:**
- [x] Create task via API → verify in UI
- [x] Update task status → verify WebSocket event
- [x] Phase transition → verify gate status
- [x] Token warning flow end-to-end
- [x] Handoff validation flow

**Test file:** `orchestrator/v4/routes_test.go` (14 tests, all passing)

---

## Decision Points

### FFI vs Subprocess?
**Decision:** FFI (shared library)  
**Rationale:** Lower latency, shared memory, tighter integration  
**Trade-off:** Harder to debug, but performance wins for hot path

### Token Counting Model?
**Decision:** tiktoken-rs with cl100k_base  
**Rationale:** GPT-4/Claude compatible, exact match not critical  
**Note:** Consistency matters more than precision

### Checkpoint Frequency?
**Options:**
1. After each phase gate
2. After N handoffs
3. On explicit save

**Recommendation:** Start with option 1 (phase gates), add option 2 later

---

## File Locations After v4

```
mission-control/
├── core/                    # NEW - Rust core
│   ├── workflow/
│   ├── knowledge/
│   ├── runtime/
│   └── ffi/
│
├── orchestrator/            # UPDATED - Go API
│   ├── strategy/           # NEW
│   ├── workflow/           # NEW
│   ├── knowledge/          # NEW
│   └── runtime/            # Existing + updates
│
└── web/src/                 # UPDATED - React
    └── domains/             # NEW - domain organization
        ├── strategy/
        ├── workflow/
        ├── knowledge/
        └── runtime/
```

---

## What's Deferred to v5

- King agent full implementation (Opus integration)
- Briefing generation via LLM (Sonnet)
- Worker spawning with briefings
- Finding synthesis
- Gate approval logic beyond simple approve/reject

See `TODO.md` → v5 section for full list.