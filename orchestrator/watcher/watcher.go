package watcher

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event represents a state change event
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// StageState represents the stage.json structure
type StageState struct {
	Current   string `json:"current"`
	UpdatedAt string `json:"updated_at"`
}

// Task represents a task from tasks.json
type Task struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Stage     string `json:"stage"`
	Zone      string `json:"zone"`
	Persona   string `json:"persona"`
	Status    string `json:"status"`
	WorkerID  string `json:"worker_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Worker represents a worker from workers.json
type Worker struct {
	ID        string `json:"id"`
	Persona   string `json:"persona"`
	TaskID    string `json:"task_id"`
	Zone      string `json:"zone"`
	Status    string `json:"status"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

// WorkersState represents the workers.json structure
type WorkersState struct {
	Workers []Worker `json:"workers"`
}

// Gate represents a gate from gates.json
type Gate struct {
	Stage      string   `json:"stage"`
	Status     string   `json:"status"`
	Criteria   []string `json:"criteria"`
	ApprovedAt string   `json:"approved_at,omitempty"`
}

// GatesState represents the gates.json structure
type GatesState struct {
	Gates map[string]Gate `json:"gates"`
}

// Watcher watches the .mission/state/ directory for changes
type Watcher struct {
	missionDir string
	events     chan Event
	stopCh     chan struct{}
	mu         sync.RWMutex

	// Last known state for diffing
	lastStage   StageState
	lastTasks   map[string]Task
	lastWorkers map[string]Worker
	lastGates   map[string]Gate
}

// NewWatcher creates a new state watcher
func NewWatcher(missionDir string) *Watcher {
	return &Watcher{
		missionDir:  missionDir,
		events:      make(chan Event, 100),
		stopCh:      make(chan struct{}),
		lastTasks:   make(map[string]Task),
		lastWorkers: make(map[string]Worker),
		lastGates:   make(map[string]Gate),
	}
}

// Events returns the channel for state change events
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	stateDir := filepath.Join(w.missionDir, "state")

	// Check if state directory exists
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		return fmt.Errorf(".mission/state/ not found: %s", stateDir)
	}

	// Load initial state
	w.loadInitialState()

	// Start polling (simple approach - could use fsnotify for production)
	go w.poll()

	log.Printf("Watcher started on %s", stateDir)
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.stopCh)
}

// poll checks for file changes periodically
func (w *Watcher) poll() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkForChanges()
		}
	}
}

// loadInitialState loads the current state files
func (w *Watcher) loadInitialState() {
	w.mu.Lock()
	defer w.mu.Unlock()

	stateDir := filepath.Join(w.missionDir, "state")

	// Load stage
	if data, err := os.ReadFile(filepath.Join(stateDir, "stage.json")); err == nil {
		json.Unmarshal(data, &w.lastStage)
	}

	// Load tasks (JSONL format, one task per line)
	if tasks, err := readTasksJSONLFile(filepath.Join(stateDir, "tasks.jsonl")); err == nil {
		for _, t := range tasks {
			w.lastTasks[t.ID] = t
		}
	}

	// Load workers
	var workersState WorkersState
	if data, err := os.ReadFile(filepath.Join(stateDir, "workers.json")); err == nil {
		json.Unmarshal(data, &workersState)
		for _, wr := range workersState.Workers {
			w.lastWorkers[wr.ID] = wr
		}
	}

	// Load gates
	var gatesState GatesState
	if data, err := os.ReadFile(filepath.Join(stateDir, "gates.json")); err == nil {
		json.Unmarshal(data, &gatesState)
		w.lastGates = gatesState.Gates
	}
}

