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
)

// AgentType represents the type of agent
type AgentType string

const (
	AgentTypePython AgentType = "python"
	AgentTypeClaude AgentType = "claude"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	StatusStarting AgentStatus = "starting"
	StatusRunning  AgentStatus = "running"
	StatusIdle     AgentStatus = "idle"
	StatusError    AgentStatus = "error"
	StatusStopped  AgentStatus = "stopped"
)

// Agent represents a running agent process
type Agent struct {
	ID        string      `json:"id"`
	Type      AgentType   `json:"type"`
	Task      string      `json:"task"`
	Workdir   string      `json:"workdir"`
	Status    AgentStatus `json:"status"`
	PID       int         `json:"pid"`
	Tokens    int         `json:"tokens"`
	CreatedAt time.Time   `json:"created_at"`
	Error     string      `json:"error,omitempty"`

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// Event represents a normalized event from an agent
type Event struct {
	Type    string          `json:"type"`
	AgentID string          `json:"agent_id"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Manager manages all agent processes
type Manager struct {
	agents     map[string]*Agent
	mu         sync.RWMutex
	eventsChan chan Event
	agentsDir  string
}

// NewManager creates a new agent manager
func NewManager(agentsDir string) *Manager {
	return &Manager{
		agents:     make(map[string]*Agent),
		eventsChan: make(chan Event, 100),
		agentsDir:  agentsDir,
	}
}

// Events returns the channel for agent events
func (m *Manager) Events() <-chan Event {
	return m.eventsChan
}

// SpawnRequest represents a request to spawn an agent
type SpawnRequest struct {
	Type    AgentType `json:"type"`
	Task    string    `json:"task"`
	Workdir string    `json:"workdir"`
	Agent   string    `json:"agent"` // For python type: v0_minimal, v1_basic, etc.
}

// Spawn creates and starts a new agent
func (m *Manager) Spawn(req SpawnRequest) (*Agent, error) {
	agent := &Agent{
		ID:        uuid.New().String()[:8],
		Type:      req.Type,
		Task:      req.Task,
		Workdir:   req.Workdir,
		Status:    StatusStarting,
		CreatedAt: time.Now(),
	}

	var cmd *exec.Cmd

	switch req.Type {
	case AgentTypePython:
		agentFile := req.Agent
		if agentFile == "" {
			agentFile = "v1_basic"
		}
		agentPath := fmt.Sprintf("%s/%s.py", m.agentsDir, agentFile)
		cmd = exec.Command("python3", agentPath, req.Task)

	case AgentTypeClaude:
		cmd = exec.Command("claude", "-p", req.Task, "--output-format", "stream-json")

	default:
		return nil, fmt.Errorf("unknown agent type: %s", req.Type)
	}

	if req.Workdir != "" {
		cmd.Dir = req.Workdir
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
	agent.Status = StatusRunning

	// Store agent
	m.mu.Lock()
	m.agents[agent.ID] = agent
	m.mu.Unlock()

	// Emit spawn event
	m.emitEvent("agent_spawned", agent.ID, agent)

	// Start reading output
	go m.readOutput(agent)

	// Wait for process to complete
	go m.waitForCompletion(agent)

	return agent, nil
}

// readOutput reads stdout from the agent and emits events
func (m *Manager) readOutput(agent *Agent) {
	scanner := bufio.NewScanner(agent.stdout)
	for scanner.Scan() {
		line := scanner.Text()

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

	// Also read stderr
	go func() {
		scanner := bufio.NewScanner(agent.stderr)
		for scanner.Scan() {
			line := scanner.Text()
			m.emitEvent("agent_error", agent.ID, map[string]string{"text": line})
		}
	}()
}

// waitForCompletion waits for the agent process to finish
func (m *Manager) waitForCompletion(agent *Agent) {
	err := agent.cmd.Wait()

	m.mu.Lock()
	if err != nil {
		agent.Status = StatusError
		agent.Error = err.Error()
	} else {
		agent.Status = StatusStopped
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
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	if agent.cmd.Process == nil {
		return fmt.Errorf("agent process not running")
	}

	if err := agent.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill agent: %w", err)
	}

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
