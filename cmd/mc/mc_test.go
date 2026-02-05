package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMcInit tests that mc init creates a valid .mission/ directory
func TestMcInit(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Run init
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Verify directory structure
	requiredDirs := []string{
		".mission",
		".mission/state",
		".mission/specs",
		".mission/findings",
		".mission/handoffs",
		".mission/checkpoints",
		".mission/prompts",
		".mission/orchestrator",
		".mission/orchestrator/checkpoints",
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory not created: %s", dir)
		}
	}

	// Verify state files
	stateFiles := []string{
		".mission/state/stage.json",
		".mission/state/tasks.json",
		".mission/state/workers.json",
		".mission/state/gates.json",
	}

	for _, file := range stateFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("State file not created: %s", file)
		}
	}

	// Verify CLAUDE.md exists
	claudeMD := filepath.Join(tmpDir, ".mission", "CLAUDE.md")
	if _, err := os.Stat(claudeMD); os.IsNotExist(err) {
		t.Error("CLAUDE.md not created")
	}

	// Verify stage.json has valid content
	stageFile := filepath.Join(tmpDir, ".mission/state/stage.json")
	data, err := os.ReadFile(stageFile)
	if err != nil {
		t.Fatalf("Failed to read stage.json: %v", err)
	}

	var stage StageState
	if err := json.Unmarshal(data, &stage); err != nil {
		t.Fatalf("Invalid stage.json: %v", err)
	}

	if stage.Current != "discovery" {
		t.Errorf("Expected stage 'discovery', got '%s'", stage.Current)
	}
}

// TestTaskCreateDirect tests task creation using direct function call
func TestTaskCreateDirect(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Create task by directly writing to tasks.json
	missionDir := filepath.Join(tmpDir, ".mission")
	tasksPath := filepath.Join(missionDir, "state", "tasks.json")

	var state TasksState
	if err := readJSON(tasksPath, &state); err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	task := Task{
		ID:        "test-task-1",
		Name:      "Research authentication options",
		Stage:     "discovery",
		Zone:      "research",
		Persona:   "researcher",
		Status:    "pending",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	state.Tasks = append(state.Tasks, task)
	if err := writeJSON(tasksPath, state); err != nil {
		t.Fatalf("Failed to write task: %v", err)
	}

	// Verify task was created
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		t.Fatalf("Failed to read tasks.json: %v", err)
	}

	var readState TasksState
	if err := json.Unmarshal(data, &readState); err != nil {
		t.Fatalf("Invalid tasks.json: %v", err)
	}

	if len(readState.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(readState.Tasks))
	}

	if readState.Tasks[0].Name != "Research authentication options" {
		t.Errorf("Task name mismatch: got '%s'", readState.Tasks[0].Name)
	}
}

// TestStageTransition tests stage transitions
func TestStageTransition(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Test stage transition using runStage with "next" arg
	err = runStage(nil, []string{"next"})
	if err != nil {
		t.Fatalf("mc stage next failed: %v", err)
	}

	// Verify stage changed
	stageFile := filepath.Join(tmpDir, ".mission/state/stage.json")
	data, err := os.ReadFile(stageFile)
	if err != nil {
		t.Fatalf("Failed to read stage.json: %v", err)
	}

	var stage StageState
	if err := json.Unmarshal(data, &stage); err != nil {
		t.Fatalf("Invalid stage.json: %v", err)
	}

	if stage.Current != "goal" {
		t.Errorf("Expected stage 'goal', got '%s'", stage.Current)
	}
}

// TestHandoffValidation tests handoff file validation
func TestHandoffValidation(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Create a valid handoff file
	handoff := map[string]interface{}{
		"task_id":   "task-1",
		"worker_id": "worker-1",
		"status":    "complete",
		"findings": []map[string]string{
			{"type": "discovery", "summary": "Found existing auth implementation"},
		},
		"artifacts":      []string{},
		"open_questions": []string{},
	}

	handoffData, _ := json.Marshal(handoff)
	handoffFile := filepath.Join(tmpDir, "test-handoff.json")
	if err := os.WriteFile(handoffFile, handoffData, 0644); err != nil {
		t.Fatalf("Failed to write handoff file: %v", err)
	}

	// Run handoff command
	err = runHandoff(nil, []string{handoffFile})
	if err != nil {
		t.Fatalf("mc handoff failed: %v", err)
	}

	// Verify handoff was stored
	handoffsDir := filepath.Join(tmpDir, ".mission/handoffs")
	entries, err := os.ReadDir(handoffsDir)
	if err != nil {
		t.Fatalf("Failed to read handoffs dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 handoff file, got %d", len(entries))
	}
}

// TestGateCheck tests gate checking
func TestGateCheck(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Run gate check
	err = runGateCheck(nil, []string{"discovery"})
	if err != nil {
		t.Fatalf("mc gate check failed: %v", err)
	}
}

// TestPromptGeneration tests that all persona prompts are generated
func TestPromptGeneration(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Verify all persona prompts exist
	personas := []string{
		"researcher", "analyst", "requirements-engineer", "designer",
		"architect", "developer", "debugger", "reviewer", "security",
		"tester", "qa", "docs", "devops",
	}

	for _, persona := range personas {
		promptFile := filepath.Join(tmpDir, ".mission/prompts", persona+".md")
		if _, err := os.Stat(promptFile); os.IsNotExist(err) {
			t.Errorf("Prompt file not created: %s.md", persona)
		}
	}
}

// TestStageSequence tests the full stage sequence
func TestStageSequence(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Test full stage sequence
	expectedStages := []string{"goal", "requirements", "planning", "design", "implement", "verify", "validate", "document", "release"}

	for _, expected := range expectedStages {
		err = runStage(nil, []string{"next"})
		if err != nil {
			t.Fatalf("mc stage next failed: %v", err)
		}

		stageFile := filepath.Join(tmpDir, ".mission/state/stage.json")
		data, err := os.ReadFile(stageFile)
		if err != nil {
			t.Fatalf("Failed to read stage.json: %v", err)
		}

		var stage StageState
		if err := json.Unmarshal(data, &stage); err != nil {
			t.Fatalf("Invalid stage.json: %v", err)
		}

		if stage.Current != expected {
			t.Errorf("Expected stage '%s', got '%s'", expected, stage.Current)
		}
	}

	// Final stage should not transition
	err = runStage(nil, []string{"next"})
	if err == nil {
		t.Error("Expected error when transitioning from final stage")
	}
}

// TestHandoffValidationError tests that invalid handoffs are rejected
func TestHandoffValidationError(t *testing.T) {
	// Create temp directory with .mission
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initialize
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Create an invalid handoff file (missing status)
	handoff := map[string]interface{}{
		"task_id":   "task-1",
		"worker_id": "worker-1",
		// status is missing
		"findings":       []map[string]string{},
		"artifacts":      []string{},
		"open_questions": []string{},
	}

	handoffData, _ := json.Marshal(handoff)
	handoffFile := filepath.Join(tmpDir, "invalid-handoff.json")
	if err := os.WriteFile(handoffFile, handoffData, 0644); err != nil {
		t.Fatalf("Failed to write handoff file: %v", err)
	}

	// Run handoff command - should fail
	err = runHandoff(nil, []string{handoffFile})
	if err == nil {
		t.Error("Expected error for invalid handoff")
	}
}
