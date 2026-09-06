// Package metrics 性能指标采集 — 基于 Prometheus
//
// 覆盖：
// - QPS / 并发数
// - 错误率（按错误类型分组）
// - 延迟分位数（P50/P95/P99）
// - Tool 调用耗时（按 tool 名称分组）
// - 端到端对话耗时
// - LLM token 用量统计
package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal 请求总数（按 agent_id / status 分组）
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_mechanic_requests_total",
			Help: "Total number of requests",
		},
		[]string{"agent_id", "status"}, // status: success / error / ratelimit / circuit_open
	)

	// RequestDuration 请求延迟直方图（端到端）
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60}, // 100ms 到 60s
		},
		[]string{"agent_id"},
	)

	// ConcurrentRequests 当前并发请求数
	ConcurrentRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_mechanic_concurrent_requests",
			Help: "Current number of concurrent requests",
		},
		[]string{"agent_id"},
	)

	// LLMCallsTotal LLM 调用次数（按 model / status 分组）
	LLMCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_mechanic_llm_calls_total",
			Help: "Total number of LLM calls",
		},
		[]string{"model", "node", "status"}, // node: agent 节点名
	)

	// LLMCallDuration LLM 调用延迟
	LLMCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_llm_call_duration_seconds",
			Help:    "LLM call duration in seconds",
			Buckets: []float64{.05, .1, .5, 1, 2, 5, 10},
		},
		[]string{"model", "node"},
	)

	// LLMTokensUsed LLM token 用量
	LLMTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_mechanic_llm_tokens_total",
			Help: "Total LLM tokens used",
		},
		[]string{"model", "node", "type"}, // type: prompt / completion / total
	)

	// LLMTimeToFirstToken 首 token 延迟（TTFT - Time To First Token）
	LLMTimeToFirstToken = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_llm_ttft_seconds",
			Help:    "Time to first token in seconds",
			Buckets: []float64{.01, .05, .1, .2, .5, 1, 2},
		},
		[]string{"model", "node"},
	)

	// LLMTokensPerSecond LLM 生成吞吐量（tokens/s）
	LLMTokensPerSecond = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_llm_tokens_per_second",
			Help:    "LLM generation throughput in tokens per second",
			Buckets: []float64{10, 20, 50, 100, 200, 500, 1000},
		},
		[]string{"model", "node"},
	)

	// LLMLatencyPerToken 每 token 平均延迟（ms/token）
	LLMLatencyPerToken = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_llm_latency_per_token_ms",
			Help:    "Average latency per token in milliseconds",
			Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
		},
		[]string{"model", "node"},
	)

	// LLMPromptLength prompt 长度分布
	LLMPromptLength = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_llm_prompt_length_tokens",
			Help:    "Distribution of prompt lengths in tokens",
			Buckets: []float64{100, 500, 1000, 2000, 4000, 8000, 16000},
		},
		[]string{"model", "node"},
	)

	// LLMCompletionLength completion 长度分布
	LLMCompletionLength = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_llm_completion_length_tokens",
			Help:    "Distribution of completion lengths in tokens",
			Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000},
		},
		[]string{"model", "node"},
	)

	// ToolCallsTotal Tool 调用次数
	ToolCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_mechanic_tool_calls_total",
			Help: "Total number of tool calls",
		},
		[]string{"tool_name", "status"},
	)

	// ToolCallDuration Tool 调用延迟
	ToolCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_mechanic_tool_call_duration_seconds",
			Help:    "Tool call duration in seconds",
			Buckets: []float64{.01, .05, .1, .5, 1, 2, 5},
		},
		[]string{"tool_name"},
	)

	// ErrorsTotal 错误计数（按 error_type 分组）
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_mechanic_errors_total",
			Help: "Total number of errors",
		},
		[]string{"error_type", "agent_id"}, // error_type: timeout / llm_error / tool_error / circuit_open
	)

	// CircuitBreakerState 熔断器状态（0=closed, 1=open, 2=half_open）
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_mechanic_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half_open)",
		},
		[]string{"service"}, // service: llm / vllm / vision / search
	)

	// RateLimitRejects 限流拒绝次数
	RateLimitRejects = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_mechanic_ratelimit_rejects_total",
			Help: "Total number of rate limit rejects",
		},
		[]string{"limiter"}, // limiter: llm / global
	)
)

