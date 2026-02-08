package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DarlingtonDeveloper/MissionControl/manager"
)

func newTestHandler() *Handler {
	m := manager.NewManager("/tmp/agents")
	return NewHandler(m)
}

func TestHealthEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %s", resp["status"])
	}
}

func TestListZonesEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("GET", "/api/zones", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var zones []manager.Zone
	_ = json.Unmarshal(w.Body.Bytes(), &zones)

	// Should have at least the default zone
	if len(zones) < 1 {
		t.Error("Expected at least 1 zone (default)")
	}

	// Find default zone
	found := false
	for _, z := range zones {
		if z.ID == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default zone not found")
	}
}

func TestCreateZoneEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	body := bytes.NewBufferString(`{"name":"Test Zone","color":"#22c55e","workingDir":"/tmp"}`)
	req := httptest.NewRequest("POST", "/api/zones", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var zone manager.Zone
	_ = json.Unmarshal(w.Body.Bytes(), &zone)

	if zone.Name != "Test Zone" {
		t.Errorf("Expected zone name 'Test Zone', got %s", zone.Name)
	}

	if zone.ID == "" {
		t.Error("Zone ID should not be empty")
	}
}

func TestCreateZoneWithoutName(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	body := bytes.NewBufferString(`{"color":"#22c55e"}`)
	req := httptest.NewRequest("POST", "/api/zones", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUpdateZoneEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	// First create a zone
	createBody := bytes.NewBufferString(`{"name":"Original","color":"#22c55e"}`)
	createReq := httptest.NewRequest("POST", "/api/zones", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	routes.ServeHTTP(createW, createReq)

	var created manager.Zone
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	// Update the zone
	updateBody := bytes.NewBufferString(`{"name":"Updated","color":"#3b82f6"}`)
	updateReq := httptest.NewRequest("PUT", "/api/zones/"+created.ID, updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	routes.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var updated manager.Zone
	_ = json.Unmarshal(updateW.Body.Bytes(), &updated)

	if updated.Name != "Updated" {
		t.Errorf("Expected updated name 'Updated', got %s", updated.Name)
	}
}

func TestDeleteZoneEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	// Create a zone
	createBody := bytes.NewBufferString(`{"name":"To Delete","color":"#22c55e"}`)
	createReq := httptest.NewRequest("POST", "/api/zones", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	routes.ServeHTTP(createW, createReq)

	var created manager.Zone
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	// Delete the zone
	deleteReq := httptest.NewRequest("DELETE", "/api/zones/"+created.ID, nil)
	deleteW := httptest.NewRecorder()
	routes.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", deleteW.Code)
	}

	// Verify it's gone
	getReq := httptest.NewRequest("GET", "/api/zones/"+created.ID, nil)
	getW := httptest.NewRecorder()
	routes.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted zone, got %d", getW.Code)
	}
}

func TestListAgentsEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var agents []manager.Agent
	_ = json.Unmarshal(w.Body.Bytes(), &agents)

	// Should be empty initially
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestSpawnAgentWithoutTask(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	body := bytes.NewBufferString(`{"type":"claude-code","name":"Test"}`)
	req := httptest.NewRequest("POST", "/api/agents", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetNonExistentAgent(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("GET", "/api/agents/non-existent", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestKillNonExistentAgent(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("DELETE", "/api/agents/non-existent", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestSendMessageToNonExistentAgent(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	body := bytes.NewBufferString(`{"content":"hello"}`)
	req := httptest.NewRequest("POST", "/api/agents/non-existent/message", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestMoveAgentEndpoint(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	// Try to move non-existent agent
	body := bytes.NewBufferString(`{"zoneId":"default"}`)
	req := httptest.NewRequest("POST", "/api/agents/non-existent/move", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// NOTE: TestKingMessageWithoutContent removed - King routes are now handled by KingHandler (api/king.go)
// not by the main Routes() handler

func TestCORSHeaders(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("OPTIONS", "/api/health", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}

	cors := w.Header().Get("Access-Control-Allow-Origin")
	if cors != "*" {
		t.Errorf("Expected CORS header '*', got %s", cors)
	}
}

func TestInvalidJSON(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest("POST", "/api/zones", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	routes := h.Routes()

	req := httptest.NewRequest("PATCH", "/api/zones", nil)
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
