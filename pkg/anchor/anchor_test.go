package anchor

import (
	"fmt"
	"strings"
	"testing"
)

func TestAnchorValidResponse(t *testing.T) {
	texts := []string{
		"Hello world",
		"Welcome to the game",
		"Let's defeat the demon king",
	}

	batch := NewAnchoredBatch(texts)
	inputPrompt := batch.FormatInput()

	// Verify input contains anchors
	for _, a := range batch.Anchors {
		if !strings.Contains(inputPrompt, a.Tag()) {
			t.Errorf("input missing anchor %s", a.Tag())
		}
	}

	// Simulate perfect response
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s สวัสดีชาวโลก\n", batch.Anchors[0].Tag()))
	sb.WriteString(fmt.Sprintf("%s ยินดีต้อนรับสู่เกม\n", batch.Anchors[1].Tag()))
	sb.WriteString(fmt.Sprintf("%s ไปปราบจอมมารกันเถอะ\n", batch.Anchors[2].Tag()))

	res := batch.ParseAndVerify(sb.String())
	if !res.Success {
		t.Fatalf("expected success, got error: %s (%s)", res.ErrorType, res.ErrorDetail)
	}

	if len(res.Translations) != 3 {
		t.Fatalf("expected 3 translations, got %d", len(res.Translations))
	}
	if res.Translations[0] != "สวัสดีชาวโลก" {
		t.Errorf("unexpected translation: %s", res.Translations[0])
	}
}

// TestFaultInjection verifies 100% detection rate when LLM misbehaves
func TestFaultInjection(t *testing.T) {
	texts := []string{
		"Line 0",
		"Line 1",
		"Line 2",
		"Line 3",
	}

	batch := NewAnchoredBatch(texts)

	t.Run("Fault: Model drops line 2", func(t *testing.T) {
		response := fmt.Sprintf("%s แปล 0\n%s แปล 1\n%s แปล 3\n",
			batch.Anchors[0].Tag(),
			batch.Anchors[1].Tag(),
			batch.Anchors[3].Tag(),
		)

		res := batch.ParseAndVerify(response)
		if res.Success {
			t.Fatal("expected failure on dropped line, but got success")
		}
		if res.ErrorType != "anchor_missing" {
			t.Errorf("expected error anchor_missing, got %s", res.ErrorType)
		}
		if res.FailedIndex != 2 {
			t.Errorf("expected failed index 2, got %d", res.FailedIndex)
		}
	})

	t.Run("Fault: Model duplicates an anchor", func(t *testing.T) {
		response := fmt.Sprintf("%s แปล 0\n%s แปล 1\n%s แปล 1 ซ้ำ\n%s แปล 2\n%s แปล 3\n",
			batch.Anchors[0].Tag(),
			batch.Anchors[1].Tag(),
			batch.Anchors[1].Tag(),
			batch.Anchors[2].Tag(),
			batch.Anchors[3].Tag(),
		)

		res := batch.ParseAndVerify(response)
		if res.Success {
			t.Fatal("expected failure on duplicate anchor, but got success")
		}
		if res.ErrorType != "duplicate_anchor" {
			t.Errorf("expected duplicate_anchor error, got %s", res.ErrorType)
		}
	})

	t.Run("Fault: Model hallucinated extra line with wrong anchor", func(t *testing.T) {
		response := fmt.Sprintf("%s แปล 0\n%s แปล 1\n%s แปล 2\n%s แปล 3\n[#999:4] หลอนเพิ่มมา\n",
			batch.Anchors[0].Tag(),
			batch.Anchors[1].Tag(),
			batch.Anchors[2].Tag(),
			batch.Anchors[3].Tag(),
		)

		res := batch.ParseAndVerify(response)
		if res.Success {
			t.Fatal("expected failure on count mismatch, but got success")
		}
		if res.ErrorType != "count_mismatch" {
			t.Errorf("expected count_mismatch error, got %s", res.ErrorType)
		}
	})
}

func BenchmarkAnchorParse(b *testing.B) {
	texts := make([]string, 50)
	for i := 0; i < 50; i++ {
		texts[i] = fmt.Sprintf("Source sentence %d in the visual novel script.", i)
	}

	batch := NewAnchoredBatch(texts)
	var sb strings.Builder
	for i, a := range batch.Anchors {
		sb.WriteString(fmt.Sprintf("%s ประโยคคำแปลภาษาไทยลำดับที่ %d\n", a.Tag(), i))
	}
	rawResp := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := batch.ParseAndVerify(rawResp)
		if !res.Success {
			b.Fatal("parse failed")
		}
	}
}
