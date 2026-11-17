package anchor

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LineAnchor binds a unique short hash and sequential index to an input line.
type LineAnchor struct {
	Hash  string // 3-character random hex hash
	Index int    // 0-based sequential line index
}

// Tag returns the string representation e.g. "[#a7x:0]"
func (la LineAnchor) Tag() string {
	return fmt.Sprintf("[#%s:%d]", la.Hash, la.Index)
}

// AnchoredBatch packages a list of original source lines with their verification anchors.
type AnchoredBatch struct {
	Anchors []LineAnchor
	Sources []string
}

// NewAnchoredBatch generates a unique anchor for each line in the batch.
func NewAnchoredBatch(texts []string) *AnchoredBatch {
	anchors := make([]LineAnchor, len(texts))
	for i := range texts {
		var buf [2]byte
		_, _ = rand.Read(buf[:])
		hash := hex.EncodeToString(buf[:])[:3]
		anchors[i] = LineAnchor{
			Hash:  hash,
			Index: i,
		}
	}
	return &AnchoredBatch{
		Anchors: anchors,
		Sources: texts,
	}
}

// FormatInput generates the anchored input block for the LLM prompt.
// Example:
// [#a7x:0] Hello world
// [#b2k:1] How are you?
func (ab *AnchoredBatch) FormatInput() string {
	var sb strings.Builder
	for i, text := range ab.Sources {
		sb.WriteString(ab.Anchors[i].Tag())
		sb.WriteString(" ")
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Regex to capture anchor tags: `\[#([a-f0-9]{3}):(\d+)\]`
var anchorRegex = regexp.MustCompile(`\[#([a-f0-9]{3}):(\d+)\]`)

// ParseResult holds the parsed translations or alignment error details.
type ParseResult struct {
	Translations []string
	Success      bool
	ErrorType    string // "count_mismatch", "anchor_missing", "index_out_of_order", "duplicate_anchor"
	ErrorDetail  string
	FailedIndex  int
}

// ParseAndVerify parses the raw LLM response text and verifies 100% alignment against expected anchors.
func (ab *AnchoredBatch) ParseAndVerify(rawResponse string) ParseResult {
	expectedCount := len(ab.Anchors)
	if expectedCount == 0 {
		return ParseResult{Success: true}
	}

	lines := strings.Split(rawResponse, "\n")
	parsedTranslations := make(map[int]string)
	seenAnchors := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		loc := anchorRegex.FindStringSubmatchIndex(trimmed)
		if loc == nil {
			continue
		}

		hash := trimmed[loc[2]:loc[3]]
		idxStr := trimmed[loc[4]:loc[5]]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}

		tag := fmt.Sprintf("[#%s:%d]", hash, idx)
		if seenAnchors[tag] {
			return ParseResult{
				Success:     false,
				ErrorType:   "duplicate_anchor",
				ErrorDetail: fmt.Sprintf("duplicate anchor tag encountered: %s", tag),
				FailedIndex: idx,
			}
		}
		seenAnchors[tag] = true

		// Content is everything after the tag
		content := strings.TrimSpace(trimmed[loc[1]:])
		parsedTranslations[idx] = content
	}

	// Verify all expected anchors exist in order
	results := make([]string, expectedCount)
	for i, expected := range ab.Anchors {
		content, exists := parsedTranslations[expected.Index]
		if !exists {
			return ParseResult{
				Success:     false,
				ErrorType:   "anchor_missing",
				ErrorDetail: fmt.Sprintf("expected anchor %s was dropped by model", expected.Tag()),
				FailedIndex: i,
			}
		}
		results[i] = content
	}

	if len(parsedTranslations) != expectedCount {
		return ParseResult{
			Success:     false,
			ErrorType:   "count_mismatch",
			ErrorDetail: fmt.Sprintf("parsed %d translations but expected %d", len(parsedTranslations), expectedCount),
			FailedIndex: -1,
		}
	}

	return ParseResult{
		Translations: results,
		Success:      true,
	}
}
