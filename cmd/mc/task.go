package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DarlingtonDeveloper/MissionControl/hashid"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskDepsCmd)
	rootCmd.AddCommand(queueCmd)

	// task create flags
	taskCreateCmd.Flags().StringP("stage", "s", "", "Stage for the task")
	taskCreateCmd.Flags().StringP("zone", "z", "", "Zone for the task")
	taskCreateCmd.Flags().String("persona", "", "Persona to assign")
	taskCreateCmd.Flags().StringSlice("depends-on", nil, "Task IDs this task depends on")
	taskCreateCmd.Flags().Bool("force", false, "Bypass stage validation")
	taskCreateCmd.Flags().String("scope-paths", "", "Comma-separated list of file paths in scope for this task")

	// task list flags
	taskListCmd.Flags().String("stage", "", "Filter by stage")
	taskListCmd.Flags().StringP("status", "s", "", "Filter by status")
	taskListCmd.Flags().Bool("ready", false, "Show only tasks ready to work on (pending + all deps met)")

	// task update flags
	taskUpdateCmd.Flags().StringP("status", "s", "", "New status")

	// task deps flags
	taskDepsCmd.Flags().Bool("tree", false, "Show ASCII dependency tree")
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

var taskDepsCmd = &cobra.Command{
	Use:   "deps <task-id>",
	Short: "Show dependencies for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDeps,
}

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show tasks ready to be worked on",
	Long:  `Shows pending tasks whose dependencies are all complete.`,
	RunE:  runQueue,
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	// Warn if objective.md is missing or empty
	objPath := filepath.Join(missionDir, "state", "objective.md")
	objContent, objErr := os.ReadFile(objPath)
	if objErr != nil || len(strings.TrimSpace(string(objContent))) == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Warning: objective.md is missing or empty — consider defining the mission objective first\n")
	}

	name := args[0]
	stage, _ := cmd.Flags().GetString("stage")
	zone, _ := cmd.Flags().GetString("zone")
	persona, _ := cmd.Flags().GetString("persona")
	dependsOn, _ := cmd.Flags().GetStringSlice("depends-on")
	scopePathsStr, _ := cmd.Flags().GetString("scope-paths")
	var scopePaths []string
	if scopePathsStr != "" {
		for _, p := range strings.Split(scopePathsStr, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				scopePaths = append(scopePaths, p)
			}
		}
	}

	force, _ := cmd.Flags().GetBool("force")

	// Read current stage
	var currentStage string
	var stageState StageState
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	if err := readJSON(stagePath, &stageState); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read stage: %w", err)
		}
	} else {
		currentStage = stageState.Current
	}

	// Default task stage to current stage if not specified
	if stage == "" {
		stage = currentStage
	}

	// Validate: reject tasks for stages ahead of current stage
	if currentStage != "" && stage != "" {
		taskIdx := stageIndex(stage)
		curIdx := stageIndex(currentStage)
		if taskIdx > curIdx && curIdx >= 0 && taskIdx >= 0 {
			if !force {
				return fmt.Errorf("cannot create task for stage %q — current stage is %q.\n       Advance to %q first, or use --force to bypass", stage, currentStage, stage)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: creating task for future stage %q (current: %q). This bypasses progressive refinement.\n", stage, currentStage)
		}
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	taskID := hashid.Generate("task", name, stage, zone, persona)

	// Check for duplicate IDs
	for _, existing := range tasks {
		if existing.ID == taskID {
			return fmt.Errorf("task with this ID already exists: %s (name=%q)", taskID, existing.Name)
		}
	}

	task := Task{
		ID:         taskID,
		Name:       name,
		Stage:      stage,
		Zone:       zone,
		Persona:    persona,
		Status:     "pending",
		DependsOn:  dependsOn,
		ScopePaths: scopePaths,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	tasks = append(tasks, task)

	if err := saveTasks(missionDir, tasks); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
	}

	writeAuditLog(missionDir, AuditTaskCreated, "cli", map[string]interface{}{
		"task_id": task.ID,
		"name":    task.Name,
		"stage":   task.Stage,
		"zone":    task.Zone,
		"persona": task.Persona,
	})

	// Auto-commit
	gitAutoCommit(missionDir, CommitCategoryTask, taskCommitMsg("create", task.ID, task.Name))

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
	readyOnly, _ := cmd.Flags().GetBool("ready")

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	taskMap := buildTaskMap(tasks)

	// Filter tasks
	var filtered []Task
	for _, task := range tasks {
		if stageFilter != "" && task.Stage != stageFilter {
			continue
		}
		if statusFilter != "" && task.Status != statusFilter {
			continue
		}
		if readyOnly && !isReady(task, taskMap) {
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
	var oldStatus string
	for i := range tasks {
		if tasks[i].ID == taskID {
			oldStatus = tasks[i].Status
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

	// Audit after successful persistence
	auditAction := AuditTaskUpdated
	if newStatus == "complete" {
		auditAction = AuditTaskCompleted
	}
	writeAuditLog(missionDir, auditAction, "cli", map[string]interface{}{
		"task_id":    taskID,
		"old_status": oldStatus,
		"new_status": newStatus,
	})

	// Auto-commit
	gitAutoCommit(missionDir, CommitCategoryTask, taskCommitMsg("update", taskID, newStatus))

	printStatusSummary(missionDir, cmd)

	return nil
}

// buildTaskMap creates a lookup map of task ID to Task.
func buildTaskMap(tasks []Task) map[string]Task {
	m := make(map[string]Task, len(tasks))
	for _, t := range tasks {
		m[t.ID] = t
	}
	return m
}

// isReady returns true if a task is pending and all its dependencies are complete.
func isReady(task Task, taskMap map[string]Task) bool {
	if task.Status != "pending" {
		return false
	}
	for _, depID := range task.DependsOn {
		dep, ok := taskMap[depID]
		if !ok || dep.Status != "done" {
			return false
		}
	}
	return true
}

func runTaskDeps(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	taskID := args[0]
	showTree, _ := cmd.Flags().GetBool("tree")

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	taskMap := buildTaskMap(tasks)

	root, ok := taskMap[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if showTree {
		visited := make(map[string]bool)
		printDepTree(root, taskMap, "", true, visited)
	} else {
		// Flat list of direct + transitive dependencies
		deps := collectDeps(root, taskMap, make(map[string]bool))
		output, _ := json.MarshalIndent(deps, "", "  ")
		fmt.Println(string(output))
	}

	return nil
}

func printDepTree(task Task, taskMap map[string]Task, prefix string, isLast bool, visited map[string]bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	statusIcon := "○"
	switch task.Status {
	case "done":
		statusIcon = "●"
	case "in_progress":
		statusIcon = "◐"
	case "blocked":
		statusIcon = "✕"
	}

	fmt.Printf("%s%s%s %s [%s]\n", prefix, connector, statusIcon, task.Name, task.ID)

	if visited[task.ID] {
		if len(task.DependsOn) > 0 {
			childPrefix := prefix + "    "
			if !isLast && prefix != "" {
				childPrefix = prefix + "│   "
			}
			fmt.Printf("%s└── (circular ref)\n", childPrefix)
		}
		return
	}
	visited[task.ID] = true

	childPrefix := prefix + "    "
	if !isLast && prefix != "" {
		childPrefix = prefix + "│   "
	}

	for i, depID := range task.DependsOn {
		dep, ok := taskMap[depID]
		if !ok {
			fmt.Printf("%s%s? unknown [%s]\n", childPrefix, map[bool]string{true: "└── ", false: "├── "}[i == len(task.DependsOn)-1], depID)
			continue
		}
		printDepTree(dep, taskMap, childPrefix, i == len(task.DependsOn)-1, visited)
	}
}

func collectDeps(task Task, taskMap map[string]Task, seen map[string]bool) []Task {
	var result []Task
	for _, depID := range task.DependsOn {
		if seen[depID] {
			continue
		}
		seen[depID] = true
		if dep, ok := taskMap[depID]; ok {
			result = append(result, dep)
			result = append(result, collectDeps(dep, taskMap, seen)...)
		}
	}
	return result
}

func runQueue(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	taskMap := buildTaskMap(tasks)

	var ready []Task
	for _, task := range tasks {
		if isReady(task, taskMap) {
			ready = append(ready, task)
		}
	}

	output, _ := json.MarshalIndent(ready, "", "  ")
	fmt.Println(string(output))
	return nil
}
