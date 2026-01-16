package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mike/agent-orchestra/manager"
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

	// Health check
	mux.HandleFunc("/api/health", h.handleHealth)

	// Wrap with CORS middleware
	return corsMiddleware(mux)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
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
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
	json.NewEncoder(w).Encode(agents)
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
		req.Type = manager.AgentTypePython // Default to python
	}

	agent, err := h.Manager.Spawn(req)
	if err != nil {
		http.Error(w, "Failed to spawn agent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

// getAgent handles GET /api/agents/:id
func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request, id string) {
	agent, ok := h.Manager.Get(id)
	if !ok {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}
