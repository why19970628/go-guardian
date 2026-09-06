// Package observability provides Eino framework integration for automatic LLM metrics collection.
//
// Hooks into Eino ChatModel lifecycle to automatically emit:
//   - Token usage (prompt/completion/total)
//   - Call duration and error rates
//   - Prometheus metrics (QPS, latency, token throughput)
//
// Part of go-guardian extensions (optional, Eino-specific).
package observability

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	einoUtilsCallbacks "github.com/cloudwego/eino/utils/callbacks"
	"github.com/why19970628/go-guardian/metrics"
	"go.uber.org/zap"
)

// startTimeKey 用于在 OnStart -> OnEnd 之间传递调用开始时间。
type startTimeKey struct{}

// ttftKey 用于在流式场景记录首 token 时间
type ttftKey struct{}

// Register 注册全局模型观测 handler。
// 必须在任何 Agent/Graph 运行之前调用一次（进程启动阶段）。
func Register(logger *zap.SugaredLogger) {
	callbacks.AppendGlobalHandlers(New(logger))
}

// New 构建模型观测 handler，监听 ChatModel 的非流式与流式调用。
// 注意：TTFT（首 token 延迟）需要在应用层 StreamReader 包装器中记录。
func New(logger *zap.SugaredLogger) callbacks.Handler {
	return einoUtilsCallbacks.NewHandlerHelper().
		ChatModel(&einoUtilsCallbacks.ModelCallbackHandler{
			// 非流式调用
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
				return context.WithValue(ctx, startTimeKey{}, time.Now())
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
				costMs := elapsedMs(ctx)

				var promptTokens, completionTokens, totalTokens int
				if output.TokenUsage != nil {
					promptTokens = output.TokenUsage.PromptTokens
					completionTokens = output.TokenUsage.CompletionTokens
					totalTokens = output.TokenUsage.TotalTokens
				}

				contentLen := 0
				if output.Message != nil {
					contentLen = len(output.Message.Content)
				}

				logger.Infow("【Trace:Model】模型调用完成",
					"node", info.Name,
					"cost_ms", costMs,
					"prompt_tokens", promptTokens,
					"completion_tokens", completionTokens,
					"total_tokens", totalTokens,
					"content_len", contentLen,
				)

				// 上报 Prometheus 指标（非流式无 TTFT）
				metrics.RecordLLMCall(
					"unknown", // model 名暂无法从 callbacks 获取，可从 ctx 传递
					info.Name,
					time.Duration(costMs)*time.Millisecond,
					promptTokens,
					completionTokens,
					totalTokens,
					nil,
				)

				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				costMs := elapsedMs(ctx)
				logger.Errorw("【Trace:Model】模型调用失败",
					"node", info.Name,
					"cost_ms", costMs,
					"error", err,
				)

				// 上报错误指标
				metrics.RecordLLMCall(
					"unknown",
					info.Name,
					time.Duration(costMs)*time.Millisecond,
					0, 0, 0,
					err,
				)
				metrics.RecordError("llm_error", info.Name)

				return ctx
			},
		}).
		Handler()
}

// elapsedMs 从 ctx 中取出开始时间并计算耗时（毫秒）。
func elapsedMs(ctx context.Context) int64 {
	start, ok := ctx.Value(startTimeKey{}).(time.Time)
	if !ok || start.IsZero() {
		return 0
	}
	return time.Since(start).Milliseconds()
}

// getTTFT 从 ctx 中取出首 token 时间并计算 TTFT
func getTTFT(ctx context.Context) time.Duration {
	ttftTime, ok := ctx.Value(ttftKey{}).(time.Time)
	if !ok || ttftTime.IsZero() {
		return 0
	}
	start, ok := ctx.Value(startTimeKey{}).(time.Time)
	if !ok || start.IsZero() {
		return 0
	}
	return ttftTime.Sub(start)
}

// RecordFirstToken 记录首 token 到达时间（由流式 reader 包装层调用）
func RecordFirstToken(ctx context.Context) context.Context {
	// 只记录第一次
	if ttftTime, ok := ctx.Value(ttftKey{}).(time.Time); ok && ttftTime.IsZero() {
		return context.WithValue(ctx, ttftKey{}, time.Now())
	}
	return ctx
}