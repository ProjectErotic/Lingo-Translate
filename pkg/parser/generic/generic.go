package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lingo-translate/pkg/model"
)

// Parser handles GameMaker and generic JSON/i18n game localizations (e.g. Adulttale, indie games)
type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Name() string {
	return "generic"
}

// Common localization directory candidate names
var locDirNames = []string{
	"lang",
	"locales",
	"localization",
	"translations",
	"i18n",
}

// Detect checks if the directory has a standard localization folder with .json files,
// or contains GameMaker data.win with a lang folder.
func (p *Parser) Detect(dir string) bool {
	// 1. Check for GameMaker data.win with lang folder
	if _, err := os.Stat(filepath.Join(dir, "data.win")); err == nil {
		if fi, err := os.Stat(filepath.Join(dir, "lang")); err == nil && fi.IsDir() {
			return true
		}
	}

	// 2. Check for any of the standard localization folders with JSON files
	for _, sub := range locDirNames {
		subPath := filepath.Join(dir, sub)
		if fi, err := os.Stat(subPath); err == nil && fi.IsDir() {
			hasJSON := false
			_ = filepath.WalkDir(subPath, func(path string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ".json" {
					hasJSON = true
					return filepath.SkipAll
				}
				return nil
			})
			if hasJSON {
				return true
			}
		}
	}

	return false
}

// FindPrimarySourceJSON finds the best candidate source language JSON file (e.g. eng.json, en.json)
func (p *Parser) FindPrimarySourceJSON(dir string) (string, error) {
	for _, sub := range locDirNames {
		subPath := filepath.Join(dir, sub)
		if fi, err := os.Stat(subPath); err == nil && fi.IsDir() {
			// Look for English candidates first
			candidates := []string{"eng.json", "en.json", "en_us.json", "strings.json", "localization.json"}
			for _, c := range candidates {
				candidatePath := filepath.Join(subPath, c)
				if _, err := os.Stat(candidatePath); err == nil {
					return candidatePath, nil
				}
			}

			// Otherwise pick the first .json file found
			var firstJSON string
			_ = filepath.WalkDir(subPath, func(path string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ".json" {
					// Skip support.json or metadata files
					base := strings.ToLower(filepath.Base(path))
					if !strings.Contains(base, "support") && !strings.Contains(base, "schema") {
						firstJSON = path
						return filepath.SkipAll
					}
				}
				return nil
			})
			if firstJSON != "" {
				return firstJSON, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable localization JSON file found in %s", dir)
}

// Extract parses the source JSON file and converts all text strings into model.TextEntry items.
func (p *Parser) Extract(ctx context.Context, dir string) ([]model.TextEntry, *model.ExtractionStats, error) {
	startTime := time.Now()

	sourceFile, err := p.FindPrimarySourceJSON(dir)
	if err != nil {
		return nil, nil, err
	}

	dataBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read source file %s: %w", sourceFile, err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &rawMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON from %s: %w", sourceFile, err)
	}

	relFilePath, _ := filepath.Rel(dir, sourceFile)
	var entries []model.TextEntry
	totalWords := 0
	uniqueTexts := make(map[string]struct{})

	for k, v := range rawMap {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		switch val := v.(type) {
		case string:
			trimmed := strings.TrimSpace(val)
			if trimmed == "" {
				continue
			}
			uniqueTexts[trimmed] = struct{}{}
			totalWords += len(strings.Fields(trimmed))
			entries = append(entries, model.TextEntry{
				ID:        k,
				Source:    val,
				FilePath:  relFilePath,
				KeyPath:   k,
				Status:    model.StatusUntranslated,
				Context:   k,
				UpdatedAt: time.Now(),
			})

		case []interface{}:
			for idx, item := range val {
				if strItem, ok := item.(string); ok {
					trimmed := strings.TrimSpace(strItem)
					if trimmed == "" {
						continue
					}
					uniqueTexts[trimmed] = struct{}{}
					totalWords += len(strings.Fields(trimmed))
					entryID := fmt.Sprintf("%s[%d]", k, idx)
					entries = append(entries, model.TextEntry{
						ID:        entryID,
						Source:    strItem,
						FilePath:  relFilePath,
						KeyPath:   entryID,
						Status:    model.StatusUntranslated,
						Context:   k,
						UpdatedAt: time.Now(),
					})
				}
			}
		}
	}

	stats := &model.ExtractionStats{
		TotalEntries: len(entries),
		UniqueTexts:  len(uniqueTexts),
		TotalWords:   totalWords,
		FilesScanned: 1,
		Duration:     time.Since(startTime),
	}

	return entries, stats, nil
}

// Inject writes translated entries into tha.json (or th.json) in the target localization folder.
func (p *Parser) Inject(ctx context.Context, sourceDir, outputDir string, entries []model.TextEntry) error {
	sourceFile, err := p.FindPrimarySourceJSON(sourceDir)
	if err != nil {
		return err
	}

	dataBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &rawMap); err != nil {
		return fmt.Errorf("failed to parse source JSON: %w", err)
	}

	// Index translated entries by ID. Workspace rows use the bare key
	// ("lang_x", "lang_y[0]"), while distribution patches carry
	// "file:key" ("lang/eng.json:lang_x") — accept both so patches
	// inject identically to workspace-driven runs.
	translations := make(map[string]string)
	for _, entry := range entries {
		if entry.Target != "" {
			translations[entry.ID] = entry.Target
			if entry.KeyPath != "" && entry.ID != entry.KeyPath {
				translations[entry.KeyPath] = entry.Target
			}
		}
	}

	// Reconstruct translated JSON map
	outMap := make(map[string]interface{})
	for k, v := range rawMap {
		switch val := v.(type) {
		case string:
			if translated, ok := translations[k]; ok {
				outMap[k] = translated
			} else {
				outMap[k] = val
			}

		case []interface{}:
			var outList []interface{}
			for idx, item := range val {
				if strItem, ok := item.(string); ok {
					entryID := fmt.Sprintf("%s[%d]", k, idx)
					if translated, ok := translations[entryID]; ok {
						outList = append(outList, translated)
					} else {
						outList = append(outList, strItem)
					}
				} else {
					outList = append(outList, item)
				}
			}
			outMap[k] = outList

		default:
			outMap[k] = v
		}
	}

	// Determine output directory and file
	relPath, _ := filepath.Rel(sourceDir, filepath.Dir(sourceFile))
	destDir := filepath.Join(outputDir, relPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	destFile := filepath.Join(destDir, "tha.json")
	outBytes, err := json.MarshalIndent(outMap, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal translated JSON: %w", err)
	}

	if err := os.WriteFile(destFile, outBytes, 0644); err != nil {
		return fmt.Errorf("failed to write translated file %s: %w", destFile, err)
	}

	return nil
}
