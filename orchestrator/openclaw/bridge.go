// Package openclaw implements a WebSocket bridge to an OpenClaw gateway.
package openclaw

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// State represents the current connection state.
type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateError        State = "error"
)

// Frame is a generic gateway protocol frame.
type Frame struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Event   string          `json:"event,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	OK      *bool           `json:"ok,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// ConnectParams is the connect request payload.
type ConnectParams struct {
	MinProtocol int             `json:"minProtocol"`
	MaxProtocol int             `json:"maxProtocol"`
	Client      ConnectClient   `json:"client"`
	Role        string          `json:"role"`
	Scopes      []string        `json:"scopes"`
	Caps        []string        `json:"caps"`
	Commands    []string        `json:"commands"`
	Permissions map[string]bool `json:"permissions"`
	Auth        *ConnectAuth    `json:"auth,omitempty"`
	Locale      string          `json:"locale"`
	UserAgent   string          `json:"userAgent"`
	Device      ConnectDevice   `json:"device"`
}

type ConnectClient struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
	Mode     string `json:"mode"`
}

type ConnectAuth struct {
	Token string `json:"token"`
}

type ConnectDevice struct {
	ID string `json:"id"`
}

// StatusInfo is returned by the /api/openclaw/status endpoint.
type StatusInfo struct {
	State       State      `json:"state"`
	GatewayURL  string     `json:"gatewayUrl"`
	Error       string     `json:"error,omitempty"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty"`
}

// Bridge manages the WebSocket connection to the OpenClaw gateway.
type Bridge struct {
	mu          sync.RWMutex
	gatewayURL  string
	token       string
	state       State
	err         string
	conn        *websocket.Conn
	connectedAt *time.Time
	deviceID    string
	stopCh      chan struct{}
	tickMs      int

	pending   map[string]chan *Frame
	pendingMu sync.Mutex

	// EventHandler is called for inbound events (optional).
	EventHandler func(event string, payload json.RawMessage)
}

// NewBridge creates a new OpenClaw bridge (does not connect yet).
func NewBridge(gatewayURL, token string) *Bridge {
	devID := make([]byte, 8)
	rand.Read(devID)
	return &Bridge{
		gatewayURL: gatewayURL,
		token:      token,
		state:      StateDisconnected,
		deviceID:   "mc-orch-" + hex.EncodeToString(devID),
		pending:    make(map[string]chan *Frame),
		tickMs:     15000,
	}
}

// Status returns the current connection status.
func (b *Bridge) Status() StatusInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return StatusInfo{
		State:       b.state,
		GatewayURL:  b.gatewayURL,
		Error:       b.err,
		ConnectedAt: b.connectedAt,
	}
}

