package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupMission creates a minimal .mission directory with stage, tasks, and gates.
func setupStageTestMission(t *testing.T, stage string, updatedAt time.Time, tasks []Task, gates *GatesFile) string {
	t.Helper()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, ".mission")
	stateDir := filepath.Join(missionDir, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write stage.json
	ss := StageState{Current: stage, UpdatedAt: updatedAt.UTC().Format(time.RFC3339)}
	writeJSONFile(t, filepath.Join(stateDir, "stage.json"), ss)

	// Write tasks.jsonl
	if tasks != nil {
		if err := writeTasksJSONL(filepath.Join(stateDir, "tasks.jsonl"), tasks); err != nil {
			t.Fatal(err)
		}
	}

	// Write gates.json — if nil, write an empty one with all criteria satisfied for the stage
	if gates != nil {
		writeJSONFile(t, filepath.Join(stateDir, "gates.json"), gates)
	} else {
		// Default: all criteria met so gate check passes
		gf := GatesFile{Gates: map[string]StageGate{
			stage: {Criteria: []GateCriterion{{Description: "auto", Satisfied: true}}},
		}}
		writeJSONFile(t, filepath.Join(stateDir, "gates.json"), gf)
	}

	// version.json for requireV6
	writeJSONFile(t, filepath.Join(missionDir, "version.json"), map[string]int{"version": 6})

	return missionDir
}

func writeJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- Tests for zero-task blocking ---

func TestStageAdvance_ZeroTaskBlock(t *testing.T) {
	// Advancing from "implement" with zero tasks should be blocked.
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-60*time.Second), nil, nil)

	err := advanceStageChecked(missionDir, "implement", false)
	if err == nil {
		t.Fatal("expected error when advancing from implement with zero tasks, got nil")
	}
	if got := err.Error(); !contains(got, "zero tasks") && !contains(got, "no tasks") {
		t.Errorf("expected error about zero/no tasks, got: %s", got)
	}
}

func TestStageAdvance_ZeroTaskBlock_WithForce(t *testing.T) {
	// Same scenario but with force — should succeed.
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-60*time.Second), nil, nil)

	err := advanceStageChecked(missionDir, "implement", true)
	if err != nil {
		t.Fatalf("expected force to bypass zero-task block, got: %v", err)
	}
}

func TestStageAdvance_ZeroTaskBlock_ExemptStages(t *testing.T) {
	// King-only stages (goal, requirements, planning, design) should NOT require tasks.
	exemptStages := []string{"goal", "requirements", "planning", "design"}
	for _, stage := range exemptStages {
		t.Run(stage, func(t *testing.T) {
			missionDir := setupStageTestMission(t, stage, time.Now().Add(-60*time.Second), nil, nil)

			err := advanceStageChecked(missionDir, stage, false)
			if err != nil {
				t.Fatalf("stage %q should be exempt from zero-task block, got: %v", stage, err)
			}
		})
	}
}

// --- Tests for velocity check ---

func TestStageAdvance_VelocityCheck(t *testing.T) {
	// Stage lasted <10s with zero completed tasks → should block.
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-5*time.Second), nil, nil)

	err := advanceStageChecked(missionDir, "implement", false)
	if err == nil {
		t.Fatal("expected velocity check to block advance (stage lasted <10s, zero completed tasks)")
	}
	got := err.Error()
	if !contains(got, "10s") && !contains(got, "velocity") && !contains(got, "too fast") {
		t.Errorf("expected velocity-related error, got: %s", got)
	}
}

func TestStageAdvance_VelocityCheck_OK(t *testing.T) {
	// Stage lasted 30s — velocity check should pass.
	// Still need at least one task for implement stage (or completed tasks).
	tasks := []Task{
		{ID: "t1", Name: "build it", Stage: "implement", Status: "complete", Persona: "developer"},
	}
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-30*time.Second), tasks, nil)

	err := advanceStageChecked(missionDir, "implement", false)
	if err != nil {
		t.Fatalf("expected velocity check to pass (30s elapsed), got: %v", err)
	}
}

// --- Tests for mandatory reviewer ---

