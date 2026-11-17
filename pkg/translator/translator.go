package translator

import (
	"context"
)

// Options configuration passed to translation requests
type Options struct {
	SourceLang string
	TargetLang string
	Model      string
	Glossary   map[string]string
	Prompt     string
	Style      string   // Translation style/persona: "standard", "nsfw", "vn_romance", etc.
	Stream     bool     // Enable streaming mode (SSE)
	Format     string   // "json" (default) or "line" ([ID] ||| [Text])
	Context    []string // Rolling Previous Context lines for reference only (LinguaGacha standard)
}

// Result represents the outcome for a single translated text string
type Result struct {
	Source     string
	Target     string
	Translator string
	Error      error
}

// Translator is the interface all translation service providers implement
type Translator interface {
	Name() string
	Translate(ctx context.Context, texts []string, opts Options) ([]Result, error)
}
