package pipeline

import (
	"context"
	"path/filepath"
	"testing"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/storage"
	"lingo-translate/pkg/translator"
	"lingo-translate/pkg/translator/mock"
)

func TestPipelineExecution(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_ws.nst")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	trans := mock.New("[TH] ")
	pipe := New(store, trans, Config{
		BatchSize:   2,
		Concurrency: 2,
	})

	entries := []model.TextEntry{
		{
			ID:       "msg1",
			Source:   `\C[1]Hero\C[0]: Let's go!`,
			FilePath: "Map001.json",
			KeyPath:  "list[0].parameters[0]",
		},
		{
			ID:       "msg2",
			Source:   `Gold: \V[10]`,
			FilePath: "Map001.json",
			KeyPath:  "list[1].parameters[0]",
		},
		{
			ID:       "msg3",
			Source:   "Village Chief",
			FilePath: "Map002.json",
			KeyPath:  "name",
		},
	}

	opts := translator.Options{
		SourceLang: "Japanese",
		TargetLang: "Thai",
	}

	progressUpdates := 0
	ctx := context.Background()

	// 1. First run: should translate all and cache them
	result, err := pipe.Run(ctx, entries, opts, func(p model.TranslationProgress) {
		progressUpdates++
	})
	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(result))
	}

	expectedHero := `[TH] \C[1]Hero\C[0]: Let's go!`
	if result[0].Target != expectedHero {
		t.Errorf("Expected '%s', got '%s'", expectedHero, result[0].Target)
	}

	expectedGold := `[TH] Gold: \V[10]`
	if result[1].Target != expectedGold {
		t.Errorf("Expected '%s', got '%s'", expectedGold, result[1].Target)
	}

	// 2. Second run with fresh entries having the same sources: should hit cache!
	freshEntries := []model.TextEntry{
		{
			ID:       "msg4",
			Source:   `Gold: \V[10]`,
			FilePath: "Map003.json",
		},
	}

	result2, err := pipe.Run(ctx, freshEntries, opts, nil)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	if result2[0].Translator != "tm_cache" {
		t.Errorf("Expected cache hit (translator='tm_cache'), got '%s'", result2[0].Translator)
	}
	if result2[0].Target != expectedGold {
		t.Errorf("Expected cached target '%s', got '%s'", expectedGold, result2[0].Target)
	}

	t.Log("Pipeline test with Masking and TM Cache passed!")
}
