# Orchestrator Rebuild Spec

**Version:** v6.1
**Date:** 2026-02-08
**Scope:** Rebuild `mc serve` as the sole orchestrator entry point, replacing the legacy `orchestrator/main.go`. Wire up WebSocket hub, file watcher, process tracking, token tracking, and REST API to power the darlington.dev dashboard over Cloudflare Tunnel. Close remaining v6 TODO items. Lay forward-compatible foundations for v7 (requirements & traceability) and v8 (infrastructure & scale).

---

## Context

MissionControl v6 shipped the 10-stage workflow, JSONL storage, task dependencies, OpenClaw bridge, and checkpoints. However:

1. The **orchestrator binary hasn't been rebuilt** since the OpenClaw bridge code was merged — the code is there but the binary is stale.
2. The **old `orchestrator/main.go`** is a legacy v4/v5 entry point that imports dead packages (`v4/`, `terminal/`) and knows nothing about OpenClaw, 10-stage workflow, or checkpoints. It must be decommissioned.
3. **Token tracking** exists in Rust (`mc-core count-tokens`, `TokenCounter`, `TokenBudget`) and Go (`orchestrator/core/client.go` wrapper) but nothing calls it in the hot path.
4. The **UI has moved to darlington.dev** (Vercel) and connects to the orchestrator via Cloudflare Tunnel, meaning `mc serve` needs CORS, auth, and API-only mode support.
5. The **WebSocket hub and file watcher** from the old orchestrator need to be carried forward into `mc serve` — they provide the real-time reactivity loop the dashboard depends on.
6. **v7 (Requirements & Traceability)** is next on the roadmap. This rebuild should design data structures and endpoints that extend cleanly when requirements, specs, and full lineage tracking land.
7. **v8 (Infrastructure & Scale)** items like multi-model routing, headless mode, and Docker deployment can be partially addressed now with minimal overhead.

### Architecture After Rebuild

```
darlington.dev (Vercel)
        │
        │ HTTPS via Cloudflare Tunnel
        ▼
┌─────────────────────────────────────────┐
│  mc serve (single binary)               │
│                                         │
│  ┌─────────┐  ┌──────────┐  ┌────────┐ │
│  │ REST API │  │ WS Hub   │  │ OpenClaw│ │
│  │         │  │ (events) │  │ Bridge │ │
│  └────┬────┘  └────┬─────┘  └───┬────┘ │
│       │            │             │       │
│  ┌────┴────┐  ┌────┴─────┐      │       │
│  │ Process │  │ File     │      │       │
│  │ Tracker │  │ Watcher  │      │       │
│  └────┬────┘  └────┬─────┘      │       │
│       │            │             │       │
│  ┌────┴────┐  ┌────┴─────┐      │       │
│  │ Token   │  │ mc-core  │      │       │
│  │ Tracker │  │ (Rust)   │      │       │
│  └─────────┘  └──────────┘      │       │
└─────────────────────────────────┼───────┘
                                  │
                          OpenClaw Gateway
                                  │
                              Kai (King)
```

---

## Phase 1: Binary Rebuild & Decommission

### 1.1 Rebuild mc binary

Run `make build` to compile Go + Rust together. Verify the `mc` binary includes the OpenClaw bridge, serve command, and all v6 CLI commands.

```bash
make build
mc serve --help  # Should show --openclaw-gateway flag
```

### 1.2 Verify OpenClaw bridge connects

```bash
mc serve --openclaw-gateway ws://127.0.0.1:18789
# Check: /api/openclaw/status returns {"state":"connected",...}
# Check: /api/openclaw/chat accepts POST with {"message":"hello"}
# Check: /api/openclaw/send accepts POST with {"method":"...","params":{}}
```

### 1.3 Decommission legacy orchestrator

- Delete `orchestrator/main.go` (the standalone entry point)
- Delete `orchestrator/v4/` package (in-memory v4 store, fully replaced by `.mission/` file state)
- Delete `orchestrator/terminal/` package (PTY handler — not needed for darlington.dev)
- **Keep** `orchestrator/manager/` — needs refactoring but has useful process tracking logic
- **Keep** `orchestrator/ws/` — needs rewrite but the hub pattern is sound
- **Keep** `orchestrator/watcher/` — file watcher is essential, needs integration with `mc serve`
- **Keep** `orchestrator/api/` — `ProjectsHandler` and `OllamaHandler` still needed (Ollama for now, decommission later)
- **Keep** `orchestrator/openclaw/` — already wired into `mc serve`
- **Keep** `orchestrator/core/` — Rust subprocess wrapper
- **Keep** `orchestrator/bridge/` — file protocol handler

### 1.4 Audit imports

After deletion, verify all remaining packages compile. Update `cmd/mc/serve.go` imports if any reference deleted packages.

---

## Phase 2: WebSocket Hub Rewrite

The old `orchestrator/ws/hub.go` is tightly coupled to the legacy manager. Rewrite it as a standalone package that `mc serve` wires up.

### 2.1 Hub architecture

