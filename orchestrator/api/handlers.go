package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// --- Helpers ---

func readJSONL(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var results []map[string]interface{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue // skip malformed lines
		}
		results = append(results, obj)
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	return results, scanner.Err()
}

func readJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (s *Server) runMC(args ...string) (string, error) {
	cmd := exec.Command("mc", args...)
	cmd.Dir = s.missionDir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func respondError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func (s *Server) statePath(parts ...string) string {
	elems := append([]string{s.missionDir, ".mission", "state"}, parts...)
	return filepath.Join(elems...)
}

func (s *Server) missionPath(parts ...string) string {
	elems := append([]string{s.missionDir, ".mission"}, parts...)
	return filepath.Join(elems...)
}

// validateTaskID rejects IDs with path traversal characters.
func validateTaskID(id string) bool {
	return id != "" && !strings.Contains(id, "..") && !strings.ContainsAny(id, "/\\")
}

// handleTaskFindings serves .mission/findings/{id}.md as text/markdown.
func (s *Server) handleTaskFindings(w http.ResponseWriter, r *http.Request, id string) {
	if !validateTaskID(id) {
		respondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}
	path := s.missionPath("findings", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, http.StatusNotFound, "findings not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to read findings")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleTaskBriefing serves .mission/handoffs/{id}-briefing.json as application/json.
func (s *Server) handleTaskBriefing(w http.ResponseWriter, r *http.Request, id string) {
	if !validateTaskID(id) {
		respondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}
	path := s.missionPath("handoffs", id+"-briefing.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, http.StatusNotFound, "briefing not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to read briefing")
		return
	}
	if !json.Valid(data) {
		respondError(w, http.StatusInternalServerError, "briefing file contains invalid JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// --- GET handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Version: "6.1"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{}

	// Read current stage
	var stage map[string]interface{}
	if err := readJSON(s.statePath("stage.json"), &stage); err == nil {
		result["stage"] = stage
	} else {
		// Fallback: try stages.jsonl and use last entry
		if entries, err := readJSONL(s.statePath("stages.jsonl")); err == nil && len(entries) > 0 {
			result["stage"] = entries[len(entries)-1]
		} else {
			result["stage"] = nil
		}
	}

	// Read tasks
	tasks, _ := readJSONL(s.statePath("tasks.jsonl"))
	result["tasks"] = tasks

	// Read gates
	var gates map[string]interface{}
	if err := readJSON(s.statePath("gates.json"), &gates); err != nil {
		gates = map[string]interface{}{}
	}
	result["gates"] = gates

	// Read zones
	var zones interface{}
	if err := readJSON(s.statePath("zones.json"), &zones); err != nil {
		zones = deriveZones(tasks)
	}
	result["zones"] = zones

	// Workers and tokens from injected deps
	if s.tracker != nil {
		result["workers"] = s.tracker.List()
	} else {
		result["workers"] = []interface{}{}
	}
	if s.tokens != nil {
		result["tokens"] = s.tokens.Summary()
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleCreateTask(w, r)
		return
	}

	tasks, err := readJSONL(s.statePath("tasks.jsonl"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply filters
	q := r.URL.Query()
	stage := q.Get("stage")
	zone := q.Get("zone")
	status := q.Get("status")
	persona := q.Get("persona")

	var filtered []map[string]interface{}
	for _, t := range tasks {
		if stage != "" && fmt.Sprint(t["stage"]) != stage {
			continue
		}
		if zone != "" && fmt.Sprint(t["zone"]) != zone {
			continue
		}
		if status != "" && fmt.Sprint(t["status"]) != status {
			continue
		}
		if persona != "" && fmt.Sprint(t["persona"]) != persona {
			continue
		}
		filtered = append(filtered, t)
	}
	if filtered == nil {
		filtered = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, filtered)
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodPatch {
		s.handleUpdateTask(w, r, id)
		return
	}

	tasks, err := readJSONL(s.statePath("tasks.jsonl"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, t := range tasks {
		if fmt.Sprint(t["id"]) == id {
			writeJSON(w, http.StatusOK, t)
			return
		}
	}
	respondError(w, http.StatusNotFound, "task not found")
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	tasks, _ := readJSONL(s.statePath("tasks.jsonl"))
	writeJSON(w, http.StatusOK, BuildGraph(tasks))
}

// BuildGraph constructs a GraphResponse from raw task data.
// Exported so serve.go can call it from buildState().
func BuildGraph(tasks []map[string]interface{}) GraphResponse {
	var nodes []GraphNode
	var edges []GraphEdge
	blockedCount := 0
	readyCount := 0

	for _, t := range tasks {
		id := fmt.Sprint(t["id"])
		status := fmt.Sprint(t["status"])
		name := fmt.Sprint(t["name"])
		persona := ""
		if p, ok := t["persona"].(string); ok {
			persona = p
		}
		workerID := ""
		if w, ok := t["worker_id"].(string); ok {
			workerID = w
		}

		nodes = append(nodes, GraphNode{
			ID:       id,
			Name:     name,
			Title:    name,
			Type:     "task",
			Status:   status,
			Stage:    fmt.Sprint(t["stage"]),
			Zone:     fmt.Sprint(t["zone"]),
			Persona:  persona,
			WorkerID: workerID,
		})

		if status == "blocked" {
			blockedCount++
		}
		if status == "pending" {
			if deps, ok := t["dependencies"].([]interface{}); !ok || len(deps) == 0 {
				readyCount++
			}
		}

		if deps, ok := t["dependencies"].([]interface{}); ok {
			for _, d := range deps {
				depStr := fmt.Sprint(d)
				edges = append(edges, GraphEdge{
					From:   depStr,
					To:     id,
					Source: depStr,
					Target: id,
					Type:   "blocks",
				})
			}
		}
	}
	if nodes == nil {
		nodes = []GraphNode{}
	}
	if edges == nil {
		edges = []GraphEdge{}
	}

	return GraphResponse{
		Nodes:        nodes,
		Edges:        edges,
		CriticalPath: []string{},
		BlockedCount: blockedCount,
		ReadyCount:   readyCount,
	}
}

func (s *Server) handleWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleSpawnWorker(w, r)
		return
	}
	if s.tracker != nil {
		writeJSON(w, http.StatusOK, s.tracker.List())
	} else {
		writeJSON(w, http.StatusOK, []interface{}{})
	}
}

func (s *Server) handleWorkerByID(w http.ResponseWriter, r *http.Request, id string) {
	if s.tracker == nil {
		respondError(w, http.StatusNotFound, "worker not found")
		return
	}
	worker, ok := s.tracker.Get(id)
	if !ok {
		respondError(w, http.StatusNotFound, "worker not found")
		return
	}
	writeJSON(w, http.StatusOK, worker)
}

func (s *Server) handleGates(w http.ResponseWriter, r *http.Request) {
	var gates map[string]interface{}
	if err := readJSON(s.statePath("gates.json"), &gates); err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]interface{}{})
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gates)
}

func (s *Server) handleGateByStage(w http.ResponseWriter, r *http.Request, stage string) {
	var gates map[string]interface{}
	if err := readJSON(s.statePath("gates.json"), &gates); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	gate, ok := gates[stage]
	if !ok {
		respondError(w, http.StatusNotFound, "gate not found")
		return
	}
	writeJSON(w, http.StatusOK, gate)
}

func (s *Server) handleZones(w http.ResponseWriter, r *http.Request) {
	var zones interface{}
	if err := readJSON(s.statePath("zones.json"), &zones); err != nil {
		tasks, _ := readJSONL(s.statePath("tasks.jsonl"))
		zones = deriveZones(tasks)
	}
	writeJSON(w, http.StatusOK, zones)
}

func deriveZones(tasks []map[string]interface{}) []string {
	seen := map[string]bool{}
	for _, t := range tasks {
		if z, ok := t["zone"].(string); ok && z != "" {
			seen[z] = true
		}
	}
	zones := make([]string, 0, len(seen))
	for z := range seen {
		zones = append(zones, z)
	}
	sort.Strings(zones)
	return zones
}

func (s *Server) handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleCreateCheckpoint(w, r)
		return
	}

	dir := s.missionPath("checkpoints")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []CheckpointInfo{})
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var checkpoints []CheckpointInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		ts := strings.TrimPrefix(name, "cp-")
		checkpoints = append(checkpoints, CheckpointInfo{ID: name, Timestamp: ts})
	}
	if checkpoints == nil {
		checkpoints = []CheckpointInfo{}
	}

	writeJSON(w, http.StatusOK, checkpoints)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	category := q.Get("category")
	actor := q.Get("actor")

	entries, err := readJSONL(s.missionPath("audit.jsonl"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter
	var filtered []map[string]interface{}
	for _, e := range entries {
		if category != "" && fmt.Sprint(e["category"]) != category {
			continue
		}
		if actor != "" && fmt.Sprint(e["actor"]) != actor {
			continue
		}
		filtered = append(filtered, e)
	}

	// Paginate
	total := len(filtered)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := filtered[offset:end]
	if page == nil {
		page = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": page,
		"total":   total,
		"offset":  offset,
		"limit":   limit,
	})
}

