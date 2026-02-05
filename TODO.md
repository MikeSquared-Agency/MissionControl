# v6 Integration TODO List

## Legend
- 🔴 = Breaking change / high risk
- 🟡 = Medium effort / some coordination
- 🟢 = Low risk / isolated change

---

## A. Rust Core (`core/`)

### A1. 10-Stage Workflow Engine 🔴

- [ ] **A1.1** Rename `core/workflow/src/phase.rs` → `core/workflow/src/stage.rs`
- [ ] **A1.2** Replace `Phase` enum with `Stage` enum (10 variants: Discovery, Goal, Requirements, Planning, Design, Implement, Verify, Validate, Document, Release)
- [ ] **A1.3** Update `Stage::next()` — 9 transitions instead of 5
- [ ] **A1.4** Update `Stage::all()` — return 10 stages
- [ ] **A1.5** Update `Stage::as_str()` — 10 string representations
- [ ] **A1.6** Update `Default for Stage` — `Discovery` instead of `Idea`
- [ ] **A1.7** Update `core/workflow/src/lib.rs` — rename `pub mod phase` → `pub mod stage`, update re-exports
- [ ] **A1.8** Global find/replace: `Phase` → `Stage` across all Rust crates (workflow, knowledge, mc-protocol, mc-core, ffi)

### A2. Gate Criteria 🟡

- [ ] **A2.1** Update `Gate::new()` to accept `Stage` instead of `Phase`
- [ ] **A2.2** Implement `default_criteria_for_stage()` with criteria for all 10 stages
- [ ] **A2.3** Update `Gate.phase` field → `Gate.stage` in struct definition
- [ ] **A2.4** Update gate ID generation: `gate-{stage.as_str()}`

### A3. Task Struct 🟡

- [ ] **A3.1** `Task.phase: Phase` → `Task.stage: Stage` in `core/workflow/src/task.rs`
- [ ] **A3.2** Update all task creation/filtering logic referencing `.phase`

### A4. Engine Updates 🟡

- [ ] **A4.1** `WorkflowEngine.current_phase` → `WorkflowEngine.current_stage`
- [ ] **A4.2** `current_phase()` → `current_stage()`
- [ ] **A4.3** `can_transition()` — works as-is (delegates to `Stage::next()`)
- [ ] **A4.4** `WorkflowEngine::new()` — initializes 10 gates instead of 6

### A5. mc-core CLI 🟡

- [ ] **A5.1** Update `check-gate` command: accept 10 stage names, update error message listing valid stages
- [ ] **A5.2** Update `validate-handoff`: if handoff JSON has `phase` field, accept as alias for `stage`
- [ ] **A5.3** Update help text and `--help` output

### A6. Knowledge & Protocol Crates 🟡

- [ ] **A6.1** `core/knowledge/` — update any `Phase` references in handoff validation
- [ ] **A6.2** `core/mc-protocol/` — update shared data structures if they reference `Phase`

### A7. Rust Tests 🔴

- [ ] **A7.1** Update all 24 workflow crate tests (phase transitions, gate checks, task creation)
- [ ] **A7.2** Add tests for new stages (Discovery, Goal, Requirements, Planning, Validate)
- [ ] **A7.3** Add test: 9 sequential transitions from Discovery → Release
- [ ] **A7.4** Update knowledge crate tests referencing phases
- [ ] **A7.5** Update mc-protocol tests if applicable
- [ ] **A7.6** `cargo test` passes across workspace
- [ ] **A7.7** `cargo clippy` clean

---

## B. Go Layer (`orchestrator/`, `cmd/mc/`)

### B1. Type Definitions 🔴

- [ ] **B1.1** `orchestrator/v4/types.go`: Rename `Phase` → `Stage`, add 4 new constants
- [ ] **B1.2** Update `AllStages()` (was `AllPhases()`) — return 10 stages
- [ ] **B1.3** Update `Stage.Next()` (was `Phase.Next()`) — 9 transitions
- [ ] **B1.4** `Task.Phase` → `Task.Stage` in struct
- [ ] **B1.5** `Gate.Phase` → `Gate.Stage` in struct
- [ ] **B1.6** `GateResult.Phase` → `GateResult.Stage` in `orchestrator/core/client.go`

