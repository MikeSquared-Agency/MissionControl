package v4

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockNotifier implements EventNotifier for testing
type mockNotifier struct {
	events []interface{}
}

func (m *mockNotifier) Notify(event interface{}) {
	m.events = append(m.events, event)
}

func newTestHandler() (*Handler, *mockNotifier) {
	store := NewStore()
	notifier := &mockNotifier{events: make([]interface{}, 0)}
	handler := NewHandler(store, notifier)
	return handler, notifier
}

func newTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// ============================================================================
// Phase 5 Integration Tests
// ============================================================================

// Test 1: Create task via API → verify response
func TestCreateTaskViaAPI(t *testing.T) {
	h, notifier := newTestHandler()
	mux := newTestMux(h)

	// Create a task
	body := bytes.NewBufferString(`{
		"name": "Build login page",
		"phase": "implement",
		"zone": "frontend",
		"persona": "developer"
	}`)
	req := httptest.NewRequest("POST", "/api/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var task Task
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify task fields
	if task.Name != "Build login page" {
		t.Errorf("Expected name 'Build login page', got %s", task.Name)
	}
	if task.Phase != "implement" {
		t.Errorf("Expected phase 'implement', got %s", task.Phase)
	}
	if task.Zone != "frontend" {
		t.Errorf("Expected zone 'frontend', got %s", task.Zone)
	}
	if task.Persona != "developer" {
		t.Errorf("Expected persona 'developer', got %s", task.Persona)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("Expected status 'pending', got %s", task.Status)
	}
	if task.ID == "" {
		t.Error("Task ID should not be empty")
	}

	// Verify WebSocket event was emitted
	if len(notifier.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(notifier.events))
	}
}

// Test 2: Update task status → verify WebSocket event
func TestUpdateTaskStatusEmitsEvent(t *testing.T) {
	h, notifier := newTestHandler()
	mux := newTestMux(h)

	// First create a task
	createBody := bytes.NewBufferString(`{
		"name": "Test task",
		"phase": "idea",
		"zone": "system",
		"persona": "researcher"
	}`)
	createReq := httptest.NewRequest("POST", "/api/tasks", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var created Task
	json.Unmarshal(createW.Body.Bytes(), &created)

	// Clear events from creation
	notifier.events = nil

	// Update status to in_progress
	updateBody := bytes.NewBufferString(`{"status": "in_progress"}`)
	updateReq := httptest.NewRequest("PUT", "/api/tasks/"+created.ID+"/status", updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	mux.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	// Verify event was emitted
	if len(notifier.events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(notifier.events))
	}

	event, ok := notifier.events[0].(map[string]interface{})
	if !ok {
		t.Fatal("Event is not a map")
	}

	if event["type"] != "task_updated" {
		t.Errorf("Expected event type 'task_updated', got %v", event["type"])
	}
	if event["task_id"] != created.ID {
		t.Errorf("Expected task_id %s, got %v", created.ID, event["task_id"])
	}
	if event["status"] != TaskStatusInProgress {
		t.Errorf("Expected status 'in_progress', got %v", event["status"])
	}
}

// Test 3: Phase transition → verify gate status
func TestPhaseTransitionRequiresGateApproval(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Get current phase - should be 'idea'
	phasesReq := httptest.NewRequest("GET", "/api/phases", nil)
	phasesW := httptest.NewRecorder()
	mux.ServeHTTP(phasesW, phasesReq)

	var phasesResp PhasesResponse
	json.Unmarshal(phasesW.Body.Bytes(), &phasesResp)

	if phasesResp.Current != PhaseIdea {
		t.Errorf("Expected current phase 'idea', got %s", phasesResp.Current)
	}

	// Check gate status - should be closed
	gateReq := httptest.NewRequest("GET", "/api/gates/gate-idea", nil)
	gateW := httptest.NewRecorder()
	mux.ServeHTTP(gateW, gateReq)

	var gate Gate
	json.Unmarshal(gateW.Body.Bytes(), &gate)

	if gate.Status != GateStatusClosed {
		t.Errorf("Expected gate status 'closed', got %s", gate.Status)
	}

	// Approve the gate
	approveBody := bytes.NewBufferString(`{"approved_by": "user"}`)
	approveReq := httptest.NewRequest("POST", "/api/gates/gate-idea/approve", approveBody)
	approveReq.Header.Set("Content-Type", "application/json")
	approveW := httptest.NewRecorder()
	mux.ServeHTTP(approveW, approveReq)

	if approveW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", approveW.Code, approveW.Body.String())
	}

	var approvalResp GateApprovalResponse
	json.Unmarshal(approveW.Body.Bytes(), &approvalResp)

	if approvalResp.Gate.Status != GateStatusOpen {
		t.Errorf("Expected approved gate status 'open', got %s", approvalResp.Gate.Status)
	}
	if approvalResp.Gate.ApprovedBy != "user" {
		t.Errorf("Expected approved_by 'user', got %s", approvalResp.Gate.ApprovedBy)
	}
}

