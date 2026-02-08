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
		".mission/state/tasks.jsonl",
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

	// Create task by directly writing to tasks.jsonl
	missionDir := filepath.Join(tmpDir, ".mission")

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

	tasks := []Task{task}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatalf("Failed to write task: %v", err)
	}

	// Verify task was created
	readTasks, err := loadTasks(missionDir)
	if err != nil {
		t.Fatalf("Failed to read tasks.jsonl: %v", err)
	}

	if len(readTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(readTasks))
	}

	if readTasks[0].Name != "Research authentication options" {
		t.Errorf("Task name mismatch: got '%s'", readTasks[0].Name)
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

// TestCheckpointCreate tests that mc checkpoint creates a file
func TestCheckpointCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")

	// Create a checkpoint
	cp, err := createCheckpoint(missionDir, "test-session")
	if err != nil {
		t.Fatalf("createCheckpoint failed: %v", err)
	}

	if cp.ID == "" {
		t.Error("Checkpoint ID should not be empty")
	}

	if cp.Stage != "discovery" {
		t.Errorf("Expected stage 'discovery', got '%s'", cp.Stage)
	}

	if cp.SessionID != "test-session" {
		t.Errorf("Expected session_id 'test-session', got '%s'", cp.SessionID)
	}

	// Verify checkpoint file was written
	cpPath := filepath.Join(missionDir, "orchestrator", "checkpoints", cp.ID+".json")
	if _, err := os.Stat(cpPath); os.IsNotExist(err) {
		t.Error("Checkpoint file not created")
	}

	// Verify current.json was updated
	currentPath := filepath.Join(missionDir, "orchestrator", "current.json")
	var current map[string]string
	if err := readJSON(currentPath, &current); err != nil {
		t.Fatalf("Failed to read current.json: %v", err)
	}

	if current["checkpoint_id"] != cp.ID {
		t.Errorf("current.json checkpoint_id mismatch: got '%s'", current["checkpoint_id"])
	}
}

// TestCheckpointIncludesTasks tests that checkpoint snapshots include tasks
func TestCheckpointIncludesTasks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")

	// Create a task first
	if err := saveTasks(missionDir, []Task{
		{ID: "task-1", Name: "Test task", Stage: "discovery", Status: "complete"},
	}); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	// Create checkpoint
	cp, err := createCheckpoint(missionDir, "")
	if err != nil {
		t.Fatalf("createCheckpoint failed: %v", err)
	}

	if len(cp.Tasks) != 1 {
		t.Fatalf("Expected 1 task in snapshot, got %d", len(cp.Tasks))
	}

	if cp.Tasks[0].Name != "Test task" {
		t.Errorf("Task name mismatch: got '%s'", cp.Tasks[0].Name)
	}
}

// TestGateApproveCreatesCheckpoint tests that gate approval auto-creates a checkpoint (G3.1)
func TestGateApproveCreatesCheckpoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")

	// Verify no checkpoints exist yet
	checkpointsDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
	entries, _ := os.ReadDir(checkpointsDir)
	if len(entries) != 0 {
		t.Fatalf("Expected 0 checkpoints before gate approve, got %d", len(entries))
	}

	// Approve gate
	err = runGateApprove(nil, []string{"discovery"})
	if err != nil {
		t.Fatalf("mc gate approve failed: %v", err)
	}

	// Verify checkpoint was auto-created
	entries, _ = os.ReadDir(checkpointsDir)
	if len(entries) != 1 {
		t.Errorf("Expected 1 checkpoint after gate approve, got %d", len(entries))
	}

	// Verify stage advanced
	stageFile := filepath.Join(missionDir, "state", "stage.json")
	data, _ := os.ReadFile(stageFile)
	var stage StageState
	json.Unmarshal(data, &stage)

	if stage.Current != "goal" {
		t.Errorf("Expected stage 'goal' after gate approve, got '%s'", stage.Current)
	}
}

// TestCheckpointRestart tests session restart with checkpoint
func TestCheckpointRestart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")

	// Create initial checkpoint
	cp1, err := createCheckpoint(missionDir, "session-1")
	if err != nil {
		t.Fatalf("createCheckpoint failed: %v", err)
	}

	// Log a session start
	appendSession(missionDir, SessionRecord{
		SessionID: "session-1",
		StartedAt: "2024-01-01T00:00:00Z",
	})

	// Log session end and new start
	appendSession(missionDir, SessionRecord{
		SessionID:    "session-1",
		EndedAt:      "2024-01-01T01:00:00Z",
		CheckpointID: cp1.ID,
		Reason:       "restart",
	})

	appendSession(missionDir, SessionRecord{
		SessionID:    "session-2",
		StartedAt:    "2024-01-01T01:00:00Z",
		CheckpointID: cp1.ID,
	})

	// Verify sessions.jsonl has entries
	sessionsPath := filepath.Join(missionDir, "orchestrator", "sessions.jsonl")
	data, err := os.ReadFile(sessionsPath)
	if err != nil {
		t.Fatalf("Failed to read sessions.jsonl: %v", err)
	}

	if len(data) == 0 {
		t.Error("sessions.jsonl should not be empty")
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
