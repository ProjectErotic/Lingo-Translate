package decision

import (
	"context"
	"time"
)

// PrimitiveType identifies the decision primitive used
type PrimitiveType string

const (
	PrimitiveNoul   PrimitiveType = "noul"   // Boolean Yes/No with calibrated probability
	PrimitiveChoice PrimitiveType = "choice" // Selection among discrete options
	PrimitiveScore  PrimitiveType = "score"  // Continuous rating (0.0 to 1.0)
)

// DecisionResult represents the outcome of a decision evaluation
type DecisionResult struct {
	Primitive  PrimitiveType `json:"primitive"`
	BoolValue  bool          `json:"bool_value,omitempty"`
	Choice     string        `json:"choice,omitempty"`
	Score      float64       `json:"score,omitempty"`
	Confidence float64       `json:"confidence"`
	Source     string        `json:"source"` // "cache", "jev", "heuristic"
	Latency    time.Duration `json:"latency"`
}

// DecisionEngine is the universal contract for non-autoregressive System One models
type DecisionEngine interface {
	Name() string

	// Noul evaluates a boolean question on input text (returns true/false with confidence)
	Noul(ctx context.Context, input string, question string) (bool, float64, error)

	// Choice selects the most accurate category from a given list of choices
	Choice(ctx context.Context, input string, choices []string) (string, float64, error)

	// Score evaluates input against criteria and returns a score from 0.0 to 1.0
	Score(ctx context.Context, input string, criteria string) (float64, error)
}
