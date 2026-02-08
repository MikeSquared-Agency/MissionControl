package v4

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler holds v4 API dependencies
type Handler struct {
	Store    *Store
	Notifier EventNotifier
}

// EventNotifier interface for WebSocket notifications
type EventNotifier interface {
	Notify(event interface{})
}

// NewHandler creates a new v4 API handler
func NewHandler(store *Store, notifier EventNotifier) *Handler {
	return &Handler{
		Store:    store,
		Notifier: notifier,
	}
}

// RegisterRoutes registers v4 routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Workflow routes
	mux.HandleFunc("/api/stages", h.handleStages)
	mux.HandleFunc("/api/tasks", h.handleTasks)
	mux.HandleFunc("/api/tasks/", h.handleTask)

	// Knowledge routes
	mux.HandleFunc("/api/handoffs", h.handleHandoffs)
	mux.HandleFunc("/api/checkpoints", h.handleCheckpoints)
	mux.HandleFunc("/api/checkpoints/", h.handleCheckpoint)
	mux.HandleFunc("/api/budgets/", h.handleBudget)

	// Session/checkpoint management routes
	mux.HandleFunc("/api/checkpoint/status", h.handleCheckpointStatus)
	mux.HandleFunc("/api/checkpoint/history", h.handleCheckpointHistory)
	mux.HandleFunc("/api/checkpoint/restart", h.handleCheckpointRestart)

	// Strategy routes
	mux.HandleFunc("/api/gates/", h.handleGate)
}

// ============================================================================
// Workflow Domain
// ============================================================================

// StagesResponse is the response for GET /api/stages
type StagesResponse struct {
	Current Stage       `json:"current"`
	Stages  []StageInfo `json:"stages"`
}

