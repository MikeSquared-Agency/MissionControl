# Data Flows

This document details the data flows within MissionControl.

## Core Data Flow Overview

```mermaid
flowchart TB
    subgraph User["User Interaction"]
        U[User Input]
        D[Dashboard View]
    end

    subgraph UI["React UI"]
        WS[WebSocket Client]
        HTTP[HTTP Client]
        Store[Zustand Stores]
    end

    subgraph Bridge["Go Bridge"]
        Hub[WebSocket Hub]
        REST[REST API]
        Watch[File Watcher]
        Proc[Process Manager]
    end

    subgraph Agents["Agent Layer"]
        King[King Agent]
        W1[Worker 1]
        W2[Worker 2]
    end

    subgraph State["State Layer"]
        FS[".mission/ Files"]
        Core[mc-core Validation]
        CLI[mc CLI]
    end

    U --> HTTP
    U --> WS
    HTTP --> REST
    WS <--> Hub
    Store --> D

    REST --> Proc
    Proc --> King
    Proc --> W1
    Proc --> W2

    King --> CLI
    W1 --> CLI
    W2 --> CLI

    CLI --> Core
    Core --> FS
    FS --> Watch
    Watch --> Hub
    Hub --> Store
```

## User Request Flow (King Mode)

When a user sends a request to King:

```mermaid
sequenceDiagram
    participant User
    participant UI as React UI
    participant WS as WebSocket
    participant Bridge as Go Bridge
    participant King as King Agent
    participant CLI as mc CLI
    participant State as .mission/
    participant Worker

    User->>UI: "Build a login page"
    UI->>WS: Send message
    WS->>Bridge: WebSocket frame
    Bridge->>King: Write to stdin

    King->>King: Parse request
    King->>State: Read existing specs
    King->>User: Clarifying questions
    User->>King: Answers

    King->>State: Write spec to specs/
    King->>CLI: mc task create "Design login"
    CLI->>State: Update tasks.json
    State-->>Bridge: File change
    Bridge-->>UI: task_created event

    King->>CLI: mc spawn designer "Design login" --zone frontend
    CLI->>Bridge: Spawn request
    Bridge->>Worker: Start Claude Code
    State-->>Bridge: File change
    Bridge-->>UI: worker_spawned event

    Worker->>Worker: Execute task
    Worker->>CLI: mc handoff findings.json
    CLI->>State: Validate & store
    State-->>Bridge: File change
    Bridge-->>UI: findings_ready event
    Bridge-->>King: findings_ready event

    King->>State: Read findings
    King->>King: Synthesize
    King->>User: Report progress
```

## Worker Handoff Flow

Detailed flow when a worker completes a task:

```mermaid
flowchart TD
    A[Worker completes task] --> B[Format findings as JSON]
    
    B --> C[Create findings.json]
    C --> D["Run: mc handoff findings.json"]
    
    D --> E{mc validates JSON schema}
    E -->|Invalid| F[Return error to worker]
    F --> G[Worker retries or reports failure]
    
    E -->|Valid| H["Call: mc-core validate-handoff"]
    
    H --> I{Semantic validation}
    I -->|Invalid| F
    
    I -->|Valid| J[Store raw JSON in .mission/handoffs/]
    J --> K[Compress findings to .mission/findings/]
    K --> L[Update task status in tasks.json]
    L --> M[Set task status = complete]
    
    M --> N[Go file watcher detects change]
    N --> O[Emit findings_ready event]
    O --> P[WebSocket broadcasts to all clients]
    
    P --> Q[King receives event]
    Q --> R[King reads .mission/findings/task-id.json]
    R --> S[King synthesizes findings]
    S --> T{More tasks needed?}
    T -->|Yes| U[Create new tasks]
    T -->|No| V[Proceed to gate check]
```

## Gate Approval Flow

```mermaid
sequenceDiagram
    participant King
    participant CLI as mc CLI
    participant Core as mc-core
    participant State as .mission/
    participant UI as React UI
    participant User

    Note over King: All stage tasks complete
    
    King->>CLI: mc gate check design
    CLI->>Core: check-gate design
    Core->>State: Read state files
    
    alt Criteria Not Met
        Core-->>CLI: {status: "not_ready", missing: [...]}
        CLI-->>King: Gate not ready
        King->>King: Spawn workers for missing criteria
    else Criteria Met
        Core-->>CLI: {status: "ready"}
        CLI-->>King: Gate ready
        King->>UI: "Design complete. Approve gate?"
        UI->>User: Show approval dialog
        
        User->>UI: Click "Approve"
        UI->>CLI: mc gate approve design
        CLI->>State: Update phase.json
        CLI->>State: Update gates.json
        
        State-->>UI: stage_changed event
        State-->>King: stage_changed event
        
        King->>King: Begin next phase (Implement)
        King->>CLI: mc task create "Implement login API"
    end
```

## WebSocket Event Flow

