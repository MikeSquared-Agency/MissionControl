package v4

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mike/mission-control/hashid"
)

// Store manages v4 workflow state
// This is an in-memory implementation that will later be backed by Rust FFI
type Store struct {
	mu sync.RWMutex

	currentStage Stage
	tasks        map[string]*Task
	gates        map[Stage]*Gate
	checkpoints  []*Checkpoint
	handoffs     []*Handoff
	findings     []Finding
	budgets      map[string]*TokenBudget
	sessions     []SessionRecord
	sessionID    string
}

// NewStore creates a new v4 store
func NewStore() *Store {
	sessionID := uuid.New().String()[:8]
	s := &Store{
		currentStage: StageDiscovery,
		tasks:        make(map[string]*Task),
		gates:        make(map[Stage]*Gate),
		checkpoints:  make([]*Checkpoint, 0),
		handoffs:     make([]*Handoff, 0),
		findings:     make([]Finding, 0),
		budgets:      make(map[string]*TokenBudget),
		sessions:     make([]SessionRecord, 0),
		sessionID:    sessionID,
	}
	// Record initial session
	s.sessions = append(s.sessions, SessionRecord{
		SessionID: sessionID,
		StartedAt: time.Now().Unix(),
		Stage:     StageDiscovery,
	})

	// Initialize gates for all stages
	for _, stage := range AllStages() {
		s.gates[stage] = s.createGateForStage(stage)
	}

	return s
}

func (s *Store) createGateForStage(stage Stage) *Gate {
	criteria := s.defaultCriteriaForStage(stage)
	return &Gate{
		ID:       fmt.Sprintf("gate-%s", stage),
		Stage:    stage,
		Status:   GateStatusClosed,
		Criteria: criteria,
	}
}

func (s *Store) defaultCriteriaForStage(stage Stage) []GateCriterion {
	switch stage {
	case StageDiscovery:
		return []GateCriterion{
			{Description: "Problem space explored", Satisfied: false},
			{Description: "Stakeholders identified", Satisfied: false},
		}
	case StageGoal:
		return []GateCriterion{
			{Description: "Goal statement defined", Satisfied: false},
			{Description: "Success metrics established", Satisfied: false},
		}
	case StageRequirements:
		return []GateCriterion{
			{Description: "Requirements documented", Satisfied: false},
			{Description: "Acceptance criteria defined", Satisfied: false},
		}
	case StagePlanning:
		return []GateCriterion{
			{Description: "Tasks broken down", Satisfied: false},
			{Description: "Dependencies mapped", Satisfied: false},
		}
	case StageDesign:
		return []GateCriterion{
			{Description: "Spec document complete", Satisfied: false},
			{Description: "Technical approach approved", Satisfied: false},
		}
	case StageImplement:
		return []GateCriterion{
			{Description: "All tasks complete", Satisfied: false},
			{Description: "Code compiles", Satisfied: false},
		}
	case StageVerify:
		return []GateCriterion{
			{Description: "Tests passing", Satisfied: false},
			{Description: "Review complete", Satisfied: false},
		}
	case StageValidate:
		return []GateCriterion{
			{Description: "Acceptance criteria met", Satisfied: false},
			{Description: "Stakeholder sign-off", Satisfied: false},
		}
	case StageDocument:
		return []GateCriterion{
			{Description: "README updated", Satisfied: false},
			{Description: "API documented", Satisfied: false},
		}
	case StageRelease:
		return []GateCriterion{
			{Description: "Deployed successfully", Satisfied: false},
			{Description: "Smoke tests pass", Satisfied: false},
		}
	default:
		return []GateCriterion{}
	}
}

// Stage operations

// CurrentStage returns the current stage
func (s *Store) CurrentStage() Stage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentStage
}

// GetStages returns all stage info
func (s *Store) GetStages() []StageInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var infos []StageInfo
	for _, stage := range AllStages() {
		var status string
		if stage == s.currentStage {
			status = "current"
		} else if s.isStageComplete(stage) {
			status = "complete"
		} else {
			status = "pending"
		}
		infos = append(infos, StageInfo{Stage: stage, Status: status})
	}
	return infos
}

func (s *Store) isStageComplete(stage Stage) bool {
	gate, ok := s.gates[stage]
	if !ok {
		return false
	}
	return gate.Status == GateStatusOpen
}

// CanTransition checks if we can transition to the given stage
func (s *Store) CanTransition(to Stage) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	next := s.currentStage.Next()
	if next != to {
		return false
	}

	gate, ok := s.gates[s.currentStage]
	if !ok {
		return false
	}

	return gate.Status == GateStatusOpen
}

// Transition moves to the next stage
func (s *Store) Transition(to Stage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.currentStage.Next()
	if next != to {
		return fmt.Errorf("cannot transition from %s to %s", s.currentStage, to)
	}

	gate, ok := s.gates[s.currentStage]
	if !ok || gate.Status != GateStatusOpen {
		return fmt.Errorf("gate not open for stage %s", s.currentStage)
	}

	s.currentStage = to
	return nil
}

// Task operations

