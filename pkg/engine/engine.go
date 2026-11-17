package engine

import (
	"fmt"
	"strings"
)

// KeyContext holds reverse-engineered engine context for a translation key.
type KeyContext struct {
	Key                 string   `json:"key"`
	Callers             []string `json:"callers,omitempty"`
	EngineType          string   `json:"engine_type"`
	DynamicSuffix       string   `json:"dynamic_suffix,omitempty"`
	Placeholders        []string `json:"placeholders,omitempty"`
	TrailingSpaceNeeded bool     `json:"trailing_space_needed"`
	SimulatedOutput     string   `json:"simulated_output,omitempty"`
	GMLSourceLines      []int    `json:"gml_source_lines,omitempty"`
}

// EngineReport contains summary of reverse-engineering findings for a game.
type EngineReport struct {
	EngineType       string                 `json:"engine_type"`
	IsYYC            bool                   `json:"is_yyc"`
	TotalStrings     int                    `json:"total_strings"`
	TotalScripts     int                    `json:"total_scripts"`
	TargetKeysTested int                    `json:"target_keys_tested"`
	KeyAnalyses      map[string]*KeyContext `json:"key_analyses"`
}

// SimulateInGameRender simulates how the game engine will concatenate and render the string at runtime.
func SimulateInGameRender(ctx *KeyContext, translatedText string) string {
	if ctx == nil {
		return translatedText
	}

	res := translatedText

	// If the key expects placeholders like {0}, {1}
	if len(ctx.Placeholders) > 0 {
		for i := range ctx.Placeholders {
			token := fmt.Sprintf("{%d}", i)
			res = strings.ReplaceAll(res, token, fmt.Sprintf("<Var%d>", i))
		}
	}

	// If engine appends a dynamic stat/percentage at runtime
	if ctx.DynamicSuffix != "" {
		if ctx.TrailingSpaceNeeded && !strings.HasSuffix(res, " ") {
			res = res + " "
		}
		res = res + "20" + ctx.DynamicSuffix
	}

	return res
}
