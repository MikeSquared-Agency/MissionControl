# MissionControl — TODO

## ✅ Completed: v1 - Agent Fundamentals

### v0_minimal.py (~66 lines)
- [x] Basic agent loop
- [x] Bash tool only
- [x] Prove the concept works

### v1_basic.py (~213 lines)
- [x] Add read, write, edit tools
- [x] Proper error handling
- [x] System prompt

### v2_todo.py (~308 lines)
- [x] Add todo tools (add/update/list)
- [x] Agent can track its own tasks
- [x] Progress display per turn

### v3_subagent.py (~423 lines)
- [x] Add task tool for spawning child agents
- [x] Isolated context per subagent
- [x] Subagent tracking and status

---

## ✅ Completed: v2 - Orchestrator

### Go Orchestrator
- [x] Project setup (go.mod)
- [x] Agent process manager (spawn/kill)
- [x] Track PID, status, tokens per agent
- [x] Environment passthrough (ANTHROPIC_API_KEY)
- [x] REST API endpoints
  - [x] POST /api/agents (spawn)
  - [x] GET /api/agents (list)
  - [x] DELETE /api/agents/:id (kill)
  - [x] POST /api/agents/:id/message
- [x] WebSocket event bus

### Rust Stream Parser
- [x] Project setup (Cargo.toml)
- [x] Parse Python agent format
- [x] Parse Claude Code stream-json format
- [x] Normalize both to unified events
- [x] Unit tests

---

## ✅ Completed: v3 - 2D Dashboard

### Setup
- [x] Vite + React + TypeScript
- [x] Tailwind CSS
- [x] Zustand state management with persistence
- [x] WebSocket connection hook with reconnection

### Core Components
- [x] Header with stats, King toggle, connection status
- [x] Sidebar with collapsible zone groups
- [x] AgentCard with persona badge, attention pulse, context menu
- [x] AgentPanel with conversation view
- [x] AgentHeader with details

### Zone System
- [x] Zone CRUD (create, edit, duplicate, delete)
- [x] Zone-based agent organization
- [x] Split/merge zones
- [x] Move agents between zones
- [x] Backend Zone API endpoints

### Conversation View
- [x] Scrollable message list with auto-scroll
- [x] User/assistant/error message bubbles
- [x] Collapsible tool call blocks
- [x] Chat input with Enter to send
- [x] Findings section

### Persona System
- [x] Default personas (Code Reviewer, Full Developer, Test Writer, Documentation)
- [x] Persona selection in spawn dialog
- [x] Persona badges on agent cards
- [x] Custom persona creation in settings

### Attention System
- [x] AttentionBar for global notifications
- [x] Quick response buttons (Yes/No, Allow/Deny)
- [x] Pulsing indicators on agent cards
- [x] Backend respond endpoint

### King Mode
- [x] KingPanel with conversation view
- [x] KingHeader with team stats
- [x] TeamOverview with agent badges
- [x] Amber-themed input
- [x] King message API endpoint

### Polish
- [x] Toast notification system
- [x] Loading spinners
- [x] Skeleton loaders
- [x] Empty states for all views
- [x] Confirmation dialogs

### Dialogs
- [x] SpawnDialog with persona/zone selection
- [x] ZoneDialog (create/edit)
- [x] SettingsPanel (General, Personas, Shortcuts)
- [x] SplitZoneDialog
- [x] MergeZoneDialog
- [x] MoveAgentDialog
- [x] ContextShareDialog

### Keyboard Shortcuts
- [x] ⌘N - spawn agent
- [x] ⌘K - kill selected agent
- [x] ↑/↓ or j/k - navigate agents
- [x] ⌘⇧K - toggle King mode
- [x] ⌘⇧N - new zone
- [x] ⌘, - settings
- [x] ⌘/ - focus chat
- [x] Esc - close modal

### Testing
- [x] Go backend unit tests (29 tests)
- [x] React frontend unit tests (52 tests)
- [x] API endpoint testing
- [x] Agent spawn/kill end-to-end

### Bug Fixes
- [x] Fixed infinite loop in useStats selector
- [x] Fixed button nesting in ZoneGroup
- [x] Fixed stdin handling for Claude Code agents
- [x] Added --dangerously-skip-permissions for headless execution

---

## 🔄 Current: v4 - 3D Visualization

### Setup
- [ ] Install React Three Fiber + drei
- [ ] Create 3D scene with isometric camera
- [ ] Integrate with existing Zustand store

### Scene Elements
- [ ] Floor/ground plane with grid
- [ ] Agent avatars (simple 3D characters)
- [ ] Zone areas (colored floor regions)
- [ ] Connection lines (parent/child agents)

### Interactions
- [ ] Click agent to select
- [ ] Floating UI panels (agent stats)
- [ ] Smooth position animations
- [ ] Camera pan/zoom controls

### Polish
- [ ] Custom 3D assets (or nice primitives)
- [ ] Marketable naming (TBD)
- [ ] Landing page
- [ ] Documentation

---

## 📋 v5: Persistence + Skills

### Persistence Layer
- [ ] Evaluate options (Beads vs SQLite vs Supabase)
- [ ] Save session findings on exit
- [ ] Load previous context on start
- [ ] Audit trail of agent actions

### Conductor Skill
- [ ] Create `.claude/skills/conductor/SKILL.md`
- [ ] CLI interface for skill to call orchestrator
- [ ] "Spin up review team" → spawns 3 agents
- [ ] Document skill usage

### Context Sharing Improvements
- [ ] Structured handoff protocol
- [ ] Shared scratchpad files
- [ ] Inter-agent MCP (stretch)

---

## 📋 v6+: Future

### Orchestrator Wizard
- [ ] Meta-agent that manages other agents
- [ ] Wizard avatar in 3D UI
- [ ] Natural language agent control

### Remote Access
- [ ] --host 0.0.0.0 flag
- [ ] Optional cloudflared tunnel integration
- [ ] Auth for remote access

### Multi-Model Support
- [ ] OpenAI Codex CLI parsing
- [ ] Gemini CLI parsing
- [ ] Aider text parsing (no JSON)
- [ ] Grok (when available)
- [ ] Model selection in UI
- [ ] Cost tracking per model

### Distribution Polish
- [ ] GoReleaser setup
- [ ] Homebrew tap
- [ ] Embed uv in Go binary
- [ ] One-command install experience

---

## Research Notes

### Rust Ecosystem
- `claude-codes` crate handles Claude Code protocol parsing
- `tiktoken-rs` for token counting
- Don't reinvent the wheel — build on existing crates

### Context Sharing
- v3-v4: Orchestrator-mediated (inject into system prompts)
- v5+: Persistent storage (Beads/SQLite/Supabase)

### Multi-Agent Parsing
- Claude Code: `--output-format stream-json` ✅
- Codex CLI: Has structured JSON modes ✅
- Aider: Intentionally outputs text (JSON hurts code quality)
- Gemini CLI: JSON support requested (issue #8022)