package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Event is the core message type broadcast through the hub.
type Event struct {
	Topic string          `json:"topic"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

// clientCommand represents a command sent from the client.
type clientCommand struct {
	Type   string   `json:"type"`
	Topics []string `json:"topics,omitempty"`
}

// Client represents a WebSocket client with optional topic subscriptions.
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool // nil = receive all topics
	mu            sync.RWMutex
}

// Hub maintains connected clients and broadcasts namespaced events.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	stateProvider func() interface{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[ws] client connected (%d total)", h.ClientCount())
			h.sendInitialState(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[ws] client disconnected (%d total)", h.ClientCount())

		case event := <-h.broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("[ws] marshal error: %v", err)
				continue
			}
			h.mu.RLock()
			for client := range h.clients {
				if client.wantsTopic(event.Topic) {
					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends an event to all subscribed clients.
func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}

// BroadcastRaw marshals data and broadcasts as an Event.
func (h *Hub) BroadcastRaw(topic, eventType string, data interface{}) {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Printf("[ws] BroadcastRaw marshal error: %v", err)
		return
	}
	h.Broadcast(Event{Topic: topic, Type: eventType, Data: raw})
}

// SetStateProvider sets the function used for initial state sync.
func (h *Hub) SetStateProvider(fn func() interface{}) {
	h.mu.Lock()
	h.stateProvider = fn
	h.mu.Unlock()
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket upgrades the HTTP connection and registers the client.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !h.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

// checkAuth validates the request against MC_API_TOKEN.
func (h *Hub) checkAuth(r *http.Request) bool {
	token := os.Getenv("MC_API_TOKEN")
	if token == "" {
		return true // auth disabled in dev mode
	}
	// Check Authorization header (Bearer token)
	if auth := r.Header.Get("Authorization"); auth == "Bearer "+token {
		return true
	}
	// Check query param
	if r.URL.Query().Get("token") == token {
		return true
	}
	return false
}

// sendInitialState sends state sync to a newly connected client.
func (h *Hub) sendInitialState(client *Client) {
	h.mu.RLock()
	provider := h.stateProvider
	h.mu.RUnlock()

	if provider == nil {
		return
	}
	state := provider()
	if state == nil {
		return
	}
	raw, err := json.Marshal(state)
	if err != nil {
		log.Printf("[ws] initial state marshal error: %v", err)
		return
	}
	event := Event{Topic: "sync", Type: "initial_state", Data: raw}
	data, _ := json.Marshal(event)
	select {
	case client.send <- data:
	default:
	}
}

// wantsTopic returns true if the client should receive events for the given topic.
func (c *Client) wantsTopic(topic string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.subscriptions == nil {
		return true // nil = receive all
	}
	return c.subscriptions[topic]
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}
		c.handleCommand(message)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// handleCommand processes client commands.
func (c *Client) handleCommand(message []byte) {
	var cmd clientCommand
	if err := json.Unmarshal(message, &cmd); err != nil {
		return
	}
	switch cmd.Type {
	case "subscribe":
		c.mu.Lock()
		if c.subscriptions == nil {
			c.subscriptions = make(map[string]bool)
		}
		for _, t := range cmd.Topics {
			c.subscriptions[t] = true
		}
		c.mu.Unlock()

	case "unsubscribe":
		c.mu.Lock()
		if c.subscriptions != nil {
			for _, t := range cmd.Topics {
				delete(c.subscriptions, t)
			}
		}
		c.mu.Unlock()

	case "request_sync":
		c.hub.sendInitialState(c)
	}
}
