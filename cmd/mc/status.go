package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	// Read gates via compat loader (handles both string and object criteria)
	gf, gatesErr := loadGates(missionDir)
	if gatesErr != nil {
		return fmt.Errorf("failed to read gates: %w", gatesErr)
	}
	status.Gates.Gates = make(map[string]Gate)
	for name, sg := range gf.Gates {
		var cs []string
		for _, c := range sg.Criteria {
			desc := c.Description
			if c.Satisfied {
				desc = "✓ " + desc
			}
			cs = append(cs, desc)
		}
		status.Gates.Gates[name] = Gate{Stage: name, Status: "pending", Criteria: cs}
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
	// 1. If --project flag is set, look up from the global registry
	if projectFlag != "" {
		reg, err := loadRegistry()
		if err != nil {
			return "", fmt.Errorf("failed to load project registry: %w", err)
		}
		missionDir, ok := reg.Projects[projectFlag]
		if !ok {
			return "", fmt.Errorf("project '%s' not found in registry (see 'mc project list')", projectFlag)
		}
		// Resolve symlinks and verify existence
		resolved, err := filepath.EvalSymlinks(missionDir)
		if err != nil {
			return "", fmt.Errorf("project '%s' .mission/ path not accessible: %s", projectFlag, missionDir)
		}
		return resolved, nil
	}

	// 2. Check MC_PROJECT env var
	if envProject := os.Getenv("MC_PROJECT"); envProject != "" {
		reg, err := loadRegistry()
		if err == nil {
			if missionDir, ok := reg.Projects[envProject]; ok {
				resolved, err := filepath.EvalSymlinks(missionDir)
				if err == nil {
					return resolved, nil
				}
			}
		}
	}

	// 3. Walk up from cwd looking for .mission/ (follows symlinks via os.Stat)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	dir := cwd
	for {
		missionDir := filepath.Join(dir, ".mission")
		// os.Stat follows symlinks, so .mission/ can be a symlink
		if info, err := os.Stat(missionDir); err == nil && info.IsDir() {
			// Resolve to real path for consistent behavior
			resolved, err := filepath.EvalSymlinks(missionDir)
			if err != nil {
				return "", fmt.Errorf(".mission/ found but symlink broken: %w", err)
			}
			return resolved, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf(".mission/ not found - run 'mc init' first")
}

// printStatusSummary prints a human-friendly status summary to stderr.
// It is called after stage and task mutations so the user gets immediate feedback
// without breaking JSON output on stdout.
func printStatusSummary(missionDir string, cmd *cobra.Command) {
	var w io.Writer = os.Stderr
	if cmd != nil {
		w = cmd.ErrOrStderr()
	}

	// Read current stage
	var state StageState
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	if err := readJSON(stagePath, &state); err != nil {
		return // silently skip if we can't read stage
	}

	// Load tasks and count by status for current stage
	tasks, err := loadTasks(missionDir)
	if err != nil {
		return
	}

	counts := map[string]int{}
	total := 0
	for _, t := range tasks {
		if t.Stage == state.Current {
			counts[t.Status]++
			total++
		}
	}

	idx := stageIndex(state.Current)

	fmt.Fprintf(w, "\n── Mission Status ──────────────────────\n")
	fmt.Fprintf(w, "Stage: %s (%d/%d)\n", state.Current, idx+1, len(stages))

	// Tasks line — only show non-zero buckets
	var parts []string
	for _, s := range []string{"complete", "in_progress", "pending", "blocked"} {
		if c := counts[s]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, s))
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, "Tasks: %s  (%d total)\n", strings.Join(parts, " · "), total)
	} else {
		fmt.Fprintf(w, "Tasks: none\n")
	}

	// Gate line — use compat loader
	if gf, err := loadGates(missionDir); err == nil {
		if allCriteriaMet(&gf, state.Current) {
			fmt.Fprintf(w, "Gate:  ✓ all criteria met\n")
		} else if sg, ok := gf.Gates[state.Current]; ok {
			met := 0
			for _, c := range sg.Criteria {
				if c.Satisfied {
					met++
				}
			}
			fmt.Fprintf(w, "Gate:  · %d/%d criteria met\n", met, len(sg.Criteria))
		}
	}

	fmt.Fprintf(w, "────────────────────────────────────────\n")
}

func readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
