package tracker

import (
	"testing"
	"time"
)

func TestNewTrackerEmpty(t *testing.T) {
	tr := NewTracker("/tmp/test", nil)
	if len(tr.List()) != 0 {
		t.Fatal("expected empty process list")
	}
}

func TestListAndGet(t *testing.T) {
	tr := NewTracker("/tmp/test", nil)

	// Manually inject a process.
	tr.mu.Lock()
	tr.processes["w1"] = &TrackedProcess{
		WorkerID:  "w1",
		Persona:   "coder",
		TaskID:    "t1",
		Status:    StatusRunning,
		PID:       99999,
		StartedAt: time.Now(),
	}
	tr.mu.Unlock()

	list := tr.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 process, got %d", len(list))
	}
	if list[0].WorkerID != "w1" {
		t.Fatalf("expected worker_id w1, got %s", list[0].WorkerID)
	}

	p, ok := tr.Get("w1")
	if !ok || p.WorkerID != "w1" {
		t.Fatal("Get failed for w1")
	}

	_, ok = tr.Get("nonexistent")
	if ok {
		t.Fatal("Get should return false for missing worker")
	}
}

func TestKillNonexistent(t *testing.T) {
	tr := NewTracker("/tmp/test", nil)
	err := tr.Kill("nope")
	if err == nil {
		t.Fatal("expected error killing non-tracked worker")
	}
}

func TestKillUpdatesStatus(t *testing.T) {
	tr := NewTracker("/tmp/test", nil)

	// Use PID 1 which exists but we can't signal — Kill should still mark killed.
	tr.mu.Lock()
	tr.processes["w1"] = &TrackedProcess{
		WorkerID: "w1",
		PID:      999999, // almost certainly not running
		Status:   StatusRunning,
	}
	tr.mu.Unlock()

	_ = tr.Kill("w1")

	p, _ := tr.Get("w1")
	if p.Status != StatusKilled {
		t.Fatalf("expected status killed, got %s", p.Status)
	}
}

func TestReset(t *testing.T) {
	tr := NewTracker("/tmp/test", nil)
	tr.mu.Lock()
	tr.processes["w1"] = &TrackedProcess{WorkerID: "w1"}
	tr.processes["w2"] = &TrackedProcess{WorkerID: "w2"}
	tr.mu.Unlock()

	tr.Reset()
	if len(tr.List()) != 0 {
		t.Fatal("expected empty after reset")
	}
}

func TestUpdateTokens(t *testing.T) {
	tr := NewTracker("/tmp/test", nil)
	tr.mu.Lock()
	tr.processes["w1"] = &TrackedProcess{WorkerID: "w1"}
	tr.mu.Unlock()

	tr.UpdateTokens("w1", 500, 0.05)

	p, _ := tr.Get("w1")
	if p.TokenCount != 500 || p.CostUSD != 0.05 {
		t.Fatalf("token update failed: got %d / %f", p.TokenCount, p.CostUSD)
	}
}

// --- LogBuffer tests ---

func TestLogBufferAppendAndLines(t *testing.T) {
	buf := NewLogBuffer(5)
	for i := 0; i < 3; i++ {
		buf.Append(LogLine{Content: "line", Stream: "stdout", Timestamp: time.Now()})
	}
	if len(buf.Lines()) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(buf.Lines()))
	}
}

func TestLogBufferMaxLines(t *testing.T) {
	buf := NewLogBuffer(3)
	for i := 0; i < 5; i++ {
		buf.Append(LogLine{Content: string(rune('a' + i)), Stream: "stdout"})
	}
	lines := buf.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// Oldest should have been dropped; first remaining is 'c' (index 2).
	if lines[0].Content != "c" {
		t.Fatalf("expected 'c', got %q", lines[0].Content)
	}
}

func TestLogBufferRecent(t *testing.T) {
	buf := NewLogBuffer(10)
	for i := 0; i < 5; i++ {
		buf.Append(LogLine{Content: string(rune('a' + i)), Stream: "stdout"})
	}
	recent := buf.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(recent))
	}
	if recent[0].Content != "d" || recent[1].Content != "e" {
		t.Fatalf("unexpected recent: %v", recent)
	}
}

func TestLogBufferClear(t *testing.T) {
	buf := NewLogBuffer(10)
	buf.Append(LogLine{Content: "x"})
	buf.Clear()
	if len(buf.Lines()) != 0 {
		t.Fatal("expected empty after clear")
	}
}

func TestLogBufferDefaultMaxLines(t *testing.T) {
	buf := NewLogBuffer(0)
	if buf.maxLines != 200 {
		t.Fatalf("expected default 200, got %d", buf.maxLines)
	}
}
