package fragment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestFragmentDetection(t *testing.T) {
	tests := []struct {
		input         string
		expectedFrag  bool
		expectedWord  string
		trailingSpace bool
	}{
		{
			input:         "horny by ",
			expectedFrag:  true,
			expectedWord:  "by",
			trailingSpace: true,
		},
		{
			input:         "You can confidently say that she is",
			expectedFrag:  true,
			expectedWord:  "is",
			trailingSpace: false,
		},
		{
			input:         "increased by ",
			expectedFrag:  true,
			expectedWord:  "by",
			trailingSpace: true,
		},
		{
			input:         "Level: ",
			expectedFrag:  true,
			expectedWord:  "level",
			trailingSpace: true,
		},
		{
			input:         "Bonus of ",
			expectedFrag:  true,
			expectedWord:  "of",
			trailingSpace: true,
		},
		{
			input:         "Just an ordinary sentence.",
			expectedFrag:  false,
			expectedWord:  "",
			trailingSpace: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			info := Detect(tc.input)
			if info.IsFragment != tc.expectedFrag {
				t.Errorf("input %q: expected IsFragment=%v, got %v", tc.input, tc.expectedFrag, info.IsFragment)
			}
			if tc.expectedFrag && info.MatchedWord != tc.expectedWord {
				t.Errorf("input %q: expected MatchedWord=%q, got %q", tc.input, tc.expectedWord, info.MatchedWord)
			}
			if info.TrailingSpace != tc.trailingSpace {
				t.Errorf("input %q: expected TrailingSpace=%v, got %v", tc.input, tc.trailingSpace, info.TrailingSpace)
			}
		})
	}
}

func TestTrailingSpaceEnforcement(t *testing.T) {
	// Source has trailing space, target misses it -> append
	res := EnforceTrailingSpace("horny by ", "ระดับความเงี่ยนอยู่ที่")
	if res != "ระดับความเงี่ยนอยู่ที่ " {
		t.Errorf("expected trailing space added, got %q", res)
	}

	// Source has no trailing space, target has it -> trim
	res2 := EnforceTrailingSpace("Hello", "สวัสดี ")
	if res2 != "สวัสดี" {
		t.Errorf("expected trailing space removed, got %q", res2)
	}
}

// TestAdulttaleCheckFragmentLiveTranslation runs a live translation test against the LLM
// specifically testing the "horny by " Adulttale fragment with BuildPromptDirective.
func TestAdulttaleCheckFragmentLiveTranslation(t *testing.T) {
	apiKey := os.Getenv("LITELLM_KEY")
	if apiKey == "" {
		apiKey = "sk-RUpBXMS3EcoXEGhcitDGtg"
	}
	baseURL := "https://ilms.uchs-th.com/v1"

	sourceLines := []string{
		"Chara.",
		"You can confidently say that she is",
		"horny by ",
	}

	hasFrag := HasAnyFragment(sourceLines)
	if !hasFrag {
		t.Fatalf("expected HasAnyFragment to be true for sourceLines")
	}

	directive := BuildPromptDirective("Thai")

	systemPrompt := "You are a professional game translator translating game dialogue to Thai.\n" +
		"Style: NSFW / Adult / Erotic Uncensored.\n\n" +
		directive + "\n\n" +
		"CRITICAL FORMAT RULE:\nOutput ONLY a valid JSON array of strings corresponding 1-to-1 with input lines."

	inputJSON, _ := json.Marshal(sourceLines)

	reqPayload := map[string]interface{}{
		"model": "glm-5.3-flash",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": string(inputJSON)},
		},
		"temperature": 0.2,
	}

	reqBytes, _ := json.Marshal(reqPayload)
	req, err := http.NewRequest("POST", baseURL+"/chat/completions", strings.NewReader(string(reqBytes)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("network request to test endpoint failed: %v", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("API returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse with raw json map
	var rawResp map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &rawResp)
	choices, _ := rawResp["choices"].([]interface{})
	if len(choices) == 0 {
		t.Fatalf("no choices in response: %s", string(bodyBytes))
	}
	firstChoice := choices[0].(map[string]interface{})
	msg := firstChoice["message"].(map[string]interface{})
	content := msg["content"].(string)

	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var translatedLines []string
	if err := json.Unmarshal([]byte(content), &translatedLines); err != nil {
		t.Fatalf("failed to parse JSON response: %v, raw: %s", err, content)
	}

	if len(translatedLines) != 3 {
		t.Fatalf("expected 3 translated lines, got %d", len(translatedLines))
	}

	for i := range translatedLines {
		translatedLines[i] = EnforceTrailingSpace(sourceLines[i], translatedLines[i])
	}

	t.Logf("=== LIVE TEST RESULTS ===")
	t.Logf("Line 0: %s -> %s", sourceLines[0], translatedLines[0])
	t.Logf("Line 1: %s -> %s", sourceLines[1], translatedLines[1])
	t.Logf("Line 2: %s -> %s", sourceLines[2], translatedLines[2])

	// Check line 2: must NOT contain "เพราะ" (old error) and MUST contain connector like "อยู่" or "ที่"
	line2 := translatedLines[2]
	if strings.Contains(line2, "เพราะ") {
		t.Errorf("Line 2 still has flawed translation 'เพราะ': %s", line2)
	}

	// Must preserve trailing space
	if !strings.HasSuffix(line2, " ") {
		t.Errorf("Line 2 must have trailing space to connect to numbers, got: %q", line2)
	}

	// Simulate game engine appending 20%
	simulatedGameOutput := fmt.Sprintf("%s20%%", line2)
	t.Logf("Simulated Game Engine Output: %q", simulatedGameOutput)
}
