package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/DarlingtonDeveloper/MissionControl/tracker"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	// Create .mission/state directory
	stateDir := filepath.Join(dir, ".mission", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .mission/checkpoints directory
	cpDir := filepath.Join(dir, ".mission", "checkpoints")
	if err := os.MkdirAll(cpDir, 0755); err != nil {
		t.Fatal(err)
	}

	s := NewServer(dir, nil, nil, nil)
	return s, dir
}

func TestHealthEndpoint(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got %s", resp.Status)
	}
	if resp.Version != "6.1" {
		t.Errorf("Expected version '6.1', got %s", resp.Version)
	}
}

func TestTasksEmptyList(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var tasks []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &tasks)
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestTasksWithData(t *testing.T) {
	s, dir := newTestServer(t)

	// Write test tasks
	tasksFile := filepath.Join(dir, ".mission", "state", "tasks.jsonl")
	data := `{"id":"abc123","name":"Test task","stage":"implement","zone":"backend","persona":"developer","status":"doing"}
{"id":"def456","name":"Another task","stage":"review","zone":"frontend","persona":"reviewer","status":"done"}
`
	os.WriteFile(tasksFile, []byte(data), 0644)

	routes := s.Routes()

	// Test unfiltered
	req := httptest.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	var tasks []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &tasks)
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Test filtered by stage
	req = httptest.NewRequest("GET", "/api/tasks?stage=implement", nil)
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	_ = json.Unmarshal(w.Body.Bytes(), &tasks)
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task filtered by stage, got %d", len(tasks))
	}

	// Test single task
	req = httptest.NewRequest("GET", "/api/tasks/abc123", nil)
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var task map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &task)
	if task["id"] != "abc123" {
		t.Errorf("Expected task id abc123, got %v", task["id"])
	}
}

func TestGatesEndpoint(t *testing.T) {
	s, dir := newTestServer(t)

	// Write gates
	gatesFile := filepath.Join(dir, ".mission", "state", "gates.json")
	gatesData := `{"implement":{"status":"approved","approved_at":"2026-02-08"},"review":{"status":"pending"}}`
	os.WriteFile(gatesFile, []byte(gatesData), 0644)

	routes := s.Routes()

	// All gates
	req := httptest.NewRequest("GET", "/api/gates", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var gates map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &gates)
	if len(gates) != 2 {
		t.Errorf("Expected 2 gates, got %d", len(gates))
	}

	// Single gate
	req = httptest.NewRequest("GET", "/api/gates/implement", nil)
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequirementsPlaceholder(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/requirements", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var result []interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("Expected empty array, got %d items", len(result))
	}
}

func TestRequirementsCoverage(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/requirements/coverage", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	var result RequirementsCoverage
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Total != 0 || result.Implemented != 0 || result.Coverage != 0.0 {
		t.Errorf("Expected zero coverage, got %+v", result)
	}
}

func TestGateApproveCallsMC(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	// This will fail because mc isn't installed, but we verify the endpoint exists and responds
	req := httptest.NewRequest("POST", "/api/gates/implement/approve", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	// Should get 500 (mc not found) not 404
	if w.Code == http.StatusNotFound || w.Code == http.StatusMethodNotAllowed {
		t.Errorf("Expected endpoint to exist (got %d)", w.Code)
	}

	// Verify it returns JSON error
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Expected JSON response, got: %s", w.Body.String())
	}
}

func TestChatPlaceholder(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	body := bytes.NewBufferString(`{"message":"hello"}`)
	req := httptest.NewRequest("POST", "/api/chat", body)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected 501, got %d", w.Code)
	}
}

func TestOpenClawStatus(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/openclaw/status", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var result OpenClawStatus
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Connected != false {
		t.Error("Expected connected=false")
	}
}

func TestCheckpointsList(t *testing.T) {
	s, dir := newTestServer(t)

	// Create a checkpoint dir
	cpDir := filepath.Join(dir, ".mission", "checkpoints", "cp-20260208-221853")
	os.MkdirAll(cpDir, 0755)

	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/checkpoints", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var cps []CheckpointInfo
	_ = json.Unmarshal(w.Body.Bytes(), &cps)
	if len(cps) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(cps))
	}
	if cps[0].ID != "cp-20260208-221853" {
		t.Errorf("Expected checkpoint id cp-20260208-221853, got %s", cps[0].ID)
	}
}

func TestGraphEndpoint(t *testing.T) {
	s, dir := newTestServer(t)

	tasksFile := filepath.Join(dir, ".mission", "state", "tasks.jsonl")
	os.WriteFile(tasksFile, []byte(`{"id":"a","name":"A","stage":"implement","zone":"backend","status":"done","dependencies":["b"]}
{"id":"b","name":"B","stage":"implement","zone":"backend","status":"doing"}
`), 0644)

	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/graph", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var graph GraphResponse
	_ = json.Unmarshal(w.Body.Bytes(), &graph)
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(graph.Edges))
	}
}

func TestWorkersWithoutTracker(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/workers", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// mockTracker implements TrackerReader for testing
type mockTracker struct{}

func (m *mockTracker) List() []*tracker.TrackedProcess {
	return []*tracker.TrackedProcess{{WorkerID: "w1"}}
}

func (m *mockTracker) Get(id string) (*tracker.TrackedProcess, bool) {
	if id == "w1" {
		return &tracker.TrackedProcess{WorkerID: "w1"}, true
	}
	return nil, false
}

func TestWorkersWithTracker(t *testing.T) {
	s, _ := newTestServer(t)
	s.tracker = &mockTracker{}
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/workers", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/workers/w1", nil)
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/workers/nonexist", nil)
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}
