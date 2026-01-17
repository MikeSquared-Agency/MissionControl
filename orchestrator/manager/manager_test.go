package manager

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/agents")

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.agentsDir != "/tmp/agents" {
		t.Errorf("Expected agentsDir to be /tmp/agents, got %s", m.agentsDir)
	}

	// Check default zone exists
	zone, ok := m.zones["default"]
	if !ok {
		t.Fatal("Default zone not created")
	}

	if zone.Name != "Default" {
		t.Errorf("Expected default zone name to be 'Default', got %s", zone.Name)
	}
}

func TestZoneCRUD(t *testing.T) {
	m := NewManager("/tmp/agents")

	// Create zone
	zone := &Zone{
		Name:       "Test Zone",
		Color:      "#22c55e",
		WorkingDir: "/tmp",
	}

	created, err := m.CreateZone(zone)
	if err != nil {
		t.Fatalf("Failed to create zone: %v", err)
	}

	if created.ID == "" {
		t.Error("Created zone has empty ID")
	}

	if created.Name != "Test Zone" {
		t.Errorf("Expected zone name 'Test Zone', got %s", created.Name)
	}

	// Get zone
	fetched, ok := m.GetZone(created.ID)
	if !ok {
		t.Fatal("Failed to get created zone")
	}

	if fetched.Name != "Test Zone" {
		t.Errorf("Expected zone name 'Test Zone', got %s", fetched.Name)
	}

	// Update zone
	updates := &Zone{
		Name:  "Updated Zone",
		Color: "#3b82f6",
	}

	updated, err := m.UpdateZone(created.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update zone: %v", err)
	}

	if updated.Name != "Updated Zone" {
		t.Errorf("Expected updated zone name 'Updated Zone', got %s", updated.Name)
	}

	if updated.Color != "#3b82f6" {
		t.Errorf("Expected updated zone color '#3b82f6', got %s", updated.Color)
	}

	// WorkingDir should be preserved
	if updated.WorkingDir != "/tmp" {
		t.Errorf("Expected workingDir to be preserved as '/tmp', got %s", updated.WorkingDir)
	}

	// List zones
	zones := m.ListZones()
	if len(zones) != 2 { // default + test zone
		t.Errorf("Expected 2 zones, got %d", len(zones))
	}

	// Delete zone
	err = m.DeleteZone(created.ID)
	if err != nil {
		t.Fatalf("Failed to delete zone: %v", err)
	}

	_, ok = m.GetZone(created.ID)
	if ok {
		t.Error("Zone should not exist after deletion")
	}
}

func TestCannotDeleteDefaultZone(t *testing.T) {
	m := NewManager("/tmp/agents")

	err := m.DeleteZone("default")
	if err == nil {
		t.Error("Should not be able to delete default zone")
	}
}

func TestCannotDeleteZoneWithActiveAgents(t *testing.T) {
	m := NewManager("/tmp/agents")

	// Create a zone
	zone, _ := m.CreateZone(&Zone{
		Name:  "Test Zone",
		Color: "#22c55e",
	})

	// Manually add an agent to the zone (without actually spawning a process)
	m.mu.Lock()
	m.agents["test-agent"] = &Agent{
		ID:     "test-agent",
		Name:   "Test Agent",
		Zone:   zone.ID,
		Status: StatusWorking,
	}
	m.mu.Unlock()

	// Try to delete the zone
	err := m.DeleteZone(zone.ID)
	if err == nil {
		t.Error("Should not be able to delete zone with active agents")
	}

	// Clean up
	m.mu.Lock()
	delete(m.agents, "test-agent")
	m.mu.Unlock()

	// Now deletion should work
	err = m.DeleteZone(zone.ID)
	if err != nil {
		t.Errorf("Failed to delete zone after removing agent: %v", err)
	}
}

