# V5 Implementation Guide

The brain of MissionControl. V4 gave us the Rust core. V5 wires it up with Claude Code as the actual orchestrator.

**Key insight:** Don't reinvent Claude Code. King IS a Claude Code session with a good system prompt. Go is just a bridge to the UI.

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
│  - Provides CLI tools King can call                         │
└───────────────────────┬─────────────────────────────────────┘
                        │ spawns processes
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  King (Claude Code)                                         │
│  - .mission/CLAUDE.md with King persona                     │
│  - Uses bash to call: mc spawn, mc status, mc handoff       │
│  - Reads/writes .mission/ files directly                    │
│  - Orchestrates via normal Claude Code capabilities         │
└───────────────────────┬─────────────────────────────────────┘
                        │ mc spawn worker
                        ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Worker 1   │  │   Worker 2   │  │   Worker 3   │
│ (Claude Code)│  │ (Claude Code)│  │ (Claude Code)│
│              │  │              │  │              │
│ Outputs JSON │  │ Outputs JSON │  │ Outputs JSON │
│   handoff    │  │   handoff    │  │   handoff    │
└──────────────┘  └──────────────┘  └──────────────┘
```

**What this means:**
- No custom LLM API calls in Go
- No message queues between King and workers
- No "Steward" or "Quartermaster" agent
- Claude Code's existing capabilities ARE the orchestration

---

## Quick Reference

| Topic | Source File |
|-------|-------------|
| Worker personas | `PERSONAS-SPEC.md` |
| 6-phase workflow | `PERSONAS-SPEC.md` → Phases & Gates |
| Handoff schema | `TOKEN-RESEARCH.md` → Layer 3 |
| Rust CLI tools | `V4-RUST-CONTRACTS.md` (now exposed as CLI) |

---

## What We're Building

| Component | What It Is |
|-----------|------------|
| `mc` CLI | Go binary with subcommands (spawn, status, handoff, gate) |
| King CLAUDE.md | System prompt that makes Claude Code act as King |
| Worker prompts | Per-persona system prompts |
| .mission/ | File-based state (specs, findings, checkpoints) |
| Go Bridge | Spawns processes, relays to WebSocket |

**What we're NOT building:**
- Custom orchestration loop
- LLM API integration in Go
- Agent-to-agent message queue
- Context compilation service

---

## Implementation Sequence

### Phase 1: `mc` CLI Tool

A single Go binary that King and the bridge call.

```
mc
├── spawn      # Spawn a worker
├── kill       # Kill a worker
├── status     # Get system status
├── workers    # List active workers
├── handoff    # Validate and store handoff
├── gate       # Check or approve gate
├── phase      # Get/set current phase
├── task       # CRUD for tasks
└── init       # Initialize .mission/ directory
```

**TODO:**
- [ ] `mc init` - Create .mission/ scaffold
- [ ] `mc spawn <persona> <task> --zone <zone>` - Spawn worker process
- [ ] `mc kill <worker-id>` - Kill worker process
- [ ] `mc status` - JSON dump of current state
- [ ] `mc workers` - List active workers with health
- [ ] `mc handoff <file>` - Validate handoff JSON, store delta
- [ ] `mc gate check <phase>` - Check if gate criteria met
- [ ] `mc gate approve <phase>` - Approve gate
- [ ] `mc phase` - Get current phase
- [ ] `mc phase next` - Transition to next phase
- [ ] `mc task create <name> --phase <phase> --zone <zone>`
- [ ] `mc task list --phase <phase>`
- [ ] `mc task update <id> --status <status>`

**Implementation:**
```go
// cmd/mc/main.go
package main

