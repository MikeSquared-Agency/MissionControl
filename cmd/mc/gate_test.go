package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to create a test GatesFile with sample criteria
func testGatesFile() GatesFile {
	return GatesFile{
		Gates: map[string]StageGate{
			"implement": {
				Criteria: []GateCriterion{
					{Description: "All unit tests pass", Satisfied: false},
					{Description: "Code review approved", Satisfied: false},
					{Description: "No lint errors", Satisfied: false},
				},
			},
		},
	}
}

func TestSaveAndLoadGates(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gf := testGatesFile()
	if err := saveGates(tmp, gf); err != nil {
		t.Fatalf("saveGates: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(stateDir, "gates.json"))
	if err != nil {
		t.Fatalf("gates.json not written: %v", err)
	}

	// Verify valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("gates.json is not valid JSON: %v", err)
	}

	// Load back
	loaded, err := loadGates(tmp)
	if err != nil {
		t.Fatalf("loadGates: %v", err)
	}

	if len(loaded.Gates) != len(gf.Gates) {
		t.Errorf("expected %d stages, got %d", len(gf.Gates), len(loaded.Gates))
	}

	impl, ok := loaded.Gates["implement"]
	if !ok {
		t.Fatal("missing 'implement' stage in loaded gates")
	}
	if len(impl.Criteria) != 3 {
		t.Errorf("expected 3 criteria, got %d", len(impl.Criteria))
	}
	if impl.Criteria[0].Description != "All unit tests pass" {
		t.Errorf("unexpected description: %s", impl.Criteria[0].Description)
	}
	if impl.Criteria[0].Satisfied != false {
		t.Error("expected criterion to be unsatisfied")
	}
}

func TestSatisfyCriterion_ExactMatch(t *testing.T) {
	gf := testGatesFile()
	desc, err := satisfyCriterion(&gf, "implement", "All unit tests pass")
	if err != nil {
		t.Fatalf("satisfyCriterion: %v", err)
	}
	if desc != "All unit tests pass" {
		t.Errorf("expected exact description returned, got: %s", desc)
	}

	// Verify the criterion is now satisfied
	for _, c := range gf.Gates["implement"].Criteria {
		if c.Description == "All unit tests pass" && !c.Satisfied {
			t.Error("criterion should be satisfied after satisfyCriterion call")
		}
	}
}

func TestSatisfyCriterion_SubstringMatch(t *testing.T) {
	gf := testGatesFile()
	desc, err := satisfyCriterion(&gf, "implement", "unit tests")
	if err != nil {
		t.Fatalf("satisfyCriterion: %v", err)
	}
	if desc != "All unit tests pass" {
		t.Errorf("expected 'All unit tests pass', got: %s", desc)
	}

	for _, c := range gf.Gates["implement"].Criteria {
		if c.Description == "All unit tests pass" && !c.Satisfied {
			t.Error("criterion should be satisfied after substring match")
		}
	}
}

func TestSatisfyCriterion_NoMatch(t *testing.T) {
	gf := testGatesFile()
	_, err := satisfyCriterion(&gf, "implement", "nonexistent criterion xyz")
	if err == nil {
		t.Fatal("expected error for no matching criterion, got nil")
	}
}

func TestSatisfyCriterion_AmbiguousMatch(t *testing.T) {
	gf := GatesFile{
		Gates: map[string]StageGate{
			"implement": {
				Criteria: []GateCriterion{
					{Description: "All unit tests pass", Satisfied: false},
					{Description: "All unit tests reviewed", Satisfied: false},
				},
			},
		},
	}
	_, err := satisfyCriterion(&gf, "implement", "unit tests")
	if err == nil {
		t.Fatal("expected error for ambiguous match, got nil")
	}
}

func TestSatisfyCriterion_AlreadySatisfied(t *testing.T) {
	gf := GatesFile{
		Gates: map[string]StageGate{
			"implement": {
				Criteria: []GateCriterion{
					{Description: "All unit tests pass", Satisfied: true},
				},
			},
		},
	}
	desc, err := satisfyCriterion(&gf, "implement", "All unit tests pass")
	if err != nil {
		t.Fatalf("satisfyCriterion on already-satisfied should not error: %v", err)
	}
	if desc != "All unit tests pass" {
		t.Errorf("unexpected description: %s", desc)
	}
	if !gf.Gates["implement"].Criteria[0].Satisfied {
		t.Error("criterion should remain satisfied")
	}
}

func TestInitGateForStage(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := initGateForStage(tmp, "implement")
	if err != nil {
		t.Skipf("initGateForStage failed (mc-core may not be available): %v", err)
	}

	// Verify gates.json was created with criteria for the stage
	gf, err := loadGates(tmp)
	if err != nil {
		t.Fatalf("loadGates after initGateForStage: %v", err)
	}

	stage, ok := gf.Gates["implement"]
	if !ok {
		t.Fatal("expected 'implement' stage in gates after init")
	}
	if len(stage.Criteria) == 0 {
		t.Error("expected at least one criterion after initGateForStage")
	}
}

func TestLoadGates_LegacyFormat(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	os.MkdirAll(stateDir, 0o755)

	legacy := `{"gates":{"implement":{"stage":"implement","status":"pending","criteria":["All tests pass","Lint clean"]}}}`
	os.WriteFile(filepath.Join(stateDir, "gates.json"), []byte(legacy), 0o644)

	gf, err := loadGates(tmp)
	if err != nil {
		t.Fatalf("loadGates legacy: %v", err)
	}
	sg, ok := gf.Gates["implement"]
	if !ok {
		t.Fatal("missing implement stage")
	}
	if len(sg.Criteria) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(sg.Criteria))
	}
	if sg.Criteria[0].Description != "All tests pass" {
		t.Errorf("unexpected description: %s", sg.Criteria[0].Description)
	}
	if sg.Criteria[0].Satisfied {
		t.Error("legacy criteria should default to unsatisfied")
	}
}

