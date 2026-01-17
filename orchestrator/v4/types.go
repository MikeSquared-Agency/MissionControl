package v4

// Phase represents a workflow phase
type Phase string

const (
	PhaseIdea      Phase = "idea"
	PhaseDesign    Phase = "design"
	PhaseImplement Phase = "implement"
	PhaseVerify    Phase = "verify"
	PhaseDocument  Phase = "document"
	PhaseRelease   Phase = "release"
)

// AllPhases returns all phases in order
func AllPhases() []Phase {
	return []Phase{
		PhaseIdea,
		PhaseDesign,
		PhaseImplement,
		PhaseVerify,
		PhaseDocument,
		PhaseRelease,
	}
}

// Next returns the next phase, or empty string if at end
func (p Phase) Next() Phase {
	switch p {
	case PhaseIdea:
		return PhaseDesign
	case PhaseDesign:
		return PhaseImplement
	case PhaseImplement:
		return PhaseVerify
	case PhaseVerify:
		return PhaseDocument
	case PhaseDocument:
		return PhaseRelease
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
	Phase         Phase      `json:"phase"`
	Zone          string     `json:"zone"`
	Status        TaskStatus `json:"status"`
	BlockedReason string     `json:"blocked_reason,omitempty"`
	Persona       string     `json:"persona"`
	Dependencies  []string   `json:"dependencies"`
	CreatedAt     int64      `json:"created_at"`
	UpdatedAt     int64      `json:"updated_at"`
}

// GateStatus represents the status of a phase gate
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

// Gate represents a phase gate
type Gate struct {
	ID         string          `json:"id"`
	Phase      Phase           `json:"phase"`
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
	Phase            Phase     `json:"phase"`
	CreatedAt        int64     `json:"created_at"`
	TasksSnapshot    []Task    `json:"tasks_snapshot"`
	FindingsSnapshot []Finding `json:"findings_snapshot"`
	Decisions        []string  `json:"decisions"`
}

// CheckpointSummary is a lightweight checkpoint reference
type CheckpointSummary struct {
	ID        string `json:"id"`
	Phase     Phase  `json:"phase"`
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

// PhaseInfo provides phase status information
type PhaseInfo struct {
	Phase  Phase  `json:"phase"`
	Status string `json:"status"` // "complete", "current", "pending"
}
