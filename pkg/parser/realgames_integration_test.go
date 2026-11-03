//go:build integration

// Real-game integration tests: exercises every engine parser against real game
// data pulled from Chanomhub catalog games (see testdata/real-games/README.md).
// Run: go test -tags integration ./pkg/parser/...
package parser_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lingo-translate/pkg/parser"
)

const fixturesDir = "../../testdata/real-games"

func fixture(t *testing.T, parts ...string) string {
	t.Helper()
	dir := filepath.Join(append([]string{fixturesDir}, parts...)...)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("fixture missing: %s", dir)
	}
	return dir
}

func mustDetect(t *testing.T, want string, dir string) {
	t.Helper()
	name, err := parser.DetectEngine(dir)
	if err != nil {
		t.Fatalf("DetectEngine(%s) failed: %v", dir, err)
	}
	if name.Name() != want {
		t.Fatalf("DetectEngine(%s) = %s, want %s", dir, name.Name(), want)
	}
}

func extractSmoke(t *testing.T, dir string, minEntries int, mustContain string) int {
	t.Helper()
	p, err := parser.DetectEngine(dir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	entries, stats, err := p.Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(entries) < minEntries {
		t.Fatalf("expected >= %d entries from %s, got %d", minEntries, dir, len(entries))
	}
	if stats != nil && stats.Duration <= 0 {
		t.Errorf("stats.Duration not set")
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Source, mustContain) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected some source to contain %q; none of %d entries matched", mustContain, len(entries))
	}
	return len(entries)
}

func TestReal_RPGM_NokosLuckyRockPaperScissors(t *testing.T) {
	dir := fixture(t, "rpgm")
	mustDetect(t, "rpgm", dir)
	// Noko's lucky rock-paper-scissors! (HJ1045) — the game behind the first published mod
	n := extractSmoke(t, dir, 100, "布の服")
	t.Logf("rpgm: %d entries", n)
}

func TestReal_Renpy_TheBloodDiary(t *testing.T) {
	dir := fixture(t, "renpy")
	mustDetect(t, "renpy", dir)
	// The Blood Diary (HJ933): Russian source scripts extracted from the real .rpa
	p, _ := parser.GetParser("renpy")
	entries, stats, err := p.Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(entries) < 50 {
		t.Fatalf("expected >= 50 entries from The Blood Diary scripts, got %d", len(entries))
	}
	t.Logf("renpy: %d entries in %v", len(entries), stats.Duration)

	// Russian dialogue actually parsed (not only engine files)
	foundRussian := false
	for _, e := range entries {
		if strings.ContainsAny(e.Source, "абвгдежз") {
			foundRussian = true
			break
		}
	}
	if !foundRussian {
		t.Errorf("no Cyrillic sources found — scenario scripts were not parsed")
	}
}

func TestReal_Unity_BunnyQuotaStruggles(t *testing.T) {
	dir := fixture(t, "unity")
	mustDetect(t, "unity", dir)
	// Real shipped builds keep text in serialized assets; the parser's plaintext
	// surface (StreamingAssets json/csv, XUnity Translation) legitimately finds
	// nothing here. Detect must still classify the game correctly, and Extract
	// must not error on a real build tree.
	p, _ := parser.GetParser("unity")
	if _, _, err := p.Extract(context.Background(), dir); err != nil {
		t.Fatalf("extract on real build should not error, got: %v", err)
	}
}

func TestReal_Godot_PvZSexLawn(t *testing.T) {
	dir := fixture(t, "godot")
	mustDetect(t, "godot", dir)
	// Scripts live inside the binary .pck; plaintext extract is legitimately empty.
	// Detect off the real .pck surface is the contract under test.
	p, _ := parser.GetParser("godot")
	if _, _, err := p.Extract(context.Background(), dir); err != nil {
		t.Fatalf("extract on real build should not error, got: %v", err)
	}
}

func TestReal_LibGDX_TalesOfAndrogyny(t *testing.T) {
	dir := fixture(t, "libgdx")
	mustDetect(t, "libgdx", dir)
	// Tales of Androgyny (HJ883): real encounters.json (11.7MB), strings.properties
	// (424KB) and encounter-trees.json pulled out of the shipped TalesOfAndrogyny.jar
	n := extractSmoke(t, dir, 1000, "Helloooooo hero!")
	t.Logf("libgdx: %d entries", n)
}

func TestReal_AllFixturesDetect(t *testing.T) {
	// DetectEngine must classify every real fixture without ambiguity
	cases := []struct{ subdir, want string }{
		{"rpgm", "rpgm"},
		{"renpy", "renpy"},
		{"unity", "unity"},
		{"godot", "godot"},
		{"libgdx", "libgdx"},
	}
	for _, c := range cases {
		dir := fixture(t, c.subdir)
		mustDetect(t, c.want, dir)
	}
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func TestReal_UnityJSONSkipsEnginePropKeys(t *testing.T) {
	// Real regression from Bunny Quota Struggles: Addressables settings.json listed
	// m_buildTarget="StandaloneWindows64" as a translatable entry — engine property
	// keys (m_*) must never reach the workspace.
	p, _ := parser.GetParser("unity")
	entries, _, err := p.Extract(context.Background(), fixture(t, "unity"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Source == "StandaloneWindows64" || strings.HasPrefix(e.KeyPath, "m_") {
			t.Errorf("engine prop leaked: key=%q source=%q", e.KeyPath, e.Source)
		}
	}
}