func (s *Server) handleTokens(w http.ResponseWriter, r *http.Request) {
	if s.tokens != nil {
		writeJSON(w, http.StatusOK, s.tokens.Summary())
	} else {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
	}
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleProjectSwitch(w, r)
		return
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".mission-control", "config.json")
	var config interface{}
	if err := readJSON(configPath, &config); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"projects": []interface{}{}})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) handleOpenClawStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, OpenClawStatus{Connected: false})
}

func (s *Server) handleRequirements(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleRequirementsCoverage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, RequirementsCoverage{Total: 0, Implemented: 0, Coverage: 0.0})
}

func (s *Server) loadSpecs() []SpecInfo {
	specsDir := s.missionPath("specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return []SpecInfo{}
	}

	tasks, _ := readJSONL(s.statePath("tasks.jsonl"))

	var specs []SpecInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := e.Name()
		id := strings.TrimSuffix(name, ".md")

		// Extract title from first # heading
		title := strings.ReplaceAll(id, "-", " ")
		if data, err := os.ReadFile(filepath.Join(specsDir, name)); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "# ") {
					title = strings.TrimPrefix(line, "# ")
					break
				}
			}
		}

		// Find linked tasks
		var linked []string
		for _, t := range tasks {
			tid := fmt.Sprint(t["id"])
			if spec, ok := t["spec"].(string); ok && spec == id {
				linked = append(linked, tid)
			}
		}
		if linked == nil {
			linked = []string{}
		}

		specs = append(specs, SpecInfo{
			ID:          id,
			Title:       title,
			Filename:    name,
			LinkedTasks: linked,
			IsOrphan:    len(linked) == 0,
		})
	}
	if specs == nil {
		specs = []SpecInfo{}
	}
	return specs
}

