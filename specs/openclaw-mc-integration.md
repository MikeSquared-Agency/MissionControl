# SPEC: OpenClaw ↔ MissionControl Integration

> Kai (OpenClaw) as King, MissionControl as the process layer.

## Problem

Kai spawns sub-agents via OpenClaw but bypasses MissionControl entirely. No briefings, no structured findings, no gate validation, no audit trail. When something breaks, there's no paper trail — just session transcripts buried in JSONL files.

## Goal

Every piece of work flows through MC's 10-stage pipeline with full traceability:

```
Objective → Tasks → Briefing → Worker → Findings → Gate → Next Stage
```

Every decision is a file on disk. Every worker gets exactly the context it needs (small, focused briefings) instead of inheriting a massive conversation context.

## Architecture

```
┌─────────────┐         ┌──────────────────┐
│   Kai        │         │  MissionControl   │
│  (King)      │────────▶│  (.mission/)      │
│  OpenClaw    │         │                   │
│  main agent  │◀────────│  Files on disk    │
└──────┬───────┘         └────────┬──────────┘
       │                          │
       │ spawn sub-agent          │ watcher detects changes
       ▼                          ▼
┌─────────────┐         ┌──────────────────┐
│  Worker      │         │  Orchestrator     │
│  (sub-agent) │────────▶│  (mc serve)       │
│  reads brief │         │  hub → dashboard  │
│  writes finds│         │                   │
└──────────────┘         └──────────────────┘
```

### Components

1. **Kai (King)** — decides what to do, creates tasks, writes briefings, reads findings, advances stages
2. **MissionControl files** — `.mission/` directory is the source of truth
3. **Workers** — OpenClaw sub-agents that read a briefing file and write a findings file
4. **Orchestrator** — `mc serve` watches files, broadcasts to dashboard via WebSocket
5. **Bridge** — connects orchestrator to OpenClaw gateway for session lifecycle events

## Worker Lifecycle

### 1. Kai Creates a Task

```bash
mc task create "Implement user auth" -z backend -s implement
# Returns task ID: a1b2c3d4e5
```

### 2. Kai Writes a Briefing

Before spawning a worker, Kai writes a structured briefing:

```
.mission/handoffs/a1b2c3d4e5-briefing.json
```

Schema:
```json
{
  "version": 1,
  "task_id": "a1b2c3d4e5",
  "task_name": "Implement user auth",
  "stage": "implement",
  "zone": "backend",
  "persona": "developer",
  
  "objective": "Add Supabase auth to the /api/protected routes",
  
  "context": {
    "summary": "Brief description of current state",
    "files_to_read": [
      "lib/supabase/server.ts",
      "middleware.ts"
    ],
    "files_to_modify": [
      "app/api/protected/route.ts"
    ],
    "decisions": [
      "Using cookie-based Supabase auth (decided 2026-02-08)",
      "No new dependencies allowed"
    ],
    "constraints": [
      "Must pass existing TypeScript strict mode",
      "Follow patterns in CLAUDE.md"
    ]
  },
  
  "dependencies": {
    "completed_tasks": ["f6g7h8i9j0"],
    "findings_to_read": [
      ".mission/findings/f6g7h8i9j0.md"
    ]
  },
  
  "budget": {
    "max_tokens": 50000,
    "timeout_seconds": 120,
    "max_files_modified": 5
  },
  
  "expected_output": {
    "findings_path": ".mission/findings/a1b2c3d4e5.md",
    "branch_name": "mc/a1b2c3d4e5-user-auth",
    "tests_required": true
  }
}
```

### 3. Kai Spawns the Worker

```
sessions_spawn(
  task: "You are an MC worker. Read your briefing at .mission/handoffs/a1b2c3d4e5-briefing.json and follow it exactly. Write findings to .mission/findings/a1b2c3d4e5.md when done. Do not exceed your token budget.",
  label: "mc-worker-a1b2c3d4e5"
)
```

Kai also updates the task status:
```bash
mc task update a1b2c3d4e5 --status in_progress
```

### 4. Worker Does the Work

The worker:
1. Reads the briefing file
2. Reads any dependency findings
3. Reads only the files listed in `files_to_read`
4. Makes changes to `files_to_modify`
5. Creates a branch if specified
6. Writes findings when done

### 5. Worker Writes Findings

```
.mission/findings/a1b2c3d4e5.md
```

Structure:
```markdown
# Findings: Implement user auth

## Task
- ID: a1b2c3d4e5
- Status: complete | partial | blocked | failed
- Worker: mc-worker-a1b2c3d4e5

## Summary
One paragraph of what was done.

## Changes Made
- `app/api/protected/route.ts` — added auth check
- `middleware.ts` — added /api/protected to matcher

## Decisions Made
- Used `getUser()` instead of `getSession()` for security (Supabase recommendation)

## Blockers / Issues
- None

## Files Modified
- app/api/protected/route.ts (new)
- middleware.ts (modified)

## Tests
- Added test for 401 on unauthenticated request
- Added test for 200 on authenticated request

## Token Usage
- ~12,000 tokens used of 50,000 budget

## Recommendations for Next Worker
- The auth middleware should be extracted to a shared util if more routes need it
```

### 6. Watcher Detects Changes

