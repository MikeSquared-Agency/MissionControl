package terminal

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// PTYHandler manages WebSocket connections to tmux sessions via PTY
type PTYHandler struct {
	upgrader websocket.Upgrader
}

// NewPTYHandler creates a new PTY handler
func NewPTYHandler() *PTYHandler {
	return &PTYHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// ResizeMessage represents a terminal resize request
type ResizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// HandleWebSocket handles WebSocket connections for terminal streaming
func (h *PTYHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionName := r.URL.Query().Get("session")
	if sessionName == "" {
		http.Error(w, "session parameter required", http.StatusBadRequest)
		return
	}

	readOnly := r.URL.Query().Get("readonly") == "true"

	// Validate session exists
	tmuxPath := findTmux()
	if err := exec.Command(tmuxPath, "has-session", "-t", sessionName).Run(); err != nil {
		http.Error(w, "Session not found: "+sessionName, http.StatusNotFound)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Start tmux attach with PTY
	cmd := exec.Command(tmuxPath, "attach-session", "-t", sessionName)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to attach: "+err.Error()))
		return
	}
	defer func() {
		_ = cmd.Process.Kill()
		ptmx.Close()
	}()

	// Set initial PTY size (will be updated by client resize message)
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	// Send Ctrl+L to force tmux to redraw the screen
	time.Sleep(100 * time.Millisecond)
	_, _ = ptmx.Write([]byte{12}) // Ctrl+L = ASCII 12

	var wg sync.WaitGroup
	done := make(chan struct{})

	// PTY → WebSocket (always enabled)
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("PTY read error: %v", err)
					}
					return
				}
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}()

	// WebSocket → PTY (handles input and resize)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				return
			}

			// Handle resize messages
			if messageType == websocket.TextMessage {
				var msg ResizeMessage
				if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
					if err := pty.Setsize(ptmx, &pty.Winsize{
						Rows: msg.Rows,
						Cols: msg.Cols,
					}); err != nil {
						log.Printf("PTY resize error: %v", err)
					}
					continue
				}
			}

			// Forward input to PTY (if not read-only)
			if !readOnly {
				if _, err := ptmx.Write(data); err != nil {
					log.Printf("PTY write error: %v", err)
					return
				}
			}
		}
	}()

	wg.Wait()
}

// findTmux returns the path to the tmux binary
func findTmux() string {
	paths := []string{
		"/opt/homebrew/bin/tmux",
		"/usr/local/bin/tmux",
		"/usr/bin/tmux",
	}
	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	if path, err := exec.LookPath("tmux"); err == nil {
		return path
	}
	return "tmux"
}
