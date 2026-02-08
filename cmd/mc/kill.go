package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolP("force", "f", false, "Force kill (SIGKILL)")
}

var killCmd = &cobra.Command{
	Use:   "kill <worker-id>",
	Short: "Kill a worker process",
	Long: `Terminates a running worker process.

Examples:
  mc kill abc123
  mc kill abc123 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runKill,
}

func runKill(cmd *cobra.Command, args []string) error {
	workerID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	workersPath := filepath.Join(missionDir, "state", "workers.json")
	var state WorkersState
	if err := readJSON(workersPath, &state); err != nil {
		return fmt.Errorf("failed to read workers: %w", err)
	}

	// Find worker
	var worker *Worker
	var workerIdx int
	for i := range state.Workers {
		if state.Workers[i].ID == workerID {
			worker = &state.Workers[i]
			workerIdx = i
			break
		}
	}

	if worker == nil {
		return fmt.Errorf("worker not found: %s", workerID)
	}

	// Kill the process
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}

	if err := syscall.Kill(worker.PID, sig); err != nil {
		// Process might already be dead
		if err != syscall.ESRCH {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Update worker status
	state.Workers[workerIdx].Status = "killed"

	if err := writeJSON(workersPath, state); err != nil {
		return fmt.Errorf("failed to update workers: %w", err)
	}

	writeAuditLog(missionDir, AuditWorkerKilled, "cli", map[string]interface{}{
		"worker_id": workerID,
		"pid":       worker.PID,
		"force":     force,
	})

	// Auto-commit worker kill
	gitAutoCommit(missionDir, CommitCategoryWorker, fmt.Sprintf("kill %s", shortID(workerID)))

	fmt.Printf("Killed worker %s (PID %d)\n", workerID, worker.PID)

	// Also update associated task if exists
	if worker.TaskID != "" {
		tasks, loadErr := loadTasks(missionDir)
		if loadErr == nil {
			for i := range tasks {
				if tasks[i].ID == worker.TaskID {
					tasks[i].Status = "blocked"
					tasks[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
					break
				}
			}
			saveTasks(missionDir, tasks)
		}
	}

	return nil
}
