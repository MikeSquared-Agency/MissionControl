package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAutoCommitConfig tests that auto_commit config is read correctly
func TestAutoCommitConfig(t *testing.T) {
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
	cfg := loadAutoCommitConfig(missionDir)

	if !cfg.Enabled {
		t.Error("Expected auto_commit.enabled to be true by default")
	}
	if !cfg.Task {
		t.Error("Expected auto_commit.task to be true by default")
	}
	if !cfg.Gate {
		t.Error("Expected auto_commit.gate to be true by default")
	}
	if !cfg.Stage {
		t.Error("Expected auto_commit.stage to be true by default")
	}
	if !cfg.Worker {
		t.Error("Expected auto_commit.worker to be true by default")
	}
	if !cfg.Handoff {
		t.Error("Expected auto_commit.handoff to be true by default")
	}
}

// TestAutoCommitDisabled tests that disabling auto_commit prevents commits
func TestAutoCommitDisabled(t *testing.T) {
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

	// Disable auto_commit
	configPath := filepath.Join(missionDir, "config.json")
	var config Config
	readJSON(configPath, &config)
	disabled := AutoCommitConfig{Enabled: false}
	config.AutoCommit = &disabled
	writeJSON(configPath, config)

	cfg := loadAutoCommitConfig(missionDir)
	if cfg.Enabled {
		t.Error("Expected auto_commit.enabled to be false after disabling")
	}
}

// TestAutoCommitSelectiveDisable tests disabling individual categories
func TestAutoCommitSelectiveDisable(t *testing.T) {
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

	// Disable only task commits
	configPath := filepath.Join(missionDir, "config.json")
	var config Config
	readJSON(configPath, &config)
	selective := AutoCommitConfig{
		Enabled:    true,
		Checkpoint: true,
		Task:       false, // disabled
		Gate:       true,
		Stage:      true,
		Worker:     true,
		Handoff:    true,
	}
	config.AutoCommit = &selective
	writeJSON(configPath, config)

	cfg := loadAutoCommitConfig(missionDir)
	if cfg.Task {
		t.Error("Expected auto_commit.task to be false")
	}
	if !cfg.Gate {
		t.Error("Expected auto_commit.gate to be true")
	}
}

// TestAutoCommitDefaultWhenMissing tests that missing config returns defaults
func TestAutoCommitDefaultWhenMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Config without auto_commit field
	missionDir := filepath.Join(tmpDir, ".mission")
	os.MkdirAll(missionDir, 0755)
	writeJSON(filepath.Join(missionDir, "config.json"), map[string]string{"version": "1.0.0"})

	cfg := loadAutoCommitConfig(missionDir)
	if !cfg.Enabled {
		t.Error("Expected default auto_commit.enabled to be true")
	}
}

// TestGitAutoCommitIntegration tests that state mutations create git commits
func TestGitAutoCommitIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Init git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tmpDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()

	// Init mission
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	// Initial commit
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	missionDir := filepath.Join(tmpDir, ".mission")

	// Count commits before
	beforeCount := countGitCommits(t, tmpDir)

	// Create a task (should trigger commit)
	tasksPath := filepath.Join(missionDir, "state", "tasks.json")
	var state TasksState
	readJSON(tasksPath, &state)
	state.Tasks = append(state.Tasks, Task{
		ID: "t1", Name: "Test", Stage: "discovery", Status: "pending",
	})
	writeJSON(tasksPath, state)
	gitAutoCommit(missionDir, CommitCategoryTask, taskCommitMsg("create", "t1", "Test"))

	afterCount := countGitCommits(t, tmpDir)
	if afterCount != beforeCount+1 {
		t.Errorf("Expected %d commits after task create, got %d", beforeCount+1, afterCount)
	}

	// Verify commit message
	lastMsg := lastGitCommitMsg(t, tmpDir)
	if !strings.Contains(lastMsg, "[mc:task]") {
		t.Errorf("Expected commit message to contain '[mc:task]', got: %s", lastMsg)
	}

	// Stage transition (should trigger commit)
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	writeJSON(stagePath, StageState{Current: "goal"})
	gitAutoCommit(missionDir, CommitCategoryStage, "advance discovery → goal")

	lastMsg = lastGitCommitMsg(t, tmpDir)
	if !strings.Contains(lastMsg, "[mc:stage]") {
		t.Errorf("Expected commit message to contain '[mc:stage]', got: %s", lastMsg)
	}
}

// TestGitAutoCommitSkipsWhenDisabled tests no commits when category disabled
func TestGitAutoCommitSkipsWhenDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Init git repo
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()

	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	missionDir := filepath.Join(tmpDir, ".mission")

	// Disable task commits
	configPath := filepath.Join(missionDir, "config.json")
	var config Config
	readJSON(configPath, &config)
	selective := AutoCommitConfig{
		Enabled:    true,
		Checkpoint: true,
		Task:       false,
		Gate:       true,
		Stage:      true,
		Worker:     true,
		Handoff:    true,
	}
	config.AutoCommit = &selective
	writeJSON(configPath, config)

	// Need to commit the config change first
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "disable task commits").Run()

	beforeCount := countGitCommits(t, tmpDir)

	// Write a task change
	tasksPath := filepath.Join(missionDir, "state", "tasks.json")
	var state TasksState
	readJSON(tasksPath, &state)
	state.Tasks = append(state.Tasks, Task{ID: "t1", Name: "Test", Status: "pending"})
	writeJSON(tasksPath, state)
	gitAutoCommit(missionDir, CommitCategoryTask, "create t1")

	afterCount := countGitCommits(t, tmpDir)
	if afterCount != beforeCount {
		t.Errorf("Expected no new commits when task commits disabled, got %d new", afterCount-beforeCount)
	}
}

// TestConfigJsonIncludesAutoCommit tests that mc init creates auto_commit in config
func TestConfigJsonIncludesAutoCommit(t *testing.T) {
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

	// Read raw config
	data, err := os.ReadFile(filepath.Join(tmpDir, ".mission", "config.json"))
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)
	if _, ok := raw["auto_commit"]; !ok {
		t.Error("Expected auto_commit field in config.json")
	}
}

func countGitCommits(t *testing.T, dir string) int {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		return 0
	}
	var count int
	_, _ = strings.NewReader(strings.TrimSpace(string(out))), nil
	count = 0
	for _, c := range strings.TrimSpace(string(out)) {
		count = count*10 + int(c-'0')
	}
	return count
}

func lastGitCommitMsg(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%s").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
