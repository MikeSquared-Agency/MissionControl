package bridge

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mike/mission-control/core"
)

// KingStatus represents the King's current status
type KingStatus string

const (
	KingStatusStopped  KingStatus = "stopped"
	KingStatusStarting KingStatus = "starting"
	KingStatusRunning  KingStatus = "running"
	KingStatusError    KingStatus = "error"
)

// KingEvent represents an event from King
type KingEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// KingQuestion represents a question from Claude requiring user input
type KingQuestion struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Selected int      `json:"selected"` // Currently highlighted option (0-indexed)
}

const (
	kingTmuxSession = "mc-king"
	tmuxWidth       = 200
	tmuxHeight      = 50
	pollInterval    = 100 * time.Millisecond
	promptTimeout   = 60 * time.Second
)

// findTmux returns the path to tmux binary
func findTmux() string {
	// Try common locations
	paths := []string{
		"/opt/homebrew/bin/tmux",
		"/usr/local/bin/tmux",
		"/usr/bin/tmux",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fallback to PATH lookup
	if path, err := exec.LookPath("tmux"); err == nil {
		return path
	}
	return "tmux"
}

// findClaude returns the path to claude binary
func findClaude() string {
	paths := []string{
		"/opt/homebrew/bin/claude",
		"/usr/local/bin/claude",
		"/usr/bin/claude",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}
	return "claude"
}

// KingAgentID is the fixed agent ID for King in the agents list
const KingAgentID = "king"

// King manages the King Claude Code process via tmux
type King struct {
	missionDir  string
	workDir     string
	status      KingStatus
	tmuxSession string
	events      chan KingEvent
	mu          sync.RWMutex
	stopChan    chan struct{}
	lastPane    string // Last captured pane state for diff detection
	totalTokens int    // Cumulative token count
	totalCost   float64 // Cumulative cost (estimated)
}

// NewKing creates a new King manager
func NewKing(workDir string) *King {
	missionDir := filepath.Join(workDir, ".mission")

	return &King{
		missionDir:  missionDir,
		workDir:     workDir,
		status:      KingStatusStopped,
		tmuxSession: kingTmuxSession,
		events:      make(chan KingEvent, 100),
		stopChan:    make(chan struct{}),
	}
}

// Events returns the channel for King events
func (k *King) Events() <-chan KingEvent {
	return k.events
}

// Status returns the current King status
func (k *King) Status() KingStatus {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.status
}

// tmuxCmd executes a tmux command and returns output
func (k *King) tmuxCmd(args ...string) (string, error) {
	cmd := exec.Command(findTmux(), args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// capturePane captures the current tmux pane content
func (k *King) capturePane() (string, error) {
	output, err := k.tmuxCmd("capture-pane", "-t", k.tmuxSession, "-p", "-S", "-1000")
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}
	return output, nil
}

// waitForPrompt waits for Claude's ❯ prompt to appear
func (k *King) waitForPrompt(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pane, err := k.capturePane()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Look for the prompt character indicating Claude is ready
		if strings.Contains(pane, "❯") || strings.Contains(pane, ">") {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for Claude prompt")
}

// sessionExists checks if the tmux session exists
func (k *King) sessionExists() bool {
	err := exec.Command(findTmux(), "has-session", "-t", k.tmuxSession).Run()
	return err == nil
}

// Start launches King in a tmux session
func (k *King) Start() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.status == KingStatusRunning {
		return fmt.Errorf("King is already running")
	}

	// Check for CLAUDE.md
	claudeMD := filepath.Join(k.missionDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); os.IsNotExist(err) {
		k.status = KingStatusError
		return fmt.Errorf(".mission/CLAUDE.md not found - run 'mc init' first")
	}

	k.status = KingStatusStarting

	// Kill existing session if present
	if k.sessionExists() {
		exec.Command(findTmux(), "kill-session", "-t", k.tmuxSession).Run()
		time.Sleep(100 * time.Millisecond)
	}

	// Create new tmux session with specified dimensions
	createCmd := exec.Command(findTmux(),
		"new-session", "-d",
		"-s", k.tmuxSession,
		"-x", fmt.Sprintf("%d", tmuxWidth),
		"-y", fmt.Sprintf("%d", tmuxHeight),
		"-c", k.workDir,
	)

	if err := createCmd.Run(); err != nil {
		k.status = KingStatusError
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Start Claude in the session
	claudePath := findClaude()
	_, err := k.tmuxCmd("send-keys", "-t", k.tmuxSession, claudePath, "Enter")
	if err != nil {
		k.killSession()
		k.status = KingStatusError
		return fmt.Errorf("failed to start Claude: %w", err)
	}

	// Wait for Claude to be ready (prompt appears)
	if err := k.waitForPrompt(promptTimeout); err != nil {
		k.killSession()
		k.status = KingStatusError
		return fmt.Errorf("Claude failed to start: %w", err)
	}

	// Capture initial pane state
	k.lastPane, _ = k.capturePane()

	k.status = KingStatusRunning
	k.stopChan = make(chan struct{})
	k.totalTokens = 0
	k.totalCost = 0
	log.Printf("King started in tmux session '%s'", k.tmuxSession)

	// Emit started event
	k.emitEvent("king_started", map[string]interface{}{
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})

	// Emit King as an agent so it appears in the agents list
	// Frontend expects agent data nested under "agent" key
	k.emitEvent("agent_spawned", map[string]interface{}{
		"agent": map[string]interface{}{
			"id":         KingAgentID,
			"name":       "King",
			"type":       "king",
			"persona":    "orchestrator",
			"zone":       "default",
			"workingDir": k.workDir,
			"status":     "running",
			"tokens":     0,
			"cost":       0,
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"task":       "Mission orchestration",
		},
	})

	return nil
}

// killSession kills the tmux session
func (k *King) killSession() {
	exec.Command(findTmux(), "kill-session", "-t", k.tmuxSession).Run()
}

// Stop kills the King tmux session
func (k *King) Stop() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.status != KingStatusRunning {
		return fmt.Errorf("King is not running")
	}

	// Signal any waiting goroutines to stop
	close(k.stopChan)

	// Send Ctrl-C to gracefully stop Claude
	k.tmuxCmd("send-keys", "-t", k.tmuxSession, "C-c")
	time.Sleep(100 * time.Millisecond)

	// Kill the tmux session
	k.killSession()

	k.status = KingStatusStopped
	k.lastPane = ""
	log.Printf("King stopped")

	k.emitEvent("king_stopped", nil)

	// Emit agent stopped so King is removed from agents list
	k.emitEvent("agent_stopped", map[string]interface{}{
		"agent_id": KingAgentID,
	})

	return nil
}

// SendMessage sends a message to King via tmux
func (k *King) SendMessage(message string) error {
	k.mu.RLock()
	status := k.status
	k.mu.RUnlock()

	if status != KingStatusRunning {
		return fmt.Errorf("King is not running")
	}

	// Emit user message event
	k.emitEvent("king_user_message", map[string]interface{}{
		"content":   message,
		"timestamp": time.Now().UnixMilli(),
	})

	// Send the message text using literal mode (-l)
	// No escaping needed - exec.Command handles args directly without shell interpretation
	_, err := k.tmuxCmd("send-keys", "-t", k.tmuxSession, "-l", message)
	if err != nil {
		k.emitEvent("king_error", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Send Escape then Enter to submit (multi-line input mode)
	time.Sleep(100 * time.Millisecond)
	k.tmuxCmd("send-keys", "-t", k.tmuxSession, "Escape")
	time.Sleep(100 * time.Millisecond)
	k.tmuxCmd("send-keys", "-t", k.tmuxSession, "Enter")

	log.Printf("King: sent message (%d chars)", len(message))

	// Wait for response and parse it
	go k.waitForResponse(message)

	return nil
}

// AnswerQuestion responds to a question from Claude by selecting an option
func (k *King) AnswerQuestion(optionIndex int) error {
	k.mu.RLock()
	status := k.status
	k.mu.RUnlock()

	if status != KingStatusRunning {
		return fmt.Errorf("King is not running")
	}

	// Capture current pane to find current selection
	pane, err := k.capturePane()
	if err != nil {
		return fmt.Errorf("failed to capture pane: %w", err)
	}

	if !k.isQuestionUI(pane) {
		return fmt.Errorf("no question UI detected")
	}

	question := k.parseQuestion(pane)
	if question == nil {
		return fmt.Errorf("failed to parse question")
	}

	if optionIndex < 0 || optionIndex >= len(question.Options) {
		return fmt.Errorf("option index %d out of range (0-%d)", optionIndex, len(question.Options)-1)
	}

	// Calculate how many arrow keys to send
	currentIdx := question.Selected
	diff := optionIndex - currentIdx

	log.Printf("King: answering question, moving from option %d to %d", currentIdx, optionIndex)

	// Send arrow keys to navigate
	if diff > 0 {
		for i := 0; i < diff; i++ {
			k.tmuxCmd("send-keys", "-t", k.tmuxSession, "Down")
			time.Sleep(50 * time.Millisecond)
		}
	} else if diff < 0 {
		for i := 0; i < -diff; i++ {
			k.tmuxCmd("send-keys", "-t", k.tmuxSession, "Up")
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Send Enter to confirm selection
	time.Sleep(100 * time.Millisecond)
	k.tmuxCmd("send-keys", "-t", k.tmuxSession, "Enter")

	log.Printf("King: answered question with option %d: %s", optionIndex, question.Options[optionIndex])

	k.emitEvent("king_answer", map[string]interface{}{
		"option_index": optionIndex,
		"option_text":  question.Options[optionIndex],
		"timestamp":    time.Now().UnixMilli(),
	})

	return nil
}

// isQuestionUI checks if the pane is showing a question/selection UI
func (k *King) isQuestionUI(pane string) bool {
	// Look for indicators of a selection UI
	return (strings.Contains(pane, "Enter to select") ||
		strings.Contains(pane, "↑/↓ to navigate")) &&
		(strings.Contains(pane, "☐") ||
			strings.Contains(pane, "○") ||
			strings.Contains(pane, "❯ 1.") ||
			strings.Contains(pane, "❯ 2."))
}

// parseQuestion extracts question details from the pane
func (k *King) parseQuestion(pane string) *KingQuestion {
	lines := strings.Split(pane, "\n")

	var question KingQuestion
	var foundQuestion bool
	var options []string
	selectedIdx := 0

	// Regex to match option lines: "❯ 1. Option" or "  2. Option"
	optionRegex := regexp.MustCompile(`^\s*(❯)?\s*(\d+)\.\s*(.+)$`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for question text (usually after ☐ Task or before options)
		if strings.Contains(line, "☐") || strings.Contains(line, "☑") {
			// Next non-empty line might be the question
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine != "" && !optionRegex.MatchString(lines[j]) && !strings.HasPrefix(nextLine, "❯") {
					question.Question = nextLine
					foundQuestion = true
					break
				}
			}
		}

		// Match option lines
		if matches := optionRegex.FindStringSubmatch(line); len(matches) > 0 {
			isSelected := matches[1] == "❯"
			optionText := strings.TrimSpace(matches[3])

			// Clean up option text (remove trailing descriptions on same line)
			if idx := strings.Index(optionText, "\t"); idx > 0 {
				optionText = strings.TrimSpace(optionText[:idx])
			}

			options = append(options, optionText)
			if isSelected {
				selectedIdx = len(options) - 1
			}
		}

		// Stop at the navigation hint line
		if strings.Contains(trimmed, "Enter to select") {
			break
		}
	}

	if len(options) == 0 {
		return nil
	}

	// If we didn't find a question, try to extract from context
	if !foundQuestion {
		// Look for a line ending with "?"
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasSuffix(trimmed, "?") && len(trimmed) > 10 {
				question.Question = trimmed
				break
			}
		}
	}

	question.Options = options
	question.Selected = selectedIdx

	return &question
}

// waitForResponse polls for Claude's response and emits events
func (k *King) waitForResponse(userMessage string) {
	deadline := time.Now().Add(5 * time.Minute) // Long timeout for complex responses
	var lastResponse string
	var lastQuestionHash string
	var sawUserMessage bool

	log.Printf("King: waiting for response to message: %q", userMessage)

	for time.Now().Before(deadline) {
		select {
		case <-k.stopChan:
			return
		default:
		}

		pane, err := k.capturePane()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Wait until we see the user's message in the pane (confirms it was sent)
		if !sawUserMessage {
			if strings.Contains(pane, userMessage) {
				sawUserMessage = true
				log.Printf("King: user message confirmed in pane")
			} else {
				time.Sleep(pollInterval)
				continue
			}
		}

		// Check if Claude is asking a question (tool use)
		if k.isQuestionUI(pane) {
			question := k.parseQuestion(pane)
			if question != nil {
				// Create a hash to avoid emitting duplicate question events
				questionHash := fmt.Sprintf("%s:%d", question.Question, len(question.Options))
				if questionHash != lastQuestionHash {
					lastQuestionHash = questionHash
					log.Printf("King: question detected with %d options", len(question.Options))
					k.emitEvent("king_question", question)
				}
			}
			time.Sleep(pollInterval)
			continue
		}

		// Parse response from current pane (only responses after the user's message)
		response := k.extractResponseAfterMessage(pane, userMessage)
		complete := k.isResponseComplete(pane)

		// Check if Claude is done (prompt visible AND we have a response)
		if complete && response != "" {
			if response != lastResponse {
				log.Printf("King: final response (%d chars)", len(response))
				k.emitEvent("king_message", map[string]interface{}{
					"role":      "assistant",
					"content":   response,
					"timestamp": time.Now().UnixMilli(),
				})

				// Count tokens in the response using mc-core
				if outputTokens, err := core.CountTokens(response); err == nil {
					// Count input tokens (user message) as well
					inputTokens := 0
					if userMessage != "" {
						if count, err := core.CountTokens(userMessage); err == nil {
							inputTokens = count
						}
					}

					// Update cumulative totals
					totalNewTokens := inputTokens + outputTokens
					// Approximate cost: $0.003/1K input, $0.015/1K output for Claude
					newCost := (float64(inputTokens) * 0.003 / 1000) + (float64(outputTokens) * 0.015 / 1000)

					k.mu.Lock()
					k.totalTokens += totalNewTokens
					k.totalCost += newCost
					currentTokens := k.totalTokens
					currentCost := k.totalCost
					k.mu.Unlock()

					log.Printf("King: token usage - input: %d, output: %d, total: %d, cost: $%.4f", inputTokens, outputTokens, currentTokens, currentCost)

					// Emit token_usage for detailed tracking
					k.emitEvent("token_usage", map[string]interface{}{
						"input_tokens":  inputTokens,
						"output_tokens": outputTokens,
						"timestamp":     time.Now().UnixMilli(),
					})

					// Emit tokens_updated so King appears with tokens in agent list
					k.emitEvent("tokens_updated", map[string]interface{}{
						"agent_id": KingAgentID,
						"tokens":   currentTokens,
						"cost":     currentCost,
					})
				} else {
					log.Printf("King: failed to count tokens: %v", err)
				}
			}

			k.mu.Lock()
			k.lastPane = pane
			k.mu.Unlock()
			return
		}

		// Emit streaming updates if response changed
		if response != "" && response != lastResponse {
			lastResponse = response
			k.emitEvent("king_output", map[string]interface{}{
				"text":      response,
				"streaming": true,
			})
		}

		time.Sleep(pollInterval)
	}

	log.Printf("King: timeout waiting for response")
	k.emitEvent("king_error", map[string]interface{}{"error": "timeout waiting for response"})
}

// isResponseComplete checks if Claude has finished responding
// Returns true when we see the ❯ prompt after response content
func (k *King) isResponseComplete(pane string) bool {
	lines := strings.Split(pane, "\n")

	// Find last non-empty line index first
	lastNonEmpty := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastNonEmpty = i
			break
		}
	}

	if lastNonEmpty < 0 {
		return false
	}

	// Check if we're in a question/selection UI (not complete)
	// Look for indicators like "Enter to select", "↑/↓ to navigate", checkbox markers
	for i := lastNonEmpty; i >= 0 && i >= lastNonEmpty-10; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, "Enter to select") ||
			strings.Contains(line, "↑/↓") ||
			strings.Contains(line, "☐") ||
			strings.Contains(line, "☑") ||
			strings.Contains(line, "○") ||
			strings.Contains(line, "●") {
			return false // In a selection UI, not complete
		}
	}

	// Look for prompt in last 15 non-empty lines from the actual content end
	checkedLines := 0
	for i := lastNonEmpty; i >= 0 && checkedLines < 15; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		checkedLines++

		// Check for working indicators - definitely not complete
		if strings.Contains(line, "∴") && strings.Contains(line, "Thinking") {
			return false
		}

		// Check if line starts with spinner character (braille patterns)
		runes := []rune(line)
		if len(runes) > 0 {
			first := runes[0]
			if first >= '⠀' && first <= '⣿' { // Braille pattern block - spinner
				return false
			}
		}

		// Found the input prompt - must be just "❯" or "❯ " followed by user input area
		// NOT a selection cursor like "❯ 1. Option"
		if strings.HasPrefix(line, "❯") {
			// Check it's not a selection indicator (❯ followed by number or option)
			rest := strings.TrimPrefix(line, "❯")
			rest = strings.TrimSpace(rest)
			// If empty or looks like user could type here, it's the prompt
			if rest == "" || !strings.ContainsAny(rest[:min(1, len(rest))], "0123456789") {
				// Make sure it's not in a selection context
				if !strings.Contains(line, ".") || len(rest) > 50 {
					return true
				}
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractResponseAfterMessage extracts Claude's response that appears after the user's message
func (k *King) extractResponseAfterMessage(pane, userMessage string) string {
	lines := strings.Split(pane, "\n")

	// Find where the user's message appears on a prompt line (starts with ❯)
	messageLineIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Only look at prompt lines (where user input appears)
		if strings.HasPrefix(trimmed, "❯") && strings.Contains(line, userMessage) {
			messageLineIdx = i
			// Don't break - we want the LAST occurrence in case message appears multiple times
		}
	}

	if messageLineIdx < 0 {
		return ""
	}

	// Only look at lines AFTER the user's message
	return k.extractResponseFromLines(lines[messageLineIdx+1:])
}

// extractResponseFromLines extracts Claude's response from a slice of lines
func (k *King) extractResponseFromLines(lines []string) string {
	var response strings.Builder

	// Regex to match Claude's response markers
	// ⏺ marks the start of Claude's response
	responseMarker := regexp.MustCompile(`⏺\s*(.*)`)

	inResponse := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines unless we're in a response
		if trimmed == "" {
			if inResponse {
				response.WriteString("\n")
			}
			continue
		}

		// Skip thinking sections
		if strings.Contains(line, "∴") {
			inResponse = false // Reset - thinking means new response coming
			continue
		}

		// Check for response marker
		if matches := responseMarker.FindStringSubmatch(line); len(matches) > 1 {
			inResponse = true
			response.Reset() // Start fresh for this response
			if matches[1] != "" {
				response.WriteString(matches[1])
				response.WriteString("\n")
			}
			continue
		}

		// If we're in a response block
		if inResponse {
			// Stop at prompt (but keep the response we've collected)
			if strings.HasPrefix(trimmed, "❯") {
				break
			}
			// Stop at horizontal lines (end of response box)
			if strings.HasPrefix(trimmed, "───") {
				break
			}
			// Clean up box-drawing characters from continuation lines
			cleaned := strings.TrimLeft(line, "│ \t")
			// Skip box decorations
			if strings.HasPrefix(cleaned, "─") || strings.HasPrefix(cleaned, "└") ||
				strings.HasPrefix(cleaned, "┘") || strings.HasPrefix(cleaned, "╰") ||
				strings.HasPrefix(cleaned, "╯") {
				continue
			}
			if cleaned != "" {
				response.WriteString(cleaned)
				response.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(response.String())
}

// emitEvent sends an event to the events channel
func (k *King) emitEvent(eventType string, data interface{}) {
	event := KingEvent{
		Type: eventType,
		Data: data,
	}

	select {
	case k.events <- event:
	default:
		log.Printf("King: event channel full, dropping event: %s", eventType)
	}
}

// IsRunning returns true if King process is running and ready for messages
func (k *King) IsRunning() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.status == KingStatusRunning
}
