package unity

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUnityExtractAndInject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nst_unity_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock Unity structure:
	// 1. Assets/
	assetsDir := filepath.Join(tmpDir, "Assets")
	_ = os.MkdirAll(assetsDir, 0755)

	// 2. Prefab with m_Text
	prefabContent := `%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!1 &100000
GameObject:
  m_ObjectHideFlags: 0
  m_CorrespondingSourceObject: {fileID: 0}
  m_PrefabInstance: {fileID: 0}
  m_PrefabAsset: {fileID: 0}
  m_Name: StartButton
--- !u!114 &100001
MonoBehaviour:
  m_ObjectHideFlags: 0
  m_Script: {fileID: 11500000, guid: 5f7201a12d95ffc409449d95f23cf332, type: 3}
  m_Name: 
  m_Text: "Click here to begin your adventure!"
`
	_ = os.WriteFile(filepath.Join(assetsDir, "Menu.prefab"), []byte(prefabContent), 0644)

	// 3. StreamingAssets/localization.json
	streamingDir := filepath.Join(tmpDir, "Assets", "StreamingAssets")
	_ = os.MkdirAll(streamingDir, 0755)
	jsonContent := `{
  "ui": {
    "title": "Fantasy Legends",
    "load_game": "Load Saved Game"
  }
}`
	_ = os.WriteFile(filepath.Join(streamingDir, "localization.json"), []byte(jsonContent), 0644)

	// 4. StreamingAssets/dialogue.csv
	csvContent := `Key,Source,Description
greeting,"Welcome, young warrior!","NPC introduction"
quest1,"The ancient dungeon lies to the north.","Quest instruction"
`
	_ = os.WriteFile(filepath.Join(streamingDir, "dialogue.csv"), []byte(csvContent), 0644)

	parser := New()

	// Test Detect
	if !parser.Detect(tmpDir) {
		t.Fatalf("expected Detect to return true for directory with Assets/")
	}

	// Test Extract
	ctx := context.Background()
	start := time.Now()
	entries, stats, err := parser.Extract(ctx, tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entries) < 4 {
		t.Fatalf("expected at least 4 entries, got %d", len(entries))
	}

	if stats.TotalEntries < 4 {
		t.Fatalf("expected at least 4 total entries in stats, got %d", stats.TotalEntries)
	}

	// Verify entries were extracted properly
	foundPrefab := false
	foundJSON := false
	foundCSV := false
	for _, e := range entries {
		if strings.Contains(e.Source, "Click here to begin") {
			foundPrefab = true
		}
		if e.Source == "Fantasy Legends" {
			foundJSON = true
		}
		if strings.Contains(e.Source, "Welcome, young warrior!") {
			foundCSV = true
		}
	}

	if !foundPrefab {
		t.Errorf("failed to extract text from Menu.prefab")
	}
	if !foundJSON {
		t.Errorf("failed to extract text from localization.json")
	}
	if !foundCSV {
		t.Errorf("failed to extract text from dialogue.csv")
	}

	// Test Inject
	outDir, err := os.MkdirTemp("", "nst_unity_out_*")
	if err != nil {
		t.Fatalf("failed to create temp out dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	// Assign translations
	for i := range entries {
		entries[i].Target = "[TH] " + entries[i].Source
	}

	if err := parser.Inject(ctx, tmpDir, outDir, entries); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify XUnity translation file
	xunityFile := filepath.Join(outDir, "Translation", "th", "Text", "lingo_translations.txt")
	xunityBytes, err := os.ReadFile(xunityFile)
	if err != nil {
		t.Fatalf("failed to read XUnity translation file: %v", err)
	}
	xunityStr := string(xunityBytes)
	if !strings.Contains(xunityStr, "Fantasy Legends=[TH] Fantasy Legends") {
		t.Errorf("XUnity file missing translated entry. Content:\n%s", xunityStr)
	}

	// Verify patched JSON file
	patchedJSON := filepath.Join(outDir, "Assets", "StreamingAssets", "localization.json")
	jsonBytes, err := os.ReadFile(patchedJSON)
	if err != nil {
		t.Fatalf("failed to read patched json file: %v", err)
	}
	if !strings.Contains(string(jsonBytes), "[TH] Fantasy Legends") {
		t.Errorf("patched JSON does not contain translated text: %s", string(jsonBytes))
	}

	// Verify patched Prefab file
	patchedPrefab := filepath.Join(outDir, "Assets", "Menu.prefab")
	prefabBytes, err := os.ReadFile(patchedPrefab)
	if err != nil {
		t.Fatalf("failed to read patched prefab: %v", err)
	}
	if !strings.Contains(string(prefabBytes), "[TH] Click here to begin") {
		t.Errorf("patched Prefab does not contain translated text: %s", string(prefabBytes))
	}

	t.Logf("Unity parser test passed in %v!", time.Since(start))
}
