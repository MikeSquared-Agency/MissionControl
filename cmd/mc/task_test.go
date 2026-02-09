package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func setupTaskTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mc-task-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)

	// Initialize mission
	if err := runInit(nil, nil); err != nil {
		t.Fatalf("mc init failed: %v", err)
	}

	cleanup := func() {
		_ = os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}
	return tmpDir, cleanup
}

func setStage(t *testing.T, missionDir, stage string) {
	t.Helper()
	state := StageState{Current: stage}
	if err := writeJSON(filepath.Join(missionDir, ".mission", "state", "stage.json"), state); err != nil {
		t.Fatalf("Failed to write stage: %v", err)
	}
}

func TestStageIndex(t *testing.T) {
	if idx := stageIndex("discovery"); idx != 0 {
		t.Errorf("stageIndex(discovery) = %d, want 0", idx)
	}
	if idx := stageIndex("release"); idx != 9 {
		t.Errorf("stageIndex(release) = %d, want 9", idx)
	}
	if idx := stageIndex("bogus"); idx != -1 {
		t.Errorf("stageIndex(bogus) = %d, want -1", idx)
	}
}

func newTaskCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create <name>",
		Args: cobra.ExactArgs(1),
		RunE: runTaskCreate,
	}
	cmd.Flags().StringP("stage", "s", "", "Stage for the task")
	cmd.Flags().StringP("zone", "z", "", "Zone for the task")
	cmd.Flags().String("persona", "", "Persona to assign")
	cmd.Flags().StringSlice("depends-on", nil, "Task IDs this task depends on")
	cmd.Flags().Bool("force", false, "Bypass stage validation")
	return cmd
}

func TestTaskCreateSameStage(t *testing.T) {
	missionDir, cleanup := setupTaskTestDir(t)
	defer cleanup()

	setStage(t, missionDir, "implement")

	cmd := newTaskCreateCmd()
	cmd.Flags().Set("stage", "implement")
	err := cmd.RunE(cmd, []string{"test-task-same"})
	if err != nil {
		t.Errorf("Expected success creating task at current stage, got: %v", err)
	}
}

func TestTaskCreatePastStage(t *testing.T) {
	missionDir, cleanup := setupTaskTestDir(t)
	defer cleanup()

	setStage(t, missionDir, "implement")

	cmd := newTaskCreateCmd()
	cmd.Flags().Set("stage", "discovery")
	err := cmd.RunE(cmd, []string{"test-task-past"})
	if err != nil {
		t.Errorf("Expected success creating task at past stage, got: %v", err)
	}
}

func TestTaskCreateFutureStageBlocked(t *testing.T) {
	missionDir, cleanup := setupTaskTestDir(t)
	defer cleanup()

	setStage(t, missionDir, "discovery")

	cmd := newTaskCreateCmd()
	cmd.Flags().Set("stage", "verify")
	err := cmd.RunE(cmd, []string{"test-task-future"})
	if err == nil {
		t.Error("Expected error creating task for future stage, got nil")
	}
}

func TestTaskCreateFutureStageForce(t *testing.T) {
	missionDir, cleanup := setupTaskTestDir(t)
	defer cleanup()

	setStage(t, missionDir, "discovery")

	cmd := newTaskCreateCmd()
	cmd.Flags().Set("stage", "verify")
	cmd.Flags().Set("force", "true")
	err := cmd.RunE(cmd, []string{"test-task-force"})
	if err != nil {
		t.Errorf("Expected success with --force, got: %v", err)
	}
}

func TestTaskCreateNoStageSet(t *testing.T) {
	tmpDir, cleanup := setupTaskTestDir(t)
	defer cleanup()

	// Remove stage.json to simulate no stage set
	os.Remove(filepath.Join(tmpDir, ".mission", "state", "stage.json"))

	cmd := newTaskCreateCmd()
	cmd.Flags().Set("stage", "verify")
	err := cmd.RunE(cmd, []string{"test-task-nostage"})
	if err != nil {
		t.Errorf("Expected success when no stage is set, got: %v", err)
	}
}