import (
    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{Use: "mc"}
    
    rootCmd.AddCommand(
        spawnCmd(),
        killCmd(),
        statusCmd(),
        workersCmd(),
        handoffCmd(),
        gateCmd(),
        phaseCmd(),
        taskCmd(),
        initCmd(),
    )
    
    rootCmd.Execute()
}
```

**Rust integration:**
```go
// mc handoff calls Rust for validation
func handoffCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "handoff <file>",
        Short: "Validate and store a worker handoff",
        Run: func(cmd *cobra.Command, args []string) {
            // Call Rust CLI or FFI
            result := validateHandoff(args[0])
            if !result.Valid {
                fmt.Println(result.Errors)
                os.Exit(1)
            }
            // Store in .mission/handoffs/
            storeHandoff(result)
        },
    }
}
```

---

### Phase 2: .mission/ Directory

File-based state that King reads/writes directly.

```
.mission/
├── CLAUDE.md                 # King's system prompt (Claude Code reads this)
├── config.json               # Project settings
│
├── state/
│   ├── phase.json           # Current phase
│   ├── tasks.json           # All tasks
│   ├── workers.json         # Active workers
│   └── gates.json           # Gate status
│
├── specs/
│   ├── SPEC-{name}.md       # Feature specs
│   └── api.md               # API design
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
    ├── researcher.md         # Researcher system prompt
    ├── designer.md           # Designer system prompt
    ├── developer.md          # Developer system prompt
    └── ...                   # Other personas
```

**TODO:**
- [ ] `mc init` creates this structure
- [ ] King CLAUDE.md template
- [ ] Worker prompt templates (11 personas)
- [ ] State JSON schemas

---

### Phase 3: King CLAUDE.md

This is the system prompt that makes Claude Code act as King.

```markdown
# King — MissionControl Coordinator

You are King, the strategic coordinator of MissionControl. You talk to the user,
decide what to build, and coordinate workers to execute.

## Your Role
- Understand what the user wants to build
- Break work into phases: Idea → Design → Implement → Verify → Document → Release
- Create tasks and spawn workers to execute them
- Synthesize findings when workers complete
- Recommend gate approvals to proceed to next phase

## Your Constraints
- You NEVER write code or implement features directly
- You coordinate and delegate - workers do the actual work
- You read/write files in .mission/ to track state
- You spawn workers using the `mc` CLI

## Commands Available

### Spawn a worker
```bash
mc spawn developer "Implement login form" --zone frontend
```

### Check status
```bash
mc status
```

### List workers
```bash
mc workers
```

### Check gate
```bash
mc gate check design
```

### Approve gate (after user confirms)
```bash
mc gate approve design
```

### Create task
```bash
mc task create "Build login API" --phase implement --zone backend --persona developer
```

## Workflow

1. User describes what they want
2. You clarify requirements, draft spec in .mission/specs/
3. You create tasks: `mc task create ...`
4. You spawn workers: `mc spawn <persona> <task> --zone <zone>`
5. Workers complete and output handoff JSON
6. You read findings from .mission/findings/
7. You synthesize and decide next steps
8. When phase complete, you ask user to approve gate
9. User approves, you run `mc gate approve <phase>`
10. Repeat for next phase

## Current State

Read current state with `mc status` or check files:
- Phase: `cat .mission/state/phase.json`
- Tasks: `cat .mission/state/tasks.json`
- Workers: `mc workers`

## Finding Synthesis

When workers complete, read their findings:
```bash
cat .mission/findings/<task-id>.json
```

Synthesize findings and update specs or create new tasks as needed.

## Important

- Always check `mc status` before making decisions
- Always read worker findings before proceeding
- Ask user for gate approval, don't auto-approve
- Keep the user informed of progress
```

**TODO:**
- [ ] Write full King CLAUDE.md
- [ ] Test King spawning workers via mc CLI
- [ ] Test King reading/writing .mission/ files

---

### Phase 4: Worker Prompts

Each persona gets a system prompt in `.mission/prompts/`.

**Developer prompt example:**
```markdown
# Developer — {zone} Zone

You are a Developer working in the {zone} zone.

## Your Task
{task_description}

## Constraints
- Stay within the {zone} directory
- Do not modify files outside your zone
- Focus only on your assigned task

## When Complete

Output your findings as JSON:

```json
{
  "task_id": "{task_id}",
  "worker_id": "{worker_id}",
  "status": "complete",
  "findings": [
    {
      "type": "decision",
      "summary": "What you decided and why"
    },
    {
      "type": "discovery",
      "summary": "What you learned"
    }
  ],
  "artifacts": [
    "path/to/file/created.ts"
  ],
  "open_questions": []
}
```

Then run:
```bash
mc handoff findings.json
```

