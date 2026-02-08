package main

import (
	"encoding/json"
	"os"
	"os/exec"
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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = json.Unmarshal(data, &stage)

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

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

// setupTestMission creates a temp dir with mc init and git, returns (tmpDir, missionDir, cleanup)
func setupTestMission(t *testing.T) (string, string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)

	// Init git repo (needed for checkpoint git commits)
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	cmd.Run()

	if err := runInit(nil, nil); err != nil {
		os.RemoveAll(tmpDir)
		os.Chdir(originalDir)
		t.Fatalf("mc init failed: %v", err)
	}

	// Initial git commit so checkpoint commits work
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tmpDir
	cmd.Run()

	missionDir := filepath.Join(tmpDir, ".mission")
	cleanup := func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}
	return tmpDir, missionDir, cleanup
}

// TestF7_CheckpointRoundTrip creates a checkpoint, then restores it, verifying state matches.
func TestF7_CheckpointRoundTrip(t *testing.T) {
	_, missionDir, cleanup := setupTestMission(t)
	defer cleanup()

	// Set up some state: add a task and advance to "goal"
	tasks := []Task{
		{ID: "t1", Name: "Research", Stage: "discovery", Status: "complete", CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"},
		{ID: "t2", Name: "Define goals", Stage: "goal", Status: "pending", CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"},
	}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatalf("Failed to write tasks: %v", err)
	}

	// Advance stage to "goal"
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	if err := writeJSON(stagePath, StageState{Current: "goal", UpdatedAt: "2024-01-01T00:00:00Z"}); err != nil {
		t.Fatalf("Failed to write stage: %v", err)
	}

	// Create checkpoint
	cp, err := createCheckpoint(missionDir, "test-session")
	if err != nil {
		t.Fatalf("createCheckpoint failed: %v", err)
	}

	if cp.Stage != "goal" {
		t.Errorf("Checkpoint stage: got %q, want %q", cp.Stage, "goal")
	}
	if len(cp.Tasks) != 2 {
		t.Errorf("Checkpoint tasks count: got %d, want 2", len(cp.Tasks))
	}

	// Now change state (advance to requirements)
	if err := writeJSON(stagePath, StageState{Current: "requirements", UpdatedAt: "2024-01-02T00:00:00Z"}); err != nil {
		t.Fatalf("Failed to update stage: %v", err)
	}

	// Query the checkpoint and verify it preserved the original state
	cpPath := filepath.Join(missionDir, "orchestrator", "checkpoints", cp.ID+".json")
	var restored CheckpointData
	if err := readJSON(cpPath, &restored); err != nil {
		t.Fatalf("Failed to read checkpoint: %v", err)
	}

	if restored.Stage != "goal" {
		t.Errorf("Restored checkpoint stage: got %q, want %q", restored.Stage, "goal")
	}
	if restored.ID != cp.ID {
		t.Errorf("Restored checkpoint ID: got %q, want %q", restored.ID, cp.ID)
	}
	if len(restored.Tasks) != 2 {
		t.Errorf("Restored tasks count: got %d, want 2", len(restored.Tasks))
	}
	if restored.Tasks[0].ID != "t1" || restored.Tasks[1].ID != "t2" {
		t.Error("Restored task IDs don't match")
	}
	if restored.SessionID != "test-session" {
		t.Errorf("Restored session ID: got %q, want %q", restored.SessionID, "test-session")
	}

	// Verify current stage is still "requirements" (checkpoint didn't mutate live state)
	var liveStage StageState
	if err := readJSON(stagePath, &liveStage); err != nil {
		t.Fatalf("Failed to read live stage: %v", err)
	}
	if liveStage.Current != "requirements" {
		t.Errorf("Live stage should still be 'requirements', got %q", liveStage.Current)
	}
}

// TestF8_AutoCheckpointOnGateApproval approves a gate and verifies a checkpoint was auto-created.
func TestF8_AutoCheckpointOnGateApproval(t *testing.T) {
	tmpDir, missionDir, cleanup := setupTestMission(t)
	defer cleanup()
	_ = tmpDir

	// Verify starting stage is "discovery"
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	var stage StageState
	if err := readJSON(stagePath, &stage); err != nil {
		t.Fatalf("Failed to read stage: %v", err)
	}
	if stage.Current != "discovery" {
		t.Fatalf("Expected initial stage 'discovery', got %q", stage.Current)
	}

	// Mark all discovery tasks complete (so gate criteria can be met)
	tasksPath := filepath.Join(missionDir, "state", "tasks.json")
	tasksState := TasksState{
		Tasks: []Task{
			{ID: "t1", Name: "Discover", Stage: "discovery", Status: "complete", CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"},
		},
	}
	if err := writeJSON(tasksPath, tasksState); err != nil {
		t.Fatalf("Failed to write tasks: %v", err)
	}

	// Count checkpoints before approval
	checkpointsDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
	beforeEntries, _ := os.ReadDir(checkpointsDir)
	beforeCount := len(beforeEntries)

	// Approve the discovery gate
	err := runGateApprove(nil, []string{"discovery"})
	if err != nil {
		t.Fatalf("runGateApprove failed: %v", err)
	}

	// Verify checkpoint was auto-created
	afterEntries, _ := os.ReadDir(checkpointsDir)
	afterCount := len(afterEntries)
	if afterCount <= beforeCount {
		t.Error("No checkpoint was created on gate approval")
	}

	// Verify stage advanced to "goal" (exactly one stage)
	if err := readJSON(stagePath, &stage); err != nil {
		t.Fatalf("Failed to read stage after approval: %v", err)
	}
	if stage.Current != "goal" {
		t.Errorf("Expected stage 'goal' after approving discovery gate, got %q", stage.Current)
	}

	// Verify the gate is marked approved
	gatesPath := filepath.Join(missionDir, "state", "gates.json")
	var gatesState GatesState
	if err := readJSON(gatesPath, &gatesState); err != nil {
		t.Fatalf("Failed to read gates: %v", err)
	}
	gate := gatesState.Gates["discovery"]
	if gate.Status != "approved" {
		t.Errorf("Discovery gate status: got %q, want 'approved'", gate.Status)
	}
	if gate.ApprovedAt == "" {
		t.Error("Discovery gate ApprovedAt should be set")
	}

	// BUG REGRESSION: Verify re-approving the same gate fails (prevents double-advance)
	err = runGateApprove(nil, []string{"discovery"})
	if err == nil {
		t.Error("Re-approving an already-approved gate should fail")
	}

	// BUG REGRESSION: Verify approving a non-current stage fails
	err = runGateApprove(nil, []string{"requirements"})
	if err == nil {
		t.Error("Approving gate for non-current stage should fail")
	}

	// Verify stage is STILL "goal" (no auto-advance beyond one stage)
	if err := readJSON(stagePath, &stage); err != nil {
		t.Fatalf("Failed to re-read stage: %v", err)
	}
	if stage.Current != "goal" {
		t.Errorf("Stage should still be 'goal', got %q (auto-advance bug!)", stage.Current)
	}
}
