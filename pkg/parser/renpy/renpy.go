package renpy

import (
	"bufio"
	"context"
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
	// Matches speaker "dialogue text" or "dialogue text"
	dialogPattern = regexp.MustCompile(`^\s*(?:[a-zA-Z_]\w*\s+)?"([^"]+)"`)
	// Matches menu choices e.g. "Choice 1":
	menuPattern = regexp.MustCompile(`^\s*"([^"]+)"\s*:`)
	// Matches translatable gettext strings e.g. Character(_("Name"))
	transStringPattern = regexp.MustCompile(`_\(\s*["']([^"']+)["']\s*\)`)
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Name() string {
	return "renpy"
}

// FindGameDir locates the game/ directory inside a Ren'Py project
func (p *Parser) FindGameDir(rootDir string) string {
	candidate := filepath.Join(rootDir, "game")
	if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
		return candidate
	}
	return rootDir
}

func (p *Parser) Detect(dir string) bool {
	gameDir := p.FindGameDir(dir)
	found := false
	_ = filepath.WalkDir(gameDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".rpy" || ext == ".rpyc" {
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
	gameDir := p.FindGameDir(dir)

	var entries []model.TextEntry
	filesScanned := 0
	uniqueSet := make(map[string]bool)

	err := filepath.WalkDir(gameDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			// Skip tl/ (translation) directory when extracting source texts
			if d.Name() == "tl" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.ToLower(filepath.Ext(path)) != ".rpy" {
			return nil
		}

		filesScanned++
		relPath, _ := filepath.Rel(gameDir, path)
		fileEntries, err := p.extractFile(path, relPath)
		if err == nil {
			entries = append(entries, fileEntries...)
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

func (p *Parser) extractFile(absPath, relPath string) ([]model.TextEntry, error) {
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

		// Skip comments and empty lines
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}

		// Check for translatable gettext strings e.g. Character(_("Name"))
		if strMatches := transStringPattern.FindAllStringSubmatch(line, -1); len(strMatches) > 0 {
			for matchIdx, sm := range strMatches {
				sText := sm[1]
				if strings.TrimSpace(sText) != "" {
					key := fmt.Sprintf("str_%s_%d_%d", filepath.Base(relPath), lineNum, matchIdx)
					entries = append(entries, model.TextEntry{
						ID:        fmt.Sprintf("%s:%s", relPath, key),
						Source:    sText,
						FilePath:  relPath,
						KeyPath:   key,
						Status:    model.StatusUntranslated,
						Context:   "String",
						UpdatedAt: time.Now(),
					})
				}
			}
		}

		if strings.HasPrefix(trimmed, "$") {
			continue
		}

		var text string
		contextLabel := ""

		if m := menuPattern.FindStringSubmatch(line); len(m) > 1 {
			text = m[1]
			contextLabel = "Choice"
		} else if m := dialogPattern.FindStringSubmatch(line); len(m) > 1 {
			text = m[1]
			contextLabel = "Dialogue"
		}

		if text != "" && strings.TrimSpace(text) != "" {
			key := fmt.Sprintf("%s:%d", filepath.Base(relPath), lineNum)
			entries = append(entries, model.TextEntry{
				ID:        fmt.Sprintf("%s:%s", relPath, key),
				Source:    text,
				FilePath:  relPath,
				KeyPath:   key,
				Status:    model.StatusUntranslated,
				Context:   contextLabel,
				UpdatedAt: time.Now(),
			})
		}
	}

	return entries, scanner.Err()
}

func escapeRpy(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// Inject generates Ren'Py native translation files: game/tl/<Lang>/lingo_translations.rpy and game/lingo_language.rpy
func (p *Parser) Inject(ctx context.Context, sourceDir, outputDir string, entries []model.TextEntry) error {
	targetBase := sourceDir
	if outputDir != "" && outputDir != sourceDir {
		targetBase = outputDir
	}

	gameDir := p.FindGameDir(targetBase)
	lang := "Thai" // Default target language folder

	tlDir := filepath.Join(gameDir, "tl", lang)
	if err := os.MkdirAll(tlDir, 0755); err != nil {
		return fmt.Errorf("failed to create Ren'Py tl directory: %w", err)
	}

	// 1. Write game/lingo_language.rpy (auto-switches to translated language on startup)
	bootScript := fmt.Sprintf("# Generated by Lingo Translate\ninit -1700 python:\n    config.language = \"%s\"\n    config.default_language = \"%s\"\n", lang, lang)
	_ = os.WriteFile(filepath.Join(gameDir, "lingo_language.rpy"), []byte(bootScript), 0644)

	// 2. Write game/tl/<Lang>/lingo_translations.rpy
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Generated by Lingo Translate\ntranslate %s strings:\n\n", lang))

	written := 0
	for _, e := range entries {
		if e.Target != "" && e.Target != e.Source {
			sb.WriteString(fmt.Sprintf("    old \"%s\"\n", escapeRpy(e.Source)))
			sb.WriteString(fmt.Sprintf("    new \"%s\"\n\n", escapeRpy(e.Target)))
			written++
		}
	}

	if written == 0 {
		return nil
	}

	rpyPath := filepath.Join(tlDir, "lingo_translations.rpy")
	return os.WriteFile(rpyPath, []byte(sb.String()), 0644)
}
