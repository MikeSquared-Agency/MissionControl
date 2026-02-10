package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupCommitTestMission creates a .mission with stage, tasks, and findings for commit validation tests.
func setupCommitTestMission(t *testing.T, stage string, tasks []Task, findings map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, ".mission")
	stateDir := filepath.Join(missionDir, "state")
	findingsDir := filepath.Join(missionDir, "findings")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(findingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write stage.json
	ss := StageState{Current: stage, UpdatedAt: time.Now().Add(-30 * time.Second).UTC().Format(time.RFC3339)}
	writeJSONFile(t, filepath.Join(stateDir, "stage.json"), ss)

	// Write tasks.jsonl
	if tasks != nil {
		if err := writeTasksJSONL(filepath.Join(stateDir, "tasks.jsonl"), tasks); err != nil {
			t.Fatal(err)
		}
	}

	// Write version.json
	writeJSONFile(t, filepath.Join(missionDir, "version.json"), map[string]int{"version": 6})

	// Write findings files
	for id, content := range findings {
		p := filepath.Join(findingsDir, id+".md")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return missionDir
}

func TestValidateCommit_NoMission(t *testing.T) {
	// No .mission directory → should error
	err := validateCommit("/tmp/nonexistent-dir-abc123")
	if err == nil {
		t.Fatal("expected error when .mission doesn't exist")
	}
	if !strings.Contains(err.Error(), "mission") {
		t.Errorf("expected error about mission, got: %s", err.Error())
	}
}

func TestValidateCommit_NoTasks(t *testing.T) {
	// Stage with no tasks → should error
	missionDir := setupCommitTestMission(t, "implement", nil, nil)
	err := validateCommit(missionDir)
	if err == nil {
		t.Fatal("expected error when stage has no tasks")
	}
	if !strings.Contains(err.Error(), "task") {
		t.Errorf("expected error about tasks, got: %s", err.Error())
	}
}

func TestValidateCommit_NoFindings(t *testing.T) {
	// Done tasks but no findings files → should error
	tasks := []Task{
		{ID: "t1", Name: "build it", Stage: "implement", Status: "done", Persona: "developer"},
	}
	missionDir := setupCommitTestMission(t, "implement", tasks, nil)
	err := validateCommit(missionDir)
	if err == nil {
		t.Fatal("expected error when done tasks have no findings")
	}
	if !strings.Contains(err.Error(), "findings") {
		t.Errorf("expected error about findings, got: %s", err.Error())
	}
}

func TestValidateCommit_StubFindings(t *testing.T) {
	// Done tasks with tiny findings (<200 bytes) → should error
	tasks := []Task{
		{ID: "t1", Name: "build it", Stage: "implement", Status: "done", Persona: "developer"},
	}
	findings := map[string]string{"t1": "stub"}
	missionDir := setupCommitTestMission(t, "implement", tasks, findings)
	err := validateCommit(missionDir)
	if err == nil {
		t.Fatal("expected error when findings are stubs (<200 bytes)")
	}
	if !strings.Contains(err.Error(), "200") || !strings.Contains(err.Error(), "bytes") {
		t.Errorf("expected error about minimum size, got: %s", err.Error())
	}
}

func TestValidateCommit_ValidMission(t *testing.T) {
	// Proper tasks + substantial findings → should pass
	tasks := []Task{
		{ID: "t1", Name: "build it", Stage: "implement", Status: "done", Persona: "developer"},
	}
	findings := map[string]string{
		"t1": "Summary: Implemented the feature.\n\n## Changes\n" + strings.Repeat("Detailed description of work done. ", 10),
	}
	missionDir := setupCommitTestMission(t, "implement", tasks, findings)
	err := validateCommit(missionDir)
	if err != nil {
		t.Fatalf("expected valid mission to pass, got: %v", err)
	}
}

func TestValidateCommit_PendingTasksOK(t *testing.T) {
	// Pending tasks don't need findings — only done tasks do
	tasks := []Task{
		{ID: "t1", Name: "build it", Stage: "implement", Status: "done", Persona: "developer"},
		{ID: "t2", Name: "review it", Stage: "implement", Status: "pending", Persona: "reviewer"},
	}
	findings := map[string]string{
		"t1": "Summary: Done.\n\n## Changes\n" + strings.Repeat("Real findings content here. ", 10),
	}
	missionDir := setupCommitTestMission(t, "implement", tasks, findings)
	err := validateCommit(missionDir)
	if err != nil {
		t.Fatalf("expected pending tasks to not require findings, got: %v", err)
	}
}
