package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mike/mission-control/manager"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts events
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	manager    *manager.Manager
	mu         sync.RWMutex

	// v4StateProvider provides v4 state for initial sync
	v4StateProvider func() interface{}
}

// NewHub creates a new Hub
func NewHub(m *manager.Manager) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		manager:    m,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	// Start listening for manager events
	go h.listenForEvents()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

			// Send current state to new client
			h.sendInitialState(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// listenForEvents listens for events from the manager and broadcasts them
func (h *Hub) listenForEvents() {
	for event := range h.manager.Events() {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}
		h.broadcast <- data
	}
}

// sendInitialState sends the current agents and zones to a client
func (h *Hub) sendInitialState(client *Client) {
	// Send agent list
	agents := h.manager.List()
	agentEvent := map[string]interface{}{
		"type":   "agent_list",
		"agents": agents,
	}
	agentData, _ := json.Marshal(agentEvent)
	client.send <- agentData

	// Send zone list
	zones := h.manager.ListZones()
	zoneEvent := map[string]interface{}{
		"type":  "zone_list",
		"zones": zones,
	}
	zoneData, _ := json.Marshal(zoneEvent)
	client.send <- zoneData

	// Send v4 state if provider is set
	if h.v4StateProvider != nil {
		v4State := h.v4StateProvider()
		if v4State != nil {
			v4Event := map[string]interface{}{
				"type":  "v4_state",
				"state": v4State,
			}
			v4Data, _ := json.Marshal(v4Event)
			client.send <- v4Data
		}
	}
}

// Notify broadcasts an event to all connected clients (implements v4.EventNotifier)
func (h *Hub) Notify(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling v4 event: %v", err)
		return
	}
	h.broadcast <- data
}

// SetV4StateProvider sets the function to get v4 state for initial sync
func (h *Hub) SetV4StateProvider(provider func() interface{}) {
	h.v4StateProvider = provider
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming commands
		c.handleCommand(message)
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

// Command represents a command from the client
type Command struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// SpawnCommand represents a spawn_agent command
type SpawnCommand struct {
	Type       manager.AgentType `json:"type"`
	Name       string            `json:"name"`
	Task       string            `json:"task"`
	Persona    string            `json:"persona"`
	Zone       string            `json:"zone"`
	WorkingDir string            `json:"workingDir"`
	Agent      string            `json:"agent"`
}

// handleCommand processes commands from the client
func (c *Client) handleCommand(message []byte) {
	var cmd Command
	if err := json.Unmarshal(message, &cmd); err != nil {
		log.Printf("Invalid command: %v", err)
		return
	}

	switch cmd.Type {
	case "spawn_agent":
		var spawn SpawnCommand
		if err := json.Unmarshal(cmd.Payload, &spawn); err != nil {
			log.Printf("Invalid spawn command: %v", err)
			return
		}
		_, err := c.hub.manager.Spawn(manager.SpawnRequest{
			Type:       spawn.Type,
			Name:       spawn.Name,
			Task:       spawn.Task,
			Persona:    spawn.Persona,
			Zone:       spawn.Zone,
			WorkingDir: spawn.WorkingDir,
			Agent:      spawn.Agent,
		})
		if err != nil {
			log.Printf("Failed to spawn agent: %v", err)
		}

	case "kill_agent":
		var payload struct {
			AgentID string `json:"agent_id"`
		}
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			log.Printf("Invalid kill command: %v", err)
			return
		}
		if err := c.hub.manager.Kill(payload.AgentID); err != nil {
			log.Printf("Failed to kill agent: %v", err)
		}

	case "send_message":
		var payload struct {
			AgentID string `json:"agent_id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			log.Printf("Invalid message command: %v", err)
			return
		}
		if err := c.hub.manager.SendMessage(payload.AgentID, payload.Content); err != nil {
			log.Printf("Failed to send message: %v", err)
		}

	case "create_zone":
		var zone manager.Zone
		if err := json.Unmarshal(cmd.Payload, &zone); err != nil {
			log.Printf("Invalid create_zone command: %v", err)
			return
		}
		if _, err := c.hub.manager.CreateZone(&zone); err != nil {
			log.Printf("Failed to create zone: %v", err)
		}

	case "update_zone":
		var payload struct {
			ID      string       `json:"id"`
			Updates manager.Zone `json:"updates"`
		}
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			log.Printf("Invalid update_zone command: %v", err)
			return
		}
		if _, err := c.hub.manager.UpdateZone(payload.ID, &payload.Updates); err != nil {
			log.Printf("Failed to update zone: %v", err)
		}

	case "delete_zone":
		var payload struct {
			ZoneID string `json:"zone_id"`
		}
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			log.Printf("Invalid delete_zone command: %v", err)
			return
		}
		if err := c.hub.manager.DeleteZone(payload.ZoneID); err != nil {
			log.Printf("Failed to delete zone: %v", err)
		}

	case "move_agent":
		var payload struct {
			AgentID string `json:"agent_id"`
			ZoneID  string `json:"zone_id"`
		}
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			log.Printf("Invalid move_agent command: %v", err)
			return
		}
		if err := c.hub.manager.MoveAgent(payload.AgentID, payload.ZoneID); err != nil {
			log.Printf("Failed to move agent: %v", err)
		}

	case "request_sync":
		// Send full state to this client
		c.hub.sendInitialState(c)

	default:
		log.Printf("Unknown command type: %s", cmd.Type)
	}
}
