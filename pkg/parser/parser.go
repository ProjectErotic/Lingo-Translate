package parser

import (
	"context"
	"fmt"
	"strings"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/parser/godot"
	"lingo-translate/pkg/parser/libgdx"
	"lingo-translate/pkg/parser/renpy"
	"lingo-translate/pkg/parser/rpgm"
	"lingo-translate/pkg/parser/unity"
)

// EngineParser defines the interface each game engine parser must implement
type EngineParser interface {
	// Name returns the engine identifier (e.g. "rpgm", "renpy", "godot", "unity")
	Name() string

	// Detect checks if the specified directory belongs to this game engine
	Detect(dir string) bool

	// Extract scans the game files and extracts all translatable text entries
	Extract(ctx context.Context, dir string) ([]model.TextEntry, *model.ExtractionStats, error)

	// Inject writes translated entries into target directory (or back into game data)
	Inject(ctx context.Context, sourceDir, outputDir string, entries []model.TextEntry) error
}

// Registry returns all registered engine parsers
func Registry() []EngineParser {
	return []EngineParser{
		rpgm.New(),
		renpy.New(),
		godot.New(),
		unity.New(),
		libgdx.New(),
	}
}

// DetectEngine inspects a directory and returns the first matching EngineParser
func DetectEngine(dir string) (EngineParser, error) {
	for _, p := range Registry() {
		if p.Detect(dir) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("could not auto-detect game engine for directory: %s", dir)
}

// GetParser returns an EngineParser by its identifier
func GetParser(name string) (EngineParser, error) {
	clean := strings.ToLower(strings.TrimSpace(name))
	for _, p := range Registry() {
		if strings.EqualFold(p.Name(), clean) {
			return p, nil
		}
		// Handle aliases like rpgm-mv, rpgm-mz -> rpgm
		if strings.HasPrefix(clean, "rpgm") && p.Name() == "rpgm" {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported game engine: %s", name)
}

// SupportedEngines returns a list of supported engine names
func SupportedEngines() []string {
	parsers := Registry()
	names := make([]string, len(parsers))
	for i, p := range parsers {
		names[i] = p.Name()
	}
	return names
}
