package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FileProtocol handles file-based communication with Claude agents.
// It uses the mc-protocol Rust binary for reliable completion detection.
type FileProtocol struct {
	missionDir  string
	sessionName string
}

// NewFileProtocol creates a new file-based protocol handler.
func NewFileProtocol(missionDir, sessionName string) *FileProtocol {
	return &FileProtocol{
		missionDir:  missionDir,
		sessionName: sessionName,
	}
}

// TaskCompletionResult represents the result from watch-task
type TaskCompletionResult struct {
	Status       string `json:"status"`        // "complete" or "timeout"
	ResponsePath string `json:"response_path"` // Path to response file
}

// ConversationResult represents the result from watch-conversation
type ConversationResult struct {
	Status   string `json:"status"`   // "complete" or "timeout"
	Response string `json:"response"` // The assistant's response text
}

// ValidationResult represents the result from validate-task
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ParsedResponse represents a parsed response file
type ParsedResponse struct {
	Summary       *string  `json:"summary"`
	Details       *string  `json:"details"`
	FilesModified []string `json:"files_modified"`
	Notes         *string  `json:"notes"`
}

// EnsureDirectories creates the required .mission subdirectories.
func (p *FileProtocol) EnsureDirectories() error {
	dirs := []string{
		"tasks",
		"responses",
		"status",
	}

	for _, dir := range dirs {
		path := filepath.Join(p.missionDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return nil
}

// WriteTask writes a task file for a worker to pick up.
func (p *FileProtocol) WriteTask(taskID, instructions, context string, priority string) error {
	if priority == "" {
		priority = "normal"
	}

	taskPath := filepath.Join(p.missionDir, "tasks", fmt.Sprintf("task-%s.md", taskID))
	responsePath := filepath.Join(p.missionDir, "responses", fmt.Sprintf("task-%s.md", taskID))
	statusPath := filepath.Join(p.missionDir, "status", fmt.Sprintf("task-%s.status", taskID))

	content := fmt.Sprintf(`# Task: %s
Created: %s
Priority: %s

## Instructions

%s

## Context

%s

## Response Instructions

When complete, write your response to %s
and create %s with content "DONE".
`, taskID, time.Now().UTC().Format(time.RFC3339), priority, instructions, context, responsePath, statusPath)

	return os.WriteFile(taskPath, []byte(content), 0644)
}

// WaitForTaskCompletion waits for a task to complete by watching for its status file.
// Returns the response content when complete, or error on timeout.
func (p *FileProtocol) WaitForTaskCompletion(ctx context.Context, taskID string, timeout time.Duration) (*ParsedResponse, error) {
	protocolBin := findMCProtocol()

	cmd := exec.CommandContext(ctx, protocolBin, "watch-task",
		"--task-id", taskID,
		"--mission-dir", p.missionDir,
		"--timeout", fmt.Sprintf("%d", int(timeout.Seconds())),
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("protocol error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run mc-protocol: %w", err)
	}

	var result TaskCompletionResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse mc-protocol output: %w", err)
	}

	if result.Status == "timeout" {
		return nil, ErrProtocolTimeout
	}

	// Read and parse the response file
	responseContent, err := os.ReadFile(result.ResponsePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read response file: %w", err)
	}

	// Parse the response using mc-protocol
	return p.ParseResponse(result.ResponsePath, responseContent)
}

// AppendConversationMessage appends a human message to conversation.md
func (p *FileProtocol) AppendConversationMessage(message string) error {
	convPath := filepath.Join(p.missionDir, "conversation.md")

	entry := fmt.Sprintf("\n## Human [%s]\n\n%s\n\n---\n",
		time.Now().UTC().Format(time.RFC3339),
		message,
	)

	f, err := os.OpenFile(convPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open conversation.md: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to conversation.md: %w", err)
	}

	return nil
}

// WaitForConversationResponse waits for the assistant to complete their response.
// Returns the response text when the ---END--- marker is detected.
func (p *FileProtocol) WaitForConversationResponse(ctx context.Context, timeout time.Duration) (string, error) {
	protocolBin := findMCProtocol()

	cmd := exec.CommandContext(ctx, protocolBin, "watch-conversation",
		"--mission-dir", p.missionDir,
		"--timeout", fmt.Sprintf("%d", int(timeout.Seconds())),
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("protocol error: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to run mc-protocol: %w", err)
	}

	var result ConversationResult
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse mc-protocol output: %w", err)
	}

	if result.Status == "timeout" {
		return "", ErrProtocolTimeout
	}

	return result.Response, nil
}

// ParseResponse parses a response file to extract structured content.
func (p *FileProtocol) ParseResponse(path string, content []byte) (*ParsedResponse, error) {
	protocolBin := findMCProtocol()

	cmd := exec.Command(protocolBin, "parse-response", "--file", path)
	output, err := cmd.Output()
	if err != nil {
		// If parsing fails, return raw content as details
		details := string(content)
		return &ParsedResponse{
			Details:       &details,
			FilesModified: []string{},
		}, nil
	}

	var result ParsedResponse
	if err := json.Unmarshal(output, &result); err != nil {
		details := string(content)
		return &ParsedResponse{
			Details:       &details,
			FilesModified: []string{},
		}, nil
	}

	return &result, nil
}

// NudgeAgent is a no-op — previously sent keys to a tmux session.
// Agent communication now happens via OpenClaw gateway.
func (p *FileProtocol) NudgeAgent(message string) error {
	return nil
}

// findMCProtocol returns the path to the mc-protocol binary
func findMCProtocol() string {
	// Try common locations
	paths := []string{
		"/usr/local/bin/mc-protocol",
		"/opt/homebrew/bin/mc-protocol",
		"./dist/mc-protocol",
		"./core/target/release/mc-protocol",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Fallback to PATH lookup
	if path, err := exec.LookPath("mc-protocol"); err == nil {
		return path
	}
	return "mc-protocol"
}

// ErrProtocolTimeout is returned when waiting for a response times out
var ErrProtocolTimeout = fmt.Errorf("protocol timeout: agent did not respond in time")
