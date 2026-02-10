package main

import (
	"fmt"
	"path/filepath"
)

// validateStrict runs Phase 1 strict checks on top of base validation.
// Returns a slice of human-readable failure messages (empty = pass).
func validateStrict(missionDir string) []string {
	var failures []string

	var stageState StageState
	if err := readJSON(filepath.Join(missionDir, "state", "stage.json"), &stageState); err != nil {
		return []string{fmt.Sprintf("cannot read stage state: %v", err)}
	}

	tasks, err := loadTasks(missionDir)
	if err != nil {
		return []string{fmt.Sprintf("cannot load tasks: %v", err)}
	}

	var stageTasks []Task
	for _, t := range tasks {
		if t.Stage == stageState.Current {
			stageTasks = append(stageTasks, t)
		}
	}

	if stageState.Current == "verify" {
		failures = append(failures, checkVerifyPersonaCoverage(stageTasks)...)
	}

	if stageState.Current == "implement" && len(stageTasks) > 1 {
		failures = append(failures, checkIntegratorPresent(stageTasks)...)
	}

	return failures
}

// checkVerifyPersonaCoverage ensures verify stage has done tasks for
// reviewer, security, and tester personas.
func checkVerifyPersonaCoverage(stageTasks []Task) []string {
	required := []string{"reviewer", "security", "tester"}
	var failures []string

	for _, persona := range required {
		found := false
		for _, t := range stageTasks {
			if t.Persona == persona && t.Status == "done" {
				found = true
				break
			}
		}
		if !found {
			failures = append(failures, fmt.Sprintf(
				"verify stage requires a done task with persona %q — none found", persona))
		}
	}
	return failures
}

// checkIntegratorPresent ensures multi-task implement stages have
// at least one task with persona "integrator".
func checkIntegratorPresent(stageTasks []Task) []string {
	for _, t := range stageTasks {
		if t.Persona == "integrator" && t.Status == "done" {
			return nil
		}
	}
	// Check if integrator exists but isn't done
	for _, t := range stageTasks {
		if t.Persona == "integrator" {
			return []string{"implement stage has an integrator task but it is not done — complete it before gating"}
		}
	}
	return []string{"implement stage has multiple tasks but no integrator task — add a task with persona \"integrator\""}
}
