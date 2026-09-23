package libgdx

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"lingo-translate/pkg/model"
)

func createMockLibGDXJar(t *testing.T, jarPath string) {
	f, err := os.Create(jarPath)
	if err != nil {
		t.Fatalf("failed to create mock jar: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// 1. libGDX marker
	fw, _ := w.Create("com/badlogic/gdx/Gdx.class")
	fw.Write([]byte("mock-gdx-class"))

	// 2. translation/strings.properties
	fw, _ = w.Create("translation/strings.properties")
	fw.Write([]byte(`# Comments
mainmenu.begin=Begin Game
mainmenu.continue=Continue
character.health_gained=Gained {0} health\!
multiline.text=Line 1 \
               Line 2
`))

	// 3. script/encounters.json
	fw, _ = w.Create("script/encounters.json")
	fw.Write([]byte(`{
  "TROJA-INTRO": [
    {
      "text": "Hello traveller!",
      "music": "CALM"
    },
    {
      "chatter": "Be careful here.",
      "speaker": "Hiro"
    }
  ]
}`))

	// 4. script/encounter-trees.json
	fw, _ = w.Create("script/encounter-trees.json")
	fw.Write([]byte(`{
  "INITIAL": {
    "nodes": {
      "1": { "type": "START" },
      "2": { "prompt": "Ask about the road" }
    }
  }
}`))
}

func TestLibGDXParser_Detect(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nst_libgdx_detect_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	p := New()

	// Initially empty directory -> false
	if p.Detect(tmpDir) {
		t.Errorf("expected Detect to be false on empty dir")
	}

	// Create mock libgdx jar
	jarPath := filepath.Join(tmpDir, "Game.jar")
	createMockLibGDXJar(t, jarPath)

	// Now should detect -> true
	if !p.Detect(tmpDir) {
		t.Errorf("expected Detect to be true with libGDX jar present")
	}
}

func TestLibGDXParser_Extract(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nst_libgdx_extract_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jarPath := filepath.Join(tmpDir, "Game.jar")
	createMockLibGDXJar(t, jarPath)

	p := New()
	ctx := context.Background()

	entries, stats, err := p.Extract(ctx, tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("expected entries to be extracted, got 0")
	}

	if stats.FilesScanned == 0 {
		t.Errorf("expected FilesScanned > 0, got %d", stats.FilesScanned)
	}

	// Verify entries extracted
	foundMap := make(map[string]string)
	for _, e := range entries {
		foundMap[e.KeyPath] = e.Source
	}

	if val, ok := foundMap["mainmenu.begin"]; !ok || val != "Begin Game" {
		t.Errorf("expected 'Begin Game', got %v", val)
	}
	if val, ok := foundMap["TROJA-INTRO[0].text"]; !ok || val != "Hello traveller!" {
		t.Errorf("expected 'Hello traveller!', got %v", val)
	}
	if val, ok := foundMap["INITIAL.nodes.2.prompt"]; !ok || val != "Ask about the road" {
		t.Errorf("expected 'Ask about the road', got %v", val)
	}
}

func TestLibGDXParser_Inject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nst_libgdx_inject_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jarPath := filepath.Join(tmpDir, "Game.jar")
	createMockLibGDXJar(t, jarPath)

	p := New()
	ctx := context.Background()

	entries := []model.TextEntry{
		{
			FilePath: "translation/strings.properties",
			KeyPath:  "mainmenu.begin",
			Source:   "Begin Game",
			Target:   "เริ่มเกม",
		},
		{
			FilePath: "script/encounters.json",
			KeyPath:  "TROJA-INTRO[0].text",
			Source:   "Hello traveller!",
			Target:   "สวัสดีนักเดินทาง!",
		},
	}

	err = p.Inject(ctx, tmpDir, tmpDir, entries)
	if err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify translation mod directory created
	manifestPath := filepath.Join(tmpDir, "translations", "thai", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest.json not created: %v", err)
	}

	propsPath := filepath.Join(tmpDir, "translations", "thai", "assets", "translation", "strings.properties")
	content, err := os.ReadFile(propsPath)
	if err != nil {
		t.Fatalf("strings.properties not created: %v", err)
	}
	if !filepath.IsAbs(propsPath) {
		t.Fatalf("path should be valid")
	}

	if len(content) == 0 {
		t.Fatalf("properties content empty")
	}

	jsonPath := filepath.Join(tmpDir, "translations", "thai", "assets", "script", "encounters.json")
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("encounters.json not created: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Fatalf("json content empty")
	}
}

func TestLibGDXParser_RealToAGame(t *testing.T) {
	toaDir := "/home/jop/Downloads/New Folder (9)/Tales Of Androgyny Win64"
	if _, err := os.Stat(toaDir); err != nil {
		t.Skip("Tales of Androgyny directory not available, skipping")
	}

	p := New()
	if !p.Detect(toaDir) {
		t.Fatalf("expected Detect(%s) to be true", toaDir)
	}

	t.Logf("Successfully detected libGDX on %s", toaDir)

	ctx := context.Background()
	entries, stats, err := p.Extract(ctx, toaDir)
	if err != nil {
		t.Fatalf("Extract failed on real game: %v", err)
	}

	t.Logf("Extracted %d entries (%d unique words) across %d files in %v",
		stats.TotalEntries, stats.TotalWords, stats.FilesScanned, stats.Duration)

	if len(entries) < 1000 {
		t.Fatalf("expected over 1000 entries from real game, got %d", len(entries))
	}
}
