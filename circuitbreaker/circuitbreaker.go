// Package circuitbreaker 熔断器 — 快速失败保护下游
package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State 熔断器状态
type State int

const (
	StateClosed   State = iota // 关闭（正常通行）
	StateOpen                  // 打开（拒绝请求）
	StateHalfOpen              // 半开（探测恢复）
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Breaker 熔断器接口
type Breaker interface {
	// Call 执行受保护的调用
	Call(ctx context.Context, fn func() error) error
	// State 当前状态
	State() State
}

// Config 熔断器配置
type Config struct {
	// Threshold 错误阈值（连续失败 N 次触发熔断）
	Threshold int
	// Timeout 熔断超时（打开后多久进入半开状态，尝试恢复）
	Timeout time.Duration
	// HalfOpenMaxCalls 半开状态允许的最大探测请求数
	HalfOpenMaxCalls int
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		Threshold:        5,              // 连续失败 5 次熔断
		Timeout:          30 * time.Second, // 30s 后尝试恢复
		HalfOpenMaxCalls: 3,              // 半开时允许 3 个探测请求
	}
}

// CircuitBreaker 熔断器实现
type CircuitBreaker struct {
	cfg          Config
	state        State
	failures     int       // 连续失败计数
	lastFailTime time.Time // 上次失败时间
	halfOpenReqs int       // 半开状态请求计数
	mu           sync.RWMutex
}

// New 创建熔断器
func New(cfg Config) *CircuitBreaker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxCalls <= 0 {
		cfg.HalfOpenMaxCalls = 3
	}
	return &CircuitBreaker{
		cfg:   cfg,
		state: StateClosed,
	}
}

// Call 执行受保护的函数
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("熔断器打开，拒绝请求")
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// allowRequest 检查是否允许请求通过
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否到达恢复时间
		if time.Since(cb.lastFailTime) > cb.cfg.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenReqs = 0
			return true
		}
		return false
	case StateHalfOpen:
		// 半开状态限制探测请求数
		return cb.halfOpenReqs < cb.cfg.HalfOpenMaxCalls
	default:
		return false
	}
}

// recordResult 记录调用结果
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		switch cb.state {
		case StateClosed:
			// 达到阈值，打开熔断器
			if cb.failures >= cb.cfg.Threshold {
				cb.state = StateOpen
			}
		case StateHalfOpen:
			// 半开状态失败，重新打开
			cb.state = StateOpen
			cb.halfOpenReqs = 0
		}
	} else {
		// 成功
		switch cb.state {
		case StateClosed:
			cb.failures = 0 // 重置失败计数
		case StateHalfOpen:
			cb.halfOpenReqs++
			// 半开状态成功达到探测数，恢复关闭
			if cb.halfOpenReqs >= cb.cfg.HalfOpenMaxCalls {
				cb.state = StateClosed
				cb.failures = 0
				cb.halfOpenReqs = 0
			}
		}
	}
}

// State 返回当前状态
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// NoopBreaker 空熔断器，不熔断
type NoopBreaker struct{}

func (NoopBreaker) Call(_ context.Context, fn func() error) error { return fn() }
func (NoopBreaker) State() State                                  { return StateClosed }
