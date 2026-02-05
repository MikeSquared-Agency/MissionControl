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

// KingHandler handles King API endpoints
type KingHandler struct {
	king       KingStarter
	missionDir string
	mcPath     string
}

// NewKingHandler creates a new King handler
// workDir is the project root directory containing .mission/
func NewKingHandler(king KingStarter, workDir string) *KingHandler {
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

	return &KingHandler{
		king:       king,
		missionDir: workDir,
		mcPath:     mcPath,
	}
}

// RegisterRoutes registers King API routes
func (h *KingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/king/start", h.handleKingStart)
	mux.HandleFunc("/api/king/stop", h.handleKingStop)
	mux.HandleFunc("/api/king/status", h.handleKingStatus)
	mux.HandleFunc("/api/king/message", h.handleKingMessage)
	mux.HandleFunc("/api/king/answer", h.handleKingAnswer)
	mux.HandleFunc("/api/mission/gates/", h.handleGates)
}

// handleKingStart starts the King process
func (h *KingHandler) handleKingStart(w http.ResponseWriter, r *http.Request) {
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
func (h *KingHandler) handleKingStop(w http.ResponseWriter, r *http.Request) {
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
func (h *KingHandler) handleKingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"is_running": h.king.IsRunning(),
	})
}

// handleKingMessage sends a message to King
func (h *KingHandler) handleKingMessage(w http.ResponseWriter, r *http.Request) {
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
func (h *KingHandler) handleKingAnswer(w http.ResponseWriter, r *http.Request) {
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
func (h *KingHandler) handleGates(w http.ResponseWriter, r *http.Request) {
	// Extract stage from path: /api/mission/gates/{stage}/approve
	path := strings.TrimPrefix(r.URL.Path, "/api/mission/gates/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Stage required", http.StatusBadRequest)
		return
	}

	stage := parts[0]

	// Check if this is an approve request
	if len(parts) >= 2 && parts[1] == "approve" {
		h.handleGateApprove(w, r, stage)
		return
	}

	// Otherwise return gate status
	h.handleGateCheck(w, r, stage)
}

// handleGateCheck checks the gate status for a stage
func (h *KingHandler) handleGateCheck(w http.ResponseWriter, r *http.Request, stage string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call mc gate check
	cmd := exec.Command(h.mcPath, "gate", "check", stage)
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
func (h *KingHandler) handleGateApprove(w http.ResponseWriter, r *http.Request, stage string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call mc gate approve
	cmd := exec.Command(h.mcPath, "gate", "approve", stage)
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
		"stage":   stage,
		"message": string(output),
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