// Test 4: Token warning flow end-to-end
func TestTokenWarningFlow(t *testing.T) {
	h, notifier := newTestHandler()
	mux := newTestMux(h)

	workerID := "worker-test-1"

	// Create a budget
	createBody := bytes.NewBufferString(`{"budget": 20000}`)
	createReq := httptest.NewRequest("POST", "/api/budgets/"+workerID, createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", createW.Code, createW.Body.String())
	}

	var budget TokenBudget
	json.Unmarshal(createW.Body.Bytes(), &budget)

	if budget.Status != BudgetStatusHealthy {
		t.Errorf("Expected status 'healthy', got %s", budget.Status)
	}

	// Clear events
	notifier.events = nil

	// Record usage to trigger warning (60% = 12000 tokens)
	usageBody := bytes.NewBufferString(`{"tokens": 12000}`)
	usageReq := httptest.NewRequest("PUT", "/api/budgets/"+workerID, usageBody)
	usageReq.Header.Set("Content-Type", "application/json")
	usageW := httptest.NewRecorder()
	mux.ServeHTTP(usageW, usageReq)

	if usageW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", usageW.Code, usageW.Body.String())
	}

	json.Unmarshal(usageW.Body.Bytes(), &budget)

	if budget.Status != BudgetStatusWarning {
		t.Errorf("Expected status 'warning' at 60%%, got %s", budget.Status)
	}
	if budget.Used != 12000 {
		t.Errorf("Expected used 12000, got %d", budget.Used)
	}
	if budget.Remaining != 8000 {
		t.Errorf("Expected remaining 8000, got %d", budget.Remaining)
	}

	// Verify warning event was emitted
	if len(notifier.events) != 1 {
		t.Fatalf("Expected 1 warning event, got %d", len(notifier.events))
	}

	event, ok := notifier.events[0].(map[string]interface{})
	if !ok {
		t.Fatal("Event is not a map")
	}

	if event["type"] != "token_warning" {
		t.Errorf("Expected event type 'token_warning', got %v", event["type"])
	}

	// Clear events and push to critical
	notifier.events = nil

	criticalBody := bytes.NewBufferString(`{"tokens": 4000}`)
	criticalReq := httptest.NewRequest("PUT", "/api/budgets/"+workerID, criticalBody)
	criticalReq.Header.Set("Content-Type", "application/json")
	criticalW := httptest.NewRecorder()
	mux.ServeHTTP(criticalW, criticalReq)

	json.Unmarshal(criticalW.Body.Bytes(), &budget)

	if budget.Status != BudgetStatusCritical {
		t.Errorf("Expected status 'critical' at 80%%, got %s", budget.Status)
	}

	// Verify critical event
	if len(notifier.events) != 1 {
		t.Fatalf("Expected 1 critical event, got %d", len(notifier.events))
	}

	event, _ = notifier.events[0].(map[string]interface{})
	if event["type"] != "token_critical" {
		t.Errorf("Expected event type 'token_critical', got %v", event["type"])
	}
}

