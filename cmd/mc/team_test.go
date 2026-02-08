package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupMissionDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mc-team-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	missionDir := filepath.Join(tmpDir, ".mission")
	os.MkdirAll(filepath.Join(missionDir, "state"), 0755)
	os.MkdirAll(filepath.Join(missionDir, "prompts"), 0755)

	return tmpDir
}

func writeTestConfig(t *testing.T, dir string, cfg Config) {
	t.Helper()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, ".mission", "config.json"), data, 0644)
}

func TestTeamInConfig(t *testing.T) {
	cfg := Config{
		Version: "1.0.0",
		Teams: map[string]Team{
			"backend-crew": {
				Personas: []string{"developer", "reviewer", "tester"},
				Zone:     "backend",
			},
			"research": {
				Personas: []string{"researcher"},
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if len(decoded.Teams) != 2 {
		t.Errorf("Expected 2 teams, got %d", len(decoded.Teams))
	}

	bc := decoded.Teams["backend-crew"]
	if bc.Zone != "backend" {
		t.Errorf("Expected zone 'backend', got %q", bc.Zone)
	}
	if len(bc.Personas) != 3 {
		t.Errorf("Expected 3 personas, got %d", len(bc.Personas))
	}

	r := decoded.Teams["research"]
	if r.Zone != "" {
		t.Errorf("Expected empty zone, got %q", r.Zone)
	}
}

func TestLoadConfigWithTeams(t *testing.T) {
	tmpDir := setupMissionDir(t)
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Version: "1.0.0",
		Teams: map[string]Team{
			"full-stack": {
				Personas: []string{"developer", "tester"},
				Zone:     "frontend",
			},
		},
	}
	writeTestConfig(t, tmpDir, cfg)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	missionDir := filepath.Join(tmpDir, ".mission")
	loaded, err := loadConfig(missionDir)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	team, ok := loaded.Teams["full-stack"]
	if !ok {
		t.Fatal("Expected 'full-stack' team in loaded config")
	}
	if team.Zone != "frontend" {
		t.Errorf("Expected zone 'frontend', got %q", team.Zone)
	}
	if len(team.Personas) != 2 {
		t.Errorf("Expected 2 personas, got %d", len(team.Personas))
	}
}

func TestTeamListNoTeams(t *testing.T) {
	tmpDir := setupMissionDir(t)
	defer os.RemoveAll(tmpDir)

	writeTestConfig(t, tmpDir, Config{Version: "1.0.0"})

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Should not error
	err := runTeamList(nil, nil)
	if err != nil {
		t.Fatalf("runTeamList failed: %v", err)
	}
}

func TestTeamSpawnUnknown(t *testing.T) {
	tmpDir := setupMissionDir(t)
	defer os.RemoveAll(tmpDir)

	writeTestConfig(t, tmpDir, Config{
		Version: "1.0.0",
		Teams:   map[string]Team{},
	})

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	err := runTeamSpawn(teamSpawnCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("Expected error for unknown team")
	}
}
