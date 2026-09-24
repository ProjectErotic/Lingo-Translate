package pipeline

import (
	"context"
	"path/filepath"
	"testing"

	"lingo-translate/pkg/decision"
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

func TestPipelineSpeculativeDraftAcceptor(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.Open(filepath.Join(tempDir, "test.nst"))
	defer store.Close()

	primaryTrans := mock.New("[Primary] ")
	fastTrans := mock.New("[FastMT] ")
	decisionEngine := decision.NewHeuristicEngine()

	pipe := New(store, primaryTrans, Config{BatchSize: 10, Concurrency: 1})
	pipe.SetTaskRouting(fastTrans, true, 30)
	pipe.SetSystemOne(decisionEngine, true, false)

	entries := []model.TextEntry{
		{
			ID:     "1",
			Source: "Play", // Short string <= 30 chars
		},
		{
			ID:     "2",
			Source: "This is a much longer narrative text exceeding the thirty character threshold for testing.", // Long string > 30 chars
		},
	}

	opts := translator.Options{SourceLang: "en", TargetLang: "th"}
	results, err := pipe.Run(context.Background(), entries, opts, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Short string should have been speculatively drafted and accepted by System One!
	if results[0].Translator != "mock" {
		t.Errorf("expected fast translator 'mock', got '%s'", results[0].Translator)
	}
	if results[0].Status != model.StatusTranslated {
		t.Errorf("expected StatusTranslated, got %s", results[0].Status)
	}
	if results[0].Target != "[FastMT] Play" {
		t.Errorf("expected '[FastMT] Play', got '%s'", results[0].Target)
	}

	// Long string should have been routed to primary translator!
	if results[1].Target != "[Primary] This is a much longer narrative text exceeding the thirty character threshold for testing." {
		t.Errorf("expected primary translation on long string, got '%s'", results[1].Target)
	}
}

func TestPipelineQAValidation(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.Open(filepath.Join(tempDir, "qa_test.nst"))
	defer store.Close()

	primaryTrans := mock.New("[Primary] ")
	pipe := New(store, primaryTrans, Config{BatchSize: 10, Concurrency: 1})

	// Decision engine that flags refusal
	mockEngine := &mockRefusalDecisionEngine{}
	pipe.SetSystemOne(mockEngine, false, true)

	entries := []model.TextEntry{
		{
			ID:     "refused_1",
			Source: "Dangerous spell",
		},
	}

	opts := translator.Options{SourceLang: "en", TargetLang: "th"}
	results, err := pipe.Run(context.Background(), entries, opts, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should be flagged as untranslated because QA detected refusal
	if results[0].Status != model.StatusUntranslated {
		t.Errorf("expected StatusUntranslated on refusal, got %s", results[0].Status)
	}
}

type mockRefusalDecisionEngine struct{}

func (m *mockRefusalDecisionEngine) Name() string { return "mock_refusal" }
func (m *mockRefusalDecisionEngine) Noul(ctx context.Context, input, question string) (bool, float64, error) {
	return true, 0.95, nil // Always flags as refusal
}
func (m *mockRefusalDecisionEngine) Choice(ctx context.Context, input string, choices []string) (string, float64, error) {
	return choices[0], 1.0, nil
}
func (m *mockRefusalDecisionEngine) Score(ctx context.Context, input, criteria string) (float64, error) {
	return 0.1, nil
}

type failingTranslator struct{}

func (f *failingTranslator) Name() string { return "failing" }
func (f *failingTranslator) Translate(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	return nil, context.DeadlineExceeded
}

func TestPipelineFallbackTranslator(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.Open(filepath.Join(tempDir, "fallback_test.nst"))
	defer store.Close()

	primaryFailing := &failingTranslator{}
	fallbackMock := mock.New("[Fallback] ")

	pipe := New(store, primaryFailing, Config{BatchSize: 10, Concurrency: 1, Retries: 1, RetryDelay: 1})
	pipe.SetFallbackTranslator(fallbackMock)

	entries := []model.TextEntry{
		{
			ID:     "fb_1",
			Source: "Hero awakens",
		},
	}

	opts := translator.Options{SourceLang: "en", TargetLang: "th"}
	results, err := pipe.Run(context.Background(), entries, opts, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 1 || results[0].Target != "[Fallback] Hero awakens" {
		t.Errorf("Expected fallback translation '[Fallback] Hero awakens', got: %v", results[0].Target)
	}
	if results[0].Status != model.StatusTranslated {
		t.Errorf("Expected StatusTranslated, got: %s", results[0].Status)
	}
}

func TestPipelineMemoryCacheToggle(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.Open(filepath.Join(tempDir, "cache_toggle_test.nst"))
	defer store.Close()

	trans := mock.New("[Trans] ")
	pipe := New(store, trans, Config{BatchSize: 10, Concurrency: 1})

	// Pre-seed cache
	_ = store.SetCache("Hello", "สวัสดี (Cached)", "en", "th", "tm_cache")

	entries := []model.TextEntry{
		{ID: "c1", Source: "Hello"},
	}
	opts := translator.Options{SourceLang: "en", TargetLang: "th"}

	// 1. With cache enabled (default)
	pipe.SetMemoryCache(true)
	res1, _ := pipe.Run(context.Background(), entries, opts, nil)
	if res1[0].Target != "สวัสดี (Cached)" {
		t.Errorf("Expected cached target, got: %s", res1[0].Target)
	}

	// 2. With cache disabled
	pipe.SetMemoryCache(false)
	entries2 := []model.TextEntry{
		{ID: "c2", Source: "Hello"},
	}
	res2, _ := pipe.Run(context.Background(), entries2, opts, nil)
	if res2[0].Target != "[Trans] Hello" {
		t.Errorf("Expected fresh translation when cache disabled, got: %s", res2[0].Target)
	}
}
