package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupStrictTestDir creates a minimal mission dir with stage and tasks for strict tests.
func setupStrictTestDir(t *testing.T, stage string, tasks []Task) string {
	t.Helper()
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	stageData, _ := json.Marshal(StageState{Current: stage})
	if err := os.WriteFile(filepath.Join(stateDir, "stage.json"), stageData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write tasks as JSONL
	var lines []string
	for _, task := range tasks {
		b, _ := json.Marshal(task)
		lines = append(lines, string(b))
	}
	if err := os.WriteFile(filepath.Join(stateDir, "tasks.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return tmp
}

// --- checkVerifyPersonaCoverage tests ---

func TestStrict_VerifyPersonaCoverage_AllPresent(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "verify", Persona: "reviewer", Status: "done"},
		{ID: "t2", Stage: "verify", Persona: "security", Status: "done"},
		{ID: "t3", Stage: "verify", Persona: "tester", Status: "done"},
	}
	failures := checkVerifyPersonaCoverage(tasks)
	if len(failures) != 0 {
		t.Errorf("expected no failures, got: %v", failures)
	}
}

func TestStrict_VerifyPersonaCoverage_MissingReviewer(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "verify", Persona: "security", Status: "done"},
		{ID: "t2", Stage: "verify", Persona: "tester", Status: "done"},
	}
	failures := checkVerifyPersonaCoverage(tasks)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
	if !strings.Contains(failures[0], "reviewer") {
		t.Errorf("expected failure about reviewer, got: %s", failures[0])
	}
}

func TestStrict_VerifyPersonaCoverage_MissingSecurity(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "verify", Persona: "reviewer", Status: "done"},
		{ID: "t2", Stage: "verify", Persona: "tester", Status: "done"},
	}
	failures := checkVerifyPersonaCoverage(tasks)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
	if !strings.Contains(failures[0], "security") {
		t.Errorf("expected failure about security, got: %s", failures[0])
	}
}

func TestStrict_VerifyPersonaCoverage_NotDone(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "verify", Persona: "reviewer", Status: "done"},
		{ID: "t2", Stage: "verify", Persona: "security", Status: "in_progress"},
		{ID: "t3", Stage: "verify", Persona: "tester", Status: "done"},
	}
	failures := checkVerifyPersonaCoverage(tasks)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
	if !strings.Contains(failures[0], "security") {
		t.Errorf("expected failure about security, got: %s", failures[0])
	}
}

func TestStrict_VerifyPersonaCoverage_NoTasks(t *testing.T) {
	failures := checkVerifyPersonaCoverage(nil)
	if len(failures) != 3 {
		t.Errorf("expected 3 failures for empty task list, got %d: %v", len(failures), failures)
	}
}

// --- checkIntegratorPresent tests ---

func TestStrict_IntegratorPresent_SingleTask(t *testing.T) {
	// Single task — integrator not needed (validateStrict skips the check)
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
	}
	dir := setupStrictTestDir(t, "implement", tasks)
	failures := validateStrict(dir)
	if len(failures) != 0 {
		t.Errorf("single implement task should not require integrator, got: %v", failures)
	}
}

func TestStrict_IntegratorPresent_MultiNoIntegrator(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t2", Stage: "implement", Persona: "developer", Status: "done"},
	}
	failures := checkIntegratorPresent(tasks)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
	if !strings.Contains(failures[0], "integrator") {
		t.Errorf("expected failure about integrator, got: %s", failures[0])
	}
}

func TestStrict_IntegratorPresent_MultiWithDoneIntegrator(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t2", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t3", Stage: "implement", Persona: "integrator", Status: "done"},
	}
	failures := checkIntegratorPresent(tasks)
	if len(failures) != 0 {
		t.Errorf("expected no failures with integrator present, got: %v", failures)
	}
}

func TestStrict_IntegratorPresent_MultiWithPendingIntegrator(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t2", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t3", Stage: "implement", Persona: "integrator", Status: "pending"},
	}
	failures := checkIntegratorPresent(tasks)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for non-done integrator, got %d: %v", len(failures), failures)
	}
	if !strings.Contains(failures[0], "not done") {
		t.Errorf("expected failure about integrator not done, got: %s", failures[0])
	}
}

func TestStrict_IntegratorPresent_MultiWithInProgressIntegrator(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t2", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t3", Stage: "implement", Persona: "integrator", Status: "in_progress"},
	}
	failures := checkIntegratorPresent(tasks)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for in_progress integrator, got %d: %v", len(failures), failures)
	}
}

// --- validateStrict integration tests ---

func TestStrict_VerifyStage_FullPass(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "verify", Persona: "reviewer", Status: "done"},
		{ID: "t2", Stage: "verify", Persona: "security", Status: "done"},
		{ID: "t3", Stage: "verify", Persona: "tester", Status: "done"},
	}
	dir := setupStrictTestDir(t, "verify", tasks)
	failures := validateStrict(dir)
	if len(failures) != 0 {
		t.Errorf("expected pass, got: %v", failures)
	}
}

func TestStrict_VerifyStage_OnlyFiltersCurrentStage(t *testing.T) {
	// Tasks exist but in wrong stage — should fail
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "reviewer", Status: "done"},
		{ID: "t2", Stage: "implement", Persona: "security", Status: "done"},
		{ID: "t3", Stage: "implement", Persona: "tester", Status: "done"},
	}
	dir := setupStrictTestDir(t, "verify", tasks)
	failures := validateStrict(dir)
	if len(failures) != 3 {
		t.Errorf("expected 3 failures (tasks in wrong stage), got %d: %v", len(failures), failures)
	}
}

func TestStrict_ImplementStage_NoChecksForSingleTask(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
	}
	dir := setupStrictTestDir(t, "implement", tasks)
	failures := validateStrict(dir)
	if len(failures) != 0 {
		t.Errorf("expected no failures for single implement task, got: %v", failures)
	}
}

func TestStrict_ImplementStage_MultiNeedsIntegrator(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "done"},
		{ID: "t2", Stage: "implement", Persona: "developer", Status: "done"},
	}
	dir := setupStrictTestDir(t, "implement", tasks)
	failures := validateStrict(dir)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}
}

func TestStrict_OtherStage_NoChecks(t *testing.T) {
	// A stage like "design" should have no strict checks
	tasks := []Task{
		{ID: "t1", Stage: "design", Persona: "architect", Status: "done"},
	}
	dir := setupStrictTestDir(t, "design", tasks)
	failures := validateStrict(dir)
	if len(failures) != 0 {
		t.Errorf("expected no strict checks for design stage, got: %v", failures)
	}
}

func TestStrict_NoTasks_VerifyStage(t *testing.T) {
	dir := setupStrictTestDir(t, "verify", nil)
	failures := validateStrict(dir)
	// No stage tasks means no personas found — should fail
	if len(failures) != 3 {
		t.Errorf("expected 3 failures for empty verify stage, got %d: %v", len(failures), failures)
	}
}
