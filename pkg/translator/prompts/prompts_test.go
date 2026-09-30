package prompts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt(PersonaDanUncensored, "Japanese", "Thai")
	if !strings.Contains(prompt, "FICTIONAL LOCALIZATION ONLY") {
		t.Errorf("expected DanUncensored prompt to contain fiction directive")
	}
	if !strings.Contains(prompt, "Japanese to Thai") {
		t.Errorf("expected language pair in prompt")
	}

	vnPrompt := BuildSystemPrompt(PersonaVisualNovelRomance, "Japanese", "Thai")
	if !strings.Contains(vnPrompt, "VISUAL NOVEL") {
		t.Errorf("expected VN prompt to contain VISUAL NOVEL")
	}

	stdPrompt := BuildSystemPrompt(PersonaStandard, "English", "Thai")
	if !strings.Contains(stdPrompt, "LOCALIZATION GUIDELINES") {
		t.Errorf("expected standard prompt")
	}

	// 1. Thai NSFW
	nsfwThai := BuildSystemPrompt(PersonaNSFW, "Japanese", "Thai")
	if !strings.Contains(nsfwThai, "ADULT & NSFW LOCALIZATION") {
		t.Errorf("expected NSFW prompt to contain NSFW directive")
	}
	if !strings.Contains(nsfwThai, "THAI SPECIFIC GUIDANCE") {
		t.Errorf("expected Thai specific guidance")
	}
	if !strings.Contains(nsfwThai, "เสียวซ่าน") {
		t.Errorf("expected Thai NSFW prompt to contain Thai keywords")
	}

	// 2. English NSFW
	nsfwEnglish := BuildSystemPrompt(PersonaNSFW, "Japanese", "English")
	if !strings.Contains(nsfwEnglish, "ENGLISH SPECIFIC GUIDANCE") {
		t.Errorf("expected English specific guidance")
	}
	if !strings.Contains(nsfwEnglish, "throbbing") {
		t.Errorf("expected English NSFW prompt to contain English keywords")
	}

	// 3. Chinese NSFW
	nsfwChinese := BuildSystemPrompt(PersonaNSFW, "Japanese", "Chinese")
	if !strings.Contains(nsfwChinese, "CHINESE SPECIFIC GUIDANCE") {
		t.Errorf("expected Chinese specific guidance")
	}
	if !strings.Contains(nsfwChinese, "酥麻") {
		t.Errorf("expected Chinese NSFW prompt to contain Chinese keywords")
	}
}

func TestResolvePrompt_CustomTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lingo_template_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tplPath := filepath.Join(tmpDir, "my_nsfw.txt")
	_ = os.WriteFile(tplPath, []byte("Translate with intense passion and explicit terms"), 0644)

	res := ResolvePrompt(tplPath, "en", "th", "")
	if !strings.Contains(res, "Translate with intense passion and explicit terms") {
		t.Errorf("expected custom template to be resolved")
	}
}

func TestNormalizeLangCode(t *testing.T) {
	cases := map[string]string{
		"Thai":       "th",
		"th-TH":      "th",
		"English":    "en",
		"en-US":      "en",
		"Chinese":    "zh",
		"zh-CN":      "zh",
		"Japanese":   "ja",
		"Spanish":    "es",
		"unknown-xx": "unknown-xx",
	}

	for input, expected := range cases {
		got := NormalizeLangCode(input)
		if got != expected {
			t.Errorf("NormalizeLangCode(%s) = %s, expected %s", input, got, expected)
		}
	}
}

func TestLoadNSFWLexicon_MultiLang(t *testing.T) {
	// Thai
	thLex := DefaultNSFWLexicon("th")
	if len(thLex.Sensations) == 0 || thLex.Sensations[0] != "เสียวซ่าน" {
		t.Errorf("expected Thai lexicon to start with เสียวซ่าน")
	}

	// English
	enLex := DefaultNSFWLexicon("en")
	if len(enLex.Sensations) == 0 || enLex.Sensations[0] != "throbbing" {
		t.Errorf("expected English lexicon to start with throbbing")
	}

	// Chinese
	zhLex := DefaultNSFWLexicon("zh")
	if len(zhLex.Sensations) == 0 || zhLex.Sensations[0] != "酥麻" {
		t.Errorf("expected Chinese lexicon to start with 酥麻")
	}

	// Formatted
	thFormatted := FormatNSFWLexiconForPrompt(thLex, "th")
	if !strings.Contains(thFormatted, "Thai 18+ Localization Reference") {
		t.Errorf("expected Thai header in formatted prompt")
	}

	enFormatted := FormatNSFWLexiconForPrompt(enLex, "en")
	if !strings.Contains(enFormatted, "English 18+ Localization Reference") {
		t.Errorf("expected English header in formatted prompt")
	}
}

func TestNSFWLexicon_Merge(t *testing.T) {
	base := []string{"เสียว", "ฟิน"}
	custom := []string{"ฟิน", "ตอดรัดแน่น"}
	merged := mergeSlice(base, custom)

	if len(merged) != 3 {
		t.Errorf("expected 3 unique items, got %d", len(merged))
	}
	expected := []string{"เสียว", "ฟิน", "ตอดรัดแน่น"}
	for i, v := range expected {
		if merged[i] != v {
			t.Errorf("expected item %d to be %s, got %s", i, v, merged[i])
		}
	}
}

func TestNSFWLexicon_CustomFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lingo_lexicon_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	customLex := NSFWLexicon{
		Sensations: []string{"เสียวสะท้านทั้งร่าง"},
		Custom:     []string{"คำแสลงพิเศษ18+"},
	}
	data, _ := json.Marshal(customLex)
	lexPath := filepath.Join(tmpDir, "nsfw_lexicon.json")
	if err := os.WriteFile(lexPath, data, 0644); err != nil {
		t.Fatalf("failed to write custom lexicon: %v", err)
	}

	raw, err := os.ReadFile(lexPath)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	var loaded NSFWLexicon
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("failed to unmarshal test file: %v", err)
	}

	defaultLex := DefaultNSFWLexicon("th")
	defaultLex.Sensations = mergeSlice(defaultLex.Sensations, loaded.Sensations)
	defaultLex.Custom = mergeSlice(defaultLex.Custom, loaded.Custom)

	formatted := FormatNSFWLexiconForPrompt(defaultLex, "th")
	if !strings.Contains(formatted, "เสียวสะท้านทั้งร่าง") {
		t.Errorf("expected formatted prompt to contain custom sensation")
	}
	if !strings.Contains(formatted, "คำแสลงพิเศษ18+") {
		t.Errorf("expected formatted prompt to contain custom term")
	}
}
