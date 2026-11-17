package fragment

import (
	"regexp"
	"strings"
)

// TrailingPrepositionRegex captures sentence fragments that end with a preposition,
// linking verb, or colon indicating an incomplete string awaiting a dynamic variable
// (e.g. "horny by ", "increased by ", "level: ", "cost is ")
var trailingPrepositionRegex = regexp.MustCompile(`(?i)(?:^|\s)(by|of|at|to|in|for|with|from|is|are|was|were|has|have|had|gets?|reaches?|cost|price|level|lv\.?|bonus)[\s:]*$`)

// Info holds metadata about a detected sentence fragment
type Info struct {
	IsFragment    bool
	TrailingSpace bool
	MatchedWord   string
}

// Detect checks if a text string is an incomplete fragment awaiting dynamic game variables.
func Detect(s string) Info {
	if s == "" {
		return Info{}
	}

	hasTrailingSpace := strings.HasSuffix(s, " ")

	matches := trailingPrepositionRegex.FindStringSubmatch(s)
	if len(matches) > 1 {
		return Info{
			IsFragment:    true,
			TrailingSpace: hasTrailingSpace,
			MatchedWord:   strings.ToLower(matches[1]),
		}
	}

	// Also catch strings ending with colon plus space like "Gold: " or "HP: "
	if strings.HasSuffix(s, ": ") || strings.HasSuffix(s, ":") {
		return Info{
			IsFragment:    true,
			TrailingSpace: hasTrailingSpace,
			MatchedWord:   ":",
		}
	}

	return Info{}
}

// HasAnyFragment checks a batch of text lines and returns true if at least one is a fragment.
func HasAnyFragment(texts []string) bool {
	for _, text := range texts {
		if Detect(text).IsFragment {
			return true
		}
	}
	return false
}

// BuildPromptDirective returns the system instruction explaining how the LLM must handle fragments.
func BuildPromptDirective(targetLang string) string {
	if strings.EqualFold(targetLang, "thai") || targetLang == "ไทย" {
		return "CRITICAL DYNAMIC FRAGMENT RULE:\n" +
			"Some lines are incomplete sentence fragments ending with connectors like 'by ', 'is ', 'of ' (e.g. 'horny by '). " +
			"The game engine dynamically appends numbers, percentages, or values at runtime. " +
			"Translate these endings so they naturally connect to numbers (e.g. use 'อยู่ที่ ', 'จำนวน ', 'คือ ') " +
			"and ALWAYS preserve the trailing space so the appended number is not glued to the Thai word."
	}

	return "CRITICAL DYNAMIC FRAGMENT RULE:\n" +
		"Some lines are incomplete sentence fragments awaiting dynamic engine variables at runtime. " +
		"Translate them so they naturally connect to appended values/numbers and preserve trailing spaces."
}

// EnforceTrailingSpace ensures that if the source had a trailing space, the target also preserves it.
func EnforceTrailingSpace(source, target string) string {
	if strings.HasSuffix(source, " ") && !strings.HasSuffix(target, " ") {
		return target + " "
	}
	if !strings.HasSuffix(source, " ") && strings.HasSuffix(target, " ") {
		return strings.TrimRight(target, " ")
	}
	return target
}
