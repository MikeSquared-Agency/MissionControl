package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// findTaskByID loads all tasks and finds one matching the given ID or unambiguous prefix.
func findTaskByID(missionDir string, taskID string) (Task, error) {
	tasks, err := loadTasks(missionDir)
	if err != nil {
		return Task{}, fmt.Errorf("failed to load tasks: %w", err)
	}

	var matches []Task
	for _, t := range tasks {
		if t.ID == taskID {
			return t, nil // exact match
		}
		if strings.HasPrefix(t.ID, taskID) {
			matches = append(matches, t)
		}
	}

	switch len(matches) {
	case 0:
		return Task{}, fmt.Errorf("task not found: %s", taskID)
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		return Task{}, fmt.Errorf("ambiguous task prefix %q matches: %s", taskID, strings.Join(ids, ", "))
	}
}

func validateCommit(missionDir string) error {
	if _, err := os.Stat(missionDir); os.IsNotExist(err) {
		return fmt.Errorf("no mission directory found at %s", missionDir)
	}

	var stageState StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err != nil {
		return fmt.Errorf("failed to read stage: %w", err)
	}

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

func validateProvenance(missionDir string) []string {
	projectDir := filepath.Dir(missionDir)

	out, err := exec.Command("git", "-C", projectDir, "rev-list", "--no-merges", "main..HEAD").Output()
	if err != nil {
		return nil
	}

	shas := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(shas) == 0 || (len(shas) == 1 && shas[0] == "") {
		return nil
	}

	var errs []string
	for _, sha := range shas {
		msgOut, err := exec.Command("git", "-C", projectDir, "log", "-1", "--format=%B", sha).Output()
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to read commit %s: %v", sha[:10], err))
			continue
		}
		msg := string(msgOut)

		// A commit is valid if it has task trailers OR a no-task reason
		hasTaskTrailers := true
		for _, prefix := range []string{"MC-Task:", "MC-Persona:", "MC-Stage:"} {
			if !strings.Contains(msg, prefix) {
				hasTaskTrailers = false
				break
			}
		}
		hasNoTaskReason := strings.Contains(msg, "MC-NoTask-Reason:")

		if !hasTaskTrailers && !hasNoTaskReason {
			shortSha := sha
			if len(shortSha) > 10 {
				shortSha = shortSha[:10]
			}
			errs = append(errs, fmt.Sprintf("commit %s missing MC-Task trailers or MC-NoTask-Reason", shortSha))
		}
	}
	return errs
}

