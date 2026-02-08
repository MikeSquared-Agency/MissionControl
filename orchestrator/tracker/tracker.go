package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ProcessStatus represents the lifecycle state of a tracked worker.
type ProcessStatus string

const (
	StatusRunning  ProcessStatus = "running"
	StatusComplete ProcessStatus = "complete"
	StatusError    ProcessStatus = "error"
	StatusKilled   ProcessStatus = "killed"
)

// TrackedProcess holds runtime state for a single worker process.
type TrackedProcess struct {
	WorkerID   string        `json:"worker_id"`
	Persona    string        `json:"persona"`
	TaskID     string        `json:"task_id"`
	Zone       string        `json:"zone"`
	Model      string        `json:"model"`
	PID        int           `json:"pid"`
	Status     ProcessStatus `json:"status"`
	StartedAt  time.Time     `json:"started_at"`
	TokenCount int           `json:"token_count"`
	CostUSD    float64       `json:"cost_usd"`
}

// EventCallback is invoked when process state changes.
type EventCallback func(eventType string, process *TrackedProcess)

// Tracker discovers and monitors worker processes by polling workers.json.
type Tracker struct {
	processes  map[string]*TrackedProcess
	mu         sync.RWMutex
	missionDir string
	callback   EventCallback
	stopCh     chan struct{}
}

// workerEntry mirrors the JSON shape inside workers.json.
type workerEntry struct {
	WorkerID string `json:"worker_id"`
	Persona  string `json:"persona"`
	TaskID   string `json:"task_id"`
	Zone     string `json:"zone"`
	Model    string `json:"model"`
	PID      int    `json:"pid"`
	Status   string `json:"status"`
}

// NewTracker creates a Tracker rooted at the given mission directory.
func NewTracker(missionDir string, callback EventCallback) *Tracker {
	return &Tracker{
		processes:  make(map[string]*TrackedProcess),
		missionDir: missionDir,
		callback:   callback,
		stopCh:     make(chan struct{}),
	}
}

// Start begins background polling. Call Stop to terminate.
func (t *Tracker) Start() {
	go t.pollLoop()
	go t.heartbeatLoop()
}

// Stop terminates background polling.
func (t *Tracker) Stop() {
	close(t.stopCh)
}

// List returns a snapshot of all tracked processes.
func (t *Tracker) List() []*TrackedProcess {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]*TrackedProcess, 0, len(t.processes))
	for _, p := range t.processes {
		cp := *p
		out = append(out, &cp)
	}
	return out
}

// Get returns a tracked process by worker ID.
func (t *Tracker) Get(workerID string) (*TrackedProcess, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	p, ok := t.processes[workerID]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// Kill sends SIGTERM to a worker, waits 5 s, then SIGKILL if still alive.
func (t *Tracker) Kill(workerID string) error {
	t.mu.RLock()
	p, ok := t.processes[workerID]
	if !ok {
		t.mu.RUnlock()
		return fmt.Errorf("worker %s not tracked", workerID)
	}
	pid := p.PID
	t.mu.RUnlock()

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	// SIGTERM
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process may already be dead — still mark killed.
		t.setStatus(workerID, StatusKilled)
		return nil
	}

	// Wait up to 5 s for exit.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			if !isAlive(pid) {
				close(done)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	// If still alive, SIGKILL.
	if isAlive(pid) {
		_ = proc.Signal(syscall.SIGKILL)
	}

	t.setStatus(workerID, StatusKilled)
	return nil
}

// Reset clears all tracked processes (useful for project switching).
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.processes = make(map[string]*TrackedProcess)
}

// UpdateTokens updates the token count and cost for a worker.
func (t *Tracker) UpdateTokens(workerID string, tokens int, cost float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.processes[workerID]; ok {
		p.TokenCount = tokens
		p.CostUSD = cost
	}
}

// --- internal helpers ---

func (t *Tracker) setStatus(workerID string, status ProcessStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.processes[workerID]; ok {
		p.Status = status
		if t.callback != nil {
			cp := *p
			t.callback("status_changed", &cp)
		}
	}
}

func (t *Tracker) workersPath() string {
	return filepath.Join(t.missionDir, ".mission", "state", "workers.json")
}

func (t *Tracker) pollLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.poll()
		}
	}
}

func (t *Tracker) heartbeatLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.emitHeartbeats()
		}
	}
}

func (t *Tracker) emitHeartbeats() {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.callback == nil {
		return
	}
	for _, p := range t.processes {
		if p.Status == StatusRunning {
			cp := *p
			t.callback("heartbeat", &cp)
		}
	}
}

func (t *Tracker) poll() {
	data, err := os.ReadFile(t.workersPath())
	if err != nil {
		return // file may not exist yet
	}

	var entries []workerEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	seen := make(map[string]bool, len(entries))

	for _, e := range entries {
		seen[e.WorkerID] = true
		existing, tracked := t.processes[e.WorkerID]

		if !tracked {
			// New worker discovered.
			p := &TrackedProcess{
				WorkerID:  e.WorkerID,
				Persona:   e.Persona,
				TaskID:    e.TaskID,
				Zone:      e.Zone,
				Model:     e.Model,
				PID:       e.PID,
				Status:    ProcessStatus(e.Status),
				StartedAt: time.Now(),
			}
			t.processes[e.WorkerID] = p
			if t.callback != nil {
				cp := *p
				t.callback("spawned", &cp)
			}
			continue
		}

		// Status change in workers.json?
		newStatus := ProcessStatus(e.Status)
		if existing.Status != newStatus {
			existing.Status = newStatus
			if t.callback != nil {
				cp := *existing
				t.callback("status_changed", &cp)
			}
			continue
		}

		// PID health check for running processes.
		if existing.Status == StatusRunning && !isAlive(existing.PID) {
			existing.Status = StatusError
			if t.callback != nil {
				cp := *existing
				t.callback("error", &cp)
			}
		}
	}
}

// isAlive checks whether a PID is still running via signal 0.
func isAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
