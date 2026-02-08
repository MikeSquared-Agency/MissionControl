package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// AutoCommitCategory defines which categories of state mutations trigger git commits.
type AutoCommitCategory string

const (
	CommitCategoryCheckpoint AutoCommitCategory = "checkpoint"
	CommitCategoryTask       AutoCommitCategory = "task"
	CommitCategoryGate       AutoCommitCategory = "gate"
	CommitCategoryStage      AutoCommitCategory = "stage"
	CommitCategoryWorker     AutoCommitCategory = "worker"
	CommitCategoryHandoff    AutoCommitCategory = "handoff"
)

// AutoCommitConfig controls which state mutations trigger git commits.
// Stored in .mission/config.json under "auto_commit".
type AutoCommitConfig struct {
	Enabled    bool `json:"enabled"`    // Master switch (default: true)
	Checkpoint bool `json:"checkpoint"` // Checkpoint creation (default: true, legacy behavior)
	Task       bool `json:"task"`       // Task create/update/complete (default: true)
	Gate       bool `json:"gate"`       // Gate approvals (default: true)
	Stage      bool `json:"stage"`      // Stage transitions (default: true)
	Worker     bool `json:"worker"`     // Worker spawn/kill (default: true)
	Handoff    bool `json:"handoff"`    // Handoff processing (default: true)
}

// DefaultAutoCommitConfig returns the default config with everything enabled.
func DefaultAutoCommitConfig() AutoCommitConfig {
	return AutoCommitConfig{
		Enabled:    true,
		Checkpoint: true,
		Task:       true,
		Gate:       true,
		Stage:      true,
		Worker:     true,
		Handoff:    true,
	}
}

// loadAutoCommitConfig reads the auto_commit config from .mission/config.json.
func loadAutoCommitConfig(missionDir string) AutoCommitConfig {
	configPath := filepath.Join(missionDir, "config.json")
	var cfg struct {
		AutoCommit *AutoCommitConfig `json:"auto_commit,omitempty"`
	}
	if err := readJSON(configPath, &cfg); err != nil || cfg.AutoCommit == nil {
		return DefaultAutoCommitConfig()
	}
	return *cfg.AutoCommit
}

// gitAutoCommit stages .mission/ changes and commits with the given message,
// if the given category is enabled in config.
func gitAutoCommit(missionDir string, category AutoCommitCategory, msg string) {
	cfg := loadAutoCommitConfig(missionDir)
	if !cfg.Enabled {
		return
	}

	switch category {
	case CommitCategoryCheckpoint:
		if !cfg.Checkpoint {
			return
		}
	case CommitCategoryTask:
		if !cfg.Task {
			return
		}
	case CommitCategoryGate:
		if !cfg.Gate {
			return
		}
	case CommitCategoryStage:
		if !cfg.Stage {
			return
		}
	case CommitCategoryWorker:
		if !cfg.Worker {
			return
		}
	case CommitCategoryHandoff:
		if !cfg.Handoff {
			return
		}
	}

	projectDir := filepath.Dir(missionDir)

	// Check if we're in a git repo
	gitCheck := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	gitCheck.Dir = projectDir
	if err := gitCheck.Run(); err != nil {
		return
	}

	// Stage all .mission/ changes
	gitAdd := exec.Command("git", "add", ".mission/")
	gitAdd.Dir = projectDir
	if err := gitAdd.Run(); err != nil {
		return
	}

	// Check if there are staged changes (avoid empty commits for non-checkpoint)
	gitDiff := exec.Command("git", "diff", "--cached", "--quiet")
	gitDiff.Dir = projectDir
	if err := gitDiff.Run(); err == nil {
		// No changes staged, skip commit
		return
	}

	// Commit
	prefix := string(category)
	commitMsg := fmt.Sprintf("[mc:%s] %s", prefix, msg)
	gitCommit := exec.Command("git", "commit", "-m", commitMsg)
	gitCommit.Dir = projectDir
	_ = gitCommit.Run()
}

// shortID returns first 8 chars of an ID for commit messages
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// taskCommitMsg generates a commit message for task mutations
func taskCommitMsg(action, taskID, name string) string {
	parts := []string{action}
	if taskID != "" {
		parts = append(parts, shortID(taskID))
	}
	if name != "" {
		parts = append(parts, fmt.Sprintf("%q", name))
	}
	return strings.Join(parts, " ")
}
