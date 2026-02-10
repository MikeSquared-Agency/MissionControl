// Package serve implements the mc serve command — the single binary entry
// point for the MissionControl orchestrator.
package serve

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/DarlingtonDeveloper/MissionControl/api"
	"github.com/DarlingtonDeveloper/MissionControl/openclaw"
	"github.com/DarlingtonDeveloper/MissionControl/tokens"
	"github.com/DarlingtonDeveloper/MissionControl/tracker"
	"github.com/DarlingtonDeveloper/MissionControl/watcher"
	"github.com/DarlingtonDeveloper/MissionControl/ws"
)

// Config holds serve command options.
type Config struct {
	Port       int
	MissionDir string
	APIOnly    bool // --api-only: disable file watcher + tracker (just serve API)
	Headless   bool // --headless: no dashboard, API only
}

// topicMap maps watcher event types to hub topics.
var topicMap = map[string]string{
	"stage_changed":         "stage",
	"task_created":          "task",
	"task_updated":          "task",
	"task_deleted":          "task",
	"worker_spawned":        "worker",
	"worker_completed":      "worker",
	"worker_status_changed": "worker",
	"gate_approved":         "gate",
	"gate_ready":            "gate",
	"zone_activity":         "zone",
	"checkpoint":            "checkpoint",
	"audit":                 "audit",
	"findings_ready":        "task",
	"handoff_created":       "task",
	"memory_updated":        "memory",
}

// Run starts the orchestrator server.
func Run(cfg Config) error {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	missionDir := cfg.MissionDir
	if missionDir == "" {
		missionDir = findMissionDir()
	}

	log.Printf("MissionControl orchestrator starting on :%d", cfg.Port)
	log.Printf("Mission directory: %s", missionDir)

	// --- Core components ---
	hub := ws.NewHub()
	go hub.Run()

	acc := tokens.NewAccumulator(0, func(workerID string, budget, used, remaining int) {
		hub.BroadcastRaw("token", "budget_warning", map[string]interface{}{
			"worker_id": workerID,
			"budget":    budget,
			"used":      used,
			"remaining": remaining,
		})
	})

	trk := tracker.NewTracker(missionDir, func(eventType string, proc *tracker.TrackedProcess) {
		topic := "worker"
		hub.BroadcastRaw(topic, eventType, proc)
	})

	// --- State provider for initial sync ---
	hub.SetStateProvider(func() interface{} {
		return buildState(missionDir, trk, acc)
	})

	// --- File watcher → hub bridge ---
	if !cfg.APIOnly {
		w := watcher.NewWatcher(filepath.Join(missionDir, ".mission"))
		if err := w.Start(); err != nil {
			log.Printf("Warning: file watcher failed to start: %v", err)
		} else {
			go bridgeWatcherToHub(w, hub)
			defer w.Stop()
		}

		trk.Start()
		defer trk.Stop()
	}

	// --- HTTP routes ---

	// Create API server (replaces all inline /api/* handlers)
	apiServer := api.NewServer(missionDir, hub, trk, acc)
	apiRoutes := apiServer.Routes()

	// Outer mux for WebSocket + OpenClaw (non-api.Server routes)
	mux := http.NewServeMux()

	// WebSocket
	mux.HandleFunc("/ws", hub.HandleWebSocket)

	// Delegate all /api/ routes to api.Server
	mux.Handle("/api/", apiRoutes)

	// --- OpenClaw bridge ---
	gatewayURL := os.Getenv("OPENCLAW_GATEWAY")
	gatewayToken := os.Getenv("OPENCLAW_TOKEN")
	bridgeConnected := false
	if gatewayURL != "" && gatewayToken != "" {
		bridge := openclaw.NewBridge(gatewayURL, gatewayToken)
		if err := bridge.Connect(); err != nil {
			log.Printf("Warning: OpenClaw bridge failed to connect: %v", err)
		} else {
			log.Printf("OpenClaw bridge connected to %s", gatewayURL)
			defer bridge.Close()
			bridgeConnected = true

			ocHandler := openclaw.NewHandler(bridge, hub, trk)
			ocHandler.RegisterRoutes(mux)
			ocHandler.RegisterChatAlias(mux)
			ocHandler.RegisterMCRoutes(mux)
		}
	}
	if !bridgeConnected {
		mux.HandleFunc("/api/openclaw/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{"connected": false})
		})
		mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(501)
			json.NewEncoder(w).Encode(map[string]string{"error": "OpenClaw bridge not configured. Set OPENCLAW_GATEWAY and OPENCLAW_TOKEN."})
		})
	}

	// Apply middleware
	handler := api.Chain(mux, api.CORSMiddleware, api.AuthMiddleware)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("Shutting down...")
		server.Close()
	}()

	log.Printf("Listening on %s", addr)
	return server.ListenAndServe()
}

