package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mike/mission-control/hashid"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskUpdateCmd)

	// task create flags
	taskCreateCmd.Flags().StringP("stage", "s", "", "Stage for the task")
	taskCreateCmd.Flags().StringP("zone", "z", "", "Zone for the task")
	taskCreateCmd.Flags().String("persona", "", "Persona to assign")

	// task list flags
	taskListCmd.Flags().String("stage", "", "Filter by stage")
	taskListCmd.Flags().StringP("status", "s", "", "Filter by status")

	// task update flags
	taskUpdateCmd.Flags().StringP("status", "s", "", "New status")
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  `Create, list, and update tasks.`,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskCreate,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE:  runTaskList,
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-id>",
	Short: "Update a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	name := args[0]
	stage, _ := cmd.Flags().GetString("stage")
	zone, _ := cmd.Flags().GetString("zone")
	persona, _ := cmd.Flags().GetString("persona")

	// Read current stage if not specified
	if stage == "" {
		var stageState StageState
		if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err == nil {
			stage = stageState.Current
		}
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	task := Task{
		ID:        hashid.Generate("task", name, stage, zone, persona),
		Name:      name,
		Stage:     stage,
		Zone:      zone,
		Persona:   persona,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	tasks = append(tasks, task)

	if err := saveTasks(missionDir, tasks); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
	}

	// Output task as JSON
	output, _ := json.MarshalIndent(task, "", "  ")
	fmt.Println(string(output))

	return nil
}

func runTaskList(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	stageFilter, _ := cmd.Flags().GetString("stage")
	statusFilter, _ := cmd.Flags().GetString("status")

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	// Filter tasks
	var filtered []Task
	for _, task := range tasks {
		if stageFilter != "" && task.Stage != stageFilter {
			continue
		}
		if statusFilter != "" && task.Status != statusFilter {
			continue
		}
		filtered = append(filtered, task)
	}

	// Output as JSON
	output, _ := json.MarshalIndent(filtered, "", "  ")
	fmt.Println(string(output))

	return nil
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	taskID := args[0]
	newStatus, _ := cmd.Flags().GetString("status")

	if newStatus == "" {
		return fmt.Errorf("--status is required")
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	found := false
	for i := range tasks {
		if tasks[i].ID == taskID {
			tasks[i].Status = newStatus
			tasks[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			found = true

			output, _ := json.MarshalIndent(tasks[i], "", "  ")
			fmt.Println(string(output))
			break
		}
	}

	if !found {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := saveTasks(missionDir, tasks); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
	}

	return nil
}
