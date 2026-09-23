package decision

import (
	"sync"
	"testing"
	"time"
)

func TestMemoCache(t *testing.T) {
	cache := NewMemoCache(100)

	// 1. Initial Miss
	_, hit := cache.Get(PrimitiveNoul, "is_code", "Hello world")
	if hit {
		t.Fatalf("expected cache miss, got hit")
	}

	// 2. Set and Hit
	res := DecisionResult{
		Primitive:  PrimitiveNoul,
		BoolValue:  true,
		Confidence: 0.95,
		Source:     "jev",
	}
	cache.Set(PrimitiveNoul, "is_code", "Hello world", res, 1*time.Hour)

	cached, hit := cache.Get(PrimitiveNoul, "is_code", "Hello world")
	if !hit {
		t.Fatalf("expected cache hit, got miss")
	}
	if !cached.BoolValue || cached.Confidence != 0.95 || cached.Source != "cache" {
		t.Errorf("unexpected cached result: %+v", cached)
	}

	// 3. Stats verification
	hits, misses, count := cache.Stats()
	if hits != 1 || misses != 1 || count != 1 {
		t.Errorf("unexpected stats: hits=%d, misses=%d, count=%d", hits, misses, count)
	}

	// 4. Expiration
	cache.Set(PrimitiveChoice, "category", "Sword", DecisionResult{Choice: "item"}, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	_, hit = cache.Get(PrimitiveChoice, "category", "Sword")
	if hit {
		t.Fatalf("expected expired item to miss")
	}

	// 5. Concurrent Access (Race Detector)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.Set(PrimitiveNoul, "concurrent_test", "text", DecisionResult{BoolValue: true}, 0)
			_, _ = cache.Get(PrimitiveNoul, "concurrent_test", "text")
		}(i)
	}
	wg.Wait()

	// 6. Clear
	cache.Clear()
	_, _, count = cache.Stats()
	if count != 0 {
		t.Errorf("expected 0 items after clear, got %d", count)
	}
}
