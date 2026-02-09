package openclaw

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DarlingtonDeveloper/MissionControl/tracker"
)

// mockBroadcaster records all broadcasts for assertion.
type mockBroadcaster struct {
	mu     sync.Mutex
	events []broadcastEvent
}

type broadcastEvent struct {
	Topic     string
	EventType string
	Data      interface{}
}

func (m *mockBroadcaster) BroadcastRaw(topic, eventType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, broadcastEvent{Topic: topic, EventType: eventType, Data: data})
}

func (m *mockBroadcaster) getEvents() []broadcastEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]broadcastEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// newTestHandler creates a Handler with no real bridge connection.
func newTestHandler(t *testing.T, hub Broadcaster, trk *tracker.Tracker) *Handler {
	t.Helper()
	bridge := &Bridge{
		state:   StateDisconnected,
		pending: make(map[string]chan *Frame),
		stopCh:  make(chan struct{}),
	}
	return NewHandler(bridge, hub, trk)
}

func registerWorker(t *testing.T, mux *http.ServeMux, sessionKey, label, taskID, persona, zone, model string) {
	t.Helper()
	body, _ := json.Marshal(workerRegisterRequest{
		SessionKey: sessionKey,
		Label:      label,
		TaskID:     taskID,
		Persona:    persona,
		Zone:       zone,
		Model:      model,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/mc/worker/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register worker: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func simulateLifecycleEvent(h *Handler, sessionKey, runID, phase string) {
	ev := agentEventPayload{
		RunID:      runID,
		Stream:     "lifecycle",
		SessionKey: sessionKey,
	}
	ev.Data.Phase = phase
	ev.Data.StartedAt = time.Now().UnixMilli()
	payload, _ := json.Marshal(ev)
	h.handleLifecycleEvent(payload)
}

func TestWorkerRegisterAndLifecycleStart(t *testing.T) {
	hub := &mockBroadcaster{}
	trk := tracker.NewTracker(t.TempDir(), nil)
	h := newTestHandler(t, hub, trk)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	sessionKey := "agent:main:subagent:abc123"
	registerWorker(t, mux, sessionKey, "worker-1", "task-42", "coder", "backend", "claude-4")

	// Simulate lifecycle/start
	simulateLifecycleEvent(h, sessionKey, "run-001", "start")

	// Tracker should have the worker
	workers := trk.List()
	if len(workers) != 1 {
		t.Fatalf("expected 1 tracked worker, got %d", len(workers))
	}
	if workers[0].WorkerID != "worker-1" || workers[0].TaskID != "task-42" {
		t.Fatalf("unexpected worker: %+v", workers[0])
	}

	// Hub should have broadcast worker_started
	events := hub.getEvents()
	found := false
	for _, e := range events {
		if e.Topic == "workers" && e.EventType == "worker_started" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected worker_started broadcast, got: %+v", events)
	}
}

func TestLifecycleEndDeregisters(t *testing.T) {
	hub := &mockBroadcaster{}
	trk := tracker.NewTracker(t.TempDir(), nil)
	h := newTestHandler(t, hub, trk)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	sessionKey := "agent:main:subagent:def456"
	registerWorker(t, mux, sessionKey, "worker-2", "task-99", "tester", "backend", "claude-4")
	simulateLifecycleEvent(h, sessionKey, "run-002", "start")

	// Now end
	simulateLifecycleEvent(h, sessionKey, "run-002", "end")

	// Tracker should be empty
	workers := trk.List()
	if len(workers) != 0 {
		t.Fatalf("expected 0 tracked workers after end, got %d", len(workers))
	}

	// Hub should have worker_stopped
	events := hub.getEvents()
	found := false
	for _, e := range events {
		if e.Topic == "workers" && e.EventType == "worker_stopped" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected worker_stopped broadcast, got: %+v", events)
	}
}

func TestWorkersListEndpoint(t *testing.T) {
	hub := &mockBroadcaster{}
	trk := tracker.NewTracker(t.TempDir(), nil)
	h := newTestHandler(t, hub, trk)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	// Empty list initially
	req := httptest.NewRequest(http.MethodGet, "/api/mc/workers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// Register and start a worker
	sessionKey := "agent:main:subagent:ghi789"
	registerWorker(t, mux, sessionKey, "worker-3", "task-10", "coder", "frontend", "gpt-4")
	simulateLifecycleEvent(h, sessionKey, "run-003", "start")

	// List should now have 1
	req = httptest.NewRequest(http.MethodGet, "/api/mc/workers", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 worker, got %d", len(list))
	}
}

func TestLifecycleStartBeforeRegistration(t *testing.T) {
	hub := &mockBroadcaster{}
	trk := tracker.NewTracker(t.TempDir(), nil)
	h := newTestHandler(t, hub, trk)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	sessionKey := "agent:main:subagent:race123"

	// Lifecycle/start arrives BEFORE registration
	simulateLifecycleEvent(h, sessionKey, "run-004", "start")

	// Should be buffered, tracker empty
	if len(trk.List()) != 0 {
		t.Fatalf("expected 0 workers before registration")
	}

	// Now register — should process the buffered start
	registerWorker(t, mux, sessionKey, "worker-race", "task-race", "coder", "backend", "claude-4")

	// Tracker should now have the worker
	workers := trk.List()
	if len(workers) != 1 {
		t.Fatalf("expected 1 worker after late registration, got %d", len(workers))
	}
	if workers[0].WorkerID != "worker-race" {
		t.Fatalf("unexpected worker: %+v", workers[0])
	}

	// Hub should have broadcast worker_started
	events := hub.getEvents()
	found := false
	for _, e := range events {
		if e.Topic == "workers" && e.EventType == "worker_started" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected worker_started after buffered start processing")
	}
}

func TestRegisterEndpointValidation(t *testing.T) {
	h := newTestHandler(t, nil, nil)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"label": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/mc/worker/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing session_key, got %d", w.Code)
	}

	// GET not allowed
	req = httptest.NewRequest(http.MethodGet, "/api/mc/worker/register", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestDedupLifecycleEvents(t *testing.T) {
	hub := &mockBroadcaster{}
	trk := tracker.NewTracker(t.TempDir(), nil)
	h := newTestHandler(t, hub, trk)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	sessionKey := "agent:main:subagent:dedup1"
	registerWorker(t, mux, sessionKey, "worker-dup", "task-dup", "coder", "backend", "claude-4")

	// Send start twice
	simulateLifecycleEvent(h, sessionKey, "run-dup", "start")
	simulateLifecycleEvent(h, sessionKey, "run-dup", "start")

	// Should only have 1 worker, not error
	workers := trk.List()
	if len(workers) != 1 {
		t.Fatalf("expected 1 worker after dedup, got %d", len(workers))
	}

	// Count worker_started broadcasts - should be exactly 1
	events := hub.getEvents()
	count := 0
	for _, e := range events {
		if e.EventType == "worker_started" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 worker_started broadcast, got %d", count)
	}
}

func TestNonSubagentEventsIgnored(t *testing.T) {
	trk := tracker.NewTracker(t.TempDir(), nil)
	h := newTestHandler(t, nil, trk)

	// Main session event (no "subagent:" in session key) should be ignored
	ev := agentEventPayload{
		RunID:      "run-main",
		Stream:     "lifecycle",
		SessionKey: "agent:main:main",
	}
	ev.Data.Phase = "start"
	payload, _ := json.Marshal(ev)
	h.handleLifecycleEvent(payload)

	if len(trk.List()) != 0 {
		t.Fatalf("main session events should be ignored")
	}
}

func TestWorkersListWithoutTracker(t *testing.T) {
	h := newTestHandler(t, nil, nil)
	mux := http.NewServeMux()
	h.RegisterMCRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/mc/workers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list without tracker")
	}
}
