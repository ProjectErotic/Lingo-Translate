package rpgm

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lingo-translate/pkg/model"
)

var (
	// Matches string literals: "..." or '...'
	stringLiteralRegex = regexp.MustCompile(`["']([^"'\r\n\\]{1,250})["']`)
)

// HarvestPluginStrings scans JS plugins to extract hardcoded UI and dialogue strings
func (p *Parser) HarvestPluginStrings(rootDir string, entries *[]model.TextEntry) int {
	searchDirs := []string{
		filepath.Join(rootDir, "js", "plugins"),
		filepath.Join(rootDir, "www", "js", "plugins"),
	}

	seenInEntries := make(map[string]bool)
	for _, e := range *entries {
		seenInEntries[e.Source] = true
	}

	harvestedCount := 0

	for _, dir := range searchDirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".js") {
				continue
			}

			// Skip standard RPG Maker core engines and translation layer itself
			baseName := strings.ToLower(f.Name())
			if strings.HasPrefix(baseName, "rpg_") || strings.HasPrefix(baseName, "nst_") ||
				baseName == "main.js" || baseName == "libs" {
				continue
			}

			filePath := filepath.Join(dir, f.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			matches := stringLiteralRegex.FindAllStringSubmatch(string(content), -1)
			for idx, match := range matches {
				val := strings.TrimSpace(match[1])
				if !isPluginTranslatableText(val) || seenInEntries[val] {
					continue
				}

				seenInEntries[val] = true
				harvestedCount++

				h := sha1.New()
				h.Write([]byte(fmt.Sprintf("%s:%d:%s", f.Name(), idx, val)))
				id := fmt.Sprintf("plugin_%x", h.Sum(nil)[:8])

				entry := model.TextEntry{
					ID:        id,
					Source:    val,
					FilePath:  "PluginUI.json",
					KeyPath:   fmt.Sprintf("%s.literal[%d]", f.Name(), idx),
					Status:    model.StatusUntranslated,
					Context:   fmt.Sprintf("Plugin %s", f.Name()),
					UpdatedAt: time.Now(),
				}

				*entries = append(*entries, entry)
			}
		}
	}

	return harvestedCount
}

func isPluginTranslatableText(text string) bool {
	if text == "" || len(text) < 2 {
		return false
	}

	// Must not be a code keyword, hex color, or CSS/file path
	if strings.HasPrefix(text, "#") || strings.HasPrefix(text, ".") ||
		strings.Contains(text, "://") || fileExtRegex.MatchString(text) ||
		strings.Contains(text, ";") || strings.Contains(text, "()") ||
		strings.HasPrefix(text, "@") || triggerRegex.MatchString(text) ||
		tagRegex.MatchString(text) {
		return false
	}

	// Contains Asian / Cyrillic characters (e.g. Korean, Japanese, Chinese)
	if hangulRegex.MatchString(text) || cjkRegex.MatchString(text) || cyrillicRegex.MatchString(text) {
		return true
	}

	return false
}
