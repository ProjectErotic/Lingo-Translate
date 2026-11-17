package engine

import (
	"os"
	"strings"
	"testing"
)

func TestGameMakerInspector_Adulttale(t *testing.T) {
	gameDir := "/home/jop/Downloads/New Folder (77)/Adulttale-0.29.2-HOTFIX"
	if _, err := os.Stat(gameDir); os.IsNotExist(err) {
		t.Skip("Adulttale game directory not found, skipping live engine inspection test")
	}

	inspector, err := NewGameMakerInspector(gameDir)
	if err != nil {
		t.Fatalf("failed to create GameMaker inspector: %v", err)
	}

	// 1. Test data.win parsing
	info, err := inspector.ParseDataWin()
	if err != nil {
		t.Fatalf("ParseDataWin failed: %v", err)
	}
	t.Logf("Found %d chunks in data.win: %v", len(info.Chunks), info.Chunks[:8])
	t.Logf("Total strings in data.win: %d", info.TotalStrings)
	t.Logf("Sample template strings: %d found", len(info.Templates))

	if info.TotalStrings < 10000 {
		t.Errorf("expected >10000 strings, got %d", info.TotalStrings)
	}

	// 2. Test Key Analysis on Chara act check
	targetKey := "lang_dialogue_battle_chara_act_check_0_0"
	ctx, err := inspector.AnalyzeKey(targetKey)
	if err != nil {
		t.Fatalf("AnalyzeKey failed: %v", err)
	}

	t.Logf("=== Key Analysis for %s ===", targetKey)
	t.Logf("Engine: %s, TrailingSpaceNeeded: %v, DynamicSuffix: %s", ctx.EngineType, ctx.TrailingSpaceNeeded, ctx.DynamicSuffix)
	t.Logf("Callers found (%d):", len(ctx.Callers))
	for _, c := range ctx.Callers {
		t.Logf("  -> %s", c)
	}

	if !ctx.TrailingSpaceNeeded {
		t.Errorf("expected TrailingSpaceNeeded = true for %s", targetKey)
	}
	if ctx.DynamicSuffix != "%" {
		t.Errorf("expected DynamicSuffix = '%%', got %q", ctx.DynamicSuffix)
	}

	foundDataManager := false
	for _, c := range ctx.Callers {
		if strings.Contains(c, "DataManager") {
			foundDataManager = true
			break
		}
	}
	if !foundDataManager {
		t.Errorf("expected DataManager caller to be detected in callers list")
	}

	// 3. Test In-Game Simulated Render
	translation := "ระดับความเงี่ยนของเธออยู่ที่ "
	simulated := SimulateInGameRender(ctx, translation)
	t.Logf("Simulated in-game engine output: %q", simulated)

	expectedSimulated := "ระดับความเงี่ยนของเธออยู่ที่ 20%"
	if simulated != expectedSimulated {
		t.Errorf("expected simulated render %q, got %q", expectedSimulated, simulated)
	}

	// 4. Test Full Report Generation
	sampleKeys := []string{
		"lang_dialogue_battle_chara_act_check_0_0",
		"tl_dialogue_chara_talk_{0}",
	}
	report, err := inspector.GenerateFullReport(sampleKeys)
	if err != nil {
		t.Fatalf("GenerateFullReport failed: %v", err)
	}

	t.Logf("=== Engine Report ===")
	t.Logf("Engine: %s, IsYYC: %v, TotalStrings: %d, TotalScripts: %d",
		report.EngineType, report.IsYYC, report.TotalStrings, report.TotalScripts)

	if !report.IsYYC {
		t.Errorf("expected Adulttale to be detected as GameMaker YYC")
	}
	if report.TotalScripts < 100 {
		t.Errorf("expected >100 scripts in YYC binary, got %d", report.TotalScripts)
	}
}

func BenchmarkGameMakerAnalyzeKey(b *testing.B) {
	gameDir := "/home/jop/Downloads/New Folder (77)/Adulttale-0.29.2-HOTFIX"
	if _, err := os.Stat(gameDir); os.IsNotExist(err) {
		b.Skip("Adulttale game directory not found")
	}

	inspector, err := NewGameMakerInspector(gameDir)
	if err != nil {
		b.Fatalf("failed to create inspector: %v", err)
	}

	targetKey := "lang_dialogue_battle_chara_act_check_0_0"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = inspector.AnalyzeKey(targetKey)
	}
}