```go
// orchestrator/ws/hub.go (rewritten)
package ws

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan Event
    register   chan *Client
    unregister chan *Client
    subscriptions map[*Client]map[string]bool  // NEW: per-client topic subscriptions
}

type Event struct {
    Topic   string          `json:"topic"`   // Namespace: "stage", "task", "worker", "gate", "token", "chat", "zone", "checkpoint", "audit"
    Type    string          `json:"type"`    // Event type within topic
    Data    json.RawMessage `json:"data"`
}
```

### 2.2 Namespaced event topics

Inspired by the OpenClaw × Discord channel model — the UI subscribes to topics it cares about rather than receiving a firehose. Each dashboard view subscribes to its relevant topics.

| Topic | Events | Consumer View |
|-------|--------|---------------|
| `stage` | `changed` | Mission View |
| `task` | `created`, `updated`, `status_changed`, `dependency_changed` | Mission View, Traceability View |
| `worker` | `spawned`, `status_changed`, `completed`, `killed`, `heartbeat`, `output` | Mission View, Zones View |
| `gate` | `criteria_updated`, `approved`, `rejected` | Mission View |
| `token` | `usage_update`, `budget_warning`, `session_total` | All views (header) |
| `chat` | `kai_message`, `kai_typing`, `user_message` | Mission View |
| `zone` | `worker_entered`, `worker_left`, `activity` | Zones View |
| `checkpoint` | `created`, `restart_initiated` | Activity View |
| `audit` | `mutation` | Activity View |
| `requirement` | `created`, `updated` (v7-ready, no events until populated) | Traceability View |
| `spec` | `created`, `updated` (v7-ready, no events until populated) | Traceability View |
| `memory` | `updated` (per-persona memory file changed) | Zones View |

### 2.3 Client subscription protocol

```json
// Client → Server: subscribe to topics
{"type": "subscribe", "topics": ["stage", "task", "worker", "gate", "token", "chat"]}

// Client → Server: unsubscribe
{"type": "unsubscribe", "topics": ["audit"]}

// Server → Client: event
{"topic": "worker", "type": "spawned", "data": {"worker_id": "mc-a1b2c", "persona": "developer", "zone": "frontend"}}
```

Default: new clients receive all topics. Subscription filtering is optional optimisation.

### 2.4 Initial state sync

On WebSocket connect, the hub sends a full state snapshot so the UI can hydrate immediately:

```json
{
    "topic": "sync",
    "type": "initial_state",
    "data": {
        "stage": { "current": "implement", "updated_at": "..." },
        "tasks": [...],
        "workers": [...],
        "gates": {...},
        "zones": [...],
        "openclaw": { "state": "connected", "gatewayUrl": "..." },
        "token_usage": { "total": 4500, "estimated_cost_usd": 0.04 },
        "project": { "name": "...", "path": "..." }
    }
}
```

### 2.5 Worker output streaming

NEW: Stream worker stdout to the dashboard in real-time, not just lifecycle events. This gives visibility into what each agent is actually doing (inspired by the `#build-logs`, `#find-logs` pattern from the OpenClaw diagram).

```json
{
    "topic": "worker",
    "type": "output",
    "data": {
        "worker_id": "mc-a1b2c",
        "persona": "developer",
        "zone": "frontend",
        "line": "Creating login component...",
        "timestamp": "2026-02-08T14:30:00Z"
    }
}
```

The process tracker (Phase 3) captures stdout from worker processes and relays lines to the hub.

---

## Phase 3: Process Tracker

Refactor `orchestrator/manager/` into a leaner process tracker that works with `mc serve`. The old manager handled spawning, stdin/stdout piping, zone management, and the King process. The new tracker only needs to:

1. **Track active processes** spawned by `mc spawn` (which is now a CLI command, not an API call)
2. **Capture stdout/stderr** and relay to the WebSocket hub
3. **Monitor health** via heartbeat/process polling
4. **Report status** for the dashboard

### 3.1 Process tracking model

```go
// orchestrator/tracker/tracker.go
package tracker

type TrackedProcess struct {
    WorkerID   string    `json:"worker_id"`
    Persona    string    `json:"persona"`
    TaskID     string    `json:"task_id"`
    Zone       string    `json:"zone"`
    PID        int       `json:"pid"`
    Status     string    `json:"status"`      // running, complete, error, killed
    StartedAt  time.Time `json:"started_at"`
    TokenCount int       `json:"token_count"` // Accumulated tokens
    CostUSD    float64   `json:"cost_usd"`    // Estimated cost
}

type Tracker struct {
    processes map[string]*TrackedProcess
    hub       *ws.Hub
    mu        sync.RWMutex
}
```

### 3.2 Process discovery

The tracker watches `.mission/state/workers.json` for new entries. When a worker appears with a PID, the tracker:

1. Attaches to the process (verify PID is alive)
2. Starts reading `/proc/<pid>/fd/1` for stdout (or uses the file watcher approach if workers write to log files)
3. Emits `worker.spawned` event to the hub
4. Polls process status on a 2-second interval for health/heartbeat

### 3.3 Worker health & heartbeat

```json
{
    "topic": "worker",
    "type": "heartbeat",
    "data": {
        "worker_id": "mc-a1b2c",
        "status": "running",
        "pid": 12345,
        "uptime_seconds": 45,
        "alive": true
    }
}
```

