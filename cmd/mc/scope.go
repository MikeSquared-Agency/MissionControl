package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// getStagedFiles returns the list of staged file paths (relative to repo root).
func getStagedFiles(repoDir string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached failed: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// matchesScopePath checks if a file matches a single scope pattern.
// Three modes: directory prefix (pattern ends with /), glob, or exact match.
func matchesScopePath(file, pattern string) bool {
	// Directory prefix
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(file, pattern)
	}

	// Exact match
	if file == pattern {
		return true
	}

	// Glob
	if matched, err := filepath.Match(pattern, file); err == nil && matched {
		return true
	}
	// For patterns without /, also try matching just the filename
	if !strings.Contains(pattern, "/") {
		if matched, err := filepath.Match(pattern, filepath.Base(file)); err == nil && matched {
			return true
		}
	}

	return false
}

// loadScopeExemptPaths reads .mission/config.json and returns the scope_exempt_paths
// array. Returns an empty slice if the file is unreadable or the key is missing.
func loadScopeExemptPaths(missionDir string) []string {
	configPath := filepath.Join(missionDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	raw, ok := cfg["scope_exempt_paths"]
	if !ok {
		return nil
	}

	var paths []string
	if err := json.Unmarshal(raw, &paths); err != nil {
		return nil
	}
	return paths
}

// validateScope checks that all staged files fall within a task's scope_paths.
// Returns a list of error strings for out-of-scope files, or nil if everything is fine.
// If the task has empty/nil scope_paths, only .mission/ files are allowed.
// Files matching scope_exempt_paths from .mission/config.json are always allowed.
func validateScope(missionDir, taskID string, stagedFiles []string) []string {
	found, err := findTaskByID(missionDir, taskID)
	if err != nil {
		return []string{err.Error()}
	}
	task := &found

	exemptPaths := loadScopeExemptPaths(missionDir)

	var outOfScope []string
	for _, file := range stagedFiles {
		if strings.HasPrefix(file, ".mission/") {
			continue
		}

		// Check exempt paths first
		exempt := false
		for _, pattern := range exemptPaths {
			if matchesScopePath(file, pattern) {
				exempt = true
				break
			}
		}
		if exempt {
			continue
		}

		// If task has no scope_paths, only .mission/ files are allowed (already handled above)
		if len(task.ScopePaths) == 0 {
			outOfScope = append(outOfScope, file)
			continue
		}

		matched := false
		for _, pattern := range task.ScopePaths {
			if matchesScopePath(file, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			outOfScope = append(outOfScope, file)
		}
	}

	if len(outOfScope) == 0 {
		return nil
	}

	errs := make([]string, 0, len(outOfScope)+1)
	scopeDesc := strings.Join(task.ScopePaths, ", ")
	if len(task.ScopePaths) == 0 {
		scopeDesc = ".mission/ only"
	}
	errs = append(errs, fmt.Sprintf("%d file(s) outside task %s scope:", len(outOfScope), taskID))
	for _, f := range outOfScope {
		errs = append(errs, fmt.Sprintf("  - %s (allowed: %s)", f, scopeDesc))
	}
	return errs
}
