package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// briefingOutput is the expected structure of generateBriefing output.
type briefingOutput struct {
	TaskID                   string            `json:"task_id"`
	TaskName                 string            `json:"task_name"`
	Stage                    string            `json:"stage"`
	Zone                     string            `json:"zone"`
	Persona                  string            `json:"persona"`
	ScopePaths               []string          `json:"scope_paths"`
	Output                   string            `json:"output"`
	PredecessorFindingsPaths []string          `json:"predecessor_findings_paths,omitempty"`
	PredecessorSummaries     map[string]string `json:"predecessor_summaries,omitempty"`
}

func setupBriefingMissionDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	missionDir := filepath.Join(tmp, ".mission")
	for _, d := range []string{"state", "findings", "handoffs"} {
		if err := os.MkdirAll(filepath.Join(missionDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return missionDir
}

func TestGenerateBriefing_Basic(t *testing.T) {
	missionDir := setupBriefingMissionDir(t)

	tasks := []Task{
		{
			ID:         "abc123",
			Name:       "Implement widget",
			Stage:      "implement",
			Zone:       "backend",
			Persona:    "unit-tester",
			Status:     "pending",
			ScopePaths: []string{"pkg/widget.go", "pkg/widget_test.go"},
		},
	}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatal(err)
	}

	data, err := generateBriefing(missionDir, "abc123", "")
	if err != nil {
		t.Fatalf("generateBriefing returned error: %v", err)
	}

	var out briefingOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to parse briefing JSON: %v", err)
	}

	if out.TaskID != "abc123" {
		t.Errorf("task_id = %q, want %q", out.TaskID, "abc123")
	}
	if out.TaskName != "Implement widget" {
		t.Errorf("task_name = %q, want %q", out.TaskName, "Implement widget")
	}
	if out.Stage != "implement" {
		t.Errorf("stage = %q, want %q", out.Stage, "implement")
	}
	if out.Zone != "backend" {
		t.Errorf("zone = %q, want %q", out.Zone, "backend")
	}
	if out.Persona != "unit-tester" {
		t.Errorf("persona = %q, want %q", out.Persona, "unit-tester")
	}
	if len(out.ScopePaths) != 2 || out.ScopePaths[0] != "pkg/widget.go" {
		t.Errorf("scope_paths = %v, want [pkg/widget.go pkg/widget_test.go]", out.ScopePaths)
	}
	if out.Output == "" {
		t.Error("output path should not be empty")
	}
}

func TestGenerateBriefing_WithDeps(t *testing.T) {
	missionDir := setupBriefingMissionDir(t)

	// Create findings for task A
	findingsA := filepath.Join(missionDir, "findings", "aaa111.md")
	if err := os.WriteFile(findingsA, []byte("Task ID: aaa111\nStatus: complete\nSummary: Did the thing\n\nDetails here."), 0644); err != nil {
		t.Fatal(err)
	}

	tasks := []Task{
		{
			ID:      "aaa111",
			Name:    "Task A",
			Stage:   "implement",
			Zone:    "backend",
			Persona: "coder",
			Status:  "done",
		},
		{
			ID:        "bbb222",
			Name:      "Task B",
			Stage:     "implement",
			Zone:      "backend",
			Persona:   "coder",
			Status:    "pending",
			DependsOn: []string{"aaa111"},
		},
	}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatal(err)
	}

	data, err := generateBriefing(missionDir, "bbb222", "")
	if err != nil {
		t.Fatalf("generateBriefing returned error: %v", err)
	}

	var out briefingOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to parse briefing JSON: %v", err)
	}

	if len(out.PredecessorFindingsPaths) != 1 {
		t.Fatalf("predecessor_findings_paths length = %d, want 1", len(out.PredecessorFindingsPaths))
	}
	if out.PredecessorFindingsPaths[0] != ".mission/findings/aaa111.md" {
		t.Errorf("predecessor_findings_paths[0] = %q, want %q", out.PredecessorFindingsPaths[0], ".mission/findings/aaa111.md")
	}
}

func TestGenerateBriefing_WithPredecessorSummaries(t *testing.T) {
	missionDir := setupBriefingMissionDir(t)

	findingsA := filepath.Join(missionDir, "findings", "aaa111.md")
	if err := os.WriteFile(findingsA, []byte("Task ID: aaa111\nStatus: complete\nSummary: Did the thing\n\nDetails here."), 0644); err != nil {
		t.Fatal(err)
	}

	tasks := []Task{
		{
			ID:      "aaa111",
			Name:    "Task A",
			Stage:   "implement",
			Zone:    "backend",
			Persona: "coder",
			Status:  "done",
		},
		{
			ID:        "bbb222",
			Name:      "Task B",
			Stage:     "implement",
			Zone:      "backend",
			Persona:   "coder",
			Status:    "pending",
			DependsOn: []string{"aaa111"},
		},
	}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatal(err)
	}

	data, err := generateBriefing(missionDir, "bbb222", "")
	if err != nil {
		t.Fatalf("generateBriefing returned error: %v", err)
	}

	var out briefingOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to parse briefing JSON: %v", err)
	}

	if out.PredecessorSummaries == nil {
		t.Fatal("predecessor_summaries should not be nil")
	}
	summary, ok := out.PredecessorSummaries["aaa111"]
	if !ok {
		t.Fatal("predecessor_summaries missing key 'aaa111'")
	}
	if summary != "Did the thing" {
		t.Errorf("predecessor_summaries[aaa111] = %q, want %q", summary, "Did the thing")
	}
}

func TestGenerateBriefing_MissingDep(t *testing.T) {
	missionDir := setupBriefingMissionDir(t)

	tasks := []Task{
		{
			ID:      "aaa111",
			Name:    "Task A",
			Stage:   "implement",
			Zone:    "backend",
			Persona: "coder",
			Status:  "pending", // NOT complete
		},
		{
			ID:        "bbb222",
			Name:      "Task B",
			Stage:     "implement",
			Zone:      "backend",
			Persona:   "coder",
			Status:    "pending",
			DependsOn: []string{"aaa111"},
		},
	}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatal(err)
	}

	_, err := generateBriefing(missionDir, "bbb222", "")
	if err == nil {
		t.Fatal("expected error when dependency is not complete, got nil")
	}
}

func TestGenerateBriefing_NoFindings(t *testing.T) {
	missionDir := setupBriefingMissionDir(t)

	// Task A is complete but has no findings file
	tasks := []Task{
		{
			ID:      "aaa111",
			Name:    "Task A",
			Stage:   "implement",
			Zone:    "backend",
			Persona: "coder",
			Status:  "done",
		},
		{
			ID:        "bbb222",
			Name:      "Task B",
			Stage:     "implement",
			Zone:      "backend",
			Persona:   "coder",
			Status:    "pending",
			DependsOn: []string{"aaa111"},
		},
	}
	if err := saveTasks(missionDir, tasks); err != nil {
		t.Fatal(err)
	}

	// No findings file for aaa111 — should error or warn
	_, err := generateBriefing(missionDir, "bbb222", "")
	if err == nil {
		t.Fatal("expected error when predecessor findings file is missing, got nil")
	}
}