### B2. CLI Commands 🟡

- [ ] **B2.1** `mc phase` → `mc stage` (new command, keep `mc phase` as deprecated alias)
- [ ] **B2.2** `mc task create --phase` → `--stage` (keep `--phase` as deprecated alias)
- [ ] **B2.3** `mc gate check/approve` — accept new stage names
- [ ] **B2.4** `mc init` — scaffold `stage.json` instead of `phase.json`, generate 10 gates in `gates.json`
- [ ] **B2.5** `mc status` — output `stage` field instead of `phase`
- [ ] **B2.6** Add `mc migrate` command: reads `phase.json` → writes `stage.json`, maps `idea` → `discovery`, regenerates `gates.json`

### B3. `.mission/` File Changes 🟡

- [ ] **B3.1** `mc init`: create `state/stage.json` (not `phase.json`)
- [ ] **B3.2** `mc init`: `gates.json` has 10 entries
- [ ] **B3.3** Update `CLAUDE.md` template with 10-stage instructions
- [ ] **B3.4** Update persona prompt templates with new stage assignments

### B4. OpenClaw Integration 🔴

- [ ] **B4.1** Create `api/openclaw.go` — `POST /api/openclaw/event`, `GET /api/openclaw/status`, `POST /api/openclaw/send`
- [ ] **B4.2** Create `bridge/openclaw.go` — WS client connecting to `ws://127.0.0.1:18789`
- [ ] **B4.3** Event relay: OpenClaw agent events → MC WebSocket hub
- [ ] **B4.4** Message relay: React UI chat → OpenClaw agent session
- [ ] **B4.5** Remove `bridge/king.go` — King tmux lifecycle
- [ ] **B4.6** Remove `api/king.go` — King start/stop/message endpoints
- [ ] **B4.7** Add `--openclaw-gateway` flag to `mc serve`
- [ ] **B4.8** Fallback logic: if OpenClaw WS disconnects, optionally spawn King as backup

### B5. Go Tests 🟡

- [ ] **B5.1** Update 8 Go CLI tests (`cmd/mc/mc_test.go`) — phase → stage references
- [ ] **B5.2** Add test for `mc migrate` command
- [ ] **B5.3** Add test for `mc stage next` transitioning through 10 stages
- [ ] **B5.4** Add test for OpenClaw endpoint handlers

---

## C. React UI (`web/`)

### C1. Type Updates 🔴

- [ ] **C1.1** Update `Persona.phase` type → `Persona.stage` with 10 stage values
- [ ] **C1.2** Update `DEFAULT_PERSONAS` — reassign stages per §6.8.5 table
- [ ] **C1.3** Update phase constants/labels → stage constants/labels throughout
- [ ] **C1.4** Update Zustand store: `phase` → `stage` in state shape

### C2. Component Updates 🟡

- [ ] **C2.1** `SettingsPanel.tsx` — update `phases` array → `stages`, update `phaseLabels` → `stageLabels`, add 4 new stages
- [ ] **C2.2** WorkflowMatrix / phase progression display — expand to 10 stages, adjust layout
- [ ] **C2.3** Gate approval dialog — accept 10 stage names
- [ ] **C2.4** King Mode panel → OpenClaw Mode panel (status, chat relay, channel badges)
- [ ] **C2.5** Workers panel — show Agent Teams members + OpenClaw sub-agents
- [ ] **C2.6** Add channel indicator badges (WhatsApp, Telegram, Slack, Discord, WebChat icons)

### C3. WebSocket Events 🟡

- [ ] **C3.1** Update WS event handlers: `phase_changed` → `stage_changed`
- [ ] **C3.2** Add handler for OpenClaw connection status events

### C4. React Tests 🔴

- [ ] **C4.1** Update `types.test.ts` — persona stage assertions (`'idea'` → `'discovery'`), coverage for all 10 stages
- [ ] **C4.2** Update remaining 130+ web tests referencing phases
- [ ] **C4.3** Add tests for OpenClaw Mode panel
- [ ] **C4.4** Add tests for 10-stage workflow display

