package renpy

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"lingo-translate/pkg/model"
)

//go:embed assets/IBMPlexSansThai-Light.otf
var fontLightBytes []byte

//go:embed assets/IBMPlexSansThai-Medium.otf
var fontMediumBytes []byte

type DeployOptions struct {
	LanguageName   string // e.g. "Thai"
	DisplayName    string // e.g. "ไทย (Thai)"
	FontFileName   string // e.g. "IBMPlexSansThai-Light.otf"
	FontBoldName   string // e.g. "IBMPlexSansThai-Medium.otf"
	OnlyTranslated bool
}

// Exporter manages non-destructive additive deployment of translations to Ren'Py games
type Exporter struct{}

func NewExporter() *Exporter {
	return &Exporter{}
}

// Deploy creates game/tl/<LanguageName>/ and exports clean translation scripts, fonts, and UI bindings
func (e *Exporter) Deploy(gamePath string, entries []model.TextEntry, opts DeployOptions) error {
	if opts.LanguageName == "" {
		opts.LanguageName = "Thai"
	}
	if opts.DisplayName == "" {
		opts.DisplayName = "ไทย (Thai)"
	}
	if opts.FontFileName == "" {
		opts.FontFileName = "IBMPlexSansThai-Light.otf"
	}
	if opts.FontBoldName == "" {
		opts.FontBoldName = "IBMPlexSansThai-Medium.otf"
	}

	gameDir := filepath.Join(gamePath, "game")
	if fi, err := os.Stat(gameDir); err != nil || !fi.IsDir() {
		gameDir = gamePath
	}

	// 1. Install Zero-Tofu Fonts into game/fonts/
	fontsDir := filepath.Join(gameDir, "fonts")
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		return fmt.Errorf("failed to create fonts directory %s: %w", fontsDir, err)
	}

	fontLightPath := filepath.Join(fontsDir, opts.FontFileName)
	if _, err := os.Stat(fontLightPath); os.IsNotExist(err) && len(fontLightBytes) > 0 {
		_ = os.WriteFile(fontLightPath, fontLightBytes, 0644)
	}

	fontBoldPath := filepath.Join(fontsDir, opts.FontBoldName)
	if _, err := os.Stat(fontBoldPath); os.IsNotExist(err) && len(fontMediumBytes) > 0 {
		_ = os.WriteFile(fontBoldPath, fontMediumBytes, 0644)
	}

	tlDir := filepath.Join(gameDir, "tl", opts.LanguageName)
	if err := os.MkdirAll(tlDir, 0755); err != nil {
		return fmt.Errorf("failed to create tl directory %s: %w", tlDir, err)
	}

	// 2. Generate 00_lingo_font_layer.rpy (Scoping Rule: MUST use translate <lang> python:)
	fontScript := fmt.Sprintf(`# ==============================================================================
# Lingo-Translate Additive Multi-Language Typography Layer
# ==============================================================================

translate %s style centered_text:
    font "fonts/%s"

translate %s python:
    gui.text_font = "fonts/%s"
    gui.name_text_font = "fonts/%s"
    gui.interface_text_font = "fonts/%s"
    gui.button_text_font = "fonts/%s"
    gui.choice_button_text_font = "fonts/%s"
    style.centered_text.font = "fonts/%s"
`, opts.LanguageName, opts.FontBoldName,
		opts.LanguageName, opts.FontFileName, opts.FontBoldName, opts.FontFileName, opts.FontFileName, opts.FontFileName, opts.FontBoldName)

	fontScriptPath := filepath.Join(tlDir, "00_lingo_font_layer.rpy")
	if err := os.WriteFile(fontScriptPath, []byte(fontScript), 0644); err != nil {
		return fmt.Errorf("failed to write font layer script: %w", err)
	}

	// Build lookup maps for dialogue and strings
	dialogueMap := make(map[string]model.TextEntry) // key: KeyPath (block ID)
	dialogueBySource := make(map[string]model.TextEntry)
	stringsMap := make(map[string]string)           // key: Source

	for _, entry := range entries {
		trans := entry.Target
		if trans == "" {
			if opts.OnlyTranslated {
				continue
			}
			trans = entry.Source
		}

		if strings.HasPrefix(entry.KeyPath, "str_") || strings.HasPrefix(entry.KeyPath, "string:") || entry.Context == "Choice" {
			stringsMap[entry.Source] = trans
		} else {
			dialogueMap[entry.KeyPath] = entry
			dialogueMap[entry.ID] = entry
			dialogueBySource[entry.Source] = entry
		}
	}

	// 3. Export translated dialogue blocks to script.rpy (Template substitution if file exists)
	scriptPath := filepath.Join(tlDir, "script.rpy")
	existingScriptBytes, err := os.ReadFile(scriptPath)
	if err == nil && len(existingScriptBytes) > 0 {
		// Non-destructive template substitution
		scriptContent := string(existingScriptBytes)
		blockRegex := regexp.MustCompile(`(?m)^(translate\s+` + regexp.QuoteMeta(opts.LanguageName) + `\s+([a-zA-Z0-9_]+):\s*\n\s*#\s*(?:([a-zA-Z_]\w*|centered)\s+)?\"([^\"]*)\"([^\n]*)\n\s*)(?:(?:[a-zA-Z_]\w*|centered)\s+)?\"[^\"]*\"([^\n]*)`)
		updatedScript := blockRegex.ReplaceAllStringFunc(scriptContent, func(match string) string {
			sub := blockRegex.FindStringSubmatch(match)
			if len(sub) < 7 {
				return match
			}
			header := sub[1]
			blockID := sub[2]
			orig := sub[4]
			withClause := sub[5]

			entry, ok := dialogueMap[blockID]
			if !ok {
				entry, ok = dialogueBySource[orig]
			}
			if !ok {
				return match
			}
			trans := entry.Target
			if trans == "" {
				if opts.OnlyTranslated {
					return match
				}
				trans = orig
			}
			transEscaped := strings.ReplaceAll(trans, `"`, `\"`)

			spk := sub[3]
			var spkPrefix string
			if spk != "" {
				spkPrefix = spk + " "
			}

			return fmt.Sprintf("%s%s\"%s\"%s", header, spkPrefix, transEscaped, withClause)
		})

		// Dynamically translate any strings: block (such as Character names) in script.rpy template
		stringsBlockRegex := regexp.MustCompile(`(?m)^\s*old\s+"([^"]+)"\s*\n\s*new\s+"[^"]*"`)
		updatedScript = stringsBlockRegex.ReplaceAllStringFunc(updatedScript, func(m string) string {
			sub := stringsBlockRegex.FindStringSubmatch(m)
			if len(sub) < 2 {
				return m
			}
			src := sub[1]
			if trans, ok := stringsMap[src]; ok && trans != "" {
				return fmt.Sprintf("    old \"%s\"\n    new \"%s\"", strings.ReplaceAll(src, `"`, `\"`), strings.ReplaceAll(trans, `"`, `\"`))
			}
			if entry, ok := dialogueBySource[src]; ok && entry.Target != "" {
				return fmt.Sprintf("    old \"%s\"\n    new \"%s\"", strings.ReplaceAll(src, `"`, `\"`), strings.ReplaceAll(entry.Target, `"`, `\"`))
			}
			return m
		})

		_ = os.WriteFile(scriptPath, []byte(updatedScript), 0644)
	} else {
		// Fresh generation from entries
		var dialogueSb strings.Builder
		dialogueSb.WriteString(fmt.Sprintf("# Lingo-Translate Generated Dialogue Translation: %s\n\n", opts.LanguageName))

		for _, entry := range entries {
			if strings.HasPrefix(entry.KeyPath, "str_") || strings.HasPrefix(entry.KeyPath, "string:") || entry.Context == "Choice" {
				continue
			}
			trans := entry.Target
			if trans == "" {
				if opts.OnlyTranslated {
					continue
				}
				trans = entry.Source
			}

			labelID := entry.KeyPath
			if labelID == "" {
				labelID = fmt.Sprintf("line_%s", entry.ID)
			}
			labelID = strings.ReplaceAll(labelID, ":", "_")

			origEscaped := strings.ReplaceAll(entry.Source, "\"", "\\\"")
			transEscaped := strings.ReplaceAll(trans, "\"", "\\\"")

			var spkPrefix string
			if entry.Context != "" && entry.Context != "Dialogue" {
				cleanSpk := strings.TrimPrefix(entry.Context, "Speaker:")
				cleanSpk = strings.TrimSpace(cleanSpk)
				if cleanSpk != "" {
					spkPrefix = cleanSpk + " "
				}
			}

			dialogueSb.WriteString(fmt.Sprintf("translate %s %s:\n\n", opts.LanguageName, labelID))
			dialogueSb.WriteString(fmt.Sprintf("    # %s\"%s\"\n", spkPrefix, origEscaped))
			dialogueSb.WriteString(fmt.Sprintf("    %s\"%s\"\n\n", spkPrefix, transEscaped))
		}

		if err := os.WriteFile(scriptPath, []byte(dialogueSb.String()), 0644); err != nil {
			return fmt.Errorf("failed to write script.rpy: %w", err)
		}
	}

	// 4. Export UI strings to screens.rpy (Non-destructive update if file exists)
	screensPath := filepath.Join(tlDir, "screens.rpy")
	if existingBytes, err := os.ReadFile(screensPath); err == nil && len(existingBytes) > 0 {
		existingContent := string(existingBytes)
		screenBlockRegex := regexp.MustCompile(`(?m)^\s*old\s+"([^"]+)"\s*\n\s*new\s+"[^"]*"`)
		updatedScreens := screenBlockRegex.ReplaceAllStringFunc(existingContent, func(m string) string {
			sub := screenBlockRegex.FindStringSubmatch(m)
			if len(sub) < 2 {
				return m
			}
			src := sub[1]
			if trans, ok := stringsMap[src]; ok && trans != "" {
				return fmt.Sprintf("    old \"%s\"\n    new \"%s\"", strings.ReplaceAll(src, `"`, `\"`), strings.ReplaceAll(trans, `"`, `\"`))
			}
			return m
		})
		if err := os.WriteFile(screensPath, []byte(updatedScreens), 0644); err != nil {
			return fmt.Errorf("failed to write screens.rpy: %w", err)
		}
	} else {
		var stringsSb strings.Builder
		stringsSb.WriteString(fmt.Sprintf("translate %s strings:\n\n", opts.LanguageName))
		for src, trans := range stringsMap {
			origEscaped := strings.ReplaceAll(src, "\"", "\\\"")
			transEscaped := strings.ReplaceAll(trans, "\"", "\\\"")
			stringsSb.WriteString(fmt.Sprintf("    old \"%s\"\n    new \"%s\"\n\n", origEscaped, transEscaped))
		}
		if err := os.WriteFile(screensPath, []byte(stringsSb.String()), 0644); err != nil {
			return fmt.Errorf("failed to write screens.rpy: %w", err)
		}
	}

	// 5. Patch screens.rpy preferences to add clean language switcher and normalize spacing
	screensRpyPath := filepath.Join(gameDir, "screens.rpy")
	if data, err := os.ReadFile(screensRpyPath); err == nil {
		content := string(data)
		btnTag := fmt.Sprintf("action Language(\"%s\")", opts.LanguageName)
		if !strings.Contains(content, btnTag) {
			// Add clean language button after Language("JP") or similar
			re := regexp.MustCompile(`(textbutton\s+"[^"]+"\s+action\s+Language\([^)]+\))`)
			locs := re.FindAllStringIndex(content, -1)
			if len(locs) > 0 {
				lastIdx := locs[len(locs)-1][1]
				indentMatch := regexp.MustCompile(`(?m)^([ \t]+)textbutton`).FindStringSubmatch(content[locs[len(locs)-1][0]:])
				indent := "                    "
				if len(indentMatch) > 1 {
					indent = indentMatch[1]
				}
				// Use font tag inside textbutton to guarantee zero-tofu rendering across all locales
				buttonText := fmt.Sprintf("{font=fonts/%s}%s{/font}", opts.FontFileName, opts.DisplayName)
				newBtn := fmt.Sprintf("\n%stextbutton \"%s\" action Language(\"%s\")", indent, buttonText, opts.LanguageName)
				content = content[:lastIdx] + newBtn + content[lastIdx:]
			}
		}

		// Normalize vertical spacing (null height 4 * spacing -> 1 * spacing) to prevent bottom collision
		content = strings.ReplaceAll(content, "null height (4 * gui.pref_spacing)", "null height (1 * gui.pref_spacing)")

		_ = os.WriteFile(screensRpyPath, []byte(content), 0644)
	}

	return nil
}
