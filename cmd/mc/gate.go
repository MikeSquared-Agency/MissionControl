package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// --- New gate management types and functions (TDD GREEN) ---

type GateCriterion struct {
	Description string `json:"description"`
	Satisfied   bool   `json:"satisfied"`
}

type StageGate struct {
	Criteria []GateCriterion `json:"criteria"`
}

type GatesFile struct {
	Gates map[string]StageGate `json:"gates"`
}

func loadGates(missionDir string) (GatesFile, error) {
	p := filepath.Join(missionDir, "state", "gates.json")
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return GatesFile{Gates: map[string]StageGate{}}, nil
		}
		return GatesFile{}, err
	}
	var gf GatesFile
	if err := json.Unmarshal(data, &gf); err != nil {
		// Try legacy format where criteria are plain strings
		var legacy struct {
			Gates map[string]struct {
				Stage    string   `json:"stage"`
				Status   string   `json:"status"`
				Criteria []string `json:"criteria"`
			} `json:"gates"`
		}
		if err2 := json.Unmarshal(data, &legacy); err2 != nil {
			return GatesFile{}, err // return original error
		}
		gf.Gates = make(map[string]StageGate)
		for name, sg := range legacy.Gates {
			var criteria []GateCriterion
			for _, c := range sg.Criteria {
				criteria = append(criteria, GateCriterion{Description: c, Satisfied: false})
			}
			gf.Gates[name] = StageGate{Criteria: criteria}
		}
	}
	if gf.Gates == nil {
		gf.Gates = map[string]StageGate{}
	}
	return gf, nil
}

func saveGates(missionDir string, gates GatesFile) error {
	p := filepath.Join(missionDir, "state", "gates.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(gates, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func satisfyCriterion(gates *GatesFile, stage string, substring string) (string, error) {
	sg, ok := gates.Gates[stage]
	if !ok {
		return "", fmt.Errorf("stage %q not found in gates", stage)
	}
	var matches []int
	for i, c := range sg.Criteria {
		if strings.Contains(c.Description, substring) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no criterion matching %q", substring)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous match for %q: %d criteria match", substring, len(matches))
	}
	sg.Criteria[matches[0]].Satisfied = true
	gates.Gates[stage] = sg
	return sg.Criteria[matches[0]].Description, nil
}