// checkForChanges compares current state with last known state
func (w *Watcher) checkForChanges() {
	stateDir := filepath.Join(w.missionDir, "state")

	// Check stage
	var currentStage StageState
	if data, err := os.ReadFile(filepath.Join(stateDir, "stage.json")); err == nil {
		json.Unmarshal(data, &currentStage)
		w.mu.Lock()
		if currentStage.Current != w.lastStage.Current {
			w.emitEvent("stage_changed", map[string]interface{}{
				"previous": w.lastStage.Current,
				"current":  currentStage.Current,
			})
			w.lastStage = currentStage
		}
		w.mu.Unlock()
	}

	// Check tasks
	if tasks, err := readTasksJSONLFile(filepath.Join(stateDir, "tasks.jsonl")); err == nil {
		w.mu.Lock()
		currentTasks := make(map[string]Task)
		for _, t := range tasks {
			currentTasks[t.ID] = t

			// Check if new or updated
			if lastTask, exists := w.lastTasks[t.ID]; !exists {
				w.emitEvent("task_created", t)
			} else if t.Status != lastTask.Status || t.UpdatedAt != lastTask.UpdatedAt {
				w.emitEvent("task_updated", map[string]interface{}{
					"task_id": t.ID,
					"status":  t.Status,
					"task":    t,
				})
			}
		}
		w.lastTasks = currentTasks
		w.mu.Unlock()
	}

	// Check workers
	var workersState WorkersState
	if data, err := os.ReadFile(filepath.Join(stateDir, "workers.json")); err == nil {
		json.Unmarshal(data, &workersState)

		w.mu.Lock()
		currentWorkers := make(map[string]Worker)
		for _, wr := range workersState.Workers {
			currentWorkers[wr.ID] = wr

			// Check if new or status changed
			if lastWorker, exists := w.lastWorkers[wr.ID]; !exists {
				w.emitEvent("worker_spawned", map[string]interface{}{
					"worker_id": wr.ID,
					"persona":   wr.Persona,
					"zone":      wr.Zone,
					"task_id":   wr.TaskID,
				})
			} else if wr.Status != lastWorker.Status {
				if wr.Status == "complete" {
					w.emitEvent("worker_completed", map[string]interface{}{
						"worker_id": wr.ID,
						"task_id":   wr.TaskID,
					})
				} else {
					w.emitEvent("worker_status_changed", map[string]interface{}{
						"worker_id": wr.ID,
						"status":    wr.Status,
					})
				}
			}
		}
		w.lastWorkers = currentWorkers
		w.mu.Unlock()
	}

	// Check gates
	var gatesState GatesState
	if data, err := os.ReadFile(filepath.Join(stateDir, "gates.json")); err == nil {
		json.Unmarshal(data, &gatesState)

		w.mu.Lock()
		for stage, gate := range gatesState.Gates {
			if lastGate, exists := w.lastGates[stage]; exists {
				if gate.Status != lastGate.Status {
					if gate.Status == "approved" {
						w.emitEvent("gate_approved", map[string]interface{}{
							"stage":      stage,
							"approved_at": gate.ApprovedAt,
						})
					} else if gate.Status == "ready" {
						w.emitEvent("gate_ready", map[string]interface{}{
							"stage":    stage,
							"criteria": gate.Criteria,
						})
					}
				}
			}
		}
		w.lastGates = gatesState.Gates
		w.mu.Unlock()
	}

	// Check for new findings
	w.checkFindings()
}

// checkFindings checks for new finding files
func (w *Watcher) checkFindings() {
	findingsDir := filepath.Join(w.missionDir, "findings")

	entries, err := os.ReadDir(findingsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// If file was modified in the last second, emit event
		if time.Since(info.ModTime()) < 1*time.Second {
			taskID := entry.Name()
			if len(taskID) > 5 {
				taskID = taskID[:len(taskID)-5] // Remove .json
			}
			w.emitEvent("findings_ready", map[string]interface{}{
				"task_id": taskID,
			})
		}
	}
}

// emitEvent sends an event to the events channel
func (w *Watcher) emitEvent(eventType string, data interface{}) {
	event := Event{
		Type: eventType,
		Data: data,
	}

	select {
	case w.events <- event:
	default:
		log.Printf("Watcher: event channel full, dropping event: %s", eventType)
	}
}

// GetCurrentState returns the current mission state
func (w *Watcher) GetCurrentState() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	tasks := make([]Task, 0, len(w.lastTasks))
	for _, t := range w.lastTasks {
		tasks = append(tasks, t)
	}

	workers := make([]Worker, 0, len(w.lastWorkers))
	for _, wr := range w.lastWorkers {
		workers = append(workers, wr)
	}

	return map[string]interface{}{
		"stage":   w.lastStage,
		"tasks":   tasks,
		"workers": workers,
		"gates":   w.lastGates,
	}
}

// readTasksJSONLFile reads tasks from a JSONL file (one JSON task per line).
func readTasksJSONLFile(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []Task
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var task Task
		if err := json.Unmarshal(line, &task); err != nil {
			continue // skip malformed lines in watcher
		}
		tasks = append(tasks, task)
	}
	return tasks, scanner.Err()
}