// Test 5: Handoff validation flow
func TestHandoffValidationFlow(t *testing.T) {
	h, notifier := newTestHandler()
	mux := newTestMux(h)

	// First create a task to handoff
	taskBody := bytes.NewBufferString(`{
		"name": "Research task",
		"phase": "idea",
		"zone": "research",
		"persona": "researcher"
	}`)
	taskReq := httptest.NewRequest("POST", "/api/tasks", taskBody)
	taskReq.Header.Set("Content-Type", "application/json")
	taskW := httptest.NewRecorder()
	mux.ServeHTTP(taskW, taskReq)

	var task Task
	json.Unmarshal(taskW.Body.Bytes(), &task)

	// Clear events
	notifier.events = nil

	// Submit a valid handoff
	handoffBody := bytes.NewBufferString(`{
		"task_id": "` + task.ID + `",
		"worker_id": "worker-1",
		"status": "complete",
		"findings": [
			{"type": "discovery", "summary": "Found existing auth implementation"},
			{"type": "decision", "summary": "Recommend using JWT"}
		],
		"artifacts": ["docs/auth-research.md"]
	}`)
	handoffReq := httptest.NewRequest("POST", "/api/handoffs", handoffBody)
	handoffReq.Header.Set("Content-Type", "application/json")
	handoffW := httptest.NewRecorder()
	mux.ServeHTTP(handoffW, handoffReq)

	if handoffW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", handoffW.Code, handoffW.Body.String())
	}

	var resp HandoffResponse
	json.Unmarshal(handoffW.Body.Bytes(), &resp)

	if !resp.Valid {
		t.Errorf("Expected valid handoff, got errors: %v", resp.Errors)
	}

	// Verify events were emitted
	if len(notifier.events) < 2 {
		t.Fatalf("Expected at least 2 events (received + validated), got %d", len(notifier.events))
	}

	// Test invalid handoff (missing task_id)
	invalidBody := bytes.NewBufferString(`{
		"worker_id": "worker-1",
		"status": "complete",
		"findings": []
	}`)
	invalidReq := httptest.NewRequest("POST", "/api/handoffs", invalidBody)
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidW := httptest.NewRecorder()
	mux.ServeHTTP(invalidW, invalidReq)

	var invalidResp HandoffResponse
	json.Unmarshal(invalidW.Body.Bytes(), &invalidResp)

	if invalidResp.Valid {
		t.Error("Expected invalid handoff due to missing task_id")
	}
	if len(invalidResp.Errors) == 0 {
		t.Error("Expected validation errors")
	}
}

// Test 6: Handoff blocked status requires reason
func TestHandoffBlockedRequiresReason(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Submit blocked handoff without reason
	handoffBody := bytes.NewBufferString(`{
		"task_id": "task-1",
		"worker_id": "worker-1",
		"status": "blocked",
		"findings": []
	}`)
	handoffReq := httptest.NewRequest("POST", "/api/handoffs", handoffBody)
	handoffReq.Header.Set("Content-Type", "application/json")
	handoffW := httptest.NewRecorder()
	mux.ServeHTTP(handoffW, handoffReq)

	var resp HandoffResponse
	json.Unmarshal(handoffW.Body.Bytes(), &resp)

	if resp.Valid {
		t.Error("Expected invalid handoff due to missing blocked_reason")
	}

	// Now with reason - should be valid
	validBody := bytes.NewBufferString(`{
		"task_id": "task-1",
		"worker_id": "worker-1",
		"status": "blocked",
		"blocked_reason": "Waiting for API documentation",
		"findings": []
	}`)
	validReq := httptest.NewRequest("POST", "/api/handoffs", validBody)
	validReq.Header.Set("Content-Type", "application/json")
	validW := httptest.NewRecorder()
	mux.ServeHTTP(validW, validReq)

	json.Unmarshal(validW.Body.Bytes(), &resp)

	if !resp.Valid {
		t.Errorf("Expected valid handoff with blocked_reason, got errors: %v", resp.Errors)
	}
}

// ============================================================================
// Additional API Tests
// ============================================================================

