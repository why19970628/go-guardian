package logger

import (
	"context"

	"github.com/why19970628/go-guardian/trace"
	"go.uber.org/zap"
)

// ZapLogger 将 zap.SugaredLogger 适配为框架无关 Logger。
type ZapLogger struct{ logger *zap.SugaredLogger }

// NewZap 创建 Zap Logger 适配器。
func NewZap(value *zap.SugaredLogger) Logger {
	if value == nil {
		return NopLogger{}
	}
	return &ZapLogger{logger: value}
}

func (l *ZapLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).logger.Debugw(msg, args...)
}
func (l *ZapLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Infow(msg, args...)
}
func (l *ZapLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Warnw(msg, args...)
}
func (l *ZapLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Errorw(msg, args...)
}
func (l *ZapLogger) Sync() error { return l.logger.Sync() }
func (l *ZapLogger) With(args ...interface{}) Logger {
	return &ZapLogger{logger: l.logger.With(args...)}
}
func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	return &ZapLogger{logger: l.logger.With("trace_id", trace.GetTraceID(ctx))}
}

var _ Logger = (*ZapLogger)(nil)
