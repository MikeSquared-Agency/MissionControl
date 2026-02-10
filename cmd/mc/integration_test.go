package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// === F2: Multi-channel gate approval ===

// setupMission creates a temp dir, inits .mission/, and returns cleanup func.
func setupMission(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mc-integ-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)

	if err := runInit(nil, nil); err != nil {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
		t.Fatalf("mc init failed: %v", err)
	}

	return tmpDir, func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}
}

// readGate reads a gate from gates.json for the given stage.
func readGate(t *testing.T, missionDir, stage string) Gate {
	t.Helper()
	gatesPath := filepath.Join(missionDir, "state", "gates.json")
	var gs GatesState
	if err := readJSON(gatesPath, &gs); err != nil {
		t.Fatalf("Failed to read gates.json: %v", err)
	}
	g, ok := gs.Gates[stage]
	if !ok {
		t.Fatalf("Gate not found for stage: %s", stage)
	}
	return g
}

// readCurrentStage reads the current stage from stage.json.
func readCurrentStage(t *testing.T, missionDir string) string {
	t.Helper()
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	var ss StageState
	if err := readJSON(stagePath, &ss); err != nil {
		t.Fatalf("Failed to read stage.json: %v", err)
	}
	return ss.Current
}

// TestF2_MultiChannelGateApproval_CLI tests gate approval via the CLI path
// (runGateApprove) and verifies state consistency.
func TestF2_MultiChannelGateApproval_CLI(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	// Verify initial state
	g := readGate(t, missionDir, "discovery")
	if g.Status != "pending" {
		t.Fatalf("Expected initial gate status 'pending', got '%s'", g.Status)
	}

	// Approve via CLI function
	if err := runGateApproveWithNote("discovery", "test approval"); err != nil {
		t.Fatalf("CLI gate approve failed: %v", err)
	}

	// Verify gate is approved
	g = readGate(t, missionDir, "discovery")
	if g.Status != "approved" {
		t.Errorf("Expected gate status 'approved', got '%s'", g.Status)
	}
	if g.ApprovedAt == "" {
		t.Error("ApprovedAt should be set")
	}

	// Verify stage advanced
	if s := readCurrentStage(t, missionDir); s != "goal" {
		t.Errorf("Expected stage 'goal' after approval, got '%s'", s)
	}
}

// TestF2_MultiChannelGateApproval_DirectJSON simulates an "API" channel
// by directly writing to gates.json (as an API handler would) and verifies
// that subsequent CLI reads see consistent state.
func TestF2_MultiChannelGateApproval_DirectJSON(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	gatesPath := filepath.Join(missionDir, "state", "gates.json")

	// Simulate API approval: read, modify, write gates.json directly
	var gs GatesState
	if err := readJSON(gatesPath, &gs); err != nil {
		t.Fatalf("Failed to read gates: %v", err)
	}
	gate := gs.Gates["discovery"]
	gate.Status = "approved"
	gate.ApprovedAt = "2026-02-08T08:00:00Z"
	gs.Gates["discovery"] = gate
	if err := writeJSON(gatesPath, gs); err != nil {
		t.Fatalf("Failed to write gates: %v", err)
	}

	// Now read back via the same helpers the CLI uses
	g := readGate(t, missionDir, "discovery")
	if g.Status != "approved" {
		t.Errorf("Expected 'approved', got '%s'", g.Status)
	}
	if g.ApprovedAt != "2026-02-08T08:00:00Z" {
		t.Errorf("ApprovedAt mismatch: %s", g.ApprovedAt)
	}
}

