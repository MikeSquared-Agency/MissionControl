package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkpointCmd)
	checkpointCmd.AddCommand(checkpointStatusCmd)
	checkpointCmd.AddCommand(checkpointHistoryCmd)
	checkpointCmd.AddCommand(checkpointQueryCmd)
	checkpointCmd.AddCommand(checkpointRestartCmd)

	checkpointRestartCmd.Flags().String("from", "", "Checkpoint ID to restart from")
}

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "Create a state checkpoint",
	Long: `Snapshot the current stage, gates, tasks, decisions, and blockers.
Writes to .mission/orchestrator/checkpoints/<timestamp>.json and auto-commits to git.

Subcommands:
  mc checkpoint            # Create a checkpoint
  mc checkpoint status     # Show session health
  mc checkpoint history    # List past sessions
  mc checkpoint query <id> # View a checkpoint
  mc checkpoint restart    # Restart session with briefing`,
	RunE: runCheckpointCreate,
}

var checkpointStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show session health and checkpoint status",
	RunE:  runCheckpointStatus,
}

var checkpointHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List past sessions",
	RunE:  runCheckpointHistory,
}

var checkpointQueryCmd = &cobra.Command{
	Use:   "query <checkpoint-id>",
	Short: "View a checkpoint",
	Args:  cobra.ExactArgs(1),
	RunE:  runCheckpointQuery,
}

var checkpointRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Create checkpoint and restart session with briefing",
	RunE:  runCheckpointRestart,
}

// CheckpointData is the JSON structure written to checkpoint files
type CheckpointData struct {
	ID        string          `json:"id"`
	Stage     string          `json:"stage"`
	CreatedAt string          `json:"created_at"`
	SessionID string          `json:"session_id,omitempty"`
	Tasks     []Task          `json:"tasks"`
	Gates     map[string]Gate `json:"gates"`
	Decisions []string        `json:"decisions"`
	Blockers  []string        `json:"blockers"`
	Summary   string          `json:"summary,omitempty"`
}

// SessionRecord is a line in sessions.jsonl
type SessionRecord struct {
	SessionID    string `json:"session_id"`
	StartedAt    string `json:"started_at"`
	EndedAt      string `json:"ended_at,omitempty"`
	CheckpointID string `json:"checkpoint_id"`
	Stage        string `json:"stage"`
	Reason       string `json:"reason,omitempty"`
}

// CheckpointStatusResult is the output of mc checkpoint status
type CheckpointStatusResult struct {
	SessionID      string `json:"session_id"`
	Stage          string `json:"stage"`
	SessionStart   string `json:"session_start"`
	DurationMin    int    `json:"duration_minutes"`
	LastCheckpoint string `json:"last_checkpoint,omitempty"`
	TasksTotal     int    `json:"tasks_total"`
	TasksComplete  int    `json:"tasks_complete"`
	Health         string `json:"health"` // green, yellow, red
	Recommendation string `json:"recommendation"`
}

func runCheckpointCreate(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	if err := requireV6(missionDir); err != nil {
		return err
	}

	cp, err := createCheckpoint(missionDir, "")
	if err != nil {
		return err
	}

	output, _ := json.MarshalIndent(cp, "", "  ")
	fmt.Println(string(output))

	return nil
}

