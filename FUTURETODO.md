# MissionControl — TODO

## Current: v5.1 — Quality of Life (Remaining)

### Repository Cleanup
- [ ] Audit for dead code / unused files
- [ ] Add `CODEOWNERS` for GitHub (optional)

### Testing
- [ ] E2E: Token usage displays correctly

### Homebrew Distribution
- [ ] Create `homebrew-tap` repo (`DarlingtonDeveloper/homebrew-tap`)
- [ ] `brew tap DarlingtonDeveloper/tap && brew install mission-control` works
- [ ] Document in README

### Configuration & Storage
- [ ] Sort sidebar by `lastOpened` descending

### UI Polish

**Loading/Error/Empty States:**
- [ ] Loading spinner while waiting for King response
- [ ] Error state with retry button (WebSocket disconnect, API error)
- [ ] Empty state: no project selected → show wizard
- [ ] Empty state: no agents → "Spawn your first agent" prompt

**UX Improvements:**
- [ ] WebSocket connection indicator (green/red dot in header)
- [ ] King conversation persists in localStorage
- [ ] Clear conversation button
- [ ] Copy response to clipboard button
- [ ] Keyboard shortcut hints (tooltips on hover)

**Visual Polish:**
- [ ] Consistent color palette
- [ ] Agent status indicators (idle/working/error)
- [ ] Typing indicator in King chat

### Developer Experience
- [ ] `.vscode/settings.json` — format on save, recommended settings
- [ ] `.vscode/extensions.json` — recommended extensions list
- [ ] Pre-commit hooks via lefthook or husky (optional)

### Dynamic Project Switching
- [ ] Orchestrator API: `POST /api/projects/select` to switch active project
- [ ] Orchestrator reloads `.mission/state/` watcher on project switch
- [ ] WebSocket broadcasts project change event
- [ ] UI reloads state when project switches (no page refresh needed)
- [ ] Remove need to restart orchestrator with `--workdir` flag

---

## v6 — State Management

Foundation for multi-agent coordination and intelligent work distribution.

### 6.1 JSONL Migration ✅ COMPLETE
**What:** Convert `tasks.json` to `tasks.jsonl` (one JSON object per line)

- [x] One-time migration script
- [x] Update all read/write functions in `mc` CLI
- [x] Update file watcher in Go bridge

**Benefit:** Git merges line-by-line, enables concurrent writes

---

### 6.2 Hash-Based Task IDs ✅ COMPLETE
**What:** Replace sequential/random IDs with content-hash IDs (`mc-a7x2k`)

- [x] SHA256(title + timestamp) truncated to 5 chars
- [x] Update ID generation in `mc task create`
- [x] Migration for existing tasks

**Benefit:** Prevents ID collisions, deterministic retries

---

### 6.3 Audit Trail ✅ COMPLETE
**What:** Append-only `audit/interactions.jsonl` logging all state mutations

- [x] Create audit log format (actor, action, target, timestamp)
- [x] Hook into all `mc` commands that mutate state
- [x] Add `mc audit` command to query history

**Benefit:** Full history, debug replay, compliance support

---

### 6.4 Task Dependencies ✅ COMPLETE
**What:** Add `blocks`/`blockedBy` fields to tasks

- [x] Schema changes to task format
- [x] `mc task create --blocks <id>` flag
- [x] `mc dep add/remove` commands
- [x] Dependency validation (no cycles)
- [x] Auto-update blocked tasks when blockers close

**Benefit:** King distributes work intelligently, workers get actionable tasks

---

### 6.5 Ready Queue Command ✅ COMPLETE
**What:** `mc ready` shows tasks with no open blockers

- [x] Query tasks where all `blockedBy` are closed
- [x] Filter by stage, assignee, labels
- [x] JSON output for King to consume

**Benefit:** Workers ask "what can I do?" and get real answers

---

### 6.6 Dependency Visualization ✅ COMPLETE
**What:** `mc dep tree <id>` shows dependency graph

- [x] Tree view of what blocks what
- [x] Show status of each node
- [x] `mc blocked` shows all blocked tasks and why

---

### 6.7 Git Commit Hooks ✅ COMPLETE
**What:** Auto-commit state changes to git

- [x] Post-mutation hook in `mc` commands (all mutation points wired up)
- [x] Commit with message: `[mc:{category}] {action} {target}`
- [x] Configurable per-category via `.mission/config.json` `auto_commit`
- [ ] Optional: push to remote

**Benefit:** Automatic version history, disaster recovery

---

### 6.8 10-Stage Workflow ✅ COMPLETE
**What:** Expand from 6 phases to 10 stages

