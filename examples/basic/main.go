package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/why19970628/go-guardian/circuitbreaker"
	"github.com/why19970628/go-guardian/fallback"
	"github.com/why19970628/go-guardian/metrics"
	"github.com/why19970628/go-guardian/ratelimit"
	"github.com/why19970628/go-guardian/trace"
)

func main() {
	// Initialize metrics
	metrics.Init("example-app")

	// Create rate limiter (10 QPS, 20 burst)
	limiter := ratelimit.NewTokenBucket(10.0, 20)

	// Create circuit breaker (open after 5 failures, retry after 10s)
	breaker := circuitbreaker.New(5, 10*time.Second)

	// Create fallback strategy
	cannedFallback := fallback.NewCannedResponse()

	// Setup Gin router
	r := gin.Default()

	// Register trace middleware
	r.Use(trace.GinMiddleware())

	// Expose /metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Example chat endpoint with layered protection
	r.POST("/chat", func(c *gin.Context) {
		ctx := c.Request.Context()
		traceID := trace.GetTraceID(ctx)

		// 1. Rate limiting
		if err := limiter.Allow(ctx); err != nil {
			metrics.RecordRateLimitReject("ingress")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":    "Rate limit exceeded",
				"trace_id": traceID,
			})
			return
		}

		// 2. Track request start
		start := time.Now()
		defer func() {
			metrics.TrackRequestEnd(ctx, start, "chat")
		}()
		metrics.TrackRequestStart(ctx, "chat")

		var req struct {
			Query string `json:"query"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// 3. Call LLM with circuit breaker
		var response string
		err := breaker.Call(func() error {
			// Simulate LLM call
			llmResp, llmErr := callLLM(ctx, req.Query)
			response = llmResp
			return llmErr
		})

		// 4. Fallback on error
		if err != nil {
			if err == circuitbreaker.ErrCircuitOpen {
				log.Printf("[%s] Circuit breaker open, using fallback", traceID)
			} else {
				log.Printf("[%s] LLM error: %v, using fallback", traceID, err)
			}

			fallbackResp, _ := cannedFallback.Execute(ctx, &fallback.Request{
				UserQuery: req.Query,
				LastError: err,
			})
			response = fallbackResp.Content
		}

		c.JSON(http.StatusOK, gin.H{
			"response": response,
			"trace_id": traceID,
		})
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":           "ok",
			"circuit_breaker":  breaker.State().String(),
			"rate_limiter":     "active",
			"metrics_endpoint": "/metrics",
		})
	})

	// Start server
	fmt.Println("Server listening on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /chat       - Chat with LLM (rate limited, circuit breaker, fallback)")
	fmt.Println("  GET  /health     - Health check")
	fmt.Println("  GET  /metrics    - Prometheus metrics")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// callLLM simulates an LLM API call
func callLLM(ctx context.Context, query string) (string, error) {
	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Simulate 20% failure rate
	if time.Now().UnixNano()%5 == 0 {
		return "", fmt.Errorf("LLM service unavailable")
	}

	// Record LLM metrics
	metrics.RecordLLMCall(
		"gpt-4",
		"chat_node",
		100*time.Millisecond,
		150, // prompt tokens
		80,  // completion tokens
		230, // total tokens
		nil,
	)

	return fmt.Sprintf("AI response to: %s", query), nil
}