## Important
- Output the JSON block when done
- Run mc handoff to validate and store
- Do not continue after handoff
```

**TODO:**
- [ ] Create prompt for each persona (11 total)
- [ ] Template variables: {zone}, {task_description}, {task_id}, {worker_id}
- [ ] Test worker outputting valid handoff JSON

---

### Phase 5: Go Bridge Updates

The existing Go service becomes a thin bridge.

**Current responsibilities (keep):**
- WebSocket connection to React UI
- Spawn/kill processes
- Relay stdout to WebSocket

**New responsibilities:**
- Spawn King as Claude Code process with .mission/CLAUDE.md
- Spawn workers as Claude Code processes with persona prompts
- Watch .mission/state/ for changes → emit WebSocket events
- Route UI chat messages to King's stdin

**TODO:**
- [ ] `POST /api/king/start` - Spawn King Claude Code session
- [ ] `POST /api/king/message` - Send message to King's stdin
- [ ] File watcher on .mission/state/ → WebSocket events
- [ ] Update spawn logic to use persona prompts
- [ ] Relay worker stdout to WebSocket

**Spawn King:**
```go
func SpawnKing(workdir string) (*Agent, error) {
    // King uses .mission/CLAUDE.md as its context
    cmd := exec.Command("claude",
        "--workdir", workdir,
        "--resume",  // or start fresh
    )
    // ... setup stdin/stdout pipes
    // ... relay to WebSocket
}
```

**Spawn Worker:**
```go
func SpawnWorker(persona, task, zone, workdir string) (*Agent, error) {
    // Create temp prompt file with task details
    promptFile := createWorkerPrompt(persona, task, zone)
    
    cmd := exec.Command("claude",
        "--workdir", filepath.Join(workdir, zone),
        "--system-prompt", promptFile,
        "--print", task,  // Initial message
    )
    // ... setup pipes, relay to WebSocket
}
```

---

### Phase 6: React UI Updates

Minimal changes - mostly wiring to new WebSocket events.

**TODO:**
- [ ] King chat connected to `/api/king/message`
- [ ] Display phase from `phase_changed` events
- [ ] Display tasks from `task_created`, `task_updated` events
- [ ] Display workers from `worker_spawned`, `worker_completed` events
- [ ] Gate approval dialog from `gate_ready` events
- [ ] Findings view from `findings_ready` events

**New WebSocket events (emitted by Go file watcher):**
```typescript
{ type: "phase_changed", phase: "design" }
{ type: "task_created", task: Task }
{ type: "task_updated", task_id: string, status: string }
{ type: "worker_spawned", worker_id: string, persona: string, zone: string }
{ type: "worker_completed", worker_id: string, task_id: string }
{ type: "gate_ready", phase: string, criteria: GateCriterion[] }
{ type: "findings_ready", task_id: string }
```

---

### Phase 7: Handoff Flow

Workers output JSON. `mc handoff` validates. King reads findings.

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
mc validates via Rust (FFI or CLI)
        │
        ├── Invalid → Error, worker retries
        │
        ▼ Valid
mc stores in .mission/handoffs/
mc computes delta, stores in .mission/findings/
mc updates .mission/state/tasks.json
        │
        ▼
Go file watcher sees change
        │
        ▼
WebSocket: findings_ready
        │
        ▼
King (prompted or polling): cat .mission/findings/<task>.json
        │
        ▼
King synthesizes, decides next
```

**TODO:**
- [ ] `mc handoff` validates JSON schema
- [ ] `mc handoff` calls Rust for semantic validation
- [ ] `mc handoff` stores raw handoff in .mission/handoffs/
- [ ] `mc handoff` compresses findings to .mission/findings/
- [ ] `mc handoff` updates task status in .mission/state/tasks.json
- [ ] Go watcher emits `findings_ready` event

---

### Phase 8: Gate Flow

King checks gates via `mc gate`. User approves in UI.

```
King: All design tasks complete
        │
        ▼
King: mc gate check design
        │
        ├── Not ready → King spawns more workers
        │
        ▼ Ready
King: "Design complete. Ready for Implement?"
        │
        ▼
User clicks "Approve" in UI
        │
        ▼
UI: POST /api/gates/design/approve
        │
        ▼
Go: mc gate approve design
        │
        ▼
mc updates .mission/state/phase.json
mc updates .mission/state/gates.json
        │
        ▼
Go watcher: phase_changed event
        │
        ▼
King sees phase change, creates Implement tasks
```

