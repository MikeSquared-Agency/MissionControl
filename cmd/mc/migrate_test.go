package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMigrateV5ToV6 tests that a v5-style project is correctly migrated to v6.
func TestMigrateV5ToV6(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-migrate-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// --- Set up a v5-style .mission/ directory ---
	missionDir := filepath.Join(tmpDir, ".mission")
	stateDir := filepath.Join(missionDir, "state")
	os.MkdirAll(stateDir, 0755)
	os.MkdirAll(filepath.Join(missionDir, "specs"), 0755)
	os.MkdirAll(filepath.Join(missionDir, "findings"), 0755)
	os.MkdirAll(filepath.Join(missionDir, "handoffs"), 0755)

	// v5 used phase.json instead of stage.json
	v5Phase := map[string]string{
		"current":    "implement",
		"updated_at": "2025-01-15T10:00:00Z",
	}
	writeJSON(filepath.Join(stateDir, "phase.json"), v5Phase)

	// v5 tasks used "phase" field instead of "stage"
	v5Tasks := map[string]interface{}{
		"tasks": []map[string]interface{}{
			{
				"id":         "task-alpha",
				"name":       "Build the widget",
				"phase":      "implement",
				"zone":       "core",
				"persona":    "developer",
				"status":     "in_progress",
				"worker_id":  "w-1",
				"created_at": "2025-01-10T08:00:00Z",
				"updated_at": "2025-01-14T12:00:00Z",
			},
			{
				"id":         "task-beta",
				"name":       "Research API options",
				"phase":      "idea",
				"zone":       "research",
				"persona":    "researcher",
				"status":     "complete",
				"created_at": "2025-01-09T08:00:00Z",
				"updated_at": "2025-01-11T16:00:00Z",
			},
		},
	}
	writeJSON(filepath.Join(stateDir, "tasks.json"), v5Tasks)

	// v5 gates (old format, will be regenerated)
	v5Gates := map[string]interface{}{
		"gates": map[string]interface{}{
			"idea":      map[string]interface{}{"stage": "idea", "status": "approved"},
			"design":    map[string]interface{}{"stage": "design", "status": "approved"},
			"implement": map[string]interface{}{"stage": "implement", "status": "pending"},
		},
	}
	writeJSON(filepath.Join(stateDir, "gates.json"), v5Gates)

	// --- Run migration ---
	err = runMigrate(nil, nil)
	if err != nil {
		t.Fatalf("runMigrate failed: %v", err)
	}

	// --- Verify v6 state ---

	// 1. phase.json should be removed
	if _, err := os.Stat(filepath.Join(stateDir, "phase.json")); !os.IsNotExist(err) {
		t.Error("phase.json should be removed after migration")
	}

	// 2. stage.json should exist with mapped stage
	var stage StageState
	if err := readJSON(filepath.Join(stateDir, "stage.json"), &stage); err != nil {
		t.Fatalf("Failed to read stage.json: %v", err)
	}
	if stage.Current != "implement" {
		t.Errorf("Expected stage 'implement' (mapped from phase 'implement'), got '%s'", stage.Current)
	}
	if stage.UpdatedAt == "" {
		t.Error("stage.json updated_at should be set")
	}

	// 3. gates.json should have all 10 v6 stages
	var gates GatesState
	if err := readJSON(filepath.Join(stateDir, "gates.json"), &gates); err != nil {
		t.Fatalf("Failed to read gates.json: %v", err)
	}
	expectedGates := []string{
		"discovery", "goal", "requirements", "planning", "design",
		"implement", "verify", "validate", "document", "release",
	}
	for _, g := range expectedGates {
		if _, ok := gates.Gates[g]; !ok {
			t.Errorf("Missing gate for stage '%s'", g)
		}
	}
	if len(gates.Gates) != 10 {
		t.Errorf("Expected 10 gates, got %d", len(gates.Gates))
	}

	// 4. tasks.json should have "stage" field (not "phase")
	tasksData, err := os.ReadFile(filepath.Join(stateDir, "tasks.json"))
	if err != nil {
		t.Fatalf("Failed to read tasks.json: %v", err)
	}
	var rawTasks map[string]interface{}
	json.Unmarshal(tasksData, &rawTasks)
	tasks := rawTasks["tasks"].([]interface{})

	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks preserved, got %d", len(tasks))
	}

	for _, raw := range tasks {
		task := raw.(map[string]interface{})
		if _, hasPhase := task["phase"]; hasPhase {
			t.Errorf("Task '%s' still has 'phase' field", task["id"])
		}
		if _, hasStage := task["stage"]; !hasStage {
			t.Errorf("Task '%s' missing 'stage' field", task["id"])
		}
	}

	// Check phase→stage mapping on tasks
	task0 := tasks[0].(map[string]interface{})
	if task0["stage"] != "implement" {
		t.Errorf("Task 'task-alpha' stage: expected 'implement', got '%s'", task0["stage"])
	}
	task1 := tasks[1].(map[string]interface{})
	if task1["stage"] != "discovery" {
		t.Errorf("Task 'task-beta' stage: expected 'discovery' (mapped from 'idea'), got '%s'", task1["stage"])
	}

	// Check other task fields are preserved
	if task0["name"] != "Build the widget" {
		t.Errorf("Task name not preserved: got '%s'", task0["name"])
	}
	if task0["status"] != "in_progress" {
		t.Errorf("Task status not preserved: got '%s'", task0["status"])
	}
	if task0["worker_id"] != "w-1" {
		t.Errorf("Task worker_id not preserved: got '%s'", task0["worker_id"])
	}

	// 5. orchestrator directory should be created
	orchDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
	if _, err := os.Stat(orchDir); os.IsNotExist(err) {
		t.Error("orchestrator/checkpoints directory should be created")
	}
}

