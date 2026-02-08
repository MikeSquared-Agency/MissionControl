package tokens

import (
	"sync"
)

type ModelTier string

const (
	ModelOpus   ModelTier = "opus"
	ModelSonnet ModelTier = "sonnet"
	ModelHaiku  ModelTier = "haiku"
)

// Cost per million tokens (input/output) in USD
type ModelCost struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

var ModelCosts = map[ModelTier]ModelCost{
	ModelOpus:   {15.0, 75.0},
	ModelSonnet: {3.0, 15.0},
	ModelHaiku:  {0.25, 1.25},
}

// Default model for each persona
var PersonaModels = map[string]ModelTier{
	"king":         ModelOpus,
	"researcher":   ModelSonnet,
	"analyst":      ModelSonnet,
	"requirements": ModelSonnet,
	"architect":    ModelSonnet,
	"designer":     ModelSonnet,
	"developer":    ModelSonnet,
	"debugger":     ModelSonnet,
	"security":     ModelSonnet,
	"reviewer":     ModelHaiku,
	"tester":       ModelHaiku,
	"qa":           ModelHaiku,
	"docs":         ModelHaiku,
	"devops":       ModelHaiku,
}

type SessionTokens struct {
	WorkerID      string    `json:"worker_id"`
	Persona       string    `json:"persona"`
	Model         ModelTier `json:"model"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	EstimatedCost float64   `json:"estimated_cost_usd"`
}

type TokenSummary struct {
	TotalTokens     int             `json:"total_tokens"`
	TotalCost       float64         `json:"total_cost_usd"`
	Sessions        []SessionTokens `json:"sessions"`
	BudgetLimit     int             `json:"budget_limit"`
	BudgetUsed      int             `json:"budget_used"`
	BudgetRemaining int             `json:"budget_remaining"`
}

type BudgetWarningCallback func(workerID string, budget, used, remaining int)

type Accumulator struct {
	sessions map[string]*SessionTokens
	total    SessionTokens
	budget   int // 0 = no budget
	callback BudgetWarningCallback
	mu       sync.RWMutex
}

func NewAccumulator(budget int, callback BudgetWarningCallback) *Accumulator {
	return &Accumulator{
		sessions: make(map[string]*SessionTokens),
		budget:   budget,
		callback: callback,
	}
}

func (a *Accumulator) Record(workerID, persona string, model ModelTier, inputTokens, outputTokens int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cost := EstimateCost(model, inputTokens, outputTokens)

	sess, ok := a.sessions[workerID]
	if !ok {
		sess = &SessionTokens{
			WorkerID: workerID,
			Persona:  persona,
			Model:    model,
		}
		a.sessions[workerID] = sess
	}

	sess.InputTokens += inputTokens
	sess.OutputTokens += outputTokens
	sess.TotalTokens = sess.InputTokens + sess.OutputTokens
	sess.EstimatedCost += cost

	a.total.InputTokens += inputTokens
	a.total.OutputTokens += outputTokens
	a.total.TotalTokens = a.total.InputTokens + a.total.OutputTokens
	a.total.EstimatedCost += cost

	// Budget warnings
	if a.budget > 0 && a.callback != nil {
		used := a.total.TotalTokens
		remaining := a.budget - used
		if remaining < 0 {
			remaining = 0
		}
		threshold80 := int(float64(a.budget) * 0.8)
		prev := used - inputTokens - outputTokens
		if prev < threshold80 && used >= threshold80 {
			a.callback(workerID, a.budget, used, remaining)
		}
		if prev < a.budget && used >= a.budget {
			a.callback(workerID, a.budget, used, remaining)
		}
	}
}

func (a *Accumulator) RecordText(workerID, persona string, model ModelTier, text string) {
	tokens := len(text) / 4
	a.Record(workerID, persona, model, tokens, 0)
}

func (a *Accumulator) GetSession(workerID string) (*SessionTokens, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	sess, ok := a.sessions[workerID]
	if !ok {
		return nil, false
	}
	copy := *sess
	return &copy, true
}

func (a *Accumulator) Summary() TokenSummary {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sessions := make([]SessionTokens, 0, len(a.sessions))
	for _, s := range a.sessions {
		sessions = append(sessions, *s)
	}

	used := a.total.TotalTokens
	remaining := a.budget - used
	if remaining < 0 {
		remaining = 0
	}

	return TokenSummary{
		TotalTokens:     a.total.TotalTokens,
		TotalCost:       a.total.EstimatedCost,
		Sessions:        sessions,
		BudgetLimit:     a.budget,
		BudgetUsed:      used,
		BudgetRemaining: remaining,
	}
}

func (a *Accumulator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions = make(map[string]*SessionTokens)
	a.total = SessionTokens{}
}

func ModelForPersona(persona string) ModelTier {
	if model, ok := PersonaModels[persona]; ok {
		return model
	}
	return ModelSonnet // default
}

func EstimateCost(model ModelTier, inputTokens, outputTokens int) float64 {
	cost, ok := ModelCosts[model]
	if !ok {
		return 0
	}
	return (float64(inputTokens)/1_000_000)*cost.InputPerMTok +
		(float64(outputTokens)/1_000_000)*cost.OutputPerMTok
}
