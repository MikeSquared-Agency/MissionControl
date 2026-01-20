# Architecture

MissionControl is a visual multi-agent orchestration system where a **King** agent coordinates **worker** agents through a **6-phase workflow**.

## System Overview

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
│                   Go Bridge                                  │
│   King Process │ Watcher │ mc CLI │ WebSocket Hub           │
└───────────┬───────────────────────────────┬─────────────────┘
            │ spawns Claude Code            │ subprocess
            │                               ▼
            │                    ┌─────────────────────┐
            │                    │   Rust Core         │
            │                    │   (mc-core)         │
            │                    ├─────────────────────┤
            │                    │ • Token counting    │
            │                    │ • Handoff validation│
            │                    │ • Gate checking     │
            │                    └─────────────────────┘
            │
  ┌─────────┼──────────────┐
  ▼         ▼              ▼
King      Worker         Worker
(Claude)  (Claude)       (Claude)
```

**Key insight:** King IS a Claude Code session with a good system prompt. Go bridge spawns processes and relays events — no custom LLM API calls. Rust core handles deterministic operations (validation, token counting) that shouldn't consume LLM tokens.

### What We're NOT Building
- Custom orchestration loop — Claude Code IS the orchestration
- LLM API integration in Go — King handles all LLM interaction
- Agent-to-agent message queue — workers write findings to files
- Context compilation service — .mission/ files ARE the context

### How King Orchestrates
King is a Claude Code session with `.mission/CLAUDE.md` as its system prompt. It orchestrates by:
- Using bash to call `mc spawn`, `mc status`, `mc handoff`, `mc gate`
- Reading/writing `.mission/` files directly (specs, findings, state)
- Leveraging normal Claude Code capabilities (file editing, bash, etc.)

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

## Directory Structure

```
/
├── cmd/mc/                  # mc CLI
├── orchestrator/            # Go bridge
│   ├── api/
│   │   ├── routes.go        # Route registration
│   │   ├── king.go          # King endpoints
│   │   ├── agents.go        # Agent endpoints
│   │   ├── zones.go         # Zone endpoints
│   │   └── projects.go      # Project/wizard endpoints
│   ├── bridge/
│   │   └── king.go          # King tmux manager
│   ├── core/
│   │   └── client.go        # Rust subprocess wrapper
│   ├── manager/
│   └── ws/
├── core/                    # Rust core
│   ├── workflow/
│   ├── knowledge/
│   ├── ffi/
│   └── README.md
├── web/                     # React UI
├── agents/                  # Python agents (educational)
├── docs/
│   └── archive/             # Historical specs
├── scripts/                 # Dev scripts
├── README.md
├── ARCHITECTURE.md
├── CONTRIBUTING.md
├── CHANGELOG.md
├── TODO.md
└── Makefile
```

## Stack

| Component | Language | Purpose |
|-----------|----------|---------|
| **Agents** | Python | Custom agents, educational |
| **mc CLI** | Go | MissionControl CLI commands |
| **Orchestrator** | Go | Process management, REST, WebSocket |
| **mc-core** | Rust | Validation, token counting, gate checking |
| **Core** | Rust | Workflow engine, knowledge manager |
| **Strategy** | Claude Opus | King agent |
| **Workers** | Claude Sonnet/Haiku | Task execution |
| **UI** | React | Dashboard with Zustand state |

## mc CLI

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

## mc-core (Rust)

```bash
mc-core validate-handoff <file>   # Schema + semantic validation
mc-core check-gate <phase>        # Gate criteria evaluation
mc-core count-tokens <file>       # Fast token counting with tiktoken
```

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
POST   /api/king/start          # Start King process
POST   /api/king/stop           # Stop King process
GET    /api/king/status         # Check if King is running
POST   /api/king/message        # Send message to King
```

### Mission Gates
```
GET    /api/mission/gates/:phase          # Check gate status
POST   /api/mission/gates/:phase/approve  # Approve gate
```

### WebSocket Events
```
mission_state      # Initial state sync
king_status        # King running status
phase_changed      # Phase transitioned
task_created       # New task created
task_updated       # Task status changed
worker_spawned     # Worker started
worker_completed   # Worker finished
gate_ready         # Gate criteria met
gate_approved      # Gate approved
findings_ready     # New findings available
king_output        # King process output
king_error         # King process error
```

## Worker Communication (Handoffs)

Workers don't communicate directly. They output structured JSON handoffs:

```
Worker completes task
        │
        ▼
Worker outputs JSON to stdout
        │
        ▼
Worker runs: mc handoff findings.json
        │
        ▼
mc-core validates (schema + semantics)
        │
        ├── Invalid → Error, worker retries
        │
        ▼ Valid
Stored in .mission/handoffs/
Compressed to .mission/findings/
Task status updated in .mission/state/tasks.json
        │
        ▼
Go file watcher sees change
        │
        ▼
WebSocket emits findings_ready event
        │
        ▼
King reads findings, decides next steps
```

This keeps workers isolated and context lean. No message passing, no shared memory — just files.

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

## .mission/ Directory

Each project has a `.mission/` directory containing all state:

```
.mission/
├── CLAUDE.md              # King system prompt
├── config.json            # Project settings (zones, personas)
├── state/
│   ├── phase.json         # Current workflow phase
│   ├── tasks.json         # Task list with status
│   ├── workers.json       # Active worker processes
│   └── gates.json         # Gate approval status
├── specs/                 # Design documents, requirements
├── findings/              # Worker output (research, reviews, etc.)
├── handoffs/              # Validated worker handoff JSONs
├── checkpoints/           # State snapshots for recovery
└── prompts/
    ├── researcher.md
    ├── designer.md
    ├── architect.md
    ├── developer.md
    ├── debugger.md
    ├── reviewer.md
    ├── security.md
    ├── tester.md
    ├── qa.md
    ├── docs.md
    └── devops.md
```

## Configuration

Global configuration is stored at `~/.mission-control/config.json`:

```json
{
  "projects": [
    {
      "path": "/Users/mike/projects/myapp",
      "name": "myapp",
      "lastOpened": "2026-01-19T10:00:00Z"
    }
  ],
  "lastProject": "/Users/mike/projects/myapp",
  "preferences": {
    "theme": "dark"
  }
}
```

Project-specific configuration lives in `.mission/config.json`.

## Design Rationale

**Why King + Workers?**
- King maintains continuity with user
- Workers are disposable, context stays lean
- Handoffs are cheap: spawn fresh vs accumulate

**Why Rust Core?**
- Deterministic logic shouldn't use LLM tokens
- Token counting needs to be fast and accurate
- Validation should be strict (JSON schemas)

**Why 6 Phases?**
- Prevents rushing to implementation
- Gates force quality checks
- Each phase has clear entry/exit criteria
