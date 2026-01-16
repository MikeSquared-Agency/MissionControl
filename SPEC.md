# Agent Orchestra — Project Spec

## Vision

A visual multi-agent orchestration system where you can spawn, monitor, and coordinate AI agents working on your codebase. Inspired by Vibecraft and Ralv.

---

## Stack

| Component | Language | Why |
|-----------|----------|-----|
| **Agents** | Python | Anthropic SDK is first-class, matches tutorials |
| **Orchestrator** | Go | Goroutines for concurrency, single binary, gastown's choice |
| **Stream Parser** | Rust | Learn Rust, real-time parsing, token counting |
| **Web UI** | React + Three.js | Interactive 3D visualization |

---

## Distribution

**User installs orchestrator via:**
- Homebrew: `brew install mike/tap/agent-orchestra`
- Go: `go install github.com/mike/agent-orchestra@latest`  
- Direct download from GitHub Releases

**All three point to the same binaries** built by GoReleaser on git tag.

**Python environment:**
- Go binary embeds `uv` (Rust-based Python manager)
- On first run: extracts to `~/.agent-orchestra/`, installs deps automatically
- User never touches pip or venv

**Web UI:**
- Deployed to Vercel
- Connects to `localhost:8080`
- Shows setup instructions if orchestrator not running

---

## Versions

### v1: Agent Fundamentals

**Goal:** Understand how agents work by building from scratch.

**The Core Loop:**
```python
while True:
    response = model(messages, tools)
    if response.stop_reason != "tool_use":
        return response.text
    results = execute(response.tool_calls)
    messages.append(results)
```

**Build progression:**

| Agent | Lines | Tools | Concept |
|-------|-------|-------|---------|
| v0_minimal | ~50 | bash | Proves agents are tiny |
| v1_basic | ~200 | bash, read, write, edit | Complete agent |
| v2_todo | ~300 | + todo | Explicit planning |
| v3_subagent | ~450 | + task | Isolated child agents |

**Deliverable:** Working CLI agents you can run locally.

---

### v2: Orchestrator

**Goal:** Manage multiple agent processes.

**Components:**

```
┌─────────────────────────────────────────┐
│           Go Orchestrator               │
│                                         │
│  ┌─────────────────────────────────────┐│
│  │     Agent Process Manager           ││
│  │  - Spawn/kill Python processes      ││
│  │  - Track PID, status, tokens        ││
│  └─────────────────────────────────────┘│
│                                         │
│  ┌─────────────────────────────────────┐│
│  │        Rust Stream Parser           ││
│  │  - Parse agent stdout               ││
│  │  - Count tokens (tiktoken-rs)       ││
│  │  - Emit structured JSON events      ││
│  └─────────────────────────────────────┘│
│                                         │
│  ┌─────────────────────────────────────┐│
│  │     WebSocket Event Bus             ││
│  │  - Broadcast to connected UIs       ││
│  │  - Receive commands from UI         ││
│  └─────────────────────────────────────┘│
│                                         │
│  ┌─────────────────────────────────────┐│
│  │         REST API                    ││
│  │  - POST /agents (spawn)             ││
│  │  - DELETE /agents/:id (kill)        ││
│  │  - POST /agents/:id/message         ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

**Data flow:**
```
Python Agent stdout
       ↓
  agent-stream (Rust binary)
       ↓
  Structured JSON events
       ↓
  Go Orchestrator
       ↓
  WebSocket → React UI
