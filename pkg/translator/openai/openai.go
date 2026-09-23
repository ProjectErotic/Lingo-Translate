package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lingo-translate/pkg/translator"
	"lingo-translate/pkg/translator/prompts"
)

type Config struct {
	BaseURL string // e.g. "https://api.openai.com/v1" or "http://localhost:11434/v1"
	APIKey  string
	Model   string
	Timeout time.Duration
	Headers map[string]string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) Name() string {
	return "openai"
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type streamChatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type streamDelta struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
	Index int         `json:"index"`
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
}

func (c *Client) Translate(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if opts.Stream || opts.Format == "line" {
		return c.translateStream(ctx, texts, opts)
	}

	systemPrompt := prompts.ResolvePrompt(opts.Style, opts.SourceLang, opts.TargetLang, opts.Prompt)
	systemPrompt += "\n\nCRITICAL FORMAT RULE:\nOutput ONLY a valid JSON array of strings corresponding 1-to-1 with the input lines, e.g. [\"line 1\", \"line 2\"]. No markdown, no explanations."

	inputJSON, err := json.Marshal(texts)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input texts: %w", err)
	}

	modelName := c.cfg.Model
	if opts.Model != "" {
		modelName = opts.Model
	}

	reqBody := chatRequest{
		Model: modelName,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(inputJSON)},
		},
		Temperature: 0.3,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	for k, v := range c.cfg.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request to LLM backend failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from LLM")
	}

	rawContent := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Strip markdown code block if model wrapped it in ```json ... ```
	rawContent = strings.TrimPrefix(rawContent, "```json")
	rawContent = strings.TrimPrefix(rawContent, "```")
	rawContent = strings.TrimSuffix(rawContent, "```")
	rawContent = strings.TrimSpace(rawContent)

	var translatedTexts []string
	if err := json.Unmarshal([]byte(rawContent), &translatedTexts); err != nil {
		return nil, fmt.Errorf("failed to parse translated JSON array: %w (raw response: %s)", err, rawContent)
	}

	if len(translatedTexts) != len(texts) {
		return nil, fmt.Errorf("translated count mismatch: expected %d, got %d", len(texts), len(translatedTexts))
	}

	results := make([]translator.Result, len(texts))
	for i := range texts {
		results[i] = translator.Result{
			Source:     texts[i],
			Target:     translatedTexts[i],
			Translator: "openai:" + modelName,
		}
	}

	return results, nil
}

func (c *Client) translateStream(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	modelName := c.cfg.Model
	if opts.Model != "" {
		modelName = opts.Model
	}

	systemPrompt := prompts.ResolvePrompt(opts.Style, opts.SourceLang, opts.TargetLang, opts.Prompt)
	systemPrompt += "\n\nCRITICAL FORMAT RULE:\n1. Output format MUST be EXACTLY line-by-line: [ID] ||| [Translation]\n2. Do NOT output markdown code blocks (NO ```), explanations, or notes. Output ONLY the numbered lines.\n3. Preserve all special tokens and tags like __NST_TAG_0__, __NST_TAG_1__ EXACTLY as they are without modifying or dropping them.\n4. Translate every single numbered item sequentially without skipping."

	var sb strings.Builder
	sb.WriteString("Translate each of the following lines sequentially. Output ONLY '[ID] ||| [Translation]':\n\n")
	for i, text := range texts {
		flat := strings.ReplaceAll(text, "\r\n", "\\n")
		flat = strings.ReplaceAll(flat, "\n", "\\n")
		flat = strings.ReplaceAll(flat, "\r", "\\n")
		sb.WriteString(fmt.Sprintf("%d ||| %s\n", i+1, flat))
	}

	reqBody := streamChatRequest{
		Model: modelName,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: sb.String()},
		},
		Temperature: 0.3,
		Stream:      true,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	for k, v := range c.cfg.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request to LLM streaming backend failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	contentMap := make(map[int]string)
	reasoningMap := make(map[int]string)

	lineRe := regexp.MustCompile(`^(\d+)\s*\|\|\|\s*(.*)$`)

	reader := bufio.NewReader(resp.Body)
	var contentLineBuf strings.Builder
	var reasoningLineBuf strings.Builder

	for {
		lineBytes, err := reader.ReadBytes('\n')
		if len(lineBytes) > 0 {
			line := strings.TrimSpace(string(lineBytes))
			if strings.HasPrefix(line, "data:") {
				dataStr := strings.TrimSpace(line[5:])
				if dataStr == "[DONE]" {
					break
				}
				var chunk streamChunk
				if err := json.Unmarshal([]byte(dataStr), &chunk); err == nil && len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta

					// Reasoning content handling (thinking models like DeepSeek R1/V4)
					if delta.ReasoningContent != "" {
						reasoningLineBuf.WriteString(delta.ReasoningContent)
						for {
							str := reasoningLineBuf.String()
							nl := strings.IndexByte(str, '\n')
							if nl == -1 {
								break
							}
							completedLine := strings.TrimSpace(str[:nl])
							reasoningLineBuf.Reset()
							reasoningLineBuf.WriteString(str[nl+1:])

							if m := lineRe.FindStringSubmatch(completedLine); len(m) == 3 {
								if id, err := strconv.Atoi(m[1]); err == nil && id >= 1 && id <= len(texts) {
									t := strings.ReplaceAll(m[2], `\n`, "\n")
									if !strings.Contains(t, "likely typo") && !strings.Contains(t, "Hmm") {
										reasoningMap[id] = t
									}
								}
							}
						}
					}

					// Content handling
					if delta.Content != "" {
						contentLineBuf.WriteString(delta.Content)
						for {
							str := contentLineBuf.String()
							nl := strings.IndexByte(str, '\n')
							if nl == -1 {
								break
							}
							completedLine := strings.TrimSpace(str[:nl])
							contentLineBuf.Reset()
							contentLineBuf.WriteString(str[nl+1:])

							if m := lineRe.FindStringSubmatch(completedLine); len(m) == 3 {
								if id, err := strconv.Atoi(m[1]); err == nil && id >= 1 && id <= len(texts) {
									t := strings.ReplaceAll(m[2], `\n`, "\n")
									contentMap[id] = t
								}
							}
						}
					}
				}
			}
		}

		if err != nil {
			// EOF or cutoff
			break
		}
	}

	// Flush remaining line in contentLineBuf
	if remaining := strings.TrimSpace(contentLineBuf.String()); remaining != "" {
		if m := lineRe.FindStringSubmatch(remaining); len(m) == 3 {
			if id, err := strconv.Atoi(m[1]); err == nil && id >= 1 && id <= len(texts) {
				contentMap[id] = strings.ReplaceAll(m[2], `\n`, "\n")
			}
		}
	}

	results := make([]translator.Result, len(texts))
	for i, orig := range texts {
		id := i + 1
		var target string
		if t, ok := contentMap[id]; ok && t != "" {
			target = t
		} else if t, ok := reasoningMap[id]; ok && t != "" {
			target = t
		}

		if target != "" {
			results[i] = translator.Result{
				Source:     orig,
				Target:     target,
				Translator: "openai:" + modelName,
			}
		} else {
			results[i] = translator.Result{
				Source: orig,
				Error:  fmt.Errorf("line %d not received from stream", id),
			}
		}
	}

	return results, nil
}

