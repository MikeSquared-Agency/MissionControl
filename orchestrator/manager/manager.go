package manager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mike/mission-control/hashid"
)

// AgentType represents the type of agent
// Note: Python agent type has been deprecated. All agents now use Claude Code.
type AgentType string

const (
	AgentTypePython     AgentType = "python"      // Deprecated: kept for backward compatibility
	AgentTypeClaudeCode AgentType = "claude-code" // Default agent type
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	StatusStarting AgentStatus = "starting"
	StatusWorking  AgentStatus = "working"
	StatusIdle     AgentStatus = "idle"
	StatusWaiting  AgentStatus = "waiting"
	StatusError    AgentStatus = "error"
	StatusStopped  AgentStatus = "stopped"
)

// Agent represents a running agent process
type Agent struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        AgentType   `json:"type"`
	Task        string      `json:"task"`
	Persona     string      `json:"persona,omitempty"`
	Zone        string      `json:"zone"`
	WorkingDir  string      `json:"workingDir"`
	Status      AgentStatus `json:"status"`
	PID         int         `json:"pid"`
	Tokens      int         `json:"tokens"`
	Cost        float64     `json:"cost"`
	CreatedAt   time.Time   `json:"created_at"`
	Error       string      `json:"error,omitempty"`
	OfflineMode bool        `json:"offlineMode"`
	Model       string      `json:"model,omitempty"`

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// Zone represents an agent grouping
type Zone struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	WorkingDir string `json:"workingDir"`
}

