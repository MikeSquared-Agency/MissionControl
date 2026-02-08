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
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate v5 project to v6 stage-based workflow",
	Long:  `Migrates a v5 MissionControl project from phases to stages.`,
	RunE:  runMigrate,
}

func runMigrate(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	// Check if already v6
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	if _, err := os.Stat(stagePath); err == nil {
		return fmt.Errorf("already a v6 project (stage.json exists)")
	}

	// Check for v5
	phasePath := filepath.Join(missionDir, "state", "phase.json")
	if _, err := os.Stat(phasePath); os.IsNotExist(err) {
		return fmt.Errorf("no phase.json found - not a v5 project")
	}

	// Read old phase state
	var oldPhase struct {
		Current   string `json:"current"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := readJSON(phasePath, &oldPhase); err != nil {
		return fmt.Errorf("failed to read phase.json: %w", err)
	}

	// Map old phase to new stage
	stageMap := map[string]string{
		"idea":      "discovery",
		"design":    "design",
		"implement": "implement",
		"verify":    "verify",
		"document":  "document",
		"release":   "release",
	}

	newStage := stageMap[oldPhase.Current]
	if newStage == "" {
		newStage = "discovery"
	}

	// Write stage.json
	if err := writeJSON(stagePath, StageState{
		Current:   newStage,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("failed to write stage.json: %w", err)
	}

	// Regenerate gates.json with 10 stages
	gatesPath := filepath.Join(missionDir, "state", "gates.json")
	if err := writeJSON(gatesPath, GatesState{
		Gates: map[string]Gate{
			"discovery":    {Stage: "discovery", Status: "pending", Criteria: []string{"Problem space explored", "Stakeholders identified"}},
			"goal":         {Stage: "goal", Status: "pending", Criteria: []string{"Goal statement defined", "Success metrics established"}},
			"requirements": {Stage: "requirements", Status: "pending", Criteria: []string{"Requirements documented", "Acceptance criteria defined"}},
			"planning":     {Stage: "planning", Status: "pending", Criteria: []string{"Tasks broken down", "Dependencies mapped"}},
			"design":       {Stage: "design", Status: "pending", Criteria: []string{"Spec document complete", "Technical approach approved"}},
			"implement":    {Stage: "implement", Status: "pending", Criteria: []string{"All tasks complete", "Code compiles"}},
			"verify":       {Stage: "verify", Status: "pending", Criteria: []string{"Tests passing", "Review complete"}},
			"validate":     {Stage: "validate", Status: "pending", Criteria: []string{"Acceptance criteria met", "Stakeholder sign-off"}},
			"document":     {Stage: "document", Status: "pending", Criteria: []string{"README updated", "API documented"}},
			"release":      {Stage: "release", Status: "pending", Criteria: []string{"Deployed successfully", "Smoke tests pass"}},
		},
	}); err != nil {
		return fmt.Errorf("failed to write gates.json: %w", err)
	}

	// Migrate tasks.json → tasks.jsonl (change "phase" field to "stage", convert to JSONL)
	oldTasksPath := filepath.Join(missionDir, "state", "tasks.json")
	newTasksPath := filepath.Join(missionDir, "state", "tasks.jsonl")
	if _, err := os.Stat(oldTasksPath); err == nil {
		data, readErr := os.ReadFile(oldTasksPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to read tasks.json for migration: %v\n", readErr)
		} else {
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				return fmt.Errorf("failed to parse tasks.json during migration: %w", err)
			}
			var migratedTasks []Task
			if tasks, ok := raw["tasks"].([]interface{}); ok {
				for _, t := range tasks {
					if taskMap, ok := t.(map[string]interface{}); ok {
						if phase, ok := taskMap["phase"]; ok {
							phaseStr, _ := phase.(string)
							stage := stageMap[phaseStr]
							if stage == "" {
								stage = phaseStr
							}
							taskMap["stage"] = stage
							delete(taskMap, "phase")
						}
						// Re-marshal/unmarshal to get proper Task struct
						b, mErr := json.Marshal(taskMap)
						if mErr != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to marshal task during migration: %v\n", mErr)
							continue
						}
						var task Task
						if uErr := json.Unmarshal(b, &task); uErr != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to unmarshal task during migration: %v\n", uErr)
							continue
						}
						migratedTasks = append(migratedTasks, task)
					}
				}
			}
			if err := writeTasksJSONL(newTasksPath, migratedTasks); err != nil {
				fmt.Printf("Warning: failed to migrate tasks to JSONL: %v\n", err)
			} else {
				_ = os.Rename(oldTasksPath, oldTasksPath+".migrated")
			}
		}
	}

	// Create orchestrator directory
	_ = os.MkdirAll(filepath.Join(missionDir, "orchestrator", "checkpoints"), 0755)

	// Delete old phase.json
	os.Remove(phasePath)

	fmt.Println("Migration complete!")
	fmt.Printf("  Phase '%s' → Stage '%s'\n", oldPhase.Current, newStage)
	fmt.Println("  Gates regenerated with 10 stages")
	fmt.Println("  Tasks migrated from phase to stage field")
	fmt.Println("  Old phase.json removed")
	return nil
}

// requireV6 checks that this is a v6 project (stage.json exists, no phase.json)
func requireV6(missionDir string) error {
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	phasePath := filepath.Join(missionDir, "state", "phase.json")

	if _, err := os.Stat(phasePath); err == nil {
		if _, err := os.Stat(stagePath); os.IsNotExist(err) {
			return fmt.Errorf("v5 project detected. Run 'mc migrate' to upgrade to v6 stage-based workflow.")
		}
	}
	return nil
}
