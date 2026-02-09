package openclaw

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/DarlingtonDeveloper/MissionControl/tracker"
)

// Broadcaster can push events to the WebSocket hub.
type Broadcaster interface {
	BroadcastRaw(topic, eventType string, data interface{})
}

// WorkerMeta holds pre-registered metadata for a worker about to be spawned.
type WorkerMeta struct {
	Label        string    `json:"label"`
	TaskID       string    `json:"task_id"`
	Persona      string    `json:"persona"`
	Zone         string    `json:"zone"`
	Model        string    `json:"model"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Handler exposes REST endpoints for the OpenClaw bridge.
type Handler struct {
	bridge  *Bridge
	hub     Broadcaster
	tracker *tracker.Tracker

	// Pending chat responses keyed by runId
	chatWaiters   map[string]chan string
	chatWaitersMu sync.Mutex

	// Worker registry: sessionKey → WorkerMeta (pre-registered before spawn)
	workerRegistry   map[string]*WorkerMeta
	workerRegistryMu sync.RWMutex

	// runId → sessionKey mapping for lifecycle event lookup
	runToSession   map[string]string
	runToSessionMu sync.RWMutex

	// Buffered lifecycle/start events that arrived before registration
	pendingStarts   map[string]*agentEventPayload
	pendingStartsMu sync.Mutex

	// Buffered lifecycle/end events that arrived before start was processed
	pendingEnds   map[string]*agentEventPayload
	pendingEndsMu sync.Mutex

	// Dedup set for processed (runId, phase) pairs
	processedEvents   map[string]bool
	processedEventsMu sync.Mutex

	// Shutdown channel for background goroutines
	stopCh chan struct{}
}

// NewHandler creates a new HTTP handler wrapping the bridge.
// If hub is non-nil, chat events are broadcast to WebSocket clients.
// If trk is non-nil, lifecycle events register/deregister workers in the tracker.
func NewHandler(bridge *Bridge, hub Broadcaster, trk ...*tracker.Tracker) *Handler {
	h := &Handler{
		bridge:          bridge,
		hub:             hub,
		chatWaiters:     make(map[string]chan string),
		workerRegistry:  make(map[string]*WorkerMeta),
		runToSession:    make(map[string]string),
		pendingStarts:   make(map[string]*agentEventPayload),
		pendingEnds:     make(map[string]*agentEventPayload),
		processedEvents: make(map[string]bool),
		stopCh:          make(chan struct{}),
	}
	if len(trk) > 0 && trk[0] != nil {
		h.tracker = trk[0]
	}

	// Listen for events from the bridge to capture chat responses
	prevHandler := bridge.EventHandler
	bridge.EventHandler = func(event string, payload json.RawMessage) {
		// DEBUG: log all events from gateway
		log.Printf("[openclaw] event: %s payload: %s", event, string(payload)[:min(len(string(payload)), 200)])

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

				if text != "" {
					// Broadcast to WebSocket hub for real-time UI
					if h.hub != nil {
						h.hub.BroadcastRaw("chat", "chat_message", map[string]interface{}{
							"id":        msg.RunID,
							"role":      "assistant",
							"content":   text,
							"timestamp": time.Now().UTC().Format(time.RFC3339),
							"event":     event,
						})
					}

					// Resolve pending HTTP waiter
					if msg.RunID != "" {
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
		}

		// Handle lifecycle events from sub-agents
		if event == "agent" {
			h.handleLifecycleEvent(payload)
		}

		// Pass through to previous handler
		if prevHandler != nil {
			prevHandler(event, payload)
		}
	}

	// Start cleanup goroutine for stale pending starts
	go h.cleanupPendingStarts()

	return h
}

// agentEventPayload represents a lifecycle event from the gateway.
type agentEventPayload struct {
	RunID      string `json:"runId"`
	Stream     string `json:"stream"`
	SessionKey string `json:"sessionKey"`
	Seq        int    `json:"seq"`
	Data       struct {
		Phase     string `json:"phase"`
		StartedAt int64  `json:"startedAt"`
		EndedAt   int64  `json:"endedAt"`
	} `json:"data"`
}

func (h *Handler) handleLifecycleEvent(payload json.RawMessage) {
	var ev agentEventPayload
	if err := json.Unmarshal(payload, &ev); err != nil {
		return
	}
	if ev.Stream != "lifecycle" {
		return
	}
	if !strings.Contains(ev.SessionKey, "subagent:") {
		return
	}

	// Dedup check
	dedupKey := ev.RunID + ":" + ev.Data.Phase
	h.processedEventsMu.Lock()
	if h.processedEvents[dedupKey] {
		h.processedEventsMu.Unlock()
		return
	}
	h.processedEvents[dedupKey] = true
	// Bound the dedup set
	if len(h.processedEvents) > 1000 {
		h.processedEvents = make(map[string]bool)
		h.processedEvents[dedupKey] = true
	}
	h.processedEventsMu.Unlock()

	switch ev.Data.Phase {
	case "start":
		h.handleWorkerStart(&ev)
	case "end":
		h.handleWorkerEnd(&ev)
	}
}

func (h *Handler) handleWorkerStart(ev *agentEventPayload) {
	h.workerRegistryMu.Lock()
	meta, ok := h.workerRegistry[ev.SessionKey]
	if !ok {
		h.workerRegistryMu.Unlock()
		// Buffer for later — registration hasn't arrived yet
		h.pendingStartsMu.Lock()
		h.pendingStarts[ev.SessionKey] = ev
		h.pendingStartsMu.Unlock()
		log.Printf("[openclaw] lifecycle/start for unregistered session %s (runId=%s), buffering", ev.SessionKey, ev.RunID)
		return
	}
	h.workerRegistryMu.Unlock()

	// Map runId → sessionKey
	h.runToSessionMu.Lock()
	h.runToSession[ev.RunID] = ev.SessionKey
	h.runToSessionMu.Unlock()

	// Register in tracker
	if h.tracker != nil {
		h.tracker.Register(meta.Label, meta.TaskID, meta.Persona, meta.Zone, meta.Model)
	}

	// Broadcast
	if h.hub != nil {
		h.hub.BroadcastRaw("workers", "worker_started", map[string]interface{}{
			"worker_id":  meta.Label,
			"task_id":    meta.TaskID,
			"persona":    meta.Persona,
			"zone":       meta.Zone,
			"model":      meta.Model,
			"started_at": time.Now().UTC().Format(time.RFC3339),
		})
	}

	log.Printf("[openclaw] worker started: %s (task=%s, session=%s)", meta.Label, meta.TaskID, ev.SessionKey)

	// Check if we have a buffered lifecycle/end for this session (fast worker)
	h.pendingEndsMu.Lock()
	pendingEnd, hasEnd := h.pendingEnds[ev.SessionKey]
	if hasEnd {
		delete(h.pendingEnds, ev.SessionKey)
	}
	h.pendingEndsMu.Unlock()

	if hasEnd {
		log.Printf("[openclaw] processing buffered lifecycle/end for session %s", ev.SessionKey)
		h.handleWorkerEnd(pendingEnd)
	}
}

func (h *Handler) handleWorkerEnd(ev *agentEventPayload) {
	// Consistent lock ordering: workerRegistryMu before runToSessionMu
	h.workerRegistryMu.Lock()
	h.runToSessionMu.Lock()
	sessionKey, ok := h.runToSession[ev.RunID]
	if !ok {
		h.runToSessionMu.Unlock()
		h.workerRegistryMu.Unlock()
		// Buffer for later — start hasn't been processed yet (fast worker)
		h.pendingEndsMu.Lock()
		h.pendingEnds[ev.SessionKey] = ev
		h.pendingEndsMu.Unlock()
		log.Printf("[openclaw] lifecycle/end for unknown runId=%s (session=%s), buffering", ev.RunID, ev.SessionKey)
		return
	}
	delete(h.runToSession, ev.RunID)
	h.runToSessionMu.Unlock()

	meta, hasMeta := h.workerRegistry[sessionKey]
	if hasMeta {
		delete(h.workerRegistry, sessionKey)
	}
	h.workerRegistryMu.Unlock()

	if !hasMeta {
		return
	}

	// Deregister from tracker
	if h.tracker != nil {
		h.tracker.Deregister(meta.Label, tracker.StatusComplete)
	}

	// Broadcast
	if h.hub != nil {
		h.hub.BroadcastRaw("workers", "worker_stopped", map[string]interface{}{
			"worker_id":  meta.Label,
			"task_id":    meta.TaskID,
			"status":     "complete",
			"stopped_at": time.Now().UTC().Format(time.RFC3339),
		})
	}

	log.Printf("[openclaw] worker stopped: %s (task=%s)", meta.Label, meta.TaskID)
}

// Close stops background goroutines.
func (h *Handler) Close() {
	close(h.stopCh)
}

// cleanupPendingStarts removes stale buffered lifecycle events.
func (h *Handler) cleanupPendingStarts() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
		}
		now := time.Now()
		h.pendingStartsMu.Lock()
		for key, ev := range h.pendingStarts {
			startedAt := time.UnixMilli(ev.Data.StartedAt)
			if now.Sub(startedAt) > 60*time.Second {
				delete(h.pendingStarts, key)
				log.Printf("[openclaw] expired pending start for session %s", key)
			}
		}
		h.pendingStartsMu.Unlock()

		h.pendingEndsMu.Lock()
		for key, ev := range h.pendingEnds {
			endedAt := time.UnixMilli(ev.Data.EndedAt)
			if now.Sub(endedAt) > 60*time.Second {
				delete(h.pendingEnds, key)
				log.Printf("[openclaw] expired pending end for session %s", key)
			}
		}
		h.pendingEndsMu.Unlock()
	}
}

// RegisterRoutes registers /api/openclaw/* routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/openclaw/status", h.handleStatus)
	mux.HandleFunc("/api/openclaw/send", h.handleSend)
	mux.HandleFunc("/api/openclaw/chat", h.handleChat)
}

// RegisterMCRoutes registers /api/mc/* routes for worker lifecycle management.
func (h *Handler) RegisterMCRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/mc/worker/register", h.handleWorkerRegister)
	mux.HandleFunc("/api/mc/workers", h.handleWorkersList)
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

	// Broadcast user message to hub
	if h.hub != nil {
		h.hub.BroadcastRaw("chat", "chat_message", map[string]interface{}{
			"id":        randomID(),
			"role":      "user",
			"content":   req.Message,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
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

// workerRegisterRequest is the JSON body for POST /api/mc/worker/register.
type workerRegisterRequest struct {
	SessionKey string `json:"session_key"`
	Label      string `json:"label"`
	TaskID     string `json:"task_id"`
	Persona    string `json:"persona"`
	Zone       string `json:"zone"`
	Model      string `json:"model"`
}

func (h *Handler) handleWorkerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req workerRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.SessionKey == "" || req.Label == "" {
		http.Error(w, "session_key and label are required", http.StatusBadRequest)
		return
	}

	meta := &WorkerMeta{
		Label:        req.Label,
		TaskID:       req.TaskID,
		Persona:      req.Persona,
		Zone:         req.Zone,
		Model:        req.Model,
		RegisteredAt: time.Now(),
	}

	h.workerRegistryMu.Lock()
	h.workerRegistry[req.SessionKey] = meta
	h.workerRegistryMu.Unlock()

	log.Printf("[openclaw] registered worker %s for session %s (task=%s)", req.Label, req.SessionKey, req.TaskID)

	// Check if we have a buffered lifecycle/start for this session
	h.pendingStartsMu.Lock()
	pendingEv, hasPending := h.pendingStarts[req.SessionKey]
	if hasPending {
		delete(h.pendingStarts, req.SessionKey)
	}
	h.pendingStartsMu.Unlock()

	if hasPending {
		log.Printf("[openclaw] processing buffered lifecycle/start for session %s", req.SessionKey)
		h.handleWorkerStart(pendingEv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) handleWorkersList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var workers interface{}
	if h.tracker != nil {
		workers = h.tracker.List()
	} else {
		workers = []interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}