// Event represents a normalized event from an agent
type Event struct {
	Type    string          `json:"type"`
	AgentID string          `json:"agent_id"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Manager manages all agent processes and zones
type Manager struct {
	agents     map[string]*Agent
	zones      map[string]*Zone
	mu         sync.RWMutex
	eventsChan chan Event
	agentsDir  string
}

// NewManager creates a new agent manager
func NewManager(agentsDir string) *Manager {
	m := &Manager{
		agents:     make(map[string]*Agent),
		zones:      make(map[string]*Zone),
		eventsChan: make(chan Event, 100),
		agentsDir:  agentsDir,
	}

	// Create default zone
	m.zones["default"] = &Zone{
		ID:         "default",
		Name:       "Default",
		Color:      "#6b7280",
		WorkingDir: "",
	}

	return m
}

// Events returns the channel for agent events
func (m *Manager) Events() <-chan Event {
	return m.eventsChan
}

// SpawnRequest represents a request to spawn an agent
type SpawnRequest struct {
	Type        AgentType `json:"type"`
	Name        string    `json:"name"`
	Task        string    `json:"task"`
	Persona     string    `json:"persona"`
	Zone        string    `json:"zone"`
	WorkingDir  string    `json:"workingDir"`
	Agent       string    `json:"agent"`       // For python type: v0_minimal, v1_basic, etc.
	OfflineMode bool      `json:"offlineMode"` // Use Ollama instead of Anthropic API
	OllamaModel string    `json:"ollamaModel"` // Model to use in offline mode, e.g., "qwen3-coder"
}

// Spawn creates and starts a new agent
func (m *Manager) Spawn(req SpawnRequest) (*Agent, error) {
	id := hashid.Generate("agent", req.Task, string(req.Type), req.Zone, req.Persona)

	// Use provided name or generate from ID
	name := req.Name
	if name == "" {
		name = id
	}

	// Default to "default" zone if not specified
	zone := req.Zone
	if zone == "" {
		zone = "default"
	}

	agent := &Agent{
		ID:          id,
		Name:        name,
		Type:        req.Type,
		Task:        req.Task,
		Persona:     req.Persona,
		Zone:        zone,
		WorkingDir:  req.WorkingDir,
		Status:      StatusStarting,
		Tokens:      0,
		Cost:        0,
		CreatedAt:   time.Now(),
		OfflineMode: req.OfflineMode,
		Model:       req.OllamaModel,
	}

	// All agents now use Claude Code (Python agents have been deprecated)
	// Use --dangerously-skip-permissions for headless execution
	// In production, consider using --permission-mode with more granular control
	args := []string{"-p", req.Task, "--output-format", "stream-json", "--dangerously-skip-permissions"}

	// Add model flag for offline mode
	if req.OfflineMode && req.OllamaModel != "" {
		args = append(args, "--model", req.OllamaModel)
	}

	cmd := exec.Command("claude", args...)
	fmt.Printf("Spawning Claude Code agent: claude %v\n", args)

	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}

	// Pass through environment variables (includes ANTHROPIC_API_KEY)
	cmd.Env = os.Environ()

	// For offline mode, override environment to point to Ollama
	if req.OfflineMode {
		cmd.Env = append(cmd.Env,
			"ANTHROPIC_BASE_URL=http://localhost:11434",
			"ANTHROPIC_AUTH_TOKEN=ollama",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		)
		fmt.Printf("Agent %s running in offline mode with Ollama (model: %s)\n", id, req.OllamaModel)
	}

	// Set up pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	agent.cmd = cmd
	agent.stdout = stdout
	agent.stderr = stderr
	agent.stdin = stdin

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	agent.PID = cmd.Process.Pid
	agent.Status = StatusWorking

	// Store agent
	m.mu.Lock()
	m.agents[agent.ID] = agent
	m.mu.Unlock()

	// Emit spawn event
	m.emitEvent("agent_spawned", agent.ID, agent)

	// For non-interactive prompts (-p flag), close stdin to signal EOF
	if req.Type == AgentTypeClaudeCode {
		stdin.Close()
		agent.stdin = nil
	}

	// Start reading output
	go m.readOutput(agent)
	go m.readStderr(agent)

	// Wait for process to complete
	go m.waitForCompletion(agent)

	return agent, nil
}

// readOutput reads stdout from the agent and emits events
func (m *Manager) readOutput(agent *Agent) {
	// Use larger buffer for JSON output (1MB)
	scanner := bufio.NewScanner(agent.stdout)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	fmt.Printf("Agent %s: Starting to read output...\n", agent.ID)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Agent %s output: %s\n", agent.ID, truncate(line, 200))

		// Try to parse as JSON event
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			// It's JSON - emit as-is
			m.emitEvent("agent_output", agent.ID, event)
		} else {
			// Plain text output
			m.emitEvent("agent_output", agent.ID, map[string]string{"text": line})
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Agent %s: Scanner error: %v\n", agent.ID, err)
	}

	fmt.Printf("Agent %s: Finished reading output\n", agent.ID)
}

// truncate truncates a string for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// readStderr reads stderr from the agent
func (m *Manager) readStderr(agent *Agent) {
	scanner := bufio.NewScanner(agent.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Agent %s stderr: %s\n", agent.ID, line)
		m.emitEvent("agent_error", agent.ID, map[string]string{"text": line})
	}
}

// waitForCompletion waits for the agent process to finish
func (m *Manager) waitForCompletion(agent *Agent) {
	err := agent.cmd.Wait()

	m.mu.Lock()
	if err != nil {
		agent.Status = StatusError
		agent.Error = err.Error()
		fmt.Printf("Agent %s (%s) exited with error: %v\n", agent.Name, agent.ID, err)
	} else {
		agent.Status = StatusStopped
		fmt.Printf("Agent %s (%s) completed successfully\n", agent.Name, agent.ID)
	}
	m.mu.Unlock()

	m.emitEvent("agent_stopped", agent.ID, map[string]interface{}{
		"status": agent.Status,
		"error":  agent.Error,
	})
}

// emitEvent sends an event to the events channel
func (m *Manager) emitEvent(eventType string, agentID string, data interface{}) {
	dataBytes, _ := json.Marshal(data)
	event := Event{
		Type:    eventType,
		AgentID: agentID,
		Data:    dataBytes,
	}

	select {
	case m.eventsChan <- event:
	default:
		// Channel full, drop event
		fmt.Fprintf(os.Stderr, "Warning: event channel full, dropping event\n")
	}
}

// Get returns an agent by ID
func (m *Manager) Get(id string) (*Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agent, ok := m.agents[id]
	return agent, ok
}

// List returns all agents
func (m *Manager) List() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

// Kill stops an agent by ID
func (m *Manager) Kill(id string) error {
	m.mu.Lock()
	agent, ok := m.agents[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("agent not found: %s", id)
	}

	// If already stopped, just remove from map
	if agent.Status == StatusStopped || agent.Status == StatusError {
		delete(m.agents, id)
		m.mu.Unlock()
		m.emitEvent("agent_removed", id, nil)
		return nil
	}
	m.mu.Unlock()

	// Try to kill the process
	if agent.cmd != nil && agent.cmd.Process != nil {
		_ = agent.cmd.Process.Kill() // Ignore error - process may already be dead
	}

	// Remove from map
	m.mu.Lock()
	delete(m.agents, id)
	m.mu.Unlock()

	m.emitEvent("agent_removed", id, nil)
	return nil
}

// SendMessage sends a message to an agent's stdin
func (m *Manager) SendMessage(id string, message string) error {
	m.mu.RLock()
	agent, ok := m.agents[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	if agent.stdin == nil {
		return fmt.Errorf("agent stdin not available")
	}

	_, err := fmt.Fprintln(agent.stdin, message)
	return err
}

// SendKingMessage sends a message to the King orchestrator
// The King is a special Claude Code agent that manages other agents
func (m *Manager) SendKingMessage(message string) error {
	m.mu.RLock()
	// Look for an agent named "king" or with persona "king"
	var kingAgent *Agent
	for _, agent := range m.agents {
		if agent.Name == "King" || agent.Persona == "king" {
			kingAgent = agent
			break
		}
	}
	m.mu.RUnlock()

	// If no king agent exists, spawn one
	if kingAgent == nil {
		req := SpawnRequest{
			Type:    AgentTypeClaudeCode,
			Name:    "King",
			Task:    message,
			Persona: "king",
			Zone:    "default",
		}
		agent, err := m.Spawn(req)
		if err != nil {
			return fmt.Errorf("failed to spawn king agent: %w", err)
		}

		// Emit king response event (the agent will send responses via WebSocket)
		m.emitEvent("king_response", agent.ID, map[string]interface{}{
			"message": map[string]interface{}{
				"role":      "assistant",
				"content":   "I'm analyzing your request and will coordinate the team to accomplish this goal...",
				"timestamp": time.Now().UnixMilli(),
			},
		})
		return nil
	}

	// Send message to existing king agent
	if kingAgent.stdin == nil {
		return fmt.Errorf("king agent stdin not available")
	}

	_, err := fmt.Fprintln(kingAgent.stdin, message)
	if err != nil {
		return err
	}

	return nil
}

// Zone management methods

// CreateZone creates a new zone
func (m *Manager) CreateZone(zone *Zone) (*Zone, error) {
	if zone.ID == "" {
		zone.ID = uuid.New().String()[:8]
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.zones[zone.ID]; exists {
		return nil, fmt.Errorf("zone already exists: %s", zone.ID)
	}

	m.zones[zone.ID] = zone
	m.emitEvent("zone_created", "", zone)
	return zone, nil
}

// GetZone returns a zone by ID
func (m *Manager) GetZone(id string) (*Zone, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	zone, ok := m.zones[id]
	return zone, ok
}

// ListZones returns all zones
func (m *Manager) ListZones() []*Zone {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zones := make([]*Zone, 0, len(m.zones))
	for _, zone := range m.zones {
		zones = append(zones, zone)
	}
	return zones
}

// UpdateZone updates a zone
func (m *Manager) UpdateZone(id string, updates *Zone) (*Zone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	zone, ok := m.zones[id]
	if !ok {
		return nil, fmt.Errorf("zone not found: %s", id)
	}

	if updates.Name != "" {
		zone.Name = updates.Name
	}
	if updates.Color != "" {
		zone.Color = updates.Color
	}
	if updates.WorkingDir != "" {
		zone.WorkingDir = updates.WorkingDir
	}

	m.emitEvent("zone_updated", "", zone)
	return zone, nil
}

// DeleteZone deletes a zone
func (m *Manager) DeleteZone(id string) error {
	if id == "default" {
		return fmt.Errorf("cannot delete default zone")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.zones[id]; !ok {
		return fmt.Errorf("zone not found: %s", id)
	}

	// Check if any agents are in this zone
	for _, agent := range m.agents {
		if agent.Zone == id {
			return fmt.Errorf("cannot delete zone with active agents")
		}
	}

	delete(m.zones, id)
	m.emitEvent("zone_deleted", "", map[string]string{"zoneId": id})
	return nil
}

// MoveAgent moves an agent to a different zone
func (m *Manager) MoveAgent(agentID, zoneID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if _, ok := m.zones[zoneID]; !ok {
		return fmt.Errorf("zone not found: %s", zoneID)
	}

	agent.Zone = zoneID
	m.emitEvent("agent_status", agentID, map[string]interface{}{
		"zone": zoneID,
	})
	return nil
}
