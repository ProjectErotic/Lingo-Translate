package jev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestJevClient(t *testing.T) {
	var requestCount int64

	// Mock TypeSafe AI Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)

		if r.Header.Get("Authorization") != "Bearer test-jev-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch req.Primitive {
		case "noul":
			trueVal := true
			_ = json.NewEncoder(w).Encode(decisionResponse{
				Primitive:  "noul",
				Result:     &trueVal,
				Confidence: 0.96,
			})
		case "choice":
			_ = json.NewEncoder(w).Encode(decisionResponse{
				Primitive:  "choice",
				Choice:     req.Choices[0],
				Confidence: 0.91,
			})
		case "score":
			scoreVal := 0.88
			_ = json.NewEncoder(w).Encode(decisionResponse{
				Primitive:  "score",
				Score:      &scoreVal,
				Confidence: 0.95,
			})
		default:
			http.Error(w, "unknown primitive", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:              "test-jev-key",
		BaseURL:             server.URL,
		FallbackToHeuristic: true,
	})
	ctx := context.Background()

	// 1. Test Noul
	val, conf, err := client.Noul(ctx, "Play Game", "Is this a menu button?")
	if err != nil {
		t.Fatalf("Noul failed: %v", err)
	}
	if !val || conf != 0.96 {
		t.Errorf("unexpected Noul response: val=%v, conf=%f", val, conf)
	}
	if atomic.LoadInt64(&requestCount) != 1 {
		t.Errorf("expected 1 HTTP request, got %d", atomic.LoadInt64(&requestCount))
	}

	// 2. Test Memoization Cache on duplicate call (Rule 1: Deduplication!)
	valCached, confCached, err := client.Noul(ctx, "Play Game", "Is this a menu button?")
	if err != nil {
		t.Fatalf("cached Noul failed: %v", err)
	}
	if !valCached || confCached != 0.96 {
		t.Errorf("unexpected cached response: val=%v, conf=%f", valCached, confCached)
	}
	// Request count must STILL be 1 because cache handled it at 0ms!
	if atomic.LoadInt64(&requestCount) != 1 {
		t.Errorf("expected 1 HTTP request (cache hit), got %d", atomic.LoadInt64(&requestCount))
	}

	// 3. Test Choice
	choice, _, err := client.Choice(ctx, "Hero", []string{"character", "location"})
	if err != nil {
		t.Fatalf("Choice failed: %v", err)
	}
	if choice != "character" {
		t.Errorf("expected choice 'character', got '%s'", choice)
	}

	// 4. Test Score
	score, err := client.Score(ctx, "Text", "Fidelity")
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}
	if score != 0.88 {
		t.Errorf("expected score 0.88, got %f", score)
	}

	// 5. Test Fallback when API key is empty
	clientNoKey := NewClient(Config{
		APIKey:              "",
		FallbackToHeuristic: true,
	})
	valFb, _, err := clientNoKey.Noul(ctx, "$gameParty.gold()", "Is this player text?")
	if err != nil {
		t.Fatalf("fallback Noul failed: %v", err)
	}
	if valFb { // Heuristic should detect code and return false
		t.Errorf("expected heuristic to detect code as false, got true")
	}
}
