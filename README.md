# MissionControl

A visual multi-agent orchestration system for spawning, monitoring, and coordinating AI agents working on your codebase.

Inspired by [Vibecraft](https://vibecraft.dev) and [Ralv](https://ralv.dev).

## What It Does

Spawn multiple AI agents, watch them work in real-time, and coordinate their efforts through a visual interface.

```
┌─────────────────────────────────────────────────────────┐
│                     Web UI (React)                       │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐                │
│   │ Agent 1 │  │ Agent 2 │  │ Agent 3 │                │
│   │ working │  │  idle   │  │ working │                │
│   └─────────┘  └─────────┘  └─────────┘                │
└───────────────────────┬─────────────────────────────────┘
                        │ WebSocket
┌───────────────────────▼─────────────────────────────────┐
│              Go Orchestrator (localhost:8080)            │
│                        │                                 │
│         ┌──────────────┼──────────────┐                 │
│         ▼              ▼              ▼                 │
│   Claude Code    Python Agent   Claude Code             │
│   (stream-json)   (our format)  (stream-json)           │
└─────────────────────────────────────────────────────────┘
```

## Agent Types

| Type | Description | Use Case |
|------|-------------|----------|
| **Python** | Our custom agents (v0-v3) | Educational, lightweight, full control |
| **Claude Code** | Anthropic's CLI agent | Production power, MCP support |

Both output structured JSON, both appear in the same UI.

## Stack

| Component | Language | Purpose |
|-----------|----------|---------|
| Agents | Python | Custom agents we build from scratch |
| Orchestrator | Go | Process management, API, WebSocket |
| Stream Parser | Rust | Normalize different agent output formats |
| Web UI | React + Three.js | 2D dashboard → 3D visualization |

## Status

| Version | Status | Description |
|---------|--------|-------------|
| v1 | ✅ Done | Python agent fundamentals |
| v2 | ✅ Done | Go orchestrator + Rust parser |
| v3 | ✅ Done | Full 2D dashboard with zones, personas, King mode |
| v4 | 🔄 Current | 3D visualization |
| v5 | 📋 Planned | Persistence + Claude Code skill |
| v6+ | 📋 Future | Multi-model, wizard agent |

### v3 Features
- Zone-based agent organization
- Persona system (Code Reviewer, Full Developer, Test Writer, Documentation)
- King Mode - AI orchestrator to manage agent teams
- Real-time conversation view with tool call display
- Attention system for agent-user interaction
- Toast notifications, loading states, empty states
- Keyboard shortcuts (⌘N spawn, ⌘K kill, ↑/↓ navigate, etc.)
- 81 unit tests (29 Go + 52 React)

## Quick Start

### Python Agents (v1)

```bash
cd agents
pip install anthropic
export ANTHROPIC_API_KEY="your-key"

# Minimal agent (~50 lines, bash only)
python3 v0_minimal.py "list files in current directory"

# Full agent (~200 lines, read/write/edit)
python3 v1_basic.py "create a hello world script"

# With task planning (~300 lines)
python3 v2_todo.py "build a calculator"

# With subagent delegation (~450 lines)
python3 v3_subagent.py "build a todo app with tests"
```

### Orchestrator (v2+)

```bash
# Start orchestrator
cd orchestrator
go run .

# Spawn agents via API
curl -X POST localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{"type": "python", "agent": "v1_basic", "task": "create hello.py"}'

curl -X POST localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{"type": "claude", "task": "review hello.py", "workdir": "."}'

# List agents
curl localhost:8080/api/agents
```

### Web Dashboard (v3+)

```bash
cd web
npm install
npm run dev
# Open http://localhost:3000
```

### Running Tests

```bash
# Go backend tests (29 tests)
cd orchestrator
go test ./...

# React frontend tests (52 tests)
cd web
npm test
```

## Requirements

- Python 3.11+ with `anthropic` package
- Go 1.21+
- Rust (for stream parser)
- Node.js 18+ (for web UI)
- `ANTHROPIC_API_KEY` environment variable
- Claude Code CLI (for claude agent type)

## Docs

- [SPEC.md](SPEC.md) — Full specification
- [TODO.md](TODO.md) — Progress tracker

## Architecture Insights

**Why both Python agents and Claude Code?**
- Python agents: Learn how agents work, ~50-450 lines, we control everything
- Claude Code: Battle-tested, MCP support, `--output-format stream-json` for structured output

**Why Rust for the parser?**
- Existing `claude-codes` crate handles Claude Code protocol
- Real value when adding text-based agents (Aider, Gemini) in v6+
- Learning Rust with a practical, contained project

**How do agents share context?**
- v3-v4: Orchestrator injects context into system prompts
- v5+: Persistent storage (Beads/SQLite/Supabase)

## Future Ideas

- **Conductor Skill** — Claude Code skill that spawns agents via our CLI
- **Wizard Agent** — Meta-agent that orchestrates other agents
- **Multi-Model** — Codex CLI, Gemini, Grok alongside Claude
- **Remote Access** — Control from phone via cloudflared tunnel