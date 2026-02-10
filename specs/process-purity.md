# Spec: Process Purity — Unbreakable Commit Enforcement

**Mission:** Make it impossible to merge code to `main` without following MissionControl's workflow.
**Status:** Design
**Dependencies:** OpenClaw agent policies, GitHub Actions, `mc commit`

-----

## Problem

MissionControl enforces a 10-stage workflow with gates, personas, and findings — but none of it matters if someone can bypass the process. Today:

- `validateCommit()` checks findings exist but not who produced them
- `ScopePaths` exists on the Task struct but no validation reads it
- `git add -A` stages everything blindly
- Git hooks are bypassable (`--no-verify`, delete the hook)
- The King agent can write code directly, undermining the delegation model
- Anyone can run `git commit` instead of `mc commit`

The persona field is a freeform string. Creating a task with `--persona security` and doing the work yourself is indistinguishable from a spawned security worker doing it.

## Three Guarantees

Everything merging to `main` must satisfy three properties. Each must be **server-side enforced** — not bypassable by local tooling, flag overrides, or agent cleverness.

### Guarantee 1: PRs require `mc commit --validate-only` to pass

No pull request merges without MissionControl validation.

### Guarantee 2: The King agent cannot write code files

The orchestrator delegates; workers implement. This must be enforced, not just convention.

### Guarantee 3: Only `mc commit` can commit — not raw `git commit`

All commits carry provenance metadata proving they came through MissionControl.

-----

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  LAYER 1: GitHub (server-side, unbreakable)                     │
│                                                                 │
│  Branch protection on main:                                     │
│    - Require PR (no direct push)                                │
│    - Require mc-validate status check from GitHub Actions app   │
│    - Require 1 approval (optional but recommended)              │
│    - No bypass actors (not even admins, if desired)             │
│                                                                 │
│  GitHub Actions CI:                                             │
│    - Runs mc commit --validate-only on every PR                 │
│    - Checks provenance, personas, findings, scope enforcement   │
│    - Workflow file protected by CODEOWNERS                      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  LAYER 2: mc commit (validation logic)                          │
│                                                                 │
│  Enhanced validateCommit() checks:                              │
│    - Provenance metadata on every commit                        │
│    - Persona verification (session ID → agent → persona)        │
│    - Scope path enforcement (files changed match task scope)    │
│    - Findings completeness (all required personas contributed)  │
│    - Stage-appropriate checks (verify needs 3 personas)         │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  LAYER 3: OpenClaw agent policies (defense-in-depth)            │
│                                                                 │
│  King agent (main):                                             │
│    - Write: DENIED                                              │
│    - Edit: DENIED                                               │
│    - exec: restricted (mc commands, sessions_spawn only)        │
│                                                                 │
│  Developer agent:                                               │
│    - Full access: read, write, edit, exec                       │
│                                                                 │
│  Reviewer/Security/Tester agents:                               │
│    - Read-only: read, exec (read-only commands)                 │
└─────────────────────────────────────────────────────────────────┘
```

Layer 1 is the hard enforcement. Layers 2 and 3 are defense-in-depth — they make it harder to cheat locally, but even if bypassed, Layer 1 catches it at merge time.

-----

## Detailed Design

### 1. GitHub Actions Workflow

File: `.github/workflows/mc-validate.yml`

```yaml
name: mc-validate
on:
  pull_request:
    branches: [main]

permissions:
  contents: read
  checks: write
  statuses: write

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for diff analysis

      - name: Install mc CLI
        run: |
          # Build from source or download release binary
          make build-cli
          sudo cp dist/mc /usr/local/bin/

      - name: Install mc-core
        run: |
          make build-core
          sudo cp core/target/release/mc-core /usr/local/bin/

      - name: Validate mission state
        run: mc commit --validate-only --strict

      - name: Validate provenance
        run: mc commit --validate-provenance

      - name: Validate scope enforcement
        run: mc commit --validate-scope --diff-base origin/main
