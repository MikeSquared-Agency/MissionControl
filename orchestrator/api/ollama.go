package api

import (
	"encoding/json"
	"net/http"

	"github.com/mike/mission-control/ollama"
)

// OllamaHandler handles Ollama-related API endpoints
type OllamaHandler struct {
	client *ollama.Client
}

// NewOllamaHandler creates a new OllamaHandler
func NewOllamaHandler() *OllamaHandler {
	return &OllamaHandler{
		client: ollama.NewClient(""),
	}
}

// OllamaStatusResponse is the response for GET /api/ollama/status
type OllamaStatusResponse struct {
	Running bool     `json:"running"`
	Models  []string `json:"models,omitempty"`
}

// RegisterRoutes adds Ollama routes to the given mux
func (h *OllamaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/ollama/status", h.handleStatus)
	mux.HandleFunc("/api/ollama/models", h.handleModels)
}

// handleStatus handles GET /api/ollama/status
func (h *OllamaHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := OllamaStatusResponse{
		Running: h.client.IsRunning(),
	}

	if response.Running {
		names, err := h.client.GetModelNames()
		if err == nil {
			response.Models = names
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleModels handles GET /api/ollama/models
func (h *OllamaHandler) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := h.client.ListModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
