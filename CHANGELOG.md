# Changelog

All notable changes to MissionControl are documented in this file.

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
