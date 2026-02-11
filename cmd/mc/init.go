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

// CLI flags for init command
var (
	initPath     string
	initGit      bool
	initOpenClaw bool
	initConfig   string
	initAutoMode bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initPath, "path", "", "Project path (default: current directory)")
	initCmd.Flags().BoolVar(&initGit, "git", false, "Initialize git repository")
	initCmd.Flags().BoolVar(&initOpenClaw, "openclaw", true, "Enable OpenClaw mode")
	initCmd.Flags().StringVar(&initConfig, "config", "", "Path to JSON config file with workflow matrix")
	initCmd.Flags().BoolVar(&initAutoMode, "auto-mode", false, "Enable automatic gate approval")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .mission directory",
	Long:  `Creates the .mission/ directory structure for MissionControl orchestration.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workDir := initPath
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Expand ~ to home directory
	if strings.HasPrefix(workDir, "~") {
		home, _ := os.UserHomeDir()
		workDir = filepath.Join(home, workDir[1:])
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	missionDir := filepath.Join(workDir, ".mission")

	// Check if already exists
	if _, err := os.Stat(missionDir); err == nil {
		return fmt.Errorf(".mission/ already exists")
	}

	// Create directory structure
	dirs := []string{
		"state",
		"specs",
		"findings",
		"handoffs",
		"checkpoints",
		"prompts",
		"orchestrator",
		"orchestrator/checkpoints",
	}

	for _, dir := range dirs {
		path := filepath.Join(missionDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	// Create initial state files
	if err := writeJSON(filepath.Join(missionDir, "state", "stage.json"), StageState{
		Current:   "discovery",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	if err := writeTasksJSONL(filepath.Join(missionDir, "state", "tasks.jsonl"), []Task{}); err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(missionDir, "state", "workers.json"), WorkersState{
		Workers: []Worker{},
	}); err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(missionDir, "state", "gates.json"), GatesState{
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
		return err
	}

	// Load matrix config if provided
	var matrixConfig map[string]interface{}
	if initConfig != "" {
		data, err := os.ReadFile(initConfig)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(data, &matrixConfig); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Create config.json with optional matrix
	config := Config{
		Version:  "1.0.0",
		Audience: "personal",
		Zones:    []string{"frontend", "backend", "database", "infra", "shared"},
		OpenClaw: initOpenClaw,
	}

	if initAutoMode {
		config.AutoMode = true
	}

	// If matrix provided, include it in config
	if matrix, ok := matrixConfig["matrix"]; ok {
		config.Matrix = matrix
	}

	if err := writeJSON(filepath.Join(missionDir, "config.json"), config); err != nil {
		return err
	}

	// Create CLAUDE.md (OpenClaw prompt)
	if err := os.WriteFile(filepath.Join(missionDir, "CLAUDE.md"), []byte(openClawPrompt), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// Create worker prompts
	prompts := map[string]string{
		"researcher.md":            researcherPrompt,
		"analyst.md":               analystPrompt,
		"requirements-engineer.md": requirementsEngineerPrompt,
		"designer.md":              designerPrompt,
		"architect.md":             architectPrompt,
		"developer.md":             developerPrompt,
		"reviewer.md":              reviewerPrompt,
		"security.md":              securityPrompt,
		"tester.md":                testerPrompt,
		"qa.md":                    qaPrompt,
		"docs.md":                  docsPrompt,
		"devops.md":                devopsPrompt,
		"debugger.md":              debuggerPrompt,
	}

	for name, content := range prompts {
		path := filepath.Join(missionDir, "prompts", name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	// Initialize git if requested
	if initGit {
		gitDir := filepath.Join(workDir, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			gitCmd := exec.Command("git", "init")
			gitCmd.Dir = workDir
			if output, err := gitCmd.CombinedOutput(); err != nil {
				fmt.Printf("Warning: git init failed: %v\n%s\n", err, output)
			} else {
				fmt.Println("Initialized git repository")
			}
		}
	}

	writeAuditLog(missionDir, AuditProjectInitialized, "cli", map[string]interface{}{
		"path":     workDir,
		"openclaw": initOpenClaw,
	})

	fmt.Printf("Initialized .mission/ directory at %s\n", workDir)
	fmt.Println("")
	fmt.Println("Created:")
	fmt.Println("  .mission/CLAUDE.md           # OpenClaw system prompt")
	fmt.Println("  .mission/config.json         # Project settings")
	fmt.Println("  .mission/state/              # Runtime state")
	fmt.Println("  .mission/specs/              # Feature specifications")
	fmt.Println("  .mission/findings/           # Worker findings")
	fmt.Println("  .mission/handoffs/           # Raw handoff records")
	fmt.Println("  .mission/checkpoints/        # State checkpoints")
	fmt.Println("  .mission/prompts/            # Worker system prompts")
	fmt.Println("  .mission/orchestrator/       # Orchestrator state")
	fmt.Println("")
	if initOpenClaw {
		fmt.Println("Next: Run 'claude' in this directory to start OpenClaw")
	} else {
		fmt.Println("OpenClaw mode disabled. Run individual agents with 'mc spawn'")
	}

	return nil
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// State types

type StageState struct {
	Current   string `json:"current"`
	UpdatedAt string `json:"updated_at"`
}

type Task struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Stage      string   `json:"stage"`
	Zone       string   `json:"zone"`
	Persona    string   `json:"persona"`
	Status     string   `json:"status"` // pending, in_progress, complete, blocked
	DependsOn  []string `json:"depends_on,omitempty"`
	ScopePaths []string `json:"scope_paths,omitempty"`
	WorkerID   string   `json:"worker_id,omitempty"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type TasksState struct {
	Tasks []Task `json:"tasks"`
}

type Worker struct {
	ID        string `json:"id"`
	Persona   string `json:"persona"`
	TaskID    string `json:"task_id"`
	Zone      string `json:"zone"`
	Status    string `json:"status"` // running, complete, failed
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

type WorkersState struct {
	Workers []Worker `json:"workers"`
}

type Gate struct {
	Stage        string   `json:"stage"`
	Status       string   `json:"status"` // pending, ready, approved
	Criteria     []string `json:"criteria"`
	ApprovedAt   string   `json:"approved_at,omitempty"`
	ApprovalNote string   `json:"approval_note,omitempty"`
}

type GatesState struct {
	Gates map[string]Gate `json:"gates"`
}

type Team struct {
	Personas []string `json:"personas"`
	Zone     string   `json:"zone,omitempty"`
}

type Config struct {
	Version        string            `json:"version"`
	Audience       string            `json:"audience"` // personal, external
	Zones          []string          `json:"zones"`
	OpenClaw       bool              `json:"openclaw"`
	Matrix         interface{}       `json:"matrix,omitempty"`
	AutoCommit     *AutoCommitConfig `json:"auto_commit,omitempty"`
	TokenThreshold int               `json:"token_threshold,omitempty"`
	Teams          map[string]Team   `json:"teams,omitempty"`
	AutoMode       bool              `json:"auto_mode,omitempty"`
}

const defaultTokenThreshold = 150000

// getTokenThreshold returns the configured token threshold or the default (150k).
func getTokenThreshold(missionDir string) int {
	configPath := filepath.Join(missionDir, "config.json")
	var cfg Config
	if err := readJSON(configPath, &cfg); err == nil && cfg.TokenThreshold > 0 {
		return cfg.TokenThreshold
	}
	return defaultTokenThreshold
}
