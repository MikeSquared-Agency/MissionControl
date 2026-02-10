package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func validateCommit(missionDir string) error {
	// Check .mission/ exists
	if _, err := os.Stat(missionDir); os.IsNotExist(err) {
		return fmt.Errorf("no mission directory found at %s", missionDir)
	}

	// Read current stage
	var stageState StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err != nil {
		return fmt.Errorf("failed to read stage: %w", err)
	}

	// Load tasks and filter to current stage
	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	var stageTasks []Task
	for _, t := range tasks {
		if t.Stage == stageState.Current {
			stageTasks = append(stageTasks, t)
		}
	}

	if len(stageTasks) == 0 {
		return fmt.Errorf("no task entries found for stage %q", stageState.Current)
	}

	// Check findings for done tasks
	for _, t := range stageTasks {
		if t.Status != "done" {
			continue
		}
		findingsPath := filepath.Join(missionDir, "findings", t.ID+".md")
		info, err := os.Stat(findingsPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("findings file missing for done task %s: %s", t.ID, findingsPath)
		}
		if err != nil {
			return fmt.Errorf("error checking findings for task %s: %w", t.ID, err)
		}
		if info.Size() < 200 {
			return fmt.Errorf("findings for task %s is only %d bytes (must be >=200 bytes)", t.ID, info.Size())
		}
	}

	return nil
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Validate mission state and commit changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		missionDir, err := findMissionDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
			os.Exit(2)
			return nil
		}

		validateOnly, _ := cmd.Flags().GetBool("validate-only")
		strict, _ := cmd.Flags().GetBool("strict")

		if strict && !validateOnly {
			fmt.Fprintf(os.Stderr, "FAIL: --strict requires --validate-only\n")
			os.Exit(2)
			return nil
		}

		if err := validateCommit(missionDir); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
			os.Exit(1)
			return nil
		}

		if strict {
			if errs := validateStrict(missionDir); len(errs) > 0 {
				for _, e := range errs {
					fmt.Fprintf(os.Stderr, "FAIL: %s\n", e)
				}
				os.Exit(1)
				return nil
			}
		}

		if validateOnly {
			fmt.Println("✅ Validation passed")
			return nil
		}

		message, _ := cmd.Flags().GetString("message")
		if message == "" {
			return fmt.Errorf("commit message required (use -m)")
		}

		projectDir := filepath.Dir(missionDir)

		gitAdd := exec.Command("git", "add", "-A")
		gitAdd.Dir = projectDir
		gitAdd.Stdout = os.Stdout
		gitAdd.Stderr = os.Stderr
		if err := gitAdd.Run(); err != nil {
			return fmt.Errorf("git add failed: %w", err)
		}

		gitCommit := exec.Command("git", "commit", "-m", message)
		gitCommit.Dir = projectDir
		gitCommit.Stdout = os.Stdout
		gitCommit.Stderr = os.Stderr
		if err := gitCommit.Run(); err != nil {
			return fmt.Errorf("git commit failed: %w", err)
		}

		fmt.Println("✅ Committed successfully")
		return nil
	},
}

func init() {
	commitCmd.Flags().StringP("message", "m", "", "Commit message")
	commitCmd.Flags().Bool("validate-only", false, "Only run validation, don't commit")
	commitCmd.Flags().Bool("strict", false, "Enable strict validation (requires --validate-only)")
	rootCmd.AddCommand(commitCmd)
}
