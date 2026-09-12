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

func (l *ZapLogger) Debug(args ...interface{})                   { l.logger.Debug(args...) }
func (l *ZapLogger) Debugf(template string, args ...interface{}) { l.logger.Debugf(template, args...) }
func (l *ZapLogger) Debugw(msg string, args ...interface{})      { l.logger.Debugw(msg, args...) }
func (l *ZapLogger) Info(args ...interface{})                    { l.logger.Info(args...) }
func (l *ZapLogger) Infof(template string, args ...interface{})  { l.logger.Infof(template, args...) }
func (l *ZapLogger) Infow(msg string, args ...interface{})       { l.logger.Infow(msg, args...) }
func (l *ZapLogger) Warn(args ...interface{})                    { l.logger.Warn(args...) }
func (l *ZapLogger) Warnf(template string, args ...interface{})  { l.logger.Warnf(template, args...) }
func (l *ZapLogger) Warnw(msg string, args ...interface{})       { l.logger.Warnw(msg, args...) }
func (l *ZapLogger) Error(args ...interface{})                   { l.logger.Error(args...) }
func (l *ZapLogger) Errorf(template string, args ...interface{}) { l.logger.Errorf(template, args...) }
func (l *ZapLogger) Errorw(msg string, args ...interface{})      { l.logger.Errorw(msg, args...) }
func (l *ZapLogger) Sync() error                                 { return l.logger.Sync() }
func (l *ZapLogger) With(args ...interface{}) Logger {
	return &ZapLogger{logger: l.logger.With(args...)}
}
func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	return &ZapLogger{logger: l.logger.With("trace_id", trace.GetTraceID(ctx))}
}

var _ Logger = (*ZapLogger)(nil)
