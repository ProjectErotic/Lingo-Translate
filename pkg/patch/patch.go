package patch

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/parser"
)

// PatchPackage represents a lightweight, portable distribution mod package (.patch.json or .patch.json.gz)
type PatchPackage struct {
	FormatVersion string            `json:"format_version"` // "1.0"
	Engine        string            `json:"engine"`         // e.g. "rpgm", "renpy", "unity"
	GameTitle     string            `json:"game_title,omitempty"`
	GameVersion   string            `json:"game_version,omitempty"`
	ChanomhubSlug string            `json:"chanomhub_slug,omitempty"`
	SourceLang    string            `json:"source_lang"`
	TargetLang    string            `json:"target_lang"`
	CreatedAt     time.Time         `json:"created_at"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Stats         PatchStats        `json:"stats"`
	Entries       []PatchEntry      `json:"entries"`
}

// PatchStats contains metrics for the patch package
type PatchStats struct {
	TotalEntries      int `json:"total_entries"`
	TranslatedEntries int `json:"translated_entries"`
	UniqueTexts       int `json:"unique_texts"`
}

// PatchEntry represents a single localized string node
type PatchEntry struct {
	FilePath string `json:"file"`
	KeyPath  string `json:"key"`
	Source   string `json:"src"`
	Target   string `json:"tgt"`
}

// SavePatch saves a PatchPackage to disk. Compresses with gzip if target filename ends with .gz
func SavePatch(p *PatchPackage, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer f.Close()

	if strings.HasSuffix(outputPath, ".gz") {
		gw, err := gzip.NewWriterLevel(f, gzip.BestCompression)
		if err != nil {
			return fmt.Errorf("failed to initialize gzip writer: %w", err)
		}
		defer gw.Close()

		enc := json.NewEncoder(gw)
		if err := enc.Encode(p); err != nil {
			return fmt.Errorf("failed to encode patch JSON to gzip: %w", err)
		}
		return gw.Flush()
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

// LoadPatch loads a PatchPackage from disk, transparently handling gzip decompression
func LoadPatch(patchPath string) (*PatchPackage, error) {
	data, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patch file %s: %w", patchPath, err)
	}

	var reader io.Reader = bytes.NewReader(data)

	// Check for gzip magic header (0x1f, 0x8b) or file extension
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var pkg PatchPackage
	if err := json.NewDecoder(reader).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("failed to parse patch JSON: %w", err)
	}

	return &pkg, nil
}

// ToModelEntries converts patch entries into model.TextEntry list
func (p *PatchPackage) ToModelEntries() []model.TextEntry {
	entries := make([]model.TextEntry, len(p.Entries))
	for i, pe := range p.Entries {
		entries[i] = model.TextEntry{
			ID:        fmt.Sprintf("%s:%s", pe.FilePath, pe.KeyPath),
			Source:    pe.Source,
			Target:    pe.Target,
			FilePath:  pe.FilePath,
			KeyPath:   pe.KeyPath,
			Status:    model.StatusTranslated,
			UpdatedAt: p.CreatedAt,
		}
	}
	return entries
}

// ApplyPatchToGame directly patches a game directory using the appropriate engine parser
func ApplyPatchToGame(ctx context.Context, patchPath, gameDir, outDir string) error {
	pkg, err := LoadPatch(patchPath)
	if err != nil {
		return err
	}

	var engineParser parser.EngineParser
	if pkg.Engine != "" {
		engineParser, _ = parser.GetParser(pkg.Engine)
	}

	if engineParser == nil {
		engineParser, err = parser.DetectEngine(gameDir)
		if err != nil {
			return fmt.Errorf("failed to detect engine for game directory: %w", err)
		}
	}

	entries := pkg.ToModelEntries()
	dest := outDir
	if dest == "" {
		dest = gameDir
	}

	return engineParser.Inject(ctx, gameDir, dest, entries)
}
