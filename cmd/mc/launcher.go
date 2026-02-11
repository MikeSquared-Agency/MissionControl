package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type launcherState int

const (
	stateSleeping launcherState = iota
	stateStarting
	stateReady
)

func (s launcherState) String() string {
	switch s {
	case stateSleeping:
		return "sleeping"
	case stateStarting:
		return "starting"
	case stateReady:
		return "ready"
	default:
		return "unknown"
	}
}

type launcher struct {
	mu          sync.Mutex
	state       launcherState
	port        int
	backendPort int
	idleTimeout time.Duration

	idleTimer *time.Timer
	serveCmd  *exec.Cmd
	proxy     *httputil.ReverseProxy
}

func newLauncher(port, backendPort int, idleTimeout time.Duration) *launcher {
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", backendPort))
	return &launcher{
		state:       stateSleeping,
		port:        port,
		backendPort: backendPort,
		idleTimeout: idleTimeout,
		proxy:       httputil.NewSingleHostReverseProxy(target),
	}
}

func (l *launcher) getState() launcherState {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.state
}

func (l *launcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health endpoint always responds immediately
	if r.URL.Path == "/api/health" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": l.getState().String()})
		return
	}

	// Wake endpoint
	if r.URL.Path == "/api/wake" && r.Method == http.MethodPost {
		l.triggerWake()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": l.getState().String()})
		return
	}

	state := l.getState()

	switch state {
	case stateSleeping:
		l.triggerWake()
		w.Header().Set("Retry-After", "5")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "starting",
			"message": "Services are waking up, please retry shortly",
		})

	case stateStarting:
		w.Header().Set("Retry-After", "3")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "starting",
			"message": "Services are starting, please retry shortly",
		})

	case stateReady:
		l.resetIdleTimer()
		l.proxy.ServeHTTP(w, r)
	}
}

func (l *launcher) triggerWake() {
	l.mu.Lock()
	if l.state != stateSleeping {
		l.mu.Unlock()
		return
	}
	l.state = stateStarting
	l.mu.Unlock()

	log.Println("launcher: waking up — starting services")
	go l.startServices()
}

func (l *launcher) startServices() {
	// Find the mc binary path (use our own executable)
	mcBin, err := os.Executable()
	if err != nil {
		log.Printf("launcher: could not resolve own executable: %v", err)
		l.mu.Lock()
		l.state = stateSleeping
		l.mu.Unlock()
		return
	}

	// Start mc serve as child process
	log.Printf("launcher: starting mc serve --port %d", l.backendPort)
	cmd := exec.Command(mcBin, "serve", "--port", fmt.Sprintf("%d", l.backendPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("launcher: failed to start mc serve: %v", err)
		l.mu.Lock()
		l.state = stateSleeping
		l.mu.Unlock()
		return
	}

	l.mu.Lock()
	l.serveCmd = cmd
	l.mu.Unlock()

	// Wait for child process in background (detect unexpected exits)
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("launcher: mc serve exited: %v", err)
		} else {
			log.Println("launcher: mc serve exited cleanly")
		}
		// If we're still in ready state, transition back to sleeping
		l.mu.Lock()
		if l.state == stateReady {
			log.Println("launcher: mc serve died unexpectedly, returning to sleeping")
			l.state = stateSleeping
			l.serveCmd = nil
			if l.idleTimer != nil {
				l.idleTimer.Stop()
			}
		}
		l.mu.Unlock()
	}()

	// Poll until both services are healthy
	if err := l.waitForHealthy(); err != nil {
		log.Printf("launcher: health check failed: %v", err)
		l.stopServices()
		return
	}

	l.mu.Lock()
	l.state = stateReady
	l.mu.Unlock()
	log.Println("launcher: services ready")

	l.resetIdleTimer()
}

func (l *launcher) waitForHealthy() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mcURL := fmt.Sprintf("http://localhost:%d/api/health", l.backendPort)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for mc serve to become healthy")
		default:
		}

		resp, err := client.Get(mcURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Println("launcher: mc serve is healthy")
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (l *launcher) resetIdleTimer() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.idleTimer != nil {
		l.idleTimer.Stop()
	}
	l.idleTimer = time.AfterFunc(l.idleTimeout, func() {
		log.Printf("launcher: idle for %s — shutting down services", l.idleTimeout)
		l.stopServices()
	})
}

func (l *launcher) stopServices() {
	l.mu.Lock()
	if l.idleTimer != nil {
		l.idleTimer.Stop()
		l.idleTimer = nil
	}
	cmd := l.serveCmd
	l.serveCmd = nil
	l.state = stateSleeping
	l.mu.Unlock()

	// Kill mc serve child process
	if cmd != nil && cmd.Process != nil {
		log.Println("launcher: stopping mc serve")
		cmd.Process.Signal(os.Interrupt)
		// Give it a moment to shut down gracefully
		done := make(chan struct{})
		go func() {
			cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			log.Println("launcher: mc serve did not exit, killing")
			cmd.Process.Kill()
		}
	}

	log.Println("launcher: services stopped, sleeping")
}

var launcherCmd = &cobra.Command{
	Use:   "launcher",
	Short: "Lightweight launcher proxy with lifecycle management for DutyBound services",
	Long: `Runs a lightweight reverse proxy on the configured port. When idle, the
MissionControl orchestrator is stopped. Incoming requests trigger a wake
cycle, and the orchestrator is shut down again after the idle timeout.
OpenClaw Gateway is expected to be managed independently.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		backendPort, _ := cmd.Flags().GetInt("backend-port")
		idleStr, _ := cmd.Flags().GetString("idle-timeout")

		idleTimeout, err := time.ParseDuration(idleStr)
		if err != nil {
			return fmt.Errorf("invalid idle-timeout: %w", err)
		}

		l := newLauncher(port, backendPort, idleTimeout)

		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}

		log.Printf("launcher: listening on %s (backend :%d, idle timeout %s)", addr, backendPort, idleTimeout)
		log.Printf("launcher: state = sleeping")

		return http.Serve(ln, l)
	},
}

func init() {
	rootCmd.AddCommand(launcherCmd)
	launcherCmd.Flags().Int("port", 8080, "Launcher listen port")
	launcherCmd.Flags().Int("backend-port", 8081, "MC orchestrator backend port")
	launcherCmd.Flags().String("idle-timeout", "30m", "Shutdown after idle duration")
}
