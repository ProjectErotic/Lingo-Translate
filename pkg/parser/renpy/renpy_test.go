package renpy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenpyExtractAndInject(t *testing.T) {
	tempGame := t.TempDir()
	gameDir := filepath.Join(tempGame, "game")
	_ = os.MkdirAll(gameDir, 0755)

	scriptContent := `# Test script
define s = Character("Sylvie")

label start:
    s "Hi there! How are you doing today?"
    menu:
        "I am doing great!":
            jump great
        "Could be better.":
            jump ok
`
	scriptPath := filepath.Join(gameDir, "script.rpy")
	_ = os.WriteFile(scriptPath, []byte(scriptContent), 0644)

	parser := New()

	// 1. Detect
	if !parser.Detect(tempGame) {
		t.Fatalf("Failed to detect Ren'Py project")
	}

	// 2. Extract
	ctx := context.Background()
	entries, stats, err := parser.Extract(ctx, tempGame)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries (1 dialogue + 2 menu choices), got %d", len(entries))
	}

	if entries[0].Source != "Hi there! How are you doing today?" {
		t.Errorf("Expected dialogue text, got: %s", entries[0].Source)
	}
	if entries[1].Source != "I am doing great!" || entries[1].Context != "Choice" {
		t.Errorf("Expected choice text, got: %s (%s)", entries[1].Source, entries[1].Context)
	}

	t.Logf("Extracted %d Ren'Py strings in %v", len(entries), stats.Duration)

	// 3. Inject
	entries[0].Target = "สวัสดี! วันนี้คุณเป็นอย่างไรบ้าง?"
	entries[1].Target = "ฉันสบายดีมาก!"
	entries[2].Target = "ก็พอไหวอยู่"

	if err := parser.Inject(ctx, tempGame, tempGame, entries); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify lingo_language.rpy
	langBootBytes, err := os.ReadFile(filepath.Join(gameDir, "lingo_language.rpy"))
	if err != nil || !strings.Contains(string(langBootBytes), `config.language = "Thai"`) {
		t.Fatalf("lingo_language.rpy invalid: %s", string(langBootBytes))
	}

	// Verify tl/Thai/lingo_translations.rpy
	tlBytes, err := os.ReadFile(filepath.Join(gameDir, "tl", "Thai", "lingo_translations.rpy"))
	if err != nil {
		t.Fatalf("Failed to read lingo_translations.rpy: %v", err)
	}

	content := string(tlBytes)
	if !strings.Contains(content, `old "Hi there! How are you doing today?"`) ||
		!strings.Contains(content, `new "สวัสดี! วันนี้คุณเป็นอย่างไรบ้าง?"`) {
		t.Errorf("Translations file missing expected translation block:\n%s", content)
	}

	t.Log("Ren'Py Parser tests passed successfully!")
}