func TestGetPhasesEndpoint(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	req := httptest.NewRequest("GET", "/api/phases", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp PhasesResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Current != PhaseIdea {
		t.Errorf("Expected current phase 'idea', got %s", resp.Current)
	}
	if len(resp.Phases) != 6 {
		t.Errorf("Expected 6 phases, got %d", len(resp.Phases))
	}
}

func TestListTasksEndpoint(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Create some tasks
	for _, task := range []string{"Task 1", "Task 2"} {
		body := bytes.NewBufferString(`{"name":"` + task + `","zone":"test","persona":"dev"}`)
		req := httptest.NewRequest("POST", "/api/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	// List tasks
	req := httptest.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp TasksResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(resp.Tasks))
	}
}

func TestListTasksWithFilter(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Create tasks in different phases
	phases := []string{"idea", "design"}
	for i, phase := range phases {
		body := bytes.NewBufferString(`{"name":"Task ` + string(rune('A'+i)) + `","phase":"` + phase + `","zone":"test","persona":"dev"}`)
		req := httptest.NewRequest("POST", "/api/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	// Filter by phase
	req := httptest.NewRequest("GET", "/api/tasks?phase=idea", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp TasksResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Tasks) != 1 {
		t.Errorf("Expected 1 task in idea phase, got %d", len(resp.Tasks))
	}
	if resp.Tasks[0].Phase != PhaseIdea {
		t.Errorf("Expected phase 'idea', got %s", resp.Tasks[0].Phase)
	}
}

func TestGetTaskEndpoint(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Create a task
	createBody := bytes.NewBufferString(`{"name":"Test Task","zone":"test","persona":"dev"}`)
	createReq := httptest.NewRequest("POST", "/api/tasks", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var created Task
	json.Unmarshal(createW.Body.Bytes(), &created)

	// Get the task
	getReq := httptest.NewRequest("GET", "/api/tasks/"+created.ID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", getW.Code)
	}

	var task Task
	json.Unmarshal(getW.Body.Bytes(), &task)

	if task.ID != created.ID {
		t.Errorf("Expected task ID %s, got %s", created.ID, task.ID)
	}
}

func TestGetNonExistentTask(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	req := httptest.NewRequest("GET", "/api/tasks/non-existent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestCheckpointCreation(t *testing.T) {
	h, notifier := newTestHandler()
	mux := newTestMux(h)

	// Create a checkpoint
	req := httptest.NewRequest("POST", "/api/checkpoints", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var summary CheckpointSummary
	json.Unmarshal(w.Body.Bytes(), &summary)

	if summary.ID == "" {
		t.Error("Checkpoint ID should not be empty")
	}
	if summary.Phase != PhaseIdea {
		t.Errorf("Expected phase 'idea', got %s", summary.Phase)
	}

	// Verify event was emitted
	found := false
	for _, e := range notifier.events {
		if event, ok := e.(map[string]interface{}); ok {
			if event["type"] == "checkpoint_created" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected checkpoint_created event")
	}
}

func TestListCheckpoints(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Create a checkpoint first
	createReq := httptest.NewRequest("POST", "/api/checkpoints", nil)
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	// List checkpoints
	listReq := httptest.NewRequest("GET", "/api/checkpoints", nil)
	listW := httptest.NewRecorder()
	mux.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", listW.Code)
	}

	var resp CheckpointsResponse
	json.Unmarshal(listW.Body.Bytes(), &resp)

	if len(resp.Checkpoints) < 1 {
		t.Error("Expected at least 1 checkpoint")
	}
}

func TestFindingSummaryTooLong(t *testing.T) {
	h, _ := newTestHandler()
	mux := newTestMux(h)

	// Create a finding with summary > 500 chars
	longSummary := make([]byte, 501)
	for i := range longSummary {
		longSummary[i] = 'x'
	}

	handoffBody := bytes.NewBufferString(`{
		"task_id": "task-1",
		"worker_id": "worker-1",
		"status": "complete",
		"findings": [{"type": "discovery", "summary": "` + string(longSummary) + `"}]
	}`)
	req := httptest.NewRequest("POST", "/api/handoffs", handoffBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp HandoffResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Valid {
		t.Error("Expected invalid handoff due to summary too long")
	}
}