```

**Deliverable:** `agent-orchestra` binary that can spawn and manage multiple agents via REST API.

---

### v3: 2D Dashboard

**Goal:** Web UI to visualize and control agents.

**Tech:**
- React 18
- Tailwind CSS
- Zustand (state)
- Native WebSocket

**Components:**

| Component | Function |
|-----------|----------|
| AgentList | Table of all agents with status |
| AgentCard | Expandable card per agent |
| ChatPanel | Send messages, see responses |
| ToolLog | Real-time tool calls stream |
| StatsBar | Total tokens, active count |
| ZoneManager | Group agents by project/directory |

**Wireframe:**
```
┌────────────────────────────────────────────────────────┐
│  Agent Orchestra                    [+New Agent] [⚙️]  │
├────────────────────────────────────────────────────────┤
│  Stats: 3 agents | 12.4k tokens | 2 working            │
├──────────────────────────┬─────────────────────────────┤
│                          │                             │
│  ┌────────────────────┐  │  Agent: code-reviewer       │
│  │ 🟢 code-reviewer   │  │  Status: working            │
│  │    working         │  │  Task: Review auth.py       │
│  │    2.1k tokens     │  │  Tokens: 2,147              │
│  └────────────────────┘  │                             │
│                          │  ┌───────────────────────┐  │
│  ┌────────────────────┐  │  │ $ cat auth.py         │  │
│  │ 🟡 test-writer     │  │  │ 📖 Reading auth.py    │  │
│  │    idle            │  │  │ ✏️ Editing line 42    │  │
│  │    1.8k tokens     │  │  └───────────────────────┘  │
│  └────────────────────┘  │                             │
│                          │  ┌───────────────────────┐  │
│  ┌────────────────────┐  │  │ > Type message...     │  │
│  │ 🔴 refactorer      │  │  └───────────────────────┘  │
│  │    error           │  │                             │
│  │    956 tokens      │  │                             │
│  └────────────────────┘  │                             │
│                          │                             │
└──────────────────────────┴─────────────────────────────┘
```

**Deliverable:** Working web dashboard on Vercel that connects to local orchestrator.

---

### v4: 3D Visualization + Polish

**Goal:** The "wow factor" — Vibecraft/Ralv style. User-facing polish.

**Tech:**
- React Three Fiber (React bindings for Three.js)
- drei (Three.js helpers)
- Same Zustand store as v3

**Scene:**
```
       ┌─────────────────────────────────────┐
      ╱                                     ╱│
     ╱     🤖        🤖        🤖         ╱ │
    ╱    Claude    Gemini    Claude      ╱  │
   ╱                                    ╱   │
  ┌─────────────────────────────────────┐   │
  │  ═══════════════════════════════════│   │
  │  ║  Zone: Frontend  ║  Zone: API   ║│   │
  │  ║                  ║              ║│   │
  │  ║    📦    📦     ║     📦       ║│  ╱
  │  ║   task   task   ║    task      ║│ ╱
  │  ═══════════════════════════════════│╱
  └─────────────────────────────────────┘
```

**3D Features:**
- Isometric camera (like Ralv)
- Agent avatars as 3D characters
- Zones as floor areas
- Connection lines for parent/child agents
- Floating UI panels per agent
- Click agent → open chat

**Polish (for release):**
- Marketable naming (Orchestra? Conductor? TBD)
- Custom 3D assets
- Landing page
- Documentation

**Deliverable:** 3D interface + release-ready polish.

---

### v5: Persistence + Skills

**Goal:** Survive restarts, enable long-running work, extend Claude Code.

**Persistence Layer:**
- Resume work after closing laptop
- Track what agents did while you were away
- Multi-agent task dependencies
- Audit trail

**Options to evaluate:**
- Beads (Steve Yegge's git-backed solution)
- SQLite (simple, embedded)
- Supabase (ties into Personal OS)

**Conductor Skill:**
- A Claude Code skill that uses our orchestrator CLI
- "Spin up a review team for this PR" → spawns agents via `agent-orchestra spawn`
- Claude Code gains multi-agent powers through our infrastructure

---

### v6+: Future Ideas

**Orchestrator Wizard**
- A meta-agent (visualized as a wizard character) that helps manage other agents
- Monitors progress, reassigns work, handles failures
- Inspired by gastown's "major" concept

**Remote Access**
- Bind to 0.0.0.0 for local network access
- Optional cloudflared tunnel for phone access from anywhere
- Control agents from phone while away from desk

**Multi-Model Support**
- OpenAI Codex CLI, Gemini CLI, Grok alongside Claude
- Compare outputs, use different models for different tasks
- Cost optimization (cheap model for simple tasks)
- Note: Aider intentionally avoids JSON output (hurts code quality) - will need text parsing

---

## File Structure

```
agent-orchestra/
├── SPEC.md
│
├── agents/                     # Python
│   ├── v0_minimal.py
│   ├── v1_basic.py
│   ├── v2_todo.py
│   └── v3_subagent.py
│
├── stream-parser/              # Rust
│   ├── Cargo.toml
│   └── src/
│       └── main.rs
│
├── orchestrator/               # Go
│   ├── go.mod
│   ├── main.go
│   ├── manager/
│   │   └── manager.go
│   ├── api/
│   │   └── routes.go
│   └── ws/
│       └── hub.go
│
├── web/                        # React
│   ├── package.json
│   ├── src/
│   │   ├── App.tsx
│   │   ├── components/
│   │   ├── stores/
│   │   └── hooks/
│   └── public/
│
└── .goreleaser.yml             # Build config
```

---

## API

### REST

```
POST   /api/agents              # Spawn agent
GET    /api/agents              # List agents
GET    /api/agents/:id          # Get agent
DELETE /api/agents/:id          # Kill agent
POST   /api/agents/:id/message  # Send message

