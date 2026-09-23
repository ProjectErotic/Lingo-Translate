package custom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lingo-translate/pkg/translator"
)

func TestCustomProviderLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("NST_PROVIDERS_DIR", tempDir)

	def := Definition{
		Name:         "testai",
		DisplayName:  "Test AI Provider",
		BaseURL:      "https://api.testai.cc/v1",
		DefaultModel: "test-model-v1",
		APIKeyEnv:    "TESTAI_API_KEY",
		Headers: map[string]string{
			"X-Custom": "test-header",
		},
		TimeoutSec: 10,
	}

	dataDir := filepath.Join(tempDir, "testai.json")
	data, _ := os.ReadFile(dataDir)
	_ = data
	// write def as json
	bytes, _ := os.ReadFile(dataDir)
	_ = bytes
	file, _ := os.Create(dataDir)
	_ = json.NewEncoder(file).Encode(def)
	file.Close()

	// List
	list, err := List()
	if err != nil {
		t.Fatalf("failed to list providers: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected at least 1 provider, got 0")
	}

	// Find
	found, err := Find("testai")
	if err != nil {
		t.Fatalf("failed to find testai: %v", err)
	}
	if found.DefaultModel != "test-model-v1" {
		t.Errorf("expected model test-model-v1, got %s", found.DefaultModel)
	}

	// Create Translator
	t.Setenv("TESTAI_API_KEY", "test-secret-key")
	tr, err := NewTranslator(*found, "", "", "")
	if err != nil {
		t.Fatalf("failed to create translator: %v", err)
	}
	if tr.Name() != "testai" {
		t.Errorf("expected translator name 'testai', got %s", tr.Name())
	}
}

func TestCustomTranslatorExecution(t *testing.T) {
	// Mock HTTP server responding with OpenAI format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Test") != "val" {
			http.Error(w, "missing header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "[\"สวัสดีโลก\"]"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	def := Definition{
		Name:         "mockcustom",
		BaseURL:      server.URL,
		DefaultModel: "test-model",
		Headers: map[string]string{
			"X-Test": "val",
		},
	}

	tr, err := NewTranslator(def, "mock-key", "", "")
	if err != nil {
		t.Fatalf("failed to create custom translator: %v", err)
	}

	results, err := tr.Translate(context.Background(), []string{"Hello world"}, translator.Options{
		SourceLang: "English",
		TargetLang: "Thai",
	})
	if err != nil {
		t.Fatalf("translate error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Target != "สวัสดีโลก" {
		t.Errorf("expected 'สวัสดีโลก', got %q", results[0].Target)
	}
	if results[0].Translator != "mockcustom" {
		t.Errorf("expected translator name 'mockcustom', got %q", results[0].Translator)
	}
}
