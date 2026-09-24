package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/storage"
)

func TestDesktopServices(t *testing.T) {
	tempDir := t.TempDir()
	tempRegPath := filepath.Join(tempDir, "projects.json")
	tempSettingsPath := filepath.Join(tempDir, "settings.json")
	tempWsPath := filepath.Join(tempDir, "test_project.nst")

	session := NewSession()
	projectSvc := NewProjectService(session, tempRegPath)
	entrySvc := NewEntryService(session)
	transSvc := NewTranslationService(session)
	deploySvc := NewDeployService(session)
	settingsSvc := NewSettingsService(tempSettingsPath)

	// 1. Test SettingsService
	settings, err := settingsSvc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.DefaultProvider != "mock" {
		t.Errorf("Expected default provider 'mock', got %s", settings.DefaultProvider)
	}
	if settings.Tasks.PrimaryTranslation.Model == "" {
		t.Errorf("Expected default Tasks.PrimaryTranslation.Model to be populated")
	}
	if settings.SystemOne.ConfidenceThreshold != 0.85 {
		t.Errorf("Expected default SystemOne.ConfidenceThreshold 0.85, got %f", settings.SystemOne.ConfidenceThreshold)
	}

	settings.DefaultModel = "gpt-4o-custom"
	settings.GeminiAPIKey = "test-api-key-123"
	settings.Tasks.PrimaryTranslation = TaskBinding{
		Provider: "openai",
		Model:    "gpt-4o",
	}
	settings.Tasks.AutoRouteShortText = true
	settings.Tasks.MaxShortLength = 45
	settings.SystemOne.Enabled = true
	settings.SystemOne.APIKey = "ts_live_mock_key_999"
	settings.SystemOne.Features.FilterAmbiguousCode = true
	settings.SystemOne.Features.AcceptUIDrafts = true
	if err := settingsSvc.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	savedSettings, err := settingsSvc.GetSettings()
	if err != nil {
		t.Fatalf("Re-read settings failed: %v", err)
	}
	if savedSettings.GeminiAPIKey != "test-api-key-123" {
		t.Errorf("Expected saved API key 'test-api-key-123', got '%s'", savedSettings.GeminiAPIKey)
	}
	if savedSettings.Tasks.PrimaryTranslation.Model != "gpt-4o" {
		t.Errorf("Expected Tasks.PrimaryTranslation.Model 'gpt-4o', got '%s'", savedSettings.Tasks.PrimaryTranslation.Model)
	}
	if !savedSettings.Tasks.AutoRouteShortText || savedSettings.Tasks.MaxShortLength != 45 {
		t.Errorf("Tasks.AutoRouteShortText or MaxShortLength not persisted properly")
	}
	if !savedSettings.SystemOne.Enabled || savedSettings.SystemOne.APIKey != "ts_live_mock_key_999" {
		t.Errorf("SystemOne settings not persisted properly")
	}

	// Test ResolveProviderAuth
	kGemini, _ := savedSettings.ResolveProviderAuth("gemini")
	if kGemini != "test-api-key-123" {
		t.Errorf("ResolveProviderAuth(gemini) failed, got '%s'", kGemini)
	}
	savedSettings.OpenAIAPIKey = "sk-openai-key"
	savedSettings.OpenAIBaseURL = "https://custom.api.com/v1"
	kOpenAI, bOpenAI := savedSettings.ResolveProviderAuth("openai")
	if kOpenAI != "sk-openai-key" || bOpenAI != "https://custom.api.com/v1" {
		t.Errorf("ResolveProviderAuth(openai) failed, got key=%s, base=%s", kOpenAI, bOpenAI)
	}

	// 2. Test DetectEngine
	gameDir := filepath.Join("..", "..", "test_game")
	engineName, err := projectSvc.DetectEngine(gameDir)
	if err != nil {
		t.Fatalf("DetectEngine failed: %v", err)
	}
	if engineName != "rpgm" {
		t.Errorf("Expected engine 'rpgm', got '%s'", engineName)
	}

	// 3. Test CreateFromGame
	stats, err := projectSvc.CreateFromGame(gameDir, tempWsPath, "Japanese", "Thai")
	if err != nil {
		t.Fatalf("CreateFromGame failed: %v", err)
	}
	if stats.TotalEntries == 0 {
		t.Fatalf("Expected extracted entries, got 0")
	}

	// 4. Test Current project
	currentProj := projectSvc.Current()
	if currentProj == nil {
		t.Fatalf("Expected current project to be non-nil")
	}
	if currentProj.Engine != "rpgm" {
		t.Errorf("Expected current project engine 'rpgm', got '%s'", currentProj.Engine)
	}

	// 5. Test ProjectService.List
	projects, err := projectSvc.List()
	if err != nil {
		t.Fatalf("ProjectService.List failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 registered project, got %d", len(projects))
	}

	// 6. Test EntryService: Files, Query, Update, Stats
	files, err := entrySvc.Files()
	if err != nil {
		t.Fatalf("EntryService.Files failed: %v", err)
	}
	if len(files) == 0 {
		t.Errorf("Expected files summary list, got empty")
	}

	queryRes, err := entrySvc.Query(storage.EntryQuery{Limit: 5})
	if err != nil {
		t.Fatalf("EntryService.Query failed: %v", err)
	}
	if len(queryRes.Entries) == 0 || queryRes.Total == 0 {
		t.Fatalf("Expected query results, got 0")
	}

	firstEntryID := queryRes.Entries[0].ID
	if err := entrySvc.Update(firstEntryID, "สวัสดีเทสต์"); err != nil {
		t.Fatalf("EntryService.Update failed: %v", err)
	}

	wsStats, err := entrySvc.Stats()
	if err != nil {
		t.Fatalf("EntryService.Stats failed: %v", err)
	}
	if wsStats.Translated < 1 {
		t.Errorf("Expected at least 1 translated entry after update, got %d", wsStats.Translated)
	}

	// 7. Test TranslationService with mock provider
	err = transSvc.Start(app.TranslateOptions{
		Provider: app.ProviderConfig{
			Name: "mock",
		},
		BatchSize:   5,
		Concurrency: 2,
		Scope:       "untranslated",
	})
	if err != nil {
		t.Fatalf("TranslationService.Start failed: %v", err)
	}

	// Concurrent run rejection
	err2 := transSvc.Start(app.TranslateOptions{
		Provider: app.ProviderConfig{Name: "mock"},
	})
	if err2 == nil {
		t.Errorf("Expected error starting concurrent translation, got nil")
	}

	// Wait for translation to complete
	deadline := time.Now().Add(5 * time.Second)
	for transSvc.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if transSvc.IsRunning() {
		t.Errorf("Translation did not complete within deadline")
	}

	// 8. Test DeployService
	// Copy test_game to temp dir for safe deploy testing
	tempGameDir := filepath.Join(tempDir, "game_copy")
	_ = os.MkdirAll(filepath.Join(tempGameDir, "data"), 0755)
	_ = os.MkdirAll(filepath.Join(tempGameDir, "js", "plugins"), 0755)
	_ = os.WriteFile(filepath.Join(tempGameDir, "js", "plugins.js"), []byte("var $plugins = [];"), 0644)

	err = deploySvc.DeployLayer(tempGameDir, "Thai")
	if err != nil {
		t.Fatalf("DeployLayer failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempGameDir, "js", "plugins", "Lingo_TranslationLayer.js")); err != nil {
		t.Errorf("Lingo_TranslationLayer.js was not deployed: %v", err)
	}

	// 9. Test ProjectService Close and OpenWorkspace
	if err := projectSvc.Close(); err != nil {
		t.Fatalf("ProjectService.Close failed: %v", err)
	}
	if projectSvc.Current() != nil {
		t.Errorf("Expected current project to be nil after close")
	}

	reopenedProj, err := projectSvc.OpenWorkspace(tempWsPath)
	if err != nil {
		t.Fatalf("OpenWorkspace failed: %v", err)
	}
	if reopenedProj == nil {
		t.Fatalf("Expected reopened project to be non-nil")
	}

	// 10. Test RemoveFromRegistry
	if err := projectSvc.RemoveFromRegistry(tempWsPath); err != nil {
		t.Fatalf("RemoveFromRegistry failed: %v", err)
	}
	projectsAfterRemove, err := projectSvc.List()
	if err != nil {
		t.Fatalf("List after remove failed: %v", err)
	}
	if len(projectsAfterRemove) != 0 {
		t.Errorf("Expected 0 projects after remove, got %d", len(projectsAfterRemove))
	}
}

func TestFetchRemoteModels(t *testing.T) {
	// 1. Mock server that requires auth and returns OpenAI format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid-test-key" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"Authentication Error, No api key passed in."}}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[{"id":"model-alpha"},{"id":"model-beta"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")
	settingsSvc := NewSettingsService(settingsPath)

	// Test with valid key
	models, err := settingsSvc.FetchRemoteModels(server.URL+"/v1", "valid-test-key")
	if err != nil {
		t.Fatalf("FetchRemoteModels failed with valid key: %v", err)
	}
	if len(models) != 2 || models[0] != "model-alpha" || models[1] != "model-beta" {
		t.Fatalf("Unexpected models returned: %v", models)
	}

	// Test with missing key: must return clean auth error, not 404 from /api/tags
	_, err = settingsSvc.FetchRemoteModels(server.URL+"/v1", "")
	if err == nil {
		t.Fatalf("Expected error for missing key, got nil")
	}
	if !strings.Contains(err.Error(), "API key required") && !strings.Contains(err.Error(), "Authentication Error") {
		t.Errorf("Expected auth error message, got: %v", err)
	}

	// Test with invalid key: must return HTTP 401 failure
	_, err = settingsSvc.FetchRemoteModels(server.URL+"/v1", "wrong-key")
	if err == nil {
		t.Fatalf("Expected error for wrong key, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 error, got: %v", err)
	}
}
