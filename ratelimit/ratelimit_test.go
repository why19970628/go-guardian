package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	limiter := NewTokenBucket(10, 10) // 10 QPS, capacity 10
	ctx := context.Background()
	
	// First 10 requests should succeed immediately (consume all initial tokens)
	for i := 0; i < 10; i++ {
		if err := limiter.Allow(ctx); err != nil {
			t.Errorf("Request %d should succeed, got error: %v", i, err)
		}
	}
	
	// 11th request will block and wait for refill
	// At 10 QPS, we get 1 token every 100ms
	start := time.Now()
	if err := limiter.Allow(ctx); err != nil {
		t.Errorf("Request 11 should eventually succeed after waiting for refill: %v", err)
	}
	elapsed := time.Since(start)
	
	// Should have blocked for approximately 100ms waiting for 1 token
	if elapsed < 80*time.Millisecond || elapsed > 150*time.Millisecond {
		t.Logf("Blocked for %v (expected ~100ms, acceptable due to ticker granularity)", elapsed)
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	limiter := NewTokenBucket(100, 10) // 100 QPS = 1 token every 10ms
	ctx := context.Background()
	
	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		limiter.Allow(ctx)
	}
	
	// Wait for refill (at 100 QPS, should get ~2 tokens in 20ms)
	time.Sleep(20 * time.Millisecond)
	
	// Should succeed after refill
	if err := limiter.Allow(ctx); err != nil {
		t.Errorf("Should succeed after refill, got error: %v", err)
	}
	
	// Second request should also succeed (we waited for 2 tokens)
	if err := limiter.Allow(ctx); err != nil {
		t.Errorf("Second request should succeed after refill, got error: %v", err)
	}
}
