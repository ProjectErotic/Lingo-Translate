package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lingo-translate/pkg/translator"
)

type Config struct {
	APIKey string
}

type Provider struct {
	apiKey     string
	httpClient *http.Client
}

func New(cfg Config) *Provider {
	return &Provider{
		apiKey: cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *Provider) Name() string {
	return "google"
}

// langCode maps language names to ISO 639-1 codes
func langCode(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch l {
	case "korean", "ko", "kr":
		return "ko"
	case "japanese", "ja", "jp":
		return "ja"
	case "thai", "th":
		return "th"
	case "english", "en":
		return "en"
	case "chinese", "zh":
		return "zh"
	default:
		if len(l) == 2 {
			return l
		}
		return "auto"
	}
}

func (p *Provider) Translate(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	results := make([]translator.Result, len(texts))
	sl := langCode(opts.SourceLang)
	tl := langCode(opts.TargetLang)
	if tl == "" || tl == "auto" {
		tl = "th"
	}

	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if strings.TrimSpace(text) == "" {
			results[i] = translator.Result{
				Source:     text,
				Target:     text,
				Translator: p.Name(),
			}
			continue
		}

		translated, err := p.translateOne(ctx, text, sl, tl)
		if err != nil {
			// Fallback to original text to avoid black screen
			results[i] = translator.Result{
				Source:     text,
				Target:     text,
				Translator: p.Name(),
				Error:      err,
			}
		} else {
			results[i] = translator.Result{
				Source:     text,
				Target:     translated,
				Translator: p.Name(),
			}
		}

		// Brief delay to be polite to endpoint
		time.Sleep(100 * time.Millisecond)
	}

	return results, nil
}

func (p *Provider) translateOne(ctx context.Context, text, sl, tl string) (string, error) {
	// Try clients5 endpoint first (Chrome extension client, highly resilient to rate limits)
	res, err := p.translateClients5(ctx, text, sl, tl)
	if err == nil && res != "" {
		return res, nil
	}

	// Fallback to GTX single endpoint
	return p.translateGTX(ctx, text, sl, tl)
}

func (p *Provider) translateClients5(ctx context.Context, text, sl, tl string) (string, error) {
	reqURL := fmt.Sprintf(
		"https://clients5.google.com/translate_a/t?client=dict-chrome-ex&sl=%s&tl=%s&q=%s",
		url.QueryEscape(sl),
		url.QueryEscape(tl),
		url.QueryEscape(text),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("clients5 returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var results []string
	if err := json.Unmarshal(body, &results); err == nil && len(results) > 0 {
		return strings.TrimSpace(results[0]), nil
	}

	var raw []interface{}
	if err := json.Unmarshal(body, &raw); err == nil && len(raw) > 0 {
		if s, ok := raw[0].(string); ok {
			return strings.TrimSpace(s), nil
		}
	}

	return "", fmt.Errorf("unrecognized clients5 response format")
}

func (p *Provider) translateGTX(ctx context.Context, text, sl, tl string) (string, error) {
	reqURL := fmt.Sprintf(
		"https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s",
		url.QueryEscape(sl),
		url.QueryEscape(tl),
		url.QueryEscape(text),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) NST/2.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google translate returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var raw []interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	if len(raw) == 0 {
		return "", fmt.Errorf("empty translation response")
	}

	sentences, ok := raw[0].([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected sentence format in response")
	}

	var sb strings.Builder
	for _, s := range sentences {
		if tuple, ok := s.([]interface{}); ok && len(tuple) > 0 {
			if seg, ok := tuple[0].(string); ok {
				sb.WriteString(seg)
			}
		}
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return text, nil
	}
	return result, nil
}