// CreateTask creates a new task
func (s *Store) CreateTask(name string, stage Stage, zone, persona string, deps []string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	taskID := hashid.Generate("task", name, string(stage), zone, persona)

	// Return existing task if duplicate
	if existing, ok := s.tasks[taskID]; ok {
		return existing
	}

	task := &Task{
		ID:           taskID,
		Name:         name,
		Stage:        stage,
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
func (s *Store) ListTasks(stage *Stage, zone, status, persona *string) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Task
	for _, task := range s.tasks {
		if stage != nil && task.Stage != *stage {
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

// GetGate returns a gate for a stage
func (s *Store) GetGate(stage Stage) (*Gate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	gate, ok := s.gates[stage]
	return gate, ok
}

// SatisfyCriterion marks a gate criterion as satisfied
func (s *Store) SatisfyCriterion(stage Stage, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	gate, ok := s.gates[stage]
	if !ok {
		return fmt.Errorf("gate not found for stage: %s", stage)
	}

	if index < 0 || index >= len(gate.Criteria) {
		return fmt.Errorf("invalid criterion index: %d", index)
	}

	gate.Criteria[index].Satisfied = true
	s.updateGateStatus(gate)
	return nil
}

// ApproveGate approves a gate
func (s *Store) ApproveGate(stage Stage, approvedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	gate, ok := s.gates[stage]
	if !ok {
		return fmt.Errorf("gate not found for stage: %s", stage)
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
	id := fmt.Sprintf("cp-%s-%d", s.currentStage, len(s.checkpoints))

	// Snapshot current tasks
	var tasksSnapshot []Task
	for _, t := range s.tasks {
		tasksSnapshot = append(tasksSnapshot, *t)
	}

	checkpoint := &Checkpoint{
		ID:               id,
		Stage:            s.currentStage,
		CreatedAt:        now,
		TasksSnapshot:    tasksSnapshot,
		FindingsSnapshot: append([]Finding{}, s.findings...),
		Decisions:        []string{},
	}

	s.checkpoints = append(s.checkpoints, checkpoint)

	return &CheckpointSummary{
		ID:        id,
		Stage:     s.currentStage,
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
			Stage:     cp.Stage,
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

// Session operations

// GetSessionStatus returns current session health
func (s *Store) GetSessionStatus() CheckpointStatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()

	// Find current session start
	var sessionStart int64
	for i := len(s.sessions) - 1; i >= 0; i-- {
		if s.sessions[i].EndedAt == 0 {
			sessionStart = s.sessions[i].StartedAt
			break
		}
	}
	if sessionStart == 0 {
		sessionStart = now
	}

	durationMin := int((now - sessionStart) / 60)

	// Count tasks
	total := len(s.tasks)
	complete := 0
	for _, t := range s.tasks {
		if t.Status == TaskStatusDone {
			complete++
		}
	}

	// Last checkpoint
	lastCP := ""
	if len(s.checkpoints) > 0 {
		lastCP = s.checkpoints[len(s.checkpoints)-1].ID
	}

	// Health assessment
	health := "green"
	recommendation := "Session is healthy"
	if durationMin > 120 {
		health = "red"
		recommendation = "Session is long. Consider restarting to preserve context."
	} else if durationMin > 60 {
		health = "yellow"
		recommendation = "Session approaching limit. Consider checkpointing soon."
	}

	return CheckpointStatusResponse{
		SessionID:      s.sessionID,
		Stage:          s.currentStage,
		SessionStart:   sessionStart,
		DurationMin:    durationMin,
		LastCheckpoint: lastCP,
		TasksTotal:     total,
		TasksComplete:  complete,
		Health:         health,
		Recommendation: recommendation,
	}
}

// GetSessionHistory returns all session records
func (s *Store) GetSessionHistory() []SessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]SessionRecord, len(s.sessions))
	copy(result, s.sessions)
	return result
}

// RestartSession creates a checkpoint, ends current session, starts new one
func (s *Store) RestartSession(fromCheckpointID string) CheckpointRestartResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	oldSessionID := s.sessionID

	// Create checkpoint
	cpID := fmt.Sprintf("cp-%s-%d", s.currentStage, len(s.checkpoints))
	var tasksSnapshot []Task
	for _, t := range s.tasks {
		tasksSnapshot = append(tasksSnapshot, *t)
	}

	checkpoint := &Checkpoint{
		ID:               cpID,
		Stage:            s.currentStage,
		SessionID:        oldSessionID,
		CreatedAt:        now,
		TasksSnapshot:    tasksSnapshot,
		FindingsSnapshot: append([]Finding{}, s.findings...),
		Decisions:        []string{},
	}
	s.checkpoints = append(s.checkpoints, checkpoint)

	// End current session
	for i := len(s.sessions) - 1; i >= 0; i-- {
		if s.sessions[i].SessionID == oldSessionID && s.sessions[i].EndedAt == 0 {
			s.sessions[i].EndedAt = now
			s.sessions[i].CheckpointID = cpID
			s.sessions[i].Reason = "restart"
			break
		}
	}

	// Start new session
	newSessionID := uuid.New().String()[:8]
	s.sessionID = newSessionID
	s.sessions = append(s.sessions, SessionRecord{
		SessionID:    newSessionID,
		StartedAt:    now,
		CheckpointID: cpID,
		Stage:        s.currentStage,
	})

	// Generate briefing
	briefing := s.generateBriefing(checkpoint)

	return CheckpointRestartResponse{
		OldSessionID: oldSessionID,
		NewSessionID: newSessionID,
		CheckpointID: cpID,
		Stage:        s.currentStage,
		Briefing:     briefing,
	}
}

func (s *Store) generateBriefing(cp *Checkpoint) string {
	total := len(cp.TasksSnapshot)
	done := 0
	pending := 0
	for _, t := range cp.TasksSnapshot {
		switch t.Status {
		case TaskStatusDone:
			done++
		case TaskStatusPending:
			pending++
		}
	}

	return fmt.Sprintf("# Session Briefing\n\n**Stage:** %s\n\n## Tasks\n- Total: %d, Done: %d, Pending: %d\n",
		cp.Stage, total, done, pending)
}

