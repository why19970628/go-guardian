// Package ratelimit 限流器 — 保护 LLM 调用不打爆下游
//
// 提供两种实现：
// - TokenBucket：单机内存限流（适合开发 / 单实例部署）
// - RedisLimiter：分布式限流（适合多实例生产环境）
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许通过，阻塞直到有额度或 ctx 取消
	Allow(ctx context.Context) error
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	rate     float64       // 每秒补充速率
	capacity int           // 桶容量
	tokens   float64       // 当前令牌数
	lastTime time.Time     // 上次补充时间
	mu       sync.Mutex
}

// NewTokenBucket 创建令牌桶限流器
// rate: 每秒补充令牌数（如 10 表示 10 QPS）
// capacity: 桶容量，允许短时突发（如 20 表示允许瞬时 20 请求）
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:     rate,
		capacity: capacity,
		tokens:   float64(capacity), // 初始满桶
		lastTime: time.Now(),
	}
}

// Allow 获取 1 个令牌，阻塞等待直到有可用令牌或 ctx 取消
func (tb *TokenBucket) Allow(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("限流等待超时: %w", ctx.Err())
		case <-ticker.C:
			if tb.tryTake() {
				return nil
			}
		}
	}
}

// tryTake 尝试获取 1 个令牌，成功返回 true
func (tb *TokenBucket) tryTake() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.lastTime = now

	// 补充令牌
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	// 尝试消费 1 个令牌
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}
	return false
}

// NoopLimiter 空限流器，不限流
type NoopLimiter struct{}

func (NoopLimiter) Allow(_ context.Context) error { return nil }

// RedisLimiter 基于 Redis 的分布式限流器（滑动窗口算法）
type RedisLimiter struct {
	client   *redis.Client
	key      string        // Redis key 前缀
	rate     int           // 每秒允许请求数
	window   time.Duration // 时间窗口
}

// NewRedisLimiter 创建 Redis 限流器
// key: Redis key 前缀（如 "ratelimit:llm"）
// rate: 每秒允许请求数（如 10 表示 10 QPS）
// window: 时间窗口（通常 1s，支持更细粒度如 100ms）
func NewRedisLimiter(client *redis.Client, key string, rate int, window time.Duration) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		key:    key,
		rate:   rate,
		window: window,
	}
}

// Allow 滑动窗口限流：用 ZSET 存时间戳，清理过期 + 计数 + 添加当前请求
func (rl *RedisLimiter) Allow(ctx context.Context) error {
	now := time.Now()
	windowStart := now.Add(-rl.window)

	pipe := rl.client.Pipeline()
	// 1. 清理过期时间戳
	pipe.ZRemRangeByScore(ctx, rl.key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	// 2. 计数窗口内请求
	zcard := pipe.ZCard(ctx, rl.key)
	// 3. 添加当前时间戳
	pipe.ZAdd(ctx, rl.key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	// 4. 设置过期（窗口 * 2，防止 key 泄漏）
	pipe.Expire(ctx, rl.key, rl.window*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("Redis 限流失败: %w", err)
	}

	count, err := zcard.Result()
	if err != nil {
		return fmt.Errorf("获取限流计数失败: %w", err)
	}

	if int(count) >= rl.rate {
		return fmt.Errorf("限流触发：当前 QPS %d 超过限制 %d", count, rl.rate)
	}

	return nil
}