**TODO:**
- [ ] `mc gate check <phase>` - Returns JSON with criteria status
- [ ] `mc gate approve <phase>` - Updates state files
- [ ] `POST /api/gates/:phase/approve` endpoint
- [ ] Gate approval dialog in React UI

---

### Phase 9: Integration Testing

Test the full loop.

**Test 1: King spawns worker**
```bash
# Start King
mc init
claude --workdir . 

# In King session:
> mc spawn developer "Create hello.py" --zone backend

# Verify worker spawned
mc workers
```

**Test 2: Worker handoff**
```bash
# Worker outputs JSON, runs mc handoff
mc handoff findings.json

# Verify stored
cat .mission/findings/<task-id>.json
cat .mission/state/tasks.json  # Status updated
```

**Test 3: Full workflow**
```
User → King: "Build a login page"
King → mc task create ...
King → mc spawn designer ...
Designer → completes → mc handoff
King → reads findings
King → "Ready for implement?"
User → approves gate
King → mc gate approve design
King → mc spawn developer ...
... continues
```

**TODO:**
- [ ] Test: King spawns worker via mc CLI
- [ ] Test: Worker outputs handoff, mc validates
- [ ] Test: King reads findings from files
- [ ] Test: Gate check and approval flow
- [ ] Test: Phase transition creates new tasks
- [ ] Test: Full Idea → Design → Implement cycle

---

## File Structure After V5

```
mission-control/
├── cmd/
│   └── mc/                   # mc CLI tool
│       ├── main.go
│       ├── spawn.go
│       ├── handoff.go
│       ├── gate.go
│       ├── phase.go
│       ├── task.go
│       └── status.go
│
├── orchestrator/             # Go Bridge (slimmed down)
│   ├── main.go
│   ├── bridge/
│   │   ├── king.go          # Spawn/manage King process
│   │   └── worker.go        # Spawn/manage worker processes
│   ├── watcher/
│   │   └── files.go         # Watch .mission/ for changes
│   └── ws/
│       └── hub.go           # Existing WebSocket hub
│
├── core/                     # Rust (from v4, now with CLI wrapper)
│   ├── workflow/
│   ├── knowledge/
│   └── cli/                  # Rust CLI wrapper
│       └── src/main.rs      # mc-core validate-handoff, etc.
│
├── web/                      # React UI (minimal changes)
│
└── .mission/                 # Per-project state (template)
    ├── CLAUDE.md            # King system prompt
    ├── config.json
    ├── state/
    ├── specs/
    ├── findings/
    ├── handoffs/
    └── prompts/
```

---

## Success Criteria

V5 is done when:

1. [ ] `mc` CLI works (init, spawn, status, handoff, gate, task)
2. [ ] King CLAUDE.md makes Claude Code act as coordinator
3. [ ] King can spawn workers via `mc spawn`
4. [ ] Workers output structured handoff JSON
5. [ ] `mc handoff` validates and stores findings
6. [ ] King reads findings from .mission/findings/
7. [ ] Gate check and approval flow works
8. [ ] Phase transitions work
9. [ ] React UI shows real-time updates via WebSocket
10. [ ] Full cycle: Idea → Design → Implement works end-to-end

---

## What's Deferred to v6

- 3D visualization
- Model cascade (Haiku/Sonnet/Opus routing)
- Token budget enforcement (workers just handoff when done)
- Parallel worker coordination
- Worker health monitoring (Watcher)

---

## Key Differences from Previous V5

| Before | After |
|--------|-------|
| Custom Go orchestration loop | Claude Code IS the orchestrator |
| King calls Anthropic API | King IS a Claude Code session |
| Message queue (King ↔ Steward) | File-based state (.mission/) |
| Steward/Quartermaster agent | Just CLI tools King calls |
| Complex context compilation | Claude Code handles its own context |
| ~2000 lines of Go orchestration | ~500 lines of Go bridge + CLI |

**The insight:** We were reinventing Claude Code. Now we just give it a good system prompt and CLI tools to call.