package logger

import "context"

// Logger 是 go-guardian 的框架无关日志契约。
//
// 业务包只依赖这个接口，不直接依赖 Zap。Zap、slog 或其它日志实现
// 通过适配器接入，避免基础库把具体日志框架传递给所有调用方。
type Logger interface {
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	WithContext(ctx context.Context) Logger
	With(args ...interface{}) Logger
	Sync() error
}

// NopLogger 丢弃所有日志，适合测试和可选依赖未配置的场景。
type NopLogger struct{}

func (NopLogger) Debug(...interface{})                 {}
func (NopLogger) Debugf(string, ...interface{})        {}
func (NopLogger) Debugw(string, ...interface{})        {}
func (NopLogger) Info(...interface{})                  {}
func (NopLogger) Infof(string, ...interface{})         {}
func (NopLogger) Infow(string, ...interface{})         {}
func (NopLogger) Warn(...interface{})                  {}
func (NopLogger) Warnf(string, ...interface{})         {}
func (NopLogger) Warnw(string, ...interface{})         {}
func (NopLogger) Error(...interface{})                 {}
func (NopLogger) Errorf(string, ...interface{})        {}
func (NopLogger) Errorw(string, ...interface{})        {}
func (n NopLogger) WithContext(context.Context) Logger { return n }
func (n NopLogger) With(...interface{}) Logger         { return n }
func (NopLogger) Sync() error                          { return nil }
