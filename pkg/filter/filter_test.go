package filter

import (
	"path/filepath"
	"testing"

	"lingo-translate/pkg/decision"
)

func TestSmartFilter(t *testing.T) {
	mgr := New()

	// Default engine skips
	if !mgr.ShouldSkip("http://example.com/asset.png") {
		t.Errorf("Expected URL to be skipped")
	}
	if !mgr.ShouldSkip("Actor1_2") {
		t.Errorf("Expected system prefix to be skipped")
	}
	if !mgr.ShouldSkip("12345.67") {
		t.Errorf("Expected number to be skipped")
	}
	if !mgr.ShouldSkip("---===---") {
		t.Errorf("Expected symbol only to be skipped")
	}

	// Normal dialogue should NOT be skipped
	if mgr.ShouldSkip("Hello traveler! Welcome to our village.") {
		t.Errorf("Dialogue should not be skipped")
	}

	// Learn custom pattern
	mgr.Learn(`^DEBUG_.*`)
	if !mgr.ShouldSkip("DEBUG_TEST_STRING") {
		t.Errorf("Learned pattern should be skipped")
	}

	// Unlearn custom pattern
	mgr.Unlearn(`^DEBUG_.*`)
	if mgr.ShouldSkip("DEBUG_TEST_STRING") {
		t.Errorf("Unlearned pattern should not be skipped")
	}

	// Export and Import rules
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "rules.json")

	mgr.Learn("CUSTOM_TOKEN_1")
	if err := mgr.ExportRules(rulesFile); err != nil {
		t.Fatalf("ExportRules failed: %v", err)
	}

	mgr2 := New()
	if err := mgr2.ImportRules(rulesFile); err != nil {
		t.Fatalf("ImportRules failed: %v", err)
	}
	if !mgr2.ShouldSkip("CUSTOM_TOKEN_1") {
		t.Errorf("Imported rule should be skipped")
	}
}

func TestFilterWithDecisionEngine(t *testing.T) {
	mgr := New()
	engine := decision.NewHeuristicEngine()
	mgr.SetDecisionEngine(engine, true)

	// 1. Ambiguous code string should be skipped by decision engine
	if !mgr.ShouldSkip("actorAttackSlash") {
		t.Errorf("Expected ambiguous code identifier 'actorAttackSlash' to be skipped")
	}

	// 2. Player-facing text in gray-zone should NOT be skipped
	if mgr.ShouldSkip("Attack +10% upon critical hit!") {
		t.Errorf("Expected player-facing buff text to NOT be skipped")
	}

	// 3. Thai text should never be skipped as code
	if mgr.ShouldSkip("สวัสดีนักเดินทาง เจ้าต้องการอะไร") {
		t.Errorf("Expected Thai text to NOT be skipped")
	}

	// 4. When decision engine is disabled, regular heuristic runs
	mgr.SetDecisionEngine(engine, false)
	// Without decision engine, single camelCase might pass if not caught by regex
	_ = mgr.ShouldSkip("actorAttackSlash")
}
