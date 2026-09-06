package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	limiter := NewTokenBucket(10, 10)
	ctx := context.Background()
	
	// First 10 requests should succeed
	for i := 0; i < 10; i++ {
		if err := limiter.Allow(ctx); err != nil {
			t.Errorf("Request %d should succeed, got error: %v", i, err)
		}
	}
	
	// 11th request should fail (no burst capacity left)
	if err := limiter.Allow(ctx); err == nil {
		t.Error("Request 11 should fail due to rate limit")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	limiter := NewTokenBucket(100, 10) // 100 QPS = 1 token per 10ms
	ctx := context.Background()
	
	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		limiter.Allow(ctx)
	}
	
	// Should fail immediately
	if err := limiter.Allow(ctx); err == nil {
		t.Error("Should fail when bucket empty")
	}
	
	// Wait for refill
	time.Sleep(20 * time.Millisecond)
	
	// Should succeed after refill
	if err := limiter.Allow(ctx); err != nil {
		t.Errorf("Should succeed after refill, got error: %v", err)
	}
}
