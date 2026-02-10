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
	var stderr io.Writer = os.Stderr
	if cmd != nil {
		force, _ = cmd.Flags().GetBool("force")
		stderr = cmd.ErrOrStderr()
	}

	if args[0] == "next" {
		// Transition to next stage
		var state StageState
		if err := readJSON(stagePath, &state); err != nil {
			return fmt.Errorf("failed to read stage: %w", err)
		}

		// Gate check before advancing — first try gates.json, then fall back to mc-core
		if !force {
			gatesFile, gatesErr := loadGates(missionDir)
			if gatesErr == nil && allCriteriaMet(&gatesFile, state.Current) {
				// All criteria satisfied in gates.json — allow advance
				fmt.Fprintf(stderr, "✓ All gate criteria met for %s (gates.json)\n", state.Current)
			} else {
				// Fall back to mc-core check
				gateResult, err := checkGateViaCore(missionDir, state.Current)
				if err != nil {
					fmt.Fprintf(stderr, "⚠ Gate check unavailable: %v\n", err)
				} else if !gateResult.CanApprove {
					fmt.Fprintf(stderr, "✗ Gate blocked for %s:\n", state.Current)
					for _, c := range gateResult.Criteria {
						icon := "✓"
						if !c.Satisfied {
							icon = "✗"
						}
						fmt.Fprintf(stderr, "  %s %s\n", icon, c.Description)
					}
					fmt.Fprintf(stderr, "\nUse --force to bypass.\n")
					return fmt.Errorf("gate criteria not met for stage %q", state.Current)
				}
			}
		} else {
			fmt.Fprintf(stderr, "⚠ --force: bypassing gate check for %s\n", state.Current)
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
		if currentIdx >= 0 && targetIdx > currentIdx && !force {
			// Forward advancement — gate check required; try gates.json first
			gatesFile, gatesErr := loadGates(missionDir)
			if gatesErr == nil && allCriteriaMet(&gatesFile, currentState.Current) {
				fmt.Fprintf(stderr, "✓ All gate criteria met for %s (gates.json)\n", currentState.Current)
			} else {
				gateResult, gateErr := checkGateViaCore(missionDir, currentState.Current)
				if gateErr != nil {
					fmt.Fprintf(stderr, "⚠ Gate check unavailable: %v\n", gateErr)
				} else if !gateResult.CanApprove {
					fmt.Fprintf(stderr, "✗ Gate blocked for %s:\n", currentState.Current)
					for _, c := range gateResult.Criteria {
						icon := "✓"
						if !c.Satisfied {
							icon = "✗"
						}
						fmt.Fprintf(stderr, "  %s %s\n", icon, c.Description)
					}
					fmt.Fprintf(stderr, "\nUse --force to bypass.\n")
					return fmt.Errorf("gate criteria not met for stage %q", currentState.Current)
				}
			}
		}
		if currentIdx >= 0 && targetIdx > currentIdx && force {
			fmt.Fprintf(stderr, "⚠ --force: bypassing gate check for %s\n", currentState.Current)
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
