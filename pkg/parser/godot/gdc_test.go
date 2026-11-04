package godot

// Self-check for pck v4 + .gdc decode/encode round-trips (stdlib only).

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// buildGdc assembles a minimal .gdc buffer: 1 identifier, 3 string constants,
// no tokens (header counts stay consistent with what we write).
func buildGdc(strs []string) []byte {
	var contents bytes.Buffer
	var l4 [4]byte
	binary.LittleEndian.PutUint32(l4[:], 1) // 1 identifier
	contents.Write(l4[:])
	binary.LittleEndian.PutUint32(l4[:], uint32(len(strs)))
	contents.Write(l4[:])
	binary.LittleEndian.PutUint32(l4[:], 0) // token lines
	contents.Write(l4[:])
	binary.LittleEndian.PutUint32(l4[:], 0) // tokens
	contents.Write(l4[:])
	// identifier "foo" as UTF-32 XOR 0xB6
	binary.LittleEndian.PutUint32(l4[:], 3)
	contents.Write(l4[:])
	for _, r := range "foo" {
		var c [4]byte
		binary.LittleEndian.PutUint32(c[:], uint32(r))
		for i := range c {
			c[i] ^= 0xB6
		}
		contents.Write(c[:])
	}
	for _, s := range strs {
		binary.LittleEndian.PutUint32(l4[:], 4) // Variant::STRING
		contents.Write(l4[:])
		binary.LittleEndian.PutUint32(l4[:], uint32(len(s)))
		contents.Write(l4[:])
		contents.WriteString(s)
		if pad := (4 - len(s)%4) % 4; pad != 0 {
			contents.Write(make([]byte, pad))
		}
	}

	zenc, _ := zstd.NewWriter(nil)
	out := []byte("GDSC")
	binary.LittleEndian.PutUint32(l4[:], 101)
	out = append(out, l4[:]...)
	binary.LittleEndian.PutUint32(l4[:], uint32(contents.Len()))
	out = append(out, l4[:]...)
	return zenc.EncodeAll(contents.Bytes(), out)
}

func TestGDCRoundTrip(t *testing.T) {
	strs := []string{"¡Hola!", "こんにちは世界", "x", "pad-me-please", ""}
	raw := buildGdc(strs)
	g, err := DecodeGDC(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Constants) != len(strs) {
		t.Fatalf("constants: got %d want %d", len(g.Constants), len(strs))
	}
	for i, s := range strs {
		if g.Constants[i].Str != s {
			t.Errorf("const %d: got %q want %q", i, g.Constants[i].Str, s)
		}
	}
	if g.Identifiers[0] != "foo" {
		t.Errorf("identifier: got %q", g.Identifiers[0])
	}
	// mutate one constant, re-encode, decode again
	g.Constants[1].Str = "สวัสดี"
	g.Constants[1].Mod = true
	out, err := g.EncodeGDC()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	g2, err := DecodeGDC(out)
	if err != nil {
		t.Fatalf("redecode: %v", err)
	}
	if g2.Constants[1].Str != "สวัสดี" {
		t.Errorf("mutated const: got %q", g2.Constants[1].Str)
	}
	if g2.Constants[0].Str != strs[0] || g2.Constants[2].Str != strs[2] {
		t.Error("other constants changed")
	}
}

func TestBinaryStringPatch(t *testing.T) {
	// encoded Variant::STRING "hello" (5 bytes + 3 pad) inside noise
	var buf bytes.Buffer
	buf.Write([]byte{0xde, 0xad, 0xbe, 0xef})
	var l4 [4]byte
	binary.LittleEndian.PutUint32(l4[:], 4)
	buf.Write(l4[:])
	binary.LittleEndian.PutUint32(l4[:], 5)
	buf.Write(l4[:])
	buf.WriteString("hello")
	buf.Write([]byte{0, 0, 0})
	buf.Write([]byte{0xca, 0xfe, 0xba, 0xbe})
	data := buf.Bytes()

	found := ""
	scanBinaryStrings(data, func(off int, s string) {
		found = s
		if s == "hello" {
			data = patchBinaryStrings(data, map[string]string{"hello": "สวัสดี"})
		}
	})
	if found != "hello" {
		t.Fatalf("scan found %q", found)
	}
	if data == nil {
		t.Fatal("patch returned nil")
	}
	after := ""
	scanBinaryStrings(data, func(_ int, s string) {
		if s == "สวัสดี" {
			after = s
		}
	})
	if after != "สวัสดี" {
		t.Errorf("patched string not found after rewrite")
	}
	// noise prefix/suffix must survive
	if data[0] != 0xde || data[3] != 0xef {
		t.Error("prefix corrupted")
	}
	if data[len(data)-1] != 0xbe {
		t.Error("suffix corrupted")
	}
}

func TestIsGameText(t *testing.T) {
	yes := []string{"Game Over", "¡Hola, mundo!", "เริ่มเกม", "Level - 000"}
	no := []string{"res://scenes/Board.tscn", "uid://8h2k", "start_cooldown", "", "x", "%s, %d"}
	for _, s := range yes {
		if !isGameText(s) {
			t.Errorf("isGameText(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isGameText(s) {
			t.Errorf("isGameText(%q) = true, want false", s)
		}
	}
}
