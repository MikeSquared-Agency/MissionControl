package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(briefingCmd)
	briefingCmd.AddCommand(generateCmd)
	generateCmd.Flags().String("objective", "", "Custom objective for the briefing")
}

var briefingCmd = &cobra.Command{
	Use:   "briefing",
	Short: "Manage task briefings",
}

var generateCmd = &cobra.Command{
	Use:   "generate <task-id>",
	Short: "Generate a briefing for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateBriefing,
}

func runGenerateBriefing(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	taskID := args[0]
	objective, _ := cmd.Flags().GetString("objective")
	data, err := generateBriefing(missionDir, taskID, objective)
	if err != nil {
		return err
	}

	outPath := filepath.Join(missionDir, "handoffs", taskID+"-briefing.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Briefing written to %s\n", outPath)
	return nil
}

var validTaskID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func generateBriefing(missionDir string, taskID string, objective string) ([]byte, error) {
	if !validTaskID.MatchString(taskID) {
		return nil, fmt.Errorf("invalid task ID %q: must match [a-zA-Z0-9_-]+", taskID)
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	taskMap := make(map[string]Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	task, ok := taskMap[taskID]
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	// Collect deps
	seen := make(map[string]bool)
	deps := collectDeps(task, taskMap, seen)

	// Check all deps are complete and collect findings
	var predPaths []string
	predSummaries := make(map[string]string)

	for _, dep := range deps {
		if dep.Status != "done" {
			return nil, fmt.Errorf("dependency %s (%s) is not complete (status: %s)", dep.ID, dep.Name, dep.Status)
		}

		findingsPath := filepath.Join(missionDir, "findings", dep.ID+".md")
		if _, err := os.Stat(findingsPath); err != nil {
			return nil, fmt.Errorf("findings file missing for dependency %s: %s", dep.ID, findingsPath)
		}

		relPath := ".mission/findings/" + dep.ID + ".md"
		predPaths = append(predPaths, relPath)

		summary, err := extractSummary(findingsPath)
		if err == nil && summary != "" {
			predSummaries[dep.ID] = summary
		}
	}

	output := ".mission/findings/" + taskID + ".md"

	briefing := map[string]interface{}{
		"task_id":     task.ID,
		"task_name":   task.Name,
		"stage":       task.Stage,
		"zone":        task.Zone,
		"persona":     task.Persona,
		"scope_paths": task.ScopePaths,
		"output":      output,
	}

	if objective != "" {
		briefing["objective"] = objective
	}

	if len(predPaths) > 0 {
		briefing["predecessor_findings_paths"] = predPaths
	}
	if len(predSummaries) > 0 {
		briefing["predecessor_summaries"] = predSummaries
	}

	return json.MarshalIndent(briefing, "", "  ")
}

func extractSummary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Summary:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Summary:")), nil
		}
	}
	return "", nil
}