POST   /api/zones               # Create zone
GET    /api/zones               # List zones
PUT    /api/zones/:id/agents    # Assign agents to zone
```

### WebSocket Events

**Server → Client:**
```typescript
{ type: "agent_spawned", agent: Agent }
{ type: "agent_status", agentId: string, status: "idle" | "working" | "error" }
{ type: "tool_call", agentId: string, tool: string, args: object }
{ type: "tool_result", agentId: string, result: string, tokens: number }
{ type: "message", agentId: string, role: "assistant" | "user", content: string }
{ type: "agent_killed", agentId: string }
```

**Client → Server:**
```typescript
{ type: "spawn_agent", name: string, task?: string, zone?: string }
{ type: "send_message", agentId: string, content: string }
{ type: "kill_agent", agentId: string }
```

---

## Rust Component: agent-stream

**Purpose:** Normalize agent output from multiple sources into unified events.

The stream parser is the **normalization layer** that lets the UI treat all agents uniformly.

### Leveraging Existing Crates

For Claude Code parsing, we use the `claude-codes` crate:

```toml
[dependencies]
claude-codes = "0.3"  # Battle-tested Claude Code protocol parsing
serde = { version = "1", features = ["derive"] }
serde_json = "1"
```

Our Rust code handles:
- Python agent format (our custom)
- Normalization layer (unified events)
- Token counting
- Future: Codex CLI, Aider, Gemini

### Input Formats

**Python Agent (our format):**
```json
{"type":"turn","number":1}
{"type":"thinking","content":"I'll read the file."}
{"type":"tool_call","tool":"bash","args":{"command":"cat auth.py"}}
```

**Claude Code (`--output-format stream-json`):**
```json
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll read..."}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"auth.py"}}]}}
```

### Output (Unified Events)

Both normalized to:
```json
{"type":"thinking","content":"I'll read the file.","tokens":8,"agentId":"abc123"}
{"type":"tool_call","tool":"read","args":{"path":"auth.py"},"agentId":"abc123"}
```

**What you'll learn:**
- String vs &str (ownership)
- Option and Result (error handling)
- serde for JSON (multiple schemas)
- Using external crates
- Pattern matching

---

## Context Sharing

### Session Context (v3-v4)

Agents share context through the **orchestrator**:

```go
type Session struct {
    ID       string
    Agents   []Agent
    Findings []Finding  // What agents discovered
}

// Inject context when spawning new agents
agentB.SystemPrompt += formatFindings(session.Findings)
```

### Persistent Memory (v5+)

Across sessions, context persists via Beads/SQLite/Supabase:

```
Session ends → Save findings/TODOs → Next session loads history
```

---

## Decisions Made

1. **Agent models** - Claude only for v1-4. Multi-model support in v6+.
2. **Zone semantics** - Arbitrary containers. Can optionally bind to one or more git branches. UI will offer "create branch" when starting a zone.
3. **3D models** - We'll create custom assets when we get to v4.
4. **Remote access** - Post-v3 feature. Will use 0.0.0.0 binding + optional cloudflared tunnel.

---

## Non-Goals (For Now)

- Multi-user collaboration
- Cloud-hosted agents
- Mobile app
- Voice interface
- Persistent storage (v5)

---

## Success Criteria

**v1:** Can run `python v3_subagent.py "build a todo app"` and watch it work.

**v2:** Can `curl localhost:8080/api/agents` and see running agents.

**v3:** Can open browser, see agents, send messages, watch tool calls.

**v4:** Can see agents as 3D characters moving around zones.

---

Ready to build?