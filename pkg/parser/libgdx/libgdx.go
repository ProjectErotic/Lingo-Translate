package libgdx

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"lingo-translate/pkg/model"
)

var (
	// Non-translatable checks
	pureNumberRegex = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
	fileExtRegex    = regexp.MustCompile(`(?i)\.(png|jpg|jpeg|gif|bmp|webp|ogg|mp3|wav|m4a|flac|json|dll|so|exe|meta|atlas|skel|atlas)$`)
	urlRegex        = regexp.MustCompile(`^(https?://|ftp://)`)
)

// Parser handles libGDX Java games and localization assets
type Parser struct{}

// New creates a new libGDX parser instance
func New() *Parser {
	return &Parser{}
}

// Name returns the engine identifier
func (p *Parser) Name() string {
	return "libgdx"
}

// Detect checks if the specified directory contains a libGDX game
func (p *Parser) Detect(dir string) bool {
	// 1. Direct check: libGDX native libraries in root or subdirectories
	nativeLibs := []string{"gdx64.dll", "gdx.dll", "libgdx64.so", "libgdx.so"}
	for _, lib := range nativeLibs {
		if _, err := os.Stat(filepath.Join(dir, lib)); err == nil {
			return true
		}
	}

	// 2. Scan for JAR files in dir and inspect for libGDX packages
	jars, err := p.FindGameJars(dir)
	if err == nil && len(jars) > 0 {
		for _, jarPath := range jars {
			if p.isLibGDXJar(jarPath) {
				return true
			}
		}
	}

	return false
}

// isLibGDXJar checks if a jar contains libGDX classes or libraries
func (p *Parser) isLibGDXJar(jarPath string) bool {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return false
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "com/badlogic/gdx") ||
			f.Name == "gdx64.dll" || f.Name == "gdx.dll" ||
			f.Name == "libgdx64.so" || f.Name == "libgdx.so" ||
			strings.HasSuffix(f.Name, "gdx.gwt.xml") {
			return true
		}
	}
	return false
}

// FindGameJars finds all .jar files in the directory (up to 2 levels deep)
func (p *Parser) FindGameJars(dir string) ([]string, error) {
	var jars []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			if strings.Count(rel, string(filepath.Separator)) >= 2 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".jar") {
			jars = append(jars, path)
		}
		return nil
	})
	return jars, err
}

// isTranslatable checks if a string is real text and not an ID, code, or asset filename
func isTranslatable(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 2 {
		return false
	}
	if pureNumberRegex.MatchString(trimmed) {
		return false
	}
	if fileExtRegex.MatchString(trimmed) {
		return false
	}
	if urlRegex.MatchString(trimmed) {
		return false
	}

	// Must contain at least one letter
	hasLetter := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	return hasLetter
}

// Extract extracts translatable texts from libGDX JAR files and external asset folders
func (p *Parser) Extract(ctx context.Context, dir string) ([]model.TextEntry, *model.ExtractionStats, error) {
	startTime := time.Now()
	var entries []model.TextEntry
	filesScanned := 0
	uniqueSet := make(map[string]bool)

	jars, err := p.FindGameJars(dir)
	if err != nil {
		return nil, nil, err
	}

	for _, jarPath := range jars {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		if !p.isLibGDXJar(jarPath) {
			continue
		}

		jarEntries, scanned, err := p.extractFromJar(ctx, jarPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extract from %s: %w", jarPath, err)
		}
		filesScanned += scanned

		for _, e := range jarEntries {
			uniqueSet[e.Source] = true
			entries = append(entries, e)
		}
	}

	totalWords := 0
	for text := range uniqueSet {
		totalWords += len(strings.Fields(text))
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

// isIgnoredJarEntry filters out third-party library properties and metadata
func isIgnoredJarEntry(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "meta-inf/") {
		return true
	}
	if strings.HasPrefix(lower, "org/apache/") ||
		strings.HasPrefix(lower, "org/eclipse/") ||
		strings.HasPrefix(lower, "org/antlr/") ||
		strings.HasPrefix(lower, "com/ibm/icu/") ||
		strings.HasPrefix(lower, "com/kotcrab/vis/ui/i18n/") ||
		strings.HasPrefix(lower, "regexodus/") {
		return true
	}
	return false
}

// extractFromJar reads translatable files (.properties and .json) from inside a JAR
func (p *Parser) extractFromJar(ctx context.Context, jarPath string) ([]model.TextEntry, int, error) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()

	var entries []model.TextEntry
	filesScanned := 0

	for _, f := range r.File {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		if f.FileInfo().IsDir() || isIgnoredJarEntry(f.Name) {
			continue
		}

		lower := strings.ToLower(f.Name)

		// 1. Java Properties files (*.properties)
		if strings.HasSuffix(lower, ".properties") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			filesScanned++
			propEntries := parseProperties(f.Name, data)
			entries = append(entries, propEntries...)
		}

		// 2. Script / Encounter JSON files (*.json)
		if strings.HasSuffix(lower, ".json") {
			// Skip UI skins, bar configurations, etc.
			if strings.Contains(lower, "bar.json") ||
				strings.Contains(lower, "skin.json") ||
				strings.Contains(lower, "battleui.json") {
				continue
			}

			// Focus on script, encounter, story, dialogue JSONs
			if strings.HasPrefix(lower, "script/") ||
				strings.HasPrefix(lower, "encounters/") ||
				strings.HasPrefix(lower, "dialogue/") ||
				strings.HasPrefix(lower, "story/") ||
				strings.Contains(lower, "encounter") {

				rc, err := f.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}
				filesScanned++
				jsonEntries := parseScriptJSON(f.Name, data)
				entries = append(entries, jsonEntries...)
			}
		}
	}

	return entries, filesScanned, nil
}

