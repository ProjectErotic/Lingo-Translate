package unity

// Serialized Unity build files (.assets, level*, sharedassets*, resources.assets)
// store MonoBehaviour string fields as length-prefixed UTF-8. Full object
// parsing needs the typetree (not shipped in release builds) — instead we walk
// every plausible length-prefix and accept it when the following bytes decode
// as real text. False positives are filtered by the same isTranslatable rules
// used for YAML, and re-parsing on inject is value-matched so a stray hit only
// wastes a workspace row, never corrupts the game.
//
// ponytail: heuristic scan, no typetree — upgrade to a full
// AssetsTools/UnityPy typetree parse only if a game ships text this misses.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"lingo-translate/pkg/model"
)

// scanSerializedStrings walks a serialized file and yields every
// length-prefixed string that decodes as plausible text.
// Format: u32 length + UTF-8 bytes + zero padding to 4-byte alignment.
func scanSerializedStrings(data []byte, visit func(off int, s string)) {
	n := len(data)
	for i := 0; i+4 <= n; i += 4 {
		l := int(int32(leU32(data, i)))
		if l <= 0 || l > 1<<20 || i+4+l > n {
			continue
		}
		// padding to 4-byte alignment after the string must be zero
		pad := (4 - l%4) % 4
		if i+4+l+pad > n {
			continue
		}
		good := true
		for j := 0; j < pad; j++ {
			if data[i+4+l+j] != 0 {
				good = false
				break
		}
		}
		if !good {
			continue
		}
		s := string(data[i+4 : i+4+l])
		if !utf8.ValidString(s) || !plausibleText(s) {
			continue
		}
		visit(i, s)
	}
}

func leU32(b []byte, o int) uint32 {
	return uint32(b[o]) | uint32(b[o+1])<<8 | uint32(b[o+2])<<16 | uint32(b[o+3])<<24
}

// plausibleText filters garbage: needs a letter/digit, no control chars,
// not a GUID/path/identifier-only string.
func plausibleText(s string) bool {
	t := strings.TrimSpace(s)
	if len(t) < 2 {
		return false
	}
	hasAlnum := false
	for _, r := range t {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasAlnum = true
			break
		}
	}
	if !hasAlnum {
		return false
	}
	for _, r := range t {
		if r < 0x20 && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
	}
	if guidRegex.MatchString(t) {
		return false
	}
	// pure lowercase snake_case = identifier
	if !strings.ContainsAny(t, " \n") && strings.Contains(t, "_") && t == strings.ToLower(t) {
		return false
	}
	// res-like paths and ids
	if strings.HasPrefix(t, "res://") || strings.HasPrefix(t, "uid://") ||
		strings.HasPrefix(t, "Assets/") || strings.HasPrefix(t, "Packages/") {
		return false
	}
	// Serialized engine files are full of setting labels, axis names, class
	// names. Real player-facing text almost always contains a space or a
	// CJK/Thai/etc non-ASCII letter; keep those, drop the rest.
	if !strings.ContainsAny(t, " \n") {
		nonASCII := false
		for _, r := range t {
			if r > 0x7f && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
				nonASCII = true
				break
			}
		}
		if !nonASCII {
			return false
		}
	}
	return true
}

// isSerializedAssetFile reports whether path is a Unity serialized build file
// worth scanning (assets containers and scene/level files).
// globalgamemanagers* is engine config (input axes, quality levels) — never
// game text, and it floods the workspace with noise, so it's skipped outright.
func isSerializedAssetFile(path string) bool {
	if strings.HasSuffix(path, ".resS") || strings.HasSuffix(path, ".resource") {
		return false
	}
	name := filepath.Base(path)
	if strings.HasPrefix(name, "globalgamemanagers") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".assets" {
		return true
	}
	// scene files in built games: level0, level1, ... and sharedassets
	return ext == "" && (strings.HasPrefix(name, "level") || strings.HasPrefix(name, "sharedassets"))
}

// extractSerialized scans one serialized Unity file for translatable strings.
func (p *Parser) extractSerialized(ctx context.Context, path, relPath string, entries *[]model.TextEntry, unique map[string]bool, stats *int) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < 64 {
		return
	}
	// .resS files are raw audio/texture blobs — skip by extension.
	if strings.HasSuffix(path, ".resS") {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	now := time.Now()
	*stats++
	scanSerializedStrings(data, func(off int, s string) {
		if !isTranslatable(s) {
			return
		}
		unique[s] = true
		*entries = append(*entries, model.TextEntry{
			ID:        fmt.Sprintf("%s:%x", relPath, off),
			Source:    s,
			FilePath:  relPath,
			KeyPath:   fmt.Sprintf("off:%x", off),
			Status:    model.StatusUntranslated,
			Context:   "Serialized string",
			UpdatedAt: now,
		})
	})
}
