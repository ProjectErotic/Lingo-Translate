package godot

// PCK injection: rebuild the game's .pck with translated strings.
// Every string constant in every .gdc (and every encoded string in binary
// resources) whose source matches a translated entry is rewritten; matching
// by value means duplicate strings inside other scripts are patched too.
// The patched pack is written to outputDir (or over the original when no
// separate output dir is given).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lingo-translate/pkg/model"
)

func (p *Parser) injectPck(ctx context.Context, sourceDir, outputDir, pckPath string, entries []model.TextEntry) error {
	repl := map[string]string{}
	for _, e := range entries {
		if e.Target != "" && e.Target != e.Source {
			repl[e.Source] = e.Target
		}
	}

	pckEntries, read, err := ReadPck(pckPath)
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	patchedGdc, patchedBin := 0, 0
	for _, e := range pckEntries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		data, err := read(e)
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Path, err)
		}
		ext := strings.ToLower(filepath.Ext(e.Path))
		if ext == ".gdc" || ext == ".gdsc" {
			if g, err := DecodeGDC(data); err == nil {
				changed := false
				for i := range g.Constants {
					c := &g.Constants[i]
					if (c.Type == vpString || c.Type == vpStringName) && c.Str != "" {
						if tgt, ok := repl[c.Str]; ok {
							c.Str = tgt
							c.Mod = true
							changed = true
						}
					}
				}
				if changed {
					if out, err := g.EncodeGDC(); err == nil {
						data = out
						patchedGdc++
					}
				}
			}
		} else if binaryTextExts[ext] {
			if out := patchBinaryStrings(data, repl); out != nil {
				data = out
				patchedBin++
			}
		}
		files[e.Path] = data
	}

	dest := outputDir
	if dest == "" || dest == sourceDir {
		dest = sourceDir
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	outPath := filepath.Join(dest, filepath.Base(pckPath))
	if err := WritePck(outPath, files); err != nil {
		return err
	}
	fmt.Printf("[godot] patched %d scripts, %d resources -> %s\n", patchedGdc, patchedBin, outPath)
	return nil
}
