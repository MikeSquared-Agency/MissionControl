# V4 Implementation Guide

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
- [ ] `cargo new core/workflow --lib`
- [ ] Phase enum with `next()` method
- [ ] TaskStatus enum (Pending, Ready, InProgress, Blocked, Done)
- [ ] Task struct with dependencies vec
- [ ] Gate struct with criteria vec
- [ ] WorkflowEngine with HashMap storage
- [ ] Implement: `current_phase()`, `can_transition()`, `transition()`
- [ ] Implement: `create_task()`, `update_task_status()`, `get_ready_tasks()`
- [ ] Implement: `check_gate()`, `approve_gate()`
- [ ] JSON serialization via serde
- [ ] Unit tests for state transitions

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
- [ ] `cargo new core/knowledge --lib`
- [ ] Add `tiktoken-rs` dependency
- [ ] TokenCounter wrapper struct
- [ ] TokenBudget with thresholds (50%, 75%, 90%)
- [ ] BudgetStatus enum (Healthy, Warning, Critical, Exceeded)
- [ ] Finding struct with FindingType enum
- [ ] Handoff struct with validation
- [ ] `validate_handoff()` - schema enforcement
- [ ] Checkpoint struct (full state snapshot)
- [ ] Delta struct (changes since checkpoint)
- [ ] `compute_delta()` - diff engine
- [ ] `compile_briefing_inputs()` - gather context for LLM
- [ ] Unit tests

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
- [ ] `cargo new core/runtime --lib`
- [ ] HealthStatus enum (Healthy, Idle, Stuck, Unresponsive, Dead)
- [ ] WorkerHealth struct with timestamps
- [ ] HealthMonitor with configurable thresholds
- [ ] `register_worker()`, `unregister_worker()`
- [ ] `mark_activity()`, `mark_tool_call()`
- [ ] `check_health()`, `get_stuck_workers()`
- [ ] Unit tests
- [ ] Move existing stream parser from `stream-parser/` crate

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
- [ ] `cargo new core/ffi --lib`
- [ ] Set `crate-type = ["cdylib"]` in Cargo.toml
- [ ] C-compatible exports for WorkflowEngine
- [ ] C-compatible exports for KnowledgeManager
- [ ] C-compatible exports for HealthMonitor
- [ ] JSON string passing (avoid complex FFI types)
- [ ] Build script for `.so`/`.dylib` output
- [ ] Test FFI from C (optional sanity check)

**Reference:** `V4-RUST-CONTRACTS.md` → FFI Bindings section

---

### Phase 3: Go Integration (Depends on Phase 2)

**TODO:**
- [ ] CGO bindings to load Rust shared library
- [ ] Wrapper functions in Go for each FFI export
- [ ] Add Strategy routes: `POST /api/gates/:id/approve`
- [ ] Add Workflow routes: `GET /api/phases`, `GET /api/tasks`, `PUT /api/tasks/:id/status`
- [ ] Add Knowledge routes: `GET /api/specs/:id`, `GET /api/briefings/:worker`, `POST /api/handoffs`
- [ ] Integration tests with Rust core

**Reference:** `V4-API-ROUTES.md` → Full endpoint specs

---

### Phase 4: React Updates (Parallel with Phase 3)

**TODO:**
- [ ] Create `src/domains/` folder structure:
  - [ ] `domains/strategy/`
  - [ ] `domains/workflow/`
  - [ ] `domains/knowledge/`
  - [ ] `domains/runtime/`
- [ ] Move existing components to appropriate domains
- [ ] Add TypeScript types for Phase, Task, Gate, etc.
- [ ] PhaseView component (workflow domain)
- [ ] TokenUsage component (knowledge domain)
- [ ] GateApproval component (strategy domain)
- [ ] WebSocket handlers for new events

**Reference:** `V4-API-ROUTES.md` → WebSocket Events section

---

### Phase 5: Integration Testing

**TODO:**
- [ ] Create task via API → verify in UI
- [ ] Update task status → verify WebSocket event
- [ ] Phase transition → verify gate status
- [ ] Token warning flow end-to-end
- [ ] Handoff validation flow

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