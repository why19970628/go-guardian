package logger

import (
	"context"

	"github.com/why19970628/go-guardian/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
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
	l.zapWithContext(ctx).Debugw(msg, args...)
}
func (l *ZapLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.zapWithContext(ctx).Infow(msg, args...)
}
func (l *ZapLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.zapWithContext(ctx).Warnw(msg, args...)
}
func (l *ZapLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.zapWithContext(ctx).Errorw(msg, args...)
}
func (l *ZapLogger) Sync() error { return l.logger.Sync() }
func (l *ZapLogger) With(args ...interface{}) Logger {
	return &ZapLogger{logger: l.logger.With(args...)}
}
func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	return &ZapLogger{logger: l.zapWithContext(ctx)}
}

func (l *ZapLogger) zapWithContext(ctx context.Context) *zap.SugaredLogger {
	fields := []interface{}{}
	if ctx == nil {
		return l.logger
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}
	spanContext := oteltrace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		fields = append(fields, "trace_id", spanContext.TraceID().String(), "span_id", spanContext.SpanID().String())
	}
	return l.logger.With(fields...)
}

var _ Logger = (*ZapLogger)(nil)