func (h *Handler) handleStages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := StagesResponse{
		Current: h.Store.CurrentStage(),
		Stages:  h.Store.GetStages(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// CreateTaskRequest is the request for POST /api/tasks
type CreateTaskRequest struct {
	Name         string   `json:"name"`
	Stage        Stage    `json:"stage"`
	Zone         string   `json:"zone"`
	Persona      string   `json:"persona"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// TasksResponse is the response for GET /api/tasks
type TasksResponse struct {
	Tasks []Task `json:"tasks"`
}

func (h *Handler) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listTasks(w, r)
	case "POST":
		h.createTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var stage *Stage
	if s := query.Get("stage"); s != "" {
		st := Stage(s)
		stage = &st
	}

	var zone *string
	if z := query.Get("zone"); z != "" {
		zone = &z
	}

	var status *string
	if s := query.Get("status"); s != "" {
		status = &s
	}

	var persona *string
	if p := query.Get("persona"); p != "" {
		persona = &p
	}

	tasks := h.Store.ListTasks(stage, zone, status, persona)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(TasksResponse{Tasks: tasks})
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Stage == "" {
		req.Stage = h.Store.CurrentStage()
	}

	task := h.Store.CreateTask(req.Name, req.Stage, req.Zone, req.Persona, req.Dependencies)

	// Notify via WebSocket
	if h.Notifier != nil {
		h.Notifier.Notify(map[string]interface{}{
			"type": "task_created",
			"task": task,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(task)
}

// UpdateTaskStatusRequest is the request for PUT /api/tasks/:id/status
type UpdateTaskStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (h *Handler) handleTask(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	id := parts[0]

	// Check for /status subpath
	if len(parts) > 1 && parts[1] == "status" {
		if r.Method == "PUT" {
			h.updateTaskStatus(w, r, id)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// GET /api/tasks/:id
	if r.Method == "GET" {
		h.getTask(w, r, id)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.Store.GetTask(id)
	if !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

func (h *Handler) updateTaskStatus(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateTaskStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	status := TaskStatus(req.Status)
	if status == TaskStatusBlocked && req.Reason == "" {
		http.Error(w, "reason is required when status is blocked", http.StatusBadRequest)
		return
	}

	oldTask, ok := h.Store.GetTask(id)
	if !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	previousStatus := oldTask.Status

	if err := h.Store.UpdateTaskStatus(id, status, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task, _ := h.Store.GetTask(id)

	// Notify via WebSocket
	if h.Notifier != nil {
		h.Notifier.Notify(map[string]interface{}{
			"type":     "task_updated",
			"task_id":  id,
			"status":   status,
			"previous": previousStatus,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

// ============================================================================
// Knowledge Domain
// ============================================================================

// HandoffRequest is the request for POST /api/handoffs
type HandoffRequest struct {
	TaskID              string            `json:"task_id"`
	WorkerID            string            `json:"worker_id"`
	Status              HandoffStatus     `json:"status"`
	Findings            []Finding         `json:"findings"`
	Artifacts           []string          `json:"artifacts"`
	OpenQuestions       []string          `json:"open_questions,omitempty"`
	ContextForSuccessor *SuccessorContext `json:"context_for_successor,omitempty"`
	BlockedReason       string            `json:"blocked_reason,omitempty"`
}

// HandoffResponse is the response for POST /api/handoffs
type HandoffResponse struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	DeltaID string   `json:"delta_id,omitempty"`
}

func (h *Handler) handleHandoffs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HandoffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	handoff := &Handoff{
		TaskID:              req.TaskID,
		WorkerID:            req.WorkerID,
		Status:              req.Status,
		Findings:            req.Findings,
		Artifacts:           req.Artifacts,
		OpenQuestions:       req.OpenQuestions,
		ContextForSuccessor: req.ContextForSuccessor,
		BlockedReason:       req.BlockedReason,
	}

	errors := h.Store.ValidateHandoff(handoff)

	if len(errors) > 0 {
		// Notify validation failure
		if h.Notifier != nil {
			h.Notifier.Notify(map[string]interface{}{
				"type":    "handoff_validated",
				"task_id": req.TaskID,
				"valid":   false,
				"errors":  errors,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HandoffResponse{
			Valid:  false,
			Errors: errors,
		})
		return
	}

	h.Store.StoreHandoff(handoff)

	// Notify success
	if h.Notifier != nil {
		h.Notifier.Notify(map[string]interface{}{
			"type":      "handoff_received",
			"task_id":   req.TaskID,
			"worker_id": req.WorkerID,
			"status":    req.Status,
		})
		h.Notifier.Notify(map[string]interface{}{
			"type":    "handoff_validated",
			"task_id": req.TaskID,
			"valid":   true,
			"errors":  []string{},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HandoffResponse{
		Valid:   true,
		DeltaID: "", // TODO: implement delta computation
	})
}

// CheckpointsResponse is the response for GET /api/checkpoints
type CheckpointsResponse struct {
	Checkpoints []CheckpointSummary `json:"checkpoints"`
}

func (h *Handler) handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		checkpoints := h.Store.ListCheckpoints()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CheckpointsResponse{Checkpoints: checkpoints})

	case "POST":
		summary := h.Store.CreateCheckpoint()

		// Notify
		if h.Notifier != nil {
			h.Notifier.Notify(map[string]interface{}{
				"type":          "checkpoint_created",
				"checkpoint_id": summary.ID,
				"stage":         summary.Stage,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(summary)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/checkpoints/")
	if id == "" {
		http.Error(w, "Checkpoint ID required", http.StatusBadRequest)
		return
	}

	checkpoint, ok := h.Store.GetCheckpoint(id)
	if !ok {
		http.Error(w, "Checkpoint not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(checkpoint)
}

func (h *Handler) handleBudget(w http.ResponseWriter, r *http.Request) {
	workerID := strings.TrimPrefix(r.URL.Path, "/api/budgets/")
	if workerID == "" {
		http.Error(w, "Worker ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		budget, ok := h.Store.GetBudget(workerID)
		if !ok {
			http.Error(w, "Budget not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(budget)

	case "POST":
		var req struct {
			Budget int `json:"budget"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		h.Store.CreateBudget(workerID, req.Budget)
		budget, _ := h.Store.GetBudget(workerID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(budget)

	case "PUT":
		var req struct {
			Tokens int `json:"tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		h.Store.RecordUsage(workerID, req.Tokens)
		budget, ok := h.Store.GetBudget(workerID)
		if !ok {
			http.Error(w, "Budget not found", http.StatusNotFound)
			return
		}

		// Notify on warning/critical thresholds
		if h.Notifier != nil && (budget.Status == BudgetStatusWarning || budget.Status == BudgetStatusCritical) {
			eventType := "token_warning"
			if budget.Status == BudgetStatusCritical {
				eventType = "token_critical"
			}
			h.Notifier.Notify(map[string]interface{}{
				"type":      eventType,
				"worker_id": workerID,
				"usage":     budget.Used,
				"budget":    budget.Budget,
				"status":    budget.Status,
				"remaining": budget.Remaining,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(budget)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================================================
// Strategy Domain
// ============================================================================

// GateApprovalRequest is the request for POST /api/gates/:id/approve
type GateApprovalRequest struct {
	ApprovedBy string `json:"approved_by"`
	Comment    string `json:"comment,omitempty"`
}

// GateApprovalResponse is the response for POST /api/gates/:id/approve
type GateApprovalResponse struct {
	Gate       *Gate `json:"gate"`
	CanProceed bool  `json:"can_proceed"`
}

func (h *Handler) handleGate(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/gates/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Gate ID required", http.StatusBadRequest)
		return
	}

	// Gate ID is "gate-{stage}", extract stage
	gateID := parts[0]
	stage := Stage(strings.TrimPrefix(gateID, "gate-"))

	// Check for /approve subpath
	if len(parts) > 1 && parts[1] == "approve" {
		if r.Method == "POST" {
			h.approveGate(w, r, stage)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// GET /api/gates/:id
	if r.Method == "GET" {
		gate, ok := h.Store.GetGate(stage)
		if !ok {
			http.Error(w, "Gate not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gate)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) approveGate(w http.ResponseWriter, r *http.Request, stage Stage) {
	var req GateApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ApprovedBy == "" {
		req.ApprovedBy = "user"
	}

	if _, ok := h.Store.GetGate(stage); !ok {
		http.Error(w, "Gate not found", http.StatusNotFound)
		return
	}

	// Gate approval is allowed for awaiting_approval and closed (manual override) states.

	if err := h.Store.ApproveGate(stage, req.ApprovedBy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gate, _ := h.Store.GetGate(stage)
	canProceed := h.Store.CanTransition(stage.Next())

	// Notify gate approval
	if h.Notifier != nil {
		h.Notifier.Notify(map[string]interface{}{
			"type":   "gate_status",
			"stage":  stage,
			"status": gate.Status,
		})
	}

	// Auto-create checkpoint on gate approval (G3.1)
	cpSummary := h.Store.CreateCheckpoint()
	if h.Notifier != nil {
		h.Notifier.Notify(map[string]interface{}{
			"type":          "checkpoint_created",
			"checkpoint_id": cpSummary.ID,
			"stage":         cpSummary.Stage,
			"trigger":       "gate_approval",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(GateApprovalResponse{
		Gate:       gate,
		CanProceed: canProceed,
	})
}

// ============================================================================
// Session / Checkpoint Management
// ============================================================================

// CheckpointStatusResponse is the response for GET /api/checkpoint/status
type CheckpointStatusResponse struct {
	SessionID      string `json:"session_id"`
	Stage          Stage  `json:"stage"`
	SessionStart   int64  `json:"session_start"`
	DurationMin    int    `json:"duration_minutes"`
	LastCheckpoint string `json:"last_checkpoint,omitempty"`
	TasksTotal     int    `json:"tasks_total"`
	TasksComplete  int    `json:"tasks_complete"`
	Health         string `json:"health"`
	Recommendation string `json:"recommendation"`
}

func (h *Handler) handleCheckpointStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.Store.GetSessionStatus()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (h *Handler) handleCheckpointHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions := h.Store.GetSessionHistory()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
	})
}

// CheckpointRestartRequest is the request for POST /api/checkpoint/restart
type CheckpointRestartRequest struct {
	FromCheckpointID string `json:"from_checkpoint_id,omitempty"`
}

// CheckpointRestartResponse is the response for POST /api/checkpoint/restart
type CheckpointRestartResponse struct {
	OldSessionID string `json:"old_session_id"`
	NewSessionID string `json:"new_session_id"`
	CheckpointID string `json:"checkpoint_id"`
	Stage        Stage  `json:"stage"`
	Briefing     string `json:"briefing"`
}

func (h *Handler) handleCheckpointRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckpointRestartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	result := h.Store.RestartSession(req.FromCheckpointID)

	// Notify via WebSocket
	if h.Notifier != nil {
		h.Notifier.Notify(map[string]interface{}{
			"type":        "session_restarted",
			"old_session": result.OldSessionID,
			"new_session": result.NewSessionID,
			"checkpoint":  result.CheckpointID,
			"stage":       result.Stage,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
