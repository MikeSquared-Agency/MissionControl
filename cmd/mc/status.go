package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current MissionControl status",
	Long:  `Displays the current stage, tasks, workers, and gate status.`,
	RunE:  runStatus,
}

type Status struct {
	Stage   StageState   `json:"stage"`
	Tasks   TasksState   `json:"tasks"`
	Workers WorkersState `json:"workers"`
	Gates   GatesState   `json:"gates"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	if err := requireV6(missionDir); err != nil {
		return err
	}

	status := Status{}

	// Read stage
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &status.Stage); err != nil {
		return fmt.Errorf("failed to read stage: %w", err)
	}

	// Read tasks
	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}
	status.Tasks = TasksState{Tasks: tasks}

	// Read workers
	if err := readJSON(filepath.Join(missionDir, "state", "workers.json"), &status.Workers); err != nil {
		return fmt.Errorf("failed to read workers: %w", err)
	}

	// Read gates
	if err := readJSON(filepath.Join(missionDir, "state", "gates.json"), &status.Gates); err != nil {
		return fmt.Errorf("failed to read gates: %w", err)
	}

	// Output as JSON
	output, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func findMissionDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up looking for .mission/
	dir := cwd
	for {
		missionDir := filepath.Join(dir, ".mission")
		if _, err := os.Stat(missionDir); err == nil {
			return missionDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf(".mission/ not found - run 'mc init' first")
}

func readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