Heartbeat interval: 5 seconds. If a process disappears without writing a handoff, emit a `worker.error` event.

### 3.4 Kill support

```go
func (t *Tracker) Kill(workerID string) error
```

Sends SIGTERM, waits 5s, then SIGKILL. Updates workers.json status to "killed". Emits `worker.killed` event.

---

## Phase 4: File Watcher Integration

Port `orchestrator/watcher/` into `mc serve`. The watcher polls `.mission/state/` and emits namespaced events to the hub.

### 4.1 Watched files → events

| File | Poll interval | Events emitted |
|------|---------------|----------------|
| `state/stage.json` | 500ms | `stage.changed` |
| `state/tasks.jsonl` | 500ms | `task.created`, `task.updated`, `task.status_changed`, `task.dependency_changed` |
| `state/workers.json` | 500ms | `worker.spawned`, `worker.status_changed`, `worker.completed` |
| `state/gates.json` | 500ms | `gate.criteria_updated`, `gate.approved` |
| `audit/interactions.jsonl` | 1s | `audit.mutation` |
| `orchestrator/checkpoints/` | 2s | `checkpoint.created` |
| `findings/` | 1s | Triggers `task.updated` with findings count |
| `requirements/` | 2s | `requirement.created`, `requirement.updated` (v7-ready, empty until populated) |
| `specs/` | 2s | `spec.created`, `spec.updated` (v7-ready, empty until populated) |
| `memory/` | 2s | `memory.updated` (per-persona memory files, Phase 9.2) |

### 4.2 Upgrade: fsnotify

The current watcher uses polling at 500ms intervals. Consider upgrading to `fsnotify` for native file system events. The Rust `mc-protocol` crate already uses the `notify` crate for this — the Go side should match. Fall back to polling if fsnotify isn't available.

### 4.3 Diffing logic

Carry forward the existing diff logic from `orchestrator/watcher/watcher.go` — it already handles:
- Stage comparison (current vs previous)
- Task diffing (new tasks, status changes, updated timestamps)
- Worker diffing (new workers, status changes)
- Gate diffing (approval changes)

Extend with:
- **Dependency change detection** — diff `blocks`/`blockedBy` arrays on tasks
- **Zone activity tracking** — aggregate workers per zone and emit `zone.activity` events
- **Findings accumulation** — when new files appear in `findings/`, emit event

---

## Phase 5: Token Tracking

Wire up the existing Rust token counter to the live event stream.

### 5.1 In-memory token accumulator

```go
// orchestrator/tokens/accumulator.go
package tokens

type SessionTokens struct {
    WorkerID       string  `json:"worker_id"`
    Persona        string  `json:"persona"`
    InputTokens    int     `json:"input_tokens"`
    OutputTokens   int     `json:"output_tokens"`
    TotalTokens    int     `json:"total_tokens"`
    EstimatedCost  float64 `json:"estimated_cost_usd"`
}

type Accumulator struct {
    sessions map[string]*SessionTokens  // worker_id → tokens
    total    SessionTokens              // aggregate
    budget   TokenBudget                // from Rust
    mu       sync.RWMutex
}

func (a *Accumulator) Record(workerID, persona, text string) {
    count, _ := core.CountTokens(text)
    // Update per-session and total
    // Check budget via mc-core
    // Emit events if approaching budget
}

func (a *Accumulator) Summary() TokenSummary {
    // Returns total, per-session breakdown, budget remaining, burn rate
}
```

### 5.2 Integration points

Token counting is triggered by:

1. **Worker stdout** — each line of output captured by the process tracker gets counted
2. **Handoff submission** — when `mc handoff` stores findings, count the findings JSON
3. **Kai chat messages** — count inbound and outbound messages through the OpenClaw bridge
4. **Checkpoint compilation** — count the compiled briefing (~500 tokens expected)

### 5.3 WebSocket events

```json
// Per-worker update (emitted on worker output)
{
    "topic": "token",
    "type": "usage_update",
    "data": {
        "worker_id": "mc-a1b2c",
        "persona": "developer",
        "session_tokens": 1250,
        "session_cost_usd": 0.011,
        "total_tokens": 4500,
        "total_cost_usd": 0.04
    }
}

// Budget warning (emitted when approaching limit)
{
    "topic": "token",
    "type": "budget_warning",
    "data": {
        "worker_id": "mc-a1b2c",
        "budget": 8000,
        "used": 6500,
        "remaining": 1500,
        "action": "consider_compaction"
    }
}
```

### 5.4 Model-tier cost estimation

For accurate cost tracking, the accumulator needs to know which model tier each worker is using. Extend the worker state with a `model` field:

| Persona | Default Model | Cost per MTok (input/output) |
|---------|--------------|------------------------------|
| King (Kai) | Opus | $15 / $75 |
| Developer, Architect, Security | Sonnet | $3 / $15 |
| Researcher, Designer | Sonnet | $3 / $15 |
| Reviewer, Tester, QA, Docs, DevOps | Haiku | $0.25 / $1.25 |

