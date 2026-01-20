package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockKing implements KingStarter for testing
type MockKing struct {
	running        bool
	startErr       error
	stopErr        error
	sendErr        error
	answerErr      error
	lastMessage    string
	lastOptionIndex int
}

func (m *MockKing) Start() error {
	if m.startErr != nil {
		return m.startErr
	}
	m.running = true
	return nil
}

func (m *MockKing) Stop() error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.running = false
	return nil
}

func (m *MockKing) IsRunning() bool {
	return m.running
}

func (m *MockKing) SendMessage(message string) error {
	m.lastMessage = message
	return m.sendErr
}

func (m *MockKing) AnswerQuestion(optionIndex int) error {
	m.lastOptionIndex = optionIndex
	return m.answerErr
}

func newTestKingHandler(mock *MockKing) *KingHandler {
	return NewKingHandler(mock, "/tmp/test")
}

func TestKingStart(t *testing.T) {
	mock := &MockKing{}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/king/start", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "started" {
		t.Errorf("Expected status 'started', got %v", resp["status"])
	}

	if !mock.running {
		t.Error("Mock King should be running")
	}
}

func TestKingStartAlreadyRunning(t *testing.T) {
	mock := &MockKing{running: true}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/king/start", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "already_running" {
		t.Errorf("Expected status 'already_running', got %v", resp["status"])
	}
}

func TestKingStartError(t *testing.T) {
	mock := &MockKing{startErr: errors.New("failed to start")}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/king/start", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestKingStartMethodNotAllowed(t *testing.T) {
	mock := &MockKing{}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/king/start", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestKingStop(t *testing.T) {
	mock := &MockKing{running: true}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/king/stop", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "stopped" {
		t.Errorf("Expected status 'stopped', got %v", resp["status"])
	}

	if mock.running {
		t.Error("Mock King should be stopped")
	}
}

func TestKingStopNotRunning(t *testing.T) {
	mock := &MockKing{running: false}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/king/stop", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "already_stopped" {
		t.Errorf("Expected status 'already_stopped', got %v", resp["status"])
	}
}

func TestKingStopError(t *testing.T) {
	mock := &MockKing{running: true, stopErr: errors.New("failed to stop")}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/king/stop", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestKingStatus(t *testing.T) {
	tests := []struct {
		name     string
		running  bool
		expected bool
	}{
		{"running", true, true},
		{"stopped", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockKing{running: tt.running}
			h := newTestKingHandler(mock)

			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			req := httptest.NewRequest("GET", "/api/king/status", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected 200, got %d", w.Code)
			}

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if resp["is_running"] != tt.expected {
				t.Errorf("Expected is_running=%v, got %v", tt.expected, resp["is_running"])
			}
		})
	}
}

func TestKingMessage(t *testing.T) {
	mock := &MockKing{running: true}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"content":"Hello King!"}`)
	req := httptest.NewRequest("POST", "/api/king/message", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if mock.lastMessage != "Hello King!" {
		t.Errorf("Expected message 'Hello King!', got '%s'", mock.lastMessage)
	}
}

func TestKingMessageNotRunning(t *testing.T) {
	mock := &MockKing{running: false}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"content":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/king/message", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestKingMessageInvalidBody(t *testing.T) {
	mock := &MockKing{running: true}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest("POST", "/api/king/message", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestKingMessageSendError(t *testing.T) {
	mock := &MockKing{running: true, sendErr: errors.New("send failed")}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"content":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/king/message", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestKingAnswer(t *testing.T) {
	mock := &MockKing{running: true}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"option_index":2}`)
	req := httptest.NewRequest("POST", "/api/king/answer", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if mock.lastOptionIndex != 2 {
		t.Errorf("Expected option_index 2, got %d", mock.lastOptionIndex)
	}
}

func TestKingAnswerNotRunning(t *testing.T) {
	mock := &MockKing{running: false}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"option_index":0}`)
	req := httptest.NewRequest("POST", "/api/king/answer", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestKingAnswerError(t *testing.T) {
	mock := &MockKing{running: true, answerErr: errors.New("answer failed")}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"option_index":0}`)
	req := httptest.NewRequest("POST", "/api/king/answer", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestKingFullLifecycle(t *testing.T) {
	mock := &MockKing{}
	h := newTestKingHandler(mock)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 1. Check status - should be stopped
	req := httptest.NewRequest("GET", "/api/king/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var statusResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &statusResp)
	if statusResp["is_running"] != false {
		t.Error("King should not be running initially")
	}

	// 2. Start King
	req = httptest.NewRequest("POST", "/api/king/start", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Start failed: %s", w.Body.String())
	}

	// 3. Send a message
	body := bytes.NewBufferString(`{"content":"Test message"}`)
	req = httptest.NewRequest("POST", "/api/king/message", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Message failed: %s", w.Body.String())
	}

	if mock.lastMessage != "Test message" {
		t.Errorf("Expected 'Test message', got '%s'", mock.lastMessage)
	}

	// 4. Answer a question
	body = bytes.NewBufferString(`{"option_index":1}`)
	req = httptest.NewRequest("POST", "/api/king/answer", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Answer failed: %s", w.Body.String())
	}

	// 5. Stop King
	req = httptest.NewRequest("POST", "/api/king/stop", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stop failed: %s", w.Body.String())
	}

	// 6. Verify stopped
	req = httptest.NewRequest("GET", "/api/king/status", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &statusResp)
	if statusResp["is_running"] != false {
		t.Error("King should be stopped")
	}
}
