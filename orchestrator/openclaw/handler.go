package openclaw

import (
	"encoding/json"
	"net/http"
)

// Handler exposes REST endpoints for the OpenClaw bridge.
type Handler struct {
	bridge *Bridge
}

// NewHandler creates a new HTTP handler wrapping the bridge.
func NewHandler(bridge *Bridge) *Handler {
	return &Handler{bridge: bridge}
}

// RegisterRoutes registers /api/openclaw/* routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/openclaw/status", h.handleStatus)
	mux.HandleFunc("/api/openclaw/send", h.handleSend)
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.bridge.Status())
}

// SendRequest is the JSON body for POST /api/openclaw/send.
type SendRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

func (h *Handler) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Method == "" {
		http.Error(w, "method is required", http.StatusBadRequest)
		return
	}

	resp, err := h.bridge.Send(req.Method, req.Params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
