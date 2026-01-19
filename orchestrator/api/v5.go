package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// KingStarter interface for starting King
type KingStarter interface {
	Start() error
	Stop() error
	IsRunning() bool
	SendMessage(message string) error
	AnswerQuestion(optionIndex int) error
}

// V5Handler handles v5 API endpoints
type V5Handler struct {
	king       KingStarter
	missionDir string
	mcPath     string
}

// NewV5Handler creates a new V5 handler
// workDir is the project root directory containing .mission/
func NewV5Handler(king KingStarter, workDir string) *V5Handler {
	// Try to find mc in common locations
	mcPath := "mc"
	commonPaths := []string{
		"/usr/local/bin/mc",
		"/opt/homebrew/bin/mc",
		workDir + "/cmd/mc/mc",
	}
	for _, p := range commonPaths {
		if _, err := exec.LookPath(p); err == nil {
			mcPath = p
			break
		}
	}

	return &V5Handler{
		king:       king,
		missionDir: workDir,
		mcPath:     mcPath,
	}
}

// RegisterRoutes registers v5 API routes
func (h *V5Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/king/start", h.handleKingStart)
	mux.HandleFunc("/api/king/stop", h.handleKingStop)
	mux.HandleFunc("/api/king/status", h.handleKingStatus)
	mux.HandleFunc("/api/king/message", h.handleKingMessage)
	mux.HandleFunc("/api/king/answer", h.handleKingAnswer)
	mux.HandleFunc("/api/mission/gates/", h.handleGates)
}

// handleKingStart starts the King process
func (h *V5Handler) handleKingStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.king.IsRunning() {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "already_running",
			"message": "King is already running",
		})
		return
	}

	if err := h.king.Start(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "started",
		"message": "King started successfully",
	})
}

// handleKingStop stops the King process
func (h *V5Handler) handleKingStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.king.IsRunning() {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "already_stopped",
			"message": "King is not running",
		})
		return
	}

	if err := h.king.Stop(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "stopped",
		"message": "King stopped successfully",
	})
}

// handleKingStatus returns the King status
func (h *V5Handler) handleKingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"is_running": h.king.IsRunning(),
	})
}

// handleKingMessage sends a message to King
func (h *V5Handler) handleKingMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.king.IsRunning() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "King is not running",
		})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	if err := h.king.SendMessage(req.Content); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "sent",
	})
}

// handleKingAnswer responds to a question from Claude
func (h *V5Handler) handleKingAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.king.IsRunning() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "King is not running",
		})
		return
	}

	var req struct {
		OptionIndex int `json:"option_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	if err := h.king.AnswerQuestion(req.OptionIndex); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "answered",
	})
}

// handleGates handles gate operations
func (h *V5Handler) handleGates(w http.ResponseWriter, r *http.Request) {
	// Extract phase from path: /api/mission/gates/{phase}/approve
	path := strings.TrimPrefix(r.URL.Path, "/api/mission/gates/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Phase required", http.StatusBadRequest)
		return
	}

	phase := parts[0]

	// Check if this is an approve request
	if len(parts) >= 2 && parts[1] == "approve" {
		h.handleGateApprove(w, r, phase)
		return
	}

	// Otherwise return gate status
	h.handleGateCheck(w, r, phase)
}

// handleGateCheck checks the gate status for a phase
func (h *V5Handler) handleGateCheck(w http.ResponseWriter, r *http.Request, phase string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call mc gate check
	cmd := exec.Command(h.mcPath, "gate", "check", phase)
	cmd.Dir = h.missionDir
	output, err := cmd.Output()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to check gate: %v", err),
		})
		return
	}

	// Parse and return the JSON output
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to parse gate check result",
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGateApprove approves a gate
func (h *V5Handler) handleGateApprove(w http.ResponseWriter, r *http.Request, phase string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call mc gate approve
	cmd := exec.Command(h.mcPath, "gate", "approve", phase)
	cmd.Dir = h.missionDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error":  fmt.Sprintf("Failed to approve gate: %v", err),
			"output": string(output),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "approved",
		"phase":   phase,
		"message": string(output),
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