// TestMigrateAlreadyV6 tests that migrating an already-v6 project fails gracefully.
func TestMigrateAlreadyV6(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-migrate-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Init creates a v6 project
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Migrate should fail
	err = runMigrate(nil, nil)
	if err == nil {
		t.Error("Expected error migrating already-v6 project")
	}
}

// TestMigrateNoV5 tests that migrating a non-v5 project fails gracefully.
func TestMigrateNoV5(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-migrate-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Create .mission/state but no phase.json
	os.MkdirAll(filepath.Join(tmpDir, ".mission", "state"), 0755)

	err = runMigrate(nil, nil)
	if err == nil {
		t.Error("Expected error migrating non-v5 project")
	}
}

// TestMigratePhaseMapping tests all v5 phase → v6 stage mappings.
func TestMigratePhaseMapping(t *testing.T) {
	phases := map[string]string{
		"idea":      "discovery",
		"design":    "design",
		"implement": "implement",
		"verify":    "verify",
		"document":  "document",
		"release":   "release",
	}

	for phase, expectedStage := range phases {
		t.Run(phase+"→"+expectedStage, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "mc-migrate-map-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			originalDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(originalDir)

			stateDir := filepath.Join(tmpDir, ".mission", "state")
			os.MkdirAll(stateDir, 0755)

			writeJSON(filepath.Join(stateDir, "phase.json"), map[string]string{
				"current":    phase,
				"updated_at": "2025-01-01T00:00:00Z",
			})

			err = runMigrate(nil, nil)
			if err != nil {
				t.Fatalf("runMigrate failed for phase '%s': %v", phase, err)
			}

			var stage StageState
			readJSON(filepath.Join(stateDir, "stage.json"), &stage)
			if stage.Current != expectedStage {
				t.Errorf("Phase '%s' → expected '%s', got '%s'", phase, expectedStage, stage.Current)
			}
		})
	}
}

// TestMigrateUnknownPhaseDefaultsToDiscovery tests that unknown phases map to discovery.
func TestMigrateUnknownPhaseDefaultsToDiscovery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-migrate-unknown-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	stateDir := filepath.Join(tmpDir, ".mission", "state")
	os.MkdirAll(stateDir, 0755)

	writeJSON(filepath.Join(stateDir, "phase.json"), map[string]string{
		"current":    "nonexistent-phase",
		"updated_at": "2025-01-01T00:00:00Z",
	})

	err = runMigrate(nil, nil)
	if err != nil {
		t.Fatalf("runMigrate failed: %v", err)
	}

	var stage StageState
	readJSON(filepath.Join(stateDir, "stage.json"), &stage)
	if stage.Current != "discovery" {
		t.Errorf("Unknown phase should map to 'discovery', got '%s'", stage.Current)
	}
}