---

## D. OpenClaw Skill & Configuration

### D1. MissionControl Skill 🟢

- [ ] **D1.1** Create `~/.openclaw/workspace/skills/mission-control/SKILL.md` with 10-stage instructions
- [ ] **D1.2** Document all `mc` CLI commands available to the agent
- [ ] **D1.3** Include stage gate criteria reference
- [ ] **D1.4** Include persona-to-stage mapping reference

### D2. OpenClaw Configuration 🟢

- [ ] **D2.1** Configure `openclaw.json` — agent model, sub-agent defaults, compaction settings
- [ ] **D2.2** Set up pre-compaction memory flush prompt referencing stages
- [ ] **D2.3** Configure channel connectivity (WhatsApp, Telegram minimum)
- [ ] **D2.4** Set up project symlinks: `~/.openclaw/workspace/projects/<name>` → project `.mission/`

### D3. Agent Teams Setup 🟡

- [ ] **D3.1** Enable `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` in settings
- [ ] **D3.2** Test Agent Teams spawning with MC worker personas
- [ ] **D3.3** Verify workers can write to `.mission/findings/` and call `mc handoff`
- [ ] **D3.4** Test file watcher picks up Agent Teams output

---

## E. Documentation & Migration

### E1. Documentation 🟢

- [ ] **E1.1** Update `ARCHITECTURE.md` — 10-stage diagram, new stage table
- [ ] **E1.2** Update `core/README.md` — Stage enum, gate criteria, state diagram
- [ ] **E1.3** Update `CHANGELOG.md` — v6 entry with all changes
- [ ] **E1.4** Update `docs/archive/V4-RUST-CONTRACTS.md` — Stage structs (or mark as superseded)
- [ ] **E1.5** Update `docs/archive/V4-IMPLEMENTATION.md` — mark Phase 1 Rust items as needing stage update
- [ ] **E1.6** Write `docs/MIGRATION-v5-to-v6.md` — step-by-step for existing projects

### E2. Migration Tooling 🟡

- [ ] **E2.1** `mc migrate` command implementation (Go)
- [ ] **E2.2** Phase-to-stage mapping: `idea` → `discovery`, others keep names
- [ ] **E2.3** Auto-regenerate `gates.json` with 10 entries
- [ ] **E2.4** Rename `phase.json` → `stage.json` preserving current value
- [ ] **E2.5** Update `tasks.json` — rewrite `phase` field → `stage` in all task records
- [ ] **E2.6** Test migration on existing `.mission/` directories

---

## F. Integration Testing

- [ ] **F1** End-to-end: OpenClaw agent → `mc task create --stage implement` → worker → handoff → gate approve → stage transition
- [ ] **F2** Multi-channel: send gate approval from WhatsApp, verify React UI updates
- [ ] **F3** Compaction: trigger memory flush, verify `.mission/` state summary persists
- [ ] **F4** Fallback: disconnect OpenClaw, verify Go Bridge falls back to King mode
- [ ] **F5** Migration: run `mc migrate` on v5 project, verify 10-stage operation
- [ ] **F6** Full stage walk: traverse all 10 stages Discovery → Release with gate approvals
- [ ] **F7** Checkpoint round-trip: create checkpoint → restart → verify briefing injected → verify state continuity
- [ ] **F8** Auto-checkpoint: approve a gate, verify checkpoint auto-created and git-committed
- [ ] **F9** `cargo test && go test ./... && npm test` — all green across all layers

---

## G. Session Continuity (Checkpoints & Briefings)

### G1. Rust: Checkpoint Schema 🟡

- [ ] **G1.1** Extend `core/knowledge/src/checkpoint.rs` `Checkpoint` struct with `session_id`, `decisions: Vec<String>`, `blockers: Vec<String>`, `stage` (replacing `phase`)
- [ ] **G1.2** Add `CheckpointCompiler` — takes checkpoint JSON → produces ~500 token markdown briefing
- [ ] **G1.3** Add `mc-core checkpoint-compile <file>` command to `mc-core` CLI
- [ ] **G1.4** Add `mc-core checkpoint-validate <file>` — schema validation for checkpoint JSON
- [ ] **G1.5** Unit tests for checkpoint compilation (verify token budget, required sections)

