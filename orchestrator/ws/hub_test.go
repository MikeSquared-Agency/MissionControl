package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func setupHub(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	hub := NewHub()
	go hub.Run()
	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	return hub, server
}

func dialWS(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func readEvent(t *testing.T, conn *websocket.Conn) Event {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ev Event
	if err := json.Unmarshal(msg, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return ev
}

func TestHubAcceptsConnections(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	conn := dialWS(t, server)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)
	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}
}

func TestBroadcastSendsToAll(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	c1 := dialWS(t, server)
	defer c1.Close()
	c2 := dialWS(t, server)
	defer c2.Close()

	time.Sleep(50 * time.Millisecond)

	hub.BroadcastRaw("worker", "spawned", map[string]string{"worker_id": "mc-a1b2c"})

	ev1 := readEvent(t, c1)
	ev2 := readEvent(t, c2)

	if ev1.Topic != "worker" || ev1.Type != "spawned" {
		t.Fatalf("unexpected event: %+v", ev1)
	}
	if ev2.Topic != "worker" || ev2.Type != "spawned" {
		t.Fatalf("unexpected event: %+v", ev2)
	}
}

func TestSubscribeFiltersTopics(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	conn := dialWS(t, server)
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// Subscribe to only "task"
	cmd, _ := json.Marshal(clientCommand{Type: "subscribe", Topics: []string{"task"}})
	conn.WriteMessage(websocket.TextMessage, cmd)
	time.Sleep(50 * time.Millisecond)

	// Broadcast a "worker" event — should be filtered out
	hub.BroadcastRaw("worker", "spawned", map[string]string{})
	// Broadcast a "task" event — should arrive
	hub.BroadcastRaw("task", "started", map[string]string{"id": "t1"})

	ev := readEvent(t, conn)
	if ev.Topic != "task" {
		t.Fatalf("expected task event, got %s", ev.Topic)
	}
}

func TestUnsubscribe(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	conn := dialWS(t, server)
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// Subscribe to task and worker
	cmd, _ := json.Marshal(clientCommand{Type: "subscribe", Topics: []string{"task", "worker"}})
	conn.WriteMessage(websocket.TextMessage, cmd)
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe from worker
	cmd, _ = json.Marshal(clientCommand{Type: "unsubscribe", Topics: []string{"worker"}})
	conn.WriteMessage(websocket.TextMessage, cmd)
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastRaw("worker", "spawned", map[string]string{})
	hub.BroadcastRaw("task", "done", map[string]string{})

	ev := readEvent(t, conn)
	if ev.Topic != "task" {
		t.Fatalf("expected task, got %s", ev.Topic)
	}
}

func TestInitialStateSync(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	hub.SetStateProvider(func() interface{} {
		return map[string]string{"status": "ready"}
	})
	time.Sleep(50 * time.Millisecond)

	conn := dialWS(t, server)
	defer conn.Close()

	ev := readEvent(t, conn)
	if ev.Topic != "sync" || ev.Type != "initial_state" {
		t.Fatalf("expected sync/initial_state, got %s/%s", ev.Topic, ev.Type)
	}
}

func TestClientDisconnect(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	conn := dialWS(t, server)
	time.Sleep(50 * time.Millisecond)
	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	conn.Close()
	time.Sleep(100 * time.Millisecond)

	// Trigger unregister by broadcasting
	hub.BroadcastRaw("task", "ping", nil)
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	t.Setenv("MC_API_TOKEN", "secret123")

	hub := NewHub()
	go hub.Run()
	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	// No token — should fail
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	// Wrong token
	_, _, err = websocket.DefaultDialer.Dial(url+"?token=wrong", nil)
	if err == nil {
		t.Fatal("expected error for wrong token")
	}

	// Correct token via query
	conn, _, err := websocket.DefaultDialer.Dial(url+"?token=secret123", nil)
	if err != nil {
		t.Fatalf("expected success with correct token: %v", err)
	}
	conn.Close()

	// Correct token via header
	header := http.Header{}
	header.Set("Authorization", "Bearer secret123")
	conn, _, err = websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatalf("expected success with bearer token: %v", err)
	}
	conn.Close()
}

func TestBroadcastRawMarshals(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	conn := dialWS(t, server)
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastRaw("gate", "passed", map[string]interface{}{
		"stage": 3,
		"name":  "implement",
	})

	ev := readEvent(t, conn)
	if ev.Topic != "gate" || ev.Type != "passed" {
		t.Fatalf("unexpected: %+v", ev)
	}
	var data map[string]interface{}
	json.Unmarshal(ev.Data, &data)
	if data["name"] != "implement" {
		t.Fatalf("unexpected data: %v", data)
	}
}

func TestRequestSync(t *testing.T) {
	hub, server := setupHub(t)
	defer server.Close()

	// Connect before state provider is set — no sync event
	conn := dialWS(t, server)
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	hub.SetStateProvider(func() interface{} {
		return map[string]string{"phase": "2"}
	})

	cmd, _ := json.Marshal(clientCommand{Type: "request_sync"})
	conn.WriteMessage(websocket.TextMessage, cmd)

	ev := readEvent(t, conn)
	if ev.Topic != "sync" || ev.Type != "initial_state" {
		t.Fatalf("expected sync, got %s/%s", ev.Topic, ev.Type)
	}
}
