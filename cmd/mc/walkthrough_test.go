package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestFullWalkthrough_F6 walks through all 10 stages end-to-end:
// discovery → goal → requirements → planning → design → implement → verify → validate → document → release
//
// For each stage it:
//   - Creates a task scoped to that stage
//   - Completes the task
//   - Approves the gate
//   - Verifies the stage advanced exactly once
//
// Finally it asserts:
//   - All 10 gates have approval timestamps
//   - All tasks persist across stages (none lost)
//   - Final state is "release" completed (no further transition possible)
func TestFullWalkthrough_F6(t *testing.T) {
	// Setup isolated temp project
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Init project
	if err := runInit(nil, nil); err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	tasksPath := filepath.Join(missionDir, "state", "tasks.json")
	gatesPath := filepath.Join(missionDir, "state", "gates.json")

	// Verify initial stage
	assertCurrentStage(t, stagePath, "discovery")

	// Walk through every stage
	allStages := []string{"discovery", "goal", "requirements", "planning", "design", "implement", "verify", "validate", "document", "release"}

	for i, stage := range allStages {
		// --- Create a task for this stage ---
		taskID := stage + "-task"
		task := Task{
			ID:        taskID,
			Name:      "Work for " + stage,
			Stage:     stage,
			Zone:      "backend",
			Persona:   "developer",
			Status:    "pending",
			CreatedAt: "2026-02-08T00:00:00Z",
			UpdatedAt: "2026-02-08T00:00:00Z",
		}
		addTask(t, tasksPath, task)

		// --- Complete the task ---
		completeTask(t, tasksPath, taskID)

		// --- Approve the gate ---
		if err := runGateApprove(nil, []string{stage}); err != nil {
			t.Fatalf("gate approve failed for stage %q: %v", stage, err)
		}

		// --- Verify gate was approved with timestamp ---
		gate := readGate(t, gatesPath, stage)
		if gate.Status != "approved" {
			t.Errorf("stage %q: gate status = %q, want %q", stage, gate.Status, "approved")
		}
		if gate.ApprovedAt == "" {
			t.Errorf("stage %q: gate missing approval timestamp", stage)
		}

		// --- Verify stage advanced (or stayed at release for final) ---
		if i < len(allStages)-1 {
			expectedNext := allStages[i+1]
			assertCurrentStage(t, stagePath, expectedNext)
		}
	}

	// --- Final assertions ---

	// 1. Final stage should still be "release" (gate approve on release doesn't advance)
	assertCurrentStage(t, stagePath, "release")

	// 2. All tasks persist (10 total, one per stage)
	var ts TasksState
	readJSONFile(t, tasksPath, &ts)
	if len(ts.Tasks) != len(allStages) {
		t.Errorf("expected %d tasks to persist, got %d", len(allStages), len(ts.Tasks))
	}

	// 3. All tasks are complete
	for _, task := range ts.Tasks {
		if task.Status != "complete" {
			t.Errorf("task %q has status %q, want %q", task.ID, task.Status, "complete")
		}
	}

	// 4. All 10 gates approved with timestamps
	var gs GatesState
	readJSONFile(t, gatesPath, &gs)
	for _, stage := range allStages {
		g, ok := gs.Gates[stage]
		if !ok {
			t.Errorf("gate missing for stage %q", stage)
			continue
		}
		if g.Status != "approved" {
			t.Errorf("gate %q: status = %q, want approved", stage, g.Status)
		}
		if g.ApprovedAt == "" {
			t.Errorf("gate %q: missing approval timestamp", stage)
		}
	}

	// 5. Each stage advanced exactly once (we ended at release, started at discovery = 9 transitions)
	// Already verified inline above, but double-check we're at release
	assertCurrentStage(t, stagePath, "release")
}

// TestFullWalkthrough_NoDoubleAdvance ensures approving an already-approved gate
// does not advance the stage again (regression for PR #15 gate auto-advance bug).
func TestFullWalkthrough_NoDoubleAdvance(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")
	stagePath := filepath.Join(missionDir, "state", "stage.json")

	// Approve discovery gate → should move to goal
	if err := runGateApprove(nil, []string{"discovery"}); err != nil {
		t.Fatalf("first gate approve failed: %v", err)
	}
	assertCurrentStage(t, stagePath, "goal")

	// Approve discovery gate AGAIN → stage should still be goal, not requirements
	if err := runGateApprove(nil, []string{"discovery"}); err != nil {
		t.Fatalf("second gate approve failed: %v", err)
	}
	// The gate for "discovery" advances from "discovery" to "goal", but we're already at "goal"
	// so this would advance to "goal" again (which is current) — it depends on implementation.
	// The key point: we should NOT be at "requirements" from a double-approve of "discovery"
	var stage StageState
	readJSONFile(t, stagePath, &stage)
	if stage.Current == "requirements" {
		t.Error("double-approving discovery gate should NOT advance to requirements")
	}
}

// --- helpers ---

func assertCurrentStage(t *testing.T, stagePath, expected string) {
	t.Helper()
	var s StageState
	readJSONFile(t, stagePath, &s)
	if s.Current != expected {
		t.Fatalf("expected current stage %q, got %q", expected, s.Current)
	}
}

func readJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
}

func addTask(t *testing.T, tasksPath string, task Task) {
	t.Helper()
	var state TasksState
	readJSONFile(t, tasksPath, &state)
	state.Tasks = append(state.Tasks, task)
	data, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(tasksPath, data, 0644); err != nil {
		t.Fatalf("failed to write task: %v", err)
	}
}

func completeTask(t *testing.T, tasksPath, taskID string) {
	t.Helper()
	var state TasksState
	readJSONFile(t, tasksPath, &state)
	found := false
	for i := range state.Tasks {
		if state.Tasks[i].ID == taskID {
			state.Tasks[i].Status = "complete"
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("task %q not found", taskID)
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(tasksPath, data, 0644); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}
}

func readGate(t *testing.T, gatesPath, stage string) Gate {
	t.Helper()
	var gs GatesState
	readJSONFile(t, gatesPath, &gs)
	g, ok := gs.Gates[stage]
	if !ok {
		t.Fatalf("gate not found for stage %q", stage)
	}
	return g
}
