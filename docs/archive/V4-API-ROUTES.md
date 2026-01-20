# V4 API Routes

New Go endpoints and WebSocket events for v4. Existing routes in `orchestrator/` remain unchanged.

---

## Strategy Domain

### POST /api/gates/:id/approve

Approve or reject a phase gate.

```go
// Request
type GateApprovalRequest struct {
    ApprovedBy string `json:"approved_by"` // "user" or "king"
    Comment    string `json:"comment,omitempty"`
}

// Response
type GateApprovalResponse struct {
    Gate       Gate `json:"gate"`
    CanProceed bool `json:"can_proceed"`
}
```

**Behavior:**
- Validates gate exists and is in `AwaitingApproval` status
- Updates gate status to `Open`
- Records approval timestamp and approver
- Returns updated gate and whether next phase can start

---

## Workflow Domain

### GET /api/phases

Get current phase and all phase statuses.

```go
// Response
type PhasesResponse struct {
    Current Phase         `json:"current"`
    Phases  []PhaseInfo   `json:"phases"`
}

type PhaseInfo struct {
    Phase  string `json:"phase"`   // "idea", "design", etc.
    Status string `json:"status"`  // "complete", "current", "pending"
}
```

---

### GET /api/tasks

List tasks with optional filters.

```go
// Query params
// ?phase=implement
// ?zone=frontend
// ?status=ready
// ?persona=developer

// Response
type TasksResponse struct {
    Tasks []Task `json:"tasks"`
}

type Task struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    Phase        string   `json:"phase"`
    Zone         string   `json:"zone"`
    Status       string   `json:"status"`
    Persona      string   `json:"persona"`
    Dependencies []string `json:"dependencies"`
    CreatedAt    int64    `json:"created_at"`
    UpdatedAt    int64    `json:"updated_at"`
}
```

---

### GET /api/tasks/:id

Get single task by ID.

```go
// Response
Task  // Same as above
```

---

### POST /api/tasks

Create a new task.

```go
// Request
type CreateTaskRequest struct {
    Name         string   `json:"name"`
    Phase        string   `json:"phase"`
    Zone         string   `json:"zone"`
    Persona      string   `json:"persona"`
    Dependencies []string `json:"dependencies,omitempty"`
}

// Response
Task  // Created task with generated ID
```

---

### PUT /api/tasks/:id/status

Update task status.

```go
// Request
type UpdateTaskStatusRequest struct {
    Status string `json:"status"` // pending, ready, in_progress, blocked, done
    Reason string `json:"reason,omitempty"` // Required if blocked
}

// Response
Task  // Updated task
```

---

## Knowledge Domain

### GET /api/specs/:id

Get a spec file content.

```go
// Response
type SpecResponse struct {
    ID      string `json:"id"`
    Path    string `json:"path"`     // .mission/specs/SPEC-auth.md
    Content string `json:"content"`
    Version int    `json:"version"`
}
```

---

### GET /api/briefings/:worker_id

Get compiled briefing for a worker.

```go
// Response
type BriefingResponse struct {
    WorkerID string `json:"worker_id"`
    TaskID   string `json:"task_id"`
    Content  string `json:"content"`
    Tokens   int    `json:"tokens"`
}
```

**Note:** In v4, this returns briefing inputs. Full LLM-generated briefings come in v5.

---

### POST /api/handoffs

Submit a worker handoff for validation and storage.

```go
// Request
type HandoffRequest struct {
    TaskID              string            `json:"task_id"`
    WorkerID            string            `json:"worker_id"`
    Status              string            `json:"status"` // complete, blocked, partial
    Findings            []Finding         `json:"findings"`
    Artifacts           []string          `json:"artifacts"`
    OpenQuestions       []string          `json:"open_questions,omitempty"`
    ContextForSuccessor *SuccessorContext `json:"context_for_successor,omitempty"`
    BlockedReason       string            `json:"blocked_reason,omitempty"`
    ProgressPercentage  int               `json:"progress_percentage,omitempty"`
}

type Finding struct {
    Type        string `json:"type"` // discovery, blocker, decision, concern
    Summary     string `json:"summary"`
    DetailsPath string `json:"details_path,omitempty"`
    Severity    string `json:"severity,omitempty"` // low, medium, high
}

type SuccessorContext struct {
    KeyDecisions        []string `json:"key_decisions"`
    Gotchas             []string `json:"gotchas"`
    RecommendedApproach string   `json:"recommended_approach,omitempty"`
}

// Response
type HandoffResponse struct {
    Valid   bool     `json:"valid"`
    Errors  []string `json:"errors,omitempty"`
    DeltaID string   `json:"delta_id,omitempty"`
}
```

**Validation rules (enforced by Rust):**
- `task_id` must reference existing task
- `status` must be valid enum
- `findings` array required (can be empty)
- Each finding must have `type` and `summary`
- `summary` must be < 500 chars
- If `status` is `blocked`, `blocked_reason` required

---

### GET /api/checkpoints

List all checkpoints.

```go
// Response
type CheckpointsResponse struct {
    Checkpoints []CheckpointSummary `json:"checkpoints"`
}

type CheckpointSummary struct {
    ID        string `json:"id"`
    Phase     string `json:"phase"`
    CreatedAt int64  `json:"created_at"`
}
```

