package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindMissionDirFollowsSymlink tests that findMissionDir resolves symlinks
func TestFindMissionDirFollowsSymlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-symlink-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create real .mission dir in a separate location
	realMission := filepath.Join(tmpDir, "shared", ".mission")
	os.MkdirAll(filepath.Join(realMission, "state"), 0755)
	writeJSON(filepath.Join(realMission, "config.json"), map[string]string{"version": "1.0.0"})

	// Create project dir with symlink
	projectDir := filepath.Join(tmpDir, "my-project")
	os.MkdirAll(projectDir, 0755)
	os.Symlink(realMission, filepath.Join(projectDir, ".mission"))

	// Change to project dir
	originalDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(originalDir)

	// Reset project flag
	projectFlag = ""

	found, err := findMissionDir()
	if err != nil {
		t.Fatalf("findMissionDir failed: %v", err)
	}

	// Should resolve to the real path
	expectedResolved, _ := filepath.EvalSymlinks(realMission)
	if found != expectedResolved {
		t.Errorf("Expected resolved path %s, got %s", expectedResolved, found)
	}
}

// TestFindMissionDirWithProjectFlag tests --project flag lookup
func TestFindMissionDirWithProjectFlag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-project-flag-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a real .mission dir
	realMission := filepath.Join(tmpDir, "my-project", ".mission")
	os.MkdirAll(filepath.Join(realMission, "state"), 0755)
	writeJSON(filepath.Join(realMission, "config.json"), map[string]string{"version": "1.0.0"})

	// Override registry path for test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Register the project
	reg := &ProjectRegistry{
		Projects: map[string]string{
			"test-proj": realMission,
		},
	}
	if err := saveRegistry(reg); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Set flag and find
	projectFlag = "test-proj"
	defer func() { projectFlag = "" }()

	found, err := findMissionDir()
	if err != nil {
		t.Fatalf("findMissionDir with --project failed: %v", err)
	}

	expectedResolved, _ := filepath.EvalSymlinks(realMission)
	if found != expectedResolved {
		t.Errorf("Expected %s, got %s", expectedResolved, found)
	}
}

// TestFindMissionDirProjectNotFound tests error for unknown project name
func TestFindMissionDirProjectNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-project-notfound-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	projectFlag = "nonexistent"
	defer func() { projectFlag = "" }()

	_, err = findMissionDir()
	if err == nil {
		t.Error("Expected error for nonexistent project")
	}
}

// TestProjectRegistryRoundTrip tests save/load of registry
func TestProjectRegistryRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-registry-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Save
	reg := &ProjectRegistry{
		Projects: map[string]string{
			"proj-a": "/path/to/a/.mission",
			"proj-b": "/path/to/b/.mission",
		},
	}
	if err := saveRegistry(reg); err != nil {
		t.Fatalf("saveRegistry failed: %v", err)
	}

	// Load
	loaded, err := loadRegistry()
	if err != nil {
		t.Fatalf("loadRegistry failed: %v", err)
	}

	if len(loaded.Projects) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(loaded.Projects))
	}

	if loaded.Projects["proj-a"] != "/path/to/a/.mission" {
		t.Errorf("proj-a path mismatch: %s", loaded.Projects["proj-a"])
	}
}

// TestProjectLink tests the symlink creation command
func TestProjectLink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-link-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create target .mission dir
	target := filepath.Join(tmpDir, "shared-mission")
	os.MkdirAll(target, 0755)

	// Create project dir
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(projectDir, 0755)

	originalDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(originalDir)

	// Run link command
	err = projectLinkCmd.RunE(nil, []string{target})
	if err != nil {
		t.Fatalf("project link failed: %v", err)
	}

	// Verify symlink exists
	linkPath := filepath.Join(projectDir, ".mission")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Symlink not created: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected symlink, got regular file/dir")
	}

	// Verify it points to the right place
	dest, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	if dest != target {
		t.Errorf("Symlink points to %s, expected %s", dest, target)
	}
}

// TestProjectLinkAlreadyExists tests that link fails if .mission exists
func TestProjectLinkAlreadyExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-link-exists-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	target := filepath.Join(tmpDir, "shared-mission")
	os.MkdirAll(target, 0755)

	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(filepath.Join(projectDir, ".mission"), 0755) // Already exists

	originalDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(originalDir)

	err = projectLinkCmd.RunE(nil, []string{target})
	if err == nil {
		t.Error("Expected error when .mission already exists")
	}
}
