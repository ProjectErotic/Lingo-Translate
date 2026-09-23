package decision

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// CachedDecision stores an evaluated decision
type CachedDecision struct {
	Result    DecisionResult
	ExpiresAt time.Time
}

// MemoCache provides thread-safe, in-memory deduplication for System One decisions
type MemoCache struct {
	mu       sync.RWMutex
	items    map[string]CachedDecision
	maxItems int
	hits     int64
	misses   int64
}

// NewMemoCache creates a new in-memory decision cache
func NewMemoCache(maxItems int) *MemoCache {
	if maxItems <= 0 {
		maxItems = 20000
	}
	return &MemoCache{
		items:    make(map[string]CachedDecision, 1024),
		maxItems: maxItems,
	}
}

func hashKey(primitive PrimitiveType, query, input string) string {
	sum := sha256.Sum256([]byte(string(primitive) + "|" + strings.TrimSpace(query) + "|" + strings.TrimSpace(input)))
	return hex.EncodeToString(sum[:])
}

// Get checks if a decision has already been memoized
func (c *MemoCache) Get(primitive PrimitiveType, query, input string) (DecisionResult, bool) {
	key := hashKey(primitive, query, input)

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		atomic.AddInt64(&c.misses, 1)
		return DecisionResult{}, false
	}

	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		atomic.AddInt64(&c.misses, 1)
		return DecisionResult{}, false
	}

	atomic.AddInt64(&c.hits, 1)
	res := item.Result
	res.Source = "cache"
	res.Latency = 0
	return res, true
}

// Set saves a decision outcome to the memoization cache
func (c *MemoCache) Set(primitive PrimitiveType, query, input string, res DecisionResult, ttl time.Duration) {
	key := hashKey(primitive, query, input)

	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction if max capacity reached: clear half of map
	if len(c.items) >= c.maxItems {
		count := 0
		for k := range c.items {
			delete(c.items, k)
			count++
			if count > c.maxItems/4 {
				break
			}
		}
	}

	c.items[key] = CachedDecision{
		Result:    res,
		ExpiresAt: exp,
	}
}

// Stats returns hit, miss, and cached item counts
func (c *MemoCache) Stats() (hits int64, misses int64, count int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses), len(c.items)
}

// Clear flushes all cached decisions
func (c *MemoCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]CachedDecision, 1024)
}
