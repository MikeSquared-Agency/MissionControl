package openclaw

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Handler exposes REST endpoints for the OpenClaw bridge.
type Handler struct {
	bridge *Bridge

	// Pending chat responses keyed by runId
	chatWaiters   map[string]chan string
	chatWaitersMu sync.Mutex
}

// NewHandler creates a new HTTP handler wrapping the bridge.
func NewHandler(bridge *Bridge) *Handler {
	h := &Handler{
		bridge:      bridge,
		chatWaiters: make(map[string]chan string),
	}

	// Listen for events from the bridge to capture chat responses
	prevHandler := bridge.EventHandler
	bridge.EventHandler = func(event string, payload json.RawMessage) {
		// Check for agent response events
		if event == "agent.reply" || event == "chat.message" || event == "agent.turn.complete" {
			var msg struct {
				RunID      string `json:"runId"`
				SessionKey string `json:"sessionKey"`
				Text       string `json:"text"`
				Content    string `json:"content"`
				Message    struct {
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(payload, &msg) == nil {
				text := msg.Text
				if text == "" {
					text = msg.Content
				}
				if text == "" && len(msg.Message.Content) > 0 {
					for _, c := range msg.Message.Content {
						if c.Type == "text" && c.Text != "" {
							text = c.Text
							break
						}
					}
				}

				if text != "" && msg.RunID != "" {
					h.chatWaitersMu.Lock()
					if ch, ok := h.chatWaiters[msg.RunID]; ok {
						select {
						case ch <- text:
						default:
						}
					}
					h.chatWaitersMu.Unlock()
				}
			}
		}

		// Pass through to previous handler
		if prevHandler != nil {
			prevHandler(event, payload)
		}
	}

	return h
}

// RegisterRoutes registers /api/openclaw/* routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/openclaw/status", h.handleStatus)
	mux.HandleFunc("/api/openclaw/send", h.handleSend)
	mux.HandleFunc("/api/openclaw/chat", h.handleChat)
}

// RegisterChatAlias registers /api/chat as an alias for /api/openclaw/chat.
func (h *Handler) RegisterChatAlias(mux *http.ServeMux) {
	mux.HandleFunc("/api/chat", h.handleChat)
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.bridge.Status())
}

// SendRequest is the JSON body for POST /api/openclaw/send.
type SendRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// ChatRequest is the JSON body for POST /api/openclaw/chat.
type ChatRequest struct {
	Message    string `json:"message"`
	SessionKey string `json:"sessionKey,omitempty"`
}

// ChatResponse is the JSON response from POST /api/openclaw/chat.
type ChatResponse struct {
	OK      bool            `json:"ok"`
	Reply   string          `json:"reply,omitempty"`
	Error   string          `json:"error,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	sessionKey := req.SessionKey
	if sessionKey == "" {
		sessionKey = "webchat"
	}
	idempotencyKey := randomID()
	params := map[string]interface{}{
		"message":        req.Message,
		"sessionKey":     sessionKey,
		"idempotencyKey": idempotencyKey,
	}

	resp, err := h.bridge.Send("chat.send", params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ChatResponse{OK: false, Error: err.Error()})
		return
	}

	if resp.OK != nil && !*resp.OK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ChatResponse{OK: false, Error: string(resp.Error), Payload: resp.Payload})
		return
	}

	// Extract runId from the response
	var started struct {
		RunID  string `json:"runId"`
		Status string `json:"status"`
	}
	if resp.Payload != nil {
		json.Unmarshal(resp.Payload, &started)
	}

	// If we got a runId, wait for the async response (up to 60s)
	if started.RunID != "" {
		replyCh := make(chan string, 1)
		h.chatWaitersMu.Lock()
		h.chatWaiters[started.RunID] = replyCh
		h.chatWaitersMu.Unlock()

		defer func() {
			h.chatWaitersMu.Lock()
			delete(h.chatWaiters, started.RunID)
			h.chatWaitersMu.Unlock()
		}()

		select {
		case reply := <-replyCh:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ChatResponse{OK: true, Reply: reply})
			return
		case <-time.After(60 * time.Second):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ChatResponse{OK: true, Reply: "(still thinking...)", Payload: resp.Payload})
			return
		}
	}

	// Fallback: return whatever we got
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{OK: true, Payload: resp.Payload})
}

func (h *Handler) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Method == "" {
		http.Error(w, "method is required", http.StatusBadRequest)
		return
	}

	resp, err := h.bridge.Send(req.Method, req.Params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
