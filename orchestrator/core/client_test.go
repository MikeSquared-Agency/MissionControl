package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// skipIfNoMcCore skips the test if mc-core is not available
func skipIfNoMcCore(t *testing.T) {
	if _, err := findMcCore(); err != nil {
		t.Skip("mc-core not found, skipping test")
	}
}

func TestCountTokens(t *testing.T) {
	skipIfNoMcCore(t)

	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "simple text",
			text:    "Hello world",
			wantMin: 2,
			wantMax: 3,
		},
		{
			name:    "longer text",
			text:    "The quick brown fox jumps over the lazy dog.",
			wantMin: 8,
			wantMax: 12,
		},
		{
			name:    "empty text",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := CountTokens(tt.text)
			if err != nil {
				t.Fatalf("CountTokens() error = %v", err)
			}
			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("CountTokens() = %v, want between %v and %v", count, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestValidateHandoff(t *testing.T) {
	skipIfNoMcCore(t)

	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "handoff-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("valid handoff", func(t *testing.T) {
		validHandoff := `{
			"task_id": "task-1",
			"worker_id": "worker-1",
			"status": "complete",
			"findings": [{"finding_type": "discovery", "summary": "Found something"}],
			"artifacts": [],
			"open_questions": [],
			"timestamp": 1234567890
		}`

		path := filepath.Join(tmpDir, "valid.json")
		if err := os.WriteFile(path, []byte(validHandoff), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result, err := ValidateHandoff(path)
		if err != nil {
			t.Fatalf("ValidateHandoff() error = %v", err)
		}
		if !result.Valid {
			t.Errorf("ValidateHandoff() valid = false, want true; errors: %v", result.Errors)
		}
	})

	t.Run("invalid handoff - missing task_id", func(t *testing.T) {
		invalidHandoff := `{
			"task_id": "",
			"worker_id": "worker-1",
			"status": "complete",
			"findings": [],
			"artifacts": [],
			"open_questions": [],
			"timestamp": 1234567890
		}`

		path := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(path, []byte(invalidHandoff), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result, err := ValidateHandoff(path)
		if err != nil {
			t.Fatalf("ValidateHandoff() error = %v", err)
		}
		if result.Valid {
			t.Errorf("ValidateHandoff() valid = true, want false")
		}
	})
}

func TestCheckGate(t *testing.T) {
	skipIfNoMcCore(t)

	// Create a temp mission directory
	tmpDir, err := os.MkdirTemp("", "gate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("check gate without state file", func(t *testing.T) {
		result, err := CheckGate("discovery", tmpDir)
		if err != nil {
			t.Fatalf("CheckGate() error = %v", err)
		}
		if result.Stage != "discovery" {
			t.Errorf("CheckGate() stage = %v, want discovery", result.Stage)
		}
		// Default gate status is "closed" when no state file exists
		if result.Status != "closed" {
			t.Errorf("CheckGate() status = %v, want closed", result.Status)
		}
	})
}

func TestFindMcCore(t *testing.T) {
	// This test verifies findMcCore works when mc-core exists
	path, err := findMcCore()
	if err != nil {
		// Not necessarily an error - mc-core might not be installed
		t.Logf("findMcCore() returned error (expected if not installed): %v", err)
		return
	}

	// Verify the path exists and is executable
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("findMcCore() returned path that doesn't exist: %v", path)
		return
	}

	if info.IsDir() {
		t.Errorf("findMcCore() returned a directory, not a file: %v", path)
	}

	// Try to execute it with --help to verify it's the right binary
	cmd := exec.Command(path, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("mc-core --help failed: %v", err)
		return
	}

	if len(output) == 0 {
		t.Error("mc-core --help produced no output")
	}
}
