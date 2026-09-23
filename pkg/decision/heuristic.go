package decision

import (
	"context"
	"regexp"
	"strings"
	"unicode"
)

var (
	codePatternRegex = regexp.MustCompile(`(?i)(\$game|SceneManager|AudioManager|DataManager|\.prototype|\.setValue|\.getValue|\bfunction\b|\bconst\b|\bvar\b|\blet\b|===|!==|=>|;|\[\s*\])`)
	camelCaseNoSpace = regexp.MustCompile(`^[a-z]+[A-Z][a-zA-Z0-9_]*$`)
)

// HeuristicEngine provides deterministic, zero-cost fallback decisions
type HeuristicEngine struct{}

// NewHeuristicEngine creates a new deterministic rule-based decision engine
func NewHeuristicEngine() *HeuristicEngine {
	return &HeuristicEngine{}
}

func (h *HeuristicEngine) Name() string {
	return "heuristic"
}

// Noul evaluates boolean questions using deterministic heuristics
func (h *HeuristicEngine) Noul(ctx context.Context, input string, question string) (bool, float64, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false, 1.0, nil
	}

	qLower := strings.ToLower(question)

	// Case 1: Code vs Player-Facing Text question
	if strings.Contains(qLower, "code") || strings.Contains(qLower, "player") || strings.Contains(qLower, "display") {
		// If it matches clear code syntax
		if codePatternRegex.MatchString(trimmed) {
			return false, 0.95, nil // It is code -> Not player-facing text
		}
		if camelCaseNoSpace.MatchString(trimmed) && len(trimmed) > 4 {
			return false, 0.90, nil // Single camelCase identifier like actorAttackSlash
		}

		// Count spaces and words
		words := strings.Fields(trimmed)
		hasSpaces := len(words) > 1
		hasPunct := strings.ContainsAny(trimmed, ".!?,:;\"'()[]{}")

		// If it contains natural spaces or CJK/Thai script without code symbols -> natural display text
		for _, r := range trimmed {
			if unicode.Is(unicode.Thai, r) || unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
				return true, 0.95, nil
			}
		}

		if hasSpaces || hasPunct {
			return true, 0.85, nil
		}

		// Single word without spaces: check if all uppercase or lowercase code
		if strings.ToUpper(trimmed) == trimmed && len(trimmed) > 3 && !strings.ContainsAny(trimmed, "AEIOU") {
			return false, 0.85, nil // e.g. CONST_NAME or EV001
		}

		return true, 0.70, nil
	}

	// Case 2: Draft acceptable question (Speculative Acceptor)
	if strings.Contains(qLower, "accurate") || strings.Contains(qLower, "acceptable") || strings.Contains(qLower, "draft") {
		// For heuristic, if draft is non-empty and reasonably sized
		return true, 0.85, nil
	}

	// Default fallback
	return true, 0.50, nil
}

// Choice selects the best matching option based on content heuristics
func (h *HeuristicEngine) Choice(ctx context.Context, input string, choices []string) (string, float64, error) {
	if len(choices) == 0 {
		return "", 0.0, nil
	}
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return choices[0], 0.5, nil
	}

	// Simple heuristic classification
	isShort := len(trimmed) <= 30
	for _, c := range choices {
		cLower := strings.ToLower(c)
		if isShort && (strings.Contains(cLower, "ui") || strings.Contains(cLower, "menu") || strings.Contains(cLower, "item")) {
			return c, 0.85, nil
		}
		if !isShort && (strings.Contains(cLower, "dialogue") || strings.Contains(cLower, "narrative")) {
			return c, 0.85, nil
		}
	}

	return choices[0], 0.60, nil
}

// Score evaluates input against criteria and returns a heuristic score (0.0 to 1.0)
func (h *HeuristicEngine) Score(ctx context.Context, input string, criteria string) (float64, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0.0, nil
	}

	// Baseline score for non-empty string
	return 0.85, nil
}
