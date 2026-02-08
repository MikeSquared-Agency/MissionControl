# Data Flows

## Core Data Flow

```mermaid
flowchart TB
    subgraph User["User"]
        U[User Input]
        D[Dashboard]
    end

    subgraph UI["darlington.dev"]
        WS[WebSocket Client]
        HTTP[HTTP Client]
    end

    subgraph Bridge["Go Orchestrator"]
        Hub[WebSocket Hub]
        REST[REST API]
        Watch[File Watcher]
        Proc[Process Manager]
        OC[OpenClaw Bridge]
    end

    subgraph Agents["Agents"]
        King["King (Kai/OpenClaw)"]
        W1[Worker 1]
        W2[Worker 2]
    end

    subgraph State["State"]
        FS[".mission/ Files"]
        Core[mc-core]
        CLI[mc CLI]
    end

    U --> HTTP & WS
    HTTP --> REST
    WS <--> Hub
    OC <-->|Gateway WS| King
    REST --> Proc
    Proc --> W1 & W2
    King --> CLI
    W1 & W2 --> CLI
    CLI --> Core --> FS
    FS --> Watch --> Hub --> UI
```

## User Request Flow

```mermaid
sequenceDiagram
    participant User
    participant UI as darlington.dev
    participant Bridge as Go Bridge
    participant OC as OpenClaw Bridge
    participant King as Kai (OpenClaw)
    participant CLI as mc CLI
    participant State as .mission/
    participant Worker

    User->>UI: "Build a login page"
    UI->>Bridge: WebSocket message
    Bridge->>OC: Relay to OpenClaw
    OC->>King: Gateway WebSocket

    King->>State: Read existing specs
    King->>CLI: mc task create "Design login"
    CLI->>State: Update tasks.jsonl
    State-->>Bridge: File change → task_created event
    Bridge-->>UI: WebSocket broadcast

    King->>CLI: mc spawn designer "Design login" --zone frontend
    CLI->>Bridge: Spawn request
    Bridge->>Worker: Start Claude Code session
    State-->>UI: worker_spawned event

    Worker->>Worker: Execute task
    Worker->>CLI: mc handoff findings.json
    CLI->>State: Validate (mc-core) → store → update task
    State-->>Bridge: findings_ready event
    Bridge-->>King: Via OpenClaw bridge

    King->>State: Read findings, synthesize
    King->>User: Report progress (via OpenClaw → WhatsApp/web)
```

## Worker Handoff Flow

```mermaid
flowchart TD
    A[Worker completes task] --> B[Format findings as JSON]
    B --> C["mc handoff findings.json"]
    C --> D{Schema validation}
    D -->|Invalid| E[Error → worker retries]
    D -->|Valid| F["mc-core validate-handoff"]
    F --> G{Semantic validation}
    G -->|Invalid| E
    G -->|Valid| H[Store in handoffs/]
    H --> I[Compress to findings/]
    I --> J[Update tasks.jsonl]
    J --> K["[mc:handoff] git auto-commit"]
    K --> L[File watcher → findings_ready event]
    L --> M[King reads & synthesizes]
```

## Gate Approval Flow

```mermaid
sequenceDiagram
    participant King as Kai
    participant CLI as mc CLI
    participant Core as mc-core
    participant State as .mission/

    Note over King: All stage tasks complete

    King->>CLI: mc gate check design
    CLI->>Core: check-gate design
    Core->>State: Read state

    alt Criteria Not Met
        Core-->>King: {status: "not_ready", missing: [...]}
        King->>King: Spawn workers for gaps
    else Criteria Met
        Core-->>King: {status: "ready"}
        King->>CLI: mc gate approve design
        CLI->>State: Update gates.json + stage.json
        CLI->>CLI: Auto-checkpoint
        CLI->>CLI: [mc:gate] git commit
        State-->>King: stage_changed event
        King->>King: Begin next stage
    end
```

## Checkpoint Flow

```mermaid
sequenceDiagram
    participant Trigger as Trigger (gate/threshold/manual)
    participant CLI as mc CLI
    participant Core as mc-core
    participant State as .mission/
    participant Git as Git

    Trigger->>CLI: mc checkpoint
    CLI->>State: Read stage + gates + tasks + decisions
    CLI->>State: Write checkpoints/{timestamp}.json
    CLI->>Git: [mc:checkpoint] auto-commit

    Note over CLI: On restart:

    CLI->>Core: mc-core checkpoint-compile {file}
    Core->>Core: Compile ~500 token briefing
    Core-->>CLI: Markdown briefing
    CLI->>State: Update sessions.jsonl
    CLI->>CLI: Inject briefing into new session
```

## WebSocket Events

| Event | Source | Description |
|-------|--------|-------------|
| `mission_state` | Connect | Initial state sync |
| `stage_changed` | Gate approval | Stage transitioned |
| `task_created` | mc task create | New task |
| `task_updated` | mc task update | Status changed |
| `gate_ready` | File watcher | Gate criteria met |
| `gate_approved` | mc gate approve | Gate approved |
| `worker_spawned` | mc spawn | Worker started |
| `worker_completed` | Handoff | Worker finished |
| `worker_errored` | Process crash | Worker failed |
| `findings_ready` | Handoff stored | New findings |
| `checkpoint_created` | Auto/manual | State snapshot |
| `session_restarted` | Restart | New session with briefing |

## State Synchronization

The file watcher polls `.mission/state/` every 500ms, computes deltas, and broadcasts events through the WebSocket hub. All connected clients (darlington.dev dashboard, other tools) receive real-time updates.

## Error Handling

- **Validation errors**: mc-core returns structured errors, worker retries or fails
- **Process crashes**: Go bridge detects, emits `worker_errored`, King decides to retry or mark failed
- **Network disconnects**: WebSocket auto-reconnect with backoff, full state resync on connect
- **Gateway disconnects**: OpenClaw bridge auto-reconnects to Kai's gateway
