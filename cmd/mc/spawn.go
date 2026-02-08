package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().StringP("zone", "z", "", "Zone to work in")
	spawnCmd.Flags().String("task-id", "", "Task ID to associate with")
}

var spawnCmd = &cobra.Command{
	Use:   "spawn <persona> <task-description>",
	Short: "Spawn a worker process",
	Long: `Spawns a Claude Code worker with the specified persona.

Examples:
  mc spawn developer "Implement login form" --zone frontend
  mc spawn researcher "Research auth solutions" --zone backend`,
	Args: cobra.ExactArgs(2),
	RunE: runSpawn,
}

var validPersonas = map[string]bool{
	"researcher": true,
	"designer":   true,
	"architect":  true,
	"developer":  true,
	"reviewer":   true,
	"security":   true,
	"tester":     true,
	"qa":         true,
	"docs":       true,
	"devops":     true,
	"debugger":   true,
}

func runSpawn(cmd *cobra.Command, args []string) error {
	persona := strings.ToLower(args[0])
	taskDesc := args[1]
	zone, _ := cmd.Flags().GetString("zone")
	taskID, _ := cmd.Flags().GetString("task-id")

	if !validPersonas[persona] {
		return fmt.Errorf("invalid persona: %s", persona)
	}

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	// Generate worker ID
	workerID := uuid.New().String()[:8]

	// Create worker prompt from template
	promptPath := filepath.Join(missionDir, "prompts", persona+".md")
	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt template: %w", err)
	}

	// Substitute template variables
	prompt := string(promptData)
	prompt = strings.ReplaceAll(prompt, "{{zone}}", zone)
	prompt = strings.ReplaceAll(prompt, "{{task_description}}", taskDesc)
	prompt = strings.ReplaceAll(prompt, "{{task_id}}", taskID)
	prompt = strings.ReplaceAll(prompt, "{{worker_id}}", workerID)

	// Write temp prompt file
	tmpPrompt := filepath.Join(os.TempDir(), fmt.Sprintf("mc-worker-%s.md", workerID))
	if err := os.WriteFile(tmpPrompt, []byte(prompt), 0644); err != nil {
		return fmt.Errorf("failed to write temp prompt: %w", err)
	}

	// Determine working directory
	workDir := filepath.Dir(missionDir) // Project root
	if zone != "" {
		zoneDir := filepath.Join(workDir, zone)
		if info, err := os.Stat(zoneDir); err == nil && info.IsDir() {
			workDir = zoneDir
		}
	}

	// Spawn Claude Code process
	claudeCmd := exec.Command("claude",
		"--print", taskDesc,
	)
	claudeCmd.Dir = workDir
	claudeCmd.Env = append(os.Environ(),
		fmt.Sprintf("CLAUDE_SYSTEM_PROMPT=%s", tmpPrompt),
	)

	// Start the process
	if err := claudeCmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn worker: %w", err)
	}

	// Record worker in state
	workersPath := filepath.Join(missionDir, "state", "workers.json")
	var state WorkersState
	if err := readJSON(workersPath, &state); err != nil {
		// If file doesn't exist or is empty, start fresh
		state = WorkersState{Workers: []Worker{}}
	}

	worker := Worker{
		ID:        workerID,
		Persona:   persona,
		TaskID:    taskID,
		Zone:      zone,
		Status:    "running",
		PID:       claudeCmd.Process.Pid,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	state.Workers = append(state.Workers, worker)

	if err := writeJSON(workersPath, state); err != nil {
		return fmt.Errorf("failed to update workers state: %w", err)
	}

	writeAuditLog(missionDir, AuditWorkerSpawned, "cli", map[string]interface{}{
		"worker_id": workerID,
		"persona":   persona,
		"task_id":   taskID,
		"zone":      zone,
		"pid":       claudeCmd.Process.Pid,
	})

	// Auto-commit
	gitAutoCommit(missionDir, CommitCategoryWorker, fmt.Sprintf("spawn %s (%s)", shortID(workerID), persona))

	// Output worker info
	output, _ := json.MarshalIndent(worker, "", "  ")
	fmt.Println(string(output))

	return nil
}