func createCheckpoint(missionDir string, sessionID string) (*CheckpointData, error) {
	// Read current state
	var stageState StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err != nil {
		return nil, fmt.Errorf("failed to read stage: %w", err)
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks: %w", err)
	}
	tasksState := TasksState{Tasks: tasks}

	var gatesState GatesState
	if err := readJSON(filepath.Join(missionDir, "state", "gates.json"), &gatesState); err != nil {
		return nil, fmt.Errorf("failed to read gates: %w", err)
	}

	// Read decisions (from .mission/orchestrator/decisions.json if it exists)
	var decisions []string
	decisionsPath := filepath.Join(missionDir, "orchestrator", "decisions.json")
	if data, err := os.ReadFile(decisionsPath); err == nil {
		json.Unmarshal(data, &decisions)
	}

	// Read blockers (from .mission/orchestrator/blockers.json if it exists)
	var blockers []string
	blockersPath := filepath.Join(missionDir, "orchestrator", "blockers.json")
	if data, err := os.ReadFile(blockersPath); err == nil {
		json.Unmarshal(data, &blockers)
	}

	// Load or create session ID
	if sessionID == "" {
		sessionID = getCurrentSessionID(missionDir)
	}

	now := time.Now().UTC()
	cp := &CheckpointData{
		ID:        fmt.Sprintf("cp-%s", now.Format("20060102-150405")),
		Stage:     stageState.Current,
		CreatedAt: now.Format(time.RFC3339),
		SessionID: sessionID,
		Tasks:     tasksState.Tasks,
		Gates:     gatesState.Gates,
		Decisions: decisions,
		Blockers:  blockers,
	}

	// Write checkpoint file
	checkpointsDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
	os.MkdirAll(checkpointsDir, 0755)

	cpPath := filepath.Join(checkpointsDir, cp.ID+".json")
	if err := writeJSON(cpPath, cp); err != nil {
		return nil, fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// Update current.json pointer
	currentPath := filepath.Join(missionDir, "orchestrator", "current.json")
	writeJSON(currentPath, map[string]string{
		"checkpoint_id": cp.ID,
		"created_at":    cp.CreatedAt,
		"session_id":    sessionID,
	})

	writeAuditLog(missionDir, AuditCheckpointCreated, "cli", map[string]interface{}{
		"checkpoint_id": cp.ID,
		"stage":         cp.Stage,
		"session_id":    sessionID,
	})

	// Auto-commit to git
	gitAutoCommit(missionDir, CommitCategoryCheckpoint, fmt.Sprintf("checkpoint %s", cp.ID))

	return cp, nil
}

func getCurrentSessionID(missionDir string) string {
	currentPath := filepath.Join(missionDir, "orchestrator", "current.json")
	var current map[string]string
	if err := readJSON(currentPath, &current); err == nil {
		if sid, ok := current["session_id"]; ok && sid != "" {
			return sid
		}
	}
	return uuid.New().String()[:8]
}

func runCheckpointStatus(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	if err := requireV6(missionDir); err != nil {
		return err
	}

	// Read current state
	var stageState StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err != nil {
		return fmt.Errorf("failed to read stage: %w", err)
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks: %w", err)
	}
	tasksState := TasksState{Tasks: tasks}

	// Read current session info
	sessionID := getCurrentSessionID(missionDir)

	// Find last checkpoint
	lastCP := ""
	checkpointsDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
	entries, _ := os.ReadDir(checkpointsDir)
	if len(entries) > 0 {
		lastCP = strings.TrimSuffix(entries[len(entries)-1].Name(), ".json")
	}

	// Count tasks
	total := len(tasksState.Tasks)
	complete := 0
	for _, t := range tasksState.Tasks {
		if t.Status == "complete" {
			complete++
		}
	}

	// Read session start time from sessions.jsonl
	sessionStart := time.Now().UTC().Format(time.RFC3339) // default: now
	sessionsPath := filepath.Join(missionDir, "orchestrator", "sessions.jsonl")
	if data, err := os.ReadFile(sessionsPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			var rec SessionRecord
			if json.Unmarshal([]byte(lines[i]), &rec) == nil && rec.EndedAt == "" {
				sessionStart = rec.StartedAt
				break
			}
		}
	}

	// Calculate duration
	startTime, _ := time.Parse(time.RFC3339, sessionStart)
	durationMin := int(time.Since(startTime).Minutes())

	// Determine health
	health := "green"
	recommendation := "Session is healthy"
	if durationMin > 120 {
		health = "red"
		recommendation = "Session is long. Consider running 'mc checkpoint restart' to preserve context."
	} else if durationMin > 60 {
		health = "yellow"
		recommendation = "Session approaching limit. Consider checkpointing soon."
	}

	result := CheckpointStatusResult{
		SessionID:      sessionID,
		Stage:          stageState.Current,
		SessionStart:   sessionStart,
		DurationMin:    durationMin,
		LastCheckpoint: lastCP,
		TasksTotal:     total,
		TasksComplete:  complete,
		Health:         health,
		Recommendation: recommendation,
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))

	return nil
}

