package registry

import (
	"path/filepath"
	"testing"
	"time"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/storage"
)

func TestProjectRegistry(t *testing.T) {
	tempDir := t.TempDir()
	regFile := filepath.Join(tempDir, "projects.json")

	reg, err := NewWithFile(regFile)
	if err != nil {
		t.Fatalf("NewWithFile failed: %v", err)
	}

	// Create dummy workspace
	wsPath := filepath.Join(tempDir, "game1.nst")
	store, err := storage.Open(wsPath)
	if err != nil {
		t.Fatal(err)
	}

	_ = store.SaveProject(&model.Project{
		ID:         "game1",
		Name:       "Test RPG 1",
		Engine:     "rpgm",
		SourcePath: "/games/rpg1",
		SourceLang: "Japanese",
		TargetLang: "Thai",
		CreatedAt:  time.Now(),
	})

	_ = store.SaveEntries([]model.TextEntry{
		{ID: "1", Source: "Hello", Target: "สวัสดี", Status: model.StatusTranslated, FilePath: "Map001.json"},
		{ID: "2", Source: "World", Target: "", Status: model.StatusUntranslated, FilePath: "Map001.json"},
	})
	store.Close()

	// Register project
	entry, err := reg.Register(wsPath)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if entry.DisplayName != "Test RPG 1" {
		t.Errorf("Expected name 'Test RPG 1', got '%s'", entry.DisplayName)
	}
	if entry.TotalEntries != 2 || entry.TranslatedEntries != 1 {
		t.Errorf("Expected 2 total, 1 translated, got %d/%d", entry.TotalEntries, entry.TranslatedEntries)
	}
	if entry.TranslatedPercent != 50.0 {
		t.Errorf("Expected 50%%, got %.2f%%", entry.TranslatedPercent)
	}

	// Verify persistence: load fresh registry from file
	reg2, err := NewWithFile(regFile)
	if err != nil {
		t.Fatalf("Failed to reload registry: %v", err)
	}

	list := reg2.List()
	if len(list) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(list))
	}

	// Remove project
	if err := reg2.Remove(wsPath); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if len(reg2.List()) != 0 {
		t.Errorf("Expected 0 projects after removal, got %d", len(reg2.List()))
	}
}