func TestMoveAgent(t *testing.T) {
	m := NewManager("/tmp/agents")

	// Create a zone
	zone, _ := m.CreateZone(&Zone{
		Name:  "New Zone",
		Color: "#22c55e",
	})

	// Manually add an agent
	m.mu.Lock()
	m.agents["test-agent"] = &Agent{
		ID:     "test-agent",
		Name:   "Test Agent",
		Zone:   "default",
		Status: StatusWorking,
	}
	m.mu.Unlock()

	// Move agent to new zone
	err := m.MoveAgent("test-agent", zone.ID)
	if err != nil {
		t.Fatalf("Failed to move agent: %v", err)
	}

	// Verify agent is in new zone
	agent, _ := m.Get("test-agent")
	if agent.Zone != zone.ID {
		t.Errorf("Expected agent zone to be %s, got %s", zone.ID, agent.Zone)
	}
}

func TestMoveAgentToNonExistentZone(t *testing.T) {
	m := NewManager("/tmp/agents")

	// Manually add an agent
	m.mu.Lock()
	m.agents["test-agent"] = &Agent{
		ID:     "test-agent",
		Name:   "Test Agent",
		Zone:   "default",
		Status: StatusWorking,
	}
	m.mu.Unlock()

	// Try to move to non-existent zone
	err := m.MoveAgent("test-agent", "non-existent")
	if err == nil {
		t.Error("Should not be able to move agent to non-existent zone")
	}
}

func TestAgentList(t *testing.T) {
	m := NewManager("/tmp/agents")

	// Initially empty
	agents := m.List()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}

	// Add some agents manually
	m.mu.Lock()
	m.agents["agent-1"] = &Agent{ID: "agent-1", Name: "Agent 1", Status: StatusWorking}
	m.agents["agent-2"] = &Agent{ID: "agent-2", Name: "Agent 2", Status: StatusIdle}
	m.mu.Unlock()

	agents = m.List()
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestAgentGet(t *testing.T) {
	m := NewManager("/tmp/agents")

	// Agent doesn't exist
	_, ok := m.Get("non-existent")
	if ok {
		t.Error("Should not find non-existent agent")
	}

	// Add an agent
	m.mu.Lock()
	m.agents["test-agent"] = &Agent{
		ID:     "test-agent",
		Name:   "Test Agent",
		Status: StatusWorking,
	}
	m.mu.Unlock()

	agent, ok := m.Get("test-agent")
	if !ok {
		t.Error("Should find existing agent")
	}

	if agent.Name != "Test Agent" {
		t.Errorf("Expected agent name 'Test Agent', got %s", agent.Name)
	}
}

func TestEventsChannel(t *testing.T) {
	m := NewManager("/tmp/agents")

	events := m.Events()
	if events == nil {
		t.Error("Events channel should not be nil")
	}

	// Test emitting an event
	go func() {
		m.emitEvent("test_event", "agent-1", map[string]string{"key": "value"})
	}()

	select {
	case event := <-events:
		if event.Type != "test_event" {
			t.Errorf("Expected event type 'test_event', got %s", event.Type)
		}
		if event.AgentID != "agent-1" {
			t.Errorf("Expected agent ID 'agent-1', got %s", event.AgentID)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestSpawnRequest(t *testing.T) {
	// Test that SpawnRequest struct has all required fields
	req := SpawnRequest{
		Type:       AgentTypeClaudeCode,
		Name:       "Test Agent",
		Task:       "do something",
		Persona:    "developer",
		Zone:       "default",
		WorkingDir: "/tmp",
		Agent:      "",
	}

	if req.Type != AgentTypeClaudeCode {
		t.Errorf("Expected type claude-code, got %s", req.Type)
	}

	if req.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got %s", req.Name)
	}
}

func TestAgentStatus(t *testing.T) {
	// Test all status constants
	statuses := []AgentStatus{
		StatusStarting,
		StatusWorking,
		StatusIdle,
		StatusWaiting,
		StatusError,
		StatusStopped,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("Status should not be empty")
		}
	}
}

func TestAgentType(t *testing.T) {
	// Test agent type constants
	if AgentTypePython != "python" {
		t.Errorf("Expected AgentTypePython to be 'python', got %s", AgentTypePython)
	}

	if AgentTypeClaudeCode != "claude-code" {
		t.Errorf("Expected AgentTypeClaudeCode to be 'claude-code', got %s", AgentTypeClaudeCode)
	}
}
