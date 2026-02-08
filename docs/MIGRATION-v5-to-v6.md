# Migration Guide: v5 → v6

## Overview

v6 replaces the 6-phase workflow with a 10-stage workflow and adds session continuity via checkpoints and briefings.

## Automatic Migration

Run `mc migrate` in your project directory. This will:

1. Read `.mission/state/phase.json` → write `.mission/state/stage.json`
2. Map phase names: `idea` → `discovery` (all others keep their names)
3. Regenerate `.mission/state/gates.json` with 10 entries
4. Rewrite `.mission/state/tasks.json` — `phase` field → `stage`
5. Create `.mission/orchestrator/` directory for checkpoints

```bash
cd /path/to/your/project
mc migrate
```

## What Changed

### Terminology

| v5 | v6 |
|----|-----|
| Phase | Stage |
| `phase.json` | `stage.json` |
| `mc phase` | `mc stage` (alias: `mc phase`) |
| `mc phase next` | `mc stage next` |
| `mc task create --phase` | `mc task create --stage` |
| `mc gate check <phase>` | `mc gate check <stage>` |
| `phase_changed` event | `stage_changed` event |

### Stage Mapping

| v5 Phase | v6 Stage | Notes |
|----------|----------|-------|
| Idea | Discovery | Renamed |
| — | Goal | New |
| — | Requirements | New |
| — | Planning | New |
| Design | Design | Unchanged |
| Implement | Implement | Unchanged |
| Verify | Verify | Unchanged |
| — | Validate | New |
| Document | Document | Unchanged |
| Release | Release | Unchanged |

### Persona Changes

| Persona | v5 Phase | v6 Stage |
|---------|----------|----------|
| Researcher | Idea | Discovery |
| Analyst | — | Goal (new) |
| Requirements Engineer | — | Requirements (new) |
| Architect | Design | Planning |
| Designer | Design | Design |
| Developer | Implement | Implement |
| Debugger | Implement | Implement |
| Reviewer | Verify | Verify |
| Security | Verify | Verify |
| Tester | Verify | Verify |
| QA | Verify | Validate |
| Docs | Document | Document |
| DevOps | Release | Release |

### New CLI Commands

```bash
# Session continuity
mc checkpoint                    # Create checkpoint snapshot
mc checkpoint restart [--from <id>]  # Restart session with briefing
mc checkpoint status             # Session health check
mc checkpoint history            # List past sessions

# Migration
mc migrate                       # Convert v5 project to v6
```

### New `.mission/` Structure

```
.mission/
├── state/
│   ├── stage.json      # Was: phase.json
│   ├── tasks.json      # Updated: "phase" field → "stage"
│   ├── gates.json      # Updated: 10 entries instead of 6
│   └── workers.json    # Unchanged
├── orchestrator/       # New
│   ├── checkpoints/    # Checkpoint JSON snapshots
│   ├── current.json    # Current session state
│   └── sessions.jsonl  # Session history log
└── ...                 # Other dirs unchanged
```

### New API Endpoints

```
POST   /api/checkpoints           # Create checkpoint
GET    /api/checkpoint/status     # Session health (green/yellow/red)
GET    /api/checkpoint/history    # Session history
POST   /api/checkpoint/restart    # Restart with briefing
```

### New WebSocket Events

| Event | Description |
|-------|-------------|
| `checkpoint_created` | Auto-checkpoint created (gate approval, shutdown) |
| `session_restarted` | Session restarted with new briefing |

## Manual Migration (if `mc migrate` unavailable)

1. Rename `state/phase.json` → `state/stage.json`
2. If current phase is `idea`, change to `discovery`
3. In `tasks.json`, rename all `"phase"` fields to `"stage"` and map `"idea"` → `"discovery"`
4. Regenerate `gates.json` by running `mc init` in a temp directory and copying the gates
5. Create `orchestrator/` directory: `mkdir -p .mission/orchestrator/checkpoints`