// parseProperties parses Java ResourceBundle .properties content
func parseProperties(filePath string, data []byte) []model.TextEntry {
	var entries []model.TextEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var currentKey, currentValue strings.Builder
	inMultiLine := false

	for scanner.Scan() {
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)

		// Check for continuation
		if inMultiLine {
			hasCont := strings.HasSuffix(rawLine, "\\")
			linePart := rawLine
			if hasCont {
				linePart = strings.TrimSuffix(linePart, "\\")
			}
			currentValue.WriteString(strings.TrimLeft(linePart, " \t"))
			if !hasCont {
				inMultiLine = false
				val := unescapeJavaString(currentValue.String())
				if isTranslatable(val) {
					key := currentKey.String()
					entries = append(entries, model.TextEntry{
						ID:        fmt.Sprintf("%s:%s", filePath, key),
						Source:    val,
						FilePath:  filePath,
						KeyPath:   key,
						Status:    model.StatusUntranslated,
						Context:   "Properties: " + key,
						UpdatedAt: time.Now(),
					})
				}
				currentKey.Reset()
				currentValue.Reset()
			}
			continue
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			continue
		}

		// Find separator (= or :)
		sepIdx := -1
		for i := 0; i < len(rawLine); i++ {
			c := rawLine[i]
			if (c == '=' || c == ':') && (i == 0 || rawLine[i-1] != '\\') {
				sepIdx = i
				break
			}
		}

		if sepIdx == -1 {
			continue
		}

		key := strings.TrimSpace(rawLine[:sepIdx])
		valPart := rawLine[sepIdx+1:]

		// Check if line continues
		if strings.HasSuffix(valPart, "\\") {
			inMultiLine = true
			currentKey.WriteString(key)
			currentValue.WriteString(strings.TrimSuffix(valPart, "\\"))
			continue
		}

		val := unescapeJavaString(strings.TrimSpace(valPart))
		if isTranslatable(val) {
			entries = append(entries, model.TextEntry{
				ID:        fmt.Sprintf("%s:%s", filePath, key),
				Source:    val,
				FilePath:  filePath,
				KeyPath:   key,
				Status:    model.StatusUntranslated,
				Context:   "Properties: " + key,
				UpdatedAt: time.Now(),
			})
		}
	}

	return entries
}

// unescapeJavaString unescapes standard Java property escape sequences
func unescapeJavaString(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	n := len(runes)

	for i := 0; i < n; i++ {
		if runes[i] == '\\' && i+1 < n {
			next := runes[i+1]
			switch next {
			case 't':
				sb.WriteRune('\t')
				i++
			case 'r':
				sb.WriteRune('\r')
				i++
			case 'n':
				sb.WriteRune('\n')
				i++
			case 'f':
				sb.WriteRune('\f')
				i++
			case '\\':
				sb.WriteRune('\\')
				i++
			case '=', ':', '!', '#':
				sb.WriteRune(next)
				i++
			case 'u':
				if i+5 < n {
					hex := string(runes[i+2 : i+6])
					if code, err := strconv.ParseInt(hex, 16, 32); err == nil {
						sb.WriteRune(rune(code))
						i += 5
						continue
					}
				}
				sb.WriteRune(next)
				i++
			default:
				sb.WriteRune(next)
				i++
			}
		} else {
			sb.WriteRune(runes[i])
		}
	}

	return sb.String()
}

