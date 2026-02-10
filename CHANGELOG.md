# Changelog

All notable changes to MissionControl are documented in this file.

## v6.11 — Process Purity Phase 2 (2026-02-10)

### Trusted Validator (CI)
- Two-step CI build: `main` branch produces a trusted `mc` binary, PR validation uses that pre-built binary
- Prevents PRs from modifying the validator to bypass their own checks
- `mc-validate` workflow updated to fetch trusted artifact before running `--strict`

### Agent Tool Policies (OpenClaw)
- Per-persona tool restrictions in OpenClaw agent config
- Developers cannot approve gates; reviewers cannot write implementation files
- Enforces separation of duties at the agent level, not just workflow level

---

## v6.10 — Process Purity Phase 1 (2026-02-10)

### Strict Validation (`--strict` flag)
- `mc commit --validate-only --strict` — runs Phase 1 process checks on top of base validation
- **Verify persona coverage**: verify stage requires `done` tasks for `reviewer`, `security`, and `tester` personas
- **Integrator requirement**: multi-task implement stages require a `done` integrator task

### CI Pipeline
- `mc-validate` GitHub Actions workflow — runs `mc commit --validate-only --strict` on all PRs to `main`
- `make build-ci` — lightweight CI build target (mc + mc-core, no web assets)
- Pipeline: checkout → Go 1.22 + Rust toolchain → cargo cache → build-ci → validate

### Testing
- `strict_test.go` — unit tests for verify persona coverage and integrator requirement checks

---

## v6.9 — Gate UX (2026-02-10)

### Gate Satisfy & Status Commands
- `mc gate satisfy <substring>` — mark a single gate criterion as satisfied by substring match
- `mc gate satisfy --all` — satisfy all criteria for the current stage at once
- `mc gate status` — display current stage gate criteria with ✓/✗ status and progress count

### Gate-Aware Stage Advance
- `mc stage next` now checks `gates.json` before advancing — if all criteria are satisfied, the gate passes automatically
- Eliminates the need for `--force` in normal workflows; satisfy criteria, then advance
- Falls back to `mc-core check-gate` if `gates.json` has no entry for the stage

### Configuration
- `auto_mode` field in `.mission/config.json` — enables autonomous workflow mode

### Legacy Compatibility
- `loadGates()` auto-detects and converts legacy `gates.json` format (plain string criteria → structured `{description, satisfied}` objects)
- Malformed or null gates handled gracefully

### Testing
- 16 gate tests covering: satisfy by exact/substring match, ambiguous match rejection, missing stage/criterion errors, `--all` bulk satisfy, `allCriteriaMet` logic, legacy format loading, corrupt JSON handling, null gates, empty substring rejection, `gate status` output, and `initGateForStage`

---

## v6.8 — Briefing Generation, Scope Paths & Integrator Gate (2026-02-10)

### Briefing Generation
- `mc briefing generate <task-id>` — auto-compose worker briefings from task metadata + predecessor findings
- Validates all dependencies are complete before generating
- Extracts Summary headers from predecessor findings and includes them in the briefing
- Output: `.mission/handoffs/<task-id>-briefing.json`

### Task Scope Paths
- `scope_paths` field on tasks (`--scope-paths` flag on `mc task create`)
- Comma-separated list of files/directories a worker should touch
- Finer-grained than zones — file-level worker boundaries

### Integrator Gate Check (Implement Stage)
- `Gate::check_integrator_requirement()` in Rust core
- When multiple implement-stage tasks exist, requires at least one `integrator` persona task to be done
- Ensures integration verification always runs for parallelized work
- Single-task implement stages skip this requirement

### JSONL Compatibility Deserializer (mc-core)
- Lightweight `JsonlTask` struct for cross-language JSONL compatibility
- Handles Go-written string dates/status values in Rust gate checker
- Silently skips malformed lines for forward compatibility

---

## v6 — State Management & OpenClaw Integration (2026-02-08)

Major release: 10-stage workflow, JSONL storage, task dependencies, OpenClaw bridge, auto-checkpoint, CI pipeline. Merged to main 2026-02-08.

