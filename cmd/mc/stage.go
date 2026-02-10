package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	stageCmd.Flags().Bool("force", false, "Bypass gate check when advancing stages")
	rootCmd.AddCommand(stageCmd)
	rootCmd.AddCommand(phaseCmd)
}

// RustGateResult mirrors mc-core check-gate JSON output.
type RustGateResult struct {
	Stage      string          `json:"stage"`
	Status     string          `json:"status"`
	Criteria   []RustCriterion `json:"criteria"`
	CanApprove bool            `json:"can_approve"`
}

// RustCriterion is a single gate criterion from mc-core.
type RustCriterion struct {
	Description string `json:"description"`
	Satisfied   bool   `json:"satisfied"`
}

func checkGateViaCore(missionDir, stage string) (*RustGateResult, error) {
	bin := findMcCore()
	if bin == "" {
		return nil, fmt.Errorf("mc-core not found (install it or place it alongside mc)")
	}
	cmd := exec.Command(bin, "check-gate", stage, "--mission-dir", missionDir)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("mc-core check-gate failed (exit %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("mc-core check-gate exec error: %w", err)
	}
	var result RustGateResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse mc-core output: %w", err)
	}
	return &result, nil
}

// phaseCmd is a deprecated alias for stageCmd
var phaseCmd = &cobra.Command{
	Use:        "phase [next]",
	Short:      "Deprecated: use 'mc stage' instead",
	Deprecated: "use 'mc stage' instead",
	RunE:       runStage,
}

var stageCmd = &cobra.Command{
	Use:   "stage [next]",
	Short: "Get or set the current stage",
	Long: `Get the current stage, or use 'mc stage next' to transition.

Examples:
  mc stage         # Show current stage
  mc stage next    # Transition to next stage`,
	RunE: runStage,
}

var stages = []string{"discovery", "goal", "requirements", "planning", "design", "implement", "verify", "validate", "document", "release"}

