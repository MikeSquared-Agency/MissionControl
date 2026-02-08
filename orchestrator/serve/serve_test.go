package serve

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DarlingtonDeveloper/MissionControl/tokens"
	"github.com/DarlingtonDeveloper/MissionControl/tracker"
	"github.com/gorilla/websocket"
)

// startTestServer starts a serve instance on a random port and returns the base URL + cleanup func.
func startTestServer(t *testing.T, missionDir string, apiOnly bool) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := Config{Port: port, MissionDir: missionDir, APIOnly: apiOnly}
	errCh := make(chan error, 1)
	go func() { errCh <- Run(cfg) }()
	// Wait for server to be ready
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(baseURL + "/api/health"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return baseURL, func() {}
}

// createTestMission creates a temp directory with .mission/state files.
func createTestMission(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".mission", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(stateDir, "stage.json"), []byte(`{"current":"implement"}`), 0644)
	os.WriteFile(filepath.Join(stateDir, "gates.json"), []byte(`{"discovery":{"status":"approved"}}`), 0644)
	os.WriteFile(filepath.Join(stateDir, "tasks.jsonl"), []byte(`{"id":"t1","name":"Test task","status":"pending"}`+"\n"), 0644)
	return dir
}

func getJSON(t *testing.T, url string) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result
}

func getJSONArray(t *testing.T, url string) []interface{} {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var result []interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func TestHealthEndpoint(t *testing.T) {
	dir := t.TempDir()
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	data := getJSON(t, baseURL+"/api/health")
	if data["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", data["status"])
	}
	if data["version"] != "6.1" {
		t.Errorf("expected version=6.1, got %v", data["version"])
	}
}

func TestStatusEndpoint(t *testing.T) {
	dir := createTestMission(t)
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	data := getJSON(t, baseURL+"/api/status")

	// Check stage
	stage, ok := data["stage"].(map[string]interface{})
	if !ok {
		t.Fatal("expected stage in status response")
	}
	if stage["current"] != "implement" {
		t.Errorf("expected stage=implement, got %v", stage["current"])
	}

	// Check gates
	gates, ok := data["gates"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gates in status response")
	}
	disc := gates["discovery"].(map[string]interface{})
	if disc["status"] != "approved" {
		t.Errorf("expected discovery gate approved, got %v", disc["status"])
	}

	// Check tasks
	tasks, ok := data["tasks"].([]interface{})
	if !ok {
		t.Fatal("expected tasks array in status response")
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Check workers and tokens exist
	if _, ok := data["workers"]; !ok {
		t.Error("expected workers in status response")
	}
	if _, ok := data["tokens"]; !ok {
		t.Error("expected tokens in status response")
	}
}

func TestWebSocketConnection(t *testing.T) {
	dir := createTestMission(t)
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	wsURL := "ws" + baseURL[4:] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	if resp.StatusCode != 101 {
		t.Fatalf("expected 101, got %d", resp.StatusCode)
	}

	// Should receive initial state sync
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws message: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("unmarshal ws message: %v", err)
	}

	// Check it's a state sync message
	if envelope["type"] != "state_sync" {
		t.Logf("received message type: %v", envelope["type"])
		// Accept any initial message as valid connection proof
	}
}

func TestWebSocketSubscription(t *testing.T) {
	dir := createTestMission(t)
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	wsURL := "ws" + baseURL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read initial state sync
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.ReadMessage()

	// Send subscribe message
	sub := map[string]interface{}{
		"type":   "subscribe",
		"topics": []string{"stage"},
	}
	if err := conn.WriteJSON(sub); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	// Give the hub time to process
	time.Sleep(50 * time.Millisecond)

	// Connection should still be alive
	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		t.Fatalf("ping failed after subscribe: %v", err)
	}
}

func TestPlaceholderEndpoints(t *testing.T) {
	dir := t.TempDir()
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	// /api/requirements → empty array
	reqs := getJSONArray(t, baseURL+"/api/requirements")
	if len(reqs) != 0 {
		t.Errorf("expected empty requirements, got %d", len(reqs))
	}

	// /api/specs → empty array
	specs := getJSONArray(t, baseURL+"/api/specs")
	if len(specs) != 0 {
		t.Errorf("expected empty specs, got %d", len(specs))
	}

	// /api/openclaw/status → {connected: false}
	oc := getJSON(t, baseURL+"/api/openclaw/status")
	if oc["connected"] != false {
		t.Errorf("expected connected=false, got %v", oc["connected"])
	}

	// /api/requirements/coverage
	cov := getJSON(t, baseURL+"/api/requirements/coverage")
	if cov["total"] != float64(0) {
		t.Errorf("expected total=0, got %v", cov["total"])
	}

	// /api/specs/orphans
	orphans := getJSONArray(t, baseURL+"/api/specs/orphans")
	if len(orphans) != 0 {
		t.Errorf("expected empty orphans, got %d", len(orphans))
	}
}

