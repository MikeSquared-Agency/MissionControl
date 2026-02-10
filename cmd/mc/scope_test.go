package main

import (
	"os"
	"path/filepath"
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

func TestScopeValidateScope_EmptyScopePathsRestricted(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress"},
	})
	// Empty scope_paths should restrict to .mission/ only
	errs := validateScope(dir, "t1", []string{"anywhere/file.go"})
	if len(errs) == 0 {
		t.Error("empty scope_paths should reject non-.mission/ files")
	}
	// .mission/ files should still be allowed
	errs = validateScope(dir, "t1", []string{".mission/state/stage.json"})
	if len(errs) != 0 {
		t.Errorf("empty scope_paths should allow .mission/ files, got: %v", errs)
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

// --- Phase 5: Empty scope, exempt paths, edge cases ---

func TestScopeEmptyScopeOnlyMission(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: nil},
	})

	// Non-.mission files should be rejected
	errs := validateScope(dir, "t1", []string{"src/main.go", "README.md"})
	if len(errs) == 0 {
		t.Fatal("empty scope_paths should reject non-.mission/ files")
	}
	if !scopeContains(errs[0], "2 file(s) outside") {
		t.Errorf("expected 2 files outside scope, got: %v", errs)
	}

	// .mission/ files should pass
	errs = validateScope(dir, "t1", []string{".mission/state/tasks.jsonl", ".mission/findings/t1.md"})
	if len(errs) != 0 {
		t.Errorf("empty scope should allow .mission/ files, got: %v", errs)
	}

	// Mix: only non-.mission files should be rejected
	errs = validateScope(dir, "t1", []string{".mission/state/stage.json", "pkg/lib.go"})
	if len(errs) == 0 {
		t.Fatal("expected error for pkg/lib.go with empty scope")
	}
	found := false
	for _, e := range errs {
		if scopeContains(e, "pkg/lib.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected pkg/lib.go in errors, got: %v", errs)
	}
}

func TestScopeExemptPaths(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: []string{"cmd/mc/"}},
	})

	// Write config.json with exempt paths
	configData := `{"scope_exempt_paths": ["AGENTS.md", "docs/"]}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(configData), 0o644); err != nil {
		t.Fatal(err)
	}

	// Exempt exact file should pass
	errs := validateScope(dir, "t1", []string{"cmd/mc/main.go", "AGENTS.md"})
	if len(errs) != 0 {
		t.Errorf("AGENTS.md should be exempt, got: %v", errs)
	}

	// Exempt directory prefix should pass
	errs = validateScope(dir, "t1", []string{"docs/README.md"})
	if len(errs) != 0 {
		t.Errorf("docs/ should be exempt, got: %v", errs)
	}

	// Non-exempt, non-scope file should fail
	errs = validateScope(dir, "t1", []string{"pkg/other.go"})
	if len(errs) == 0 {
		t.Error("pkg/other.go should not be exempt or in scope")
	}
}

func TestScopeExemptPathsMissing(t *testing.T) {
	dir := setupStrictTestDir(t, "implement", []Task{
		{ID: "t1", Stage: "implement", Persona: "developer", Status: "in_progress", ScopePaths: []string{"cmd/mc/"}},
	})

	// No config.json at all — should work fine
	errs := validateScope(dir, "t1", []string{"cmd/mc/main.go"})
	if len(errs) != 0 {
		t.Errorf("missing config.json should not cause errors, got: %v", errs)
	}

	// Out of scope still fails
	errs = validateScope(dir, "t1", []string{"pkg/other.go"})
	if len(errs) == 0 {
		t.Error("out of scope should still fail without config.json")
	}

	// Config exists but no exempt key
	configData := `{"some_other_key": true}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(configData), 0o644); err != nil {
		t.Fatal(err)
	}
	errs = validateScope(dir, "t1", []string{"cmd/mc/main.go"})
	if len(errs) != 0 {
		t.Errorf("config without exempt key should not cause errors, got: %v", errs)
	}
}

func TestMatchesScopePath_EdgeCases(t *testing.T) {
	// Empty pattern
	if matchesScopePath("file.go", "") {
		t.Error("empty pattern should not match")
	}

	// Root-level file with directory pattern
	if matchesScopePath("file.go", "cmd/") {
		t.Error("root file should not match cmd/ prefix")
	}

	// Deeply nested file
	if !matchesScopePath("a/b/c/d/e.go", "a/b/c/") {
		t.Error("deeply nested file should match directory prefix")
	}

	// Pattern that looks like directory but file has same prefix
	if matchesScopePath("cmd-tool/main.go", "cmd/") {
		t.Error("cmd-tool should not match cmd/ prefix")
	}

	// Exact match with path
	if !matchesScopePath("Makefile", "Makefile") {
		t.Error("exact match for root file should work")
	}
}
