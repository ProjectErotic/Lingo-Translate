package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lingo-translate/pkg/translator"
	"lingo-translate/pkg/translator/prompts"
)

type Config struct {
	APIKey  string
	Model   string // e.g. "gemini-2.5-flash"
	Timeout time.Duration
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config) *Client {
	if cfg.Model == "" {
		cfg.Model = "gemini-3.8-flash"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) Name() string {
	return "gemini"
}

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type generationConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

type geminiRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

type geminiCandidate struct {
	Content content `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) Translate(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	systemInstruction := prompts.ResolvePrompt(opts.Style, opts.SourceLang, opts.TargetLang, opts.Prompt)
	systemInstruction += "\n\nCRITICAL FORMAT RULE:\nReturn a valid JSON array of strings corresponding 1-to-1 with input lines. No commentary."

	inputJSON, err := json.Marshal(texts)
	if err != nil {
		return nil, err
	}

	modelName := c.cfg.Model
	if opts.Model != "" {
		modelName = opts.Model
	}

	reqBody := geminiRequest{
		Contents: []content{
			{
				Role:  "user",
				Parts: []part{{Text: string(inputJSON)}},
			},
		},
		SystemInstruction: &content{
			Parts: []part{{Text: systemInstruction}},
		},
		GenerationConfig: &generationConfig{
			Temperature:      0.2,
			ResponseMimeType: "application/json",
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, c.cfg.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini api request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty candidates in gemini response")
	}

	rawText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	var translatedTexts []string
	if err := json.Unmarshal([]byte(rawText), &translatedTexts); err != nil {
		return nil, fmt.Errorf("failed to parse translated JSON: %w (raw: %s)", err, rawText)
	}

	if len(translatedTexts) != len(texts) {
		return nil, fmt.Errorf("translated count mismatch: expected %d, got %d", len(texts), len(translatedTexts))
	}

	results := make([]translator.Result, len(texts))
	for i := range texts {
		results[i] = translator.Result{
			Source:     texts[i],
			Target:     translatedTexts[i],
			Translator: "gemini:" + modelName,
		}
	}

	return results, nil
}
