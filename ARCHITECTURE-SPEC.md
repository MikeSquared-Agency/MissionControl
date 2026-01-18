# MissionControl — Architecture Spec

## Overview

MissionControl is a visual multi-agent orchestration system. The key insight: **don't reinvent Claude Code**. King IS a Claude Code session with a good system prompt. Go is just a bridge to the UI.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  React UI                                                   │
│  - Visualize agents, phases, findings                       │
│  - Chat with King                                           │
│  - Approve gates                                            │
└───────────────────────┬─────────────────────────────────────┘
                        │ WebSocket
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  Go Bridge (NOT an orchestrator)                            │
│  - Spawns King (Claude Code + CLAUDE.md)                    │
│  - Spawns Workers (Claude Code + worker prompt)             │
│  - Relays stdout/events to UI                               │
│  - Serves REST API for UI actions                           │
└───────────────────────┬─────────────────────────────────────┘
                        │ spawns processes
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  mc CLI (State Management)                                  │
│  - spawn, kill, status, workers                             │
│  - handoff, gate, phase, task                               │
│  - Wraps Rust core for validation                           │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  King        │ │  Worker 1    │ │  Worker 2    │
│ (Claude Code)│ │ (Claude Code)│ │ (Claude Code)│
│              │ │              │ │              │
│ Orchestrates │ │ Executes     │ │ Executes     │
│ via mc CLI   │ │ outputs JSON │ │ outputs JSON │
└──────────────┘ └──────────────┘ └──────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  .mission/ (File-based State)                               │
│  - CLAUDE.md (King system prompt)                           │
│  - state/ (phase, tasks, workers, gates)                    │
│  - specs/, findings/, handoffs/, checkpoints/               │
│  - prompts/ (worker system prompts)                         │
└─────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  Rust Core (Deterministic Logic)                            │
│  - Workflow engine (phases, gates, tasks)                   │
│  - Knowledge manager (tokens, validation)                   │
│  - Health monitor (stuck detection)                         │
│  - Called by mc CLI, not FFI                                │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Insight