// escapePropertiesValue formats string back for Java .properties
func escapePropertiesValue(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// parseScriptJSON parses game script JSONs (like ToA encounters.json, encounter-trees.json, etc.)
func parseScriptJSON(filePath string, data []byte) []model.TextEntry {
	var entries []model.TextEntry

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return entries
	}

	baseName := filepath.Base(filePath)

	switch baseName {
	case "encounters.json":
		// Format: map[string][]map[string]interface{}
		if rootMap, ok := raw.(map[string]interface{}); ok {
			for encID, sceneListRaw := range rootMap {
				sceneList, ok := sceneListRaw.([]interface{})
				if !ok {
					continue
				}
				for idx, itemRaw := range sceneList {
					item, ok := itemRaw.(map[string]interface{})
					if !ok {
						continue
					}

					// Extract dialogue "text"
					if textVal, ok := item["text"].(string); ok && isTranslatable(textVal) {
						keyPath := fmt.Sprintf("%s[%d].text", encID, idx)
						entries = append(entries, model.TextEntry{
							ID:        fmt.Sprintf("%s:%s", filePath, keyPath),
							Source:    textVal,
							FilePath:  filePath,
							KeyPath:   keyPath,
							Status:    model.StatusUntranslated,
							Context:   fmt.Sprintf("Encounter: %s", encID),
							UpdatedAt: time.Now(),
						})
					}

					// Extract companion "chatter"
					if chatterVal, ok := item["chatter"].(string); ok && isTranslatable(chatterVal) {
						keyPath := fmt.Sprintf("%s[%d].chatter", encID, idx)
						entries = append(entries, model.TextEntry{
							ID:        fmt.Sprintf("%s:%s", filePath, keyPath),
							Source:    chatterVal,
							FilePath:  filePath,
							KeyPath:   keyPath,
							Status:    model.StatusUntranslated,
							Context:   fmt.Sprintf("Chatter: %s", encID),
							UpdatedAt: time.Now(),
						})
					}
				}
			}
		}

	case "encounter-trees.json":
		// Format: map[string]struct { nodes: map[string]struct { prompt: string } }
		if rootMap, ok := raw.(map[string]interface{}); ok {
			for treeID, treeRaw := range rootMap {
				treeObj, ok := treeRaw.(map[string]interface{})
				if !ok {
					continue
				}
				nodesObj, ok := treeObj["nodes"].(map[string]interface{})
				if !ok {
					continue
				}
				for nodeID, nodeRaw := range nodesObj {
					node, ok := nodeRaw.(map[string]interface{})
					if !ok {
						continue
					}
					if promptVal, ok := node["prompt"].(string); ok && isTranslatable(promptVal) {
						keyPath := fmt.Sprintf("%s.nodes.%s.prompt", treeID, nodeID)
						entries = append(entries, model.TextEntry{
							ID:        fmt.Sprintf("%s:%s", filePath, keyPath),
							Source:    promptVal,
							FilePath:  filePath,
							KeyPath:   keyPath,
							Status:    model.StatusUntranslated,
							Context:   fmt.Sprintf("Choice: %s", treeID),
							UpdatedAt: time.Now(),
						})
					}
				}
			}
		}

	default:
		// Generic recursive JSON extractor
		extractGenericJSON(filePath, "", raw, &entries)
	}

	return entries
}

// extractGenericJSON recursively traverses JSON structures looking for translatable string values
func extractGenericJSON(filePath, currentPath string, val interface{}, entries *[]model.TextEntry) {
	switch v := val.(type) {
	case map[string]interface{}:
		for k, child := range v {
			childPath := k
			if currentPath != "" {
				childPath = currentPath + "." + k
			}

			// Check if key itself indicates translatable content
			lowerKey := strings.ToLower(k)
			if str, ok := child.(string); ok {
				if (lowerKey == "text" || lowerKey == "prompt" || lowerKey == "dialogue" ||
					lowerKey == "dialog" || lowerKey == "message" || lowerKey == "description" ||
					lowerKey == "caption" || lowerKey == "title") && isTranslatable(str) {
					*entries = append(*entries, model.TextEntry{
						ID:        fmt.Sprintf("%s:%s", filePath, childPath),
						Source:    str,
						FilePath:  filePath,
						KeyPath:   childPath,
						Status:    model.StatusUntranslated,
						Context:   "JSON: " + k,
						UpdatedAt: time.Now(),
					})
				}
			} else {
				extractGenericJSON(filePath, childPath, child, entries)
			}
		}
	case []interface{}:
		for idx, item := range v {
			childPath := fmt.Sprintf("%s[%d]", currentPath, idx)
			extractGenericJSON(filePath, childPath, item, entries)
		}
	}
}

