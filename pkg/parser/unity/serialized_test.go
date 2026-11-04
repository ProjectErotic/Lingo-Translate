package unity

import "testing"

func TestScanSerializedStrings(t *testing.T) {
	// build a buffer: noise + len-prefixed "こんにちは" (15 bytes, pad 1) + noise + "ok"(needs space rule → filtered later)
	data := []byte{}
	data = append(data, 0xde, 0xad, 0xbe, 0xef)
	data = append(data, 0x0f, 0, 0, 0)
	data = append(data, []byte("こんにちは")...)
	data = append(data, 0x00) // pad
	data = append(data, 0xca, 0xfe, 0xba, 0xbe)
	var got []string
	scanSerializedStrings(data, func(off int, s string) {
		got = append(got, s)
	})
	if len(got) != 1 || got[0] != "こんにちは" {
		t.Fatalf("scan got %v", got)
	}

	// trailing padding nonzero → reject
	bad := append([]byte{0x0f, 0, 0, 0}, []byte("こんにちは")...)
	bad = append(bad, 0xff, 0xca, 0xfe, 0xba)
	n := 0
	scanSerializedStrings(bad, func(int, string) { n++ })
	if n != 0 {
		t.Fatal("nonzero padding accepted")
	}
}

func TestPlausibleText(t *testing.T) {
	yes := []string{"キャンセル", "You look pale. I", "体力（黄色）：行動すると減少", "Stage: Day"}
	no := []string{"player_default", "Fire1", "BunnyQuotaStruggles", "0f9e8d7c123456789abcdef012345678", "Assets/scenes/x.unity", "horizontal"}
	for _, s := range yes {
		if !plausibleText(s) {
			t.Errorf("plausibleText(%q)=false want true", s)
		}
	}
	for _, s := range no {
		if plausibleText(s) {
			t.Errorf("plausibleText(%q)=true want false", s)
		}
	}
}

func TestIsSerializedAssetFile(t *testing.T) {
	yes := []string{"/g/BunnyQuotaStruggles_Data/resources.assets", "/g/BunnyQuotaStruggles_Data/level3",
		"/g/BunnyQuotaStruggles_Data/sharedassets2.assets"}
	no := []string{"/g/BunnyQuotaStruggles_Data/globalgamemanagers", "/g/BunnyQuotaStruggles_Data/globalgamemanagers.assets",
		"/g/BunnyQuotaStruggles_Data/sharedassets0.assets.resS", "/g/BunnyQuotaStruggles_Data/unity default resources.resource",
		"/g/game.exe"}
	for _, s := range yes {
		if !isSerializedAssetFile(s) {
			t.Errorf("isSerializedAssetFile(%q)=false want true", s)
		}
	}
	for _, s := range no {
		if isSerializedAssetFile(s) {
			t.Errorf("isSerializedAssetFile(%q)=true want false", s)
		}
	}
}