### G2. Go: CLI Commands 🟡

- [ ] **G2.1** `mc checkpoint` — snapshot stage + gates + decisions + tasks + blockers → write to `.mission/orchestrator/checkpoints/<timestamp>.json`, auto-commit to git
- [ ] **G2.2** `mc checkpoint restart [--from <id>]` — create final checkpoint, call `mc-core checkpoint-compile`, restart OpenClaw session with briefing, log to `sessions.jsonl`
- [ ] **G2.3** `mc checkpoint status` — read `current.json` + token estimate + session duration → output health recommendation
- [ ] **G2.4** `mc checkpoint history` — parse `sessions.jsonl`, display session list with final checkpoint summaries
- [ ] **G2.5** `mc checkpoint query <id>` — load historical checkpoint, display formatted summary
- [ ] **G2.6** Create `.mission/orchestrator/` directory in `mc init` scaffold

### G3. Go: Auto-Checkpoint Triggers 🟡

- [ ] **G3.1** Wire gate approval handler → auto-create checkpoint after `mc gate approve`
- [ ] **G3.2** Token threshold monitor — periodically check conversation token count, checkpoint at 50k (configurable in `config.json`)
- [ ] **G3.3** Graceful shutdown hook — checkpoint on `mc serve` stop / SIGTERM
- [ ] **G3.4** Pre-compaction integration — OpenClaw skill calls `mc checkpoint` before memory flush

### G4. Go: API Endpoints 🟢

- [ ] **G4.1** `POST /api/checkpoint` — create checkpoint (used by UI + auto-triggers)
- [ ] **G4.2** `POST /api/checkpoint/restart` — restart with briefing
- [ ] **G4.3** `GET /api/checkpoint/status` — session health JSON
- [ ] **G4.4** `GET /api/checkpoint/history` — session list JSON

### G5. React UI 🟢

- [ ] **G5.1** Token health indicator in OpenClaw Mode panel (green/yellow/red based on token count)
- [ ] **G5.2** "Restart Session" button with confirmation dialog
- [ ] **G5.3** Checkpoint history viewer (expandable list with summaries)
- [ ] **G5.4** Auto-checkpoint notification toast when triggered

### G6. OpenClaw Skill Integration 🟢

- [ ] **G6.1** Update MissionControl skill: on startup, read `.mission/orchestrator/current.json` and include latest briefing if restart just occurred
- [ ] **G6.2** Update pre-compaction flush prompt: call `mc checkpoint` first, then write briefing summary to `memory/YYYY-MM-DD.md`
- [ ] **G6.3** Skill documents `mc checkpoint` commands as available tools

### G7. Session Continuity Tests 🟡

- [ ] **G7.1** Rust: checkpoint compile produces valid markdown under 500 tokens
- [ ] **G7.2** Rust: checkpoint validate rejects missing required fields
- [ ] **G7.3** Go: `mc checkpoint` creates file + git commits
- [ ] **G7.4** Go: `mc checkpoint restart` logs session transition to `sessions.jsonl`
- [ ] **G7.5** Go: auto-checkpoint fires on gate approval
- [ ] **G7.6** React: health indicator reflects token count thresholds

---

## Recommended Execution Order

1. **A1–A7** — Rust Stage enum + tests (foundation, everything depends on this)
2. **B1–B3** — Go types + CLI + `.mission/` files (consume Rust changes)
3. **C1–C4** — React UI updates (consume Go API changes)
4. **G1** — Rust checkpoint schema extension (builds on A1 stage work)
5. **G2–G4** — Go checkpoint commands + API (builds on B1 Go types)
6. **G5** — React checkpoint UI (builds on C1 UI updates)
7. **E1–E2** — Documentation + migration tooling
8. **B4** — OpenClaw integration (can parallel with stage + checkpoint work)
9. **D1–D3, G6** — OpenClaw skill + Agent Teams + checkpoint skill integration (depends on B4)
10. **F1–F9** — Integration testing (final validation)