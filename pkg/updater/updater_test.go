package updater

import (
	"testing"
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
