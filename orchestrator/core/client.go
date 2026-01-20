// Package core provides a Go client for the mc-core Rust CLI.
// It handles token counting, handoff validation, and gate checking via subprocess calls.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// findMcCore locates the mc-core binary.
// It checks PATH first, then falls back to common installation locations.
func findMcCore() (string, error) {
	// Try PATH first
	if path, err := exec.LookPath("mc-core"); err == nil {
		return path, nil
	}

	// Try same directory as current executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		path := filepath.Join(exeDir, "mc-core")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		// Try ../dist relative to executable (development layout)
		path = filepath.Join(exeDir, "..", "dist", "mc-core")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try relative to current working directory (development)
	if cwd, err := os.Getwd(); err == nil {
		devPaths := []string{
			filepath.Join(cwd, "dist", "mc-core"),
			filepath.Join(cwd, "..", "dist", "mc-core"),
			filepath.Join(cwd, "core", "target", "release", "mc-core"),
			filepath.Join(cwd, "..", "core", "target", "release", "mc-core"),
		}
		for _, p := range devPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	// Common installation paths
	paths := []string{
		"/usr/local/bin/mc-core",
		"/opt/homebrew/bin/mc-core",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("mc-core binary not found in PATH or common locations")
}

// TokenResult is the JSON response from mc-core count-tokens
type TokenResult struct {
	Tokens int `json:"tokens"`
}

// CountTokens counts the tokens in the given text using tiktoken (cl100k_base encoding).
// It calls mc-core count-tokens - with the text piped to stdin.
func CountTokens(text string) (int, error) {
	mcCore, err := findMcCore()
	if err != nil {
		return 0, err
	}

	cmd := exec.Command(mcCore, "count-tokens", "-")
	cmd.Stdin = strings.NewReader(text)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("mc-core count-tokens failed: %s", string(exitErr.Stderr))
		}
		return 0, fmt.Errorf("mc-core count-tokens failed: %w", err)
	}

	var result TokenResult
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse mc-core output: %w (output: %s)", err, string(output))
	}

	return result.Tokens, nil
}

// ValidationResult is the JSON response from mc-core validate-handoff
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// ValidateHandoff validates a handoff JSON file against the schema.
func ValidateHandoff(path string) (*ValidationResult, error) {
	mcCore, err := findMcCore()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(mcCore, "validate-handoff", path)
	output, err := cmd.Output()
	if err != nil {
		// validate-handoff returns exit code 1 on validation failure,
		// but still outputs valid JSON, so we parse it anyway
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Try to parse the stdout first (validation result)
			if len(output) > 0 {
				var result ValidationResult
				if jsonErr := json.Unmarshal(output, &result); jsonErr == nil {
					return &result, nil
				}
			}
			return nil, fmt.Errorf("mc-core validate-handoff failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("mc-core validate-handoff failed: %w", err)
	}

	var result ValidationResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse mc-core output: %w", err)
	}

	return &result, nil
}

// GateCriterion represents a single gate criterion
type GateCriterion struct {
	Description string `json:"description"`
	Satisfied   bool   `json:"satisfied"`
}

// GateResult is the JSON response from mc-core check-gate
type GateResult struct {
	Phase      string          `json:"phase"`
	Status     string          `json:"status"` // "open", "closed", "awaiting_approval"
	Criteria   []GateCriterion `json:"criteria"`
	CanApprove bool            `json:"can_approve"`
}

// CheckGate checks the gate status for a given phase.
func CheckGate(phase string, missionDir string) (*GateResult, error) {
	mcCore, err := findMcCore()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(mcCore, "check-gate", phase, "--mission-dir", missionDir)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("mc-core check-gate failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("mc-core check-gate failed: %w", err)
	}

	var result GateResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse mc-core output: %w", err)
	}

	return &result, nil
}