// Inject writes translated entries into the target directory.
// For games supporting native translation folders (e.g. Tales of Androgyny), it generates
// translations/<Lang>/manifest.json and localized assets.
// It also supports direct JAR patching when target output directory is specified.
func (p *Parser) Inject(ctx context.Context, sourceDir, outputDir string, entries []model.TextEntry) error {
	targetBase := sourceDir
	if outputDir != "" && outputDir != sourceDir {
		targetBase = outputDir
	}

	// Filter translated entries
	var translated []model.TextEntry
	entriesByFile := make(map[string][]model.TextEntry)
	for _, e := range entries {
		if e.Target != "" && e.Target != e.Source {
			translated = append(translated, e)
			entriesByFile[e.FilePath] = append(entriesByFile[e.FilePath], e)
		}
	}

	if len(translated) == 0 {
		return nil
	}

	// Find game jar
	jars, _ := p.FindGameJars(sourceDir)
	var mainJarPath string
	for _, j := range jars {
		if p.isLibGDXJar(j) {
			mainJarPath = j
			break
		}
	}

	// Determine if game has ToA-style translation support
	isToA := false
	for filePath := range entriesByFile {
		if strings.HasPrefix(filePath, "translation/") || strings.HasPrefix(filePath, "script/") {
			isToA = true
			break
		}
	}

	langFolder := "thai"
	langDisplay := "Thai"
	localeID := "th"

	// 1. If ToA style, write native translation mod folder (clean & non-destructive!)
	if isToA {
		transRoot := filepath.Join(targetBase, "translations", langFolder)
		manifestPath := filepath.Join(transRoot, "manifest.json")
		assetsRoot := filepath.Join(transRoot, "assets")

		if err := os.MkdirAll(assetsRoot, 0755); err != nil {
			return fmt.Errorf("failed to create translation mod directory: %w", err)
		}

		// Write manifest.json
		manifestContent := fmt.Sprintf("{\n  \"languageName\": \"%s\",\n  \"localeId\": \"%s\"\n}\n", langDisplay, localeID)
		if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
			return fmt.Errorf("failed to write translation manifest: %w", err)
		}

		// Write translated asset files
		for filePath, fileEntries := range entriesByFile {
			outPath := filepath.Join(assetsRoot, filepath.FromSlash(filePath))
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}

			// Generate translated file content
			content, err := p.renderTranslatedFile(mainJarPath, filePath, fileEntries)
			if err != nil {
				return fmt.Errorf("failed to render translated file %s: %w", filePath, err)
			}

			if err := os.WriteFile(outPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outPath, err)
			}
		}
	}

	// 2. If outputDir is different from sourceDir, also copy game files and patch JAR directly
	if outputDir != "" && outputDir != sourceDir && mainJarPath != "" {
		if err := p.patchGameJar(sourceDir, outputDir, mainJarPath, entriesByFile); err != nil {
			return fmt.Errorf("failed to patch game JAR: %w", err)
		}
	}

	return nil
}

