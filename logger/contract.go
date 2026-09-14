package logger

import "context"

// Logger 是 go-guardian 的框架无关日志契约。
//
// 业务包只依赖这个接口，不直接依赖 Zap。Zap、slog 或其它日志实现
// 通过适配器接入，避免基础库把具体日志框架传递给所有调用方。
// 每条日志显式接收 context，以便自动注入 trace_id、span_id 等链路字段。
type Logger interface {
	Debug(ctx context.Context, msg string, args ...interface{})
	Info(ctx context.Context, msg string, args ...interface{})
	Warn(ctx context.Context, msg string, args ...interface{})
	Error(ctx context.Context, msg string, args ...interface{})
	WithContext(ctx context.Context) Logger
	With(args ...interface{}) Logger
	Sync() error
}

// NopLogger 丢弃所有日志，适合测试和可选依赖未配置的场景。
type NopLogger struct{}

func (NopLogger) Debug(context.Context, string, ...interface{}) {}
func (NopLogger) Info(context.Context, string, ...interface{})  {}
func (NopLogger) Warn(context.Context, string, ...interface{})  {}
func (NopLogger) Error(context.Context, string, ...interface{}) {}
func (n NopLogger) WithContext(context.Context) Logger          { return n }
func (n NopLogger) With(...interface{}) Logger                  { return n }
func (NopLogger) Sync() error                                   { return nil }