Inspired by [Gastown](https://github.com/steveyegge/gastown) and [Claude-Flow](https://github.com/ruvnet/claude-flow):

| What We Thought | What We Learned |
|-----------------|-----------------|
| Build custom orchestration loop in Go | Claude Code IS the orchestration loop |
| Go calls Anthropic API for King | King IS a Claude Code session |
| Message queues between agents | File-based state (.mission/) |
| Complex context compilation | Claude Code manages its own context |
| ~2000 lines of orchestration code | ~500 lines of bridge + CLI |

**Gastown's pattern:** `gt` CLI manages state. Claude Code sessions orchestrate themselves by reading state and calling CLI commands.

**Our pattern:** `mc` CLI manages state. King (Claude Code) orchestrates by reading .mission/ files and calling `mc` commands.

---

## Components

### 1. React UI (web/)

**Tech:** React 18, TypeScript, Tailwind, Zustand, Three.js (future)

**Responsibilities:**
- Display agents, phases, tasks, findings
- King chat interface
- Gate approval dialogs
- Real-time updates via WebSocket

**Does NOT:**
- Make LLM API calls
- Manage agent processes
- Store persistent state

### 2. Go Bridge (orchestrator/)

**Tech:** Go, gorilla/mux, gorilla/websocket

**Responsibilities:**
- Spawn King process (Claude Code with .mission/CLAUDE.md)
- Spawn worker processes (Claude Code with persona prompts)
- Relay stdout from agents to WebSocket
- Watch .mission/state/ for changes → emit WebSocket events
- Serve REST API for UI actions (gate approval, etc.)

**Does NOT:**
- Call Anthropic API directly
- Make orchestration decisions
- Compile briefings or context

**Key change from v3:** The "orchestrator" is now just a process manager and event relay. All intelligence is in Claude Code sessions.

### 3. mc CLI (cmd/mc/)

**Tech:** Go, cobra, calls Rust core

**Commands:**
```
mc init                              # Create .mission/ scaffold
mc spawn <persona> <task> --zone <z> # Spawn worker
mc kill <worker-id>                  # Kill worker
mc status                            # JSON dump of state
mc workers                           # List active workers
mc handoff <file>                    # Validate and store handoff
mc gate check <phase>                # Check gate criteria
mc gate approve <phase>              # Approve gate
mc phase                             # Get current phase
mc phase next                        # Transition to next phase
mc task create <n> --phase <p>       # Create task
mc task list                         # List tasks
mc task update <id> --status <s>     # Update task status
mc serve                             # Start Go bridge + UI
```

**Who uses it:**

| User | When | Commands |
|------|------|----------|
| You (human) | King mode OFF, manual control | All of them |
| King (Claude Code) | King mode ON | spawn, task, gate, status |
| Workers (Claude Code) | Always | handoff |
| Go Bridge | Always | spawn (to start King/workers) |

### 4. King (Claude Code)

**Tech:** Claude Code with .mission/CLAUDE.md

**Responsibilities:**
- Talk to user, understand intent
- Create tasks via `mc task create`
- Spawn workers via `mc spawn`
- Read findings from .mission/findings/
- Synthesize and decide next steps
- Recommend gate approvals

**Constraints:**
- Never writes code directly
- Never implements features
- Delegates all work to workers
- Uses `mc` CLI for state management

**System prompt:** Lives in `.mission/CLAUDE.md`, created by `mc init`.

### 5. Workers (Claude Code)

**Tech:** Claude Code with persona-specific prompts

**Responsibilities:**
- Execute assigned task
- Stay within assigned zone
- Output structured JSON handoff when done
- Run `mc handoff` to validate and store

**11 Personas:**

| Persona | Phase | Model | Focus |
|---------|-------|-------|-------|
| Researcher | Idea | Sonnet | Feasibility research |
| Designer | Design | Sonnet | UI mockups |
| Architect | Design | Sonnet | System design |
| Developer | Implement | Sonnet | Build features |
| Debugger | Implement | Sonnet | Fix issues |
| Reviewer | Verify | Haiku | Code review |
| Security | Verify | Sonnet | Security audit |
| Tester | Verify | Haiku | Write tests |
| QA | Verify | Haiku | E2E testing |
| Docs | Document | Haiku | Documentation |
| DevOps | Release | Haiku | Deployment |

### 6. Rust Core (core/)

**Tech:** Rust

**Responsibilities:**
- Workflow engine (phase state machine, gate logic)
- Knowledge manager (token counting, handoff validation)
- Health monitor (stuck detection thresholds)

**Exposed as:** CLI commands called by `mc`, not FFI.

```bash
# mc handoff internally calls:
mc-core validate-handoff findings.json

# mc gate check internally calls:
mc-core check-gate design
```

### 7. .mission/ Directory

**Created by:** `mc init`

**Structure:**
```
.mission/
├── CLAUDE.md                 # King system prompt
├── config.json               # Project settings
│
├── state/
│   ├── phase.json           # Current phase
│   ├── tasks.json           # All tasks
│   ├── workers.json         # Active workers
│   └── gates.json           # Gate status
│
├── specs/
│   └── SPEC-{name}.md       # Feature specs
│
├── findings/
│   └── {task-id}.json       # Compressed findings per task
│
├── handoffs/
│   └── {worker-id}-{ts}.json # Raw handoff records
│
├── checkpoints/
│   └── checkpoint-{ts}.json  # Periodic full state
│
└── prompts/
    ├── researcher.md         # Persona prompts
    ├── designer.md
    ├── developer.md
    └── ...
```

---

## Domains

The four-domain model still applies, but implementation is simpler:

| Domain | What | Who Handles It |
|--------|------|----------------|
| **Strategy** | What to build, phase decisions | King (Claude Code) |
| **Workflow** | Phase state, gates, tasks | Rust core + mc CLI |
| **Knowledge** | Specs, findings, token budgets | Files + Rust validation |
| **Runtime** | Process management, health | Go bridge + mc CLI |

---

## Data Flow

### User requests a feature (King mode ON)

```
User: "Build a login page"
        │
        ▼
React UI → WebSocket → Go Bridge → King stdin
        │
        ▼
King (Claude Code):
  1. Asks clarifying questions
  2. Writes spec to .mission/specs/
  3. Runs: mc task create "Design login UI" --phase design
  4. Runs: mc spawn designer "Design login UI" --zone frontend
        │
        ▼
Go Bridge spawns Designer (Claude Code with designer prompt)
        │
        ▼
Designer:
  1. Creates mockups
  2. Outputs JSON handoff
  3. Runs: mc handoff findings.json
        │
        ▼
mc validates (via Rust), stores in .mission/findings/
        │
        ▼
Go file watcher sees change → WebSocket event
        │
        ▼
King reads findings, synthesizes, continues...
```

### Manual control (King mode OFF)

```
You: mc status
You: mc task create "Fix bug" --phase implement
You: mc spawn developer "Fix bug" --zone backend
You: # ... worker completes ...
You: mc gate check implement
You: mc gate approve implement
```

---

## Distribution

**Install via Brew:**
```bash
brew install mission-control
```

**This installs:**
- `mc` CLI (Go binary)
- `mc-core` (Rust binary, called by mc)
- `mission-control` (alias for `mc serve`)

**Setup flow:**
```bash
# In your project
mc init                    # Creates .mission/

# Start everything
mc serve                   # Starts Go bridge + opens UI

# Or manual
mc serve --no-ui           # Just the bridge
cd web && npm run dev      # UI separately
```

---

## Tech Stack Summary

| Component | Tech | Lines (est) |
|-----------|------|-------------|
| React UI | React, TypeScript, Tailwind, Zustand | ~3000 (done in v3) |
| Go Bridge | Go, gorilla/websocket | ~500 |
| mc CLI | Go, cobra | ~400 |
| Rust Core | Rust | ~800 (done in v4) |
| King CLAUDE.md | Markdown | ~100 |
| Worker prompts | Markdown | ~50 each × 11 |

**Total new code for v5:** ~900 lines of Go + markdown prompts.

---

## What We're NOT Building

| Thing | Why Not |
|-------|---------|
| Custom LLM API calls in Go | Claude Code does this |
| Message queues between agents | File-based state works |
| Context compilation service | Claude Code manages context |
| Steward/Quartermaster agent | Just CLI tools |
| FFI bindings | CLI is simpler |

---

## Success Criteria

MissionControl works when:

1. `mc init` creates .mission/ with King prompt and worker prompts
2. `mc serve` starts Go bridge and connects UI
3. King (Claude Code) can spawn workers via `mc spawn`
4. Workers output structured handoffs, validated by `mc handoff`
5. King reads findings and continues orchestrating
6. Gates can be checked and approved via `mc gate`
7. Full workflow: Idea → Design → Implement → Verify → Document → Release
8. UI shows real-time updates via WebSocket

---

## Version History

| Version | Focus | Status |
|---------|-------|--------|
| v1 | Agent fundamentals (Python) | ✅ Done |
| v2 | Go orchestrator + Rust parser | ✅ Done |
| v3 | React UI (2D dashboard) | ✅ Done |
| v4 | Rust core (workflow, knowledge, health) | ✅ Done |
| v5 | King + mc CLI + integration | 🔄 Current |
| v6 | 3D visualization | Future |
| v7 | Persistence, multi-model | Future |