func TestLoadGates_CorruptJSON(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	os.MkdirAll(stateDir, 0o755)
	os.WriteFile(filepath.Join(stateDir, "gates.json"), []byte("{not valid json!!!"), 0o644)

	_, err := loadGates(tmp)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestLoadGates_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	gf, err := loadGates(tmp)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if gf.Gates == nil {
		t.Fatal("expected initialized Gates map")
	}
	if len(gf.Gates) != 0 {
		t.Errorf("expected empty Gates map, got %d entries", len(gf.Gates))
	}
}

func TestLoadGates_NilGates(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	os.MkdirAll(stateDir, 0o755)
	os.WriteFile(filepath.Join(stateDir, "gates.json"), []byte(`{"gates": null}`), 0o644)

	gf, err := loadGates(tmp)
	if err != nil {
		t.Fatalf("loadGates: %v", err)
	}
	if gf.Gates == nil {
		t.Fatal("expected initialized Gates map, got nil")
	}
}

func TestAllCriteriaMet_EmptyCriteria(t *testing.T) {
	gf := GatesFile{Gates: map[string]StageGate{"empty": {Criteria: []GateCriterion{}}}}
	if allCriteriaMet(&gf, "empty") {
		t.Error("expected false for empty criteria list")
	}
}

func TestSatisfyCriterion_UnknownStage(t *testing.T) {
	gf := testGatesFile()
	_, err := satisfyCriterion(&gf, "nonexistent", "anything")
	if err == nil {
		t.Fatal("expected error for unknown stage")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestGateSatisfyAll(t *testing.T) {
	gf := testGatesFile()
	sg := gf.Gates["implement"]
	for i := range sg.Criteria {
		sg.Criteria[i].Satisfied = true
	}
	gf.Gates["implement"] = sg

	if !allCriteriaMet(&gf, "implement") {
		t.Error("expected all criteria met after bulk satisfy")
	}
}

func TestSatisfyCriterion_EmptySubstring(t *testing.T) {
	gf := testGatesFile()
	_, err := satisfyCriterion(&gf, "implement", "")
	if err == nil {
		t.Fatal("expected error for empty substring (matches all = ambiguous)")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got: %v", err)
	}
}

func TestAllCriteriaMet(t *testing.T) {
	// All unsatisfied
	gf := testGatesFile()
	if allCriteriaMet(&gf, "implement") {
		t.Error("expected false when criteria are unsatisfied")
	}

	// Satisfy all
	stage := gf.Gates["implement"]
	for i := range stage.Criteria {
		stage.Criteria[i].Satisfied = true
	}
	gf.Gates["implement"] = stage

	if !allCriteriaMet(&gf, "implement") {
		t.Error("expected true when all criteria are satisfied")
	}

	// Non-existent stage
	if allCriteriaMet(&gf, "nonexistent") {
		t.Error("expected false for non-existent stage")
	}
}
