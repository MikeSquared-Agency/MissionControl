# MissionControl — TODO

## Getting Started (v5)

1. Read V5-IMPLEMENTATION.md for detailed specs
2. Start with `mc init` command - it's the foundation
3. Reference V4-RUST-CONTRACTS.md for Rust structs
4. Reference PERSONAS-SPEC.md for persona details

## Completed

### v1: Agent Fundamentals ✅
- [x] v0_minimal.py (~50 lines, bash only)
- [x] v1_basic.py (~200 lines, full tools)
- [x] v2_todo.py (~300 lines, explicit planning)
- [x] v3_subagent.py (~450 lines, child agents)

### v2: Orchestrator ✅
- [x] Go process manager (spawn/kill agents)
- [x] REST API endpoints
- [x] WebSocket event hub
- [x] Rust stream parser

### v3: 2D Dashboard ✅
- [x] Zustand state + persistence
- [x] Header, Sidebar, AgentCard, AgentPanel
- [x] Zone System (CRUD, split/merge)
- [x] Persona System (defaults + custom)
- [x] King Mode UI (KingPanel, KingHeader)
- [x] Attention System (notifications)
- [x] Settings Panel + keyboard shortcuts
- [x] 81 unit tests

### v4: Rust Core ✅
- [x] Workflow engine (phases, gates, tasks)
- [x] Knowledge manager (tokens, checkpoints, validation)
- [x] Health monitor (stuck detection)
- [x] Struct definitions and logic

---

## Current: v5 — King + mc CLI

The brain of MissionControl. Make King actually orchestrate.

### Phase 1: mc CLI Foundation

```
mc
├── init       # Create .mission/ scaffold
├── spawn      # Spawn worker process
├── kill       # Kill worker process
├── status     # JSON dump of state
├── workers    # List active workers
├── handoff    # Validate and store handoff
├── gate       # Check/approve gates
├── phase      # Get/set phase
├── task       # CRUD for tasks
└── serve      # Start Go bridge + UI
```