func (s *Server) handleSpecs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.loadSpecs())
}

func (s *Server) handleSpecsOrphans(w http.ResponseWriter, r *http.Request) {
	all := s.loadSpecs()
	var orphans []SpecInfo
	for _, sp := range all {
		if sp.IsOrphan {
			orphans = append(orphans, sp)
		}
	}
	if orphans == nil {
		orphans = []SpecInfo{}
	}
	writeJSON(w, http.StatusOK, orphans)
}

func (s *Server) handleSpecByID(w http.ResponseWriter, r *http.Request, id string) {
	if !validateTaskID(id) {
		respondError(w, http.StatusBadRequest, "invalid spec ID")
		return
	}
	path := s.missionPath("specs", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, http.StatusNotFound, "spec not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to read spec")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// --- POST/PATCH handlers ---

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "OpenClaw bridge not configured")
}

func (s *Server) handleGateApprove(w http.ResponseWriter, r *http.Request, stage string) {
	out, err := s.runMC("gate", "approve", stage)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc gate approve failed: %s", out))
		return
	}
	if s.hub != nil {
		s.hub.BroadcastRaw("gates", "gate_approved", map[string]string{"stage": stage})
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleGateReject(w http.ResponseWriter, r *http.Request, stage string) {
	var req GateActionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	args := []string{"gate", "reject", stage}
	if req.Reason != "" {
		args = append(args, "--reason", req.Reason)
	}

	out, err := s.runMC(args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc gate reject failed: %s", out))
		return
	}
	if s.hub != nil {
		s.hub.BroadcastRaw("gates", "gate_rejected", map[string]string{"stage": stage, "reason": req.Reason})
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleSpawnWorker(w http.ResponseWriter, r *http.Request) {
	out, err := s.runMC("worker", "spawn")
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc worker spawn failed: %s", out))
		return
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleKillWorker(w http.ResponseWriter, r *http.Request, id string) {
	out, err := s.runMC("worker", "kill", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc worker kill failed: %s", out))
		return
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleCreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	out, err := s.runMC("checkpoint")
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc checkpoint failed: %s", out))
		return
	}
	writeJSON(w, http.StatusCreated, CommandResult{Success: true, Output: out})
}

func (s *Server) handleRestartCheckpoint(w http.ResponseWriter, r *http.Request, id string) {
	out, err := s.runMC("checkpoint", "restart", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc checkpoint restart failed: %s", out))
		return
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	args := []string{"task", "create", req.Title}
	if req.Stage != "" {
		args = append(args, "--stage", req.Stage)
	}
	if req.Zone != "" {
		args = append(args, "--zone", req.Zone)
	}

	out, err := s.runMC(args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc task create failed: %s", out))
		return
	}
	writeJSON(w, http.StatusCreated, CommandResult{Success: true, Output: out})
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	args := []string{"task", "update", id}
	if req.Status != "" {
		args = append(args, "--status", req.Status)
	}
	if req.Stage != "" {
		args = append(args, "--stage", req.Stage)
	}

	out, err := s.runMC(args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc task update failed: %s", out))
		return
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleTaskDependencies(w http.ResponseWriter, r *http.Request, id string) {
	var req TaskDepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	action := "add"
	if req.Action == "remove" {
		action = "remove"
	}

	out, err := s.runMC("task", "dep", action, id, req.DepID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc task dep failed: %s", out))
		return
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleStageOverride(w http.ResponseWriter, r *http.Request) {
	var req StageOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Stage == "" {
		respondError(w, http.StatusBadRequest, "stage is required")
		return
	}

	out, err := s.runMC("stage", "set", req.Stage)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("mc stage set failed: %s", out))
		return
	}
	writeJSON(w, http.StatusOK, CommandResult{Success: true, Output: out})
}

func (s *Server) handleProjectSwitch(w http.ResponseWriter, r *http.Request) {
	var req ProjectSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Validate path has .mission directory
	if _, err := os.Stat(filepath.Join(req.Path, ".mission")); err != nil {
		respondError(w, http.StatusBadRequest, "path does not contain a .mission directory")
		return
	}

	s.missionDir = req.Path
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"missionDir": s.missionDir,
	})
}
