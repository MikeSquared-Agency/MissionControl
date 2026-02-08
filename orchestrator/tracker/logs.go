package tracker

import (
	"sync"
	"time"
)

// LogLine represents a single line of output from a worker process.
type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	Stream    string    `json:"stream"` // "stdout" or "stderr"
}

// LogBuffer is a bounded ring buffer of log lines for a single worker.
type LogBuffer struct {
	lines    []LogLine
	maxLines int
	mu       sync.RWMutex
}

// NewLogBuffer creates a LogBuffer that retains at most maxLines entries.
// If maxLines <= 0 it defaults to 200.
func NewLogBuffer(maxLines int) *LogBuffer {
	if maxLines <= 0 {
		maxLines = 200
	}
	return &LogBuffer{
		lines:    make([]LogLine, 0, maxLines),
		maxLines: maxLines,
	}
}

// Append adds a log line, dropping the oldest if the buffer is full.
func (b *LogBuffer) Append(line LogLine) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.lines) >= b.maxLines {
		b.lines = b.lines[1:]
	}
	b.lines = append(b.lines, line)
}

// Lines returns a copy of all buffered lines.
func (b *LogBuffer) Lines() []LogLine {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]LogLine, len(b.lines))
	copy(out, b.lines)
	return out
}

// Recent returns the last n lines (or fewer if the buffer is smaller).
func (b *LogBuffer) Recent(n int) []LogLine {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if n > len(b.lines) {
		n = len(b.lines)
	}
	out := make([]LogLine, n)
	copy(out, b.lines[len(b.lines)-n:])
	return out
}

// Clear empties the buffer.
func (b *LogBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = b.lines[:0]
}
