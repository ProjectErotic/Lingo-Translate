package godot

// Real-game extraction for exported Godot projects: read the .pck, decode
// compiled scripts (.gdc) and pull translatable strings out of string
// constants; also scan binary resources (.scn/.res) for encoded string
// variants. res:// paths and engine-internal strings are skipped.

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"lingo-translate/pkg/model"
)

var (
	gdResPath   = regexp.MustCompile(`^(res|uid)://`)
	gdLooksCode = regexp.MustCompile(`^[\[\]{}().,%*/+=<>!&|:;@#$\\^-]*$`)
	// scene files carry little user text but the few labels matter; scan them
	// with the same variant walker as .res.
	binaryTextExts = map[string]bool{".scn": true, ".res": true}
)

func (p *Parser) extractPck(ctx context.Context, pckPath string) ([]model.TextEntry, *model.ExtractionStats, error) {
	startTime := time.Now()
	entriesRef, read, err := ReadPck(pckPath)
	if err != nil {
		return nil, nil, err
	}
	unique := map[string]int{}
	var entries []model.TextEntry
	filesScanned := 0

	add := func(file, key, text, ctxKind string) {
		if text == "" || !isGameText(text) {
			return
		}
		unique[text]++
		entries = append(entries, model.TextEntry{
			ID:        fmt.Sprintf("%s:%s", file, key),
			Source:    text,
			FilePath:  file,
			KeyPath:   key,
			Status:    model.StatusUntranslated,
			Context:   ctxKind,
			UpdatedAt: time.Now(),
		})
	}

	for _, e := range entriesRef {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}
		ext := strings.ToLower(filepath.Ext(e.Path))
		name := filepath.Base(e.Path)
		switch {
		case ext == ".gdc" || ext == ".gdsc":
			filesScanned++
			raw, err := read(e)
			if err != nil {
				continue
			}
			g, err := DecodeGDC(raw)
			if err != nil {
				continue
			}
			for i, c := range g.Constants {
				if c.Type == vpString || c.Type == vpStringName {
					add(name, fmt.Sprintf("const:%d", i), c.Str, "GDScript constant")
				}
			}
		case binaryTextExts[ext]:
			filesScanned++
			raw, err := read(e)
			if err != nil {
				continue
			}
			scanBinaryStrings(raw, func(off int, s string) {
				add(name, fmt.Sprintf("var:%x", off), s, "Scene string")
			})
		}
	}

	// Deduplicate by source text: one workspace row per unique string, keyed
	// to the first file that carried it.
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	seen := map[string]bool{}
	var out []model.TextEntry
	totalWords := 0
	for _, e := range entries {
		if seen[e.Source] {
			continue
		}
		seen[e.Source] = true
		totalWords += len(strings.Fields(e.Source))
		out = append(out, e)
	}

	stats := &model.ExtractionStats{
		TotalEntries: len(out),
		UniqueTexts:  len(out),
		TotalWords:   totalWords,
		FilesScanned: filesScanned,
		Duration:     time.Since(startTime),
	}
	return out, stats, nil
}

// isGameText filters engine-internal strings so they don't reach the workspace.
func isGameText(s string) bool {
	t := strings.TrimSpace(s)
	if len(t) < 2 {
		return false
	}
	if gdResPath.MatchString(t) {
		return false
	}
	if strings.HasPrefix(t, "uid://") || strings.HasPrefix(t, "class_name") {
		return false
	}
	// Pure punctuation / operator soup = engine syntax, not text.
	if gdLooksCode.MatchString(t) {
		return false
	}
	// Must contain a real letter or digit (Thai/Japanese/etc included).
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
	// printf-style format strings are code.
	if strings.Count(t, "%") >= 2 {
		return false
	}
	// snake_case identifiers (no spaces) are code, not text.
	if !strings.ContainsAny(t, " \n") && strings.Contains(t, "_") {
		return false
	}
	return true
}

// scanBinaryStrings walks a binary resource buffer and calls visit for every
// encoded String variant that looks like game text.
func scanBinaryStrings(data []byte, visit func(off int, s string)) {
	for i := 0; i+8 <= len(data); i += 4 {
		hdr := leU32(data, i)
		if hdr&0xFF != vpString {
			continue
		}
		n := int(int32(leU32(data, i+4)))
		if n <= 0 || n > 1<<20 || i+8+n > len(data) {
			continue
		}
		pad := (4 - n%4) % 4
		if i+8+n+pad > len(data) {
			continue
		}
		good := true
		for j := 0; j < pad; j++ {
			if data[i+8+n+j] != 0 {
				good = false
				break
			}
		}
		if !good {
			continue
		}
		s := string(data[i+8 : i+8+n])
		if !isGameText(s) {
			continue
		}
		visit(i, s)
	}
}

// patchBinaryStrings rewrites string variants in a binary resource using the
// given map, preserving length+padding by only replacing equal-or-shorter
// targets padded with NUL... but Godot strings carry explicit lengths, so we
// can write a different-length string cleanly (header length + realign pad).
// Returns the new buffer, or nil when nothing changed.
func patchBinaryStrings(data []byte, repl map[string]string) []byte {
	type edit struct {
		start, end int
		newEnc     []byte
	}
	var edits []edit
	for i := 0; i+8 <= len(data); i += 4 {
		hdr := leU32(data, i)
		if hdr&0xFF != vpString {
			continue
		}
		n := int(int32(leU32(data, i+4)))
		if n <= 0 || n > 1<<20 || i+8+n > len(data) {
			continue
		}
		pad := (4 - n%4) % 4
		if i+8+n+pad > len(data) {
			continue
		}
		s := string(data[i+8 : i+8+n])
		tgt, ok := repl[s]
		if !ok || tgt == s {
			continue
		}
		tpad := (4 - len(tgt)%4) % 4
		enc := make([]byte, 0, 8+len(tgt)+tpad)
		enc = append(enc, data[i:i+8]...)
		binary.LittleEndian.PutUint32(enc[4:], uint32(len(tgt)))
		enc = append(enc, tgt...)
		enc = append(enc, make([]byte, tpad)...)
		edits = append(edits, edit{i, i + 8 + n + pad, enc})
	}
	if len(edits) == 0 {
		return nil
	}
	out := make([]byte, 0, len(data)+64)
	last := 0
	for _, e := range edits {
		out = append(out, data[last:e.start]...)
		out = append(out, e.newEnc...)
		last = e.end
	}
	out = append(out, data[last:]...)
	return out
}
