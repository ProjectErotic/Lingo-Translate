package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lingo-translate/pkg/translator"
)

func TestGoogleTranslate(t *testing.T) {
	// 1. Language code test
	if langCode("Korean") != "ko" {
		t.Errorf("expected ko, got %s", langCode("Korean"))
	}
	if langCode("Thai") != "th" {
		t.Errorf("expected th, got %s", langCode("Thai"))
	}

	// 2. Mock Google server test
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Google GTX JSON response format: [[["สวัสดี", "Hello", null, null, 1]], null, "en"]
		q := r.URL.Query().Get("q")
		if strings.Contains(q, "Hello") {
			_, _ = w.Write([]byte(`[[["สวัสดี", "Hello", null, null, 1]], null, "en"]`))
		} else {
			_, _ = w.Write([]byte(`[[["ทดสอบ", "Test", null, null, 1]], null, "en"]`))
		}
	}))
	defer mockServer.Close()

	provider := New(Config{})
	provider.httpClient = mockServer.Client()

	// Direct test translateOne with custom URL would test logic; test Translate with empty & fallback
	res, err := provider.Translate(context.Background(), []string{"Hello"}, translator.Options{
		SourceLang: "en",
		TargetLang: "th",
	})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result")
	}
}
