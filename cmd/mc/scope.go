package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// getStagedFiles returns the list of staged file paths (relative to repo root).
func getStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
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

// validateScope checks that all staged files fall within a task's scope_paths.
// Returns a list of error strings for out-of-scope files, or nil if everything is fine.
// If the task has empty/nil scope_paths, validation is skipped entirely.
func validateScope(missionDir, taskID string, stagedFiles []string) []string {
	found, err := findTaskByID(missionDir, taskID)
	if err != nil {
		return []string{err.Error()}
	}
	task := &found

	if len(task.ScopePaths) == 0 {
		return nil
	}

	var outOfScope []string
	for _, file := range stagedFiles {
		if strings.HasPrefix(file, ".mission/") {
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
	errs = append(errs, fmt.Sprintf("%d file(s) outside task %s scope:", len(outOfScope), taskID))
	for _, f := range outOfScope {
		errs = append(errs, fmt.Sprintf("  - %s (allowed: %s)", f, strings.Join(task.ScopePaths, ", ")))
	}
	return errs
}