```

**Branch protection configuration:**
- Protected branch: `main`
- Require status checks: `validate` (from GitHub Actions app — not "any source")
- Require branches to be up to date before merging: Yes (strict mode)
- Require PR before merging: Yes
- Include administrators: Yes (nobody bypasses)

**CODEOWNERS protection:**
```
# .github/CODEOWNERS
.github/workflows/mc-validate.yml @DarlingtonDeveloper
```

This ensures the workflow file itself cannot be modified without the repo owner's approval.

### 2. Enhanced `mc commit` Validation

#### 2a. Provenance Metadata

Every `mc commit` embeds provenance in the commit message as a structured trailer:

```
feat: implement login form

MC-Task: mc-3e798
MC-Persona: developer
MC-Session: openclaw-sess-abc123
MC-Agent: developer-agent
MC-Stage: implement
MC-Scope: cmd/mc/login.go,cmd/mc/login_test.go
```

**Implementation:**
```go
// cmd/mc/commit.go — enhanced commit message
func buildCommitMessage(msg string, task Task, sessionInfo SessionInfo) string {
    trailers := fmt.Sprintf(
        "\nMC-Task: %s\nMC-Persona: %s\nMC-Session: %s\nMC-Agent: %s\nMC-Stage: %s\nMC-Scope: %s",
        task.ID, task.Persona,
        sessionInfo.SessionID, sessionInfo.AgentID,
        task.Stage,
        strings.Join(task.ScopePaths, ","),
    )
    return msg + trailers
}
```

#### 2b. Scope Path Enforcement

`mc commit --validate-scope --diff-base origin/main` checks that files changed in the PR match the scope paths declared on tasks.

```go
// validateScope compares git diff against task scope_paths
func validateScope(missionDir string, diffBase string) error {
    // 1. Get files changed vs diff base
    changedFiles := gitDiffFiles(diffBase)

    // 2. Load all done tasks for current stage
    tasks := loadDoneTasks(missionDir)

    // 3. Build allowed file set from task scope_paths
    allowedPaths := collectScopePaths(tasks)

    // 4. Every changed file must be covered by at least one task's scope
    for _, file := range changedFiles {
        if !isPathCovered(file, allowedPaths) {
            // Exempt .mission/ files (state management)
            if strings.HasPrefix(file, ".mission/") {
                continue
            }
            return fmt.Errorf("file %s changed but not covered by any task scope_paths", file)
        }
    }
    return nil
}
```

**Path matching rules:**
- Exact file match: `cmd/mc/login.go`
- Directory glob: `cmd/mc/` covers all files under that directory
- `.mission/` files are always exempt (state management)
- `go.mod`, `go.sum`, `Makefile` exempt (build infrastructure)
- Exemptions configurable in `.mission/config.json`

#### 2c. Persona Verification for Verify Stage

The verify stage requires findings from three distinct personas: `reviewer`, `security`, and `tester`. This is already partially enforced by mc-core's `check_reviewer_requirement()` — extend it.

```rust
// core/mc-core/src/main.rs — enhanced verify gate check
pub fn check_verify_requirements(tasks: &[JsonlTask]) -> Vec<String> {
    let required_personas = ["reviewer", "security", "tester"];
    let mut failures = Vec::new();

    for persona in &required_personas {
        let has_done = tasks.iter().any(|t| {
            t.persona.as_deref() == Some(*persona)
                && t.status.as_deref() == Some("done")
        });
        if !has_done {
            failures.push(format!(
                "Verify stage requires a done task with persona '{}' — none found",
                persona
            ));
        }
    }
    failures
}
```

#### 2d. Provenance Validation in CI

`mc commit --validate-provenance` checks commit trailers:

```go
func validateProvenance(missionDir string) error {
    // 1. Parse MC-* trailers from commit messages in the PR
    commits := getCommitsInPR()

    for _, commit := range commits {
        trailers := parseTrailers(commit.Message)

        // 2. Every code-changing commit must have MC-Task trailer
        if hasCodeChanges(commit) && trailers["MC-Task"] == "" {
            return fmt.Errorf(
                "commit %s changes code but has no MC-Task provenance trailer",
                commit.SHA[:8],
            )
        }

        // 3. Validate task exists and is done
        if taskID := trailers["MC-Task"]; taskID != "" {
            task, err := loadTask(missionDir, taskID)
            if err != nil {
                return fmt.Errorf("commit %s references unknown task %s", commit.SHA[:8], taskID)
            }
            if task.Status != "done" {
                return fmt.Errorf("commit %s references task %s which is not done", commit.SHA[:8], taskID)
            }
        }

        // 4. Validate persona matches task
        if persona := trailers["MC-Persona"]; persona != "" {
            task, _ := loadTask(missionDir, trailers["MC-Task"])
            if task.Persona != persona {
                return fmt.Errorf(
                    "commit %s claims persona %s but task %s has persona %s",
                    commit.SHA[:8], persona, task.ID, task.Persona,
                )
            }
        }
    }

    return nil
}
```

#### 2e. Strict Mode

`mc commit --validate-only --strict` runs all checks:

|Check                |Description                                  |Bypassable?|
|---------------------|---------------------------------------------|-----------|
|Mission exists       |`.mission/` directory present                |No         |
|Stage valid          |Current stage is set                         |No         |
|Tasks exist          |Non-exempt stage has ≥1 task                 |No         |
|Findings complete    |Done tasks have findings ≥200 bytes          |No         |
|Persona coverage     |Verify stage has reviewer + security + tester|No         |
|Integrator check     |Multi-task implement has integrator          |No         |
|Scope enforcement    |Changed files covered by task scope_paths    |No         |
|Provenance trailers  |Code commits have MC-Task metadata           |No         |
|Task-provenance match|Trailer persona matches task persona         |No         |

**No `--force` flag in CI.** The strict flag removes all bypass options. `--force` only exists for local development iteration.

### 3. OpenClaw Agent Policies

Defense-in-depth layer. Even if these are somehow bypassed, Layer 1 (GitHub) catches it.

#### Agent Configuration

```json5
{
  agents: {
    list: [
      {
        id: "main",
        default: true,
        name: "King (Kai)",
        workspace: "~/.openclaw/workspace",
        tools: {
          deny: ["write", "edit", "apply_patch"]
        },
        subagents: {
          allowAgents: ["developer", "researcher", "reviewer", "security", "tester"]
        }
      },
      {
        id: "developer",
        name: "Developer Worker",
        workspace: "~/.openclaw/workspace",
        tools: {
          allow: ["read", "write", "edit", "apply_patch", "exec", "process"]
        }
      },
      {
        id: "reviewer",
        name: "Reviewer Worker",
        workspace: "~/.openclaw/workspace",
        tools: {
          allow: ["read", "exec", "process"],
          deny: ["write", "edit", "apply_patch"]
        }
      },
      {
        id: "security",
        name: "Security Worker",
        workspace: "~/.openclaw/workspace",
        tools: {
          allow: ["read", "exec", "process"],
          deny: ["write", "edit", "apply_patch"]
        }
      },
      {
        id: "tester",
        name: "Tester Worker",
        workspace: "~/.openclaw/workspace",
        tools: {
          allow: ["read", "write", "edit", "apply_patch", "exec", "process"]
        }
      }
    ]
  }
}
```

**Known limitation:** Denying Write/Edit doesn't prevent `echo 'code' > file.go` via exec. The exec allowlist question remains open — test it. If per-agent exec restriction doesn't work, the agent policy is a soft barrier and Layer 1 is the hard one.

### 4. `mc commit` Replaces `git commit`

`mc commit` already wraps `git add -A` + `git commit`. Enhancements:

#### 4a. Selective Staging (replaces `git add -A`)

```go
func stageFiles(missionDir string, task Task) error {
    // Always stage .mission/ state files
    exec.Command("git", "add", ".mission/").Run()

    // Stage only files within task scope_paths
    for _, scopePath := range task.ScopePaths {
        exec.Command("git", "add", scopePath).Run()
    }

    // Warn about unstaged changes outside scope
    unstaged := getUnstagedFiles()
    for _, f := range unstaged {
        if !strings.HasPrefix(f, ".mission/") {
            fmt.Fprintf(os.Stderr, "⚠ Unstaged (out of scope): %s\n", f)
        }
    }

    return nil
}
```

#### 4b. Task-Linked Commits

`mc commit` requires a task ID:

```bash
# Worker commits their work
mc commit -t mc-3e798 -m "implement login form"

