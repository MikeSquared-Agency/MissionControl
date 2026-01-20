# MissionControl — Project Spec

## Vision

A visual multi-agent orchestration system where a **King** agent coordinates **worker** agents through a **6-phase workflow**. Workers spawn, complete tasks, and die. Context lives in files, not conversation memory.

Inspired by [Vibecraft](https://vibecraft.dev), [Ralv](https://ralv.dev), and [Gastown](https://gastown.dev).

---

## Architecture

### Layers × Domains

```
                         LAYERS
     ┌─────────────────────┼─────────────────────┐
     ▼                     ▼                     ▼
┌─────────┐          ┌──────────┐          ┌──────────┐
│   UI    │   ────►  │   API    │   ────►  │   CORE   │
│ (React) │          │   (Go)   │          │(Rust/LLM)│
└─────────┘          └──────────┘          └──────────┘
                           │
          ┌────────────────┼────────────────┬────────────────┐
          ▼                ▼                ▼                ▼
     ┌─────────┐     ┌──────────┐    ┌───────────┐    ┌─────────┐
     │STRATEGY │     │ WORKFLOW │    │ KNOWLEDGE │    │ RUNTIME │
     └─────────┘     └──────────┘    └───────────┘    └─────────┘
     
                         DOMAINS
```

| Domain | Responsibility | Intelligence | Owner |
|--------|---------------|--------------|-------|
| **Strategy** | High-level decisions, user conversation | LLM (Opus) | King |
| **Workflow** | Phases, gates, task state | Deterministic | Rust Engine |
| **Knowledge** | Specs, findings, briefings, tokens | Rust + LLM | Knowledge Manager |
| **Runtime** | Process management, health | Deterministic | Go + Rust |

### Stack

| Component | Language | Why |
|-----------|----------|-----|
| **Agents** | Python | Anthropic SDK, educational |
| **API** | Go | Goroutines, single binary |
| **Core** | Rust | Deterministic logic, token counting |
| **Strategy** | Claude Opus | Judgment, synthesis |
| **Workers** | Claude Sonnet/Haiku | Implementation |
| **UI** | React + Three.js | Interactive 3D |

---

## Workflow

### Six Phases

```
IDEA → DESIGN → IMPLEMENT → VERIFY → DOCUMENT → RELEASE
  │       │          │          │         │          │
  ▼       ▼          ▼          ▼         ▼          ▼
 Gate    Gate       Gate       Gate      Gate       Gate
```

| Phase | Purpose | Workers | Gate Criteria |
|-------|---------|---------|---------------|
| **Idea** | Research feasibility | Researcher | Spec drafted |
| **Design** | UI mockups + system design | Designer, Architect | Mockups + API design approved |
| **Implement** | Build features | Developer (per zone) | Code complete, builds |
| **Verify** | Quality checks | Reviewer, Security, Tester, QA | All checks pass |
| **Document** | README + docs | Docs | Docs complete |
| **Release** | Deploy | DevOps | Deployed, verified |

Gates require explicit approval before proceeding.

### Zones

Zones are WHERE in the codebase:

```
System (root)
├── Frontend
├── Backend
├── Database
├── Infra
└── Shared
```

Workers are assigned to zones. A Developer in Frontend doesn't touch Backend.

---

## Agents

### King (Persistent)

The King is the only persistent agent. It:
- Talks to the user
- Approves phase gates
- Spawns workers with briefings
- Synthesizes findings
- Never implements directly

Model: Claude Opus

### Workers (Ephemeral)

Workers spawn, complete a task, and die. They receive:
- A briefing (~300 tokens)
- Zone assignment
- Tool/MCP restrictions

| Persona | Phase | Tools | Model |
|---------|-------|-------|-------|
| Researcher | Idea | web_search, read | Sonnet |
| Designer | Design | figma (read), write | Sonnet |
| Architect | Design | read, write | Sonnet |
| Developer | Implement | bash, read, write, edit | Sonnet |
| Debugger | Implement | bash, read, edit | Sonnet |
| Reviewer | Verify | read, github | Haiku |
| Security | Verify | read, bash (limited) | Sonnet |
| Tester | Verify | bash, read, write | Haiku |
| QA | Verify | bash, read | Haiku |
| Docs | Document | read, write | Haiku |
| DevOps | Release | bash, vercel, github | Haiku |

---

## Token Efficiency

### Problem

Context accumulates → costs explode → performance degrades.

### Solution: Files are truth, briefings are context

```
┌─────────────────────────────────────────────────────────────┐
│  SOURCE OF TRUTH (.mission/ files)                          │
│  Complete specs, full history, git-tracked                  │
│  Tokens: 2000+                                              │
└─────────────────────────────────────────────────────────────┘
                      │
                      ▼ Knowledge Manager compiles
┌─────────────────────────────────────────────────────────────┐
│  BRIEFING (what worker receives)                            │
│  - Task description                                         │
│  - Key requirements (3-5 bullets)                           │
│  - Relevant decisions                                       │
│  - File paths for deep-dive                                 │
│  Tokens: ~300                                               │
└─────────────────────────────────────────────────────────────┘
```

### Token Budgets

| Threshold | Action |
|-----------|--------|
| < 50% | Continue |
| 50-75% | Warning, consider handoff |
| > 75% | Prepare handoff |
| > 90% | Force handoff |

### Handoff Protocol

```typescript
interface WorkerHandoff {
  task_id: string;
  status: 'complete' | 'blocked' | 'partial';
  findings: Finding[];
  artifacts: string[];        // File paths
  open_questions: string[];
  context_for_successor?: {
    key_decisions: string[];
    gotchas: string[];
  };
}
```

Workers output structured JSON. Rust validates. Fresh worker spawns with lean briefing.

---

## File Structure

### Project State (.mission/)

```
.mission/
├── config.md              # Project settings
├── ideas/                 # IDEA-{name}.md
├── specs/                 # SPEC-{name}.md, api.md, models.md
├── mockups/               # UI iterations
├── progress/              # TODO-{name}.md
├── reviews/               # REVIEW-{name}.md
├── checkpoints/           # Periodic full state
├── handoffs/              # Worker handoff records
└── releases/              # RELEASE-{version}.md
```

### Codebase

```
mission-control/
├── SPEC.md
├── ARCHITECTURE.md
│
├── web/                     # UI Layer (v3 ✅)
│   └── src/
│       ├── components/      # AgentCard, KingPanel, ZoneGroup, etc.
│       ├── stores/          # Zustand (useStore)
│       ├── hooks/           # useWebSocket, useKeyboardShortcuts
│       └── types/           # TypeScript definitions
│
├── orchestrator/            # API Layer (v3 ✅)
│   ├── manager/             # Agent + Zone management
│   ├── api/                 # REST routes
│   └── ws/                  # WebSocket hub
│
├── core/                    # Core Layer (v4 - TODO)
│   ├── workflow/            # Rust state machine
│   ├── knowledge/           # Rust token/checkpoint mgmt
│   ├── runtime/             # Rust health monitor
│   └── ffi/                 # Go bindings
│
└── agents/                  # Python agents (v1 ✅)
    └── v0-v3
```

---

## Versions

### v1: Agent Fundamentals ✅

Build agents from scratch to understand the core loop.

| Agent | Lines | Tools | Concept |
|-------|-------|-------|---------|
| v0_minimal | ~50 | bash | Proves agents are tiny |
| v1_basic | ~200 | bash, read, write, edit | Complete agent |
| v2_todo | ~300 | + todo | Explicit planning |
| v3_subagent | ~450 | + task | Isolated child agents |

### v2: Orchestrator ✅

Go orchestrator + Rust stream parser.

- Agent process manager (spawn/kill)
- REST API endpoints
- WebSocket event bus
- Stream parsing (Python + Claude Code formats)

### v3: 2D Dashboard ✅

Full-featured React dashboard with 81 unit tests.

**Completed Features:**
- Zustand state with persistence + WebSocket reconnection
- Header, Sidebar, AgentCard, AgentPanel
- Zone System (CRUD, split/merge, move agents)
- Persona System (4 defaults + custom creation)
- King Mode UI (KingPanel, KingHeader, TeamOverview)
- Attention System (notifications, quick responses)
- Settings Panel + Keyboard shortcuts
- Zone API endpoints + King message endpoint

### v4: Architecture Foundation 🔄 CURRENT

Implement the Rust core and domain organization.

**Rust Core:**
- Workflow engine (phases, gates, tasks)
- Knowledge manager (tokens, checkpoints, handoffs)
- Health monitor
- FFI bindings for Go

**Go API:**
- Strategy routes (full King logic)
- Workflow routes (phases, gates)
- Knowledge routes (briefings, handoffs)

**React UI:**
- Domain-organized structure
- Phase/workflow view
- Token usage display

### v5: King + Workflow

Full King agent implementation with 6-phase workflow.

### v6: 3D + Polish

3D visualization with Three.js/React Three Fiber.

### v7+: Future

Persistence, Conductor Skill, Multi-Model, Remote Access.

---

## API

### REST (Current v3)

```
# Agents
POST   /api/agents              # Spawn agent
GET    /api/agents              # List agents
DELETE /api/agents/:id          # Kill agent
POST   /api/agents/:id/message  # Send message
POST   /api/agents/:id/respond  # Respond to attention

# Zones
POST   /api/zones               # Create zone
GET    /api/zones               # List zones
PUT    /api/zones/:id           # Update zone
DELETE /api/zones/:id           # Delete zone

# King (UI shell)
POST   /api/king/message        # Send to King
```

### REST (v4+ additions)

```
# Strategy
POST   /api/gates/:id/approve   # Approve gate

# Workflow
GET    /api/phases              # List phases
GET    /api/tasks               # List tasks
PUT    /api/tasks/:id/status    # Update task

# Knowledge
GET    /api/specs/:id           # Get spec
GET    /api/briefings/:worker   # Get briefing
POST   /api/handoffs            # Submit handoff
```

---

## Model Allocation

| Role | Model | Rationale |
|------|-------|-----------|
| King | Opus | Strategic judgment |
| Briefing generation | Sonnet | Distillation |
| Designer, Architect, Developer | Sonnet | Complex work |
| Reviewer, Tester, QA, Docs, DevOps | Haiku | Pattern matching |

---

## Success Criteria

**v4:** Rust core compiles, Go integrates via FFI, domain routes work.

**v5:** King conversation works end-to-end, workers spawn with briefings.

**v6:** 3D visualization renders agents in zones.

---

Ready to build?