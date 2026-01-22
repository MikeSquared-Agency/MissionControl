package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mike/mission-control/api"
	"github.com/mike/mission-control/bridge"
	"github.com/mike/mission-control/manager"
	"github.com/mike/mission-control/terminal"
	"github.com/mike/mission-control/v4"
	"github.com/mike/mission-control/watcher"
	"github.com/mike/mission-control/ws"
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
		// Default to ../agents relative to executable or current directory
		execPath, err := os.Executable()
		if err == nil {
			*agentsDir = filepath.Join(filepath.Dir(execPath), "..", "agents")
		} else {
			*agentsDir = "./agents"
		}
	}

	// Verify agents directory exists
	if _, err := os.Stat(*agentsDir); os.IsNotExist(err) {
		// Try relative to current working directory
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

	// Create King bridge
	king := bridge.NewKing(absWorkDir)
	hub.SetKing(king)

	// Start listening for King events and broadcast to WebSocket
	go func() {
		for event := range king.Events() {
			data, _ := json.Marshal(event)
			hub.Notify(json.RawMessage(data))
		}
	}()

	// Create and start mission watcher if .mission/ exists
	var missionWatcher *watcher.Watcher
	if _, err := os.Stat(missionDir); err == nil {
		missionWatcher = watcher.NewWatcher(missionDir)
		if err := missionWatcher.Start(); err != nil {
			log.Printf("Warning: failed to start mission watcher: %v", err)
		} else {
			// Set mission state provider
			hub.SetMissionStateProvider(func() interface{} {
				return missionWatcher.GetCurrentState()
			})

			// Broadcast watcher events to WebSocket
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

	// Set v4 state provider for WebSocket initial sync
	hub.SetV4StateProvider(func() interface{} {
		return map[string]interface{}{
			"current_phase": v4Store.CurrentPhase(),
			"phases":        v4Store.GetPhases(),
			"tasks":         v4Store.ListTasks(nil, nil, nil, nil),
			"checkpoints":   v4Store.ListCheckpoints(),
		}
	})

	// Create API handler
	apiHandler := api.NewHandler(mgr)

	// Create King API handler
	kingHandler := api.NewKingHandler(king, absWorkDir)

	// Create Projects API handler
	projectsHandler := api.NewProjectsHandler()

	// Create Ollama API handler
	ollamaHandler := api.NewOllamaHandler()

	// Set up routes
	mux := http.NewServeMux()

	// Projects API routes (must be before generic /api/ handler)
	projectsHandler.RegisterRoutes(mux)

	// Ollama API routes (for offline mode)
	ollamaHandler.RegisterRoutes(mux)

	// King API routes (King and gates)
	kingHandler.RegisterRoutes(mux)

	// v4 API routes (register first for specificity)
	v4Handler.RegisterRoutes(mux)

	// Existing API routes
	mux.Handle("/api/", apiHandler.Routes())

	// WebSocket endpoint
	mux.HandleFunc("/ws", hub.HandleWebSocket)

	// Terminal PTY WebSocket endpoint
	ptyHandler := terminal.NewPTYHandler()
	mux.HandleFunc("/api/terminal", ptyHandler.HandleWebSocket)

	// Simple status page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>MissionControl</title></head>
<body>
<h1>MissionControl</h1>
<p>Orchestrator is running.</p>

<h2>Runtime Endpoints</h2>
<ul>
<li><a href="/api/health">GET /api/health</a> - Health check</li>
<li><a href="/api/agents">GET /api/agents</a> - List agents</li>
<li>POST /api/agents - Spawn agent</li>
<li>DELETE /api/agents/:id - Kill agent</li>
<li>POST /api/agents/:id/message - Send message</li>
<li>WS /ws - WebSocket connection</li>
</ul>

<h2>v4 Workflow Endpoints</h2>
<ul>
<li><a href="/api/phases">GET /api/phases</a> - Current phase and all phases</li>
<li><a href="/api/tasks">GET /api/tasks</a> - List tasks</li>
<li>POST /api/tasks - Create task</li>
<li>GET /api/tasks/:id - Get task</li>
<li>PUT /api/tasks/:id/status - Update task status</li>
</ul>

<h2>v4 Knowledge Endpoints</h2>
<ul>
<li>POST /api/handoffs - Submit handoff</li>
<li><a href="/api/checkpoints">GET /api/checkpoints</a> - List checkpoints</li>
<li>POST /api/checkpoints - Create checkpoint</li>
<li>GET /api/checkpoints/:id - Get checkpoint</li>
<li>GET /api/budgets/:worker_id - Get token budget</li>
</ul>

<h2>v4 Strategy Endpoints</h2>
<ul>
<li>GET /api/gates/:id - Get gate status</li>
<li>POST /api/gates/:id/approve - Approve gate</li>
</ul>

<h2>Ollama Endpoints</h2>
<ul>
<li><a href="/api/ollama/status">GET /api/ollama/status</a> - Ollama status and models</li>
<li><a href="/api/ollama/models">GET /api/ollama/models</a> - List Ollama models</li>
</ul>

<h2>v5 King Endpoints</h2>
<ul>
<li>POST /api/king/start - Start King process</li>
<li>POST /api/king/stop - Stop King process</li>
<li><a href="/api/king/status">GET /api/king/status</a> - King status</li>
<li>POST /api/king/message - Send message to King</li>
<li>GET /api/mission/gates/:phase - Check gate status</li>
<li>POST /api/mission/gates/:phase/approve - Approve gate</li>
</ul>

<h2>WebSocket Events (v5)</h2>
<ul>
<li>mission_state - Initial mission state sync</li>
<li>king_status - King running status</li>
<li>phase_changed - Phase transitioned</li>
<li>task_created - New task created</li>
<li>task_updated - Task status changed</li>
<li>worker_spawned - Worker started</li>
<li>worker_completed - Worker finished</li>
<li>gate_ready - Gate criteria met</li>
<li>gate_approved - Gate approved</li>
<li>findings_ready - New findings available</li>
</ul>

<h2>WebSocket Commands (v5)</h2>
<ul>
<li>king_message - Send message to King</li>
</ul>

<h2>Spawn Example</h2>
<pre>
curl -X POST http://localhost:%d/api/agents \
  -H "Content-Type: application/json" \
  -d '{"type": "python", "task": "list files in current directory", "agent": "v0_minimal"}'
</pre>
</body>
</html>`, *port)
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting MissionControl on http://localhost%s", addr)
	log.Printf("WebSocket available at ws://localhost%s/ws", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