// bridgeWatcherToHub reads watcher events and broadcasts them on the hub.
func bridgeWatcherToHub(w *watcher.Watcher, hub *ws.Hub) {
	for event := range w.Events() {
		topic, ok := topicMap[event.Type]
		if !ok {
			// Use first segment as topic
			parts := strings.SplitN(event.Type, ".", 2)
			topic = parts[0]
		}
		hub.BroadcastRaw(topic, event.Type, event.Data)

		// Handle findings_ready: mark the corresponding task as done
		if event.Type == "findings_ready" {
			if data, ok := event.Data.(map[string]interface{}); ok {
				if taskID, ok := data["task_id"].(string); ok {
					handleFindingsReady(w, taskID)
				}
			}
		}
	}
}

// handleFindingsReady marks a task as complete when its findings file appears.
func handleFindingsReady(w *watcher.Watcher, taskID string) {
	// Derive the .mission dir from the watcher (it watches <missionDir>/.mission)
	// We need the .mission/state/tasks.jsonl path
	state := w.GetCurrentState()
	// Find the task in watcher's known state to confirm it exists
	tasks, ok := state["tasks"].([]watcher.Task)
	if !ok {
		log.Printf("findings_ready: could not get tasks from watcher state")
		return
	}

	found := false
	for _, t := range tasks {
		if t.ID == taskID {
			if t.Status == "complete" {
				log.Printf("findings_ready: task %s already complete", taskID)
				return
			}
			found = true
			break
		}
	}
	if !found {
		log.Printf("findings_ready: task %s not found in watcher state", taskID)
		return
	}

	// Update via tasks.jsonl directly
	missionDir := w.MissionDir()
	tasksPath := filepath.Join(missionDir, "state", "tasks.jsonl")
	if err := markTaskComplete(tasksPath, taskID); err != nil {
		log.Printf("findings_ready: failed to mark task %s complete: %v", taskID, err)
		return
	}
	log.Printf("findings_ready: marked task %s as complete", taskID)
}

