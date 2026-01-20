# Changelog

All notable changes to MissionControl are documented in this file.

## v5 - King + mc CLI

The brain of MissionControl. King orchestration with CLI tooling.

### mc CLI
- `mc init` - Create .mission/ scaffold with King + worker prompts
- `mc status` - JSON dump of phase, tasks, workers, gates
- `mc phase` - Get/set current workflow phase
- `mc task` - Create, list, update tasks
- `mc spawn` - Spawn Claude Code worker process
- `mc kill` - Kill worker process
- `mc handoff` - Validate and store handoff (supports `--rust` flag)
- `mc gate` - Check/approve phase gates
- `mc workers` - List active workers with health check

### mc-core (Rust)
- `mc-core validate-handoff <file>` - Schema + semantic validation
- `mc-core check-gate <phase>` - Gate criteria evaluation
- `mc-core count-tokens <file>` - Fast token counting with tiktoken

### .mission/ Structure
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
└── prompts/               # 11 persona prompts
```

### Go Bridge
- Spawn King as Claude Code process
- Route UI chat to King stdin
- Spawn workers as Claude Code processes
- File watcher on .mission/state/ with WebSocket events
- REST endpoint: POST /api/mission/gates/:phase/approve

### React UI Updates
- King chat connected to actual King process
- Phase/tasks/workers display from WebSocket events
- Gate approval dialog
- Findings viewer with type filtering

### Testing
- 64 tests total (8 Go CLI + 56 Rust core)

---

## v4 - Rust Core

Deterministic workflow engine in Rust.

- Workflow engine (phases, gates, tasks)
- Knowledge manager (tokens, checkpoints, validation)
- Health monitor (stuck detection)
- Struct definitions and business logic

---

## v3 - 2D Dashboard

Full React dashboard with 81 unit tests.

- Zustand state management with persistence
- Header, Sidebar, AgentCard, AgentPanel components
- Zone System (CRUD, split/merge)
- Persona System (4 defaults + custom creation)
- King Mode UI (KingPanel, KingHeader)
- Attention System (notifications with quick response)
- Settings Panel with keyboard shortcuts
- 81 unit tests (29 Go + 52 React)

---

## v2 - Orchestrator

Go process manager with REST API.

- Go process manager (spawn/kill agents)
- REST API endpoints
- WebSocket event hub
- Rust stream parser

---

## v1 - Agent Fundamentals

Educational Python agents demonstrating core patterns.

- `v0_minimal.py` (~50 lines, bash only)
- `v1_basic.py` (~200 lines, full tools)
- `v2_todo.py` (~300 lines, explicit planning)
- `v3_subagent.py` (~450 lines, child agents)