// renderTranslatedFile creates the patched file content for properties or JSON
func (p *Parser) renderTranslatedFile(jarPath, filePath string, entries []model.TextEntry) ([]byte, error) {
	// Read original bytes from JAR if available
	var origData []byte
	if jarPath != "" {
		if r, err := zip.OpenReader(jarPath); err == nil {
			defer r.Close()
			for _, f := range r.File {
				if f.Name == filePath {
					if rc, err := f.Open(); err == nil {
						origData, _ = io.ReadAll(rc)
						rc.Close()
					}
					break
				}
			}
		}
	}

	lower := strings.ToLower(filePath)
	if strings.HasSuffix(lower, ".properties") {
		return renderProperties(origData, entries), nil
	} else if strings.HasSuffix(lower, ".json") {
		return renderJSON(origData, entries)
	}

	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// renderProperties generates the localized .properties file
func renderProperties(origData []byte, entries []model.TextEntry) []byte {
	transMap := make(map[string]string)
	for _, e := range entries {
		transMap[e.KeyPath] = e.Target
	}

	var sb strings.Builder
	sb.WriteString("# Generated by NST-V2\n")

	if len(origData) > 0 {
		scanner := bufio.NewScanner(bytes.NewReader(origData))
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
				sb.WriteString(line + "\n")
				continue
			}

			sepIdx := strings.IndexAny(line, "=:")
			if sepIdx != -1 {
				key := strings.TrimSpace(line[:sepIdx])
				if tgt, ok := transMap[key]; ok {
					sb.WriteString(fmt.Sprintf("%s=%s\n", key, escapePropertiesValue(tgt)))
					delete(transMap, key)
					continue
				}
			}
			sb.WriteString(line + "\n")
		}
	}

	// Append any remaining keys
	for k, v := range transMap {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, escapePropertiesValue(v)))
	}

	return []byte(sb.String())
}

// renderJSON updates JSON data with translated entries
func renderJSON(origData []byte, entries []model.TextEntry) ([]byte, error) {
	var root interface{}
	if err := json.Unmarshal(origData, &root); err != nil {
		return nil, err
	}

	for _, e := range entries {
		setJSONValue(root, e.KeyPath, e.Target)
	}

	return json.MarshalIndent(root, "", "  ")
}

// setJSONValue traverses and sets a value in arbitrary JSON structures by KeyPath
func setJSONValue(root interface{}, path, value string) {
	tokens := tokenizePath(path)
	if len(tokens) == 0 {
		return
	}

	curr := root
	for i := 0; i < len(tokens)-1; i++ {
		tok := tokens[i]
		if idx, err := strconv.Atoi(tok); err == nil {
			if arr, ok := curr.([]interface{}); ok && idx >= 0 && idx < len(arr) {
				curr = arr[idx]
			} else {
				return
			}
		} else {
			if m, ok := curr.(map[string]interface{}); ok {
				if next, found := m[tok]; found {
					curr = next
				} else {
					return
				}
			} else {
				return
			}
		}
	}

	lastTok := tokens[len(tokens)-1]
	if idx, err := strconv.Atoi(lastTok); err == nil {
		if arr, ok := curr.([]interface{}); ok && idx >= 0 && idx < len(arr) {
			arr[idx] = value
		}
	} else {
		if m, ok := curr.(map[string]interface{}); ok {
			m[lastTok] = value
		}
	}
}

// tokenizePath splits path like "TROJA-INTRO[0].text" into ["TROJA-INTRO", "0", "text"]
func tokenizePath(path string) []string {
	var tokens []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '.' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if c == '[' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if c == ']' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// patchGameJar creates an updated JAR archive in outputDir with translated files injected
func (p *Parser) patchGameJar(sourceDir, outputDir, srcJarPath string, entriesByFile map[string][]model.TextEntry) error {
	relJar, err := filepath.Rel(sourceDir, srcJarPath)
	if err != nil {
		relJar = filepath.Base(srcJarPath)
	}
	destJarPath := filepath.Join(outputDir, relJar)

	if err := os.MkdirAll(filepath.Dir(destJarPath), 0755); err != nil {
		return err
	}

	srcReader, err := zip.OpenReader(srcJarPath)
	if err != nil {
		return err
	}
	defer srcReader.Close()

	destFile, err := os.Create(destJarPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	destWriter := zip.NewWriter(destFile)
	defer destWriter.Close()

	for _, f := range srcReader.File {
		if fileEntries, shouldPatch := entriesByFile[f.Name]; shouldPatch {
			// Write patched content
			content, err := p.renderTranslatedFile(srcJarPath, f.Name, fileEntries)
			if err != nil {
				return err
			}
			header := &zip.FileHeader{
				Name:     f.Name,
				Method:   f.Method,
				Modified: time.Now(),
			}
			w, err := destWriter.CreateHeader(header)
			if err != nil {
				return err
			}
			if _, err := w.Write(content); err != nil {
				return err
			}
		} else {
			// Copy original entry verbatim
			w, err := destWriter.CreateHeader(&f.FileHeader)
			if err != nil {
				return err
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			_, err = io.Copy(w, rc)
			rc.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