func TestStageAdvance_MandatoryReviewer(t *testing.T) {
	// verify stage with tasks but none having persona "reviewer" → gate failure.
	tasks := []Task{
		{ID: "t1", Name: "test it", Stage: "verify", Status: "complete", Persona: "tester"},
	}
	missionDir := setupStageTestMission(t, "verify", time.Now().Add(-60*time.Second), tasks, nil)

	err := advanceStageChecked(missionDir, "verify", false)
	if err == nil {
		t.Fatal("expected error: verify stage requires at least one task with persona 'reviewer'")
	}
	got := err.Error()
	if !contains(got, "reviewer") {
		t.Errorf("expected error mentioning 'reviewer', got: %s", got)
	}
}

// --- Tests for mandatory discovery task ---

func TestStageAdvance_MandatoryDiscoveryTask(t *testing.T) {
	// discovery stage with zero tasks should block (discovery is NOT in the exempt list).
	missionDir := setupStageTestMission(t, "discovery", time.Now().Add(-60*time.Second), nil, nil)

	err := advanceStageChecked(missionDir, "discovery", false)
	if err == nil {
		t.Fatal("expected error: discovery stage with zero tasks should block")
	}
	got := err.Error()
	if !contains(got, "task") {
		t.Errorf("expected error about tasks, got: %s", got)
	}
}

// --- Tests for reviewer completion status ---

func TestStageAdvance_ReviewerNotDone(t *testing.T) {
	// verify stage with a reviewer task that isn't done → should block.
	tasks := []Task{
		{ID: "t1", Name: "code review", Stage: "verify", Status: "active", Persona: "reviewer"},
	}
	missionDir := setupStageTestMission(t, "verify", time.Now().Add(-60*time.Second), tasks, nil)

	err := advanceStageChecked(missionDir, "verify", false)
	if err == nil {
		t.Fatal("expected error: reviewer task not done should block verify")
	}
	if !contains(err.Error(), "reviewer") {
		t.Errorf("expected error about reviewer, got: %s", err.Error())
	}
}

func TestStageAdvance_ReviewerDone(t *testing.T) {
	// verify stage with a done reviewer → should pass.
	tasks := []Task{
		{ID: "t1", Name: "code review", Stage: "verify", Status: "done", Persona: "reviewer"},
	}
	missionDir := setupStageTestMission(t, "verify", time.Now().Add(-60*time.Second), tasks, nil)

	err := advanceStageChecked(missionDir, "verify", false)
	if err != nil {
		t.Fatalf("expected done reviewer to pass, got: %v", err)
	}
}

// --- Tests for wrong-stage task filtering ---

func TestStageAdvance_WrongStageTasks(t *testing.T) {
	// implement stage with tasks only in "verify" stage → should still block (zero tasks for implement).
	tasks := []Task{
		{ID: "t1", Name: "review", Stage: "verify", Status: "done", Persona: "reviewer"},
	}
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-60*time.Second), tasks, nil)

	err := advanceStageChecked(missionDir, "implement", false)
	if err == nil {
		t.Fatal("expected zero-task block for implement (tasks are in wrong stage)")
	}
	if !contains(err.Error(), "task") {
		t.Errorf("expected error about tasks, got: %s", err.Error())
	}
}

// --- Tests for velocity with completed tasks ---

func TestStageAdvance_VelocityBypassWithCompletedTasks(t *testing.T) {
	// Stage lasted <10s but has completed tasks → should pass velocity check.
	tasks := []Task{
		{ID: "t1", Name: "build it", Stage: "implement", Status: "done", Persona: "developer"},
	}
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-5*time.Second), tasks, nil)

	err := advanceStageChecked(missionDir, "implement", false)
	if err != nil {
		t.Fatalf("expected velocity bypass with completed tasks, got: %v", err)
	}
}

// --- Tests for force bypassing velocity ---

func TestStageAdvance_ForceBypassesVelocity(t *testing.T) {
	// Force should bypass velocity check even with zero completed tasks.
	missionDir := setupStageTestMission(t, "implement", time.Now().Add(-2*time.Second), nil, nil)

	err := advanceStageChecked(missionDir, "implement", true)
	if err != nil {
		t.Fatalf("expected force to bypass velocity check, got: %v", err)
	}
}

// contains is a simple substring check helper.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
