package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditWriteAndRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	_ = os.MkdirAll(missionDir, 0755)

	// Write some entries
	writeAuditLog(missionDir, AuditTaskCreated, "cli", map[string]interface{}{
		"task_id": "t1",
		"name":    "Test task",
	})
	writeAuditLog(missionDir, AuditGateApproved, "cli", map[string]interface{}{
		"stage": "discovery",
	})
	writeAuditLog(missionDir, AuditStageAdvanced, "cli", map[string]interface{}{
		"from_stage": "discovery",
		"to_stage":   "goal",
	})

	// Read them back
	entries, err := readAuditLog(missionDir)
	if err != nil {
		t.Fatalf("readAuditLog failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	if entries[0].Action != AuditTaskCreated {
		t.Errorf("Expected action %s, got %s", AuditTaskCreated, entries[0].Action)
	}
	if entries[0].Actor != "cli" {
		t.Errorf("Expected actor 'cli', got '%s'", entries[0].Actor)
	}
	if entries[0].Details["task_id"] != "t1" {
		t.Errorf("Expected task_id 't1', got '%v'", entries[0].Details["task_id"])
	}

	if entries[1].Action != AuditGateApproved {
		t.Errorf("Expected action %s, got %s", AuditGateApproved, entries[1].Action)
	}

	if entries[2].Action != AuditStageAdvanced {
		t.Errorf("Expected action %s, got %s", AuditStageAdvanced, entries[2].Action)
	}
}

func TestAuditLogFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	_ = os.MkdirAll(missionDir, 0755)

	writeAuditLog(missionDir, AuditWorkerSpawned, "cli", map[string]interface{}{
		"worker_id": "w1",
		"persona":   "developer",
	})

	// Verify it's valid JSONL
	data, err := os.ReadFile(filepath.Join(missionDir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("Failed to read audit.jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	var entry AuditEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("Invalid JSON line: %v", err)
	}

	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if entry.Action != AuditWorkerSpawned {
		t.Errorf("Expected action %s, got %s", AuditWorkerSpawned, entry.Action)
	}
}

func TestAuditReadEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	_ = os.MkdirAll(missionDir, 0755)

	// No audit.jsonl exists
	entries, err := readAuditLog(missionDir)
	if err != nil {
		t.Fatalf("readAuditLog failed: %v", err)
	}
	if entries != nil {
		t.Errorf("Expected nil entries for empty log, got %d", len(entries))
	}
}

func TestAuditInitCreatesEntry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")
	entries, err := readAuditLog(missionDir)
	if err != nil {
		t.Fatalf("readAuditLog failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 audit entry after init, got %d", len(entries))
	}

	if entries[0].Action != AuditProjectInitialized {
		t.Errorf("Expected action %s, got %s", AuditProjectInitialized, entries[0].Action)
	}
}

func TestAuditIntegrationWithStageAndGate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")

	// Create+complete task and approve gate for discovery (auto-advances to goal)
	addTask(t, missionDir, Task{ID: "d1", Name: "discover", Stage: "discovery", Status: "pending", Persona: "researcher", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"})
	completeTask(t, missionDir, "d1")
	if err := runGateApprove(nil, []string{"discovery"}); err != nil {
		t.Fatalf("gate approve for discovery failed: %v", err)
	}

	// Approve gate for goal (auto-advances to requirements)
	addTask(t, missionDir, Task{ID: "g1", Name: "goal", Stage: "goal", Status: "pending", Persona: "dev", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"})
	completeTask(t, missionDir, "g1")
	err = runGateApprove(nil, []string{"goal"})
	if err != nil {
		t.Fatalf("mc gate approve failed: %v", err)
	}

	entries, err := readAuditLog(missionDir)
	if err != nil {
		t.Fatalf("readAuditLog failed: %v", err)
	}

	// Expect: project_initialized, stage_advanced, gate_approved, checkpoint_created, stage_advanced
	actions := make([]string, len(entries))
	for i, e := range entries {
		actions[i] = e.Action
	}

	// Verify key actions are present
	found := map[string]bool{}
	for _, a := range actions {
		found[a] = true
	}

	for _, expected := range []string{AuditProjectInitialized, AuditStageAdvanced, AuditGateApproved, AuditCheckpointCreated} {
		if !found[expected] {
			t.Errorf("Expected action %s in audit trail, got: %v", expected, actions)
		}
	}
}