// RecordRequest 记录请求（端到端）
func RecordRequest(agentID string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	RequestsTotal.WithLabelValues(agentID, status).Inc()
	RequestDuration.WithLabelValues(agentID).Observe(duration.Seconds())
}

// LLMMetrics LLM 调用的详细指标
type LLMMetrics struct {
	Model            string
	Node             string
	Duration         time.Duration
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	TimeToFirstToken time.Duration // 首 token 延迟（流式场景）
	Error            error
}

// RecordLLMCall 记录 LLM 调用（兼容旧签名）
func RecordLLMCall(model, node string, duration time.Duration, promptTokens, completionTokens, totalTokens int, err error) {
	RecordLLMMetrics(&LLMMetrics{
		Model:            model,
		Node:             node,
		Duration:         duration,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Error:            err,
	})
}

// RecordLLMMetrics 记录 LLM 调用的完整指标
func RecordLLMMetrics(m *LLMMetrics) {
	status := "success"
	if m.Error != nil {
		status = "error"
	}
	LLMCallsTotal.WithLabelValues(m.Model, m.Node, status).Inc()
	LLMCallDuration.WithLabelValues(m.Model, m.Node).Observe(m.Duration.Seconds())

	if m.Error == nil {
		// Token 用量
		LLMTokensUsed.WithLabelValues(m.Model, m.Node, "prompt").Add(float64(m.PromptTokens))
		LLMTokensUsed.WithLabelValues(m.Model, m.Node, "completion").Add(float64(m.CompletionTokens))
		LLMTokensUsed.WithLabelValues(m.Model, m.Node, "total").Add(float64(m.TotalTokens))

		// Prompt 和 Completion 长度分布
		LLMPromptLength.WithLabelValues(m.Model, m.Node).Observe(float64(m.PromptTokens))
		LLMCompletionLength.WithLabelValues(m.Model, m.Node).Observe(float64(m.CompletionTokens))

		// 首 token 延迟（仅流式场景有效）
		if m.TimeToFirstToken > 0 {
			LLMTimeToFirstToken.WithLabelValues(m.Model, m.Node).Observe(m.TimeToFirstToken.Seconds())
		}

		// 吞吐量（tokens/s）与每 token 延迟（ms/token）
		if m.CompletionTokens > 0 && m.Duration > 0 {
			tokensPerSecond := float64(m.CompletionTokens) / m.Duration.Seconds()
			LLMTokensPerSecond.WithLabelValues(m.Model, m.Node).Observe(tokensPerSecond)

			latencyPerToken := m.Duration.Milliseconds() / int64(m.CompletionTokens)
			LLMLatencyPerToken.WithLabelValues(m.Model, m.Node).Observe(float64(latencyPerToken))
		}
	}
}

// RecordToolCall 记录 Tool 调用
func RecordToolCall(toolName string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	ToolCallsTotal.WithLabelValues(toolName, status).Inc()
	ToolCallDuration.WithLabelValues(toolName).Observe(duration.Seconds())
}

// RecordError 记录错误
func RecordError(errorType, agentID string) {
	ErrorsTotal.WithLabelValues(errorType, agentID).Inc()
}

// TrackConcurrency 跟踪并发数（defer 调用 done）
func TrackConcurrency(agentID string) (done func()) {
	ConcurrentRequests.WithLabelValues(agentID).Inc()
	return func() {
		ConcurrentRequests.WithLabelValues(agentID).Dec()
	}
}

// UpdateCircuitBreakerState 更新熔断器状态
func UpdateCircuitBreakerState(service string, state int) {
	CircuitBreakerState.WithLabelValues(service).Set(float64(state))
}

// RecordRateLimitReject 记录限流拒绝
func RecordRateLimitReject(limiter string) {
	RateLimitRejects.WithLabelValues(limiter).Inc()
}

// ctxKey 用于在 context 中传递开始时间
type ctxKey string

const startTimeKey ctxKey = "metrics_start_time"

// ContextWithStartTime 在 context 中存储开始时间
func ContextWithStartTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, startTimeKey, time.Now())
}

// ElapsedFrom 从 context 中取出开始时间并计算耗时
func ElapsedFrom(ctx context.Context) time.Duration {
	start, ok := ctx.Value(startTimeKey).(time.Time)
	if !ok || start.IsZero() {
		return 0
	}
	return time.Since(start)
}
