package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DarlingtonDeveloper/MissionControl/manager"
	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("clients map should be initialized")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel should be initialized")
	}

	if hub.register == nil {
		t.Error("register channel should be initialized")
	}

	if hub.unregister == nil {
		t.Error("unregister channel should be initialized")
	}
}

func TestHubNotify(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	// Start hub in background
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Notify should not block even with no clients
	done := make(chan bool)
	go func() {
		hub.Notify(map[string]string{"type": "test_event"})
		done <- true
	}()

	select {
	case <-done:
		// Success - didn't block
	case <-time.After(time.Second):
		t.Error("Notify blocked with no clients")
	}
}

func TestHubSetProviders(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	// Test SetV4StateProvider
	hub.SetV4StateProvider(func() interface{} {
		return map[string]string{"state": "test"}
	})

	if hub.v4StateProvider == nil {
		t.Error("v4StateProvider should be set")
	}

	// Test SetMissionStateProvider
	hub.SetMissionStateProvider(func() interface{} {
		return map[string]string{"mission": "test"}
	})

	if hub.missionStateProvider == nil {
		t.Error("missionStateProvider should be set")
	}
}

func TestWebSocketConnection(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	// Start hub
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Should receive initial state (agent_list)
	ws.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var event map[string]interface{}
	if err := json.Unmarshal(message, &event); err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	if event["type"] != "agent_list" {
		t.Errorf("Expected first message to be agent_list, got %v", event["type"])
	}
}

func TestWebSocketReceivesZoneList(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Read messages until we get zone_list
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	foundZoneList := false

	for i := 0; i < 5; i++ {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var event map[string]interface{}
		json.Unmarshal(message, &event)

		if event["type"] == "zone_list" {
			foundZoneList = true
			zones, ok := event["zones"].([]interface{})
			if !ok {
				t.Error("zones should be an array")
			}
			if len(zones) < 1 {
				t.Error("Should have at least default zone")
			}
			break
		}
	}

	if !foundZoneList {
		t.Error("Did not receive zone_list event")
	}
}

func TestWebSocketBroadcast(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	go hub.Run()
	time.Sleep(50 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect a client
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer ws1.Close()

	// Wait for client to register
	time.Sleep(100 * time.Millisecond)

	// Broadcast a message - it will be queued after initial sync messages
	testEvent := map[string]string{"type": "test_broadcast", "data": "hello"}
	hub.Notify(testEvent)

	// Read messages until we get our broadcast or timeout
	ws1.SetReadDeadline(time.Now().Add(2 * time.Second))
	found := false
	for i := 0; i < 10; i++ {
		_, msg, err := ws1.ReadMessage()
		if err != nil {
			break
		}
		var event map[string]string
		json.Unmarshal(msg, &event)
		if event["type"] == "test_broadcast" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Did not receive broadcast event")
	}
}

func TestWebSocketCommandRequestSync(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	go hub.Run()
	time.Sleep(50 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Wait for initial sync and then send request_sync
	time.Sleep(100 * time.Millisecond)

	// Send request_sync command
	cmd := Command{Type: "request_sync"}
	cmdData, _ := json.Marshal(cmd)
	ws.WriteMessage(websocket.TextMessage, cmdData)

	// Read messages - should get agent_list (from initial or sync)
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	foundAgentList := false
	for i := 0; i < 10; i++ {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var event map[string]interface{}
		json.Unmarshal(message, &event)
		if event["type"] == "agent_list" {
			foundAgentList = true
			// Verify it has agents array
			if _, ok := event["agents"]; !ok {
				t.Error("agent_list should have agents field")
			}
			break
		}
	}

	if !foundAgentList {
		t.Error("Did not receive agent_list event")
	}
}

func TestWebSocketDisconnect(t *testing.T) {
	m := manager.NewManager("/tmp/test")
	hub := NewHub(m)

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Close connection
	ws.Close()

	// Wait for unregistration
	time.Sleep(50 * time.Millisecond)

	// Hub should handle disconnect gracefully (no panic)
}
