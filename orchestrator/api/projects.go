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
	Path        string `json:"path"`
	Name        string `json:"name"`
	LastOpened  string `json:"lastOpened"`
	Mode        string `json:"mode,omitempty"`        // "online" or "offline"
	OllamaModel string `json:"ollamaModel,omitempty"` // e.g., "qwen3-coder"
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
	mux.HandleFunc("/api/projects/check", h.handleCheckPath)
	mux.HandleFunc("/api/projects/", h.handleProject)
	mux.HandleFunc("/api/browse", h.handleBrowse)
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
	urlPath := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if urlPath == "" || urlPath == "check" {
		// Handled by handleCheckPath or handleProjects
		return
	}

	// Check if this is a persona route: {project-path}/personas[/{persona-id}[/prompt]]
	if idx := strings.Index(urlPath, "/personas"); idx != -1 {
		projectPath := urlPath[:idx]
		personaPath := urlPath[idx+len("/personas"):]
		h.handlePersonas(w, r, projectPath, personaPath)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.deleteProject(w, r, urlPath)
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
	Path        string       `json:"path"`
	Import      bool         `json:"import"` // If true, import existing .mission project without running mc init
	InitGit     bool         `json:"initGit"`
	EnableKing  bool         `json:"enableKing"`
	Matrix      []MatrixCell `json:"matrix"`
	Mode        string       `json:"mode"`        // "online" or "offline"
	OllamaModel string       `json:"ollamaModel"` // For offline mode, e.g., "qwen3-coder"
}

// MatrixCell represents a cell in the workflow matrix
type MatrixCell struct {
	Stage   string `json:"stage"`
	Zone    string `json:"zone"`
	Persona string `json:"persona"`
	Enabled bool   `json:"enabled"`
}

// PersonaConfig represents persona configuration in .mission/config.json
type PersonaConfig struct {
	Enabled bool `json:"enabled"`
}

// ProjectConfig represents .mission/config.json
type ProjectConfig struct {
	Version     string                   `json:"version"`
	Audience    string                   `json:"audience"`
	Zones       []string                 `json:"zones"`
	King        bool                     `json:"king"`
	Matrix      []MatrixCell             `json:"matrix,omitempty"`
	Personas    map[string]PersonaConfig `json:"personas,omitempty"`
	Mode        string                   `json:"mode,omitempty"`        // "online" or "offline"
	OllamaModel string                   `json:"ollamaModel,omitempty"` // For offline mode
}

// PersonaResponse represents persona data returned by API
type PersonaResponse struct {
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	HasPrompt bool   `json:"hasPrompt"`
}