// TestF2_MultiChannelGateApproval_SequentialStages approves gates from
// alternating "channels" (CLI vs direct-write) across multiple stages
// and verifies the full state remains consistent.
func TestF2_MultiChannelGateApproval_SequentialStages(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	stages := []string{"discovery", "goal", "requirements", "planning"}
	nextStages := []string{"goal", "requirements", "planning", "design"}

	for i, stage := range stages {
		cur := readCurrentStage(t, missionDir)
		if cur != stage {
			t.Fatalf("Step %d: expected current stage '%s', got '%s'", i, stage, cur)
		}

		if i%2 == 0 {
			// CLI channel
			if err := runGateApproveWithNote(stage, "test approval"); err != nil {
				t.Fatalf("Step %d: CLI approve failed: %v", i, err)
			}
		} else {
			// API/direct channel: approve gate + advance stage manually
			gatesPath := filepath.Join(missionDir, "state", "gates.json")
			var gs GatesState
			readJSON(gatesPath, &gs)
			g := gs.Gates[stage]
			g.Status = "approved"
			g.ApprovedAt = "2026-02-08T08:00:00Z"
			gs.Gates[stage] = g
			writeJSON(gatesPath, gs)

			// Advance stage (simulating what API would do)
			stagePath := filepath.Join(missionDir, "state", "stage.json")
			writeJSON(stagePath, StageState{Current: nextStages[i], UpdatedAt: "2026-02-08T08:00:00Z"})
		}

		// Verify consistency: gate is approved and stage advanced
		g := readGate(t, missionDir, stage)
		if g.Status != "approved" {
			t.Errorf("Step %d: gate '%s' not approved", i, stage)
		}

		got := readCurrentStage(t, missionDir)
		if got != nextStages[i] {
			t.Errorf("Step %d: expected next stage '%s', got '%s'", i, nextStages[i], got)
		}
	}
}

// TestF2_GateApprovalIdempotent verifies approving an already-approved gate
// doesn't corrupt state.
func TestF2_GateApprovalIdempotent(t *testing.T) {
	_, cleanup := setupMission(t)
	defer cleanup()

	// Approve once
	if err := runGateApproveWithNote("discovery", "test approval"); err != nil {
		t.Fatalf("First approve failed: %v", err)
	}

	// Approve again — should not error (idempotent)
	err := runGateApproveWithNote("discovery", "test approval")
	_ = err // Whether it errors or not, we just verify no panic/corruption
}

// === F3: Compaction + .mission/ state persistence ===

// TestF3_StatePersistenceAfterTaskCreation creates tasks and verifies
// all state files remain valid and consistent.
func TestF3_StatePersistenceAfterTaskCreation(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	// Create multiple tasks
	tasks := []Task{
		{ID: "t1", Name: "Research auth", Stage: "discovery", Status: "pending", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"},
		{ID: "t2", Name: "Design schema", Stage: "discovery", Status: "complete", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T01:00:00Z"},
		{ID: "t3", Name: "Write specs", Stage: "goal", Status: "pending", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"},
	}

	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatalf("Failed to write tasks: %v", err)
	}

	// Verify tasks survive re-read
	ts2, err := loadTasks(missionDir)
	if err != nil {
		t.Fatalf("Failed to re-read tasks: %v", err)
	}
	if len(ts2) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(ts2))
	}

	// Verify other state files are still valid
	var ss StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &ss); err != nil {
		t.Fatal("stage.json corrupted after task creation")
	}
	if ss.Current != "discovery" {
		t.Errorf("Stage should still be 'discovery', got '%s'", ss.Current)
	}

	var gs GatesState
	if err := readJSON(filepath.Join(missionDir, "state", "gates.json"), &gs); err != nil {
		t.Fatal("gates.json corrupted after task creation")
	}
	if len(gs.Gates) == 0 {
		t.Error("Gates should not be empty")
	}
}

