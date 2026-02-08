package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(gateCmd)
	gateCmd.AddCommand(gateCheckCmd)
	gateCmd.AddCommand(gateApproveCmd)
}

var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Manage stage gates",
	Long:  `Check gate criteria or approve gates to transition stages.`,
}

var gateCheckCmd = &cobra.Command{
	Use:   "check <stage>",
	Short: "Check if gate criteria are met for a stage",
	Args:  cobra.ExactArgs(1),
	RunE:  runGateCheck,
}

var gateApproveCmd = &cobra.Command{
	Use:   "approve <stage>",
	Short: "Approve a gate and transition to next stage",
	Args:  cobra.ExactArgs(1),
	RunE:  runGateApprove,
}

type GateCheckResult struct {
	Stage    string            `json:"stage"`
	Status   string            `json:"status"`
	Ready    bool              `json:"ready"`
	Criteria []CriterionStatus `json:"criteria"`
	Tasks    TasksSummary      `json:"tasks"`
}

type CriterionStatus struct {
	Name string `json:"name"`
	Met  bool   `json:"met"`
}

type TasksSummary struct {
	Total    int `json:"total"`
	Complete int `json:"complete"`
	Pending  int `json:"pending"`
	Blocked  int `json:"blocked"`
}

func runGateCheck(cmd *cobra.Command, args []string) error {
	stage := args[0]

	if !isValidStage(stage) {
		return fmt.Errorf("invalid stage: %s", stage)
	}

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	if err := requireV6(missionDir); err != nil {
		return err
	}

	// Read gates
	gatesPath := filepath.Join(missionDir, "state", "gates.json")
	var gatesState GatesState
	if err := readJSON(gatesPath, &gatesState); err != nil {
		return fmt.Errorf("failed to read gates: %w", err)
	}

	gate, ok := gatesState.Gates[stage]
	if !ok {
		return fmt.Errorf("gate not found: %s", stage)
	}

	// Read tasks to calculate summary
	tasksPath := filepath.Join(missionDir, "state", "tasks.json")
	var tasksState TasksState
	if err := readJSON(tasksPath, &tasksState); err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	// Calculate task summary for this stage
	var summary TasksSummary
	for _, task := range tasksState.Tasks {
		if task.Stage == stage {
			summary.Total++
			switch task.Status {
			case "complete":
				summary.Complete++
			case "pending":
				summary.Pending++
			case "blocked":
				summary.Blocked++
			}
		}
	}

	// Check criteria (simplified - all tasks complete means ready)
	var criteria []CriterionStatus
	for _, c := range gate.Criteria {
		// For now, mark criteria as met if there are no pending/blocked tasks
		met := summary.Total > 0 && summary.Pending == 0 && summary.Blocked == 0
		criteria = append(criteria, CriterionStatus{
			Name: c,
			Met:  met,
		})
	}

	// Overall ready check
	ready := true
	for _, c := range criteria {
		if !c.Met {
			ready = false
			break
		}
	}

	result := GateCheckResult{
		Stage:    stage,
		Status:   gate.Status,
		Ready:    ready,
		Criteria: criteria,
		Tasks:    summary,
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))

	return nil
}

func runGateApprove(cmd *cobra.Command, args []string) error {
	stage := args[0]

	if !isValidStage(stage) {
		return fmt.Errorf("invalid stage: %s", stage)
	}

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	if err := requireV6(missionDir); err != nil {
		return err
	}

	// Update gate status
	gatesPath := filepath.Join(missionDir, "state", "gates.json")
	var gatesState GatesState
	if err := readJSON(gatesPath, &gatesState); err != nil {
		return fmt.Errorf("failed to read gates: %w", err)
	}

	gate, ok := gatesState.Gates[stage]
	if !ok {
		return fmt.Errorf("gate not found: %s", stage)
	}

	gate.Status = "approved"
	gate.ApprovedAt = time.Now().UTC().Format(time.RFC3339)
	gatesState.Gates[stage] = gate

	if err := writeJSON(gatesPath, gatesState); err != nil {
		return fmt.Errorf("failed to update gate: %w", err)
	}

	gitAutoCommit(missionDir, CommitCategoryGate, fmt.Sprintf("approve gate: %s", stage))

	// Auto-checkpoint on gate approval (G3.1)
	if cp, err := createCheckpoint(missionDir, ""); err == nil {
		fmt.Printf("Checkpoint created: %s\n", cp.ID)
	}

	// Transition to next stage
	nextStage, err := getNextStage(stage)
	if err != nil {
		fmt.Printf("Gate approved: %s (final stage)\n", stage)
		return nil
	}

	stagePath := filepath.Join(missionDir, "state", "stage.json")
	stageState := StageState{
		Current:   nextStage,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := writeJSON(stagePath, stageState); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	gitAutoCommit(missionDir, CommitCategoryStage, fmt.Sprintf("advance stage: %s → %s", stage, nextStage))

	fmt.Printf("Gate approved: %s → %s\n", stage, nextStage)

	return nil
}
