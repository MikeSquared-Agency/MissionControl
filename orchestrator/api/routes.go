package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/DarlingtonDeveloper/MissionControl/manager"
)

// Handler holds dependencies for API handlers
type Handler struct {
	Manager *manager.Manager
}

// NewHandler creates a new API handler
func NewHandler(m *manager.Manager) *Handler {
	return &Handler{Manager: m}
}

// Routes returns the HTTP handler with all routes
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// Agent routes
	mux.HandleFunc("/api/agents", h.handleAgents)
	mux.HandleFunc("/api/agents/", h.handleAgent)

	// Zone routes
	mux.HandleFunc("/api/zones", h.handleZones)
	mux.HandleFunc("/api/zones/", h.handleZone)

	// NOTE: King routes (/api/king/*) are handled by KingHandler, not here
	// The KingHandler uses bridge.King which runs Claude in tmux

	// Health check
	mux.HandleFunc("/api/health", h.handleHealth)

	// Wrap with CORS middleware
	return corsMiddleware(mux)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth handles GET /api/health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAgents handles /api/agents
func (h *Handler) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listAgents(w, r)
	case "POST":
		h.spawnAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAgent handles /api/agents/:id
func (h *Handler) handleAgent(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	id := parts[0]

	// Check for /message subpath
	if len(parts) > 1 && parts[1] == "message" {
		if r.Method == "POST" {
			h.sendMessage(w, r, id)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for /move subpath
	if len(parts) > 1 && parts[1] == "move" {
		if r.Method == "POST" {
			h.moveAgent(w, r, id)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for /respond subpath (attention response)
	if len(parts) > 1 && parts[1] == "respond" {
		if r.Method == "POST" {
			h.respondToAttention(w, r, id)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case "GET":
		h.getAgent(w, r, id)
	case "DELETE":
		h.killAgent(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listAgents handles GET /api/agents
func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	agents := h.Manager.List()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agents)
}

// spawnAgent handles POST /api/agents
func (h *Handler) spawnAgent(w http.ResponseWriter, r *http.Request) {
	var req manager.SpawnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate
	if req.Task == "" {
		http.Error(w, "task is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = manager.AgentTypeClaudeCode // Default to Claude Code
	}

	agent, err := h.Manager.Spawn(req)
	if err != nil {
		http.Error(w, "Failed to spawn agent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(agent)
}

// getAgent handles GET /api/agents/:id
func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request, id string) {
	agent, ok := h.Manager.Get(id)
	if !ok {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agent)
}

// killAgent handles DELETE /api/agents/:id
func (h *Handler) killAgent(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.Manager.Kill(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MessageRequest represents a message to send to an agent
type MessageRequest struct {
	Content string `json:"content"`
}

// sendMessage handles POST /api/agents/:id/message
func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request, id string) {
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Manager.SendMessage(id, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// MoveAgentRequest represents a request to move an agent to a zone
type MoveAgentRequest struct {
	ZoneID string `json:"zoneId"`
}

// moveAgent handles POST /api/agents/:id/move
func (h *Handler) moveAgent(w http.ResponseWriter, r *http.Request, id string) {
	var req MoveAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Manager.MoveAgent(id, req.ZoneID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "moved"})
}

// RespondRequest represents a response to an attention request
type RespondRequest struct {
	Response string `json:"response"`
}

// respondToAttention handles POST /api/agents/:id/respond
func (h *Handler) respondToAttention(w http.ResponseWriter, r *http.Request, id string) {
	var req RespondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Send the response as a message to the agent
	// The manager will handle clearing attention state
	if err := h.Manager.SendMessage(id, req.Response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "responded"})
}

// Zone handlers

// handleZones handles /api/zones
func (h *Handler) handleZones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listZones(w, r)
	case "POST":
		h.createZone(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleZone handles /api/zones/:id
func (h *Handler) handleZone(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/zones/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Zone ID required", http.StatusBadRequest)
		return
	}

	id := parts[0]

	switch r.Method {
	case "GET":
		h.getZone(w, r, id)
	case "PUT":
		h.updateZone(w, r, id)
	case "DELETE":
		h.deleteZone(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listZones handles GET /api/zones
func (h *Handler) listZones(w http.ResponseWriter, r *http.Request) {
	zones := h.Manager.ListZones()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(zones)
}

// createZone handles POST /api/zones
func (h *Handler) createZone(w http.ResponseWriter, r *http.Request) {
	var zone manager.Zone
	if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if zone.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	created, err := h.Manager.CreateZone(&zone)
	if err != nil {
		http.Error(w, "Failed to create zone: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// getZone handles GET /api/zones/:id
func (h *Handler) getZone(w http.ResponseWriter, r *http.Request, id string) {
	zone, ok := h.Manager.GetZone(id)
	if !ok {
		http.Error(w, "Zone not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(zone)
}

// updateZone handles PUT /api/zones/:id
func (h *Handler) updateZone(w http.ResponseWriter, r *http.Request, id string) {
	var updates manager.Zone
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	zone, err := h.Manager.UpdateZone(id, &updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(zone)
}

// deleteZone handles DELETE /api/zones/:id
func (h *Handler) deleteZone(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.Manager.DeleteZone(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NOTE: King Mode handlers are in api/king.go via KingHandler
// They use bridge.King which runs Claude in a tmux session

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
