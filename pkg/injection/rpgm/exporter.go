package rpgm

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lingo-translate/pkg/model"
)

//go:embed assets/Lingo_TranslationLayer.js
var pluginScriptContent []byte

type DeployOptions struct {
	LanguageName   string
	SourceLocale   string
	OnlyTranslated bool
	Format         string // "json" or "txt"
}

// Exporter manages non-destructive deployment of translations to RPG Maker MV/MZ
type Exporter struct{}

func NewExporter() *Exporter {
	return &Exporter{}
}

// Deploy creates the lingo_translations directory, exports dictionary files, copies the runtime plugin, and patches plugins.js
func (e *Exporter) Deploy(gamePath string, entries []model.TextEntry, opts DeployOptions) error {
	if opts.LanguageName == "" {
		opts.LanguageName = "Thai"
	}
	if opts.SourceLocale == "" {
		opts.SourceLocale = "ja"
	}

	// 1. Locate translation directories (support both root and www/ for NW.js)
	transDirs := []string{filepath.Join(gamePath, "lingo_translations")}
	if fi, err := os.Stat(filepath.Join(gamePath, "www")); err == nil && fi.IsDir() {
		transDirs = append(transDirs, filepath.Join(gamePath, "www", "lingo_translations"))
	}

	// Group entries by base filename (e.g. "Map001.json" -> "Map001")
	fileMap := make(map[string][]model.TextEntry)
	for _, entry := range entries {
		base := strings.TrimSuffix(filepath.Base(entry.FilePath), filepath.Ext(entry.FilePath))
		if base == "" {
			continue
		}
		fileMap[base] = append(fileMap[base], entry)
	}

	for _, tDir := range transDirs {
		if err := os.MkdirAll(tDir, 0755); err != nil {
			return fmt.Errorf("failed to create translation directory %s: %w", tDir, err)
		}
		if err := e.writeConfigFile(tDir, opts); err != nil {
			return fmt.Errorf("failed to write config.json in %s: %w", tDir, err)
		}
		for baseName, fileEntries := range fileMap {
			if err := e.writeTranslationFiles(tDir, baseName, fileEntries, opts); err != nil {
				return fmt.Errorf("failed to write translations for %s in %s: %w", baseName, tDir, err)
			}
		}
	}

	// 5. Install Lingo_TranslationLayer.js into plugins directory
	pluginDir, err := e.findPluginDir(gamePath)
	if err != nil {
		return fmt.Errorf("failed to locate plugin directory: %w", err)
	}

	pluginDest := filepath.Join(pluginDir, "Lingo_TranslationLayer.js")
	if err := os.WriteFile(pluginDest, pluginScriptContent, 0644); err != nil {
		return fmt.Errorf("failed to write Lingo_TranslationLayer.js: %w", err)
	}

	// 6. Patch plugins.js to auto-register Lingo_TranslationLayer
	_ = e.patchPluginsJs(gamePath)

	return nil
}

func (e *Exporter) findPluginDir(gamePath string) (string, error) {
	candidates := []string{
		filepath.Join(gamePath, "www", "js", "plugins"),
		filepath.Join(gamePath, "js", "plugins"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c, nil
		}
	}

	// Default to creating js/plugins
	fallback := filepath.Join(gamePath, "js", "plugins")
	if err := os.MkdirAll(fallback, 0755); err != nil {
		return "", err
	}
	return fallback, nil
}

func (e *Exporter) writeConfigFile(transDir string, opts DeployOptions) error {
	config := map[string]interface{}{
		"__sourceLocale__": opts.SourceLocale,
		"__languageName__": opts.LanguageName,
		"__textFields__": []string{
			"name", "description", "displayName", "nickname", "profile",
			"message1", "message2", "message3", "message4", "gameTitle",
			"terms", "messages",
		},
		"__textCommands__": map[string][]int{
			"101": {4},
			"102": {0},
			"320": {1},
			"324": {1},
			"325": {1},
			"402": {1},
			"405": {0},
		},
		"__controlCharPatterns__": []string{
			`\\[VNP]\[\d+\]`, `\\I\[\d+\]`, `\\C\[\d+\]`, `\\G`,
			`\\[{}]`, `\\\$`, `\\[.|]`, `\\!`, `\\[><]`,
			`\\\\\^`, `\\\\\\\\`, `\\FS\[\d+\]`, `\\P[XY]\[-?\d+\]`,
			`\\[OT]C\[\d+\]`, `\\(?:MSGCORE|MSGSND)\[[^\]]*\]`,
		},
		"__ignorePatterns__": []string{
			`^.$`, `^\s*$`, `^[\d\s.,\-+%$!/\\:;()[\]{}=*#@!?<>~` + "`" + `'\"^&|_]+$`,
			`^\d+$`, `\.(png|jpe?g|gif|bmp|webp|svg|ico)$`, `\.(ogg|m4a|mp3|wav|flac|aac|wma)$`,
			`^https?://`, `^data:`,
		},
		"__fontConfig__": map[string]interface{}{
			"fontName":      "NotoSans",
			"fontUrl":       "fonts/NotoSans-Regular.woff2",
			"offsetSize":    0,
			"maxSizeOffset": 0,
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(transDir, "config.json"), data, 0644)
}

func (e *Exporter) writeTranslationFiles(transDir, baseName string, entries []model.TextEntry, opts DeployOptions) error {
	jsonDict := make(map[string]string)
	var txtBuilder strings.Builder

	for _, entry := range entries {
		target := entry.Target
		if target == "" {
			if opts.OnlyTranslated {
				continue
			}
			target = entry.Source
		}

		jsonDict[entry.Source] = target

		txtBuilder.WriteString("<<<ORIGINAL>>>\n")
		txtBuilder.WriteString(entry.Source)
		txtBuilder.WriteString("\n<<<TRANSLATED>>>\n")
		txtBuilder.WriteString(target)
		txtBuilder.WriteString("\n<<<END>>>\n\n")
	}

	if len(jsonDict) == 0 {
		return nil
	}

	// 1. Write JSON format (for fast runtime lookup)
	jsonData, err := json.MarshalIndent(jsonDict, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(transDir, baseName+".json"), jsonData, 0644); err != nil {
		return err
	}

	// 2. Write TXT format (for easy manual editing & backup)
	return os.WriteFile(filepath.Join(transDir, baseName+".txt"), []byte(txtBuilder.String()), 0644)
}

func (e *Exporter) patchPluginsJs(gamePath string) error {
	candidates := []string{
		filepath.Join(gamePath, "www", "js", "plugins.js"),
		filepath.Join(gamePath, "js", "plugins.js"),
	}

	var targetPath string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			targetPath = c
			break
		}
	}

	if targetPath == "" {
		return nil // No plugins.js to patch
	}

	contentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	content := string(contentBytes)
	if strings.Contains(content, "Lingo_TranslationLayer") {
		return nil // Already patched
	}

	entry := `{"name":"Lingo_TranslationLayer","status":true,"description":"Lingo Translation Layer (Auto-Injected)","parameters":{}}`

	closeBracket := strings.LastIndex(content, "];")
	if closeBracket < 0 {
		return nil
	}

	before := strings.TrimSpace(content[:closeBracket])
	needComma := len(before) > 0 && !strings.HasSuffix(before, "[")

	insertion := "\n"
	if needComma {
		insertion = ",\n"
	}
	insertion += entry + "\n"

	patched := content[:closeBracket] + insertion + content[closeBracket:]
	return os.WriteFile(targetPath, []byte(patched), 0644)
}