func runCheckpointHistory(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	sessionsPath := filepath.Join(missionDir, "orchestrator", "sessions.jsonl")
	data, err := os.ReadFile(sessionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("[]")
			return nil
		}
		return fmt.Errorf("failed to read sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var records []SessionRecord
	for _, line := range lines {
		if line == "" {
			continue
		}
		var rec SessionRecord
		if err := json.Unmarshal([]byte(line), &rec); err == nil {
			records = append(records, rec)
		}
	}

	output, _ := json.MarshalIndent(records, "", "  ")
	fmt.Println(string(output))

	return nil
}

func runCheckpointQuery(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	cpID := args[0]

	// Try to find checkpoint file
	cpPath := filepath.Join(missionDir, "orchestrator", "checkpoints", cpID+".json")
	if _, err := os.Stat(cpPath); os.IsNotExist(err) {
		// Try without prefix
		checkpointsDir := filepath.Join(missionDir, "orchestrator", "checkpoints")
		entries, _ := os.ReadDir(checkpointsDir)
		found := false
		for _, e := range entries {
			if strings.Contains(e.Name(), cpID) {
				cpPath = filepath.Join(checkpointsDir, e.Name())
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("checkpoint not found: %s", cpID)
		}
	}

	var cp CheckpointData
	if err := readJSON(cpPath, &cp); err != nil {
		return fmt.Errorf("failed to read checkpoint: %w", err)
	}

	output, _ := json.MarshalIndent(cp, "", "  ")
	fmt.Println(string(output))

	return nil
}

func runCheckpointRestart(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	if err := requireV6(missionDir); err != nil {
		return err
	}

	fromID, _ := cmd.Flags().GetString("from")

	// Create final checkpoint for current session
	oldSessionID := getCurrentSessionID(missionDir)
	cp, err := createCheckpoint(missionDir, oldSessionID)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}

	// If --from specified, load that checkpoint instead for the briefing
	briefingCP := cp
	if fromID != "" {
		cpPath := filepath.Join(missionDir, "orchestrator", "checkpoints", fromID+".json")
		var fromCP CheckpointData
		if err := readJSON(cpPath, &fromCP); err != nil {
			return fmt.Errorf("failed to read checkpoint %s: %w", fromID, err)
		}
		briefingCP = &fromCP
	}

	// Compile briefing using mc-core (if available)
	briefing := compileBriefing(missionDir, briefingCP)

	writeAuditLog(missionDir, AuditSessionEnded, "cli", map[string]interface{}{
		"session_id":    oldSessionID,
		"checkpoint_id": cp.ID,
	})

	// Log session end
	now := time.Now().UTC().Format(time.RFC3339)
	endRecord := SessionRecord{
		SessionID:    oldSessionID,
		EndedAt:      now,
		CheckpointID: cp.ID,
		Stage:        cp.Stage,
		Reason:       "restart",
	}
	appendSession(missionDir, endRecord)

	// Start new session
	newSessionID := uuid.New().String()[:8]
	startRecord := SessionRecord{
		SessionID:    newSessionID,
		StartedAt:    now,
		CheckpointID: cp.ID,
		Stage:        cp.Stage,
	}
	appendSession(missionDir, startRecord)

	writeAuditLog(missionDir, AuditSessionStarted, "cli", map[string]interface{}{
		"session_id":    newSessionID,
		"checkpoint_id": cp.ID,
		"stage":         cp.Stage,
	})

	// Update current.json with new session
	currentPath := filepath.Join(missionDir, "orchestrator", "current.json")
	writeJSON(currentPath, map[string]string{
		"checkpoint_id": cp.ID,
		"session_id":    newSessionID,
		"created_at":    now,
		"briefing":      briefing,
	})

	// Git commit the session transition
	gitAutoCommit(missionDir, CommitCategoryCheckpoint, fmt.Sprintf("session restart %s → %s", oldSessionID, newSessionID))

	result := map[string]string{
		"old_session":   oldSessionID,
		"new_session":   newSessionID,
		"checkpoint_id": cp.ID,
		"stage":         cp.Stage,
		"briefing":      briefing,
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))

	return nil
}

func compileBriefing(missionDir string, cp *CheckpointData) string {
	// Try to use mc-core checkpoint-compile if the binary is available
	// Write checkpoint to temp file, call mc-core, read output
	tmpFile := filepath.Join(missionDir, "orchestrator", ".tmp-checkpoint.json")
	if err := writeJSON(tmpFile, cp); err == nil {
		defer os.Remove(tmpFile)

		mcCore := exec.Command("mc-core", "checkpoint-compile", tmpFile)
		if output, err := mcCore.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}

	// Fallback: generate simple briefing in Go
	return generateFallbackBriefing(cp)
}

func generateFallbackBriefing(cp *CheckpointData) string {
	var b strings.Builder

	b.WriteString("# Session Briefing\n\n")
	b.WriteString(fmt.Sprintf("**Stage:** %s\n", cp.Stage))
	if cp.SessionID != "" {
		b.WriteString(fmt.Sprintf("**Previous Session:** %s\n", cp.SessionID))
	}
	b.WriteString("\n")

	if len(cp.Decisions) > 0 {
		b.WriteString("## Decisions\n")
		for _, d := range cp.Decisions {
			b.WriteString(fmt.Sprintf("- %s\n", d))
		}
		b.WriteString("\n")
	}

	// Task summary
	total := len(cp.Tasks)
	done := 0
	pending := 0
	for _, t := range cp.Tasks {
		switch t.Status {
		case "complete":
			done++
		case "pending":
			pending++
		}
	}
	b.WriteString("## Tasks\n")
	b.WriteString(fmt.Sprintf("- Total: %d, Done: %d, Pending: %d\n\n", total, done, pending))

	if len(cp.Blockers) > 0 {
		b.WriteString("## Blockers\n")
		for _, bl := range cp.Blockers {
			b.WriteString(fmt.Sprintf("- %s\n", bl))
		}
		b.WriteString("\n")
	}

	// Gate summary
	approved := []string{}
	pending_gates := []string{}
	for stage, gate := range cp.Gates {
		if gate.Status == "approved" {
			approved = append(approved, stage)
		} else {
			pending_gates = append(pending_gates, stage)
		}
	}
	sort.Strings(approved)
	sort.Strings(pending_gates)

	if len(approved) > 0 {
		b.WriteString(fmt.Sprintf("## Gates Approved\n%s\n\n", strings.Join(approved, ", ")))
	}

	return b.String()
}

func appendSession(missionDir string, record SessionRecord) {
	sessionsPath := filepath.Join(missionDir, "orchestrator", "sessions.jsonl")
	data, _ := json.Marshal(record)
	f, err := os.OpenFile(sessionsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(data)
	f.WriteString("\n")
}