```mermaid
flowchart TD
    subgraph Sources["Event Sources"]
        FW[File Watcher<br/>Poll .mission/state/]
        KP[King Process<br/>stdout/stderr]
        WP[Worker Processes<br/>stdout/stderr]
        API[REST API<br/>Actions]
    end

    subgraph Processing["Event Processing"]
        Parse[Parse Event]
        Format[Format JSON]
        Type[Determine Type]
    end

    subgraph Hub["WebSocket Hub"]
        Queue[Event Queue]
        Broadcast[Broadcast]
    end

    subgraph Clients["Connected Clients"]
        C1[Browser 1]
        C2[Browser 2]
        C3[Browser N]
    end

    FW --> Parse
    KP --> Parse
    WP --> Parse
    API --> Parse

    Parse --> Format
    Format --> Type
    Type --> Queue
    Queue --> Broadcast
    Broadcast --> C1 & C2 & C3
```

### Event Types

```mermaid
graph TD
    subgraph Mission["Mission Events"]
        MS[mission_state<br/>Initial sync]
        PC[stage_changed]
        TC[task_created]
        TU[task_updated]
        GR[gate_ready]
        GA[gate_approved]
    end

    subgraph Agent["Agent Events"]
        AS[agent_spawned]
        AST[agent_status]
        ASTOP[agent_stopped]
        AK[agent_killed]
        AA[agent_attention]
        TK[tokens_updated]
    end

    subgraph Worker["Worker Events"]
        WS[worker_spawned]
        WC[worker_completed]
        WE[worker_errored]
        FR[findings_ready]
    end

    subgraph King["King Events"]
        KS[king_status]
        KO[king_output]
        KE[king_error]
        KM[king_message]
    end

    subgraph Zone["Zone Events"]
        ZC[zone_created]
        ZU[zone_updated]
        ZD[zone_deleted]
        ZL[zone_list]
    end
```

## State Synchronization

```mermaid
sequenceDiagram
    participant FS as .mission/ Files
    participant Watcher as File Watcher
    participant Hub as WebSocket Hub
    participant UI as React UI
    participant Store as Zustand Store

    Note over Watcher: Polls every 500ms

    FS->>Watcher: File modified
    Watcher->>Watcher: Compare with last state
    
    alt State Changed
        Watcher->>Watcher: Compute delta
        Watcher->>Hub: Emit event
        Hub->>UI: WebSocket message
        UI->>Store: Update state
        Store->>UI: Re-render
    else No Change
        Watcher->>Watcher: Continue polling
    end
```

## Token Budget Flow

```mermaid
flowchart TD
    A[File to count] --> B["mc-core count-tokens file"]
    B --> C[tiktoken-rs encodes]
    C --> D[Return token count]
    
    D --> E{Within budget?}
    E -->|Yes| F[Proceed with file]
    E -->|No| G[Compress/summarize]
    G --> D
    
    subgraph Budget["Token Tracking"]
        T1[King: ~8K context]
        T2[Worker briefing: ~300 tokens]
        T3[Handoff: ~500 tokens]
    end
```

## Checkpoint/Recovery Flow

```mermaid
sequenceDiagram
    participant King
    participant CLI as mc CLI
    participant State as .mission/
    participant CP as checkpoints/

    Note over King: Periodic checkpoint

    King->>CLI: mc checkpoint create
    CLI->>State: Read all state files
    CLI->>CLI: Bundle into snapshot
    CLI->>CP: Write checkpoint-{timestamp}.json
    CLI-->>King: Checkpoint created

    Note over King: Later: Recovery needed

    King->>CLI: mc checkpoint restore {id}
    CLI->>CP: Read checkpoint file
    CLI->>State: Restore all state files
    CLI-->>King: State restored
```

## Multi-Zone Data Flow

```mermaid
flowchart TB
    subgraph System["System Zone"]
        SPEC[specs/]
        CONFIG[config.json]
    end

    subgraph Frontend["Frontend Zone"]
        FE_SRC[src/components/]
        FE_TEST[tests/]
    end

    subgraph Backend["Backend Zone"]
        BE_SRC[src/api/]
        BE_TEST[tests/]
    end

    subgraph Workers["Zone-Specific Workers"]
        W_FE[Frontend Developer]
        W_BE[Backend Developer]
        W_SYS[Architect]
    end

    W_SYS -->|writes| SPEC
    W_SYS -->|reads| CONFIG

    W_FE -->|reads| SPEC
    W_FE -->|writes| FE_SRC
    W_FE -->|writes| FE_TEST

    W_BE -->|reads| SPEC
    W_BE -->|writes| BE_SRC
    W_BE -->|writes| BE_TEST

    SPEC -.->|referenced by| FE_SRC
    SPEC -.->|referenced by| BE_SRC
```

## Error Handling Flow

```mermaid
flowchart TD
    A[Error occurs] --> B{Error type}
    
    B -->|Validation| C[mc-core returns error]
    C --> D[CLI reports to caller]
    D --> E[Worker retries or fails]
    
    B -->|Process crash| F[Go bridge detects]
    F --> G[Emit worker_errored]
    G --> H[UI shows error]
    H --> I[King notified]
    I --> J{Recoverable?}
    J -->|Yes| K[Spawn replacement]
    J -->|No| L[Mark task failed]
    
    B -->|Network| M[WebSocket disconnects]
    M --> N[UI shows disconnected]
    N --> O[Auto-reconnect with backoff]
    O --> P[Resync state on connect]
```