The file watcher in `mc serve` detects:
- New file in `.mission/findings/` → broadcasts `findings_ready` event
- Task status change → broadcasts `task_updated` event

Dashboard updates in real time.

### 7. Kai Reads Findings

Kai reads the small findings file (not the worker's full transcript). This is the handoff — structured, focused, cheap.

Kai then:
- Updates task status: `mc task update a1b2c3d4e5 --status done`
- Decides next steps based on findings
- May spawn another worker with a new briefing

### 8. Gate Validation

Before advancing stages, Kai runs gate checks:

```bash
mc-core check-gate implement .mission/
```

This validates:
- All tasks for the stage are done
- Required criteria are met
- No unresolved blockers in findings

Only when the gate passes does Kai advance:
```bash
mc stage next
```

## Bridge: Session Lifecycle → Tracker

The Go bridge in `mc serve` listens for OpenClaw gateway events:

| Gateway Event | Bridge Action |
|---|---|
| `session.created` (label starts with `mc-worker-`) | Register worker in tracker |
| `session.updated` | Update worker status (busy/idle) |
| `session.completed` | Mark worker complete, deregister |
| `session.error` | Mark worker failed |

This is ~50-80 lines in the bridge event handler. The tracker already broadcasts to the hub, dashboard picks it up.

### Worker Registration

When the bridge sees a new MC worker session:

```go
// Extract task ID from label: "mc-worker-a1b2c3d4e5" → "a1b2c3d4e5"
taskID := strings.TrimPrefix(label, "mc-worker-")

tracker.Register(TrackedProcess{
    ID:       sessionKey,
    Persona:  briefing.Persona,  // read from briefing file
    TaskID:   taskID,
    Zone:     briefing.Zone,
    Status:   "running",
    StartedAt: time.Now(),
})
```

## File Structure

```
.mission/
├── config.json              # Project config
├── CLAUDE.md                # Worker system prompt
├── state/
│   ├── stage.json           # Current stage
│   ├── gates.json           # Gate status
│   ├── tasks.jsonl          # All tasks
│   └── workers.json         # Active workers
├── handoffs/
│   ├── {task_id}-briefing.json   # Worker briefings (Kai writes)
│   └── {task_id}-debrief.json    # Post-completion summary (optional)
├── findings/
│   ├── {task_id}.md              # Worker findings (worker writes)
│   └── {task_id}-review.md       # Kai's review (optional)
├── specs/                   # Feature specifications
├── checkpoints/             # State snapshots
└── orchestrator/            # Orchestrator internal state
```

## Audit Trail

Every action is logged to `.mission/state/audit.jsonl`:

```jsonl
{"ts":"2026-02-09T20:00:00Z","actor":"kai","action":"task_created","task_id":"a1b2c3d4e5","details":"Implement user auth"}
{"ts":"2026-02-09T20:00:05Z","actor":"kai","action":"briefing_written","task_id":"a1b2c3d4e5","path":".mission/handoffs/a1b2c3d4e5-briefing.json"}
{"ts":"2026-02-09T20:00:10Z","actor":"kai","action":"worker_spawned","task_id":"a1b2c3d4e5","session":"mc-worker-a1b2c3d4e5"}
{"ts":"2026-02-09T20:02:30Z","actor":"mc-worker-a1b2c3d4e5","action":"findings_written","task_id":"a1b2c3d4e5","status":"complete"}
{"ts":"2026-02-09T20:02:35Z","actor":"kai","action":"task_completed","task_id":"a1b2c3d4e5"}
{"ts":"2026-02-09T20:02:40Z","actor":"kai","action":"gate_checked","stage":"implement","result":"pass"}
{"ts":"2026-02-09T20:02:45Z","actor":"kai","action":"stage_advanced","from":"implement","to":"verify"}
```

## What Changes

### Go (orchestrator)
1. **Bridge event handler** — listen for `session.*` events, register/deregister workers (~80 lines)
2. **buildState** — already fixed to unwrap gates/tasks properly
3. **Watcher** — already watches `.mission/state/`. Add `.mission/findings/` and `.mission/handoffs/` to watch paths

### Kai (behaviour)
1. **Always create tasks** before spawning workers
2. **Always write briefings** to `.mission/handoffs/`
3. **Always read findings** instead of relying on session results
4. **Always run gate checks** before advancing stages
5. **Label workers** with `mc-worker-{task_id}` convention

### Rust (mc-core)
- No changes needed. `check-gate` and `validate-handoff` already work.
- May add `compile-briefing` command later to auto-generate briefings from task + context.

### Dashboard (Darlington)
- Worker cards already render from tracker data — will work automatically
- May add a "findings" panel to show findings content inline

## Implementation Order

1. **Kai behaviour change** — start writing briefings and readings findings NOW (no code needed)
2. **Bridge session events** — ~80 lines Go, PR on MC
3. **Watcher expansion** — add findings/handoffs to watch paths
4. **Dashboard findings panel** — show findings inline in mission view
5. **Gate validation integration** — Kai calls `mc-core check-gate` before stage transitions

## Success Criteria

- Every spawned worker has a briefing file on disk
- Every completed worker has a findings file on disk
- Dashboard shows workers appearing and completing in real time
- Stage transitions only happen after gate validation
- Any issue can be traced: objective → task → briefing → findings → decision
