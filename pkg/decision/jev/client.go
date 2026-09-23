package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"lingo-translate/pkg/decision"
)

// Config holds connection details for TypeSafe AI Jev
type Config struct {
	APIKey              string
	BaseURL             string
	ConfidenceThreshold float64
	Timeout             time.Duration
	FallbackToHeuristic bool
}

// Client interacts with the TypeSafe AI System One (Jev) Decision Engine
type Client struct {
	cfg       Config
	http      *http.Client
	cache     *decision.MemoCache
	heuristic *decision.HeuristicEngine
}

type decisionRequest struct {
	Primitive string   `json:"primitive"`
	Input     string   `json:"input"`
	Question  string   `json:"question,omitempty"`
	Choices   []string `json:"choices,omitempty"`
	Criteria  string   `json:"criteria,omitempty"`
}

type decisionResponse struct {
	Primitive  string   `json:"primitive"`
	Result     *bool    `json:"result,omitempty"`
	Choice     string   `json:"choice,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Confidence float64  `json:"confidence"`
	Error      string   `json:"error,omitempty"`
}

// NewClient creates a new Jev decision client with memoization cache and heuristic fallback
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.typesafe.ai/v1"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout == 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.ConfidenceThreshold <= 0 {
		cfg.ConfidenceThreshold = 0.85
	}

	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
		cache:     decision.NewMemoCache(20000),
		heuristic: decision.NewHeuristicEngine(),
	}
}

func (c *Client) Name() string {
	return "typesafe_jev"
}

// Cache returns the internal memoization cache
func (c *Client) Cache() *decision.MemoCache {
	return c.cache
}

// Noul evaluates a boolean Yes/No condition on input text
func (c *Client) Noul(ctx context.Context, input string, question string) (bool, float64, error) {
	// 1. Check in-memory memoization cache (0ms lookup)
	if cached, hit := c.cache.Get(decision.PrimitiveNoul, question, input); hit {
		return cached.BoolValue, cached.Confidence, nil
	}

	// 2. If API Key is not set or Fallback requested, use deterministic heuristic
	if c.cfg.APIKey == "" {
		res, conf, err := c.heuristic.Noul(ctx, input, question)
		c.cache.Set(decision.PrimitiveNoul, question, input, decision.DecisionResult{
			Primitive:  decision.PrimitiveNoul,
			BoolValue:  res,
			Confidence: conf,
			Source:     "heuristic",
		}, 24*time.Hour)
		return res, conf, err
	}

	reqBody := decisionRequest{
		Primitive: string(decision.PrimitiveNoul),
		Input:     input,
		Question:  question,
	}

	start := time.Now()
	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		if c.cfg.FallbackToHeuristic {
			log.Printf("[Lingo] [Jev] API error, falling back to heuristic: %v", err)
			return c.heuristic.Noul(ctx, input, question)
		}
		return false, 0.0, err
	}

	val := false
	if resp.Result != nil {
		val = *resp.Result
	}

	c.cache.Set(decision.PrimitiveNoul, question, input, decision.DecisionResult{
		Primitive:  decision.PrimitiveNoul,
		BoolValue:  val,
		Confidence: resp.Confidence,
		Source:     "jev",
		Latency:    time.Since(start),
	}, 24*time.Hour)

	return val, resp.Confidence, nil
}

// Choice selects the most accurate option from a set of choices
func (c *Client) Choice(ctx context.Context, input string, choices []string) (string, float64, error) {
	queryKey := strings.Join(choices, ",")
	if cached, hit := c.cache.Get(decision.PrimitiveChoice, queryKey, input); hit {
		return cached.Choice, cached.Confidence, nil
	}

	if c.cfg.APIKey == "" {
		res, conf, err := c.heuristic.Choice(ctx, input, choices)
		c.cache.Set(decision.PrimitiveChoice, queryKey, input, decision.DecisionResult{
			Primitive:  decision.PrimitiveChoice,
			Choice:     res,
			Confidence: conf,
			Source:     "heuristic",
		}, 24*time.Hour)
		return res, conf, err
	}

	reqBody := decisionRequest{
		Primitive: string(decision.PrimitiveChoice),
		Input:     input,
		Choices:   choices,
	}

	start := time.Now()
	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		if c.cfg.FallbackToHeuristic {
			log.Printf("[Lingo] [Jev] API error, falling back to heuristic: %v", err)
			return c.heuristic.Choice(ctx, input, choices)
		}
		return "", 0.0, err
	}

	c.cache.Set(decision.PrimitiveChoice, queryKey, input, decision.DecisionResult{
		Primitive:  decision.PrimitiveChoice,
		Choice:     resp.Choice,
		Confidence: resp.Confidence,
		Source:     "jev",
		Latency:    time.Since(start),
	}, 24*time.Hour)

	return resp.Choice, resp.Confidence, nil
}

// Score evaluates input against criteria and returns a score
func (c *Client) Score(ctx context.Context, input string, criteria string) (float64, error) {
	if cached, hit := c.cache.Get(decision.PrimitiveScore, criteria, input); hit {
		return cached.Score, nil
	}

	if c.cfg.APIKey == "" {
		res, err := c.heuristic.Score(ctx, input, criteria)
		c.cache.Set(decision.PrimitiveScore, criteria, input, decision.DecisionResult{
			Primitive:  decision.PrimitiveScore,
			Score:      res,
			Confidence: 1.0,
			Source:     "heuristic",
		}, 24*time.Hour)
		return res, err
	}

	reqBody := decisionRequest{
		Primitive: string(decision.PrimitiveScore),
		Input:     input,
		Criteria:  criteria,
	}

	start := time.Now()
	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		if c.cfg.FallbackToHeuristic {
			log.Printf("[Lingo] [Jev] API error, falling back to heuristic: %v", err)
			return c.heuristic.Score(ctx, input, criteria)
		}
		return 0.0, err
	}

	score := 0.0
	if resp.Score != nil {
		score = *resp.Score
	}

	c.cache.Set(decision.PrimitiveScore, criteria, input, decision.DecisionResult{
		Primitive:  decision.PrimitiveScore,
		Score:      score,
		Confidence: resp.Confidence,
		Source:     "jev",
		Latency:    time.Since(start),
	}, 24*time.Hour)

	return score, nil
}

func (c *Client) doRequest(ctx context.Context, reqBody decisionRequest) (*decisionResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal decision request failed: %w", err)
	}

	reqURL := fmt.Sprintf("%s/decision", c.cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create http request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.APIKey))

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jev api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var dResp decisionResponse
	if err := json.Unmarshal(bodyBytes, &dResp); err != nil {
		return nil, fmt.Errorf("unmarshal decision response failed: %w", err)
	}

	if dResp.Error != "" {
		return nil, fmt.Errorf("jev api error: %s", dResp.Error)
	}

	return &dResp, nil
}