This maps to the "smart model" vs "fast model" pattern from the OpenClaw diagram. Full multi-model routing is v8, but cost estimation and model selection start now.

**CLI extension:**

```bash
mc spawn developer "Build login form" --zone frontend              # Uses default: sonnet
mc spawn developer "Build login form" --zone frontend --model haiku # Override to haiku
```

Add `--model` flag to `cmd/mc/spawn.go`. Store in `workers.json` as `"model": "sonnet"`. The accumulator reads this field for cost calculation. When v8.1 lands, the model field enables automatic routing based on task complexity rather than just persona defaults.

---

## Phase 6: REST API

All endpoints served by `mc serve`. The API reads from `.mission/` state files (via `mc` CLI or direct file reads) and writes via `mc` CLI commands.

### 6.1 Read endpoints

```
GET  /api/health                    → {"status":"ok","version":"..."}
GET  /api/status                    → Full mission snapshot (stage, tasks, workers, gates, zones, tokens, project)
GET  /api/tasks                     → Task list (query: ?stage=&zone=&status=&persona=)
GET  /api/tasks/:id                 → Single task with dependencies
GET  /api/graph                     → Traceability graph (nodes + edges, multi-type for Traceability View)
GET  /api/workers                   → Active workers with health, tokens, uptime
GET  /api/workers/:id               → Single worker detail
GET  /api/workers/:id/logs          → Recent stdout lines for this worker
GET  /api/gates                     → All gate criteria and status per stage
GET  /api/gates/:stage              → Gate criteria for specific stage
GET  /api/zones                     → Zone list with worker counts and activity
GET  /api/checkpoints               → Checkpoint history
GET  /api/audit                     → Paginated audit log (?limit=&offset=&category=&actor=&after=&before=)
GET  /api/tokens                    → Token usage summary (total, per-session, budget, burn rate)
GET  /api/projects                  → Project list from ~/.mission-control/config.json (sorted by lastOpened desc)
GET  /api/openclaw/status           → OpenClaw bridge connection state
GET  /api/requirements              → [] placeholder (v7: requirements with hierarchy)
GET  /api/requirements/coverage     → {"total":0,"implemented":0,"coverage":0.0} placeholder (v7: coverage report)
GET  /api/specs                     → [] placeholder (v7: specs with section status)
GET  /api/specs/orphans             → [] placeholder (v7: orphaned spec sections)
```

### 6.2 Action endpoints

```
POST /api/chat                      → Send message to Kai via OpenClaw (body: {"message":"..."})
POST /api/gates/:stage/approve      → Approve gate, advance stage
POST /api/gates/:stage/reject       → Reject gate with reason
POST /api/workers/spawn             → Spawn worker (body: {"persona":"developer","task":"...","zone":"frontend","model":"sonnet"})
POST /api/workers/:id/kill          → Kill worker
POST /api/checkpoints               → Create manual checkpoint
POST /api/checkpoints/:id/restart   → Restart from checkpoint
POST /api/tasks                     → Create task (body: {"title":"...","stage":"...","zone":"...","dependencies":[],"req":"REQ-xxx"})
POST /api/tasks/:id/dependencies    → Add/remove dependencies (body: {"add":["mc-xxx"],"remove":["mc-yyy"]})
PATCH /api/tasks/:id                → Update task (status, assignment)
POST /api/stages/override           → Force stage change (body: {"stage":"implement","direction":"advance|rollback"})
POST /api/projects/switch           → Hot-swap active project without restart (body: {"path":"/path/to/project"})
```

**Notes on action endpoints:**

- `POST /api/workers/spawn` accepts an optional `model` field (`haiku`, `sonnet`, `opus`). If omitted, defaults by persona per the model-tier table in Phase 5.4. This prepares for v8.1 multi-model routing.
- `POST /api/tasks` accepts an optional `req` field for requirement linkage. Ignored until v7 populates requirements, but the field is stored on the task immediately.
- `POST /api/projects/switch` hot-swaps the `.mission/` directory the watcher, tracker, and token accumulator point at. The hub broadcasts a `sync.initial_state` event with the new project's state so all connected dashboards refresh without reconnecting.

### 6.3 Traceability graph endpoint

The `/api/graph` endpoint (renamed from `/api/tasks/graph`) returns a unified traceability structure designed to grow from task dependencies today into full requirements → specs → tasks lineage when v7 lands. The schema supports multiple node and edge types from day one.

```json
{
    "nodes": [
        {
            "id": "mc-a1b2c",
            "type": "task",
            "title": "Build login form",
            "stage": "implement",
            "zone": "frontend",
            "status": "in_progress",
            "persona": "developer",
            "worker_id": "mc-x1y2z"
        }
    ],
    "edges": [
        {
            "source": "mc-a1b2c",
            "target": "mc-d3e4f",
            "type": "blocks"
        }
    ],
    "critical_path": ["mc-a1b2c", "mc-d3e4f", "mc-g5h6i"],
    "blocked_count": 3,
    "ready_count": 5
}
```

**Node types (v6.1):** `task`
**Node types (v7):** `requirement`, `spec`, `task`

