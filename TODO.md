# TODO

## Completed: v1 - Agent Fundamentals

### v0_minimal.py (66 lines)
- [x] Basic agent loop
- [x] Bash tool only
- [x] Prove the concept works

### v1_basic.py (213 lines)
- [x] Add read, write, edit tools
- [x] Proper error handling
- [x] System prompt

### v2_todo.py (308 lines)
- [x] Add todo tools (add/update/list)
- [x] Agent can track its own tasks
- [x] Progress display per turn

### v3_subagent.py (423 lines)
- [x] Add task tool for spawning child agents
- [x] Isolated context per subagent
- [x] Subagent tracking and status

---

## Completed: v2 - Orchestrator

### Go Orchestrator
- [x] Project setup (go.mod)
- [x] Agent process manager (spawn/kill processes)
- [x] Track PID, status, tokens per agent
- [x] Environment passthrough (ANTHROPIC_API_KEY)
- [x] REST API endpoints
  - [x] POST /api/agents (spawn)
  - [x] GET /api/agents (list)
  - [x] DELETE /api/agents/:id (kill)
  - [x] POST /api/agents/:id/message
- [x] WebSocket event bus
  - [x] Broadcast events to connected UIs
  - [x] Receive commands from UI

### Rust Stream Parser
- [x] Project setup (Cargo.toml)
- [x] Parse Python agent format (JSON + plain text)
- [x] Parse Claude Code stream-json format
- [x] Normalize both to unified events
- [x] Unit tests

---

## Current Focus: v3 - 2D Dashboard

### Project Setup
- [ ] Create React app with Vite
- [ ] Install Tailwind CSS
- [ ] Install Zustand for state
- [ ] WebSocket connection hook

### Components
- [ ] Layout (header, sidebar, main)
- [ ] AgentList - list all agents with status
- [ ] AgentCard - expandable card per agent
- [ ] SpawnDialog - form to spawn new agent
- [ ] ChatPanel - send messages, see responses
- [ ] ToolLog - real-time tool calls stream
- [ ] StatsBar - total tokens, active count

### State Management
- [ ] Agents store (list, add, update, remove)
- [ ] WebSocket store (connection, events)
- [ ] UI store (selected agent, panels)

---

## Later

- [ ] v4: 3D Visualization (React Three Fiber)
- [ ] v5: Persistence layer