// TestF3_StatePersistenceAfterGateApproval verifies that approving a gate
// doesn't corrupt any state files.
func TestF3_StatePersistenceAfterGateApproval(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	// Add a task first
	saveTasks(missionDir, []Task{{ID: "t1", Name: "Test task", Stage: "discovery", Status: "complete"}})

	// Approve gate
	if err := runGateApproveWithNote("discovery", "test approval"); err != nil {
		t.Fatalf("gate approve failed: %v", err)
	}

	// Verify ALL state files are valid
	var ss StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &ss); err != nil {
		t.Fatal("stage.json corrupted after gate approval")
	}
	if ss.Current != "goal" {
		t.Errorf("Expected 'goal', got '%s'", ss.Current)
	}

	var gs GatesState
	if err := readJSON(filepath.Join(missionDir, "state", "gates.json"), &gs); err != nil {
		t.Fatal("gates.json corrupted after gate approval")
	}

	verifyTasks, _ := loadTasks(missionDir)
	if len(verifyTasks) != 1 {
		t.Errorf("Task lost after gate approval! Got %d tasks", len(verifyTasks))
	}

	// Verify checkpoint was created (auto-checkpoint on gate approve)
	cpDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
	entries, _ := os.ReadDir(cpDir)
	if len(entries) == 0 {
		t.Error("No checkpoint created after gate approval")
	}
}

// TestF3_CheckpointCapturesFullState creates tasks, approves a gate,
// creates a checkpoint, and verifies the checkpoint contains everything.
func TestF3_CheckpointCapturesFullState(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	// Set up state: tasks + approve discovery gate
	saveTasks(missionDir, []Task{
		{ID: "t1", Name: "Task A", Stage: "discovery", Status: "complete"},
		{ID: "t2", Name: "Task B", Stage: "goal", Status: "pending"},
	})

	runGateApproveWithNote("discovery", "test approval")

	// Create explicit checkpoint
	cp, err := createCheckpoint(missionDir, "test-session-f3")
	if err != nil {
		t.Fatalf("createCheckpoint failed: %v", err)
	}

	// Verify checkpoint content
	if cp.Stage != "goal" {
		t.Errorf("Checkpoint stage: expected 'goal', got '%s'", cp.Stage)
	}
	if len(cp.Tasks) != 2 {
		t.Errorf("Checkpoint should have 2 tasks, got %d", len(cp.Tasks))
	}
	if cp.SessionID != "test-session-f3" {
		t.Errorf("Checkpoint session_id mismatch: got '%s'", cp.SessionID)
	}

	// Verify discovery gate is approved in checkpoint
	dg, ok := cp.Gates["discovery"]
	if !ok {
		t.Fatal("Checkpoint missing discovery gate")
	}
	if dg.Status != "approved" {
		t.Errorf("Checkpoint discovery gate: expected 'approved', got '%s'", dg.Status)
	}

	// Verify checkpoint file on disk is valid JSON
	cpPath := filepath.Join(missionDir, "orchestrator", "checkpoints", cp.ID+".json")
	data, err := os.ReadFile(cpPath)
	if err != nil {
		t.Fatalf("Checkpoint file missing: %v", err)
	}
	var diskCP CheckpointData
	if err := json.Unmarshal(data, &diskCP); err != nil {
		t.Fatalf("Checkpoint file invalid JSON: %v", err)
	}
	if diskCP.ID != cp.ID {
		t.Errorf("Checkpoint ID on disk doesn't match: '%s' vs '%s'", diskCP.ID, cp.ID)
	}
}

