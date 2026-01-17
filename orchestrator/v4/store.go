package v4

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store manages v4 workflow state
// This is an in-memory implementation that will later be backed by Rust FFI
type Store struct {
	mu sync.RWMutex

	currentPhase Phase
	tasks        map[string]*Task
	gates        map[Phase]*Gate
	checkpoints  []*Checkpoint
	handoffs     []*Handoff
	findings     []Finding
	budgets      map[string]*TokenBudget
}

// NewStore creates a new v4 store
func NewStore() *Store {
	s := &Store{
		currentPhase: PhaseIdea,
		tasks:        make(map[string]*Task),
		gates:        make(map[Phase]*Gate),
		checkpoints:  make([]*Checkpoint, 0),
		handoffs:     make([]*Handoff, 0),
		findings:     make([]Finding, 0),
		budgets:      make(map[string]*TokenBudget),
	}

	// Initialize gates for all phases
	for _, phase := range AllPhases() {
		s.gates[phase] = s.createGateForPhase(phase)
	}

	return s
}

func (s *Store) createGateForPhase(phase Phase) *Gate {
	criteria := s.defaultCriteriaForPhase(phase)
	return &Gate{
		ID:       fmt.Sprintf("gate-%s", phase),
		Phase:    phase,
		Status:   GateStatusClosed,
		Criteria: criteria,
	}
}

func (s *Store) defaultCriteriaForPhase(phase Phase) []GateCriterion {
	switch phase {
	case PhaseIdea:
		return []GateCriterion{
			{Description: "Problem statement defined", Satisfied: false},
			{Description: "Feasibility assessed", Satisfied: false},
		}
	case PhaseDesign:
		return []GateCriterion{
			{Description: "Spec document complete", Satisfied: false},
			{Description: "Technical approach approved", Satisfied: false},
		}
	case PhaseImplement:
		return []GateCriterion{
			{Description: "All tasks complete", Satisfied: false},
			{Description: "Code compiles", Satisfied: false},
		}
	case PhaseVerify:
		return []GateCriterion{
			{Description: "Tests passing", Satisfied: false},
			{Description: "Review complete", Satisfied: false},
		}
	case PhaseDocument:
		return []GateCriterion{
			{Description: "README updated", Satisfied: false},
			{Description: "API documented", Satisfied: false},
		}
	case PhaseRelease:
		return []GateCriterion{
			{Description: "Deployed successfully", Satisfied: false},
			{Description: "Smoke tests pass", Satisfied: false},
		}
	default:
		return []GateCriterion{}
	}
}

// Phase operations

// CurrentPhase returns the current phase
func (s *Store) CurrentPhase() Phase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentPhase
}

// GetPhases returns all phase info
func (s *Store) GetPhases() []PhaseInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var infos []PhaseInfo
	for _, phase := range AllPhases() {
		var status string
		if phase == s.currentPhase {
			status = "current"
		} else if s.isPhaseComplete(phase) {
			status = "complete"
		} else {
			status = "pending"
		}
		infos = append(infos, PhaseInfo{Phase: phase, Status: status})
	}
	return infos
}

func (s *Store) isPhaseComplete(phase Phase) bool {
	gate, ok := s.gates[phase]
	if !ok {
		return false
	}
	return gate.Status == GateStatusOpen
}

// CanTransition checks if we can transition to the given phase
func (s *Store) CanTransition(to Phase) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	next := s.currentPhase.Next()
	if next != to {
		return false
	}

	gate, ok := s.gates[s.currentPhase]
	if !ok {
		return false
	}

	return gate.Status == GateStatusOpen
}

// Transition moves to the next phase
func (s *Store) Transition(to Phase) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.currentPhase.Next()
	if next != to {
		return fmt.Errorf("cannot transition from %s to %s", s.currentPhase, to)
	}

	gate, ok := s.gates[s.currentPhase]
	if !ok || gate.Status != GateStatusOpen {
		return fmt.Errorf("gate not open for phase %s", s.currentPhase)
	}

	s.currentPhase = to
	return nil
}

// Task operations

// CreateTask creates a new task
func (s *Store) CreateTask(name string, phase Phase, zone, persona string, deps []string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	task := &Task{
		ID:           uuid.New().String(),
		Name:         name,
		Phase:        phase,
		Zone:         zone,
		Status:       TaskStatusPending,
		Persona:      persona,
		Dependencies: deps,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.tasks[task.ID] = task
	return task
}

// GetTask retrieves a task by ID
func (s *Store) GetTask(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

// UpdateTaskStatus updates a task's status
func (s *Store) UpdateTaskStatus(id string, status TaskStatus, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Status = status
	task.BlockedReason = reason
	task.UpdatedAt = time.Now().Unix()

	return nil
}

// ListTasks returns all tasks, optionally filtered
func (s *Store) ListTasks(phase *Phase, zone, status, persona *string) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Task
	for _, task := range s.tasks {
		if phase != nil && task.Phase != *phase {
			continue
		}
		if zone != nil && task.Zone != *zone {
			continue
		}
		if status != nil && string(task.Status) != *status {
			continue
		}
		if persona != nil && task.Persona != *persona {
			continue
		}
		result = append(result, *task)
	}
	return result
}

// GetReadyTasks returns tasks that are ready to be worked on
func (s *Store) GetReadyTasks() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ready []Task
	for _, task := range s.tasks {
		if task.Status != TaskStatusPending {
			continue
		}

		// Check if all dependencies are done
		allDone := true
		for _, depID := range task.Dependencies {
			dep, ok := s.tasks[depID]
			if !ok || dep.Status != TaskStatusDone {
				allDone = false
				break
			}
		}

		if allDone {
			ready = append(ready, *task)
		}
	}
	return ready
}