---

### GET /api/checkpoints/:id

Get full checkpoint data.

```go
// Response
type Checkpoint struct {
    ID               string    `json:"id"`
    Phase            string    `json:"phase"`
    CreatedAt        int64     `json:"created_at"`
    TasksSnapshot    []Task    `json:"tasks_snapshot"`
    FindingsSnapshot []Finding `json:"findings_snapshot"`
    Decisions        []string  `json:"decisions"`
}
```

---

## WebSocket Events

All events sent over existing `/ws` connection.

### Workflow Events

```typescript
// Phase changed
{
  type: "phase_changed",
  phase: "implement",
  previous: "design"
}

// Task created
{
  type: "task_created",
  task: Task
}

// Task status updated
{
  type: "task_updated",
  task_id: "task-123",
  status: "in_progress",
  previous: "ready"
}

// Gate status changed
{
  type: "gate_status",
  phase: "design",
  status: "awaiting_approval",  // open, closed, awaiting_approval
  criteria: [
    { description: "Mockups exist", satisfied: true },
    { description: "API design approved", satisfied: false }
  ]
}
```

### Knowledge Events

```typescript
// Token budget warning (50-75%)
{
  type: "token_warning",
  worker_id: "worker-456",
  usage: 15000,
  budget: 20000,
  status: "warning",
  remaining: 5000
}

// Token budget critical (>75%)
{
  type: "token_critical",
  worker_id: "worker-456",
  usage: 18000,
  budget: 20000,
  remaining: 2000
}

// Checkpoint created
{
  type: "checkpoint_created",
  checkpoint_id: "cp-789",
  phase: "design"
}

// Handoff received
{
  type: "handoff_received",
  task_id: "task-123",
  worker_id: "worker-456",
  status: "complete"
}

// Handoff validation result
{
  type: "handoff_validated",
  task_id: "task-123",
  valid: true,
  errors: []  // or error strings if invalid
}
```

### Runtime Events

```typescript
// Agent health status
{
  type: "agent_health",
  agent_id: "agent-789",
  health: "healthy"  // healthy, idle, stuck, unresponsive, dead
}

// Agent stuck alert
{
  type: "agent_stuck",
  agent_id: "agent-789",
  since_ms: 65000
}
```

---

## TypeScript Types (React)

For `web/src/types/v4.ts`:

```typescript
// Workflow
export type Phase = 'idea' | 'design' | 'implement' | 'verify' | 'document' | 'release';
export type TaskStatus = 'pending' | 'ready' | 'in_progress' | 'blocked' | 'done';
export type GateStatus = 'open' | 'closed' | 'awaiting_approval';

export interface Task {
  id: string;
  name: string;
  phase: Phase;
  zone: string;
  status: TaskStatus;
  persona: string;
  dependencies: string[];
  created_at: number;
  updated_at: number;
}

export interface Gate {
  id: string;
  phase: Phase;
  status: GateStatus;
  criteria: GateCriterion[];
  approved_at?: number;
  approved_by?: string;
}

export interface GateCriterion {
  description: string;
  satisfied: boolean;
}

// Knowledge
export type FindingType = 'discovery' | 'blocker' | 'decision' | 'concern';
export type BudgetStatus = 'healthy' | 'warning' | 'critical' | 'exceeded';

export interface Finding {
  type: FindingType;
  summary: string;
  details_path?: string;
  severity?: 'low' | 'medium' | 'high';
}

export interface TokenBudget {
  worker_id: string;
  budget: number;
  used: number;
  status: BudgetStatus;
  remaining: number;
}

// Runtime
export type HealthStatus = 'healthy' | 'idle' | 'stuck' | 'unresponsive' | 'dead';

export interface WorkerHealth {
  worker_id: string;
  status: HealthStatus;
  since_ms?: number;
}
```

---

## Go Route Registration

Add to `orchestrator/api/routes.go`:

```go
func RegisterV4Routes(r *mux.Router, core *RustCore) {
    // Strategy
    r.HandleFunc("/api/gates/{id}/approve", core.ApproveGate).Methods("POST")
    
    // Workflow
    r.HandleFunc("/api/phases", core.GetPhases).Methods("GET")
    r.HandleFunc("/api/tasks", core.ListTasks).Methods("GET")
    r.HandleFunc("/api/tasks", core.CreateTask).Methods("POST")
    r.HandleFunc("/api/tasks/{id}", core.GetTask).Methods("GET")
    r.HandleFunc("/api/tasks/{id}/status", core.UpdateTaskStatus).Methods("PUT")
    
    // Knowledge
    r.HandleFunc("/api/specs/{id}", core.GetSpec).Methods("GET")
    r.HandleFunc("/api/briefings/{worker_id}", core.GetBriefing).Methods("GET")
    r.HandleFunc("/api/handoffs", core.SubmitHandoff).Methods("POST")
    r.HandleFunc("/api/checkpoints", core.ListCheckpoints).Methods("GET")
    r.HandleFunc("/api/checkpoints/{id}", core.GetCheckpoint).Methods("GET")
}
```