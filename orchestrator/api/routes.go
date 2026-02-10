package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/DarlingtonDeveloper/MissionControl/tokens"
	"github.com/DarlingtonDeveloper/MissionControl/tracker"
)

// Server holds dependencies for the API.
// All external dependencies are injected via interfaces.
type Server struct {
	missionDir string
	hub        HubBroadcaster
	tracker    TrackerReader
	tokens     TokenReader
}

// HubBroadcaster is satisfied by ws.Hub
type HubBroadcaster interface {
	BroadcastRaw(topic, eventType string, data interface{})
}

// TrackerReader is satisfied by tracker.Tracker
type TrackerReader interface {
	List() []*tracker.TrackedProcess
	Get(workerID string) (*tracker.TrackedProcess, bool)
}

// TokenReader is satisfied by tokens.Accumulator
type TokenReader interface {
	Summary() tokens.TokenSummary
}

// NewServer creates a new API server.
func NewServer(missionDir string, hub HubBroadcaster, tracker TrackerReader, tokens TokenReader) *Server {
	return &Server{
		missionDir: missionDir,
		hub:        hub,
		tracker:    tracker,
		tokens:     tokens,
	}
}

// Routes returns the HTTP handler with all API routes.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/api/health", s.methodGET(s.handleHealth))

	// Status
	mux.HandleFunc("/api/status", s.methodGET(s.handleStatus))

	// Tasks
	mux.HandleFunc("/api/tasks", s.handleTasksRouter)
	mux.HandleFunc("/api/tasks/", s.handleTaskRouter)

	// Graph
	mux.HandleFunc("/api/graph", s.methodGET(s.handleGraph))

	// Workers
	mux.HandleFunc("/api/workers", s.handleWorkersRouter)
	mux.HandleFunc("/api/workers/", s.handleWorkerRouter)

	// Gates
	mux.HandleFunc("/api/gates", s.methodGET(s.handleGates))
	mux.HandleFunc("/api/gates/", s.handleGateRouter)

	// Zones
	mux.HandleFunc("/api/zones", s.methodGET(s.handleZones))

	// Checkpoints
	mux.HandleFunc("/api/checkpoints", s.handleCheckpointsRouter)
	mux.HandleFunc("/api/checkpoints/", s.handleCheckpointRouter)

	// Audit
	mux.HandleFunc("/api/audit", s.methodGET(s.handleAudit))

	// Tokens
	mux.HandleFunc("/api/tokens", s.methodGET(s.handleTokens))

	// Projects (new endpoint for reading config)
	mux.HandleFunc("/api/projects", s.handleProjectsRouter)

	// Chat
	mux.HandleFunc("/api/chat", s.methodPOST(s.handleChat))

	// Stages
	mux.HandleFunc("/api/stages/override", s.methodPOST(s.handleStageOverride))

	// Placeholders
	mux.HandleFunc("/api/openclaw/status", s.methodGET(s.handleOpenClawStatus))
	mux.HandleFunc("/api/requirements", s.methodGET(s.handleRequirements))
	mux.HandleFunc("/api/requirements/coverage", s.methodGET(s.handleRequirementsCoverage))
	mux.HandleFunc("/api/specs", s.methodGET(s.handleSpecs))
	mux.HandleFunc("/api/specs/orphans", s.methodGET(s.handleSpecsOrphans))

	return mux
}

// --- Routers for path-based dispatch ---

func (s *Server) handleTasksRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleTasks(w, r)
	case http.MethodPost:
		s.handleCreateTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTaskRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}
	id := parts[0]

	if len(parts) > 1 {
		switch parts[1] {
		case "dependencies":
			if r.Method == http.MethodPost {
				s.handleTaskDependencies(w, r, id)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		case "findings":
			if r.Method == http.MethodGet {
				s.handleTaskFindings(w, r, id)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		case "briefing":
			if r.Method == http.MethodGet {
				s.handleTaskBriefing(w, r, id)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		s.handleTaskByID(w, r, id)
	case http.MethodPatch:
		s.handleUpdateTask(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkersRouter(w http.ResponseWriter, r *http.Request) {
	// Check for /api/workers/spawn
	if r.URL.Path == "/api/workers/spawn" && r.Method == http.MethodPost {
		s.handleSpawnWorker(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleWorkers(w, r)
	case http.MethodPost:
		s.handleSpawnWorker(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWorkerRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/workers/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "worker ID required")
		return
	}

	id := parts[0]

	// POST /api/workers/spawn
	if id == "spawn" && r.Method == http.MethodPost {
		s.handleSpawnWorker(w, r)
		return
	}

	if len(parts) > 1 && parts[1] == "kill" && r.Method == http.MethodPost {
		s.handleKillWorker(w, r, id)
		return
	}

	if r.Method == http.MethodGet {
		s.handleWorkerByID(w, r, id)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleGateRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/gates/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "stage required")
		return
	}

	stage := parts[0]

	if len(parts) > 1 {
		switch parts[1] {
		case "approve":
			if r.Method == http.MethodPost {
				s.handleGateApprove(w, r, stage)
				return
			}
		case "reject":
			if r.Method == http.MethodPost {
				s.handleGateReject(w, r, stage)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodGet {
		s.handleGateByStage(w, r, stage)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleCheckpointsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleCheckpoints(w, r)
	case http.MethodPost:
		s.handleCreateCheckpoint(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCheckpointRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/checkpoints/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "checkpoint ID required")
		return
	}

	id := parts[0]
	if len(parts) > 1 && parts[1] == "restart" && r.Method == http.MethodPost {
		s.handleRestartCheckpoint(w, r, id)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func (s *Server) handleProjectsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleProjects(w, r)
	case http.MethodPost:
		// Check path for /api/projects/switch
		if r.URL.Path == "/api/projects/switch" {
			s.handleProjectSwitch(w, r)
			return
		}
		s.handleProjectSwitch(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Method helpers ---

func (s *Server) methodGET(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

func (s *Server) methodPOST(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// writeJSON writes a JSON response with the given status code.
// Used by handlers and also by projects.go.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