// Connect dials the gateway and performs the protocol handshake.
func (b *Bridge) Connect() error {
	b.mu.Lock()
	b.state = StateConnecting
	b.err = ""
	b.mu.Unlock()

	conn, _, err := websocket.DefaultDialer.Dial(b.gatewayURL, nil)
	if err != nil {
		b.setError(fmt.Sprintf("dial: %v", err))
		return err
	}

	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()

	// Read the connect.challenge event
	var challenge Frame
	if err := conn.ReadJSON(&challenge); err != nil {
		conn.Close()
		b.setError(fmt.Sprintf("read challenge: %v", err))
		return err
	}
	if challenge.Event != "connect.challenge" {
		conn.Close()
		b.setError("expected connect.challenge, got: " + challenge.Event)
		return fmt.Errorf("unexpected first frame: %s", challenge.Event)
	}

	// Send connect request
	reqID := randomID()
	params := ConnectParams{
		MinProtocol: 3,
		MaxProtocol: 3,
		Client: ConnectClient{
			ID:       "mc-orchestrator",
			Version:  "0.1.0",
			Platform: "linux",
			Mode:     "operator",
		},
		Role:        "operator",
		Scopes:      []string{"operator.read", "operator.write"},
		Caps:        []string{},
		Commands:    []string{},
		Permissions: map[string]bool{},
		Locale:      "en-US",
		UserAgent:   "mc-orchestrator/0.1.0",
		Device:      ConnectDevice{ID: b.deviceID},
	}
	if b.token != "" {
		params.Auth = &ConnectAuth{Token: b.token}
	}

	paramsJSON, _ := json.Marshal(params)
	connectReq := Frame{
		Type:   "req",
		ID:     reqID,
		Method: "connect",
		Params: paramsJSON,
	}
	if err := conn.WriteJSON(connectReq); err != nil {
		conn.Close()
		b.setError(fmt.Sprintf("write connect: %v", err))
		return err
	}

	// Read connect response
	var resp Frame
	if err := conn.ReadJSON(&resp); err != nil {
		conn.Close()
		b.setError(fmt.Sprintf("read connect response: %v", err))
		return err
	}
	if resp.OK == nil || !*resp.OK {
		conn.Close()
		errMsg := string(resp.Error)
		if errMsg == "" {
			errMsg = "connect rejected"
		}
		b.setError(errMsg)
		return fmt.Errorf("connect failed: %s", errMsg)
	}

	// Parse tick interval from hello-ok payload
	var helloOK struct {
		Policy struct {
			TickIntervalMs int `json:"tickIntervalMs"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(resp.Payload, &helloOK); err == nil && helloOK.Policy.TickIntervalMs > 0 {
		b.tickMs = helloOK.Policy.TickIntervalMs
	}

	now := time.Now()
	b.mu.Lock()
	b.state = StateConnected
	b.connectedAt = &now
	b.mu.Unlock()

	log.Printf("[openclaw] connected to %s (tick=%dms)", b.gatewayURL, b.tickMs)

	b.stopCh = make(chan struct{})
	go b.readLoop()
	go b.tickLoop()

	return nil
}

// Send sends a request and waits for a response (up to 30s).
func (b *Bridge) Send(method string, params interface{}) (*Frame, error) {
	b.mu.RLock()
	conn := b.conn
	state := b.state
	b.mu.RUnlock()

	if state != StateConnected || conn == nil {
		return nil, fmt.Errorf("not connected (state=%s)", state)
	}

	reqID := randomID()
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Frame, 1)
	b.pendingMu.Lock()
	b.pending[reqID] = ch
	b.pendingMu.Unlock()

	frame := Frame{
		Type:   "req",
		ID:     reqID,
		Method: method,
		Params: paramsJSON,
	}
	if err := conn.WriteJSON(frame); err != nil {
		b.pendingMu.Lock()
		delete(b.pending, reqID)
		b.pendingMu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(30 * time.Second):
		b.pendingMu.Lock()
		delete(b.pending, reqID)
		b.pendingMu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response to %s", method)
	}
}

// Close shuts down the bridge.
func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopCh != nil {
		close(b.stopCh)
		b.stopCh = nil
	}
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
	b.state = StateDisconnected
}

func (b *Bridge) readLoop() {
	for {
		var frame Frame
		if err := b.conn.ReadJSON(&frame); err != nil {
			select {
			case <-b.stopCh:
				return
			default:
			}
			log.Printf("[openclaw] read error: %v", err)
			b.setError(fmt.Sprintf("read: %v", err))
			return
		}

		switch frame.Type {
		case "res":
			b.pendingMu.Lock()
			if ch, ok := b.pending[frame.ID]; ok {
				ch <- &frame
				delete(b.pending, frame.ID)
			}
			b.pendingMu.Unlock()
		case "event":
			if b.EventHandler != nil {
				b.EventHandler(frame.Event, frame.Payload)
			}
		}
	}
}

func (b *Bridge) tickLoop() {
	ticker := time.NewTicker(time.Duration(b.tickMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.mu.RLock()
			conn := b.conn
			b.mu.RUnlock()
			if conn != nil {
				tick := Frame{Type: "req", ID: randomID(), Method: "tick"}
				if err := conn.WriteJSON(tick); err != nil {
					log.Printf("[openclaw] tick error: %v", err)
				}
			}
		}
	}
}

func (b *Bridge) setError(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateError
	b.err = msg
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