**TODO:**
- [ ] Create cmd/mc/ with cobra
- [ ] `mc init` - Create .mission/ directory structure
- [ ] `mc status` - Read and dump .mission/state/*.json
- [ ] `mc phase` - Get current phase
- [ ] `mc phase next` - Transition phase
- [ ] `mc task create <name> --phase <p> --zone <z> --persona <p>`
- [ ] `mc task list [--phase <p>]`
- [ ] `mc task update <id> --status <s>`
- [ ] `mc workers` - List active workers from state
- [ ] `mc spawn <persona> <task> --zone <zone>` - Spawn Claude Code process
- [ ] `mc kill <worker-id>` - Kill worker process
- [ ] `mc handoff <file>` - Validate JSON, store in .mission/
- [ ] `mc gate check <phase>` - Check gate criteria
- [ ] `mc gate approve <phase>` - Approve and transition
- [ ] `mc serve` - Start Go bridge (WebSocket + file watcher)

### Phase 2: .mission/ Structure

**TODO:**
- [ ] Define JSON schemas for state files
- [ ] Template for CLAUDE.md (King prompt)
- [ ] Templates for worker prompts (11 personas)
- [ ] `mc init` creates full structure

**.mission/ layout:**
```
.mission/
├── CLAUDE.md              # King system prompt
├── config.json            # Project settings
├── state/
│   ├── phase.json
│   ├── tasks.json
│   ├── workers.json
│   └── gates.json
├── specs/
├── findings/
├── handoffs/
├── checkpoints/
└── prompts/
    ├── researcher.md
    ├── designer.md
    ├── developer.md
    └── ... (11 total)
```

### Phase 3: King CLAUDE.md

**TODO:**
- [ ] Write King system prompt (~100 lines)
- [ ] Document available mc commands
- [ ] Explain workflow (phases, gates, tasks)
- [ ] Define constraints (never code, always delegate)
- [ ] Test King spawning workers via mc

**Key sections:**
- Role and responsibilities
- Available commands (mc spawn, mc task, etc.)
- Workflow explanation
- Constraints (no coding, only coordinating)
- How to read findings and synthesize

### Phase 4: Worker Prompts

**TODO:**
- [ ] Researcher prompt (Idea phase)
- [ ] Designer prompt (Design phase)
- [ ] Architect prompt (Design phase)
- [ ] Developer prompt (Implement phase)
- [ ] Debugger prompt (Implement phase)
- [ ] Reviewer prompt (Verify phase)
- [ ] Security prompt (Verify phase)
- [ ] Tester prompt (Verify phase)
- [ ] QA prompt (Verify phase)
- [ ] Docs prompt (Document phase)
- [ ] DevOps prompt (Release phase)

**Each prompt includes:**
- Role and focus
- Zone constraint
- Handoff JSON format
- `mc handoff` instruction

### Phase 5: Go Bridge Updates

**TODO:**
- [ ] Spawn King as Claude Code process
- [ ] Route UI chat to King stdin
- [ ] Spawn workers as Claude Code processes
- [ ] Relay agent stdout to WebSocket
- [ ] File watcher on .mission/state/ → WebSocket events
- [ ] REST endpoint: POST /api/gates/:phase/approve

**WebSocket events:**
```
{ type: "phase_changed", phase: "design" }
{ type: "task_created", task: Task }
{ type: "task_updated", task_id: string, status: string }
{ type: "worker_spawned", worker_id: string, persona: string }
{ type: "worker_completed", worker_id: string }
{ type: "findings_ready", task_id: string }
{ type: "gate_ready", phase: string }
```

### Phase 6: Rust Integration

**TODO:**
- [ ] Create mc-core binary (or integrate into mc)
- [ ] `mc-core validate-handoff <file>` - Schema + semantic validation
- [ ] `mc-core check-gate <phase>` - Gate criteria evaluation
- [ ] `mc-core count-tokens <file>` - Token counting
- [ ] mc CLI calls mc-core for validation

### Phase 7: React UI Updates

**TODO:**
- [ ] Connect King chat to actual King process
- [ ] Display phase from WebSocket events
- [ ] Display tasks from WebSocket events
- [ ] Display active workers
- [ ] Gate approval dialog
- [ ] Findings viewer

### Phase 8: Integration Testing

**TODO:**
- [ ] Test: `mc init` creates valid .mission/
- [ ] Test: `mc spawn` creates Claude Code process
- [ ] Test: Worker outputs handoff, `mc handoff` validates
- [ ] Test: King reads findings from files
- [ ] Test: Gate check and approval flow
- [ ] Test: Full cycle Idea → Design → Implement

### Phase 9: Distribution

**TODO:**
- [ ] Homebrew formula for mission-control
- [ ] Bundles: mc CLI + mc-core (Rust)
- [ ] `brew install mission-control`
- [ ] README with install instructions

---

## Future: v6 — 3D Visualization

- [ ] React Three Fiber setup
- [ ] Agent avatars in 3D space
- [ ] Zone visualization
- [ ] Camera controls
- [ ] Animations (spawn, complete, handoff)

---

## Future: v7+ — Polish & Scale

- [ ] Persistence (PostgreSQL or SQLite)
- [ ] Multi-model routing (Haiku/Sonnet/Opus)
- [ ] Token budget enforcement
- [ ] Worker health monitoring in UI
- [ ] Dark/light themes
- [ ] Remote access (deploy bridge)
- [ ] Conductor Skill (MissionControl builds MissionControl)

---

## Quick Reference

| Version | Focus | Status |
|---------|-------|--------|
| v1 | Agent fundamentals | ✅ Done |
| v2 | Go orchestrator | ✅ Done |
| v3 | React UI | ✅ Done |
| v4 | Rust core | ✅ Done |
| v5 | King + mc CLI | 🔄 Current |
| v6 | 3D visualization | Future |
| v7+ | Polish & scale | Future |

---

## Files to Create (v5)

```
cmd/
└── mc/
    ├── main.go
    ├── init.go
    ├── spawn.go
    ├── kill.go
    ├── status.go
    ├── workers.go
    ├── handoff.go
    ├── gate.go
    ├── phase.go
    ├── task.go
    └── serve.go

templates/
├── CLAUDE.md              # King prompt template
├── config.json            # Default config
└── prompts/
    ├── researcher.md
    ├── designer.md
    ├── developer.md
    ├── debugger.md
    ├── reviewer.md
    ├── security.md
    ├── tester.md
    ├── qa.md
    ├── docs.md
    └── devops.md
```

---

## Notes

**Key insight from Gastown/Claude-Flow:** Don't reinvent Claude Code. The CLI manages state, Claude Code sessions do the orchestration.

**What mc CLI does:**
- State management (CRUD on .mission/ files)
- Process management (spawn/kill Claude Code)
- Validation (calls Rust core)

**What mc CLI does NOT do:**
- LLM API calls
- Orchestration decisions
- Context management