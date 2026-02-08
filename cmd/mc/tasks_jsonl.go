package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	tasksJSONLFile = "tasks.jsonl"
	tasksJSONFile  = "tasks.json"
)

// tasksPath returns the path to tasks.jsonl in the given mission state dir.
func tasksPath(missionDir string) string {
	return filepath.Join(missionDir, "state", tasksJSONLFile)
}

// readTasksJSONL reads tasks from a JSONL file (one JSON task per line).
func readTasksJSONL(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []Task
	scanner := bufio.NewScanner(f)
	// Increase buffer for potentially large lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var task Task
		if err := json.Unmarshal(line, &task); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		tasks = append(tasks, task)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// writeTasksJSONL writes tasks to a JSONL file (one JSON task per line).
func writeTasksJSONL(path string, tasks []Task) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".tasks-*.jsonl")
	if err != nil {
		return err
	}
	tmpPath := f.Name()

	w := bufio.NewWriter(f)
	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			f.Close()
			os.Remove(tmpPath)
			return err
		}
		_, _ = w.Write(data)
		_ = w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// loadTasks loads tasks from tasks.jsonl, auto-migrating from tasks.json if needed.
func loadTasks(missionDir string) ([]Task, error) {
	jsonlPath := tasksPath(missionDir)
	jsonPath := filepath.Join(missionDir, "state", tasksJSONFile)

	// If tasks.jsonl exists, use it
	if _, err := os.Stat(jsonlPath); err == nil {
		return readTasksJSONL(jsonlPath)
	}

	// If tasks.json exists but tasks.jsonl doesn't, migrate
	if _, err := os.Stat(jsonPath); err == nil {
		tasks, err := migrateTasksJSONToJSONL(jsonPath, jsonlPath)
		if err != nil {
			return nil, fmt.Errorf("migration from tasks.json failed: %w", err)
		}
		return tasks, nil
	}

	// Neither exists — return empty
	return []Task{}, nil
}

// saveTasks saves tasks to tasks.jsonl.
func saveTasks(missionDir string, tasks []Task) error {
	return writeTasksJSONL(tasksPath(missionDir), tasks)
}

// migrateTasksJSONToJSONL reads tasks.json, writes tasks.jsonl, returns the tasks.
func migrateTasksJSONToJSONL(jsonPath, jsonlPath string) ([]Task, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var state TasksState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("invalid tasks.json: %w", err)
	}

	if err := writeTasksJSONL(jsonlPath, state.Tasks); err != nil {
		return nil, err
	}

	// Rename old file so we don't migrate again
	_ = os.Rename(jsonPath, jsonPath+".migrated")

	return state.Tasks, nil
}
