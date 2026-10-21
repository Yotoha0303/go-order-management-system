package bizcache

import (
	"testing"
	"time"
)

func TestNullCacheTTLWithJitter(t *testing.T) {
	for i := 0; i < 100; i++ {
		ttl := nullCacheTTLWithJitter()
		if ttl < ProductDetailNullCacheTTL {
			t.Fatalf("expected ttl >= %v, got %v", ProductDetailNullCacheTTL, ttl)
		}
		maxTTL := ProductDetailNullCacheTTL + 15*time.Second
		if ttl > maxTTL {
			t.Fatalf("expected ttl <= %v, got %v", maxTTL, ttl)
		}
	}
}
