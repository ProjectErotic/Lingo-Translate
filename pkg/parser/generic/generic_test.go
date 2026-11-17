package generic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lingo-translate/pkg/model"
)

func TestGenericParserDetectionAndExtract(t *testing.T) {
	tempDir := t.TempDir()
	langDir := filepath.Join(tempDir, "lang")
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}

	sampleJSON := map[string]interface{}{
		"game_title": "My Awesome Game",
		"menu_start": "Start Game",
		"credits": []string{
			"Produced by Studio A",
			"Music by Composer B",
		},
	}

	raw, err := json.MarshalIndent(sampleJSON, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(langDir, "eng.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}

	parser := New()

	// 1. Test Detection
	if !parser.Detect(tempDir) {
		t.Fatalf("expected generic parser to detect %s", tempDir)
	}

	// 2. Test Extraction
	ctx := context.Background()
	entries, stats, err := parser.Extract(ctx, tempDir)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if stats.TotalEntries != 4 {
		t.Fatalf("expected 4 total entries, got %d", stats.TotalEntries)
	}

	entryMap := make(map[string]model.TextEntry)
	for _, e := range entries {
		entryMap[e.ID] = e
	}

	if entryMap["game_title"].Source != "My Awesome Game" {
		t.Errorf("unexpected game_title: %s", entryMap["game_title"].Source)
	}
	if entryMap["credits[0]"].Source != "Produced by Studio A" {
		t.Errorf("unexpected credits[0]: %s", entryMap["credits[0]"].Source)
	}

	// 3. Test Injection
	outDir := t.TempDir()
	translatedEntries := []model.TextEntry{
		{ID: "game_title", Target: "เกมสุดเจ๋งของฉัน"},
		{ID: "menu_start", Target: "เริ่มเกม"},
		{ID: "credits[0]", Target: "สร้างโดย สตูดิโอ A"},
		{ID: "credits[1]", Target: "เพลงโดย ผู้แต่ง B"},
	}

	if err := parser.Inject(ctx, tempDir, outDir, translatedEntries); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	destFile := filepath.Join(outDir, "lang", "tha.json")
	destBytes, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read injected file: %v", err)
	}

	var injectedMap map[string]interface{}
	if err := json.Unmarshal(destBytes, &injectedMap); err != nil {
		t.Fatalf("injected file is not valid JSON: %v", err)
	}

	if injectedMap["game_title"] != "เกมสุดเจ๋งของฉัน" {
		t.Errorf("injected game_title mismatch: %v", injectedMap["game_title"])
	}

	credits, ok := injectedMap["credits"].([]interface{})
	if !ok || len(credits) != 2 || credits[0] != "สร้างโดย สตูดิโอ A" {
		t.Errorf("injected credits mismatch: %v", injectedMap["credits"])
	}
}

// TestAdulttaleIntegration directly tests extraction on the actual Adulttale game folder
func TestAdulttaleIntegration(t *testing.T) {
	gameDir := "/home/jop/Downloads/New Folder (77)/Adulttale-0.29.2-HOTFIX"
	if _, err := os.Stat(gameDir); err != nil {
		t.Skip("Adulttale folder not available in environment, skipping integration test")
	}

	parser := New()
	if !parser.Detect(gameDir) {
		t.Fatalf("expected generic parser to auto-detect Adulttale game folder")
	}

	entries, stats, err := parser.Extract(context.Background(), gameDir)
	if err != nil {
		t.Fatalf("failed to extract Adulttale entries: %v", err)
	}

	t.Logf("Adulttale Extracted Entries: %d (slice len %d), Unique Texts: %d, Duration: %v",
		stats.TotalEntries, len(entries), stats.UniqueTexts, stats.Duration)

	if stats.TotalEntries < 1000 {
		t.Fatalf("expected > 1000 entries from Adulttale, got %d", stats.TotalEntries)
	}
}
