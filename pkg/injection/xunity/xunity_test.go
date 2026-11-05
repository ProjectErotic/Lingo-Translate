package xunity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lingo-translate/pkg/model"
)

func TestDeploy(t *testing.T) {
	game := t.TempDir()
	if err := os.MkdirAll(filepath.Join(game, "BepInEx", "plugins"), 0755); err != nil {
		t.Fatal(err)
	}

	entries := []model.TextEntry{
		{Source: "Hello", Target: "สวัสดี", Status: model.StatusTranslated},
		{Source: "Multi\nline", Target: "หลาย\nบรรทัด", Status: model.StatusTranslated},
		{Source: "Untranslated", Target: "", Status: model.StatusUntranslated},
		{Source: "Identity", Target: "Identity", Status: model.StatusTranslated},
	}

	if err := Deploy(game, entries, DeployOptions{LanguageName: "Thai"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(game, "BepInEx", "Translation", "th", "Text", "lingo_translations.txt"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "Hello=สวัสดี\n") {
		t.Errorf("missing translated pair, got:\n%s", s)
	}
	if !strings.Contains(s, "Multi\\nline=หลาย\\nบรรทัด\n") {
		t.Errorf("newline escaping broken, got:\n%s", s)
	}
	if strings.Contains(s, "Untranslated=") || strings.Contains(s, "Identity=Identity") {
		t.Errorf("empty/identity targets must be skipped, got:\n%s", s)
	}
}

func TestDeployRequiresBepInEx(t *testing.T) {
	game := t.TempDir()
	err := Deploy(game, nil, DeployOptions{LanguageName: "th"})
	if err == nil || !strings.Contains(err.Error(), "BepInEx") {
		t.Fatalf("expected BepInEx-missing error, got %v", err)
	}
}

func TestDeploySyncsCfgLanguage(t *testing.T) {
	game := t.TempDir()
	if err := os.MkdirAll(filepath.Join(game, "BepInEx", "plugins"), 0755); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(game, "BepInEx", "config")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := "[General]\nLanguage=Japanese\nFromLanguage=ja\nEndpoint=\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "XUnity.AutoTranslator.cfg"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	entries := []model.TextEntry{{Source: "Hello", Target: "สวัสดี", Status: model.StatusTranslated}}
	if err := Deploy(game, entries, DeployOptions{LanguageName: "Thai"}); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(filepath.Join(game, "BepInEx", "Translation", "th", "Text", "lingo_translations.txt"))
	if err != nil {
		t.Fatalf("translation file: %v", err)
	}
	if !strings.Contains(string(out), "Hello=สวัสดี\n") {
		t.Errorf("missing pair, got:\n%s", out)
	}
	got, err := os.ReadFile(filepath.Join(cfgDir, "XUnity.AutoTranslator.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Language=th\n") {
		t.Errorf("cfg Language not synced to folder, got:\n%s", got)
	}
}
