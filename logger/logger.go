// Package logger provides structured logging with automatic trace context injection.
//
// Built on Zap, with:
//   - Automatic trace_id and caller (file:line) injection
//   - Log rotation (via lumberjack)
//   - Configurable levels and outputs
//
// Part of go-guardian core packages (framework-agnostic).
package logger

import (
	"context"
	"os"

	"github.com/why19970628/go-guardian/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level      string `yaml:"level"`       // 日志级别: debug, info, warn, error
	Filename   string `yaml:"filename"`    // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 单个文件最大大小 (MB)
	MaxBackups int    `yaml:"max_backups"` // 保留旧文件最大个数
	MaxAge     int    `yaml:"max_age"`     // 保留旧文件最大天数
	Compress   bool   `yaml:"compress"`    // 是否压缩旧文件
}

// Init 初始化日志
func Init(cfg *Config) {
	// 解析日志级别
	level := zapcore.InfoLevel
	if cfg != nil && cfg.Level != "" {
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			level = zapcore.InfoLevel
		}
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 输出配置
	var writers []zapcore.WriteSyncer
	writers = append(writers, zapcore.AddSync(os.Stdout))

	if cfg != nil && cfg.Filename != "" {
		writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}))
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	globalLogger = logger.Sugar()
}

// WithContext 从 context 中提取 trace_id 和 caller，返回带上下文的 logger
func WithContext(ctx context.Context) *zap.SugaredLogger {
	if globalLogger == nil {
		Init(nil) // 使用默认配置
	}

	fields := []interface{}{}

	// 注入 trace_id
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}

	// 注入 caller (file:line)
	if caller := trace.GetCaller(ctx); caller != "" {
		fields = append(fields, "caller", caller)
	}

	return globalLogger.With(fields...)
}

// Debug 快捷方法
func Debug(args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Debugf(template, args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Debugw(msg, keysAndValues...)
}

// Info 快捷方法
func Info(args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Infof(template, args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Infow(msg, keysAndValues...)
}

// Warn 快捷方法
func Warn(args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Warnf(template, args...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Warnw(msg, keysAndValues...)
}

// Error 快捷方法
func Error(args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Errorf(template, args...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Errorw(msg, keysAndValues...)
}

// Fatal 快捷方法
func Fatal(args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Fatalf(template, args...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	if globalLogger == nil {
		Init(nil)
	}
	globalLogger.Fatalw(msg, keysAndValues...)
}
