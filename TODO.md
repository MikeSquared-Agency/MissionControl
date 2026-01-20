# MissionControl — TODO

## Getting Started

1. Run `mc init` in your project directory to create `.mission/` structure
2. Start the orchestrator: `cd orchestrator && go run . --workdir /path/to/project`
3. Start the UI: `cd web && npm run dev`
4. Open http://localhost:3000 — wizard will guide you through setup

For detailed specs, UI mockups, and decisions log, see `docs/archive/V5.1-SPEC.md`.

---

## Current: v5.1 — Quality of Life

### 1. Documentation Cleanup
- [x] Consolidate specs into 5 root files (README, ARCHITECTURE, CONTRIBUTING, CHANGELOG, TODO)
- [x] Move historical specs to `docs/archive/`
- [x] Remove `web/README.md` Vite boilerplate (or replace with real content)

### 2. Repository Cleanup

**File Renames:**
- [x] `orchestrator/api/v5.go` → `orchestrator/api/king.go`
- [x] `orchestrator/api/v4.go` → removed (not needed)
- [x] Any other `v4`/`v5` prefixed code files → sensible names

**Other Cleanup:**
- [ ] Audit for dead code / unused files
- [x] Ensure `.gitignore` covers: `dist/`, `target/`, `node_modules/`, `.mission/`
- [x] Remove any accidentally committed build artifacts
- [ ] Add `CODEOWNERS` for GitHub (optional)

### 3. Testing Improvements

**Rust Tests:**
- [x] `core/workflow/` — state machine transitions (14 tests)
- [x] `core/knowledge/` — token counting accuracy (23 tests)
- [x] Handoff JSON validation
- [x] Gate criteria checking
- [x] `cargo test` passes all (56 tests)

**Go Integration Tests:**
- [x] King tmux session lifecycle (start → message → response → stop)
- [x] WebSocket connection + event flow
- [x] API endpoints with mocked Claude
- [x] Rust core subprocess calls (`mc-core tokens`, `mc-core validate`)

**React Tests (additions):**
- [x] Project wizard component (13 tests)
- [x] Multi-project switching (Sidebar tests)
- [x] Matrix toggle interactions (11 tests)

**E2E Tests (Playwright):**
- [x] Project wizard flow end-to-end (test written, needs backend)
- [x] King Mode: send message, receive response (test written, needs backend)
- [x] Agent spawning + count updates (test written, needs backend)
- [x] Zone management CRUD (test written, needs backend)
- [ ] Token usage displays correctly
- [x] WebSocket reconnection (test written, needs backend)

**Test Infrastructure:**
- [x] `make test` — runs Go + React + Rust
- [x] `make test-rust` — Rust only (`cargo test`)
- [x] `make test-go` — Go only (`go test ./...`)
- [x] `make test-web` — React only (`npm test`)
- [x] `make test-integration` — Go integration tests
- [x] `make test-e2e` — Playwright
- [x] GitHub Actions CI workflow for PRs

### 4. Startup Simplification

**Makefile Commands:**
- [x] `make dev` — starts vite + orchestrator together (single command)
- [x] `make dev-ui` — vite only
- [x] `make dev-api` — orchestrator only
- [x] `make build` — production build (Go + Rust + React)
- [x] `make install` — install binaries to `/usr/local/bin`
- [x] `make clean` — remove build artifacts

**Single Binary Distribution:**
- [x] `mc serve` command — starts orchestrator + serves built React UI
- [x] Embed built `web/dist/` in Go binary using `embed` package
- [x] Single binary contains everything

**Homebrew:**
- [ ] Create `homebrew-tap` repo (`DarlingtonDeveloper/homebrew-tap`)
- [x] Formula downloads release binary (see homebrew/mission-control.rb)
- [ ] `brew tap DarlingtonDeveloper/tap && brew install mission-control` works
- [ ] Document in README

### 5. Project Wizard
- [x] `ProjectWizard` component with step state machine
- [x] `WorkflowMatrix` component with toggle logic
- [x] Typing indicator component (300ms delay)
- [x] `POST /api/projects` — calls `mc init` subprocess
- [x] `GET /api/projects` — reads from `~/.mission-control/config.json`
- [x] `DELETE /api/projects/:id` — removes from list (not disk)
- [x] Sidebar project list with switch capability
- [x] `mc init` accepts `--path`, `--git`, `--king`, `--config` flags
- [x] Wizard passes matrix config as JSON file to `mc init`

