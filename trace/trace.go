// Package trace provides distributed tracing utilities for any Go application.
//
// Core functionality (no framework dependencies):
//   - Trace ID generation (32-char hex: timestamp + random)
//   - Context propagation
//   - Caller information capture (file:line)
//
// Framework adapters available in middleware/{gin,grpc,hertz}/
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime"
	"time"
)

// Context keys
type traceIDKey struct{}
type callerKey struct{}

// GenerateTraceID generates a 32-character hex trace ID (timestamp + 16 random bytes)
func GenerateTraceID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	return fmt.Sprintf("%016x%s", timestamp, hex.EncodeToString(randomBytes[:8]))
}

// WithTraceID attaches a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// GetTraceID retrieves the trace ID from the context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// WithCaller attaches caller information to the context
func WithCaller(ctx context.Context, file string, line int) context.Context {
	return context.WithValue(ctx, callerKey{}, fmt.Sprintf("%s:%d", file, line))
}

// GetCaller retrieves caller information from the context
func GetCaller(ctx context.Context) string {
	if caller, ok := ctx.Value(callerKey{}).(string); ok {
		return caller
	}
	return ""
}

// CaptureCaller captures the current caller information (file:line)
// skip: number of stack frames to skip (0 = caller of CaptureCaller, 1 = caller's caller)
func CaptureCaller(skip int) (file string, line int) {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown", 0
	}
	// Trim path to just filename for readability
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			file = file[i+1:]
			break
		}
	}
	return file, line
}
