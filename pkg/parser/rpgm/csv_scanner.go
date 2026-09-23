package rpgm

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lingo-translate/pkg/model"
)

var (
	hangulRegex   = regexp.MustCompile(`[\x{ac00}-\x{d7a3}]`)
	cjkRegex      = regexp.MustCompile(`[\x{4e00}-\x{9faf}\x{3040}-\x{30ff}]`)
	cyrillicRegex = regexp.MustCompile(`[\x{0400}-\x{04ff}]`)
	tagRegex      = regexp.MustCompile(`^<@[^>]+>$`)
	triggerRegex  = regexp.MustCompile(`^sub_[a-zA-Z0-9_\x{ac00}-\x{d7a3}]+$`)
)

// ScanCustomDataFiles scans dataDir for CSV, TSV, RCSV, and custom data files
func (p *Parser) ScanCustomDataFiles(rootDir, dataDir string, keys []string, entries *[]model.TextEntry) int {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return 0
	}

	supportedExts := map[string]bool{
		".csv":   true,
		".tsv":   true,
		".rcsv":  true,
		".txt":   true,
		".dat":   true,
		".rdata": true,
	}

	scannedCount := 0
	seenTexts := make(map[string]bool)
	for _, e := range *entries {
		seenTexts[e.Source] = true
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if !supportedExts[ext] {
			continue
		}

		filePath := filepath.Join(dataDir, f.Name())
		rawBytes, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Try decrypting if encrypted
		decrypted, _, _ := TryDecrypt(rawBytes, keys)
		if len(decrypted) == 0 {
			continue
		}

		scannedCount++
		p.extractFromDelimitedText(decrypted, f.Name(), entries, seenTexts)
	}

	return scannedCount
}

func (p *Parser) extractFromDelimitedText(data []byte, fileName string, entries *[]model.TextEntry, seenTexts map[string]bool) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // variable columns allowed

	// Test if TSV
	if bytes.Contains(data, []byte("\t")) && !bytes.Contains(data, []byte(",")) {
		reader.Comma = '\t'
	}

	rowIdx := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed line and continue
			rowIdx++
			continue
		}

		rowIdx++
		// Skip header rows (typically row 1 or 2)
		if rowIdx <= 2 && (strings.HasPrefix(strings.ToLower(strings.Join(record, ",")), "desc") ||
			strings.HasPrefix(strings.ToLower(strings.Join(record, ",")), "key") ||
			strings.HasPrefix(strings.ToLower(strings.Join(record, ",")), "#")) {
			continue
		}

		for colIdx, val := range record {
			val = strings.TrimSpace(val)
			if !isTranslatableText(val) {
				continue
			}

			// Avoid duplicates if identical source is already in this file
			id := fmt.Sprintf("%s:%d:%d", fileName, rowIdx, colIdx)
			entry := model.TextEntry{
				ID:        id,
				Source:    val,
				FilePath:  fileName,
				KeyPath:   fmt.Sprintf("row[%d].col[%d]", rowIdx, colIdx),
				Status:    model.StatusUntranslated,
				UpdatedAt: time.Now(),
			}

			*entries = append(*entries, entry)
		}
	}
}

func isTranslatableText(text string) bool {
	if text == "" || text == "-" || text == "none" || text == "null" {
		return false
	}

	// Reject tags and triggers
	if tagRegex.MatchString(text) || triggerRegex.MatchString(text) {
		return false
	}

	// Reject file paths, URLs, code expressions
	if strings.Contains(text, "://") || fileExtRegex.MatchString(text) || strings.HasPrefix(text, "@") {
		return false
	}

	// Check if contains Asian or Cyrillic characters
	if hangulRegex.MatchString(text) || cjkRegex.MatchString(text) || cyrillicRegex.MatchString(text) {
		return true
	}

	// English / Latin: must have whitespace (sentence/phrase) and not be an identifier or single token
	words := strings.Fields(text)
	if len(words) >= 2 && !strings.ContainsAny(text, "{};$()") {
		return true
	}

	return false
}
