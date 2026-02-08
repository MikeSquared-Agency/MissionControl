package tokens

import (
	"math"
	"sync"
	"testing"
)

func TestRecordUpdatesSessionAndTotal(t *testing.T) {
	acc := NewAccumulator(0, nil)
	acc.Record("w1", "developer", ModelSonnet, 1000, 500)

	sess, ok := acc.GetSession("w1")
	if !ok {
		t.Fatal("expected session for w1")
	}
	if sess.InputTokens != 1000 || sess.OutputTokens != 500 || sess.TotalTokens != 1500 {
		t.Fatalf("unexpected session tokens: %+v", sess)
	}

	summary := acc.Summary()
	if summary.TotalTokens != 1500 {
		t.Fatalf("expected total 1500, got %d", summary.TotalTokens)
	}
}

func TestCostEstimation(t *testing.T) {
	tests := []struct {
		model  ModelTier
		input  int
		output int
		want   float64
	}{
		{ModelOpus, 1_000_000, 1_000_000, 15.0 + 75.0},
		{ModelSonnet, 1_000_000, 1_000_000, 3.0 + 15.0},
		{ModelHaiku, 1_000_000, 1_000_000, 0.25 + 1.25},
	}
	for _, tt := range tests {
		got := EstimateCost(tt.model, tt.input, tt.output)
		if math.Abs(got-tt.want) > 0.001 {
			t.Errorf("EstimateCost(%s, %d, %d) = %f, want %f", tt.model, tt.input, tt.output, got, tt.want)
		}
	}
}

func TestModelForPersona(t *testing.T) {
	if ModelForPersona("king") != ModelOpus {
		t.Error("king should be opus")
	}
	if ModelForPersona("developer") != ModelSonnet {
		t.Error("developer should be sonnet")
	}
	if ModelForPersona("reviewer") != ModelHaiku {
		t.Error("reviewer should be haiku")
	}
	if ModelForPersona("unknown") != ModelSonnet {
		t.Error("unknown should default to sonnet")
	}
}

func TestBudgetWarning(t *testing.T) {
	var warnings []struct {
		workerID  string
		budget    int
		used      int
		remaining int
	}
	var mu sync.Mutex

	cb := func(workerID string, budget, used, remaining int) {
		mu.Lock()
		warnings = append(warnings, struct {
			workerID  string
			budget    int
			used      int
			remaining int
		}{workerID, budget, used, remaining})
		mu.Unlock()
	}

	acc := NewAccumulator(1000, cb)

	// Record 700 tokens — no warning
	acc.Record("w1", "developer", ModelSonnet, 500, 200)
	if len(warnings) != 0 {
		t.Fatalf("expected 0 warnings at 700, got %d", len(warnings))
	}

	// Record 200 more — crosses 80% (900 >= 800)
	acc.Record("w1", "developer", ModelSonnet, 100, 100)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning at 900, got %d", len(warnings))
	}

	// Record 200 more — crosses 100% (1100 >= 1000)
	acc.Record("w1", "developer", ModelSonnet, 100, 100)
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings at 1100, got %d", len(warnings))
	}
}

func TestSummary(t *testing.T) {
	acc := NewAccumulator(5000, nil)
	acc.Record("w1", "developer", ModelSonnet, 1000, 500)
	acc.Record("w2", "reviewer", ModelHaiku, 2000, 300)

	s := acc.Summary()
	if s.TotalTokens != 3800 {
		t.Errorf("expected 3800 total, got %d", s.TotalTokens)
	}
	if s.BudgetLimit != 5000 || s.BudgetUsed != 3800 || s.BudgetRemaining != 1200 {
		t.Errorf("budget fields wrong: %+v", s)
	}
	if len(s.Sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(s.Sessions))
	}
}

func TestReset(t *testing.T) {
	acc := NewAccumulator(0, nil)
	acc.Record("w1", "developer", ModelSonnet, 1000, 500)
	acc.Reset()

	s := acc.Summary()
	if s.TotalTokens != 0 || len(s.Sessions) != 0 {
		t.Error("reset should clear everything")
	}
	_, ok := acc.GetSession("w1")
	if ok {
		t.Error("session should not exist after reset")
	}
}

func TestMultipleSessionsAccumulate(t *testing.T) {
	acc := NewAccumulator(0, nil)
	acc.Record("w1", "developer", ModelSonnet, 1000, 500)
	acc.Record("w2", "tester", ModelHaiku, 2000, 1000)
	acc.Record("w1", "developer", ModelSonnet, 500, 200)

	s1, _ := acc.GetSession("w1")
	if s1.InputTokens != 1500 || s1.OutputTokens != 700 {
		t.Errorf("w1 should accumulate: %+v", s1)
	}

	s2, _ := acc.GetSession("w2")
	if s2.InputTokens != 2000 || s2.OutputTokens != 1000 {
		t.Errorf("w2 unexpected: %+v", s2)
	}

	summary := acc.Summary()
	if summary.TotalTokens != 5200 {
		t.Errorf("expected 5200 total, got %d", summary.TotalTokens)
	}
}

func TestRecordText(t *testing.T) {
	acc := NewAccumulator(0, nil)
	text := "hello world!!!!!" // 16 chars -> 4 tokens
	acc.RecordText("w1", "developer", ModelSonnet, text)

	sess, ok := acc.GetSession("w1")
	if !ok {
		t.Fatal("expected session")
	}
	if sess.InputTokens != 4 {
		t.Errorf("expected 4 input tokens, got %d", sess.InputTokens)
	}
}
