package updater

import (
	"testing"

	"lingo-translate/pkg/translator/prompts"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest   string
		current  string
		expected bool
	}{
		{"2.4.1", "2.4.0", true},
		{"2.5.0", "2.4.0", true},
		{"3.0.0", "2.4.0", true},
		{"2.4.0", "2.4.0", false},
		{"2.3.0", "2.4.0", false},
		{"2.4.0.1", "2.4.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.latest+"_vs_"+tt.current, func(t *testing.T) {
			got := isNewer(tt.latest, tt.current)
			if got != tt.expected {
				t.Errorf("isNewer(%q, %q) = %v, expected %v", tt.latest, tt.current, got, tt.expected)
			}
		})
	}
}

func TestFindMatchingAsset(t *testing.T) {
	assets := []githubAsset{
		{Name: "lingo-v2.4.0-linux-amd64.tar.gz", DownloadURL: "https://example.com/linux-amd64.tar.gz"},
		{Name: "lingo-v2.4.0-linux-arm64.tar.gz", DownloadURL: "https://example.com/linux-arm64.tar.gz"},
		{Name: "lingo-v2.4.0-windows-amd64.zip", DownloadURL: "https://example.com/windows-amd64.zip"},
		{Name: "lingo-v2.4.0-darwin-arm64.tar.gz", DownloadURL: "https://example.com/darwin-arm64.tar.gz"},
	}

	assetLinux := findMatchingAsset(assets, "linux", "amd64")
	if assetLinux == nil || assetLinux.Name != "lingo-v2.4.0-linux-amd64.tar.gz" {
		t.Errorf("expected linux amd64 asset, got %v", assetLinux)
	}

	assetWin := findMatchingAsset(assets, "windows", "amd64")
	if assetWin == nil || assetWin.Name != "lingo-v2.4.0-windows-amd64.zip" {
		t.Errorf("expected windows amd64 asset, got %v", assetWin)
	}

	assetUnknown := findMatchingAsset(assets, "solaris", "sparc")
	if assetUnknown != nil {
		t.Errorf("expected nil for unsupported platform, got %v", assetUnknown)
	}
}

func TestMergeLexicons(t *testing.T) {
	local := &prompts.NSFWLexicon{
		Sensations: []string{"เสียว", "คำเฉพาะของผู้ใช้"},
		Anatomy: prompts.AnatomyTerms{
			Male: []string{"ดุ้น"},
		},
		Custom: []string{"คำพิเศษ"},
	}

	remote := &prompts.NSFWLexicon{
		Sensations: []string{"เสียว", "ฟินจนตาค้าง"},
		Anatomy: prompts.AnatomyTerms{
			Male:   []string{"แท่งร้อน", "ดุ้น"},
			Female: []string{"ร่องสวาท"},
		},
	}

	merged := mergeLexicons(local, remote)

	// Local user edits should be preserved
	if len(merged.Sensations) != 3 {
		t.Errorf("expected 3 sensations, got %d", len(merged.Sensations))
	}
	if merged.Sensations[0] != "เสียว" || merged.Sensations[1] != "คำเฉพาะของผู้ใช้" || merged.Sensations[2] != "ฟินจนตาค้าง" {
		t.Errorf("unexpected sensations order/content: %v", merged.Sensations)
	}

	// Anatomy should merge both
	if len(merged.Anatomy.Male) != 2 {
		t.Errorf("expected 2 male terms, got %d", len(merged.Anatomy.Male))
	}
	if len(merged.Anatomy.Female) != 1 {
		t.Errorf("expected 1 female term, got %d", len(merged.Anatomy.Female))
	}

	// Custom array from local must be kept
	if len(merged.Custom) != 1 || merged.Custom[0] != "คำพิเศษ" {
		t.Errorf("expected local custom terms to be preserved, got %v", merged.Custom)
	}

	if countTerms(merged) != 7 {
		t.Errorf("expected 7 total terms, got %d", countTerms(merged))
	}
}
