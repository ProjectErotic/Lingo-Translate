package glossary

import (
	"fmt"
	"strings"
	"unicode"
)

// Entry defines a single glossary term with optional translation notes
type Entry struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	Target        string `json:"target"`
	Note          string `json:"note,omitempty"`
	CaseSensitive bool   `json:"case_sensitive"`
	WordBoundary  bool   `json:"word_boundary"` // Enforce whole-word boundary for Latin/alphanumeric terms
}

// FormatPromptEntry formats an entry according to the LinguaGacha/GalTransl standard:
// <Source> -> <Target> #<Note>
func (e Entry) FormatPromptEntry() string {
	if e.Note != "" {
		return fmt.Sprintf("%s -> %s #%s", e.Source, e.Target, e.Note)
	}
	return fmt.Sprintf("%s -> %s", e.Source, e.Target)
}

// trieNode represents a node in the Aho-Corasick automaton
type trieNode struct {
	children map[rune]*trieNode
	fail     *trieNode
	output   []int // Indices of matched entries
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[rune]*trieNode),
		output:   nil,
	}
}

// Matcher provides fast, sub-millisecond multi-term matching over text batches
type Matcher struct {
	entries []Entry
	rootCS  *trieNode // Case-sensitive trie
	rootCI  *trieNode // Case-insensitive trie
}

// NewMatcher builds a compiled Matcher from a list of glossary entries
func NewMatcher(entries []Entry) *Matcher {
	m := &Matcher{
		entries: entries,
		rootCS:  newTrieNode(),
		rootCI:  newTrieNode(),
	}

	for idx, entry := range entries {
		src := strings.TrimSpace(entry.Source)
		if src == "" {
			continue
		}

		if entry.CaseSensitive {
			m.insert(m.rootCS, src, idx)
		} else {
			m.insert(m.rootCI, strings.ToLower(src), idx)
		}
	}

	m.buildFailureLinks(m.rootCS)
	m.buildFailureLinks(m.rootCI)

	return m
}

func (m *Matcher) insert(root *trieNode, pattern string, entryIdx int) {
	curr := root
	for _, r := range pattern {
		if _, exists := curr.children[r]; !exists {
			curr.children[r] = newTrieNode()
		}
		curr = curr.children[r]
	}
	curr.output = append(curr.output, entryIdx)
}

// buildFailureLinks builds BFS failure transitions for Aho-Corasick
func (m *Matcher) buildFailureLinks(root *trieNode) {
	queue := make([]*trieNode, 0)

	// Level 1 nodes fail to root
	for _, child := range root.children {
		child.fail = root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for r, child := range curr.children {
			// Find fail state
			failNode := curr.fail
			for failNode != nil && failNode.children[r] == nil {
				failNode = failNode.fail
			}
			if failNode == nil {
				child.fail = root
			} else {
				child.fail = failNode.children[r]
			}

			// Merge outputs from fail node
			if child.fail != nil && len(child.fail.output) > 0 {
				child.output = append(child.output, child.fail.output...)
			}

			queue = append(queue, child)
		}
	}
}

// MatchBatch scans a slice of text strings and returns ONLY the glossary entries
// that appear in the texts. Duplicate matches are deduplicated.
func (m *Matcher) MatchBatch(texts []string) []Entry {
	if len(texts) == 0 || len(m.entries) == 0 {
		return nil
	}

	matchedIndices := make(map[int]struct{})

	for _, text := range texts {
		if text == "" {
			continue
		}

		// 1. Scan case-sensitive
		m.scan(m.rootCS, text, matchedIndices)

		// 2. Scan case-insensitive
		m.scan(m.rootCI, strings.ToLower(text), matchedIndices)
	}

	if len(matchedIndices) == 0 {
		return nil
	}

	result := make([]Entry, 0, len(matchedIndices))
	for idx := range matchedIndices {
		result = append(result, m.entries[idx])
	}

	return result
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (m *Matcher) scan(root *trieNode, text string, matchedIndices map[int]struct{}) {
	runes := []rune(text)
	curr := root
	for idx, r := range runes {
		for curr != nil && curr.children[r] == nil {
			curr = curr.fail
		}
		if curr == nil {
			curr = root
			continue
		}
		curr = curr.children[r]
		if curr != nil && len(curr.output) > 0 {
			for _, entryIdx := range curr.output {
				entry := m.entries[entryIdx]
				if entry.WordBoundary {
					patLen := len([]rune(entry.Source))
					startIdx := idx - patLen + 1
					if startIdx > 0 && isWordChar(runes[startIdx-1]) {
						continue
					}
					if idx+1 < len(runes) && isWordChar(runes[idx+1]) {
						continue
					}
				}
				matchedIndices[entryIdx] = struct{}{}
			}
		}
	}
}

// FormatPrompt generates the prompt block to inject into the LLM prompt.
// If no entries are matched, returns empty string.
func FormatPrompt(entries []Entry) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Glossary (strictly enforce these character names & terms):\n")
	for _, e := range entries {
		sb.WriteString("- ")
		sb.WriteString(e.FormatPromptEntry())
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

// ApproximateTokens estimates token count for a text string (avg ~4 chars per token)
func ApproximateTokens(s string) int {
	if s == "" {
		return 0
	}
	words := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}
	// Fallback/standard heuristic: max(words * 1.3, chars / 4)
	charTokens := len(s) / 4
	wordTokens := int(float64(words) * 1.3)
	if wordTokens > charTokens {
		return wordTokens
	}
	return charTokens
}
