package bridge

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewKing(t *testing.T) {
	workDir := "/tmp/test-mission"
	king := NewKing(workDir)

	if king.workDir != workDir {
		t.Errorf("expected workDir %s, got %s", workDir, king.workDir)
	}

	expectedMissionDir := filepath.Join(workDir, ".mission")
	if king.missionDir != expectedMissionDir {
		t.Errorf("expected missionDir %s, got %s", expectedMissionDir, king.missionDir)
	}

	if king.status != KingStatusStopped {
		t.Errorf("expected initial status %s, got %s", KingStatusStopped, king.status)
	}

	if king.tmuxSession != kingTmuxSession {
		t.Errorf("expected tmuxSession %s, got %s", kingTmuxSession, king.tmuxSession)
	}

	if king.events == nil {
		t.Error("events channel should not be nil")
	}
}

func TestKingStatus(t *testing.T) {
	king := NewKing("/tmp/test")

	if king.Status() != KingStatusStopped {
		t.Errorf("expected status %s, got %s", KingStatusStopped, king.Status())
	}
}

func TestKingIsRunning(t *testing.T) {
	king := NewKing("/tmp/test")

	if king.IsRunning() {
		t.Error("expected IsRunning to be false initially")
	}
}

func TestKingStartWithoutCLAUDE_MD(t *testing.T) {
	// Create temp dir without CLAUDE.md
	tmpDir, err := os.MkdirTemp("", "king-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	king := NewKing(tmpDir)
	err = king.Start()

	if err == nil {
		t.Error("expected error when CLAUDE.md doesn't exist")
	}

	if king.Status() != KingStatusError {
		t.Errorf("expected status %s after failed start, got %s", KingStatusError, king.Status())
	}
}

func TestKingStartWithCLAUDE_MD(t *testing.T) {
	// Create temp dir with .mission/CLAUDE.md
	tmpDir, err := os.MkdirTemp("", "king-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	if err := os.MkdirAll(missionDir, 0755); err != nil {
		t.Fatal(err)
	}

	claudeMD := filepath.Join(missionDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("# King Prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	king := NewKing(tmpDir)
	err = king.Start()

	// Note: Start() may fail if claude CLI is not installed, which is OK for unit tests
	if err != nil {
		t.Skipf("skipping test - claude CLI may not be available: %v", err)
	}

	defer king.Stop() // Clean up the spawned process

	if king.Status() != KingStatusRunning {
		t.Errorf("expected status %s after start, got %s", KingStatusRunning, king.Status())
	}

	if !king.IsRunning() {
		t.Error("expected IsRunning to be true after start")
	}
}

func TestKingStartAlreadyRunning(t *testing.T) {
	// Create temp dir with .mission/CLAUDE.md
	tmpDir, err := os.MkdirTemp("", "king-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	if err := os.MkdirAll(missionDir, 0755); err != nil {
		t.Fatal(err)
	}

	claudeMD := filepath.Join(missionDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("# King Prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	king := NewKing(tmpDir)

	// Start first time
	if err := king.Start(); err != nil {
		t.Skipf("skipping test - claude CLI may not be available: %v", err)
	}
	defer king.Stop()

	// Try to start again
	err = king.Start()
	if err == nil {
		t.Error("expected error when starting already running King")
	}
}

func TestKingStop(t *testing.T) {
	// Create temp dir with .mission/CLAUDE.md
	tmpDir, err := os.MkdirTemp("", "king-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	if err := os.MkdirAll(missionDir, 0755); err != nil {
		t.Fatal(err)
	}

	claudeMD := filepath.Join(missionDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("# King Prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	king := NewKing(tmpDir)

	// Start
	if err := king.Start(); err != nil {
		t.Skipf("skipping test - claude CLI may not be available: %v", err)
	}

	// Stop
	if err := king.Stop(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if king.Status() != KingStatusStopped {
		t.Errorf("expected status %s after stop, got %s", KingStatusStopped, king.Status())
	}

	if king.lastPane != "" {
		t.Error("expected lastPane to be cleared after stop")
	}
}

func TestKingStopNotRunning(t *testing.T) {
	king := NewKing("/tmp/test")

	err := king.Stop()
	if err == nil {
		t.Error("expected error when stopping non-running King")
	}
}

func TestKingEventsChannel(t *testing.T) {
	king := NewKing("/tmp/test")

	events := king.Events()
	if events == nil {
		t.Error("Events() should return a channel")
	}

	// Emit an event
	go king.emitEvent("test_event", map[string]interface{}{"foo": "bar"})

	// Receive the event with timeout
	select {
	case event := <-events:
		if event.Type != "test_event" {
			t.Errorf("expected event type 'test_event', got '%s'", event.Type)
		}
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Error("expected event data to be a map")
		} else if data["foo"] != "bar" {
			t.Errorf("expected data[foo] = 'bar', got '%v'", data["foo"])
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestKingEventChannelFull(t *testing.T) {
	king := NewKing("/tmp/test")

	// Fill the event channel (capacity 100)
	for i := 0; i < 100; i++ {
		king.emitEvent("filler", nil)
	}

	// This should not block - event should be dropped
	done := make(chan bool)
	go func() {
		king.emitEvent("overflow", nil)
		done <- true
	}()

	select {
	case <-done:
		// Good - didn't block
	case <-time.After(time.Second):
		t.Error("emitEvent blocked when channel was full")
	}
}

func TestKingSessionExists(t *testing.T) {
	king := NewKing("/tmp/test")

	// With no tmux session created, this should return false
	// (assumes no mc-king session is running from other tests)
	if king.sessionExists() {
		// Clean up any stale session
		king.killSession()
	}

	if king.sessionExists() {
		t.Error("expected sessionExists to be false when no session exists")
	}
}

func TestKingExtractResponseAfterMessage(t *testing.T) {
	king := NewKing("/tmp/test")

	testCases := []struct {
		name        string
		pane        string
		userMessage string
		expected    string
	}{
		{
			name:        "simple response",
			pane:        "❯ Say hello\n⏺ Hello, I'm Claude!\n❯ ",
			userMessage: "Say hello",
			expected:    "Hello, I'm Claude!",
		},
		{
			name:        "no message found",
			pane:        "❯ \n",
			userMessage: "nonexistent",
			expected:    "",
		},
		{
			name:        "multiline response",
			pane:        "❯ Test\n⏺ Line one\n  Line two\n  Line three\n───\n❯ ",
			userMessage: "Test",
			expected:    "Line one\nLine two\nLine three",
		},
		{
			name:        "response with thinking",
			pane:        "❯ Hi\n∴ Thinking...\n⏺ Hello there!\n❯ ",
			userMessage: "Hi",
			expected:    "Hello there!",
		},
		{
			name:        "multiple exchanges - get latest",
			pane:        "❯ First\n⏺ First response\n❯ Second\n⏺ Second response\n❯ ",
			userMessage: "Second",
			expected:    "Second response",
		},
		{
			name:        "ignore old response",
			pane:        "❯ Old message\n⏺ Old response\n❯ New message\n⏺ New response\n❯ ",
			userMessage: "New message",
			expected:    "New response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := king.extractResponseAfterMessage(tc.pane, tc.userMessage)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestKingIsQuestionUI(t *testing.T) {
	king := NewKing("/tmp/test")

	testCases := []struct {
		name     string
		pane     string
		expected bool
	}{
		{
			name:     "normal prompt",
			pane:     "⏺ Hello!\n❯ ",
			expected: false,
		},
		{
			name:     "question UI with options",
			pane:     "☐ Task\nWhat would you like?\n❯ 1. First\n  2. Second\nEnter to select · ↑/↓ to navigate",
			expected: true,
		},
		{
			name:     "question UI with circles",
			pane:     "Choose an option:\n○ Option A\n● Option B\nEnter to select",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := king.isQuestionUI(tc.pane)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestKingParseQuestion(t *testing.T) {
	king := NewKing("/tmp/test")

	pane := `☐ Task

What would you like to work on today?

❯ 1. Review changes
     Review the uncommitted changes
  2. New feature
     Implement a new feature
  3. Bug fix
     Fix an existing bug

Enter to select · ↑/↓ to navigate`

	question := king.parseQuestion(pane)

	if question == nil {
		t.Fatal("expected question to be parsed")
	}

	if question.Question != "What would you like to work on today?" {
		t.Errorf("expected question text, got %q", question.Question)
	}

	if len(question.Options) != 3 {
		t.Errorf("expected 3 options, got %d", len(question.Options))
	}

	if question.Selected != 0 {
		t.Errorf("expected selected index 0, got %d", question.Selected)
	}

	expectedOptions := []string{"Review changes", "New feature", "Bug fix"}
	for i, opt := range expectedOptions {
		if i < len(question.Options) && question.Options[i] != opt {
			t.Errorf("option %d: expected %q, got %q", i, opt, question.Options[i])
		}
	}
}

func TestKingIsResponseComplete(t *testing.T) {
	king := NewKing("/tmp/test")

	testCases := []struct {
		name     string
		pane     string
		expected bool
	}{
		{
			name:     "prompt visible",
			pane:     "Some output\n❯ ",
			expected: true,
		},
		{
			name:     "thinking indicator",
			pane:     "Some output\n∴ Thinking...",
			expected: false,
		},
		{
			name:     "spinner visible",
			pane:     "Some output\n⠋ Working...",
			expected: false,
		},
		{
			name:     "empty pane",
			pane:     "",
			expected: false,
		},
		{
			name:     "prompt after response",
			pane:     "⏺ Hello!\n───\n❯ ",
			expected: true,
		},
		{
			name:     "selection UI - not complete",
			pane:     "⏺ Question?\n❯ 1. Option one\n  2. Option two\nEnter to select · ↑/↓ to navigate",
			expected: false,
		},
		{
			name:     "checkbox UI - not complete",
			pane:     "☐ Task\nWhat would you like?\n❯ 1. First\nEnter to select",
			expected: false,
		},
		{
			name:     "prompt with user text",
			pane:     "⏺ Response\n❯ hello world",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := king.isResponseComplete(tc.pane)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
