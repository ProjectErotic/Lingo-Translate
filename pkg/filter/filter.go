package filter

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"lingo-translate/pkg/decision"
)

// Default regex patterns for system strings, file names, URLs, and code tokens
var defaultIgnoreRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(https?|ftp|file)://`),
	regexp.MustCompile(`(?i)\.(png|jpe?g|gif|bmp|webp|svg|ico|ogg|m4a|mp3|wav|json|js|css|html)$`),
	regexp.MustCompile(`^[^a-zA-Z0-9\p{Thai}\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}]+$`), // Symbols only
	regexp.MustCompile(`(?i)^EV\d{3,}$`),
	regexp.MustCompile(`(?i)^[A-Z][a-zA-Z]+ (open|close|add|remove|set|get|show|hide|enable|disable)`),
	regexp.MustCompile(`(?i)^\\[a-z]\[\d+\]$`),
}

var systemPrefixes = []string{
	"img/", "audio/", "data/", "js/", "fonts/",
	"Actor", "Class", "Skill", "Item", "Weapon", "Armor", "Enemy", "Troop",
	"State", "Animation", "Tileset", "CommonEvent", "System", "MapInfo",
}

// Manager handles smart skipping of non-translatable text entries
type Manager struct {
	mu              sync.RWMutex
	customPatterns  []string
	compiledCustom  []*regexp.Regexp
	engineFilterOn  bool
	decisionEngine  decision.DecisionEngine
	decisionEnabled bool
}

func New() *Manager {
	return &Manager{
		engineFilterOn: true,
	}
}

// SetDecisionEngine configures the auxiliary System One decision engine for ambiguous cases
func (m *Manager) SetDecisionEngine(engine decision.DecisionEngine, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.decisionEngine = engine
	m.decisionEnabled = enabled
}

// SetEngineFilterEnabled toggles engine-level heuristic filtering
func (m *Manager) SetEngineFilterEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engineFilterOn = enabled
}

// Learn adds a new pattern (or exact text) to the ignore list
func (m *Manager) Learn(pattern string) {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.customPatterns {
		if p == trimmed {
			return
		}
	}

	m.customPatterns = append(m.customPatterns, trimmed)
	if compiled, err := regexp.Compile(trimmed); err == nil {
		m.compiledCustom = append(m.compiledCustom, compiled)
	}
}

// Unlearn removes a pattern from the ignore list
func (m *Manager) Unlearn(pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var newPatterns []string
	var newCompiled []*regexp.Regexp

	for _, p := range m.customPatterns {
		if p != pattern {
			newPatterns = append(newPatterns, p)
			if compiled, err := regexp.Compile(p); err == nil {
				newCompiled = append(newCompiled, compiled)
			}
		}
	}

	m.customPatterns = newPatterns
	m.compiledCustom = newCompiled
}

// ShouldSkip checks whether a string should be skipped from translation
func (m *Manager) ShouldSkip(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}

	// Pure numeric check
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return true
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. Check custom learned patterns
	for _, p := range m.customPatterns {
		if p == trimmed {
			return true
		}
	}
	for _, reg := range m.compiledCustom {
		if reg.MatchString(trimmed) {
			return true
		}
	}

	// 2. Check engine heuristics (if enabled)
	if m.engineFilterOn {
		for _, reg := range defaultIgnoreRegexes {
			if reg.MatchString(trimmed) {
				return true
			}
		}
		for _, prefix := range systemPrefixes {
			if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(prefix)) {
				return true
			}
		}
		if strings.Contains(strings.ToLower(trimmed), "$game") {
			return true
		}
	}

	// 3. Ambiguous Gray-Zone check using System One (if enabled and in gray zone)
	if m.decisionEnabled && m.decisionEngine != nil && isAmbiguousGrayZone(trimmed) {
		isPlayerText, conf, err := m.decisionEngine.Noul(context.Background(), trimmed, "Is this player-facing text or code/engine script?")
		if err == nil && conf >= 0.80 {
			// If not player text (i.e. is code), skip it!
			return !isPlayerText
		}
	}

	return false
}

// ShouldSkipWithContext checks whether a string should be skipped from translation using provided context
func (m *Manager) ShouldSkipWithContext(ctx context.Context, text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}

	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return true
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.customPatterns {
		if p == trimmed {
			return true
		}
	}
	for _, reg := range m.compiledCustom {
		if reg.MatchString(trimmed) {
			return true
		}
	}

	if m.engineFilterOn {
		for _, reg := range defaultIgnoreRegexes {
			if reg.MatchString(trimmed) {
				return true
			}
		}
		for _, prefix := range systemPrefixes {
			if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(prefix)) {
				return true
			}
		}
		if strings.Contains(strings.ToLower(trimmed), "$game") {
			return true
		}
	}

	if m.decisionEnabled && m.decisionEngine != nil && isAmbiguousGrayZone(trimmed) {
		isPlayerText, conf, err := m.decisionEngine.Noul(ctx, trimmed, "Is this player-facing text or code/engine script?")
		if err == nil && conf >= 0.80 {
			return !isPlayerText
		}
	}

	return false
}

// isAmbiguousGrayZone determines if a string is borderline between code identifier and player text
func isAmbiguousGrayZone(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Thai, r) || unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			return false
		}
	}

	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 3 {
		return false
	}

	hasUnderscore := strings.Contains(trimmed, "_")
	hasCodePunct := strings.ContainsAny(trimmed, "=();[]{}<>$\\/")
	words := strings.Fields(trimmed)

	if hasUnderscore || hasCodePunct {
		return true
	}

	if len(words) == 1 && len(trimmed) > 4 {
		hasLower := false
		hasUpper := false
		for i, r := range trimmed {
			if i > 0 && unicode.IsUpper(r) {
				hasUpper = true
			}
			if unicode.IsLower(r) {
				hasLower = true
			}
		}
		if hasLower && hasUpper {
			return true
		}
	}

	return false
}

// ExportRules saves custom filter rules to a JSON file
func (m *Manager) ExportRules(filePath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.customPatterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// ImportRules loads and merges filter rules from a JSON file
func (m *Manager) ImportRules(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var patterns []string
	if err := json.Unmarshal(data, &patterns); err != nil {
		return err
	}

	for _, p := range patterns {
		m.Learn(p)
	}
	return nil
}
