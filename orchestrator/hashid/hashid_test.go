package hashid

import "testing"

func TestGenerate(t *testing.T) {
	// Deterministic: same input → same output
	id1 := Generate("task", "build-frontend")
	id2 := Generate("task", "build-frontend")
	if id1 != id2 {
		t.Errorf("expected deterministic output, got %s and %s", id1, id2)
	}

	// Correct length (10 hex chars)
	if len(id1) != 10 {
		t.Errorf("expected 10 chars, got %d: %s", len(id1), id1)
	}

	// Different input → different output
	id3 := Generate("task", "deploy-backend")
	if id1 == id3 {
		t.Errorf("expected different IDs for different content")
	}
}