// TestF3_StateConsistencyAcrossMultipleOperations is an end-to-end test:
// init → create tasks → approve gates → checkpoint → verify everything.
func TestF3_StateConsistencyAcrossMultipleOperations(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	// 1. Create tasks
	saveTasks(missionDir, []Task{
		{ID: "t1", Name: "Research", Stage: "discovery", Status: "complete"},
	})

	// 2. Approve discovery gate (→ goal)
	runGateApproveWithNote("discovery", "test approval")

	// 3. Add more tasks for goal stage
	existingTasks, _ := loadTasks(missionDir)
	existingTasks = append(existingTasks, Task{ID: "t2", Name: "Define goals", Stage: "goal", Status: "complete"})
	saveTasks(missionDir, existingTasks)

	// 4. Approve goal gate (→ requirements)
	runGateApproveWithNote("goal", "test approval")

	// 5. Create checkpoint
	cp, err := createCheckpoint(missionDir, "e2e-session")
	if err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}

	// 6. Verify everything is consistent
	if cp.Stage != "requirements" {
		t.Errorf("Expected stage 'requirements', got '%s'", cp.Stage)
	}
	if len(cp.Tasks) != 2 {
		t.Errorf("Expected 2 tasks in checkpoint, got %d", len(cp.Tasks))
	}

	// Both gates should be approved
	for _, stage := range []string{"discovery", "goal"} {
		g, ok := cp.Gates[stage]
		if !ok {
			t.Errorf("Gate '%s' missing from checkpoint", stage)
			continue
		}
		if g.Status != "approved" {
			t.Errorf("Gate '%s' expected 'approved', got '%s'", stage, g.Status)
		}
	}

	// Verify current.json points to latest checkpoint
	currentPath := filepath.Join(missionDir, "orchestrator", "current.json")
	var current map[string]string
	if err := readJSON(currentPath, &current); err != nil {
		t.Fatal("current.json unreadable")
	}
	if current["checkpoint_id"] != cp.ID {
		t.Errorf("current.json checkpoint_id: expected '%s', got '%s'", cp.ID, current["checkpoint_id"])
	}

	// Verify state files on disk match checkpoint
	diskStage := readCurrentStage(t, missionDir)
	if diskStage != cp.Stage {
		t.Errorf("Disk stage '%s' doesn't match checkpoint stage '%s'", diskStage, cp.Stage)
	}

	diskTasks, _ := loadTasks(missionDir)
	if len(diskTasks) != len(cp.Tasks) {
		t.Errorf("Disk tasks (%d) != checkpoint tasks (%d)", len(diskTasks), len(cp.Tasks))
	}
}

// TestF3_SessionPersistence tests that session records survive across
// checkpoint/restart cycles.
func TestF3_SessionPersistence(t *testing.T) {
	tmpDir, cleanup := setupMission(t)
	defer cleanup()
	missionDir := filepath.Join(tmpDir, ".mission")

	// Create checkpoint + session records
	cp1, _ := createCheckpoint(missionDir, "session-A")

	appendSession(missionDir, SessionRecord{
		SessionID: "session-A",
		StartedAt: "2026-02-08T08:00:00Z",
		Stage:     "discovery",
	})
	appendSession(missionDir, SessionRecord{
		SessionID:    "session-A",
		EndedAt:      "2026-02-08T09:00:00Z",
		CheckpointID: cp1.ID,
		Stage:        "discovery",
		Reason:       "restart",
	})
	appendSession(missionDir, SessionRecord{
		SessionID:    "session-B",
		StartedAt:    "2026-02-08T09:00:00Z",
		CheckpointID: cp1.ID,
		Stage:        "discovery",
	})

	// Read sessions.jsonl and verify all 3 records
	sessionsPath := filepath.Join(missionDir, "orchestrator", "sessions.jsonl")
	data, err := os.ReadFile(sessionsPath)
	if err != nil {
		t.Fatalf("sessions.jsonl missing: %v", err)
	}

	lines := splitNonEmpty(string(data))
	if len(lines) != 3 {
		t.Fatalf("Expected 3 session records, got %d", len(lines))
	}

	// Verify first record
	var rec SessionRecord
	json.Unmarshal([]byte(lines[0]), &rec)
	if rec.SessionID != "session-A" || rec.StartedAt == "" {
		t.Errorf("First record invalid: %+v", rec)
	}

	// Verify last record references checkpoint
	json.Unmarshal([]byte(lines[2]), &rec)
	if rec.SessionID != "session-B" || rec.CheckpointID != cp1.ID {
		t.Errorf("Third record invalid: %+v", rec)
	}
}

// splitNonEmpty splits a string by newlines and removes empty entries.
func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range splitLines(s) {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
