package main

import (
	"testing"
)

// --- matchesScopePath tests ---

func TestScopeMatchesScopePath_ExactMatch(t *testing.T) {
	if !matchesScopePath("cmd/mc/scope.go", "cmd/mc/scope.go") {
		t.Error("exact match should return true")
	}
}

func TestScopeMatchesScopePath_DirectoryPrefix(t *testing.T) {
	if !matchesScopePath("cmd/mc/scope.go", "cmd/mc/") {
		t.Error("directory prefix should match")
	}
	if matchesScopePath("pkg/other.go", "cmd/mc/") {
		t.Error("non-matching directory should not match")
	}
}

func TestScopeMatchesScopePath_Glob(t *testing.T) {
	if !matchesScopePath("cmd/mc/scope.go", "cmd/mc/*.go") {
		t.Error("glob should match")
	}
	if matchesScopePath("cmd/mc/scope.go", "cmd/mc/*.txt") {
		t.Error("non-matching glob should not match")
	}
}

func TestScopeMatchesScopePath_NoMatch(t *testing.T) {
	if matchesScopePath("pkg/util.go", "cmd/mc/scope.go") {
		t.Error("unrelated file should not match")
	}
}

func TestScopeMatchesScopePath_BasenameFallback(t *testing.T) {
	// Pattern without "/" should match against basename
	if !matchesScopePath("cmd/mc/scope.go", "*.go") {
		t.Error("basename glob should match")
	}
	if matchesScopePath("cmd/mc/scope.go", "*.txt") {
		t.Error("non-matching basename glob should not match")
	}
}

// --- validateScope tests ---

func TestScopeValidateScope_AllInScope(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: []string{"cmd/mc/"}},
	})
	errs := validateScope(dir, "t1", []string{"cmd/mc/scope.go", "cmd/mc/commit.go"})
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestScopeValidateScope_SomeOutOfScope(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: []string{"cmd/mc/"}},
	})
	errs := validateScope(dir, "t1", []string{"cmd/mc/scope.go", "pkg/util.go"})
	if len(errs) == 0 {
		t.Error("expected errors for out-of-scope file")
	}
	found := false
	for _, e := range errs {
		if scopeContains(e, "pkg/util.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected pkg/util.go in errors, got: %v", errs)
	}
}

func TestScopeValidateScope_MissionAutoAllowed(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: []string{"cmd/mc/"}},
	})
	errs := validateScope(dir, "t1", []string{".mission/state/tasks.jsonl", "cmd/mc/scope.go"})
	if len(errs) != 0 {
		t.Errorf(".mission/ files should be auto-allowed, got: %v", errs)
	}
}

func TestScopeValidateScope_EmptyScopePathsSkipped(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress"},
	})
	errs := validateScope(dir, "t1", []string{"anywhere/file.go"})
	if len(errs) != 0 {
		t.Errorf("empty scope_paths should skip validation, got: %v", errs)
	}
}

func TestScopeValidateScope_TaskNotFound(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress"},
	})
	errs := validateScope(dir, "nonexistent", []string{"file.go"})
	if len(errs) == 0 {
		t.Error("expected error for missing task")
	}
	if !scopeContains(errs[0], "not found") {
		t.Errorf("expected 'not found' error, got: %s", errs[0])
	}
}

// --- findTaskByID tests ---

func TestFindTaskByID_ExactMatch(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "abc123", Stage: "implement", Persona: "developer"},
		{ID: "abc456", Stage: "implement", Persona: "tester"},
	})
	task, err := findTaskByID(dir, "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != "abc123" {
		t.Errorf("expected abc123, got %s", task.ID)
	}
}

func TestFindTaskByID_PrefixMatch(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "abc123", Stage: "implement", Persona: "developer"},
		{ID: "def456", Stage: "implement", Persona: "tester"},
	})
	task, err := findTaskByID(dir, "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != "abc123" {
		t.Errorf("expected abc123, got %s", task.ID)
	}
}

func TestFindTaskByID_AmbiguousPrefix(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "abc123", Stage: "implement", Persona: "developer"},
		{ID: "abc456", Stage: "implement", Persona: "tester"},
	})
	_, err := findTaskByID(dir, "abc")
	if err == nil {
		t.Fatal("expected error for ambiguous prefix")
	}
	if !scopeContains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' error, got: %v", err)
	}
}

func TestFindTaskByID_NotFound(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "abc123", Stage: "implement", Persona: "developer"},
	})
	_, err := findTaskByID(dir, "zzz")
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !scopeContains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// helper
func scopeContains(s, substr string) bool {
	return len(s) >= len(substr) && scopeSearchSubstr(s, substr)
}

func scopeSearchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestScopeValidateScope_PrefixIDMatch(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "abc123def", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: []string{"cmd/mc/"}},
	})
	errs := validateScope(dir, "abc123", []string{"cmd/mc/scope.go"})
	if len(errs) != 0 {
		t.Errorf("expected no errors with prefix ID match, got: %v", errs)
	}
}