**Edge types (v6.1):** `blocks`
**Edge types (v7):** `blocks`, `implements` (task → requirement), `derives_from` (requirement → requirement), `traces_to` (task → spec section)

The frontend Traceability View renders this as a layered graph: requirements at top, specs in middle, tasks at bottom. Until v7 populates requirements and specs, only the task layer is visible.

### 6.4 v7-ready placeholder endpoints

These return empty arrays now but establish the API contract so the frontend can scaffold the Traceability View:

```
GET  /api/requirements              → [] (v7: requirement list with hierarchy)
GET  /api/requirements/:id          → 404 (v7: single requirement with derivedFrom, linked tasks)
GET  /api/requirements/coverage     → {"total":0,"implemented":0,"coverage":0.0} (v7: implementation status rollup)
GET  /api/specs                     → [] (v7: spec list with section status)
GET  /api/specs/:file/status        → 404 (v7: completion per section)
GET  /api/specs/orphans             → [] (v7: sections without implementing tasks)
```

When v7 CLI commands (`mc req create`, `mc trace`, etc.) populate `.mission/requirements/` and the watcher detects changes, these endpoints start returning real data with zero API changes.

---

## Phase 7: Infrastructure — CORS, Auth, Tunnel

### 7.1 CORS

`mc serve` must accept requests from `darlington.dev`:

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        allowed := []string{
            "https://darlington.dev",
            "https://www.darlington.dev",
            "http://localhost:3000",  // Local dev
        }
        for _, a := range allowed {
            if origin == a {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 7.2 WebSocket auth

Once exposed via Cloudflare Tunnel, the WebSocket needs auth. Simple bearer token approach:

```go
// On WebSocket upgrade, check Authorization header or ?token= query param
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    token := r.Header.Get("Authorization")
    if token == "" {
        token = r.URL.Query().Get("token")
    }
    if token != expectedToken {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    // ... upgrade connection
}
```

Token set via `MC_API_TOKEN` environment variable. Optional — if not set, auth is disabled (local dev mode).

### 7.3 API-only flag

```bash
mc serve --api-only    # Skip embedded UI, only serve REST + WS
mc serve               # Serve embedded UI + REST + WS (default, fallback for offline)
```

### 7.4 Headless mode (v8.4 forward-compatible)

```bash
mc serve --headless    # No HTTP server, outputs events as JSON lines to stdout
```

Headless mode is for CI/CD pipelines that run automated missions. Instead of serving HTTP, the orchestrator:
- Watches `.mission/state/` for changes
- Outputs events as newline-delimited JSON to stdout
- Accepts commands via stdin (same JSON format as WebSocket messages)
- Exits with code 0 on successful release gate, non-zero on error

This prepares for v8.4 without requiring the full CI/CD integration now. The flag just needs to swap the HTTP server for stdio.

### 7.5 Cloudflare Tunnel setup

```bash
# On your machine
cloudflared tunnel --url http://localhost:8080 --hostname mc.darlington.dev
```

The darlington.dev React app connects to `wss://mc.darlington.dev/ws` and `https://mc.darlington.dev/api/`.

### 7.6 Dockerfile (v8.5 forward-compatible)

Basic containerisation for remote deployment:

```dockerfile
FROM golang:1.22 AS go-builder
WORKDIR /src
COPY . .
RUN cd core && cargo build --release
RUN cd cmd/mc && go build -o /usr/local/bin/mc .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=go-builder /usr/local/bin/mc /usr/local/bin/mc
COPY --from=go-builder /src/core/target/release/mc-core /usr/local/bin/mc-core
EXPOSE 8080
ENTRYPOINT ["mc", "serve", "--api-only"]
```

```yaml
# docker-compose.yml
services:
  mc:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./project:/workspace
    environment:
      - MC_API_TOKEN=${MC_API_TOKEN}
      - OPENCLAW_TOKEN=${OPENCLAW_TOKEN}
    working_dir: /workspace
```

Multi-tenant support (v8.5) is out of scope — this is single-project, single-user. But the container runs headless and can be deployed anywhere.

---

## Phase 8: Dashboard Views

This phase defines what each view needs from the orchestrator — not the frontend implementation itself.

### 8.1 Mission View (main operational dashboard)

**Data sources:**
- WebSocket topics: `stage`, `task`, `worker`, `gate`, `chat`, `token`
- REST: `GET /api/status` (initial load)

**Displays:**
- 10-stage pipeline progress bar with current stage highlighted
- Active workers list with persona, zone, status, uptime, live token count
- Gate status panel for current stage (criteria checklist, approve button)
- Kai chat panel (send/receive via `POST /api/chat` and `chat.*` events)
- Token usage in header (total tokens, estimated cost, burn rate)

### 8.2 Zones View

**Data sources:**
- WebSocket topics: `worker`, `zone`
- REST: `GET /api/zones`, `GET /api/workers`

**Displays:**
- Zone cards showing: zone name, assigned workers (with persona icons), task count, activity indicator
- Worker cards within zones: persona, task title, status badge, uptime, live stdout preview
- Workers entering/leaving zones animate between cards

### 8.3 Traceability View

**Data sources:**
- WebSocket topics: `task`, `requirement`, `spec`
- REST: `GET /api/graph`, `GET /api/requirements`, `GET /api/requirements/coverage`, `GET /api/specs`

**Displays:**
- Interactive dependency graph (D3 force-directed or dagre layout)
- **Three-layer layout** (v7-ready): requirements at top, specs in middle, tasks at bottom. Until v7, only the task layer is populated.
- Node colours by status (pending/in_progress/complete/blocked)
- Node shapes by type: rectangle for tasks, diamond for requirements (v7), hexagon for specs (v7)
- Node badges by stage and persona
- Edge arrows by type: `blocks` (red), `implements` (blue, v7), `derives_from` (grey, v7), `traces_to` (green, v7)
- Critical path highlighted
- `mc ready` tasks highlighted (green border — no open blockers)
- `mc blocked` tasks highlighted (red border — has unresolved dependencies)
- Click node → detail panel with task info, assigned worker, findings, linked requirements
- Filter controls: by stage, zone, status, persona, node type
- Coverage indicator (v7): percentage of requirements with implementing tasks
- Orphan indicator (v7): spec sections with no linked tasks

### 8.4 Activity View

**Data sources:**
- WebSocket topics: `audit`, `checkpoint`, `token`
- REST: `GET /api/audit`, `GET /api/checkpoints`, `GET /api/tokens`

**Displays:**
- Audit trail timeline (filterable by category, actor, time range)
- Checkpoint history with restore buttons
- Token usage over time chart (line graph: tokens burned per minute/hour)
- Session cost breakdown (pie chart: cost by persona/model tier)

---

## Phase 9: New Capabilities (from OpenClaw diagram insights)

### 9.1 Per-worker log streams

Each worker gets a dedicated log buffer (last 200 lines). The dashboard can view logs per worker, inspired by the `#build-logs` / `#find-logs` channel pattern.

```go
type LogBuffer struct {
    lines    []LogLine
    maxLines int
    mu       sync.RWMutex
}

type LogLine struct {
    Timestamp time.Time `json:"timestamp"`
    Content   string    `json:"content"`
    Stream    string    `json:"stream"` // "stdout" or "stderr"
}
```

Accessible via `GET /api/workers/:id/logs` and streamed in real-time via `worker.output` events.

### 9.2 Worker memory (per-persona accumulation)

NEW concept from the OpenClaw diagram (`#find-mem`, `#build-mem`). When a persona completes a task, key findings are appended to a per-persona memory file:

```
.mission/memory/
├── researcher.md    # What researchers have found across spawns
├── developer.md     # Implementation decisions, patterns established
├── reviewer.md      # Review findings, recurring issues
└── ...
```

When a new worker spawns, their briefing includes the relevant memory file. This means the third researcher spawned in a project knows what the first two already found.

Implementation: `mc handoff` appends a summary to `.mission/memory/<persona>.md` after storing findings. Worker prompts are updated to include `{{persona_memory}}` template variable.

### 9.3 Stage override

Force-advance or rollback the workflow stage. Useful for development and debugging.

```bash
mc stage override implement          # Force to implement stage
mc stage override design --rollback  # Roll back to design
```

REST: `POST /api/stages/override` with body `{"stage": "implement", "direction": "advance"}`.

Emits `stage.changed` event with `"override": true` flag. Audit log records the override with human actor.

---

## Task Execution Order

### Critical Path (must be sequential)

```
1.1 Rebuild binary
    └→ 1.2 Verify OpenClaw connects
        └→ 1.3 Decommission old orchestrator
            └→ 2.* WebSocket hub rewrite
                └→ 4.* File watcher integration (depends on hub)
```

### Parallel Tracks (after hub is wired)

```
Track A: Process Tracker (Phase 3)
    3.1 Tracker model
    3.2 Process discovery
    3.3 Heartbeat
    3.4 Kill support

Track B: Token Tracking (Phase 5)
    5.1 Accumulator
    5.2 Integration points
    5.3 Events
    5.4 Cost estimation + --model flag on mc spawn

Track C: REST API (Phase 6)
    6.1 Read endpoints
    6.2 Action endpoints
    6.3 Traceability graph endpoint
    6.4 v7 placeholder endpoints

Track D: Infrastructure (Phase 7)
    7.1 CORS
    7.2 WS auth
    7.3 API-only flag
    7.4 Headless flag
    7.5 Tunnel setup
    7.6 Dockerfile

Track E: v6 TODO Closure (Phase 10)
    10.1 Verify checkpoint round-trip test (F7)
    10.2 Verify auto-checkpoint on gate test (F8)
    10.3 Agent Teams test (D3)
    10.4 Checkpoint skill startup briefing (G6.1)
    10.5 Checkpoint skill pre-compaction (G6.2)
    10.6 Dynamic project switching without restart
```

### Later

```
Phase 8: Dashboard views (frontend, separate repo/track)
Phase 9: New capabilities (per-worker logs, worker memory, stage override)
```

---

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `orchestrator/main.go` | **Delete** | Legacy entry point |
| `orchestrator/v4/` | **Delete** | Legacy in-memory store |
| `orchestrator/terminal/` | **Delete** | PTY handler (not needed) |
| `orchestrator/ws/hub.go` | **Rewrite** | Namespaced event hub with subscriptions |
| `orchestrator/ws/client.go` | **Rewrite** | Client with topic filtering |
| `orchestrator/tracker/tracker.go` | **Create** | Process tracker (replaces parts of manager/) |
| `orchestrator/tracker/logs.go` | **Create** | Per-worker log buffer |
| `orchestrator/tokens/accumulator.go` | **Create** | Token tracking accumulator |
| `orchestrator/watcher/watcher.go` | **Modify** | Add dependency diffing, zone activity, requirements/specs watching, fsnotify |
| `orchestrator/api/routes.go` | **Rewrite** | New REST endpoint registration |
| `orchestrator/api/mission.go` | **Create** | Mission status, tasks endpoints |
| `orchestrator/api/graph.go` | **Create** | Traceability graph endpoint (multi-type nodes/edges) |
| `orchestrator/api/workers.go` | **Create** | Worker endpoints (spawn with --model, kill, logs) |
| `orchestrator/api/gates.go` | **Create** | Gate endpoints (check, approve, reject) |
| `orchestrator/api/stages.go` | **Create** | Stage override endpoint |
| `orchestrator/api/tokens.go` | **Create** | Token usage endpoints |
| `orchestrator/api/traceability.go` | **Create** | v7 placeholder endpoints (requirements, specs, coverage, orphans) |
| `orchestrator/api/middleware.go` | **Create** | CORS, auth middleware |
| `cmd/mc/serve.go` | **Modify** | Wire up hub, watcher, tracker, token accumulator, headless flag |
| `cmd/mc/stage.go` | **Modify** | Add `mc stage override` subcommand |
| `cmd/mc/spawn.go` | **Modify** | Add `--model` flag, store model in workers.json |
| `cmd/mc/handoff.go` | **Modify** | Append summary to `.mission/memory/<persona>.md` (Phase 9.2) |
| `cmd/mc/prompts.go` | **Modify** | Add `{{persona_memory}}` template variable |
| `Dockerfile` | **Create** | Basic container for remote deployment |
| `docker-compose.yml` | **Create** | Single-service compose for easy startup |

## Phase 10: Close Remaining v6 TODOs

These items from `TODO.md` must be closed as part of this rebuild.

### 10.1 Checkpoint round-trip test (F7)

The test exists in `cmd/mc/mc_test.go` (`TestF7_CheckpointRoundTrip`). Run it and verify it passes:
- Create checkpoint → change state → read checkpoint → verify original state preserved
- Verify live state was not mutated by checkpoint read

### 10.2 Auto-checkpoint on gate approval test (F8)

The test exists in `cmd/mc/mc_test.go` (`TestF8_AutoCheckpointOnGateApproval`). Run and verify:
- Approve gate → verify checkpoint auto-created in `orchestrator/checkpoints/`
- Verify git commit created with `[mc:checkpoint]` prefix

### 10.3 Agent Teams test (D3)

Test spawning workers with MC worker personas via OpenClaw Agent Teams:
- Kai spawns a team of workers (e.g., developer + tester for implement stage)
- Workers receive team context in their briefing
- Verify coordination: tester waits for developer's handoff before starting

### 10.4 Checkpoint skill startup briefing (G6.1)

Update the OpenClaw MissionControl skill (`~/.openclaw/workspace/skills/missioncontrol/SKILL.md`) so that on startup, Kai reads `.mission/orchestrator/checkpoints/current.json` (if present) for a compiled briefing of where the project left off.

### 10.5 Checkpoint skill pre-compaction (G6.2)

Update the skill so that before OpenClaw's memory compaction triggers, it calls `mc checkpoint` to preserve state. The compaction hook should:
1. Run `mc checkpoint` to snapshot current state
2. Run `mc checkpoint restart` to get a compiled briefing
3. Include the briefing in the compacted memory

### 10.6 Dynamic project switching without restart

`POST /api/projects/switch` must hot-swap the active project:

```go
func (s *Server) handleProjectSwitch(newPath string) error {
    // 1. Stop current file watcher
    s.watcher.Stop()

    // 2. Reset tracker (kill tracked processes for old project)
    s.tracker.Reset()

    // 3. Reset token accumulator
    s.tokens.Reset()

    // 4. Update mission directory
    s.missionDir = filepath.Join(newPath, ".mission")

    // 5. Start new watcher on new directory
    s.watcher = watcher.NewWatcher(s.missionDir)
    s.watcher.Start()

    // 6. Broadcast new initial state to all clients
    s.hub.BroadcastInitialState(s.buildSnapshot())

    // 7. Update lastOpened in global config
    updateLastOpened(newPath)

    return nil
}
```

### 10.7 Sort projects by lastOpened

`GET /api/projects` returns the project list from `~/.mission-control/config.json` sorted by `lastOpened` descending, so the sidebar shows most recently used projects first.

---

## Roadmap Alignment

How this spec maps to the FUTURETODO roadmap:

### v7 — Requirements & Traceability (Next)

| v7 Item | Foundation Laid in This Spec |
|---------|------------------------------|
| 7.1 Requirements directory | File watcher watches `requirements/` (Phase 4.1) |
| 7.2 `mc req` CRUD | Placeholder `GET /api/requirements` endpoint (Phase 6.4) |
| 7.3 Requirement hierarchy | Graph endpoint supports `derives_from` edge type (Phase 6.3) |
| 7.4 Task-to-requirement refs | Task create accepts `req` field (Phase 6.2), graph has `implements` edge type |
| 7.5 Task-to-spec refs | Graph supports `traces_to` edge type, `GET /api/specs` placeholder |
| 7.6 Impact analysis | Graph endpoint provides the data; v7 adds the traversal logic |
| 7.7 Coverage report | Placeholder `GET /api/requirements/coverage` endpoint (Phase 6.4) |
| 7.8–7.9 Spec status/orphans | Placeholder `GET /api/specs/orphans` endpoint (Phase 6.4) |
| 7.10 Trace command | Graph endpoint with multi-type nodes enables full lineage queries |
| 7.11 Requirements index cache | Not started — v7 scope |
| 7.12 Auto-generate tasks from spec | Not started — v7 scope |

**v7 can land as a pure CLI + Rust extension.** The orchestrator already watches the directories, serves the graph, and has placeholder endpoints. v7 just populates the data.

### v8 — Infrastructure & Scale

| v8 Item | Foundation Laid in This Spec |
|---------|------------------------------|
| 8.1 Multi-model routing | `--model` flag on spawn, model field in workers.json, per-model cost tracking (Phase 5.4) |
| 8.2 Cost tracking & budgets | Full token accumulator with per-worker/session/total tracking, budget warnings (Phase 5) |
| 8.3 Worker health monitoring | Heartbeat, process polling, health events (Phase 3.3) |
| 8.4 Headless mode for CI/CD | `--headless` flag with stdio JSON protocol (Phase 7.4) |
| 8.5 Remote bridge deployment | Dockerfile + docker-compose.yml, CORS, auth token (Phase 7) |

### v9 — Advanced UI (darlington.dev)

| v9 Item | Foundation Laid in This Spec |
|---------|------------------------------|
| 9.1 Requirements panel | Traceability View designed with three-layer layout (Phase 8.3) |
| 9.2 Dependency graph | Graph endpoint with D3-ready structure (Phase 6.3) |
| 9.3 Traceability view | Full view spec with multi-type nodes, edge types, coverage indicators (Phase 8.3) |
| 9.4 Audit log viewer | Activity View with timeline, filters, charts (Phase 8.4) |

---

## Decommission Checklist

- [ ] Delete `orchestrator/main.go`
- [ ] Delete `orchestrator/v4/` directory
- [ ] Delete `orchestrator/terminal/` directory
- [ ] Remove Ollama handler references (flag for future removal, decommission in later version)
- [ ] Update Makefile — remove any `cd orchestrator && go run .` targets
- [ ] Update ARCHITECTURE.md — remove references to standalone orchestrator
- [ ] Update README.md — `mc serve` is the only way to start
- [ ] Update CONTRIBUTING.md — development workflow uses `mc serve`
- [ ] Remove `web/vite.config.ts` proxy rules (UI now on Vercel, not proxied through Vite to local orchestrator)
- [ ] Update FUTURETODO.md — mark v8.2, v8.3, v8.4, v8.5 as "foundations laid in v6.1"
- [ ] Update TODO.md — close items completed in Phase 10

---

## Testing

### Unit Tests

- Hub: subscribe/unsubscribe, topic filtering, broadcast, initial state sync
- Watcher: diff detection for all watched files, dependency change detection, requirements/specs directory watching
- Tracker: process discovery, heartbeat, kill, status tracking
- Accumulator: token counting, budget checking, cost estimation, per-model cost rates
- REST endpoints: all read and action endpoints with mock state
- Graph endpoint: multi-type nodes, edge types, critical path calculation
- v7 placeholders: return empty arrays/objects with correct schema

### Integration Tests

- `mc serve` starts and serves health endpoint
- OpenClaw bridge connects and relays chat
- File change → watcher → hub → WebSocket client receives event
- Worker spawned → tracker picks up → heartbeat events emitted
- Token counting on worker output → accumulator updates → event emitted
- CORS headers present for darlington.dev origin
- WebSocket auth rejects invalid tokens
- Project switch → watcher restarts → new initial state broadcast
- `mc spawn --model haiku` → model stored in workers.json → accumulator uses haiku rates

### v6 TODO Closure Tests

- F7: Checkpoint round-trip (create → restart → verify briefing → verify state)
- F8: Auto-checkpoint on gate approval (approve → verify checkpoint + git commit)
- D3: Agent Teams spawning with MC worker personas via OpenClaw

### E2E Tests

- Dashboard loads initial state via `GET /api/status`
- Dashboard receives real-time events via WebSocket
- Chat message sent → Kai responds → response appears
- Gate approved → stage advances → UI updates
- Worker spawned → appears in zones view → logs stream in
- Traceability View renders task dependency graph from `GET /api/graph`
- Project switch via dashboard → all views refresh with new project data