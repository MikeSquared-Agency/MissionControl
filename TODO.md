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
- [ ] Remove `web/README.md` Vite boilerplate (or replace with real content)

### 2. Repository Cleanup

**File Renames:**
- [ ] `orchestrator/api/v5.go` → `orchestrator/api/king.go`
- [ ] `orchestrator/api/v4.go` → `orchestrator/api/mission.go`
- [ ] Any other `v4`/`v5` prefixed code files → sensible names

**Other Cleanup:**
- [ ] Audit for dead code / unused files
- [ ] Ensure `.gitignore` covers: `dist/`, `target/`, `node_modules/`, `.mission/`
- [ ] Remove any accidentally committed build artifacts
- [ ] Add `CODEOWNERS` for GitHub (optional)

### 3. Testing Improvements

**Rust Tests:**
- [ ] `core/workflow/` — state machine transitions
- [ ] `core/knowledge/` — token counting accuracy
- [ ] Handoff JSON validation
- [ ] Gate criteria checking
- [ ] `cargo test` passes all

**Go Integration Tests:**
- [ ] King tmux session lifecycle (start → message → response → stop)
- [ ] WebSocket connection + event flow
- [ ] API endpoints with mocked Claude
- [ ] Rust core subprocess calls (`mc-core tokens`, `mc-core validate`)

**React Tests (additions):**
- [ ] Project wizard component
- [ ] Multi-project switching
- [ ] Matrix toggle interactions

**E2E Tests (Playwright):**
- [ ] Project wizard flow end-to-end
- [ ] King Mode: send message, receive response
- [ ] Agent spawning + count updates
- [ ] Zone management CRUD
- [ ] Token usage displays correctly
- [ ] WebSocket reconnection

**Test Infrastructure:**
- [ ] `make test` — runs Go + React + Rust
- [ ] `make test-rust` — Rust only (`cargo test`)
- [ ] `make test-go` — Go only (`go test ./...`)
- [ ] `make test-web` — React only (`npm test`)
- [ ] `make test-integration` — Go integration tests
- [ ] `make test-e2e` — Playwright
- [ ] GitHub Actions CI workflow for PRs

### 4. Startup Simplification

**Makefile Commands:**
- [ ] `make dev` — starts vite + orchestrator together (single command)
- [ ] `make dev-ui` — vite only
- [ ] `make dev-api` — orchestrator only
- [ ] `make build` — production build (Go + Rust + React)
- [ ] `make install` — install binaries to `/usr/local/bin`
- [ ] `make clean` — remove build artifacts

**Single Binary Distribution:**
- [ ] `mc serve` command — starts orchestrator + serves built React UI
- [ ] Embed built `web/dist/` in Go binary using `embed` package
- [ ] Single binary contains everything

**Homebrew:**
- [ ] Create `homebrew-tap` repo (`DarlingtonDeveloper/homebrew-tap`)
- [ ] Formula downloads release binary
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
- [ ] Create `~/.mission-control/` on first run
- [ ] `mc` CLI reads/writes config
- [ ] Orchestrator reads/writes config
- [ ] Add project to list when created via wizard
- [ ] Update `lastOpened` when project opened
- [ ] Sort sidebar by `lastOpened` descending

### 7. Bug Fix: Rust Core Not Integrated

Rust `core/` built but never called. Go does its own parsing.

- [ ] Verify `mc-core` binary builds with `make build`
- [ ] Implement CLI commands in Rust (if not already)
- [ ] Create `orchestrator/core/client.go` wrapper
- [ ] Replace inline Go parsing with `core.CountTokens()` etc.
- [ ] Update `make install` to install both `mc` and `mc-core`
- [ ] Update `make build` to compile Rust before Go

### 8. Bug Fix: Token Usage Display
- [ ] After King response, pipe text through `mc-core tokens`
- [ ] Emit `token_usage` WebSocket event
- [ ] UI: update store from event
- [ ] UI: display in header/status bar

### 9. Bug Fix: Agent Count
- [ ] Verify `agent_spawned` event emits on spawn
- [ ] Verify `agent_stopped` event emits on kill
- [ ] UI: listen to events, update `agents` array in store
- [ ] Playwright test: spawn agent → verify count increments

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
- [ ] `make lint` — Go (`golangci-lint`) + Rust (`clippy`) + TypeScript (`eslint`)
- [ ] `make fmt` — format all code (`go fmt`, `cargo fmt`, `prettier`)

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
| 1 | `make dev` starts everything with one command | |
| 2 | `make test` passes Go + React + Rust tests | |
| 3 | `make test-e2e` passes Playwright tests | |
| 4 | README gets new user running in <5 minutes | |
| 5 | ≤5 markdown files in root | Done |
| 6 | No version-prefixed filenames (v4, v5) | |
| 7 | Project wizard creates working `.mission/` | Done |
| 8 | Token usage displays correctly | |
| 9 | Agent count updates in real-time | |
| 10 | Rust core called for token counting + validation | |
| 11 | Multi-project switching works | |
| 12 | `brew install mission-control` works | |
| 13 | Global config at `~/.mission-control/config.json` | |

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
