// Package gin provides Gin middleware adapters for llm-guard components.
package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/go-guardian/ratelimit"
	"github.com/yourusername/go-guardian/trace"
)

// RateLimit returns a Gin middleware that applies rate limiting
func RateLimit(limiter ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := limiter.Allow(c.Request.Context()); err != nil {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Trace returns a Gin middleware that adds trace ID to requests
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get trace ID from header
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = trace.GenerateTraceID()
		}

		// Attach to context
		ctx := trace.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// Set response header
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
