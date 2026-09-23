package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lingo-translate/pkg/translator"
)

func TestTranslateJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"choices":[{"message":{"role":"assistant","content":"[\"สวัสดี\",\"ขอบคุณ\"]"}}]}`)
	}))
	defer ts.Close()

	client := New(Config{
		BaseURL: ts.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	})

	res, err := client.Translate(context.Background(), []string{"Hello", "Thank you"}, translator.Options{
		SourceLang: "English",
		TargetLang: "Thai",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Target != "สวัสดี" || res[1].Target != "ขอบคุณ" {
		t.Errorf("unexpected targets: %+v", res)
	}
}

func TestTranslateStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		// Stream reasoning chunk
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thinking about lines...\"}}]}\n\n")
		flusher.Flush()

		// Stream line 1
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"1 ||| สวัสดีครับ\\n\"}}]}\n\n")
		flusher.Flush()

		// Stream line 2 in multiple deltas
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"2 ||| ขอบ\"}}]}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"คุณครับ\\n\"}}]}\n\n")
		flusher.Flush()

		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer ts.Close()

	client := New(Config{
		BaseURL: ts.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})

	res, err := client.Translate(context.Background(), []string{"Hello", "Thank you"}, translator.Options{
		SourceLang: "English",
		TargetLang: "Thai",
		Stream:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Target != "สวัสดีครับ" {
		t.Errorf("expected 'สวัสดีครับ', got %q", res[0].Target)
	}
	if res[1].Target != "ขอบคุณครับ" {
		t.Errorf("expected 'ขอบคุณครับ', got %q", res[1].Target)
	}
}