- [x] Rename phase.json → stage.json
- [x] Add DISCOVERY, GOAL, REQUIREMENTS, PLANNING, VALIDATION stages
- [x] Update gates.json with new stage gates
- [x] Update UI phase display
- [x] Full Rust + Go + React implementation (see TODO.md for details)
- [x] Migration tooling (`mc migrate`)
- [x] Checkpoint system (Rust schema + Go CLI + API + React UI)
- [x] Ran full pipeline: built Kai Chat UI with 7 sub-agents (2026-02-07)

---

## v7 — Requirements & Traceability

Full traceability from business needs to implementation.

### 7.1 Requirements Directory Structure
**What:** Create `requirements/` with application, entities, capabilities levels

- [ ] Create directory structure
- [ ] Define JSONL schemas for each level
- [ ] `mc req create` command

---

### 7.2 Requirement CRUD Commands
**What:** `mc req create/list/show/update` commands

- [ ] Create requirements at each level
- [ ] List with filters (level, entity, capability, status)
- [ ] Show with full details
- [ ] Update status, acceptance criteria

---

### 7.3 Requirement Hierarchy (derivedFrom)
**What:** Link requirements to parent requirements via `derivedFrom` field

- [ ] Add derivedFrom field to requirement schema
- [ ] Validate parent exists
- [ ] Build hierarchy index for fast traversal

---

### 7.4 Task-to-Requirement Refs
**What:** Add `refs.requirements` field to tasks

- [ ] Schema change to tasks
- [ ] `mc task create --req REQ-CAP-001` flag
- [ ] Display refs in task show

---

### 7.5 Task-to-Spec Refs
**What:** Add `refs.spec` and `refs.specSections` to tasks

- [ ] Schema change to tasks
- [ ] `mc task create --spec SPEC-auth.md --section FR-1` flags
- [ ] Display refs in task show

---

### 7.6 Impact Analysis
**What:** `mc req impact <id>` shows what depends on a requirement

- [ ] Traverse hierarchy downward
- [ ] Find all descendant requirements
- [ ] Find all linked specs and tasks
- [ ] Summary of blast radius

---

### 7.7 Coverage Report
**What:** `mc req coverage` shows implementation status of all requirements

- [ ] For each requirement, find linked tasks
- [ ] Calculate completion percentage
- [ ] Roll up to parent requirements
- [ ] Summary by level/category

---

### 7.8 Spec Status Command
**What:** `mc spec status <file>` shows implementation status of spec sections

- [ ] Parse spec file for FR-X sections
- [ ] Match to tasks via refs
- [ ] Show completion status per section

---

### 7.9 Spec Orphans Detection
**What:** `mc spec orphans` finds spec sections without implementing tasks

- [ ] Parse all specs for sections
- [ ] Find sections with no linked tasks
- [ ] Report gaps

---

### 7.10 Trace Command
**What:** `mc trace <id>` shows full lineage up and down

- [ ] Trace task → spec → requirements → app-level
- [ ] Trace requirement → children → tasks
- [ ] Tree visualization
- [ ] Both directions

---

### 7.11 Requirements Index Cache
**What:** `requirements/index.json` caches hierarchy for fast queries

- [ ] Generate on requirement changes
- [ ] Store parent/child/descendant relationships
- [ ] Invalidate and regenerate on mutations

---

### 7.12 Auto-Generate Tasks from Spec
**What:** `mc task create --from-spec <file>` creates tasks for spec sections

- [ ] Parse spec for FR-X sections
- [ ] Generate task per section (or prompt for confirmation)
- [ ] Auto-link refs

---

## v8 — Infrastructure & Scale

Cost optimization and team features.

### 8.1 Multi-Model Routing (Haiku/Sonnet/Opus)
**What:** Route different task types to different Claude models

- [ ] Configuration for model per persona/task-type
- [ ] Update worker spawn to pass model flag
- [ ] Routing logic (simple tasks → Haiku, complex → Opus)
- [ ] Cost tracking per model

**Sub-features:**
- [ ] 8.1a: Config schema for model mapping
- [ ] 8.1b: Per-persona model defaults
- [ ] 8.1c: Per-task model override
- [ ] 8.1d: Cost tracking and reporting
- [ ] 8.1e: Auto-routing based on task complexity (advanced)

**Benefit:** 10-50x cost reduction for simple tasks

---

### 8.2 Cost Tracking & Budgets
**What:** Track token usage and costs, enforce budgets

- [ ] Parse Claude API responses for token counts
- [ ] Store usage per task/worker/session
- [ ] `mc costs` command for reporting
- [ ] Budget limits with warnings/stops

---

### 8.3 Worker Health Monitoring
**What:** Detect stuck/crashed workers and alert/recover

- [ ] Heartbeat mechanism
- [ ] Timeout detection
- [ ] Alert to King/user
- [ ] Optional auto-restart

---

### 8.4 Headless Mode
**What:** Run MissionControl without UI for CI/CD pipelines

