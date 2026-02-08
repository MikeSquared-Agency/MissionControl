package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/DarlingtonDeveloper/MissionControl/api"
	"github.com/DarlingtonDeveloper/MissionControl/manager"
	"github.com/DarlingtonDeveloper/MissionControl/terminal"
	"github.com/DarlingtonDeveloper/MissionControl/v4"
	"github.com/DarlingtonDeveloper/MissionControl/watcher"
	"github.com/DarlingtonDeveloper/MissionControl/ws"
)

func main() {
	// Parse flags
	port := flag.Int("port", 8080, "Port to listen on")
	agentsDir := flag.String("agents", "", "Path to agents directory")
	workDir := flag.String("workdir", "", "Working directory with .mission/")
	flag.Parse()

	// Determine working directory
	if *workDir == "" {
		cwd, _ := os.Getwd()
		*workDir = cwd
	}
	absWorkDir, _ := filepath.Abs(*workDir)
	missionDir := filepath.Join(absWorkDir, ".mission")

	// Determine agents directory
	if *agentsDir == "" {
		execPath, err := os.Executable()
		if err == nil {
			*agentsDir = filepath.Join(filepath.Dir(execPath), "..", "agents")
		} else {
			*agentsDir = "./agents"
		}
	}

	// Verify agents directory exists
	if _, err := os.Stat(*agentsDir); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		altPath := filepath.Join(cwd, "..", "agents")
		if _, err := os.Stat(altPath); err == nil {
			*agentsDir = altPath
		} else {
			log.Printf("Warning: agents directory not found at %s", *agentsDir)
		}
	}

	absAgentsDir, _ := filepath.Abs(*agentsDir)
	log.Printf("Agents directory: %s", absAgentsDir)
	log.Printf("Working directory: %s", absWorkDir)

	// Create manager
	mgr := manager.NewManager(absAgentsDir)

	// Create WebSocket hub
	hub := ws.NewHub(mgr)
	go hub.Run()

	// Create and start mission watcher if .mission/ exists
	if _, err := os.Stat(missionDir); err == nil {
		missionWatcher := watcher.NewWatcher(missionDir)
		if err := missionWatcher.Start(); err != nil {
			log.Printf("Warning: failed to start mission watcher: %v", err)
		} else {
			hub.SetMissionStateProvider(func() interface{} {
				return missionWatcher.GetCurrentState()
			})
			go func() {
				for event := range missionWatcher.Events() {
					data, _ := json.Marshal(event)
					hub.Notify(json.RawMessage(data))
				}
			}()
		}
	} else {
		log.Printf("No .mission/ directory found - run 'mc init' to create one")
	}

	// Create v4 store and handler
	v4Store := v4.NewStore()
	v4Handler := v4.NewHandler(v4Store, hub)

	hub.SetV4StateProvider(func() interface{} {
		return map[string]interface{}{
			"current_stage": v4Store.CurrentStage(),
			"stages":        v4Store.GetStages(),
			"tasks":         v4Store.ListTasks(nil, nil, nil, nil),
			"checkpoints":   v4Store.ListCheckpoints(),
		}
	})

	// Create API handlers
	apiHandler := api.NewHandler(mgr)
	projectsHandler := api.NewProjectsHandler()
	ollamaHandler := api.NewOllamaHandler()

	// Set up routes
	mux := http.NewServeMux()

	projectsHandler.RegisterRoutes(mux)
	ollamaHandler.RegisterRoutes(mux)
	v4Handler.RegisterRoutes(mux)
	mux.Handle("/api/", apiHandler.Routes())
	mux.HandleFunc("/ws", hub.HandleWebSocket)

	// Terminal PTY WebSocket endpoint
	ptyHandler := terminal.NewPTYHandler()
	mux.HandleFunc("/api/terminal", ptyHandler.HandleWebSocket)

	// Health check at root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": "mc-orchestrator",
			"status":  "ok",
		})
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Println("╔══════════════════════════════════════════╗")
	log.Println("║       MissionControl Orchestrator        ║")
	log.Println("║         Kai is the King now 👑           ║")
	log.Println("╚══════════════════════════════════════════╝")
	log.Printf("API endpoints:")
	log.Printf("  GET  /api/health          - Health check")
	log.Printf("  GET  /api/agents          - List agents")
	log.Printf("  POST /api/agents          - Spawn agent")
	log.Printf("  DEL  /api/agents/:id      - Kill agent")
	log.Printf("  POST /api/agents/:id/message  - Message agent")
	log.Printf("  POST /api/agents/:id/respond  - Respond to agent")
	log.Printf("  GET  /api/zones           - List zones")
	log.Printf("  GET  /api/stages          - v4 stages")
	log.Printf("  GET  /api/tasks           - v4 tasks")
	log.Printf("  WS   /ws                  - WebSocket events")
	log.Printf("Listening on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
