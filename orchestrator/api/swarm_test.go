package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockService spins up a test HTTP server that serves canned JSON responses
// keyed by request path.
func mockService(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := responses[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
}

func TestSwarmOverviewEndpoint(t *testing.T) {
	// Spin up mock HTTP servers for all 5 services.
	warren := mockService(t, map[string]string{
		"/admin/health": `{"status":"ok","uptime":12345}`,
		"/admin/agents": `[{"id":"a1","name":"Agent1","state":"ready"}]`,
	})
	defer warren.Close()

	chronicle := mockService(t, map[string]string{
		"/api/v1/metrics/summary": `{"total_events":100}`,
		"/api/v1/dlq/stats":       `{"depth":3}`,
	})
	defer chronicle.Close()

	dispatch := mockService(t, map[string]string{
		"/api/v1/stats":  `{"pending":2,"in_progress":5,"completed":10}`,
		"/api/v1/agents": `[{"id":"d1","status":"active"}]`,
	})
	defer dispatch.Close()

	promptforge := mockService(t, map[string]string{
		"/api/prompts": `[{"id":"p1"},{"id":"p2"},{"id":"p3"}]`,
	})
	defer promptforge.Close()

	alexandria := mockService(t, map[string]string{
		"/api/collections": `[{"id":"c1"}]`,
	})
	defer alexandria.Close()

	// Override package-level URL vars to point at our mock servers.
	origWarren := warrenURL
	origChronicle := chronicleURL
	origDispatch := dispatchURL
	origPromptForge := promptForgeURL
	origAlexandria := alexandriaURL
	warrenURL = warren.URL
	chronicleURL = chronicle.URL
	dispatchURL = dispatch.URL
	promptForgeURL = promptforge.URL
	alexandriaURL = alexandria.URL
	defer func() {
		warrenURL = origWarren
		chronicleURL = origChronicle
		dispatchURL = origDispatch
		promptForgeURL = origPromptForge
		alexandriaURL = origAlexandria
	}()

	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/swarm/overview", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var overview map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// All 5 service keys should be present.
	for _, key := range []string{"warren", "chronicle", "dispatch", "promptforge", "alexandria"} {
		if _, ok := overview[key]; !ok {
			t.Errorf("Expected key %q in response", key)
		}
	}

	// Must have fetched_at.
	if _, ok := overview["fetched_at"]; !ok {
		t.Error("Expected fetched_at in response")
	}

	// Errors should be empty.
	var errMap map[string]string
	if err := json.Unmarshal(overview["errors"], &errMap); err != nil {
		t.Fatalf("Failed to parse errors: %v", err)
	}
	if len(errMap) != 0 {
		t.Errorf("Expected empty errors, got %v", errMap)
	}

	// PromptForge should have been counted as array.
	var pf map[string]int
	if err := json.Unmarshal(overview["promptforge"], &pf); err != nil {
		t.Fatalf("Failed to parse promptforge: %v", err)
	}
	if pf["prompt_count"] != 3 {
		t.Errorf("Expected prompt_count=3, got %d", pf["prompt_count"])
	}

	// Alexandria should have been counted as array.
	var ax map[string]int
	if err := json.Unmarshal(overview["alexandria"], &ax); err != nil {
		t.Fatalf("Failed to parse alexandria: %v", err)
	}
	if ax["collection_count"] != 1 {
		t.Errorf("Expected collection_count=1, got %d", ax["collection_count"])
	}
}

func TestSwarmOverviewPartialFailure(t *testing.T) {
	// Only mock Warren and PromptForge; the rest point at dead URLs.
	warren := mockService(t, map[string]string{
		"/admin/health": `{"status":"ok"}`,
		"/admin/agents": `[]`,
	})
	defer warren.Close()

	promptforge := mockService(t, map[string]string{
		"/api/prompts": `[]`,
	})
	defer promptforge.Close()

	origWarren := warrenURL
	origChronicle := chronicleURL
	origDispatch := dispatchURL
	origPromptForge := promptForgeURL
	origAlexandria := alexandriaURL
	warrenURL = warren.URL
	chronicleURL = "http://127.0.0.1:1" // dead
	dispatchURL = "http://127.0.0.1:1"
	promptForgeURL = promptforge.URL
	alexandriaURL = "http://127.0.0.1:1"
	defer func() {
		warrenURL = origWarren
		chronicleURL = origChronicle
		dispatchURL = origDispatch
		promptForgeURL = origPromptForge
		alexandriaURL = origAlexandria
	}()

	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/swarm/overview", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 even with partial failure, got %d", w.Code)
	}

	var overview map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Warren and PromptForge should have data.
	if overview["warren"] == nil {
		t.Error("Expected warren data")
	}
	if overview["promptforge"] == nil {
		t.Error("Expected promptforge data")
	}

	// Errors should contain the three dead services.
	var errMap map[string]string
	if err := json.Unmarshal(overview["errors"], &errMap); err != nil {
		t.Fatalf("Failed to parse errors: %v", err)
	}
	for _, svc := range []string{"chronicle", "dispatch", "alexandria"} {
		if _, ok := errMap[svc]; !ok {
			t.Errorf("Expected error for %q", svc)
		}
	}
	// Warren and promptforge should NOT be in errors.
	for _, svc := range []string{"warren", "promptforge"} {
		if _, ok := errMap[svc]; ok {
			t.Errorf("Did not expect error for %q", svc)
		}
	}
}

func TestSwarmOverviewMethodNotAllowed(t *testing.T) {
	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("POST", "/api/swarm/overview", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestSwarmWarrenHealthProxy(t *testing.T) {
	warren := mockService(t, map[string]string{
		"/admin/health": `{"status":"ok","uptime":9999,"version":"1.2.3"}`,
	})
	defer warren.Close()

	origWarren := warrenURL
	warrenURL = warren.URL
	defer func() { warrenURL = origWarren }()

	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/swarm/warren/health", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	// Verify the proxied JSON matches.
	var health map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if health["status"] != "ok" {
		t.Errorf("Expected status ok, got %v", health["status"])
	}
	if health["version"] != "1.2.3" {
		t.Errorf("Expected version 1.2.3, got %v", health["version"])
	}
}

func TestSwarmWarrenHealthDown(t *testing.T) {
	origWarren := warrenURL
	warrenURL = "http://127.0.0.1:1" // dead
	defer func() { warrenURL = origWarren }()

	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/swarm/warren/health", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected 502, got %d", w.Code)
	}
}

func TestSwarmWarrenEventsSSE(t *testing.T) {
	// Create a mock SSE stream from Warren.
	sseData := "data: {\"type\":\"heartbeat\"}\n\n"
	warren := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/events" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}
		fmt.Fprint(w, sseData)
		if ok {
			flusher.Flush()
		}
		// Close after sending one event so the proxy returns.
	}))
	defer warren.Close()

	origWarren := warrenURL
	warrenURL = warren.URL
	defer func() { warrenURL = origWarren }()

	s, _ := newTestServer(t)
	routes := s.Routes()

	req := httptest.NewRequest("GET", "/api/swarm/warren/events", nil)
	w := httptest.NewRecorder()

	// Run in goroutine with a timeout to avoid hanging.
	done := make(chan struct{})
	go func() {
		routes.ServeHTTP(w, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE proxy timed out")
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Expected Content-Type text/event-stream, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "heartbeat") {
		t.Errorf("Expected SSE data passthrough, got: %s", body)
	}
}