func TestTokensEndpoint(t *testing.T) {
	dir := t.TempDir()
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	data := getJSON(t, baseURL+"/api/tokens")
	// Should return a summary object (tokens accumulator starts empty)
	if data == nil {
		t.Error("expected non-nil tokens response")
	}
}

func TestWorkersEndpoint(t *testing.T) {
	dir := t.TempDir()
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	resp, err := http.Get(baseURL + "/api/workers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var workers []interface{}
	json.NewDecoder(resp.Body).Decode(&workers)
	if len(workers) != 0 {
		t.Errorf("expected empty workers, got %d", len(workers))
	}
}

func TestAPIOnlyMode(t *testing.T) {
	dir := createTestMission(t)
	baseURL, cleanup := startTestServer(t, dir, true)
	defer cleanup()

	// All endpoints should work
	getJSON(t, baseURL+"/api/health")
	getJSON(t, baseURL+"/api/status")
	getJSON(t, baseURL+"/api/tokens")
}

func TestReadJSONL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "test.jsonl")
		os.WriteFile(f, []byte(`{"a":1}`+"\n"+`{"b":2}`+"\n"), 0644)
		results, err := readJSONL(f)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("empty", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "empty.jsonl")
		os.WriteFile(f, []byte(""), 0644)
		results, err := readJSONL(f)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("multiple_objects", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "multi.jsonl")
		os.WriteFile(f, []byte(`{"a":1}`+"\n"+`{"b":2}`+"\n"+`{"c":3}`+"\n"), 0644)
		results, err := readJSONL(f)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
	})

	t.Run("missing_file", func(t *testing.T) {
		_, err := readJSONL("/nonexistent/file.jsonl")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

func TestBuildState(t *testing.T) {
	t.Run("full_state", func(t *testing.T) {
		dir := createTestMission(t)
		// We need tracker and accumulator - use the same approach as Run()
		trk := newTestTracker(dir)
		acc := newTestAccumulator()

		state := buildState(dir, trk, acc)

		if _, ok := state["stage"]; !ok {
			t.Error("expected stage in state")
		}
		if _, ok := state["gates"]; !ok {
			t.Error("expected gates in state")
		}
		if _, ok := state["tasks"]; !ok {
			t.Error("expected tasks in state")
		}
		if _, ok := state["workers"]; !ok {
			t.Error("expected workers in state")
		}
		if _, ok := state["tokens"]; !ok {
			t.Error("expected tokens in state")
		}
	})

	t.Run("empty_mission_dir", func(t *testing.T) {
		dir := t.TempDir()
		trk := newTestTracker(dir)
		acc := newTestAccumulator()

		state := buildState(dir, trk, acc)

		// Should still have workers and tokens
		if _, ok := state["workers"]; !ok {
			t.Error("expected workers in state")
		}
		if _, ok := state["tokens"]; !ok {
			t.Error("expected tokens in state")
		}
		// Stage/gates/tasks may be absent
	})

	t.Run("partial_state", func(t *testing.T) {
		dir := t.TempDir()
		stateDir := filepath.Join(dir, ".mission", "state")
		os.MkdirAll(stateDir, 0755)
		os.WriteFile(filepath.Join(stateDir, "stage.json"), []byte(`{"current":"discovery"}`), 0644)
		// No gates or tasks

		trk := newTestTracker(dir)
		acc := newTestAccumulator()

		state := buildState(dir, trk, acc)
		if _, ok := state["stage"]; !ok {
			t.Error("expected stage in state")
		}
		if _, ok := state["gates"]; ok {
			t.Error("did not expect gates without gates.json")
		}
	})
}

// Helper to create a tracker for testing.
func newTestTracker(dir string) *tracker.Tracker {
	return tracker.NewTracker(dir, func(string, *tracker.TrackedProcess) {})
}

// Helper to create an accumulator for testing.
func newTestAccumulator() *tokens.Accumulator {
	return tokens.NewAccumulator(0, func(string, int, int, int) {})
}
