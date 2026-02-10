package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var prTitle string
var prBase string

func init() {
	prCmd.Flags().StringVar(&prTitle, "title", "", "PR title (required)")
	prCmd.Flags().StringVar(&prBase, "base", "main", "Base branch for the PR")
	_ = prCmd.MarkFlagRequired("title")
	rootCmd.AddCommand(prCmd)
}

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create a pull request from mission state",
	Long:  `Auto-generates a PR body from .mission state (objective, stages, tasks, findings) and calls gh pr create.`,
	RunE:  runPR,
}

func runPR(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	var body strings.Builder

	// Objective
	objective, err := os.ReadFile(filepath.Join(missionDir, "state", "objective.md"))
	if err != nil {
		return fmt.Errorf("failed to read objective.md: %w", err)
	}
	body.WriteString("## Objective\n\n")
	body.WriteString(strings.TrimSpace(string(objective)))
	body.WriteString("\n\n")

	// Stage Progression
	var stage StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stage); err != nil {
		return fmt.Errorf("failed to read stage.json: %w", err)
	}
	body.WriteString("## Stage Progression\n\n")
	body.WriteString(fmt.Sprintf("Current stage: **%s** (updated: %s)\n\n", stage.Current, stage.UpdatedAt))

	// Tasks
	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}
	body.WriteString("## Tasks\n\n")
	body.WriteString("| ID | Name | Stage | Status |\n")
	body.WriteString("|---|---|---|---|\n")
	for _, t := range tasks {
		body.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", t.ID, t.Name, t.Stage, t.Status))
	}
	body.WriteString("\n")

	// Findings Summaries (for done tasks)
	var summaries []string
	findingsDir := filepath.Join(missionDir, "findings")
	for _, t := range tasks {
		if t.Status != "done" {
			continue
		}
		fPath := filepath.Join(findingsDir, t.ID+".md")
		summary, err := extractFindingsSummary(fPath)
		if err != nil {
			continue // skip if no findings file
		}
		summaries = append(summaries, fmt.Sprintf("- **%s** (%s): %s", t.Name, t.ID, summary))
	}
	if len(summaries) > 0 {
		body.WriteString("## Findings Summaries\n\n")
		for _, s := range summaries {
			body.WriteString(s + "\n")
		}
		body.WriteString("\n")
	}

	// Shell out to gh pr create
	ghArgs := []string{"pr", "create", "--title", prTitle, "--body", body.String(), "--base", prBase}
	ghCmd := exec.Command("gh", ghArgs...)
	ghCmd.Stderr = os.Stderr
	out, err := ghCmd.Output()
	if err != nil {
		return fmt.Errorf("gh pr create failed: %w", err)
	}

	fmt.Print(string(out))
	return nil
}

// extractFindingsSummary reads a findings markdown file and returns the Summary line value.
func extractFindingsSummary(path string) (string, error) {
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
	return "", fmt.Errorf("no Summary line found in %s", path)
}
