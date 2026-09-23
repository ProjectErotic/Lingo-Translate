package masker

import (
	"fmt"
	"regexp"
	"strings"
)

// Combined RPG Maker / Game Escape Code Regular Expression
var rpgmControlRegex = regexp.MustCompile(`(?i)(` +
	`\\\\[VNP]\[\d+\]` +
	`|\\\\I\[\d+\]` +
	`|\\\\C\[\d+\]` +
	`|\\\\G` +
	`|\\\\[{}]` +
	`|\\\\\$` +
	`|\\\\[.|]` +
	`|\\\\!` +
	`|\\\\[><]` +
	`|\\\\\\^` +
	`|\\\\\\\\` +
	`|\\\\FS\[\d+\]` +
	`|\\\\P[XY]\[-?\d+\]` +
	`|\\\\[OT]C\[\d+\]` +
	`|\\\\(?:MSGCORE|MSGSND)\[[^\]]*\]` +
	`|\\[VNP]\[\d+\]` +
	`|\\[IC]\[\d+\]` +
	`|\\[G!^\$]` +
	`|\\[{}]` +
	`|\\[.|]` +
	`|\\FS\[\d+\]` +
	`|\\P[XY]\[-?\d+\]` +
	`|\\[OT]C\[\d+\]` +
	`|\\(?:MSGCORE|MSGSND)\[[^\]]*\]` +
	`)`)

// MaskResult contains the safe masked text and the dictionary to restore original codes
type MaskResult struct {
	OriginalText string
	MaskedText   string
	TagMap       map[string]string // Tag (e.g. "__LINGO_TAG_0__") -> Code (e.g. "\C[1]")
}

// Mask replaces control codes in source text with safe placeholders that LLMs won't translate
func Mask(sourceText string) MaskResult {
	if sourceText == "" {
		return MaskResult{
			OriginalText: sourceText,
			MaskedText:   sourceText,
			TagMap:       make(map[string]string),
		}
	}

	tagMap := make(map[string]string)
	matches := rpgmControlRegex.FindAllString(sourceText, -1)

	tagIndex := 0
	for _, code := range matches {
		alreadyMapped := false
		for _, mappedCode := range tagMap {
			if mappedCode == code {
				alreadyMapped = true
				break
			}
		}

		if !alreadyMapped {
			tag := fmt.Sprintf("__LINGO_TAG_%d__", tagIndex)
			tagIndex++
			tagMap[tag] = code
		}
	}

	maskedText := sourceText
	for tag, code := range tagMap {
		maskedText = strings.ReplaceAll(maskedText, code, tag)
	}

	return MaskResult{
		OriginalText: sourceText,
		MaskedText:   maskedText,
		TagMap:       tagMap,
	}
}

// Unmask restores the original control codes from the TagMap.
// It also provides fuzzy recovery for common LLM mutations (extra spaces, lowercasing, etc.)
func Unmask(translatedText string, tagMap map[string]string) string {
	if translatedText == "" || len(tagMap) == 0 {
		return translatedText
	}

	result := translatedText

	// 1. Direct exact replacement
	for tag, originalCode := range tagMap {
		result = strings.ReplaceAll(result, tag, originalCode)
	}

	// 2. Fuzzy recovery for common LLM mutations: "__ LINGO_TAG_0 __" or "__nst_tag_0__"
	for tag, originalCode := range tagMap {
		// If tag still remains un-restored due to slight formatting changes
		numMatch := regexp.MustCompile(`\d+`).FindString(tag)
		if numMatch != "" {
			fuzzyPattern := regexp.MustCompile(fmt.Sprintf(`(?i)__\s*(?:LINGO|NST)_TAG_%s\s*__`, numMatch))
			result = fuzzyPattern.ReplaceAllString(result, originalCode)
		}
	}

	return result
}
