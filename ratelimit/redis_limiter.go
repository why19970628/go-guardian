package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/token_bucket.lua
var tokenBucketScript string

// RedisTokenBucket Redis 令牌桶限流器（Lua 脚本实现，原子性保证）
type RedisTokenBucket struct {
	client    *redis.Client
	scriptSHA string
	mu        sync.Mutex
}

// NewRedisTokenBucket 创建 Redis 令牌桶限流器
func NewRedisTokenBucket(ctx context.Context, client *redis.Client) (*RedisTokenBucket, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	// 预加载 Lua 脚本
	scriptSHA, err := client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load token bucket script: %w", err)
	}

	return &RedisTokenBucket{
		client:    client,
		scriptSHA: scriptSHA,
	}, nil
}

// TokenBucketConfig 令牌桶配置
type TokenBucketConfig struct {
	Capacity  int64 // 桶容量（最大令牌数）
	Rate      int64 // 令牌生成速率（每秒）
	Requested int64 // 请求消耗令牌数（通常为 1）
}

// Allow 判断是否允许请求（令牌桶算法）
// key: 限流器唯一标识（例如：user:123:api_calls）
// config: 令牌桶配置
func (rtb *RedisTokenBucket) Allow(ctx context.Context, key string, config TokenBucketConfig) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("rate limit key is empty")
	}
	if config.Capacity <= 0 {
		return false, fmt.Errorf("rate limit capacity must be positive")
	}
	if config.Rate <= 0 {
		return false, fmt.Errorf("rate limit rate must be positive")
	}
	if config.Requested <= 0 {
		config.Requested = 1
	}

	// 执行 Lua 脚本
	result, err := rtb.client.EvalSha(
		ctx,
		rtb.scriptSHA,
		[]string{key},
		config.Requested,
		config.Rate,
		config.Capacity,
	).Int()

	if err != nil {
		// 脚本未加载，重新加载
		if err.Error() == "NOSCRIPT No matching script. Please use EVAL." {
			rtb.mu.Lock()
			scriptSHA, loadErr := rtb.client.ScriptLoad(ctx, tokenBucketScript).Result()
			if loadErr != nil {
				rtb.mu.Unlock()
				return false, fmt.Errorf("failed to reload token bucket script: %w", loadErr)
			}
			rtb.scriptSHA = scriptSHA
			rtb.mu.Unlock()

			// 重试执行
			result, err = rtb.client.EvalSha(
				ctx,
				rtb.scriptSHA,
				[]string{key},
				config.Requested,
				config.Rate,
				config.Capacity,
			).Int()
			if err != nil {
				return false, fmt.Errorf("rate limit failed: %w", err)
			}
		} else {
			return false, fmt.Errorf("rate limit failed: %w", err)
		}
	}

	return result == 1, nil
}

// AllowN 判断是否允许请求 N 个令牌
func (rtb *RedisTokenBucket) AllowN(ctx context.Context, key string, capacity, rate, requested int64) (bool, error) {
	return rtb.Allow(ctx, key, TokenBucketConfig{
		Capacity:  capacity,
		Rate:      rate,
		Requested: requested,
	})
}
