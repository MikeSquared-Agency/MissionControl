package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GlobalConfig represents ~/.mission-control/config.json
type GlobalConfig struct {
	Projects    []Project   `json:"projects"`
	LastProject string      `json:"lastProject"`
	Preferences Preferences `json:"preferences"`
}

// Project represents a MissionControl project
type Project struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	LastOpened string `json:"lastOpened"`
}

// Preferences stores user preferences
type Preferences struct {
	Theme string `json:"theme"`
}

// ProjectsHandler handles project-related endpoints
type ProjectsHandler struct {
	configPath string
	mcPath     string
}

// NewProjectsHandler creates a new projects handler
func NewProjectsHandler() *ProjectsHandler {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".mission-control", "config.json")

	// Find mc binary
	mcPath := "mc"
	commonPaths := []string{
		"/usr/local/bin/mc",
		"/opt/homebrew/bin/mc",
		// Development: look relative to working directory
		"../cmd/mc/mc",
		"cmd/mc/mc",
	}
	// Also try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		commonPaths = append(commonPaths,
			filepath.Join(exeDir, "..", "cmd", "mc", "mc"),
			filepath.Join(exeDir, "mc"),
		)
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			mcPath = p
			break
		}
	}
	if path, err := exec.LookPath("mc"); err == nil {
		mcPath = path
	}

	return &ProjectsHandler{
		configPath: configPath,
		mcPath:     mcPath,
	}
}

// RegisterRoutes registers project API routes
func (h *ProjectsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/projects", h.handleProjects)
	mux.HandleFunc("/api/projects/", h.handleProject)
	mux.HandleFunc("/api/projects/check", h.handleCheckPath)
}

func (h *ProjectsHandler) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProjects(w, r)
	case http.MethodPost:
		h.createProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProjectsHandler) handleProject(w http.ResponseWriter, r *http.Request) {
	// Extract path from URL (URL-encoded)
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if path == "" || path == "check" {
		// Handled by handleCheckPath or handleProjects
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.deleteProject(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProjectsHandler) handleCheckPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path parameter required"})
		return
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	result := map[string]bool{
		"exists":     false,
		"hasGit":     false,
		"hasMission": false,
	}

	if info, err := os.Stat(path); err == nil && info.IsDir() {
		result["exists"] = true

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			result["hasGit"] = true
		}
		if _, err := os.Stat(filepath.Join(path, ".mission")); err == nil {
			result["hasMission"] = true
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ProjectsHandler) listProjects(w http.ResponseWriter, r *http.Request) {
	config, err := h.loadConfig()
	if err != nil {
		// Return empty list if config doesn't exist
		writeJSON(w, http.StatusOK, GlobalConfig{
			Projects:    []Project{},
			Preferences: Preferences{Theme: "dark"},
		})
		return
	}

	writeJSON(w, http.StatusOK, config)
}

// CreateProjectRequest is the request body for creating a project
type CreateProjectRequest struct {
	Path       string       `json:"path"`
	InitGit    bool         `json:"initGit"`
	EnableKing bool         `json:"enableKing"`
	Matrix     []MatrixCell `json:"matrix"`
}

// MatrixCell represents a cell in the workflow matrix
type MatrixCell struct {
	Phase   string `json:"phase"`
	Zone    string `json:"zone"`
	Persona string `json:"persona"`
	Enabled bool   `json:"enabled"`
}

func (h *ProjectsHandler) createProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path is required"})
		return
	}

	// Expand ~ to home directory
	path := req.Path
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create directory: %v", err),
		})
		return
	}

	// Write matrix config to temp file
	configFile, err := os.CreateTemp("", "mc-config-*.json")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create config file"})
		return
	}
	defer os.Remove(configFile.Name())

	matrixConfig := map[string]interface{}{
		"matrix": req.Matrix,
	}
	json.NewEncoder(configFile).Encode(matrixConfig)
	configFile.Close()

	// Build mc init command
	args := []string{"init", "--path", path}
	if req.InitGit {
		args = append(args, "--git")
	}
	if req.EnableKing {
		args = append(args, "--king")
	} else {
		args = append(args, "--king=false")
	}
	args = append(args, "--config", configFile.Name())

	// Execute mc init
	cmd := exec.Command(h.mcPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  fmt.Sprintf("mc init failed: %v", err),
			"output": string(output),
		})
		return
	}

	// Add to global config
	project := Project{
		Path:       path,
		Name:       filepath.Base(path),
		LastOpened: time.Now().UTC().Format(time.RFC3339),
	}

	config, _ := h.loadConfig()
	if config == nil {
		config = &GlobalConfig{
			Projects:    []Project{},
			Preferences: Preferences{Theme: "dark"},
		}
	}

	// Remove duplicate if exists
	filtered := []Project{}
	for _, p := range config.Projects {
		if p.Path != path {
			filtered = append(filtered, p)
		}
	}
	config.Projects = append(filtered, project)
	config.LastProject = path

	if err := h.saveConfig(config); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save config"})
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (h *ProjectsHandler) deleteProject(w http.ResponseWriter, r *http.Request, path string) {
	config, err := h.loadConfig()
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Config not found"})
		return
	}

	// Remove from list (don't delete from disk)
	filtered := []Project{}
	found := false
	for _, p := range config.Projects {
		if p.Path != path {
			filtered = append(filtered, p)
		} else {
			found = true
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}

	config.Projects = filtered
	if config.LastProject == path {
		if len(filtered) > 0 {
			config.LastProject = filtered[0].Path
		} else {
			config.LastProject = ""
		}
	}

	if err := h.saveConfig(config); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save config"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectsHandler) loadConfig() (*GlobalConfig, error) {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		return nil, err
	}

	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (h *ProjectsHandler) saveConfig(config *GlobalConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(h.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.configPath, data, 0644)
}
