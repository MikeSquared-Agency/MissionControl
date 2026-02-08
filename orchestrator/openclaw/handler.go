package openclaw

import (
	"encoding/json"
	"net/http"
)

// Handler exposes REST endpoints for the OpenClaw bridge.
type Handler struct {
	bridge *Bridge
}

// NewHandler creates a new HTTP handler wrapping the bridge.
func NewHandler(bridge *Bridge) *Handler {
	return &Handler{bridge: bridge}
}

// RegisterRoutes registers /api/openclaw/* routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/openclaw/status", h.handleStatus)
	mux.HandleFunc("/api/openclaw/send", h.handleSend)
	mux.HandleFunc("/api/openclaw/chat", h.handleChat)
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

// RegisterChatAlias registers /api/chat as an alias for /api/openclaw/chat.
func (h *Handler) RegisterChatAlias(mux *http.ServeMux) {
	mux.HandleFunc("/api/chat", h.handleChat)
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

	// Build chat.send params
	params := map[string]interface{}{
		"message": req.Message,
	}
	if req.SessionKey != "" {
		params["sessionKey"] = req.SessionKey
	}

	resp, err := h.bridge.Send("chat.send", params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ChatResponse{OK: false, Error: err.Error()})
		return
	}

	// Check if the gateway returned an error
	if resp.OK != nil && !*resp.OK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ChatResponse{OK: false, Error: string(resp.Error), Payload: resp.Payload})
		return
	}

	// Extract reply text from payload if possible
	var payload struct {
		Reply string `json:"reply"`
		Text  string `json:"text"`
	}
	reply := ""
	if resp.Payload != nil {
		if err := json.Unmarshal(resp.Payload, &payload); err == nil {
			reply = payload.Reply
			if reply == "" {
				reply = payload.Text
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{OK: true, Reply: reply, Payload: resp.Payload})
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