- [ ] CLI-only operation mode
- [ ] Script-friendly JSON output
- [ ] Exit codes for success/failure
- [ ] Pipeline examples (GitHub Actions, etc.)

---

### 8.5 Remote Bridge Deployment
**What:** Deploy Go bridge to remote server for team access

- [ ] Dockerize Go bridge
- [ ] Authentication layer (API keys, OAuth)
- [ ] WebSocket proxying
- [ ] State sync between instances
- [ ] Multi-tenant isolation

**Sub-features:**
- [ ] 8.5a: Docker containerization
- [ ] 8.5b: Authentication middleware
- [ ] 8.5c: HTTPS/WSS termination
- [ ] 8.5d: Multi-project routing
- [ ] 8.5e: User session management
- [ ] 8.5f: Kubernetes deployment manifests (optional)

---

## v9 — Advanced UI

Visual tools for complex workflows.

### 9.1 Requirements Panel
**What:** UI panel showing requirements hierarchy and coverage

- [ ] Tree view of requirements
- [ ] Coverage indicators
- [ ] Click to see linked tasks
- [ ] Filter by level/status

---

### 9.2 Dependency Graph Visualization
**What:** Visual graph of task dependencies

- [ ] D3 or similar graph library
- [ ] Interactive (click nodes, zoom, pan)
- [ ] Show blocked/ready status
- [ ] Critical path highlighting

---

### 9.3 Traceability View
**What:** UI for exploring trace relationships

- [ ] Select task/requirement/spec
- [ ] Show lineage tree
- [ ] Navigate by clicking nodes

---

### 9.4 Audit Log Viewer
**What:** UI for browsing audit history

- [ ] Timeline view of state changes
- [ ] Filter by actor, action, target
- [ ] Diff view for changes

---

## Future: v10 — 3D Visualization

- [ ] React Three Fiber setup
- [ ] Agent avatars in 3D space
- [ ] Zone visualization
- [ ] Camera controls
- [ ] Animations (spawn, complete, handoff)

---

## Summary Table

| ID | Feature | Benefit | Dependencies |
|----|---------|---------|--------------|
| **v6 - State Management** |
| 6.1 | JSONL Migration | High | None |
| 6.2 | Hash-Based IDs | High | None |
| 6.3 | Audit Trail | High | None |
| 6.4 | Task Dependencies | Very High | 6.1 |
| 6.5 | Ready Queue | Very High | 6.4 |
| 6.6 | Dependency Visualization | Medium | 6.4 |
| 6.7 | Git Commit Hooks | High | 6.1 |
| 6.8 | 10-Stage Workflow | Medium | None |
| **v7 - Requirements & Traceability** |
| 7.1 | Requirements Directory | Med-High | None |
| 7.2 | Requirement CRUD | Medium | 7.1 |
| 7.3 | Requirement Hierarchy | High | 7.1 |
| 7.4 | Task-to-Requirement Refs | High | 7.1 |
| 7.5 | Task-to-Spec Refs | High | None |
| 7.6 | Impact Analysis | High | 7.3 |
| 7.7 | Coverage Report | Very High | 7.4 |
| 7.8 | Spec Status | High | 7.5 |
| 7.9 | Spec Orphans | Med-High | 7.5 |
| 7.10 | Trace Command | Very High | 7.3, 7.4, 7.5 |
| 7.11 | Requirements Index | Medium | 7.1 |
| 7.12 | Auto-Generate Tasks | Medium | 7.5 |
| **v8 - Infrastructure** |
| 8.1 | Multi-Model Routing | Very High | None |
| 8.2 | Cost Tracking | High | None |
| 8.3 | Worker Health | Med-High | None |
| 8.4 | Headless Mode | Med-High | None |
| 8.5 | Remote Bridge | High | None |
| **v9 - Advanced UI** |
| 9.1 | Requirements Panel | Med-High | 7.1-7.7 |
| 9.2 | Dependency Graph | Medium | 6.4 |
| 9.3 | Traceability View | Medium | 7.10 |
| 9.4 | Audit Log Viewer | Medium | 6.3 |

---

## Quick Reference

| Version | Focus | Status |
|---------|-------|--------|
| v1 | Agent fundamentals | Done |
| v2 | Go orchestrator | Done |
| v3 | React UI | Done |
| v4 | Rust core | Done |
| v5 | King + mc CLI | Done |
| v5.1 | Quality of life | Partial (UI items superseded → darlington.dev) |
| v6 | 10-stage workflow + OpenClaw | ✅ Done (merged to main) |
| v7 | Requirements & traceability | Future |
| v8 | Infrastructure & scale | Future |
| v9 | Advanced UI (darlington.dev) | Future |
| v10 | 3D visualization | Future |

See [CHANGELOG.md](CHANGELOG.md) for completed version details.
