package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	os.MkdirAll(stateDir, 0755)
	os.MkdirAll(filepath.Join(dir, "findings"), 0755)
	os.MkdirAll(filepath.Join(dir, "handoffs"), 0755)

	os.WriteFile(filepath.Join(stateDir, "stage.json"),
		[]byte(`{"current":"implement","updated_at":"2026-01-01T00:00:00Z"}`), 0644)
	os.WriteFile(filepath.Join(stateDir, "gates.json"),
		[]byte(`{"discovery":{"stage":"discovery","status":"approved"}}`), 0644)
	os.WriteFile(filepath.Join(stateDir, "workers.json"),
		[]byte(`{"workers":[]}`), 0644)
	os.WriteFile(filepath.Join(stateDir, "tasks.jsonl"),
		[]byte(`{"id":"t1","name":"Test","status":"pending","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`+"\n"), 0644)

	return dir
}

func TestNewWatcher(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if w == nil {
		t.Fatal("NewWatcher returned nil")
	}
}

func TestStartStop(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	w.Stop()
}

func TestEventsChannel(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	ch := w.Events()
	if ch == nil {
		t.Fatal("Events() returned nil")
	}
}

func TestDetectsStageChange(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	// Let initial state load
	time.Sleep(600 * time.Millisecond)

	// Change stage
	os.WriteFile(filepath.Join(dir, "state", "stage.json"),
		[]byte(`{"current":"verify","updated_at":"2026-01-02T00:00:00Z"}`), 0644)

	// Wait for poll
	timeout := time.After(3 * time.Second)
	for {
		select {
		case event := <-w.Events():
			if event.Type == "stage_changed" {
				return // Pass
			}
		case <-timeout:
			t.Fatal("timeout waiting for stage.changed event")
		}
	}
}

func TestDetectsNewTask(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	time.Sleep(600 * time.Millisecond)

	// Add a new task
	f, _ := os.OpenFile(filepath.Join(dir, "state", "tasks.jsonl"), os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(`{"id":"t2","name":"New task","status":"pending","created_at":"2026-01-02T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}` + "\n")
	f.Close()

	timeout := time.After(3 * time.Second)
	for {
		select {
		case event := <-w.Events():
			if event.Type == "task_created" {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for task.created event")
		}
	}
}

func TestGetCurrentState(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	time.Sleep(600 * time.Millisecond)

	state := w.GetCurrentState()
	if state == nil {
		t.Fatal("GetCurrentState returned nil")
	}
	if state["stage"] == nil {
		t.Error("state missing 'stage'")
	}
}

func TestReadTasksJSONLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.jsonl")

	lines := []string{
		`{"id":"t1","name":"Task 1","status":"pending"}`,
		`{"id":"t2","name":"Task 2","status":"done"}`,
	}
	os.WriteFile(path, []byte(lines[0]+"\n"+lines[1]+"\n"), 0644)

	tasks, err := readTasksJSONLFile(path)
	if err != nil {
		t.Fatalf("readTasksJSONLFile failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "t1" {
		t.Errorf("expected id t1, got %s", tasks[0].ID)
	}
}

func TestReadTasksJSONLFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	tasks, err := readTasksJSONLFile(path)
	if err != nil {
		t.Fatalf("readTasksJSONLFile failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestReadTasksJSONLFileMissing(t *testing.T) {
	_, err := readTasksJSONLFile("/nonexistent/tasks.jsonl")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDetectsFindings(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	time.Sleep(600 * time.Millisecond)

	// Create a finding
	finding := map[string]string{"title": "Test finding"}
	data, _ := json.Marshal(finding)
	os.WriteFile(filepath.Join(dir, "findings", "abc123.md"), data, 0644)

	timeout := time.After(3 * time.Second)
	for {
		select {
		case event := <-w.Events():
			if event.Type == "findings_ready" {
				// Verify task_id extraction
				if m, ok := event.Data.(map[string]interface{}); ok {
					if m["task_id"] != "abc123" {
						t.Errorf("expected task_id abc123, got %v", m["task_id"])
					}
				}
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for findings_ready event")
		}
	}
}

func TestDetectsHandoffs(t *testing.T) {
	dir := createTestDir(t)
	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	time.Sleep(600 * time.Millisecond)

	// Create a handoff briefing
	os.WriteFile(filepath.Join(dir, "handoffs", "abc123-briefing.json"), []byte(`{"task_id":"abc123"}`), 0644)

	timeout := time.After(3 * time.Second)
	for {
		select {
		case event := <-w.Events():
			if event.Type == "handoff_created" {
				if m, ok := event.Data.(map[string]interface{}); ok {
					if m["task_id"] != "abc123" {
						t.Errorf("expected task_id abc123, got %v", m["task_id"])
					}
				}
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for handoff_created event")
		}
	}
}

func TestIgnoresExistingFindings(t *testing.T) {
	dir := createTestDir(t)

	// Pre-create a finding before watcher starts
	os.WriteFile(filepath.Join(dir, "findings", "existing.md"), []byte("old"), 0644)

	w := NewWatcher(dir)
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	time.Sleep(1200 * time.Millisecond)

	// Drain events — should have no findings_ready for "existing"
	for {
		select {
		case event := <-w.Events():
			if event.Type == "findings_ready" {
				if m, ok := event.Data.(map[string]interface{}); ok {
					if m["task_id"] == "existing" {
						t.Fatal("should not emit findings_ready for pre-existing file")
					}
				}
			}
		default:
			return // no more events, pass
		}
	}
}