// matchesSelectiveScope returns true if file should be staged for a task.
func matchesSelectiveScope(file string, scopePaths, exemptPaths []string) bool {
	if strings.HasPrefix(file, ".mission/") {
		return true
	}
	for _, p := range scopePaths {
		if matchesScopePath(file, p) {
			return true
		}
	}
	for _, p := range exemptPaths {
		if matchesScopePath(file, p) {
			return true
		}
	}
	return false
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Validate mission state and commit changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		missionDir, err := findMissionDir()
		if err != nil {
			// If --validate-only and no .mission/, exit cleanly (nothing to validate)
			validateOnly, _ := cmd.Flags().GetBool("validate-only")
			if validateOnly {
				fmt.Fprintf(os.Stderr, "OK: no .mission/ directory found — nothing to validate\n")
				return nil
			}
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

		validateProv, _ := cmd.Flags().GetBool("validate-provenance")
		if validateProv && !validateOnly {
			fmt.Fprintf(os.Stderr, "FAIL: --validate-provenance requires --validate-only\n")
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

		if validateProv {
			if provErrs := validateProvenance(missionDir); len(provErrs) > 0 {
				for _, e := range provErrs {
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

		// --- Guard logic for mandatory task binding ---
		taskFlag, _ := cmd.Flags().GetString("task")
		noTask, _ := cmd.Flags().GetBool("no-task")
		reason, _ := cmd.Flags().GetString("reason")

		if taskFlag != "" && noTask {
			fmt.Fprintf(os.Stderr, "FAIL: --task and --no-task are mutually exclusive\n")
			os.Exit(1)
			return nil
		}

		if taskFlag == "" && !noTask {
			fmt.Fprintf(os.Stderr, "FAIL: mc commit requires --task <id> or --no-task --reason <reason>\n")
			os.Exit(1)
			return nil
		}

		if noTask && reason == "" {
			fmt.Fprintf(os.Stderr, "FAIL: --no-task requires --reason\n")
			os.Exit(1)
			return nil
		}

		message, _ := cmd.Flags().GetString("message")
		if message == "" {
			return fmt.Errorf("commit message required (use -m)")
		}

		// Task-based provenance trailers
		if noTask {
			message = fmt.Sprintf("%s\n\nMC-NoTask-Reason: %s", message, reason)
		} else if taskFlag != "" {
			var stageState StageState
			if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL: cannot read stage: %v\n", err)
				os.Exit(1)
				return nil
			}

			task, err := findTaskByID(missionDir, taskFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
				os.Exit(1)
				return nil
			}

			if task.Stage != stageState.Current {
				fmt.Fprintf(os.Stderr, "FAIL: task %s belongs to stage %q, current stage is %q\n",
					task.ID, task.Stage, stageState.Current)
				os.Exit(1)
				return nil
			}

			message = fmt.Sprintf("%s\n\nMC-Task: %s\nMC-Persona: %s\nMC-Stage: %s",
				message, task.ID, task.Persona, task.Stage)
		}

		projectDir := filepath.Dir(missionDir)

		// --- Staging ---
		if noTask {
			gitAdd := exec.Command("git", "add", "-A")
			gitAdd.Dir = projectDir
			gitAdd.Stdout = os.Stdout
			gitAdd.Stderr = os.Stderr
			if err := gitAdd.Run(); err != nil {
				return fmt.Errorf("git add failed: %w", err)
			}
		} else {
			// Selective staging based on task scope
			task, _ := findTaskByID(missionDir, taskFlag)
			exemptPaths := loadScopeExemptPaths(missionDir)

			statusCmd := exec.Command("git", "status", "--porcelain")
			statusCmd.Dir = projectDir
			statusOut, err := statusCmd.Output()
			if err != nil {
				return fmt.Errorf("git status failed: %w", err)
			}

			var toStage []string
			for _, line := range strings.Split(strings.TrimSpace(string(statusOut)), "\n") {
				if len(line) < 4 {
					continue
				}
				file := strings.TrimSpace(line[3:])
				if idx := strings.Index(file, " -> "); idx >= 0 {
					file = file[idx+4:]
				}
				if matchesSelectiveScope(file, task.ScopePaths, exemptPaths) {
					toStage = append(toStage, file)
				}
			}

			if len(toStage) > 0 {
				fmt.Printf("Staging %d file(s) matching task scope:\n", len(toStage))
				for _, f := range toStage {
					fmt.Printf("  + %s\n", f)
				}
				gitAddArgs := append([]string{"add", "--"}, toStage...)
				gitAdd := exec.Command("git", gitAddArgs...)
				gitAdd.Dir = projectDir
				gitAdd.Stdout = os.Stdout
				gitAdd.Stderr = os.Stderr
				if err := gitAdd.Run(); err != nil {
					return fmt.Errorf("git add failed: %w", err)
				}
			}

			// Scope validation after staging
			stagedFiles, err := getStagedFiles(projectDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
				os.Exit(1)
				return nil
			}
			if scopeErrs := validateScope(missionDir, taskFlag, stagedFiles); len(scopeErrs) > 0 {
				for _, e := range scopeErrs {
					fmt.Fprintf(os.Stderr, "FAIL: %s\n", e)
				}
				os.Exit(1)
				return nil
			}
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
	commitCmd.Flags().StringP("task", "t", "", "Task ID to link this commit to (appends MC trailers)")
	commitCmd.Flags().Bool("no-task", false, "Commit without task attribution (requires --reason)")
	commitCmd.Flags().String("reason", "", "Reason for --no-task commit")
	commitCmd.Flags().Bool("validate-only", false, "Only run validation, don't commit")
	commitCmd.Flags().Bool("strict", false, "Enable strict validation (requires --validate-only)")
	commitCmd.Flags().Bool("validate-provenance", false, "Validate MC trailers on all non-merge commits (requires --validate-only)")
	rootCmd.AddCommand(commitCmd)
}

// validateCommitFlags checks the mutual exclusion and dependency rules for commit flags.
func validateCommitFlags(task string, noTask bool, reason string) error {
	if task != "" && noTask {
		return fmt.Errorf("--task and --no-task are mutually exclusive")
	}
	if task == "" && !noTask {
		return fmt.Errorf("mc commit requires --task <id> or --no-task --reason <reason>")
	}
	if noTask && reason == "" {
		return fmt.Errorf("--no-task requires --reason")
	}
	return nil
}
