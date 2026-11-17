package glossary

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestMatcherCorrectness(t *testing.T) {
	entries := []Entry{
		{ID: "1", Source: "Hero", Target: "ผู้กล้า", Note: "main protagonist", CaseSensitive: true},
		{ID: "2", Source: "Muffet", Target: "มัฟเฟต", Note: "spider boss", CaseSensitive: false},
		{ID: "3", Source: "Excalibur", Target: "เอ็กซ์คาลิเบอร์", CaseSensitive: true},
		{ID: "4", Source: "healing potion", Target: "โพชั่นฟื้นฟู", CaseSensitive: false},
		{ID: "5", Source: "dragon", Target: "มังกร", CaseSensitive: false},
	}

	matcher := NewMatcher(entries)

	tests := []struct {
		name          string
		texts         []string
		expectedTerms []string
	}{
		{
			name:          "Match case-insensitive Muffet and Healing Potion",
			texts:         []string{"Look at muffet over there!", "Drink this HEALING POTION now."},
			expectedTerms: []string{"Muffet", "healing potion"},
		},
		{
			name:          "Match case-sensitive Hero but not hero",
			texts:         []string{"The Hero arrives.", "A true hero never dies."},
			expectedTerms: []string{"Hero"},
		},
		{
			name:          "No match returns empty",
			texts:         []string{"Just an ordinary day in town."},
			expectedTerms: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := matcher.MatchBatch(tc.texts)
			if len(tc.expectedTerms) == 0 {
				if len(matched) != 0 {
					t.Fatalf("expected 0 matches, got %d", len(matched))
				}
				return
			}

			matchedMap := make(map[string]bool)
			for _, m := range matched {
				matchedMap[m.Source] = true
			}

			for _, expected := range tc.expectedTerms {
				if !matchedMap[expected] {
					t.Errorf("expected term %q not found in matches: %v", expected, matched)
				}
			}
		})
	}
}

// TestTokenReductionMetrics verifies that filtering a large 500-term glossary
// down to matched terms in a batch reduces token overhead by > 80%.
func TestTokenReductionMetrics(t *testing.T) {
	// Generate 500 realistic game terms
	allEntries := make([]Entry, 500)
	for i := 0; i < 500; i++ {
		allEntries[i] = Entry{
			ID:           fmt.Sprintf("term_%d", i),
			Source:       fmt.Sprintf("CharacterOrSkill_%d", i),
			Target:       fmt.Sprintf("คำแปล_%d", i),
			Note:         "fictional game entity",
			WordBoundary: true,
		}
	}

	// Pick 3 terms to appear in a typical 10-line dialogue batch
	batchTexts := []string{
		"Hello there, CharacterOrSkill_42! Are you ready?",
		"I have equipped CharacterOrSkill_105 for the upcoming fight.",
		"Watch out for CharacterOrSkill_300!",
		"Let's move onward to the castle.",
		"Understood, leader.",
	}

	matcher := NewMatcher(allEntries)
	matched := matcher.MatchBatch(batchTexts)

	if len(matched) != 3 {
		t.Fatalf("expected exactly 3 matched terms, got %d", len(matched))
	}

	rawGlossaryPrompt := FormatPrompt(allEntries)
	filteredGlossaryPrompt := FormatPrompt(matched)

	rawTokens := ApproximateTokens(rawGlossaryPrompt)
	filteredTokens := ApproximateTokens(filteredGlossaryPrompt)

	reductionPct := (1.0 - float64(filteredTokens)/float64(rawTokens)) * 100.0

	t.Logf("Full Glossary Tokens:     %d tokens (%d terms)", rawTokens, len(allEntries))
	t.Logf("Filtered Glossary Tokens: %d tokens (%d terms)", filteredTokens, len(matched))
	t.Logf("Token Reduction:          %.2f%%", reductionPct)

	if reductionPct < 80.0 {
		t.Errorf("Expected token reduction > 80%%, got %.2f%%", reductionPct)
	}
}

// BenchmarkGlossaryMatch benchmarks the Aho-Corasick matching speed
// against 1,000 glossary terms across 50 text lines.
func BenchmarkGlossaryMatch(b *testing.B) {
	// Setup 1,000 glossary entries
	entries := make([]Entry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = Entry{
			ID:            fmt.Sprintf("term_%d", i),
			Source:        fmt.Sprintf("Character_%d", i),
			Target:        fmt.Sprintf("ตัวละคร_%d", i),
			Note:          "important character",
			CaseSensitive: i%2 == 0,
		}
	}

	matcher := NewMatcher(entries)

	// Sample 50 dialogue lines
	rng := rand.New(rand.NewSource(42))
	texts := make([]string, 50)
	for i := 0; i < 50; i++ {
		texts[i] = fmt.Sprintf("Line %d: Welcome to the realm of Character_%d, traveler! May Character_%d guide you.",
			i, rng.Intn(1000), rng.Intn(1000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matched := matcher.MatchBatch(texts)
		if len(matched) == 0 {
			b.Fatal("expected matches")
		}
	}
}