// UpdatePersonaRequest is the request body for updating a persona
type UpdatePersonaRequest struct {
	Enabled *bool `json:"enabled,omitempty"`
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

	// Handle import mode: just validate and add to global config
	if req.Import {
		// Validate .mission directory exists
		missionDir := filepath.Join(path, ".mission")
		if _, err := os.Stat(missionDir); os.IsNotExist(err) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "No .mission directory found. Cannot import - use create instead.",
			})
			return
		}
		// Note: config.json validation is optional - project may still work without it
	} else {
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
		_ = json.NewEncoder(configFile).Encode(matrixConfig)
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

		// Update project config with mode and ollamaModel if specified
		if req.Mode != "" || req.OllamaModel != "" {
			projectConfig, err := h.loadProjectConfig(path)
			if err == nil && projectConfig != nil {
				if req.Mode != "" {
					projectConfig.Mode = req.Mode
				}
				if req.OllamaModel != "" {
					projectConfig.OllamaModel = req.OllamaModel
				}
				_ = h.saveProjectConfig(path, projectConfig)
			}
		}
	}

	// Add to global config
	mode := req.Mode
	if mode == "" {
		mode = "online" // Default to online mode
	}
	project := Project{
		Path:        path,
		Name:        filepath.Base(path),
		LastOpened:  time.Now().UTC().Format(time.RFC3339),
		Mode:        mode,
		OllamaModel: req.OllamaModel,
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

// The 11 builtin persona IDs
var builtinPersonas = []string{
	"researcher", "designer", "architect", "developer", "debugger",
	"reviewer", "security", "tester", "qa", "docs", "devops",
}

// handlePersonas routes persona-related requests
func (h *ProjectsHandler) handlePersonas(w http.ResponseWriter, r *http.Request, projectPath, personaPath string) {
	// Expand ~ in project path
	if strings.HasPrefix(projectPath, "~") {
		home, _ := os.UserHomeDir()
		projectPath = filepath.Join(home, projectPath[1:])
	}

	// Check project exists
	missionDir := filepath.Join(projectPath, ".mission")
	if _, err := os.Stat(missionDir); os.IsNotExist(err) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found or not initialized"})
		return
	}

	// Route based on persona path:
	// "" or "/" -> list personas
	// "/{id}" -> get/update persona
	// "/{id}/prompt" -> get/update prompt
	personaPath = strings.TrimPrefix(personaPath, "/")
	parts := strings.Split(personaPath, "/")

	if personaPath == "" {
		// GET /api/projects/{path}/personas
		if r.Method == http.MethodGet {
			h.listPersonas(w, r, projectPath)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	personaID := parts[0]
	if len(parts) == 1 {
		// GET/PUT /api/projects/{path}/personas/{id}
		switch r.Method {
		case http.MethodGet:
			h.getPersona(w, r, projectPath, personaID)
		case http.MethodPut:
			h.updatePersona(w, r, projectPath, personaID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "prompt" {
		// GET/PUT /api/projects/{path}/personas/{id}/prompt
		switch r.Method {
		case http.MethodGet:
			h.getPersonaPrompt(w, r, projectPath, personaID)
		case http.MethodPut:
			h.updatePersonaPrompt(w, r, projectPath, personaID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// listPersonas returns all persona configurations for a project
func (h *ProjectsHandler) listPersonas(w http.ResponseWriter, r *http.Request, projectPath string) {
	config, err := h.loadProjectConfig(projectPath)
	if err != nil {
		// Return default config if not found
		config = &ProjectConfig{
			Personas: make(map[string]PersonaConfig),
		}
	}

	// Build response with all builtin personas
	personas := make([]PersonaResponse, 0, len(builtinPersonas))
	promptsDir := filepath.Join(projectPath, ".mission", "prompts")

	for _, id := range builtinPersonas {
		personaConfig, exists := config.Personas[id]
		enabled := true
		if exists {
			enabled = personaConfig.Enabled
		}

		// Check if prompt file exists
		promptPath := filepath.Join(promptsDir, id+".md")
		_, hasPrompt := os.Stat(promptPath)

		personas = append(personas, PersonaResponse{
			ID:        id,
			Enabled:   enabled,
			HasPrompt: hasPrompt == nil,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"personas": personas,
	})
}

// getPersona returns a single persona's configuration
func (h *ProjectsHandler) getPersona(w http.ResponseWriter, r *http.Request, projectPath, personaID string) {
	config, err := h.loadProjectConfig(projectPath)
	if err != nil {
		config = &ProjectConfig{Personas: make(map[string]PersonaConfig)}
	}

	personaConfig, exists := config.Personas[personaID]
	enabled := true
	if exists {
		enabled = personaConfig.Enabled
	}

	promptPath := filepath.Join(projectPath, ".mission", "prompts", personaID+".md")
	_, hasPrompt := os.Stat(promptPath)

	writeJSON(w, http.StatusOK, PersonaResponse{
		ID:        personaID,
		Enabled:   enabled,
		HasPrompt: hasPrompt == nil,
	})
}

// updatePersona updates a persona's configuration
func (h *ProjectsHandler) updatePersona(w http.ResponseWriter, r *http.Request, projectPath, personaID string) {
	var req UpdatePersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	config, err := h.loadProjectConfig(projectPath)
	if err != nil {
		config = &ProjectConfig{
			Version:  "1.0.0",
			Personas: make(map[string]PersonaConfig),
		}
	}

	if config.Personas == nil {
		config.Personas = make(map[string]PersonaConfig)
	}

	// Update enabled state if provided
	if req.Enabled != nil {
		config.Personas[personaID] = PersonaConfig{
			Enabled: *req.Enabled,
		}
	}

	if err := h.saveProjectConfig(projectPath, config); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save config"})
		return
	}

	// Return updated persona
	h.getPersona(w, r, projectPath, personaID)
}

// getPersonaPrompt returns the prompt content for a persona
func (h *ProjectsHandler) getPersonaPrompt(w http.ResponseWriter, r *http.Request, projectPath, personaID string) {
	promptPath := filepath.Join(projectPath, ".mission", "prompts", personaID+".md")

	content, err := os.ReadFile(promptPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Prompt not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read prompt"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"id":      personaID,
		"content": string(content),
	})
}

// updatePersonaPrompt updates the prompt content for a persona
func (h *ProjectsHandler) updatePersonaPrompt(w http.ResponseWriter, r *http.Request, projectPath, personaID string) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	promptPath := filepath.Join(projectPath, ".mission", "prompts", personaID+".md")

	// Ensure prompts directory exists
	promptsDir := filepath.Dir(promptPath)
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create prompts directory"})
		return
	}

	if err := os.WriteFile(promptPath, []byte(req.Content), 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to write prompt"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"id":      personaID,
		"content": req.Content,
	})
}

// loadProjectConfig loads .mission/config.json
func (h *ProjectsHandler) loadProjectConfig(projectPath string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectPath, ".mission", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// saveProjectConfig saves .mission/config.json
func (h *ProjectsHandler) saveProjectConfig(projectPath string, config *ProjectConfig) error {
	configPath := filepath.Join(projectPath, ".mission", "config.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// DirEntry represents a directory entry for browsing
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// BrowseResponse is the response for directory browsing
type BrowseResponse struct {
	Path    string     `json:"path"`
	Entries []DirEntry `json:"entries"`
}

// handleBrowse handles directory browsing requests
func (h *ProjectsHandler) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")

	// Default to home directory if no path specified
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get home directory"})
			return
		}
		path = home
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	// Clean and resolve the path
	path = filepath.Clean(path)

	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Path does not exist"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}

	if !info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Path is not a directory"})
		return
	}

	// Read directory contents
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read directory"})
		return
	}

	// Filter to directories only and exclude hidden files
	entries := make([]DirEntry, 0)
	for _, entry := range dirEntries {
		// Skip hidden files/directories (starting with .)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		// Only include directories
		if entry.IsDir() {
			entries = append(entries, DirEntry{
				Name:  entry.Name(),
				IsDir: true,
			})
		}
	}

	writeJSON(w, http.StatusOK, BrowseResponse{
		Path:    path,
		Entries: entries,
	})
}
