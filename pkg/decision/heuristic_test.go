package decision

import (
	"context"
	"testing"
)

func TestHeuristicEngine(t *testing.T) {
	h := NewHeuristicEngine()
	ctx := context.Background()

	// 1. Engine code identification (Noul)
	codeInputs := []string{
		"$gameVariables.setValue(1, 10);",
		"SceneManager.push(Scene_Skill);",
		"AudioManager.playBgm('battle')",
		"var x = 5;",
		"actorAttackSlash",
	}

	for _, input := range codeInputs {
		isPlayerText, conf, err := h.Noul(ctx, input, "Is this player-facing text or code?")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isPlayerText {
			t.Errorf("expected '%s' to be detected as code (false), got true", input)
		}
		if conf < 0.80 {
			t.Errorf("expected high confidence for code detection on '%s', got %f", input, conf)
		}
	}

	// 2. Player-facing dialogue identification (Noul)
	dialogueInputs := []string{
		"Hello, traveler! Where are you headed today?",
		"ยินดีต้อนรับสู่อาณาจักร",
		"魔王を倒す時が来た！",
		"Attack power increased by 10%!",
	}

	for _, input := range dialogueInputs {
		isPlayerText, conf, err := h.Noul(ctx, input, "Is this player-facing text or code?")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isPlayerText {
			t.Errorf("expected '%s' to be detected as player-facing text (true), got false", input)
		}
		if conf < 0.70 {
			t.Errorf("expected confidence >= 0.70 on '%s', got %f", input, conf)
		}
	}

	// 3. Choice Test
	choices := []string{"menu_ui", "narrative_dialogue"}
	choice, _, err := h.Choice(ctx, "Start Game", choices)
	if err != nil {
		t.Fatalf("Choice failed: %v", err)
	}
	if choice != "menu_ui" {
		t.Errorf("expected 'menu_ui' for 'Start Game', got '%s'", choice)
	}

	// 4. Score Test
	score, err := h.Score(ctx, "Valid translation target", "fidelity")
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score <= 0 {
		t.Errorf("expected positive score, got %f", score)
	}
}
