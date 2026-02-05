package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stageCmd)
}

var stageCmd = &cobra.Command{
	Use:   "stage [next]",
	Short: "Get or set the current stage",
	Long: `Get the current stage, or use 'mc stage next' to transition.

Examples:
  mc stage         # Show current stage
  mc stage next    # Transition to next stage`,
	RunE: runStage,
}

var stages = []string{"discovery", "goal", "requirements", "planning", "design", "implement", "verify", "validate", "document", "release"}

func runStage(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	stagePath := filepath.Join(missionDir, "state", "stage.json")

	if len(args) == 0 {
		// Just show current stage
		var state StageState
		if err := readJSON(stagePath, &state); err != nil {
			return fmt.Errorf("failed to read stage: %w", err)
		}
		fmt.Println(state.Current)
		return nil
	}

	if args[0] == "next" {
		// Transition to next stage
		var state StageState
		if err := readJSON(stagePath, &state); err != nil {
			return fmt.Errorf("failed to read stage: %w", err)
		}

		nextStage, err := getNextStage(state.Current)
		if err != nil {
			return err
		}

		state.Current = nextStage
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		if err := writeJSON(stagePath, state); err != nil {
			return fmt.Errorf("failed to write stage: %w", err)
		}

		fmt.Printf("Stage transitioned: %s → %s\n", getPrevStage(nextStage), nextStage)
		return nil
	}

	// Set specific stage
	targetStage := args[0]
	if !isValidStage(targetStage) {
		return fmt.Errorf("invalid stage: %s (valid: %v)", targetStage, stages)
	}

	state := StageState{
		Current:   targetStage,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := writeJSON(stagePath, state); err != nil {
		return fmt.Errorf("failed to write stage: %w", err)
	}

	fmt.Printf("Stage set to: %s\n", targetStage)
	return nil
}

func getNextStage(current string) (string, error) {
	for i, stage := range stages {
		if stage == current {
			if i == len(stages)-1 {
				return "", fmt.Errorf("already at final stage: %s", current)
			}
			return stages[i+1], nil
		}
	}
	return "", fmt.Errorf("unknown stage: %s", current)
}

func getPrevStage(current string) string {
	for i, stage := range stages {
		if stage == current && i > 0 {
			return stages[i-1]
		}
	}
	return ""
}

func isValidStage(stage string) bool {
	for _, s := range stages {
		if s == stage {
			return true
		}
	}
	return false
}

func writeJSONAtomic(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
