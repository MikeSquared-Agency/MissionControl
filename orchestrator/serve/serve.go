// Package serve implements the mc serve command — the single binary entry
// point for the MissionControl orchestrator.
package serve

import (
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
	mux := http.NewServeMux()

	// WebSocket
	mux.HandleFunc("/ws", hub.HandleWebSocket)

	// Health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "version": "6.1"})
	})

	// Status — full mission snapshot
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, buildState(missionDir, trk, acc))
	})

	// Tokens
	mux.HandleFunc("/api/tokens", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, acc.Summary())
	})

	// Workers
	mux.HandleFunc("/api/workers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, 200, trk.List())
			return
		}
		http.Error(w, "method not allowed", 405)
	})

	// Placeholders
	mux.HandleFunc("/api/requirements", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []interface{}{})
	})
	mux.HandleFunc("/api/requirements/coverage", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"total": 0, "implemented": 0, "coverage": 0.0})
	})
	mux.HandleFunc("/api/specs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []interface{}{})
	})
	mux.HandleFunc("/api/specs/orphans", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []interface{}{})
	})
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

			ocHandler := openclaw.NewHandler(bridge)
			ocHandler.RegisterRoutes(mux)
			ocHandler.RegisterChatAlias(mux)
		}
	}
	if !bridgeConnected {
		mux.HandleFunc("/api/openclaw/status", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, map[string]interface{}{"connected": false})
		})
		mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 501, map[string]string{"error": "OpenClaw bridge not configured. Set OPENCLAW_GATEWAY and OPENCLAW_TOKEN."})
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
	}
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

	// Gates
	if data, err := os.ReadFile(filepath.Join(missionPath, "gates.json")); err == nil {
		var gates interface{}
		if json.Unmarshal(data, &gates) == nil {
			state["gates"] = gates
		}
	}

	// Tasks (JSONL)
	if tasks, err := readJSONL(filepath.Join(missionPath, "tasks.jsonl")); err == nil {
		state["tasks"] = tasks
	}
	// Fallback to tasks.json
	if _, ok := state["tasks"]; !ok {
		if tasks, err := readJSONL(filepath.Join(missionPath, "tasks.json")); err == nil {
			state["tasks"] = tasks
		}
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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
