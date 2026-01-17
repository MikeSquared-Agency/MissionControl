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

### King Mode (UI Shell)
- [x] KingPanel with conversation view
- [x] KingHeader with team stats
- [x] TeamOverview with agent badges
- [x] Amber-themed input
- [x] King message API endpoint (POST /api/king/message)
- [x] SendKingMessage in manager

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

## 🔄 Current: v4 - Architecture Foundation

Building the layers × domains architecture with Rust core.

### Rust Core: Workflow Engine
- [ ] Project setup (core/workflow/Cargo.toml)
- [ ] Phase enum (Idea, Design, Implement, Verify, Document, Release)
- [ ] Task struct with status (Pending, Ready, InProgress, Blocked, Done)
- [ ] Gate struct with criteria checking
- [ ] WorkflowEngine state machine
  - [ ] Phase transitions
  - [ ] Task dependency resolution
  - [ ] Gate status computation
- [ ] Unit tests

### Rust Core: Knowledge Manager
- [ ] Project setup (core/knowledge/Cargo.toml)
- [ ] Token counting (tiktoken-rs)
- [ ] Token budget tracking per worker
- [ ] Handoff schema validation (serde)
- [ ] Checkpoint serialization
- [ ] Delta computation (diff between checkpoints)
- [ ] Pruning rules (stale, duplicate, superseded)
- [ ] Briefing input compilation
- [ ] Unit tests

### Rust Core: Runtime Monitor
- [ ] Project setup (core/runtime/Cargo.toml)
- [ ] Health status enum
- [ ] Worker health tracking
- [ ] Stuck detection (timeout-based)
- [ ] Activity timestamps
- [ ] Unit tests

### Rust FFI
- [ ] Project setup (core/ffi/Cargo.toml)
- [ ] C bindings for Go
- [ ] Workflow engine exports
- [ ] Knowledge manager exports
- [ ] Runtime monitor exports
- [ ] Build script for shared library

### Go API: Domain Routes
- [ ] Strategy routes
  - [ ] POST /api/gates/:id/approve
- [ ] Workflow routes
  - [ ] GET /api/phases
  - [ ] GET /api/tasks
  - [ ] PUT /api/tasks/:id/status
- [ ] Knowledge routes
  - [ ] GET /api/specs/:id
  - [ ] GET /api/briefings/:worker
  - [ ] POST /api/handoffs
- [ ] Integrate Rust via FFI

### React UI: Domain Structure
- [ ] Reorganize src/ into domains/
  - [ ] domains/strategy/
  - [ ] domains/workflow/
  - [ ] domains/knowledge/
  - [ ] domains/runtime/
- [ ] Keep existing components working
- [ ] Add PhaseView component
- [ ] Add TokenUsage component

### WebSocket Events
- [ ] phase_changed event
- [ ] task_updated event
- [ ] gate_status event
- [ ] token_warning event
- [ ] checkpoint_created event
- [ ] agent_health event

---

## 📋 v5: King + Workflow

### King Agent (Full Implementation)
- [ ] King system prompt design (Opus)
- [ ] Strategic decision making
- [ ] Worker spawning with briefings
- [ ] Finding synthesis
- [ ] Gate approval logic
- [ ] Never implements directly

### Briefing System
- [ ] Briefing template per persona
- [ ] LLM-based briefing generation (Sonnet)
- [ ] Spec → briefing distillation (~300 tokens)
- [ ] Zone-scoped context filtering

### Worker Personas
- [ ] 11 persona definitions (Researcher → DevOps)
- [ ] Tool restrictions per persona
- [ ] MCP access levels per persona
- [ ] Model selection per persona (Sonnet/Haiku)

### Handoff System
- [ ] Structured handoff output from workers
- [ ] Rust validation of handoff JSON
- [ ] Delta storage in .mission/handoffs/
- [ ] Fresh worker spawning with briefing
- [ ] Context continuity across handoffs

### Phase Progression
- [ ] Idea phase (Researcher)
- [ ] Design phase (Designer, Architect)
- [ ] Implement phase (Developer per zone)
- [ ] Verify phase (Reviewer, Security, Tester, QA)
- [ ] Document phase (Docs)
- [ ] Release phase (DevOps)
- [ ] Gate approval UI flow

### .mission/ Directory
- [ ] File watcher for state changes
- [ ] Spec file management
- [ ] Progress file management
- [ ] Checkpoint creation
- [ ] Delta tracking

---

## 📋 v6: 3D + Polish

### 3D Visualization
- [ ] Install React Three Fiber + drei
- [ ] Isometric camera setup
- [ ] Floor/ground plane with grid
- [ ] Zone areas (colored regions)

### Agent Avatars
- [ ] Simple 3D characters per persona
- [ ] Status-based coloring
- [ ] Working animations
- [ ] Spawn/despawn effects

### Interactions
- [ ] Click agent to select
- [ ] Floating UI panels
- [ ] Smooth position animations
- [ ] Camera pan/zoom controls

### King Visualization
- [ ] King avatar (distinct from workers)
- [ ] Connection lines to workers
- [ ] Conversation indicator

### Polish
- [ ] Dark/light themes
- [ ] Landing page
- [ ] Documentation site
- [ ] Demo video

---

## 📋 v7+: Future

### Persistence
- [ ] Evaluate Beads vs SQLite vs Supabase
- [ ] Cross-session state restoration
- [ ] Audit trail of agent actions
- [ ] Session replay

### Conductor Skill
- [ ] Claude Code skill (SKILL.md)
- [ ] CLI interface for orchestrator
- [ ] "Spin up review team" → spawns agents
- [ ] Skill documentation

### Multi-Model Support
- [ ] OpenAI Codex CLI parsing
- [ ] Gemini CLI parsing
- [ ] Model selection per persona
- [ ] Cost tracking per model

### Remote Access
- [ ] --host 0.0.0.0 flag
- [ ] cloudflared tunnel integration
- [ ] Auth for remote access
- [ ] Mobile-friendly UI

### Wizard Agent
- [ ] Meta-agent design
- [ ] Progress monitoring
- [ ] Work reassignment
- [ ] Failure handling

---

## Implementation Order (v4)

Recommended sequence:

```
1. Rust workflow engine (no dependencies)
   └── Phase, Task, Gate structs
   └── State machine logic
   └── Unit tests

2. Rust knowledge manager (no dependencies)
   └── Token counting
   └── Handoff validation
   └── Checkpoint/delta

3. Rust runtime monitor (no dependencies)
   └── Health tracking
   └── Stuck detection

4. Rust FFI (depends on 1-3)
   └── C bindings
   └── Shared library build

5. Go API routes (depends on 4)
   └── Import Rust library
   └── Domain routes

6. React domain structure (parallel with 5)
   └── Reorganize folders
   └── New components

7. WebSocket events (depends on 5-6)
   └── New event types
   └── UI handlers

8. Integration testing
   └── End-to-end flow
```

---

## Research Notes

### Token Efficiency
- Context rot is real (middle content forgotten)
- Simple masking often beats LLM summarization
- 40-60% token savings possible via pruning
- Handoffs should be structured JSON, not prose

### Rust Ecosystem
- `tiktoken-rs` for token counting
- `serde` + `serde_json` for serialization
- `thiserror` for error types

### Model Allocation
- Opus: King (strategic judgment)
- Sonnet: Designer, Architect, Developer, Security
- Haiku: Reviewer, Tester, QA, Docs, DevOps

### Gastown Lessons
- External state (files) over conversation memory
- Handoffs are cheap: spawn fresh vs accumulate context
- Workers should be disposable