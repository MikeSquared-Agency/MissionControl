# MissionControl — Development Guide

## Overview
Multi-agent orchestration system. King agent (Kai/OpenClaw) coordinates ephemeral workers through a 10-stage workflow. State lives in `.mission/` files.

## Architecture
- **Go**: `cmd/mc/` (CLI), `orchestrator/` (bridge, API, WebSocket)
- **Rust**: `core/` (workflow engine, validation, token counting, checkpoint compilation)
- **React**: `web/` (legacy UI, now on darlington.dev)
- **State**: `.mission/` directory per project (stage, tasks, gates, findings, checkpoints)

## Key Commands
```bash
make build          # Build all (Go + Rust)
make test           # Run all tests
make lint           # golangci-lint + clippy + eslint
make fmt            # Format all code

# Go tests
cd cmd/mc && go test -v
cd orchestrator && go test ./...

# Rust tests
cd core && cargo test

# React tests
cd web && npm test
```

## Conventions
- Go module: `github.com/DarlingtonDeveloper/MissionControl`
- `cmd/mc` uses `replace` directive for cross-module hashid import
- Tasks stored in JSONL format (one JSON object per line)
- Task IDs are hash-based: `mc-xxxxx` (SHA256 truncated)
- All state mutations auto-commit to git with `[mc:{category}]` prefix
- Tests must use `saveTasks(missionDir, tasks)` / `loadTasks(missionDir)` — not direct file I/O
- Pre-commit hooks run `gofmt` + `go vet`
- CI requires `build-and-test` check to pass before merge

## 10 Stages
discovery → goal → requirements → planning → design → implement → verify → validate → document → release

## Process Discipline

- **Hard stop at every gate** — one approval per stage, no batching transitions
- **Planning produces a plan, not code** — planning output is task breakdown + approach
- **Role removal is a design decision** — removing personas must be flagged at Design stage
- **Small changes still need Verify** — don't skip because "it's small"
- **Retro after every mission** — mandatory, score honestly, never revise scores upward
- **Update docs on every PR** — ARCHITECTURE.md + CHANGELOG.md must reflect changes
- **Scope paths need type awareness** — include type-definition files alongside consumer files
- **Findings must have Summary header** — `## Summary` required for briefing chain

## Canonical Status Values

- **Task status:** `pending` → `active` → `done` (not "complete", not "finished")
- **Gate status:** `pending` → `approved`
- **Stage exempt from task requirements:** goal, requirements, planning, design
- **Velocity check:** blocks if stage lasted <10s with no completed tasks (non-exempt only)

## Context Management
When running multi-agent tasks:
- Compact at ~60% context usage — don't wait for the limit
- Instruct sub-agents to self-compact at 60%
- Break large tasks into smaller sequential steps
- Max 4 concurrent agents
- Summarize sub-agent results concisely before passing to next task