# Equivalent to:
# git add <scope_paths for mc-3e798>
# git add .mission/
# git commit -m "implement login form\n\nMC-Task: mc-3e798\nMC-Persona: developer\n..."
```

#### 4c. CI Enforcement of `mc commit`

Even if someone runs `git commit` directly, the CI check catches it:
- Commits without `MC-Task` trailers fail `--validate-provenance`
- The branch protection prevents merge
- No local enforcement needed — server handles it

-----

## Implementation Plan

### Phase 1: CI Pipeline (unbreakable foundation)
1. Create `.github/workflows/mc-validate.yml`
1. Add `mc commit --validate-only --strict` (enhanced checks)
1. Configure branch protection on `main` with required status check
1. Add `CODEOWNERS` for workflow protection
1. Test: verify a PR without proper mission state cannot merge

### Phase 2: Provenance & Scope
1. Add `MC-*` trailers to `mc commit` message builder
1. Implement `mc commit --validate-provenance`
1. Implement `mc commit --validate-scope --diff-base`
1. Add `--task` / `-t` flag to `mc commit` for task-linked commits
1. Replace `git add -A` with selective staging based on scope_paths
1. Test: verify commits without trailers are rejected in CI

### Phase 3: Verify Stage Personas
1. Extend `check_verify_requirements()` in mc-core to require 3 personas
1. Update `advanceStageChecked()` in stage.go to call enhanced check
1. Update existing TDD tests that expect 3-persona enforcement
1. Test: verify stage cannot advance without reviewer + security + tester findings

### Phase 4: OpenClaw Agent Policies (defense-in-depth)
1. Test exec allowlist per-agent (the critical unknown)
1. Configure King agent with Write/Edit denied
1. Configure worker agents with appropriate permissions
1. Test: verify King cannot create files, workers can
1. Document findings on exec restriction limitations

### Phase 5: Scope Path Validation in `mc commit`
1. Implement `isPathCovered()` with glob matching
1. Add configurable exemptions (`.mission/`, `go.mod`, `Makefile`)
1. Wire into `validateCommit()` flow
1. Test: verify out-of-scope file changes are rejected

-----

## Migration

Existing workflows continue to work. Changes are additive:

- Old commits without `MC-*` trailers: CI only validates trailers on new commits (check diff base)
- `mc commit` without `-t` flag: backwards-compatible warning, not error (grace period)
- Branch protection: enable after CI workflow is proven stable
- Grace period: 1 week of CI running in report-only mode before enforcement

-----

## What This Doesn't Solve

- **Local development iteration:** Developers can still `git commit` locally for WIP. The enforcement is at merge time, not commit time.
- **Admin bypass:** GitHub org owners can always bypass branch protection. This is a people problem, not a tooling problem. Recommendation: enable "include administrators" on the rule.
- **Workflow file tampering via PR:** The `pull_request` trigger runs the workflow from the PR branch's version. Use `pull_request_target` if this is a concern (runs workflow from `main`). Trade-off: `pull_request_target` has different security implications for fork-based workflows.
- **Cost of enforcement:** Every PR now runs `mc commit --validate-only` which requires `mc` and `mc-core` binaries in CI. Build time is ~30s. Cache the binaries.

-----

## Success Criteria

1. A PR with code changes but no `MC-Task` trailer cannot merge
1. A PR where changed files don't match any task's `scope_paths` cannot merge
1. A PR advancing past verify without reviewer + security + tester findings cannot merge
1. The King agent (OpenClaw main) cannot create or edit code files
1. The CI workflow file cannot be modified without CODEOWNERS approval
1. All of the above hold even if `--force`, `--no-verify`, or manual `git commit` are used locally
