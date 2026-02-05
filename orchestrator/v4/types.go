package v4

// Stage represents a workflow stage
type Stage string

const (
	StageDiscovery Stage = "discovery"
	StageGoal      Stage = "goal"
	StageRequirements Stage = "requirements"
	StagePlanning  Stage = "planning"
	StageDesign    Stage = "design"
	StageImplement Stage = "implement"
	StageVerify    Stage = "verify"
	StageValidate  Stage = "validate"
	StageDocument  Stage = "document"
	StageRelease   Stage = "release"
)

// AllStages returns all stages in order
func AllStages() []Stage {
	return []Stage{
		StageDiscovery,
		StageGoal,
		StageRequirements,
		StagePlanning,
		StageDesign,
		StageImplement,
		StageVerify,
		StageValidate,
		StageDocument,
		StageRelease,
	}
}

// Next returns the next stage, or empty string if at end
func (s Stage) Next() Stage {
	switch s {
	case StageDiscovery:
		return StageGoal
	case StageGoal:
		return StageRequirements
	case StageRequirements:
		return StagePlanning
	case StagePlanning:
		return StageDesign
	case StageDesign:
		return StageImplement
	case StageImplement:
		return StageVerify
	case StageVerify:
		return StageValidate
	case StageValidate:
		return StageDocument
	case StageDocument:
		return StageRelease
	default:
		return ""
	}
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusReady      TaskStatus = "ready"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusDone       TaskStatus = "done"
)

// Task represents a unit of work
type Task struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Stage         Stage      `json:"stage"`
	Zone          string     `json:"zone"`
	Status        TaskStatus `json:"status"`
	BlockedReason string     `json:"blocked_reason,omitempty"`
	Persona       string     `json:"persona"`
	Dependencies  []string   `json:"dependencies"`
	CreatedAt     int64      `json:"created_at"`
	UpdatedAt     int64      `json:"updated_at"`
}

// GateStatus represents the status of a stage gate
type GateStatus string

const (
	GateStatusOpen             GateStatus = "open"
	GateStatusClosed           GateStatus = "closed"
	GateStatusAwaitingApproval GateStatus = "awaiting_approval"
)

// GateCriterion represents a single gate criterion
type GateCriterion struct {
	Description string `json:"description"`
	Satisfied   bool   `json:"satisfied"`
}

// Gate represents a stage gate
type Gate struct {
	ID         string          `json:"id"`
	Stage      Stage           `json:"stage"`
	Status     GateStatus      `json:"status"`
	Criteria   []GateCriterion `json:"criteria"`
	ApprovedAt *int64          `json:"approved_at,omitempty"`
	ApprovedBy string          `json:"approved_by,omitempty"`
}

// FindingType represents the type of finding
type FindingType string

const (
	FindingTypeDiscovery FindingType = "discovery"
	FindingTypeBlocker   FindingType = "blocker"
	FindingTypeDecision  FindingType = "decision"
	FindingTypeConcern   FindingType = "concern"
)

// Finding represents a worker finding
type Finding struct {
	Type        FindingType `json:"type"`
	Summary     string      `json:"summary"`
	DetailsPath string      `json:"details_path,omitempty"`
	Severity    string      `json:"severity,omitempty"`
}

// SuccessorContext provides context for the next worker
type SuccessorContext struct {
	KeyDecisions        []string `json:"key_decisions"`
	Gotchas             []string `json:"gotchas"`
	RecommendedApproach string   `json:"recommended_approach,omitempty"`
}

// HandoffStatus represents the status of a handoff
type HandoffStatus string

const (
	HandoffStatusComplete HandoffStatus = "complete"
	HandoffStatusBlocked  HandoffStatus = "blocked"
	HandoffStatusPartial  HandoffStatus = "partial"
)

// Handoff represents a worker handoff
type Handoff struct {
	TaskID              string            `json:"task_id"`
	WorkerID            string            `json:"worker_id"`
	Status              HandoffStatus     `json:"status"`
	Findings            []Finding         `json:"findings"`
	Artifacts           []string          `json:"artifacts"`
	OpenQuestions       []string          `json:"open_questions,omitempty"`
	ContextForSuccessor *SuccessorContext `json:"context_for_successor,omitempty"`
	BlockedReason       string            `json:"blocked_reason,omitempty"`
	Timestamp           int64             `json:"timestamp"`
}

// BudgetStatus represents token budget status
type BudgetStatus string

const (
	BudgetStatusHealthy  BudgetStatus = "healthy"
	BudgetStatusWarning  BudgetStatus = "warning"
	BudgetStatusCritical BudgetStatus = "critical"
	BudgetStatusExceeded BudgetStatus = "exceeded"
)

// TokenBudget represents a worker's token budget
type TokenBudget struct {
	WorkerID  string       `json:"worker_id"`
	Budget    int          `json:"budget"`
	Used      int          `json:"used"`
	Status    BudgetStatus `json:"status"`
	Remaining int          `json:"remaining"`
}

// Checkpoint represents a project state snapshot
type Checkpoint struct {
	ID               string    `json:"id"`
	Stage            Stage     `json:"stage"`
	SessionID        string    `json:"session_id"`
	CreatedAt        int64     `json:"created_at"`
	TasksSnapshot    []Task    `json:"tasks_snapshot"`
	FindingsSnapshot []Finding `json:"findings_snapshot"`
	Decisions        []string  `json:"decisions"`
	Blockers         []string  `json:"blockers"`
}

// CheckpointSummary is a lightweight checkpoint reference
type CheckpointSummary struct {
	ID        string `json:"id"`
	Stage     Stage  `json:"stage"`
	CreatedAt int64  `json:"created_at"`
}

// HealthStatus represents worker health
type HealthStatus string

const (
	HealthStatusHealthy      HealthStatus = "healthy"
	HealthStatusIdle         HealthStatus = "idle"
	HealthStatusStuck        HealthStatus = "stuck"
	HealthStatusUnresponsive HealthStatus = "unresponsive"
	HealthStatusDead         HealthStatus = "dead"
)

// WorkerHealth represents a worker's health status
type WorkerHealth struct {
	WorkerID string       `json:"worker_id"`
	Status   HealthStatus `json:"status"`
	SinceMs  int64        `json:"since_ms,omitempty"`
}

// StageInfo provides stage status information
type StageInfo struct {
	Stage  Stage  `json:"stage"`
	Status string `json:"status"` // "complete", "current", "pending"
}

// SessionRecord tracks session lifecycle
type SessionRecord struct {
	SessionID    string `json:"session_id"`
	StartedAt    int64  `json:"started_at"`
	EndedAt      int64  `json:"ended_at,omitempty"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
	Stage        Stage  `json:"stage"`
	Reason       string `json:"reason,omitempty"`
}
