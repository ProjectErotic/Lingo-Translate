package storage

import (
	"path/filepath"
	"testing"
	"time"

	"lingo-translate/pkg/model"
)

func TestStorageWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "workspace.nst")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	// 1. Save and Get Project
	proj := &model.Project{
		ID:         "proj-1",
		Name:       "Test RPG",
		Engine:     "rpgm",
		SourcePath: "/games/rpg1",
		SourceLang: "Japanese",
		TargetLang: "Thai",
		CreatedAt:  time.Now(),
	}

	if err := s.SaveProject(proj); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	loadedProj, err := s.GetProject()
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if loadedProj.Name != "Test RPG" {
		t.Errorf("Expected project name 'Test RPG', got '%s'", loadedProj.Name)
	}

	// 2. Save Entries
	entries := []model.TextEntry{
		{
			ID:       "Map001.json:events[1].pages[0].parameters[0]",
			Source:   "こんにちは",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[0]",
			Status:   model.StatusUntranslated,
		},
		{
			ID:       "Map001.json:events[1].pages[0].parameters[1]",
			Source:   "さようなら",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[1]",
			Status:   model.StatusUntranslated,
		},
	}

	if err := s.SaveEntries(entries); err != nil {
		t.Fatalf("SaveEntries failed: %v", err)
	}

	allEntries, err := s.GetEntries("all")
	if err != nil {
		t.Fatalf("GetEntries failed: %v", err)
	}
	if len(allEntries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(allEntries))
	}

	// 3. Update entry translation
	if err := s.UpdateEntryTarget(entries[0].ID, "สวัสดี", model.StatusTranslated, "mock"); err != nil {
		t.Fatalf("UpdateEntryTarget failed: %v", err)
	}

	untranslated, err := s.GetEntries(string(model.StatusUntranslated))
	if err != nil {
		t.Fatalf("GetEntries untranslated failed: %v", err)
	}
	if len(untranslated) != 1 {
		t.Errorf("Expected 1 untranslated entry, got %d", len(untranslated))
	}

	// 4. Test Translation Memory (Cache)
	cached, found, err := s.GetCache("こんにちは", "Japanese", "Thai")
	if err != nil || found {
		t.Errorf("Expected cache miss, got found=%v, err=%v", found, err)
	}

	if err := s.SetCache("こんにちは", "สวัสดี", "Japanese", "Thai", "gemini"); err != nil {
		t.Fatalf("SetCache failed: %v", err)
	}

	cached, found, err = s.GetCache("こんにちは", "Japanese", "Thai")
	if err != nil || !found || cached != "สวัสดี" {
		t.Errorf("Expected cache hit 'สวัสดี', got '%s', found=%v, err=%v", cached, found, err)
	}

	// 5. Test Stats
	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.Total != 2 || stats.Translated != 1 || stats.Pending != 1 || stats.Percent != 50.0 {
		t.Errorf("Unexpected stats: %+v", stats)
	}

	// 6. Test ListFiles
	files, err := s.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(files) != 1 || files[0].Path != "Map001.json" || files[0].Total != 2 || files[0].Translated != 1 {
		t.Errorf("Unexpected files: %+v", files)
	}

	// 7. Test QueryEntries with filtering, search, and pagination
	// 7a. Query all
	res, total, err := s.QueryEntries(EntryQuery{})
	if err != nil || total != 2 || len(res) != 2 {
		t.Fatalf("QueryEntries all failed: total=%d, len=%d, err=%v", total, len(res), err)
	}

	// 7b. Filter by status: translated
	res, total, err = s.QueryEntries(EntryQuery{Status: "translated"})
	if err != nil || total != 1 || len(res) != 1 || res[0].Target != "สวัสดี" {
		t.Fatalf("QueryEntries translated failed: total=%d, len=%d, err=%v", total, len(res), err)
	}

	// 7c. Filter by status: untranslated
	res, total, err = s.QueryEntries(EntryQuery{Status: "untranslated"})
	if err != nil || total != 1 || len(res) != 1 || res[0].Source != "さようなら" {
		t.Fatalf("QueryEntries untranslated failed: total=%d, len=%d, err=%v", total, len(res), err)
	}

	// 7d. Search query
	res, total, err = s.QueryEntries(EntryQuery{Search: "สวัสดี"})
	if err != nil || total != 1 || len(res) != 1 {
		t.Fatalf("QueryEntries search failed: total=%d, len=%d, err=%v", total, len(res), err)
	}

	// 7e. Pagination (Limit 1, Offset 1)
	res, total, err = s.QueryEntries(EntryQuery{Limit: 1, Offset: 1})
	if err != nil || total != 2 || len(res) != 1 {
		t.Fatalf("QueryEntries pagination failed: total=%d, len=%d, err=%v", total, len(res), err)
	}
}

