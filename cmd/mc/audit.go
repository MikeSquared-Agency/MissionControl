package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp string                 `json:"timestamp"`
	Action    string                 `json:"action"`
	Actor     string                 `json:"actor"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Audit action type constants
const (
	AuditTaskCreated        = "task_created"
	AuditTaskUpdated        = "task_updated"
	AuditTaskCompleted      = "task_completed"
	AuditGateApproved       = "gate_approved"
	AuditGateChecked        = "gate_checked"
	AuditStageAdvanced      = "stage_advanced"
	AuditStageSet           = "stage_set"
	AuditWorkerSpawned      = "worker_spawned"
	AuditWorkerCompleted    = "worker_completed"
	AuditWorkerKilled       = "worker_killed"
	AuditCheckpointCreated  = "checkpoint_created"
	AuditSessionStarted     = "session_started"
	AuditSessionEnded       = "session_ended"
	AuditHandoffReceived    = "handoff_received"
	AuditProjectInitialized = "project_initialized"
)

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditFilterCmd)

	auditListCmd.Flags().IntP("last", "n", 20, "Number of entries to show")
	auditListCmd.Flags().Bool("json", false, "Output raw JSON lines")

	auditFilterCmd.Flags().StringP("action", "a", "", "Filter by action type")
	auditFilterCmd.Flags().String("actor", "", "Filter by actor")
	auditFilterCmd.Flags().String("since", "", "Show entries since (RFC3339 or duration like 1h, 24h)")
	auditFilterCmd.Flags().IntP("last", "n", 50, "Max entries to show")
	auditFilterCmd.Flags().Bool("json", false, "Output raw JSON lines")
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View the audit trail",
	Long: `View and query the MissionControl audit trail.

The audit trail records all significant state mutations:
  task_created, task_updated, task_completed,
  gate_approved, gate_checked, stage_advanced, stage_set,
  worker_spawned, worker_completed, worker_killed,
  checkpoint_created, session_started, session_ended,
  handoff_received, project_initialized

Examples:
  mc audit                           # Show last 20 entries
  mc audit list -n 50                # Show last 50 entries
  mc audit filter -a gate_approved   # Show gate approvals
  mc audit filter --since 1h         # Last hour's activity
  mc audit filter --actor cli        # Actions by CLI user`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show last 20
		return runAuditList(cmd, args)
	},
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent audit entries",
	RunE:  runAuditList,
}

var auditFilterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Filter audit entries",
	RunE:  runAuditFilter,
}

func runAuditList(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	n := 20
	if cmd.Flags().Changed("last") {
		n, _ = cmd.Flags().GetInt("last")
		if n <= 0 {
			n = 20
		}
	}
	jsonOutput, _ := cmd.Flags().GetBool("json")

	entries, err := readAuditLog(missionDir)
	if err != nil {
		return err
	}

	// Take last N
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}

	return printAuditEntries(entries, jsonOutput)
}

func runAuditFilter(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	actionFilter, _ := cmd.Flags().GetString("action")
	actorFilter, _ := cmd.Flags().GetString("actor")
	sinceStr, _ := cmd.Flags().GetString("since")
	n, _ := cmd.Flags().GetInt("last")
	if n <= 0 {
		n = 20
	}
	jsonOutput, _ := cmd.Flags().GetBool("json")

	var sinceTime time.Time
	if sinceStr != "" {
		// Try duration first (e.g., "1h", "24h")
		if d, err := time.ParseDuration(sinceStr); err == nil {
			sinceTime = time.Now().UTC().Add(-d)
		} else if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			sinceTime = t
		} else {
			return fmt.Errorf("invalid --since value: %s (use duration like 1h or RFC3339)", sinceStr)
		}
	}

	entries, err := readAuditLog(missionDir)
	if err != nil {
		return err
	}

	// Apply filters
	var filtered []AuditEntry
	for _, e := range entries {
		if actionFilter != "" && e.Action != actionFilter {
			continue
		}
		if actorFilter != "" && !strings.Contains(e.Actor, actorFilter) {
			continue
		}
		if !sinceTime.IsZero() {
			if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
				if t.Before(sinceTime) {
					continue
				}
			}
		}
		filtered = append(filtered, e)
	}

	// Take last N
	if len(filtered) > n {
		filtered = filtered[len(filtered)-n:]
	}

	return printAuditEntries(filtered, jsonOutput)
}

func printAuditEntries(entries []AuditEntry, jsonOutput bool) error {
	if len(entries) == 0 {
		if jsonOutput {
			return nil
		}
		fmt.Println("No audit entries found.")
		return nil
	}

	if jsonOutput {
		for _, e := range entries {
			data, _ := json.Marshal(e)
			fmt.Println(string(data))
		}
		return nil
	}

	// Pretty print
	for _, e := range entries {
		ts := e.Timestamp
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			ts = t.Format("2006-01-02 15:04:05")
		}

		detailStr := ""
		if len(e.Details) > 0 {
			parts := make([]string, 0, len(e.Details))
			for k, v := range e.Details {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			detailStr = " " + strings.Join(parts, " ")
		}

		fmt.Printf("[%s] %-24s actor=%-10s%s\n", ts, e.Action, e.Actor, detailStr)
	}

	return nil
}

// writeAuditLog appends an entry to .mission/audit.jsonl
func writeAuditLog(missionDir string, action string, actor string, details map[string]interface{}) {
	entry := AuditEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Actor:     actor,
		Details:   details,
	}

	auditPath := filepath.Join(missionDir, "audit.jsonl")
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to marshal audit entry: %v\n", err)
		return
	}

	f, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to open audit log: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write audit entry: %v\n", err)
	}
	f.WriteString("\n")
}

// readAuditLog reads all entries from .mission/audit.jsonl
func readAuditLog(missionDir string) ([]AuditEntry, error) {
	auditPath := filepath.Join(missionDir, "audit.jsonl")
	data, err := os.ReadFile(auditPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	var entries []AuditEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e AuditEntry
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			entries = append(entries, e)
		}
	}

	return entries, nil
}
