# Changelog

All notable changes to MissionControl are documented in this file.

## v5.1 - Quality of Life

Improved developer experience, workflow management, and infrastructure.

### Documentation Cleanup
- Consolidated specs into 5 root files (README, ARCHITECTURE, CONTRIBUTING, CHANGELOG, TODO)
- Moved historical specs to `docs/archive/`
- Removed `web/README.md` Vite boilerplate

### Repository Cleanup
- Renamed `orchestrator/api/v5.go` → `king.go`
- Removed `orchestrator/api/v4.go`
- Updated `.gitignore` to cover dist/, target/, node_modules/, .mission/
- Removed accidentally committed build artifacts

### Testing Improvements
- 56 Rust tests (workflow state machine, token counting, handoff validation, gate criteria)
- Go integration tests (King lifecycle, WebSocket flow, API endpoints, Rust core subprocess)
- React tests: Project wizard (13), Multi-project switching, Matrix toggle (11)
- E2E Playwright tests (wizard flow, King Mode, agent spawning, zone CRUD, WebSocket reconnection)
- Test infrastructure: `make test`, `make test-rust`, `make test-go`, `make test-web`, `make test-integration`, `make test-e2e`
- GitHub Actions CI workflow for PRs

### Startup Simplification
- `make dev` starts vite + orchestrator with single command
- `make dev-ui` and `make dev-api` for individual services
- `make build` production build (Go + Rust + React)
- `make install` installs binaries to `/usr/local/bin`
- `make clean` removes build artifacts
- `mc serve` single binary with embedded React UI via Go `embed` package
- Homebrew formula created

### Project Wizard
- `ProjectWizard` component with step state machine
- `WorkflowMatrix` component with toggle logic
- Typing indicator component (300ms delay)
- API endpoints: `POST/GET/DELETE /api/projects`
- Sidebar project list with switch capability
- `mc init` accepts `--path`, `--git`, `--king`, `--config` flags
- Wizard passes matrix config as JSON file to `mc init`

### Configuration & Storage
- `~/.mission-control/` directory created on first run
- `mc` CLI and Orchestrator read/write config
- Project added to list when created via wizard
- `lastOpened` timestamp updated when project opened

### Bug Fixes
- **Rust Core Integration**: `mc-core` binary builds, CLI commands implemented, `orchestrator/core/client.go` wrapper created, inline Go parsing replaced with `core.CountTokens()`
- **Token Usage Display**: Piped through `mc-core tokens`, `token_usage` WebSocket event, UI header/status bar display
- **Agent Count**: `agent_spawned`/`agent_stopped` events emit correctly, UI listens and updates `agents` array, Playwright test verifies count increments

### Developer Experience
- `make lint` runs Go (golangci-lint) + Rust (clippy) + TypeScript (eslint)
- `make fmt` formats all code (go fmt, cargo fmt, prettier)

### Personas Management
- 11 workflow personas (researcher, designer, architect, developer, debugger, reviewer, security, tester, qa, docs, devops)
- Enable/disable personas per-project in Settings
- Persona configuration stored in `.mission/config.json`
- Prompt preview and edit capability for each persona
- WorkflowMatrix respects disabled personas
- Tools and skills defined for each persona

### Testing Summary
- 130 web tests (React + types)
- 56 Rust core tests
- 12 persona-related tests

---

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
