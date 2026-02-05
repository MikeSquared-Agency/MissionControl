package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

// Embed the dist directory - populated by copying web/dist during build
// Use 'make build' to ensure the UI is embedded
//
//go:embed all:dist
var webUI embed.FS

var (
	servePort            int
	serveWorkdir         string
	serveOpenClawGateway string
)

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to serve on")
	serveCmd.Flags().StringVarP(&serveWorkdir, "workdir", "w", "", "Working directory (default: current)")
	serveCmd.Flags().StringVar(&serveOpenClawGateway, "openclaw-gateway", "ws://127.0.0.1:18789", "OpenClaw gateway WebSocket URL")
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MissionControl server with embedded UI",
	Long: `Starts the MissionControl orchestrator with the embedded web UI.

This runs everything in a single process - no need to start
the orchestrator and UI separately.

Example:
  mc serve                    # Start on port 8080
  mc serve -p 3000            # Start on port 3000
  mc serve -w /path/to/proj   # Specify working directory`,
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workDir := serveWorkdir
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Make absolute
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("failed to resolve working directory: %w", err)
	}
	workDir = absWorkDir

	// Check for .mission directory
	missionDir := filepath.Join(workDir, ".mission")
	if _, err := os.Stat(missionDir); os.IsNotExist(err) {
		return fmt.Errorf(".mission/ not found in %s - run 'mc init' first", workDir)
	}

	log.Printf("Starting MissionControl server...")
	log.Printf("  Working directory: %s", workDir)
	log.Printf("  Port: %d", servePort)

	// Create HTTP mux
	mux := http.NewServeMux()

	// Try to serve embedded UI
	uiFS, err := fs.Sub(webUI, "dist")
	if err != nil {
		log.Printf("Warning: Embedded UI not available: %v", err)
		log.Printf("The UI may not have been embedded at build time.")
		log.Printf("Run 'make build' to build with embedded UI.")
	} else {
		// Serve static files from embedded FS
		fileServer := http.FileServer(http.FS(uiFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// For SPA routing: serve index.html for non-file requests
			path := r.URL.Path

			// Check if this looks like a file request (has extension)
			if filepath.Ext(path) != "" {
				fileServer.ServeHTTP(w, r)
				return
			}

			// For all other paths, serve index.html (SPA routing)
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
		log.Printf("  UI: Embedded (serving at /)")
	}

	// Health endpoint
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"` + version + `"}`))
	})

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", servePort),
		Handler: mux,
	}

	// Handle shutdown gracefully with auto-checkpoint (G3.3)
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")

		// Auto-checkpoint on shutdown
		if cp, err := createCheckpoint(missionDir, ""); err == nil {
			log.Printf("Shutdown checkpoint created: %s", cp.ID)
		}

		server.Close()
	}()

	fmt.Printf("\n  Open http://localhost:%d in your browser\n\n", servePort)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

var version = "dev"