func initGateForStage(missionDir string, stage string) error {
	// Find mc-core binary
	var mcCorePath string
	if exePath, err := os.Executable(); err == nil {
		// Try same directory as mc binary (e.g. dist/mc-core alongside dist/mc)
		candidate := filepath.Join(filepath.Dir(exePath), "mc-core")
		if _, err := os.Stat(candidate); err == nil {
			mcCorePath = candidate
		}
	}
	if mcCorePath == "" {
		var err error
		mcCorePath, err = exec.LookPath("mc-core")
		if err != nil {
			return fmt.Errorf("mc-core not found: %w", err)
		}
	}

	cmd := exec.Command(mcCorePath, "check-gate", stage, "--mission-dir", missionDir)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("mc-core check-gate failed: %w", err)
	}

	var resp struct {
		Criteria []struct {
			Description string `json:"description"`
			Satisfied   bool   `json:"satisfied"`
		} `json:"criteria"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return fmt.Errorf("failed to parse mc-core output: %w", err)
	}

	gf, err := loadGates(missionDir)
	if err != nil {
		return err
	}

	var criteria []GateCriterion
	for _, c := range resp.Criteria {
		criteria = append(criteria, GateCriterion{Description: c.Description, Satisfied: c.Satisfied})
	}
	gf.Gates[stage] = StageGate{Criteria: criteria}

	return saveGates(missionDir, gf)
}

func allCriteriaMet(gates *GatesFile, stage string) bool {
	sg, ok := gates.Gates[stage]
	if !ok {
		return false
	}
	if len(sg.Criteria) == 0 {
		return false
	}
	for _, c := range sg.Criteria {
		if !c.Satisfied {
			return false
		}
	}
	return true
}

func init() {
	rootCmd.AddCommand(gateCmd)
	gateCmd.AddCommand(gateCheckCmd)
	gateCmd.AddCommand(gateApproveCmd)
	gateCmd.AddCommand(gateSatisfyCmd)
	gateCmd.AddCommand(gateStatusCmd)
	gateApproveCmd.Flags().String("note", "", "Reason for approving this gate (required)")
	gateSatisfyCmd.Flags().Bool("all", false, "Satisfy all criteria at once")
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

	// Read gates via compat loader
	gf, err := loadGates(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read gates: %w", err)
	}

	sg, ok := gf.Gates[stage]
	if !ok {
		return fmt.Errorf("gate not found: %s", stage)
	}
	// Convert to legacy Gate for downstream compat
	var criteriaStrings []string
	for _, c := range sg.Criteria {
		criteriaStrings = append(criteriaStrings, c.Description)
	}
	gate := Gate{Stage: stage, Status: "pending", Criteria: criteriaStrings}

	// Read tasks to calculate summary
	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	// Calculate task summary for this stage
	var summary TasksSummary
	for _, task := range tasks {
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

// runGateApproveWithNote is the internal helper for approving a gate with a note.
func runGateApproveWithNote(stage, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return fmt.Errorf("--note is required (explain why you're approving this gate)")
	}

	if !isValidStage(stage) {
		return fmt.Errorf("invalid stage: %s", stage)
	}

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	return doGateApprove(missionDir, stage, note)
}

func runGateApprove(cmd *cobra.Command, args []string) error {
	stage := args[0]

	var note string
	if cmd != nil {
		note, _ = cmd.Flags().GetString("note")
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return fmt.Errorf("--note is required (explain why you're approving this gate)")
	}

	if !isValidStage(stage) {
		return fmt.Errorf("invalid stage: %s", stage)
	}

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	return doGateApprove(missionDir, stage, note)
}

func doGateApprove(missionDir, stage, note string) error {
	if err := requireV6(missionDir); err != nil {
		return err
	}

	// Read current stage — only allow approving the gate for the CURRENT stage
	stagePath := filepath.Join(missionDir, "state", "stage.json")
	var currentStage StageState
	if err := readJSON(stagePath, &currentStage); err != nil {
		return fmt.Errorf("failed to read current stage: %w", err)
	}

	if currentStage.Current != stage {
		return fmt.Errorf("cannot approve gate for %q: current stage is %q (gate approval only allowed for the current stage)", stage, currentStage.Current)
	}

	// Update gate status — read via compat loader, write back as legacy format for other consumers
	gatesPath := filepath.Join(missionDir, "state", "gates.json")
	var gatesState GatesState
	if err := readJSON(gatesPath, &gatesState); err != nil {
		// Try loading via compat loader and convert
		gf, err2 := loadGates(missionDir)
		if err2 != nil {
			return fmt.Errorf("failed to read gates: %w", err)
		}
		gatesState.Gates = make(map[string]Gate)
		for name, sg := range gf.Gates {
			var cs []string
			for _, c := range sg.Criteria {
				cs = append(cs, c.Description)
			}
			gatesState.Gates[name] = Gate{Stage: name, Status: "pending", Criteria: cs}
		}
	}

	gate, ok := gatesState.Gates[stage]
	if !ok {
		return fmt.Errorf("gate not found: %s", stage)
	}

	// Prevent re-approving an already-approved gate (which would trigger duplicate transitions)
	if gate.Status == "approved" {
		return fmt.Errorf("gate for %q is already approved", stage)
	}

	gate.Status = "approved"
	gate.ApprovedAt = time.Now().UTC().Format(time.RFC3339)
	gate.ApprovalNote = note
	gatesState.Gates[stage] = gate

	if err := writeJSON(gatesPath, gatesState); err != nil {
		return fmt.Errorf("failed to update gate: %w", err)
	}

	writeAuditLog(missionDir, AuditGateApproved, "cli", map[string]interface{}{
		"stage": stage,
		"note":  note,
	})

	// Auto-commit gate approval
	gitAutoCommit(missionDir, CommitCategoryGate, fmt.Sprintf("approve %s", stage))

	// Auto-checkpoint on gate approval (G3.1)
	if cp, err := createCheckpoint(missionDir, ""); err == nil {
		fmt.Printf("Checkpoint created: %s\n", cp.ID)
	}

	// Transition to next stage — only ONE stage forward
	nextStage, err := getNextStage(stage)
	if err != nil {
		fmt.Printf("Gate approved: %s (final stage)\n", stage)
		return nil
	}

	stageState := StageState{
		Current:   nextStage,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := writeJSON(stagePath, stageState); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	writeAuditLog(missionDir, AuditStageAdvanced, "cli", map[string]interface{}{
		"from_stage": stage,
		"to_stage":   nextStage,
	})

	gitAutoCommit(missionDir, CommitCategoryStage, fmt.Sprintf("advance %s → %s (gate approved)", stage, nextStage))

	fmt.Printf("Gate approved: %s → %s\n", stage, nextStage)

	return nil
}

var gateSatisfyCmd = &cobra.Command{
	Use:   "satisfy [substring]",
	Short: "Satisfy a gate criterion by substring match",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		missionDir, err := findMissionDir()
		if err != nil {
			return err
		}
		gf, err := loadGates(missionDir)
		if err != nil {
			return err
		}
		stagePath := filepath.Join(missionDir, "state", "stage.json")
		var currentStage StageState
		if err := readJSON(stagePath, &currentStage); err != nil {
			return fmt.Errorf("failed to read current stage: %w", err)
		}
		stage := currentStage.Current

		satisfyAll, _ := cmd.Flags().GetBool("all")
		if satisfyAll {
			sg, ok := gf.Gates[stage]
			if !ok {
				return fmt.Errorf("no gate for stage %q", stage)
			}
			for i := range sg.Criteria {
				sg.Criteria[i].Satisfied = true
			}
			gf.Gates[stage] = sg
			fmt.Printf("All criteria for %s satisfied\n", stage)
		} else if len(args) > 0 {
			desc, err := satisfyCriterion(&gf, stage, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Satisfied: %s\n", desc)
		} else {
			return fmt.Errorf("provide a criterion substring or use --all")
		}
		return saveGates(missionDir, gf)
	},
}

var gateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show gate status for current stage",
	RunE: func(cmd *cobra.Command, args []string) error {
		missionDir, err := findMissionDir()
		if err != nil {
			return err
		}
		gf, err := loadGates(missionDir)
		if err != nil {
			return err
		}
		// Read current stage
		stagePath := filepath.Join(missionDir, "state", "stage.json")
		var currentStage struct {
			Current string `json:"current"`
		}
		if err := readJSON(stagePath, &currentStage); err != nil {
			return fmt.Errorf("failed to read current stage: %w", err)
		}
		stage := currentStage.Current
		sg, ok := gf.Gates[stage]
		if !ok {
			fmt.Printf("No gate criteria for stage: %s\n", stage)
			return nil
		}
		satisfied := 0
		total := len(sg.Criteria)
		fmt.Printf("\n── Gate: %s ─────────────────────\n", stage)
		for _, c := range sg.Criteria {
			mark := "✗"
			if c.Satisfied {
				mark = "✓"
				satisfied++
			}
			fmt.Printf("  %s %s\n", mark, c.Description)
		}
		fmt.Printf("\nStatus: %d/%d criteria met\n", satisfied, total)
		fmt.Println("────────────────────────────────────────")
		return nil
	},
}