// Gate operations

// GetGate returns a gate for a phase
func (s *Store) GetGate(phase Phase) (*Gate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	gate, ok := s.gates[phase]
	return gate, ok
}

// SatisfyCriterion marks a gate criterion as satisfied
func (s *Store) SatisfyCriterion(phase Phase, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	gate, ok := s.gates[phase]
	if !ok {
		return fmt.Errorf("gate not found for phase: %s", phase)
	}

	if index < 0 || index >= len(gate.Criteria) {
		return fmt.Errorf("invalid criterion index: %d", index)
	}

	gate.Criteria[index].Satisfied = true
	s.updateGateStatus(gate)
	return nil
}

// ApproveGate approves a gate
func (s *Store) ApproveGate(phase Phase, approvedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	gate, ok := s.gates[phase]
	if !ok {
		return fmt.Errorf("gate not found for phase: %s", phase)
	}

	now := time.Now().Unix()
	gate.ApprovedAt = &now
	gate.ApprovedBy = approvedBy
	gate.Status = GateStatusOpen

	return nil
}

func (s *Store) updateGateStatus(gate *Gate) {
	allSatisfied := true
	for _, c := range gate.Criteria {
		if !c.Satisfied {
			allSatisfied = false
			break
		}
	}

	if allSatisfied {
		if gate.ApprovedAt != nil {
			gate.Status = GateStatusOpen
		} else {
			gate.Status = GateStatusAwaitingApproval
		}
	} else {
		gate.Status = GateStatusClosed
	}
}

// Handoff operations

// ValidateHandoff validates a handoff
func (s *Store) ValidateHandoff(h *Handoff) []string {
	var errors []string

	if h.TaskID == "" {
		errors = append(errors, "task_id is required")
	}
	if h.WorkerID == "" {
		errors = append(errors, "worker_id is required")
	}
	if h.Status == HandoffStatusBlocked && h.BlockedReason == "" {
		errors = append(errors, "blocked_reason is required when status is blocked")
	}

	for i, f := range h.Findings {
		if f.Summary == "" {
			errors = append(errors, fmt.Sprintf("findings[%d].summary is required", i))
		}
		if len(f.Summary) > 500 {
			errors = append(errors, fmt.Sprintf("findings[%d].summary exceeds 500 chars", i))
		}
	}

	return errors
}

// StoreHandoff stores a validated handoff
func (s *Store) StoreHandoff(h *Handoff) {
	s.mu.Lock()
	defer s.mu.Unlock()

	h.Timestamp = time.Now().Unix()
	s.handoffs = append(s.handoffs, h)

	// Store findings
	s.findings = append(s.findings, h.Findings...)
}

// Checkpoint operations

// CreateCheckpoint creates a new checkpoint
func (s *Store) CreateCheckpoint() *CheckpointSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	id := fmt.Sprintf("cp-%s-%d", s.currentPhase, len(s.checkpoints))

	// Snapshot current tasks
	var tasksSnapshot []Task
	for _, t := range s.tasks {
		tasksSnapshot = append(tasksSnapshot, *t)
	}

	checkpoint := &Checkpoint{
		ID:               id,
		Phase:            s.currentPhase,
		CreatedAt:        now,
		TasksSnapshot:    tasksSnapshot,
		FindingsSnapshot: append([]Finding{}, s.findings...),
		Decisions:        []string{},
	}

	s.checkpoints = append(s.checkpoints, checkpoint)

	return &CheckpointSummary{
		ID:        id,
		Phase:     s.currentPhase,
		CreatedAt: now,
	}
}

// ListCheckpoints returns all checkpoint summaries
func (s *Store) ListCheckpoints() []CheckpointSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var summaries []CheckpointSummary
	for _, cp := range s.checkpoints {
		summaries = append(summaries, CheckpointSummary{
			ID:        cp.ID,
			Phase:     cp.Phase,
			CreatedAt: cp.CreatedAt,
		})
	}
	return summaries
}

// GetCheckpoint returns a full checkpoint by ID
func (s *Store) GetCheckpoint(id string) (*Checkpoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, cp := range s.checkpoints {
		if cp.ID == id {
			return cp, true
		}
	}
	return nil, false
}

// Budget operations

// CreateBudget creates a token budget for a worker
func (s *Store) CreateBudget(workerID string, budget int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.budgets[workerID] = &TokenBudget{
		WorkerID:  workerID,
		Budget:    budget,
		Used:      0,
		Status:    BudgetStatusHealthy,
		Remaining: budget,
	}
}

// RecordUsage records token usage for a worker
func (s *Store) RecordUsage(workerID string, tokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.budgets[workerID]
	if !ok {
		return
	}

	b.Used += tokens
	b.Remaining = b.Budget - b.Used
	if b.Remaining < 0 {
		b.Remaining = 0
	}

	// Update status based on usage ratio
	ratio := float64(b.Used) / float64(b.Budget)
	if ratio >= 1.0 {
		b.Status = BudgetStatusExceeded
	} else if ratio >= 0.75 {
		b.Status = BudgetStatusCritical
	} else if ratio >= 0.5 {
		b.Status = BudgetStatusWarning
	} else {
		b.Status = BudgetStatusHealthy
	}
}

// GetBudget returns a worker's budget
func (s *Store) GetBudget(workerID string) (*TokenBudget, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.budgets[workerID]
	return b, ok
}