// taskEntry is a minimal task struct for JSONL read/write in the serve package.
type taskEntry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Stage     string   `json:"stage"`
	Zone      string   `json:"zone"`
	Persona   string   `json:"persona"`
	Status    string   `json:"status"`
	DependsOn []string `json:"depends_on,omitempty"`
	WorkerID  string   `json:"worker_id,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// markTaskComplete reads tasks.jsonl, sets the matching task to "complete", and writes back atomically.
func markTaskComplete(tasksPath, taskID string) error {
	f, err := os.Open(tasksPath)
	if err != nil {
		return err
	}

	var tasks []taskEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var t taskEntry
		if err := json.Unmarshal(line, &t); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	if err := scanner.Err(); err != nil {
		f.Close()
		return err
	}
	f.Close()

	found := false
	for i := range tasks {
		if tasks[i].ID == taskID {
			if tasks[i].Status == "complete" {
				return nil // idempotent
			}
			tasks[i].Status = "complete"
			tasks[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("task %s not found in tasks.jsonl", taskID)
	}

	// Atomic write via temp file + rename
	tmp, err := os.CreateTemp(filepath.Dir(tasksPath), ".tasks-*.jsonl")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, t := range tasks {
		data, err := json.Marshal(t)
		if err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return err
		}
		w.Write(data)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, tasksPath)
}

// buildState returns a full mission state snapshot.
func buildState(missionDir string, trk *tracker.Tracker, acc *tokens.Accumulator) map[string]interface{} {
	state := map[string]interface{}{
		"workers": trk.List(),
		"tokens":  acc.Summary(),
	}

	missionPath := filepath.Join(missionDir, ".mission", "state")

	// Stage
	if data, err := os.ReadFile(filepath.Join(missionPath, "stage.json")); err == nil {
		var stage interface{}
		if json.Unmarshal(data, &stage) == nil {
			state["stage"] = stage
		}
	}

	// Gates — file is {"gates": {...}}, unwrap to just the inner map
	if data, err := os.ReadFile(filepath.Join(missionPath, "gates.json")); err == nil {
		var wrapper map[string]interface{}
		if json.Unmarshal(data, &wrapper) == nil {
			if inner, ok := wrapper["gates"]; ok {
				state["gates"] = inner
			} else {
				state["gates"] = wrapper
			}
		}
	}

	// Tasks — try JSONL first, then JSON
	// File may be {"tasks": [...]} (wrapped) or bare JSONL
	if tasks, err := readJSONL(filepath.Join(missionPath, "tasks.jsonl")); err == nil && len(tasks) > 0 {
		state["tasks"] = tasks
	}
	if _, ok := state["tasks"]; !ok {
		if data, err := os.ReadFile(filepath.Join(missionPath, "tasks.json")); err == nil {
			var wrapper map[string]interface{}
			if json.Unmarshal(data, &wrapper) == nil {
				if inner, ok := wrapper["tasks"]; ok {
					state["tasks"] = inner
				}
			}
			// If not wrapped, try as array
			if _, ok := state["tasks"]; !ok {
				var arr []interface{}
				if json.Unmarshal(data, &arr) == nil {
					state["tasks"] = arr
				}
			}
		}
	}

	// Checkpoints
	cpDir := filepath.Join(missionDir, ".mission", "orchestrator", "checkpoints")
	if cpEntries, err := os.ReadDir(cpDir); err == nil {
		var checkpoints []map[string]interface{}
		for _, e := range cpEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".json")
			cpPath := filepath.Join(cpDir, e.Name())
			data, err := os.ReadFile(cpPath)
			if err != nil {
				continue
			}
			var cpData map[string]interface{}
			if json.Unmarshal(data, &cpData) != nil {
				continue
			}
			cp := map[string]interface{}{
				"id":         name,
				"created_at": name,
			}
			if stage, ok := cpData["stage"].(string); ok {
				cp["stage"] = stage
			}
			if tc, ok := cpData["task_count"].(float64); ok {
				cp["task_count"] = int(tc)
			}
			if auto, ok := cpData["auto"].(bool); ok {
				cp["auto"] = auto
			}
			checkpoints = append(checkpoints, cp)
		}
		if checkpoints == nil {
			checkpoints = []map[string]interface{}{}
		}
		state["checkpoints"] = checkpoints
	} else {
		state["checkpoints"] = []map[string]interface{}{}
	}

	// Audit
	if audit, err := readJSONL(filepath.Join(missionDir, ".mission", "audit.jsonl")); err == nil {
		state["audit"] = audit
	}

	// Graph — compute from tasks for initial sync
	if rawTasks, ok := state["tasks"].([]interface{}); ok {
		taskMaps := make([]map[string]interface{}, 0, len(rawTasks))
		for _, rt := range rawTasks {
			if m, ok := rt.(map[string]interface{}); ok {
				taskMaps = append(taskMaps, m)
			}
		}
		state["graph"] = api.BuildGraph(taskMaps)
	}

	return state
}

// readJSONL reads a JSONL file and returns a slice of objects.
func readJSONL(path string) ([]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []interface{}
	dec := json.NewDecoder(f)
	for dec.More() {
		var obj interface{}
		if err := dec.Decode(&obj); err != nil {
			continue
		}
		results = append(results, obj)
	}
	return results, nil
}

// findMissionDir tries to find the project directory.
func findMissionDir() string {
	// Check current directory
	if _, err := os.Stat(".mission"); err == nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	// Fall back to home
	home, _ := os.UserHomeDir()
	return home
}
