package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mike/agent-orchestra/api"
	"github.com/mike/agent-orchestra/manager"
	"github.com/mike/agent-orchestra/ws"
)

func main() {
	// Parse flags
	port := flag.Int("port", 8080, "Port to listen on")
	agentsDir := flag.String("agents", "", "Path to agents directory")
	flag.Parse()

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

	// Create manager
	mgr := manager.NewManager(absAgentsDir)

	// Create WebSocket hub
	hub := ws.NewHub(mgr)
	go hub.Run()

	// Create API handler
	apiHandler := api.NewHandler(mgr)

	// Set up routes
	mux := http.NewServeMux()

	// API routes
	mux.Handle("/api/", apiHandler.Routes())

	// WebSocket endpoint
	mux.HandleFunc("/ws", hub.HandleWebSocket)

	// Simple status page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Agent Orchestra</title></head>
<body>
<h1>Agent Orchestra</h1>
<p>Orchestrator is running.</p>
<h2>Endpoints</h2>
<ul>
<li><a href="/api/health">GET /api/health</a> - Health check</li>
<li><a href="/api/agents">GET /api/agents</a> - List agents</li>
<li>POST /api/agents - Spawn agent</li>
<li>DELETE /api/agents/:id - Kill agent</li>
<li>POST /api/agents/:id/message - Send message</li>
<li>WS /ws - WebSocket connection</li>
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
	log.Printf("Starting Agent Orchestra on http://localhost%s", addr)
	log.Printf("WebSocket available at ws://localhost%s/ws", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
