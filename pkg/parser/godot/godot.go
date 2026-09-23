package godot

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lingo-translate/pkg/model"
)

var (
	// Matches text properties in Godot .tscn files
	tscnPropertyPattern = regexp.MustCompile(`(?m)^\s*(?:text|label|placeholder_text|tooltip_text|dialog_text|title)\s*=\s*"([^"]+)"`)
	// Matches tr("...") function calls in .gd scripts and .tscn
	trCallPattern = regexp.MustCompile(`tr\s*\(\s*"([^"]+)"\s*\)`)
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Name() string {
	return "godot"
}

func (p *Parser) Detect(dir string) bool {
	// Check for project.godot or *.pck in root
	if _, err := os.Stat(filepath.Join(dir, "project.godot")); err == nil {
		return true
	}

	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".pck" || ext == ".tscn" {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func (p *Parser) Extract(ctx context.Context, dir string) ([]model.TextEntry, *model.ExtractionStats, error) {
	startTime := time.Now()
	var entries []model.TextEntry
	filesScanned := 0
	uniqueSet := make(map[string]bool)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".godot" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		relPath, _ := filepath.Rel(dir, path)

		switch ext {
		case ".tscn", ".gd":
			filesScanned++
			fileEntries, err := p.extractTextFile(path, relPath)
			if err == nil {
				entries = append(entries, fileEntries...)
			}
		case ".csv":
			filesScanned++
			csvEntries, err := p.extractCSVFile(path, relPath)
			if err == nil {
				entries = append(entries, csvEntries...)
			}
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	totalWords := 0
	for _, e := range entries {
		uniqueSet[e.Source] = true
		totalWords += len(strings.Fields(e.Source))
	}

	stats := &model.ExtractionStats{
		TotalEntries: len(entries),
		UniqueTexts:  len(uniqueSet),
		TotalWords:   totalWords,
		FilesScanned: filesScanned,
		Duration:     time.Since(startTime),
	}

	return entries, stats, nil
}

func (p *Parser) extractTextFile(absPath, relPath string) ([]model.TextEntry, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []model.TextEntry
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}

		// Check property pattern (text = "...")
		if m := tscnPropertyPattern.FindStringSubmatch(line); len(m) > 1 {
			txt := m[1]
			if !isSystemCode(txt) {
				entries = append(entries, model.TextEntry{
					ID:        fmt.Sprintf("%s:%d:prop", relPath, lineNum),
					Source:    txt,
					FilePath:  relPath,
					KeyPath:   fmt.Sprintf("line:%d", lineNum),
					Status:    model.StatusUntranslated,
					Context:   "UI Property",
					UpdatedAt: time.Now(),
				})
			}
		}

		// Check tr("...") calls
		if m := trCallPattern.FindAllStringSubmatch(line, -1); len(m) > 0 {
			for idx, match := range m {
				txt := match[1]
				if !isSystemCode(txt) {
					entries = append(entries, model.TextEntry{
						ID:        fmt.Sprintf("%s:%d:tr:%d", relPath, lineNum, idx),
						Source:    txt,
						FilePath:  relPath,
						KeyPath:   fmt.Sprintf("line:%d:tr", lineNum),
						Status:    model.StatusUntranslated,
						Context:   "tr() localization",
						UpdatedAt: time.Now(),
					})
				}
			}
		}
	}

	return entries, scanner.Err()
}

func (p *Parser) extractCSVFile(absPath, relPath string) ([]model.TextEntry, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, err
	}

	header := records[0]
	if len(header) < 2 {
		return nil, nil // Not a Godot translation CSV
	}

	var entries []model.TextEntry
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]
		if len(row) < 2 {
			continue
		}
		key := row[0]
		source := row[1] // Primary source language

		target := ""
		if len(row) > 2 {
			target = row[2]
		}

		status := model.StatusUntranslated
		if target != "" {
			status = model.StatusTranslated
		}

		entries = append(entries, model.TextEntry{
			ID:        fmt.Sprintf("%s:%s", relPath, key),
			Source:    source,
			Target:    target,
			FilePath:  relPath,
			KeyPath:   fmt.Sprintf("key:%s", key),
			Status:    status,
			Context:   "CSV Record",
			UpdatedAt: time.Now(),
		})
	}

	return entries, nil
}

func isSystemCode(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return true
	}
	// Skip node paths e.g. "res://", "res:", "uid://"
	if strings.HasPrefix(trimmed, "res://") || strings.HasPrefix(trimmed, "uid://") {
		return true
	}
	return false
}

// Inject writes Godot localization translations into a CSV file
func (p *Parser) Inject(ctx context.Context, sourceDir, outputDir string, entries []model.TextEntry) error {
	targetBase := sourceDir
	if outputDir != "" && outputDir != sourceDir {
		targetBase = outputDir
	}

	// Group by file path
	entriesByFile := make(map[string][]model.TextEntry)
	for _, e := range entries {
		entriesByFile[e.FilePath] = append(entriesByFile[e.FilePath], e)
	}

	for relPath, fileEntries := range entriesByFile {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ext := strings.ToLower(filepath.Ext(relPath))
		destPath := filepath.Join(targetBase, relPath)
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)

		if ext == ".csv" {
			_ = p.patchCSV(destPath, fileEntries)
		} else if ext == ".tscn" {
			_ = p.patchTSCN(filepath.Join(sourceDir, relPath), destPath, fileEntries)
		}
	}

	return nil
}

func (p *Parser) patchCSV(destPath string, entries []model.TextEntry) error {
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	// Header: id,en,th
	_ = w.Write([]string{"id", "en", "th"})

	for _, e := range entries {
		key := strings.TrimPrefix(e.KeyPath, "key:")
		target := e.Target
		if target == "" {
			target = e.Source
		}
		_ = w.Write([]string{key, e.Source, target})
	}
	return nil
}

func (p *Parser) patchTSCN(srcPath, destPath string, entries []model.TextEntry) error {
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	content := string(srcBytes)
	for _, e := range entries {
		if e.Target != "" && e.Target != e.Source {
			oldSpaced := fmt.Sprintf(` = "%s"`, e.Source)
			newSpaced := fmt.Sprintf(` = "%s"`, e.Target)
			if strings.Contains(content, oldSpaced) {
				content = strings.Replace(content, oldSpaced, newSpaced, 1)
			} else {
				oldDirect := fmt.Sprintf(`="%s"`, e.Source)
				newDirect := fmt.Sprintf(`="%s"`, e.Target)
				content = strings.Replace(content, oldDirect, newDirect, 1)
			}
		}
	}

	return os.WriteFile(destPath, []byte(content), 0644)
}
