// Package openclaw implements a WebSocket bridge to an OpenClaw gateway.
package openclaw

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

// StatusInfo is returned by the /api/openclaw/status endpoint.
type StatusInfo struct {
	State       State      `json:"state"`
	GatewayURL  string     `json:"gatewayUrl"`
	Error       string     `json:"error,omitempty"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty"`
}

// deviceIdentity holds the Ed25519 keypair for device auth.
type deviceIdentity struct {
	DeviceID   string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
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
	device      *deviceIdentity
	stopCh      chan struct{}
	tickMs      int

	pending   map[string]chan *Frame
	pendingMu sync.Mutex

	// EventHandler is called for inbound events (optional).
	EventHandler func(event string, payload json.RawMessage)
}

// NewBridge creates a new OpenClaw bridge (does not connect yet).
func NewBridge(gatewayURL, token string) *Bridge {
	device := loadOrCreateDevice()
	return &Bridge{
		gatewayURL: gatewayURL,
		token:      token,
		state:      StateDisconnected,
		device:     device,
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

	headers := make(map[string][]string)
	headers["Origin"] = []string{"https://darlington.dev"}
	conn, _, err := websocket.DefaultDialer.Dial(b.gatewayURL, headers)
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

	// Extract nonce from challenge
	var challengeData struct {
		Nonce string `json:"nonce"`
	}
	if challenge.Payload != nil {
		json.Unmarshal(challenge.Payload, &challengeData)
	}

	// Build and sign connect params
	signedAt := time.Now().UnixMilli()
	scopes := []string{"operator.read", "operator.write"}

	sigPayload := buildSignaturePayload(
		b.device.DeviceID, "webchat", "webchat",
		"operator", scopes, signedAt,
		b.token, challengeData.Nonce,
	)
	signature := ed25519.Sign(b.device.PrivateKey, []byte(sigPayload))

	params := map[string]interface{}{
		"minProtocol": 3,
		"maxProtocol": 3,
		"client": map[string]string{
			"id":       "webchat",
			"version":  "1.0.0",
			"platform": "linux",
			"mode":     "webchat",
		},
		"role":   "operator",
		"scopes": scopes,
		"caps":   []string{},
		"auth":   map[string]string{"token": b.token},
		"device": map[string]interface{}{
			"id":        b.device.DeviceID,
			"publicKey": toBase64URL(b.device.PublicKey),
			"signature": toBase64URL(signature),
			"signedAt":  signedAt,
			"nonce":     challengeData.Nonce,
		},
		"userAgent": "mc-orchestrator/1.0.0",
		"locale":    "en-US",
	}

	paramsJSON, _ := json.Marshal(params)
	reqID := randomID()
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

	// Parse tick interval
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

	log.Printf("[openclaw] connected to %s (device=%s, tick=%dms)", b.gatewayURL, b.device.DeviceID, b.tickMs)

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

// --- Device identity ---

const deviceKeyFile = ".mc-device-identity.json"

type storedIdentity struct {
	Version    int    `json:"version"`
	DeviceID   string `json:"deviceId"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func loadOrCreateDevice() *deviceIdentity {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, deviceKeyFile)

	// Try to load existing
	if data, err := os.ReadFile(path); err == nil {
		var stored storedIdentity
		if json.Unmarshal(data, &stored) == nil && stored.Version == 1 {
			pub, _ := base64.RawURLEncoding.DecodeString(stored.PublicKey)
			priv, _ := base64.RawURLEncoding.DecodeString(stored.PrivateKey)
			if len(pub) == ed25519.PublicKeySize && len(priv) == ed25519.SeedSize {
				privateKey := ed25519.NewKeyFromSeed(priv)
				return &deviceIdentity{
					DeviceID:   stored.DeviceID,
					PublicKey:  pub,
					PrivateKey: privateKey,
				}
			}
		}
	}

	// Generate new keypair
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	hash := sha256.Sum256(pub)
	deviceID := hex.EncodeToString(hash[:])

	// Persist
	stored := storedIdentity{
		Version:    1,
		DeviceID:   deviceID,
		PublicKey:  toBase64URL(pub),
		PrivateKey: toBase64URL(priv.Seed()),
	}
	data, _ := json.MarshalIndent(stored, "", "  ")
	os.WriteFile(path, data, 0600)

	log.Printf("[openclaw] generated new device identity: %s", deviceID)

	return &deviceIdentity{
		DeviceID:   deviceID,
		PublicKey:  pub,
		PrivateKey: priv,
	}
}

// --- Helpers ---

func buildSignaturePayload(deviceID, clientID, clientMode, role string, scopes []string, signedAtMs int64, token, nonce string) string {
	version := "v1"
	if nonce != "" {
		version = "v2"
	}
	parts := []string{
		version,
		deviceID,
		clientID,
		clientMode,
		role,
		strings.Join(scopes, ","),
		fmt.Sprintf("%d", signedAtMs),
		token,
	}
	if version == "v2" {
		parts = append(parts, nonce)
	}
	return strings.Join(parts, "|")
}

func toBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
