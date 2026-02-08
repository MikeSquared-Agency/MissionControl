# KAI_NOTES — MissionControl Summary

## What It Does

Multi-agent orchestration system. A **King** agent (Claude Opus) coordinates ephemeral **worker** agents (Sonnet/Haiku) through a 6-phase gated workflow (Idea → Design → Implement → Verify → Document → Release). Workers get short briefings, do their task, write findings to files, and die. Context lives in `.mission/` directory files, not conversation memory.

Key insight: King IS a Claude Code session with a system prompt. The Go bridge just spawns processes and relays events — no custom LLM API calls.

## Tech Stack

| Layer | Tech | Notes |
|-------|------|-------|
| **CLI** (`cmd/mc/`) | Go | Cobra commands: init, spawn, kill, status, gate, task, etc. |
| **Orchestrator** (`orchestrator/`) | Go | REST API, WebSocket hub, file watcher, King/worker process management |
| **Core** (`core/`) | Rust (5 crates) | Deterministic ops: handoff validation, gate checking, token counting, workflow engine |
| **UI** (`web/`) | React + Zustand + TypeScript | Dashboard with King panel, zones, agents, settings |
| **Agents** (`agents/`) | Python | Educational examples (v0-v3), not production |
| **Stream Parser** | Rust | Separate binary for parsing |

## Architecture

- `.mission/` directory = all project state (phase, tasks, workers, gates, findings, prompts)
- Go bridge watches `.mission/state/` for file changes → emits WebSocket events → React UI updates
- Workers communicate via file-based handoffs validated by Rust core (no IPC/message queues)
- 11 worker personas (Researcher, Designer, Architect, Developer, Debugger, Reviewer, Security, Tester, QA, Docs, DevOps) mapped to phases
- Zones (Frontend/Backend/Database/Infra/Shared) keep workers in their lane

## Current State

- **v1–v5: Done.** Python agents → Go orchestrator → React UI (81 tests) → Rust core → King + mc CLI (64 tests)
- **v5.1: In progress.** QoL — UI polish (loading/error states, WS indicator), Homebrew distribution, project switching
- **v6+: Planned.** JSONL migration, task dependencies, audit trail, 10-stage workflow, requirements traceability, multi-model routing, cost tracking, 3D viz

## Things to Know

- Tests: 56 Rust + 8 Go CLI + 81 React + 10 E2E (Playwright)
- Build: `make build` / `make test` / `make dev`
- Orchestrator has Ollama integration (`orchestrator/ollama/`) — local model support
- Bridge manages King via tmux (`orchestrator/bridge/king.go`)
- Homebrew formula exists but tap repo not yet created
- The `core/` Rust workspace has 5 crates: workflow, knowledge, ffi, mc-protocol, mc-core, runtime
- No auth layer yet — relevant for planned remote deployment (v8.5)