### 6. Configuration & Storage
- [x] Create `~/.mission-control/` on first run
- [x] `mc` CLI reads/writes config
- [x] Orchestrator reads/writes config
- [x] Add project to list when created via wizard
- [x] Update `lastOpened` when project opened
- [ ] Sort sidebar by `lastOpened` descending

### 7. Bug Fix: Rust Core Not Integrated

Rust `core/` built but never called. Go does its own parsing.

- [x] Verify `mc-core` binary builds with `make build`
- [x] Implement CLI commands in Rust (if not already)
- [x] Create `orchestrator/core/client.go` wrapper
- [x] Replace inline Go parsing with `core.CountTokens()` etc.
- [x] Update `make install` to install both `mc` and `mc-core`
- [x] Update `make build` to compile Rust before Go

### 8. Bug Fix: Token Usage Display
- [x] After King response, pipe text through `mc-core tokens`
- [x] Emit `token_usage` WebSocket event
- [x] UI: update store from event
- [x] UI: display in header/status bar

### 9. Bug Fix: Agent Count
- [x] Verify `agent_spawned` event emits on spawn
- [x] Verify `agent_stopped` event emits on kill
- [x] UI: listen to events, update `agents` array in store
- [x] Playwright test: spawn agent → verify count increments

### 10. UI Polish

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

### 11. Developer Experience

**Makefile Additions:**
- [x] `make lint` — Go (`golangci-lint`) + Rust (`clippy`) + TypeScript (`eslint`)
- [x] `make fmt` — format all code (`go fmt`, `cargo fmt`, `prettier`)

**Editor Setup:**
- [ ] `.vscode/settings.json` — format on save, recommended settings
- [ ] `.vscode/extensions.json` — recommended extensions list

**Optional:**
- [ ] Pre-commit hooks via lefthook or husky

### 12. Personas Management
- [ ] Settings panel: enable/disable all 11 personas individually
- [ ] Per-project persona configuration (stored in `.mission/config.json`)
- [ ] Persona descriptions visible in Settings
- [ ] Persona prompt preview/edit capability
- [ ] Sync persona settings with workflow matrix

### 13. Dynamic Project Switching
- [ ] Orchestrator API: `POST /api/projects/select` to switch active project
- [ ] Orchestrator reloads `.mission/state/` watcher on project switch
- [ ] WebSocket broadcasts project change event
- [ ] UI reloads state when project switches (no page refresh needed)
- [ ] Remove need to restart orchestrator with `--workdir` flag

---

## Success Criteria

| # | Criteria | Status |
|---|----------|--------|
| 1 | `make dev` starts everything with one command | Done |
| 2 | `make test` passes Go + React + Rust tests | Done |
| 3 | `make test-e2e` passes Playwright tests | Needs backend |
| 4 | README gets new user running in <5 minutes | |
| 5 | ≤5 markdown files in root | Done |
| 6 | No version-prefixed filenames (v4, v5) | Done |
| 7 | Project wizard creates working `.mission/` | Done |
| 8 | Token usage displays correctly | Done |
| 9 | Agent count updates in real-time | Done |
| 10 | Rust core called for token counting + validation | Done |
| 11 | Multi-project switching works | |
| 12 | `brew install mission-control` works | |
| 13 | Global config at `~/.mission-control/config.json` | Done |

---

## Future: v6 — 3D Visualization

- [ ] React Three Fiber setup
- [ ] Agent avatars in 3D space
- [ ] Zone visualization
- [ ] Camera controls
- [ ] Animations (spawn, complete, handoff)

---

## Future: v7+ — Polish & Scale

- [ ] Persistence (PostgreSQL or SQLite)
- [ ] Multi-model routing (Haiku/Sonnet/Opus)
- [ ] Token budget enforcement
- [ ] Worker health monitoring in UI
- [ ] Dark/light themes
- [ ] Remote access (deploy bridge)
- [ ] Conductor Skill (MissionControl builds MissionControl)

---

## Quick Reference

| Version | Focus | Status |
|---------|-------|--------|
| v1 | Agent fundamentals | Done |
| v2 | Go orchestrator | Done |
| v3 | React UI | Done |
| v4 | Rust core | Done |
| v5 | King + mc CLI | Done |
| v5.1 | Quality of life | Current |
| v6 | 3D visualization | Future |
| v7+ | Polish & scale | Future |

See [CHANGELOG.md](CHANGELOG.md) for completed version details.
