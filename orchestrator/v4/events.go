package v4

// WebSocket event types for v4

// WorkflowEvent types
const (
	EventPhaseChanged  = "phase_changed"
	EventTaskCreated   = "task_created"
	EventTaskUpdated   = "task_updated"
	EventGateStatus    = "gate_status"
)

// KnowledgeEvent types
const (
	EventTokenWarning      = "token_warning"
	EventTokenCritical     = "token_critical"
	EventCheckpointCreated = "checkpoint_created"
	EventHandoffReceived   = "handoff_received"
	EventHandoffValidated  = "handoff_validated"
)

// RuntimeEvent types
const (
	EventAgentHealth = "agent_health"
	EventAgentStuck  = "agent_stuck"
)

// PhaseChangedEvent is sent when the phase changes
type PhaseChangedEvent struct {
	Type     string `json:"type"`
	Phase    Phase  `json:"phase"`
	Previous Phase  `json:"previous"`
}

// TaskCreatedEvent is sent when a task is created
type TaskCreatedEvent struct {
	Type string `json:"type"`
	Task *Task  `json:"task"`
}

// TaskUpdatedEvent is sent when a task status changes
type TaskUpdatedEvent struct {
	Type     string     `json:"type"`
	TaskID   string     `json:"task_id"`
	Status   TaskStatus `json:"status"`
	Previous TaskStatus `json:"previous"`
}

// GateStatusEvent is sent when a gate status changes
type GateStatusEvent struct {
	Type     string          `json:"type"`
	Phase    Phase           `json:"phase"`
	Status   GateStatus      `json:"status"`
	Criteria []GateCriterion `json:"criteria,omitempty"`
}

// TokenWarningEvent is sent when token usage reaches warning threshold
type TokenWarningEvent struct {
	Type      string       `json:"type"`
	WorkerID  string       `json:"worker_id"`
	Usage     int          `json:"usage"`
	Budget    int          `json:"budget"`
	Status    BudgetStatus `json:"status"`
	Remaining int          `json:"remaining"`
}

// CheckpointCreatedEvent is sent when a checkpoint is created
type CheckpointCreatedEvent struct {
	Type         string `json:"type"`
	CheckpointID string `json:"checkpoint_id"`
	Phase        Phase  `json:"phase"`
}

// HandoffReceivedEvent is sent when a handoff is received
type HandoffReceivedEvent struct {
	Type     string        `json:"type"`
	TaskID   string        `json:"task_id"`
	WorkerID string        `json:"worker_id"`
	Status   HandoffStatus `json:"status"`
}

// HandoffValidatedEvent is sent after handoff validation
type HandoffValidatedEvent struct {
	Type   string   `json:"type"`
	TaskID string   `json:"task_id"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// AgentHealthEvent is sent when agent health changes
type AgentHealthEvent struct {
	Type    string       `json:"type"`
	AgentID string       `json:"agent_id"`
	Health  HealthStatus `json:"health"`
}

// AgentStuckEvent is sent when an agent is detected as stuck
type AgentStuckEvent struct {
	Type    string `json:"type"`
	AgentID string `json:"agent_id"`
	SinceMs int64  `json:"since_ms"`
}
