package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/storage"
)

func TestAppWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	wsPath := filepath.Join(tempDir, "test_ws.nst")
	gameDir, err := filepath.Abs("../../test_game")
	if err != nil {
		t.Fatalf("Failed to resolve game path: %v", err)
	}

	// 1. Test Open non-existent workspace (should fail)
	_, err = Open(filepath.Join(tempDir, "does_not_exist.nst"))
	if err == nil {
		t.Fatal("Expected error opening non-existent workspace, got nil")
	}

	// 2. Test CreateFromGame
	ws, stats, err := CreateFromGame(gameDir, wsPath, "Japanese", "Thai")
	if err != nil {
		t.Fatalf("CreateFromGame failed: %v", err)
	}
	defer ws.Close()

	if stats == nil || stats.TotalEntries == 0 {
		t.Fatalf("Expected extraction stats with entries, got %+v", stats)
	}

	proj := ws.Project()
	if proj == nil || proj.SourceLang != "Japanese" || proj.TargetLang != "Thai" {
		t.Fatalf("Unexpected project metadata: %+v", proj)
	}

	// 3. Test ListFiles and Stats
	files, err := ws.ListFiles()
	if err != nil || len(files) == 0 {
		t.Fatalf("ListFiles failed: %v (len=%d)", err, len(files))
	}

	wStats, err := ws.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if wStats.Total != stats.TotalEntries {
		t.Errorf("Expected total entries %d, got %d", stats.TotalEntries, wStats.Total)
	}

	// 4. Test UpdateEntry
	entries, total, err := ws.QueryEntries(storage.EntryQuery{})
	if err != nil || total == 0 {
		t.Fatalf("QueryEntries failed: %v, total=%d", err, total)
	}

	targetEntry := entries[0]
	err = ws.UpdateEntry(targetEntry.ID, "ข้อความทดสอบ", model.StatusTranslated, "tester")
	if err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	// 5. Test Translate with mock
	ctx := context.Background()
	var progressReports []model.TranslationProgress
	err = ws.Translate(ctx, TranslateOptions{
		Provider:    ProviderConfig{Name: "mock"},
		BatchSize:   5,
		Concurrency: 2,
		Scope:       "all",
	}, func(p model.TranslationProgress) {
		progressReports = append(progressReports, p)
	})
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if len(progressReports) == 0 {
		t.Errorf("Expected progress reports from translation")
	}

	// 6. Test Open existing workspace
	ws.Close()

	ws2, err := Open(wsPath)
	if err != nil {
		t.Fatalf("Open existing workspace failed: %v", err)
	}
	defer ws2.Close()

	wStats2, err := ws2.Stats()
	if err != nil {
		t.Fatalf("Stats on reopened workspace failed: %v", err)
	}
	if wStats2.Translated != wStats2.Total {
		t.Errorf("Expected 100%% translated after mock translation, got %d/%d", wStats2.Translated, wStats2.Total)
	}

	// 7. Test ExportCopy
	copyDest := filepath.Join(tempDir, "game_copy")
	err = ws2.ExportCopy(ctx, gameDir, copyDest)
	if err != nil {
		t.Fatalf("ExportCopy failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(copyDest, "data")); err != nil {
		t.Errorf("ExportCopy did not create data directory: %v", err)
	}
}

func TestCreateTranslator(t *testing.T) {
	// Mock
	tr, err := CreateTranslator(ProviderConfig{Name: "mock"})
	if err != nil || tr == nil {
		t.Fatalf("Failed to create mock translator: %v", err)
	}

	// Gemini without key
	_, err = CreateTranslator(ProviderConfig{Name: "gemini"})
	if err == nil {
		t.Fatal("Expected error creating Gemini translator without API key")
	}

	// Unknown
	_, err = CreateTranslator(ProviderConfig{Name: "nonexistent"})
	if err == nil {
		t.Fatal("Expected error creating nonexistent translator")
	}

	// Custom Provider via Env Dir
	tempDir := t.TempDir()
	t.Setenv("NST_PROVIDERS_DIR", tempDir)
	_ = os.WriteFile(filepath.Join(tempDir, "mycustom.json"), []byte(`{
		"name": "mycustom",
		"display_name": "My Custom AI",
		"base_url": "https://custom.example.com/v1",
		"default_model": "custom-v1",
		"api_key": "custom-key"
	}`), 0644)

	customTr, err := CreateTranslator(ProviderConfig{Name: "mycustom"})
	if err != nil || customTr == nil {
		t.Fatalf("Failed to create custom translator: %v", err)
	}
	if customTr.Name() != "mycustom" {
		t.Errorf("Expected translator name 'mycustom', got %s", customTr.Name())
	}

	providers := ListAvailableProviders()
	found := false
	for _, p := range providers {
		if p.Name == "mycustom" && p.IsCustom {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'mycustom' in ListAvailableProviders, but not found")
	}
}

func TestWorkspaceDeployLayerRenpy(t *testing.T) {
	tempDir := t.TempDir()
	wsPath := filepath.Join(tempDir, "renpy_ws.nst")
	gameDir := filepath.Join(tempDir, "RenpyGame")
	_ = os.MkdirAll(filepath.Join(gameDir, "game"), 0755)

	store, err := storage.Open(wsPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	_ = store.SaveProject(&model.Project{
		ID:         "test_renpy",
		Name:       "Test Renpy Game",
		SourcePath: gameDir,
		Engine:     "renpy",
		SourceLang: "Japanese",
		TargetLang: "Thai",
	})
	_ = store.SaveEntries([]model.TextEntry{
		{
			ID:        "dialogue_1",
			FilePath:  "script.rpy",
			KeyPath:   "start_test1",
			Source:    "こんにちは",
			Target:    "สวัสดีครับ",
			Status:    model.StatusTranslated,
		},
	})
	store.Close()

	ws, err := Open(wsPath)
	if err != nil {
		t.Fatalf("Failed to open workspace: %v", err)
	}
	defer ws.Close()

	if err := ws.DeployLayer(gameDir, "Thai"); err != nil {
		t.Fatalf("DeployLayer failed on Renpy: %v", err)
	}

	// Verify font file written
	fontPath := filepath.Join(gameDir, "game", "fonts", "IBMPlexSansThai-Light.otf")
	if fi, err := os.Stat(fontPath); err != nil || fi.Size() == 0 {
		t.Errorf("Expected font file at %s", fontPath)
	}

	// Verify font script written
	fontScript := filepath.Join(gameDir, "game", "tl", "Thai", "00_lingo_font_layer.rpy")
	if _, err := os.Stat(fontScript); err != nil {
		t.Errorf("Expected font script at %s", fontScript)
	}

	// Verify script.rpy written
	scriptFile := filepath.Join(gameDir, "game", "tl", "Thai", "script.rpy")
	data, err := os.ReadFile(scriptFile)
	if err != nil || !strings.Contains(string(data), "สวัสดีครับ") {
		t.Errorf("Expected translated text in %s", scriptFile)
	}
}


