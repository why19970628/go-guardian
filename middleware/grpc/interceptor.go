// Package grpc provides gRPC interceptor adapters for llm-guard components.
package grpc

import (
	"context"

	"github.com/yourusername/go-guardian/ratelimit"
	"github.com/yourusername/go-guardian/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryRateLimit returns a gRPC unary server interceptor that applies rate limiting
func UnaryRateLimit(limiter ratelimit.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := limiter.Allow(ctx); err != nil {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

// UnaryTrace returns a gRPC unary server interceptor that adds trace ID
func UnaryTrace() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Try to get trace ID from incoming metadata
		var traceID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-trace-id"); len(ids) > 0 {
				traceID = ids[0]
			}
		}
		if traceID == "" {
			traceID = trace.GenerateTraceID()
		}

		// Attach to context
		ctx = trace.WithTraceID(ctx, traceID)

		// Set outgoing metadata
		grpc.SetHeader(ctx, metadata.Pairs("x-trace-id", traceID))

		return handler(ctx, req)
	}
}

// StreamTrace returns a gRPC stream server interceptor that adds trace ID
func StreamTrace() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// Try to get trace ID from incoming metadata
		var traceID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-trace-id"); len(ids) > 0 {
				traceID = ids[0]
			}
		}
		if traceID == "" {
			traceID = trace.GenerateTraceID()
		}

		// Attach to context
		ctx = trace.WithTraceID(ctx, traceID)

		// Set outgoing metadata
		ss.SetHeader(metadata.Pairs("x-trace-id", traceID))

		// Wrap stream with new context
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
