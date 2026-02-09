# Spec: Enforce Stage-Scoped Task Creation

## Problem

`mc task create` allows creating tasks for any stage regardless of the current stage. This enables front-loading all tasks at mission start, which defeats the progressive refinement model where each stage's output informs the next stage's tasks.

## Solution

Add a validation check in `mc task create` that rejects tasks whose stage is ahead of the current stage.

### Stage Ordering

```
discovery < goal < requirements < planning < design < implement < verify < validate < document < release
```

Each stage has an index (0-9). A task's stage index must be <= the current stage index.

### Behaviour

```
$ mc stage
Current stage: discovery

$ mc task create --name "Write unit tests" --stage verify
Error: cannot create task for stage "verify" — current stage is "discovery".
       Advance to "verify" first, or create a task for the current stage.

$ mc task create --name "Assess existing codebase" --stage discovery
Created task abc123def0: "Assess existing codebase" (discovery)
```

### Flag: `--force`

For exceptional cases (e.g., migrating from an existing project, restructuring), allow `--force` to bypass the check with a warning:

```
$ mc task create --name "Write unit tests" --stage verify --force
Warning: creating task for future stage "verify" (current: "discovery"). This bypasses progressive refinement.
Created task abc123def0: "Write unit tests" (verify)
```

### Implementation

**File:** `cmd/mc/task.go`

1. In the `create` subcommand handler, after parsing flags:
   - Read current stage from `.mission/state/stage.json`
   - Compare task stage index against current stage index
   - If task stage > current stage and `--force` not set, error with message
   - If `--force` set, print warning and continue

2. Add `--force` flag to the `create` subcommand.

**File:** `orchestrator/core/stages.go` (or wherever stage ordering is defined)

- Export a `StageIndex(name string) int` function if one doesn't exist
- The 10-stage ordering is already defined somewhere — reuse it

### Tests

1. Create task at current stage → succeeds
2. Create task at past stage → succeeds
3. Create task at future stage → errors with clear message
4. Create task at future stage with `--force` → succeeds with warning
5. Create task when no stage is set → succeeds (no restriction)

### Edge Cases

- **No stage set yet:** Allow any task (mission hasn't started)
- **Task stage equals current stage:** Allow (same stage is fine)
- **Past stages:** Allow — sometimes you need to add a discovery task while in design (you found a gap)

### Not In Scope

- Gate criteria validation (separate TODO item)
- Auto-advancing stages (separate TODO item)
- Changes to the Go orchestrator or REST API (this is CLI only)
