package renpy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lingo-translate/pkg/model"
)

func TestRenpyExporterDeploy(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nst_renpy_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gameDir := filepath.Join(tempDir, "game")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatalf("failed to create game dir: %v", err)
	}

	// Mock screens.rpy
	mockScreens := `
screen preferences():
    vbox:
        textbutton "English" action Language("EN")
        textbutton "Japanese" action Language("JP")

    null height (4 * gui.pref_spacing)
`
	if err := os.WriteFile(filepath.Join(gameDir, "screens.rpy"), []byte(mockScreens), 0644); err != nil {
		t.Fatalf("failed to write mock screens.rpy: %v", err)
	}

	entries := []model.TextEntry{
		{
			ID:       "1",
			FilePath: "script.rpy",
			KeyPath:  "start_12345",
			Context:  "Speaker: o",
			Source:   "Hello World",
			Target:   "สวัสดีชาวโลก",
		},
		{
			ID:       "2",
			FilePath: "screens.rpy",
			KeyPath:  "string:Start",
			Context:  "Choice",
			Source:   "Start",
			Target:   "เริ่มเกม",
		},
	}

	exporter := NewExporter()
	opts := DeployOptions{
		LanguageName: "Thai",
		DisplayName:  "ไทย (Thai)",
	}

	if err := exporter.Deploy(tempDir, entries, opts); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// 1. Verify fonts were installed to game/fonts/
	lightFont := filepath.Join(gameDir, "fonts", "IBMPlexSansThai-Light.otf")
	if fi, err := os.Stat(lightFont); err != nil || fi.Size() == 0 {
		t.Errorf("expected embedded light font to be written to %s", lightFont)
	}

	// 2. Verify 00_lingo_font_layer.rpy was created with translate Thai python:
	fontLayerPath := filepath.Join(gameDir, "tl", "Thai", "00_lingo_font_layer.rpy")
	fontContent, err := os.ReadFile(fontLayerPath)
	if err != nil {
		t.Fatalf("missing font layer script: %v", err)
	}
	if !strings.Contains(string(fontContent), "translate Thai python:") {
		t.Errorf("font layer script missing 'translate Thai python:'")
	}
	if !strings.Contains(string(fontContent), "translate Thai style centered_text:") {
		t.Errorf("font layer script missing 'translate Thai style centered_text:'")
	}

	// 3. Verify script.rpy was generated with dialogue and speaker
	scriptPath := filepath.Join(gameDir, "tl", "Thai", "script.rpy")
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("missing script.rpy: %v", err)
	}
	if !strings.Contains(string(scriptContent), "สวัสดีชาวโลก") {
		t.Errorf("script.rpy missing translated text")
	}
	if !strings.Contains(string(scriptContent), `o "สวัสดีชาวโลก"`) {
		t.Errorf("script.rpy missing speaker prefix in dialogue")
	}

	// 4. Verify screens.rpy was patched with clean language switcher and normalized spacing
	patchedScreens, err := os.ReadFile(filepath.Join(gameDir, "screens.rpy"))
	if err != nil {
		t.Fatalf("failed to read patched screens: %v", err)
	}
	if !strings.Contains(string(patchedScreens), `textbutton "{font=fonts/IBMPlexSansThai-Light.otf}ไทย (Thai){/font}" action Language("Thai")`) {
		t.Errorf("screens.rpy missing language button with font tag")
	}
	if strings.Contains(string(patchedScreens), "null height (4 * gui.pref_spacing)") {
		t.Errorf("screens.rpy failed to normalize vertical spacing")
	}
	if !strings.Contains(string(patchedScreens), "null height (1 * gui.pref_spacing)") {
		t.Errorf("screens.rpy missing normalized spacing")
	}
}
