# MissionControl

A visual multi-agent orchestration system where a **King** agent coordinates **worker** agents through a **6-phase workflow**. Workers spawn, complete tasks, and die. Context lives in files, not conversation memory.

Inspired by [Vibecraft](https://vibecraft.dev), [Ralv](https://ralv.dev), and [Gastown](https://gastown.dev).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Web UI (React)                          │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐       │
│   │  King   │  │  Zones  │  │ Agents  │  │Settings │       │
│   │  Panel  │  │  List   │  │  Panel  │  │  Panel  │       │
│   └─────────┘  └─────────┘  └─────────┘  └─────────┘       │
└───────────────────────┬─────────────────────────────────────┘
                        │ WebSocket
┌───────────────────────▼─────────────────────────────────────┐
│                   Go API Layer                               │
│   Agents │ Zones │ King │ (v4: Workflow │ Knowledge)        │
└───────────────────────┬─────────────────────────────────────┘
                        │ (v4: FFI)
┌───────────────────────▼─────────────────────────────────────┐
│               Rust Core (v4 - planned)                       │
│   Workflow Engine │ Knowledge Manager │ Health Monitor       │
└───────────────────────┬─────────────────────────────────────┘
                        │
         ┌──────────────┼──────────────┐
         ▼              ▼              ▼
      King           Worker         Worker
     (Opus)         (Sonnet)        (Haiku)
```

## Key Concepts

### King Agent
The King is the only persistent agent. It talks to you, decides what to build, spawns workers, and approves phase gates. It never implements directly.

### Workers
Workers are ephemeral. They receive a **briefing** (~300 tokens), do their task, output **findings**, and die. This keeps context lean and costs low.

### 6-Phase Workflow
```
IDEA → DESIGN → IMPLEMENT → VERIFY → DOCUMENT → RELEASE
```
Each phase has a **gate** requiring approval before proceeding.

### Zones
Zones organize the codebase (Frontend, Backend, Database, Infra, Shared). Workers are assigned to zones and stay in their lane.

## Status

| Version | Status | Description |
|---------|--------|-------------|
| v1 | ✅ Done | Python agent fundamentals |
| v2 | ✅ Done | Go orchestrator + Rust parser |
| v3 | ✅ Done | Full 2D dashboard (81 tests) |
| v4 | 🔄 Current | Architecture foundation (Rust core) |
| v5 | 📋 Planned | King agent + workflow system |
| v6 | 📋 Planned | 3D visualization + polish |

## v3 Features

- **Zone System** — Create, edit, split, merge zones; move agents between zones
- **Persona System** — 4 default personas + custom creation
- **King Mode** — UI shell with KingPanel, TeamOverview (full logic in v5)
- **Attention System** — Notifications with quick response buttons
- **Settings** — General, Personas, Shortcuts tabs
- **Keyboard Shortcuts** — ⌘N spawn, ⌘K kill, ⌘⇧K king mode, etc.
- **81 Unit Tests** — 29 Go + 52 React

## Stack

| Component | Language | Purpose |
|-----------|----------|---------|
| **Agents** | Python | Custom agents, educational |
| **API** | Go | Process management, REST, WebSocket |
| **Core** | Rust | Workflow engine, token counting (v4) |
| **Strategy** | Claude Opus | King agent (v5) |
| **Workers** | Claude Sonnet/Haiku | Task execution |
| **UI** | React + Three.js | Dashboard + 3D visualization |

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
  -d '{"type": "claude-code", "task": "review hello.py", "workingDir": "."}'

# List agents
curl localhost:8080/api/agents

# Create zone
curl -X POST localhost:8080/api/zones \
  -H "Content-Type: application/json" \
  -d '{"name": "Frontend", "color": "#3b82f6"}'
```

### Web Dashboard (v3)

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
- Rust 1.75+ (for v4+ core)
- Node.js 18+ (for web UI)
- `ANTHROPIC_API_KEY` environment variable
- Claude Code CLI (for claude-code agent type)

## API Endpoints

### Agents
```
POST   /api/agents              # Spawn agent
GET    /api/agents              # List agents
DELETE /api/agents/:id          # Kill agent
POST   /api/agents/:id/message  # Send message
POST   /api/agents/:id/respond  # Respond to attention
```

### Zones
```
POST   /api/zones               # Create zone
GET    /api/zones               # List zones
PUT    /api/zones/:id           # Update zone
DELETE /api/zones/:id           # Delete zone
```

### King
```
POST   /api/king/message        # Send message to King
```

## Worker Personas

| Persona | Phase | Model | Purpose |
|---------|-------|-------|---------|
| Researcher | Idea | Sonnet | Feasibility research |
| Designer | Design | Sonnet | UI mockups |
| Architect | Design | Sonnet | System design |
| Developer | Implement | Sonnet | Build features |
| Debugger | Implement | Sonnet | Fix issues |
| Reviewer | Verify | Haiku | Code review |
| Security | Verify | Sonnet | Vulnerability check |
| Tester | Verify | Haiku | Write tests |
| QA | Verify | Haiku | E2E validation |
| Docs | Document | Haiku | Documentation |
| DevOps | Release | Haiku | Deployment |

## Docs

- [SPEC.md](SPEC.md) — Full specification
- [TODO.md](TODO.md) — Progress tracker
- [ARCHITECTURE.md](MISSIONCONTROL-ARCHITECTURE-SPEC.md) — Technical architecture

## Architecture Insights

**Why King + Workers?**
- King maintains continuity with user
- Workers are disposable, context stays lean
- Handoffs are cheap: spawn fresh vs accumulate

**Why Rust Core (v4)?**
- Deterministic logic shouldn't use LLM tokens
- Token counting needs to be fast and accurate
- Validation should be strict (JSON schemas)

**Why 6 Phases?**
- Prevents rushing to implementation
- Gates force quality checks
- Each phase has clear entry/exit criteria

## Future Ideas

- **Conductor Skill** — Claude Code skill that spawns agents via our CLI
- **Wizard Agent** — Meta-agent that orchestrates other agents
- **Multi-Model** — Codex CLI, Gemini, Grok alongside Claude
- **Remote Access** — Control from phone via cloudflared tunnel
- **Persistence** — Beads/SQLite for cross-session memory