### 10-Stage Workflow
- **Phase → Stage rename** across entire stack (Rust, Go, React)
- **10 stages**: Discovery → Goal → Requirements → Planning → Design → Implement → Verify → Validate → Document → Release
- New personas: Analyst (Goal), Requirements Engineer (Requirements)
- `mc stage` replaces `mc phase` (deprecated alias kept)
- `mc migrate` converts v5 projects (phase.json → stage.json, `idea` → `discovery`)
- Gate auto-advance fix: prevents skipping stages

### JSONL Storage (6.1)
- `tasks.json` → `tasks.jsonl` (one JSON object per line)
- Line-by-line git diffs, enables concurrent writes

### Hash-Based Task IDs (6.2)
- SHA256(title + timestamp) → deterministic `mc-xxxxx` IDs
- Prevents collisions, enables idempotent retries

### Audit Trail (6.3)
- Append-only `audit/interactions.jsonl` logging all mutations
- `mc audit` command to query history

### Task Dependencies (6.4–6.6)
- `blocks`/`blockedBy` fields with cycle detection
- `mc ready` — tasks with no open blockers
- `mc dep tree <id>` — dependency graph
- `mc blocked` — all blocked tasks and why

### Git Auto-Commit (6.7)
- All mutations auto-commit with `[mc:{category}]` prefix
- Categories: checkpoint, task, gate, stage, worker, handoff
- Per-category config in `.mission/config.json`

### Session Continuity & Checkpoints
- `mc checkpoint` — snapshot state to `.mission/orchestrator/checkpoints/`
- `mc checkpoint restart` — restart with compiled ~500 token briefing
- `mc checkpoint status` — session health (green/yellow/red)
- `mc checkpoint auto --tokens <n>` — auto-checkpoint at token threshold
- Auto-checkpoint on gate approval and graceful shutdown
- Rust: `CheckpointCompiler`, `mc-core checkpoint-compile/validate`
- React UI: health indicator, restart button, history viewer, toast notifications

### OpenClaw Integration
- WebSocket bridge connecting to OpenClaw gateway
- REST endpoints: `/api/openclaw/{event,status,send}`
- Message relay: MC UI chat ↔ OpenClaw agent session
- Kai (OpenClaw) replaces tmux-based King process

### Agent Teams
- Named worker groups with `mc team` commands
- Coordinated task assignment

### Project Symlinks
- `~/.mission-control/projects/` symlinks
- `mc project link/list` for quick switching

### CI Pipeline
- GitHub Actions: build, test, vet, golangci-lint
- Pre-commit hooks: gofmt + go vet
- Branch protection requiring CI pass

### Testing
- 79 Rust tests, Go CLI + integration tests, 136 React tests
- Integration: multi-channel gates, state persistence, full 10-stage walkthrough, v5→v6 migration

---

## v5.1 — Quality of Life

### Developer Experience
- `make dev/build/test/clean/install/lint/fmt` commands
- `.mission-control/` global config directory
- Project wizard component with step state machine
- mc-core Rust subprocess integration (validation, token counting)
- 11 workflow personas with configurable prompts

### Testing
- 130 React tests, 56 Rust tests, Go integration tests
- Playwright E2E tests

---

## v5 — King + mc CLI

### mc CLI
- `mc init/status/stage/task/spawn/kill/handoff/gate/workers/serve`

### mc-core (Rust)
- `validate-handoff`, `check-gate`, `count-tokens`

### Go Bridge
- King process management, file watcher, WebSocket hub, REST API

---

## v4 — Rust Core
- Workflow engine (stages, gates, tasks)
- Knowledge manager (tokens, checkpoints, validation)
- Health monitor (stuck detection)

## v3 — 2D Dashboard
- Full React dashboard, Zustand state, zone/persona systems, King Mode UI
- 81 unit tests (29 Go + 52 React)

## v2 — Orchestrator
- Go process manager, REST API, WebSocket hub, Rust stream parser

## v1 — Agent Fundamentals
- Educational Python agents (v0–v3: minimal → subagent delegation)