func runStage(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	stagePath := filepath.Join(missionDir, "state", "stage.json")

	if len(args) == 0 {
		// Just show current stage
		var state StageState
		if err := readJSON(stagePath, &state); err != nil {
			return fmt.Errorf("failed to read stage: %w", err)
		}
		fmt.Println(state.Current)
		return nil
	}

	var force bool
	var programmatic bool
	var stderr io.Writer = os.Stderr
	if cmd != nil {
		force, _ = cmd.Flags().GetBool("force")
		stderr = cmd.ErrOrStderr()
	} else {
		programmatic = true
	}

	if args[0] == "next" {
		// Transition to next stage
		var state StageState
		if err := readJSON(stagePath, &state); err != nil {
			return fmt.Errorf("failed to read stage: %w", err)
		}

		// Stage enforcement checks (zero-task, velocity, mandatory tasks)
		if err := advanceStageChecked(missionDir, state.Current, force || programmatic); err != nil {
			return err
		}

		// Gate check
		if err := enforceGate(missionDir, state.Current, force, stderr); err != nil {
			return err
		}

		nextStage, err := getNextStage(state.Current)
		if err != nil {
			return err
		}

		state.Current = nextStage
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		if err := writeJSON(stagePath, state); err != nil {
			return fmt.Errorf("failed to write stage: %w", err)
		}

		writeAuditLog(missionDir, AuditStageAdvanced, "cli", map[string]interface{}{
			"from_stage": getPrevStage(nextStage),
			"to_stage":   nextStage,
		})

		gitAutoCommit(missionDir, CommitCategoryStage, fmt.Sprintf("advance %s → %s", getPrevStage(nextStage), nextStage))

		// Initialize gate criteria for the new stage
		if err := initGateForStage(missionDir, nextStage); err != nil {
			fmt.Fprintf(stderr, "⚠ Could not initialize gate for %s: %v\n", nextStage, err)
		}

		fmt.Printf("Stage transitioned: %s → %s\n", getPrevStage(nextStage), nextStage)
		printStatusSummary(missionDir, cmd)
		return nil
	}

	// Set specific stage
	targetStage := args[0]
	if !isValidStage(targetStage) {
		return fmt.Errorf("invalid stage: %s (valid: %v)", targetStage, stages)
	}

	// Read current stage to detect forward jumps
	var currentState StageState
	if err := readJSON(stagePath, &currentState); err == nil {
		currentIdx := stageIndex(currentState.Current)
		targetIdx := stageIndex(targetStage)
		if currentIdx >= 0 && targetIdx > currentIdx {
			// Stage enforcement checks (zero-task, velocity, mandatory tasks)
			if err := advanceStageChecked(missionDir, currentState.Current, force || programmatic); err != nil {
				return err
			}
			// Gate check
			if err := enforceGate(missionDir, currentState.Current, force, stderr); err != nil {
				return err
			}
		}
	}

	state := StageState{
		Current:   targetStage,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := writeJSON(stagePath, state); err != nil {
		return fmt.Errorf("failed to write stage: %w", err)
	}

	writeAuditLog(missionDir, AuditStageSet, "cli", map[string]interface{}{
		"stage": targetStage,
	})

	gitAutoCommit(missionDir, CommitCategoryStage, fmt.Sprintf("set %s", targetStage))

	// Initialize gate criteria for the new stage
	if err := initGateForStage(missionDir, targetStage); err != nil {
		fmt.Fprintf(stderr, "⚠ Could not initialize gate for %s: %v\n", targetStage, err)
	}

	fmt.Printf("Stage set to: %s\n", targetStage)
	printStatusSummary(missionDir, cmd)
	return nil
}

// enforceGate checks gate criteria before allowing stage advancement.
// Checks both gates.json criteria AND mc-core structural checks (integrator, reviewer).
func enforceGate(missionDir, stage string, force bool, stderr io.Writer) error {
	if force {
		fmt.Fprintf(stderr, "⚠ --force: bypassing gate check for %s\n", stage)
		return nil
	}

	// 1. Check gates.json criteria
	gatesFile, gatesErr := loadGates(missionDir)
	gatesOK := gatesErr == nil && allCriteriaMet(&gatesFile, stage)
	if gatesOK {
		fmt.Fprintf(stderr, "✓ All gate criteria met for %s (gates.json)\n", stage)
	}

	// 2. Always run mc-core — it checks structural requirements (integrator, reviewer)
	// that gates.json doesn't know about
	gateResult, coreErr := checkGateViaCore(missionDir, stage)
	if coreErr != nil {
		// mc-core unavailable — fall back to gates.json alone
		if !gatesOK {
			fmt.Fprintf(stderr, "⚠ Gate check unavailable: %v\n", coreErr)
		}
		return nil
	}

	// mc-core returned a result — enforce it
	if !gateResult.CanApprove {
		fmt.Fprintf(stderr, "✗ Gate blocked for %s:\n", stage)
		for _, c := range gateResult.Criteria {
			icon := "✓"
			if !c.Satisfied {
				icon = "✗"
			}
			fmt.Fprintf(stderr, "  %s %s\n", icon, c.Description)
		}
		fmt.Fprintf(stderr, "\nUse --force to bypass.\n")
		return fmt.Errorf("gate criteria not met for stage %q", stage)
	}

	return nil
}

// advanceStageChecked validates pre-conditions before allowing stage advancement.
func advanceStageChecked(missionDir string, currentStage string, force bool) error {
	if force {
		return nil
	}

	// Exempt stages don't require tasks
	exemptStages := map[string]bool{
		"goal": true, "requirements": true, "planning": true, "design": true,
	}

	// Load tasks for the current stage
	tasks, _ := loadTasks(missionDir)
	var stageTasks []Task
	var completedTasks int
	for _, t := range tasks {
		if t.Stage == currentStage {
			stageTasks = append(stageTasks, t)
			if t.Status == "done" {
				completedTasks++
			}
		}
	}

	// Zero-task block (non-exempt stages)
	if !exemptStages[currentStage] && len(stageTasks) == 0 {
		return fmt.Errorf("stage %s has no tasks — create at least one or use --force", currentStage)
	}

	// Velocity check: stage lasted <10s with no completed tasks (non-exempt stages only)
	if !exemptStages[currentStage] {
		stagePath := filepath.Join(missionDir, "state", "stage.json")
		var state StageState
		if err := readJSON(stagePath, &state); err == nil {
			if updatedAt, err := time.Parse(time.RFC3339, state.UpdatedAt); err == nil {
				if time.Since(updatedAt) < 10*time.Second && completedTasks == 0 {
					return fmt.Errorf("stage %s lasted <10s with no completed tasks — are you rubber-stamping?", currentStage)
				}
			}
		}
	}

	// Mandatory reviewer for verify stage
	if currentStage == "verify" {
		hasReviewer := false
		for _, t := range stageTasks {
			if t.Persona == "reviewer" && t.Status == "done" {
				hasReviewer = true
				break
			}
		}
		if !hasReviewer {
			return fmt.Errorf("verify stage requires at least one reviewer task")
		}
	}

	return nil
}

func getNextStage(current string) (string, error) {
	for i, stage := range stages {
		if stage == current {
			if i == len(stages)-1 {
				return "", fmt.Errorf("already at final stage: %s", current)
			}
			return stages[i+1], nil
		}
	}
	return "", fmt.Errorf("unknown stage: %s", current)
}

func getPrevStage(current string) string {
	for i, stage := range stages {
		if stage == current && i > 0 {
			return stages[i-1]
		}
	}
	return ""
}

func stageIndex(name string) int {
	for i, s := range stages {
		if s == name {
			return i
		}
	}
	return -1
}

func isValidStage(stage string) bool {
	for _, s := range stages {
		if s == stage {
			return true
		}
	}
	return false
}
