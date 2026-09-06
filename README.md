# Go Guardian

[English](#) | [中文](#中文文档)

Production-ready middleware toolkit for any Go application. Provides rate limiting, circuit breaking, fallback strategies, distributed tracing, and comprehensive observability. Works with **HTTP (Gin, Hertz, net/http)**, **gRPC**, and any Go service.

**Use Cases**: Microservices, API gateways, LLM applications, distributed systems, or any Go backend requiring reliability and observability.

## Overview

`llm-guard` is a collection of reusable middleware packages designed to enhance LLM applications with production-grade reliability, observability, and performance management capabilities. Framework-agnostic design works with Gin, Hertz, standard `net/http`, and any Go web framework.

### Features

**Core Middleware (Framework-Agnostic)**
- **Rate Limiting**: Token bucket (in-memory) and Redis-based distributed rate limiting
- **Circuit Breaker**: Three-state circuit breaker (Closed → Open → Half-Open) with automatic recovery
- **Distributed Tracing**: Automatic trace ID generation, propagation, and context correlation
- **Prometheus Metrics**: Complete observability (QPS, latency percentiles, error rates, concurrency)
- **Structured Logging**: Zap-based logger with automatic trace context injection

**Framework Adapters**
- **HTTP**: Gin, Hertz, `net/http` middleware
- **gRPC**: Unary and stream interceptors
- **Coming soon**: Kratos, go-zero, Hertz

**Domain-Specific (Optional)**
- **LLM Applications**: TTFT, token throughput, prompt/completion metrics, Eino callbacks
- **Database**: Query performance tracking, connection pool metrics
- **Cache**: Hit rate, latency tracking

### Why Go Guardian?

A universal Go middleware toolkit designed for **any Go application**:
- **Protocol-agnostic**: Core logic works with HTTP, gRPC, or direct function calls
- **Framework adapters**: Pre-built adapters for Gin, Hertz, gRPC, Kratos, and more
- **Domain-specific extensions**: Generic + specialized (e.g., LLM metrics with TTFT/tokens/s)
- **Production-tested**: Battle-tested patterns from high-scale distributed systems
- **Zero lock-in**: Use individual packages independently
- **Minimal dependencies**: Each package has its own `go.mod`

## Project Structure

```
go-guardian/
├── ratelimit/              # Rate limiting (core logic)
│   ├── token_bucket.go     # In-memory token bucket
│   ├── redis.go            # Redis distributed limiter
│   └── go.mod
├── circuitbreaker/         # Circuit breaker (core logic)
│   ├── breaker.go          # Three-state circuit breaker
│   └── go.mod

├── trace/                  # Distributed tracing (core logic)
│   ├── trace.go            # Trace ID generation & propagation
│   ├── context.go          # Context helpers
│   └── go.mod
├── middleware/             # Framework adapters
│   ├── gin/                # Gin middleware
│   │   ├── ratelimit.go
│   │   ├── trace.go
│   │   └── go.mod
│   ├── grpc/               # gRPC interceptors
│   │   ├── ratelimit.go
│   │   ├── trace.go
│   │   └── go.mod
│   └── hertz/              # Hertz middleware
│       └── go.mod
├── metrics/                # Prometheus metrics (generic)
│   ├── http.go             # HTTP metrics (QPS, latency, status codes)
│   ├── grpc.go             # gRPC metrics (calls, latency, codes)
│   ├── base.go             # Common metrics (errors, concurrency)
│   └── go.mod
├── logger/                 # Structured logging
│   ├── logger.go           # Zap wrapper with trace context
│   └── go.mod
├── extensions/             # Domain-specific extensions (optional)
│   ├── llm/                # LLM-specific (TTFT, fallback, etc.)
│   │   ├── metrics.go
│   │   ├── fallback.go
│   │   └── go.mod
│   ├── eino/               # Eino framework integration
│   │   ├── callbacks.go
│   │   └── go.mod
│   └── database/           # Database metrics (coming soon)
│       └── go.mod
└── examples/               # Usage examples
    ├── http-gin/           # Gin microservice
    ├── grpc/               # gRPC service
    ├── llm-app/            # LLM application with Eino
    └── generic/            # Pure Go service (no framework)
```

### Design Philosophy

- **Core packages** (ratelimit, circuitbreaker, trace, metrics, logger): Pure logic, no framework dependencies, works anywhere
- **Middleware packages** (middleware/*): Framework-specific adapters (Gin, gRPC, Hertz)
- **Extension packages** (extensions/*): Domain-specific features (LLM metrics, Eino callbacks, database tracking)

**Dependency Direction**: `middleware → core`, `extensions → core`. Core packages have zero dependency on frameworks or domains.

## Installation

Each package is independently importable:

```bash
# Install all core packages
go get github.com/yourusername/go-guardian/...
```

## Quick Start

### Core Usage (Any Go Project)

```go
import (
    "github.com/yourusername/go-guardian/ratelimit"
    "github.com/yourusername/go-guardian/circuitbreaker"
    "github.com/yourusername/go-guardian/trace"
)

func processRequest(ctx context.Context, req *Request) error {
    // 1. Rate limiting
    limiter := ratelimit.NewTokenBucket(100, 200)
    if err := limiter.Allow(ctx); err != nil {
        return errors.New("rate limit exceeded")
    }

    // 2. Distributed tracing
    traceID := trace.GetTraceID(ctx)
    log.Printf("[%s] Processing request", traceID)

    // 3. Circuit breaker for external calls
    breaker := circuitbreaker.New(5, 10*time.Second)
    return breaker.Call(func() error {
        return externalService.Call(ctx, req)
    })
}
```

### HTTP (Gin)

```go
import (
    "github.com/gin-gonic/gin"
    ginmw "github.com/yourusername/go-guardian/middleware/gin"
    "github.com/yourusername/go-guardian/ratelimit"
)

r := gin.Default()

// Apply middlewares
limiter := ratelimit.NewTokenBucket(100, 200)
r.Use(ginmw.Trace())           // Distributed tracing
r.Use(ginmw.RateLimit(limiter)) // Rate limiting
r.Use(ginmw.Metrics())         // Prometheus metrics

r.POST("/api/users", handler)
```

### gRPC

```go
import (
    "google.golang.org/grpc"
    grpcmw "github.com/yourusername/go-guardian/middleware/grpc"
    "github.com/yourusername/go-guardian/ratelimit"
)

// Distributed rate limiter for multi-instance deployment
limiter := ratelimit.NewRedisLimiter(redisClient, "grpc:ratelimit", 1000, time.Second)

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpcmw.UnaryTrace(),         // Trace ID propagation
        grpcmw.UnaryRateLimit(limiter), // Rate limiting
        grpcmw.UnaryMetrics(),       // Prometheus metrics
    ),
    grpc.ChainStreamInterceptor(
        grpcmw.StreamTrace(),
    ),
)
```

### LLM Application (Optional Extension)

```go
import (
    "github.com/yourusername/go-guardian/extensions/llm"
    "github.com/yourusername/go-guardian/extensions/eino"
)

// Initialize LLM-specific metrics (TTFT, tokens/s, etc.)
llm.InitMetrics("my-llm-app")

// Register Eino callbacks (auto-metrics for ChatModel)
eino.RegisterCallbacks(logger)

// Fallback strategy for LLM failures
fallback := llm.NewKeywordFallback()
response, _ := fallback.Execute(ctx, &llm.Request{
    UserQuery: "发动机故障",
})
```

## Advanced Usage

### Layered Defense Architecture

Combine multiple middleware for production-grade reliability:

```go
// 1. Rate limiting at ingress
if err := rateLimiter.Allow(ctx); err != nil {
    metrics.RecordRateLimitReject("ingress")
    return http.StatusTooManyRequests
}

// 2. Circuit breaker for LLM calls
err := circuitBreaker.Call(func() error {
    // 3. Retry with exponential backoff
    return retry.Do(func() error {
        return llmClient.Generate(ctx, prompt)
    })
})

// 4. Fallback on failure
if err != nil {
    response, _ := fallbackStrategy.Execute(ctx, &fallback.Request{
        UserQuery: query,
        LastError: err,
    })
    return response
}
```

### Grafana Dashboard

Example PromQL queries for monitoring:

```promql
# LLM Response Latency P95
histogram_quantile(0.95, rate(llm_call_duration_seconds_bucket[5m]))

# Time To First Token (TTFT) P95
histogram_quantile(0.95, rate(llm_ttft_seconds_bucket[5m]))

# Token Throughput (tokens/s)
histogram_quantile(0.50, rate(llm_tokens_per_second_bucket[5m]))

# Error Rate
rate(llm_calls_total{status="error"}[5m]) / rate(llm_calls_total[5m])

# Circuit Breaker State (0=closed, 1=open, 2=half_open)
circuit_breaker_state{service="llm"}
```

## Architecture Philosophy

### Design Principles

1. **Single Responsibility**: Each package does one thing well
2. **Zero Eino Dependency** (where possible): Rate limiter, circuit breaker, and trace work standalone
3. **Context Propagation**: All middleware use `context.Context` for trace/metadata passing
4. **Interface-First**: Abstract interfaces allow custom implementations
5. **Observable by Default**: Built-in Prometheus metrics for every operation

### Comparison with eino-ext

| Aspect | eino-ext | llm-guard |
| --- | --- | --- |
| **Focus** | Component implementations (models, tools, retrievers) | Cross-cutting concerns (reliability, observability) |
| **Dependencies** | Heavy (model SDKs, vector DBs, etc.) | Light (Redis optional, Prometheus optional) |
| **Integration Point** | Eino Components | HTTP Middleware + Optional Eino Callbacks |
| **Use Case** | "What model/tool to use?" | "How to protect and observe models?" |
| **Framework** | Eino-specific | Framework-agnostic (works with any Go framework) |

Both complement each other: `eino-ext` provides the building blocks, `llm-guard` provides the operational safeguards.

## Examples

See [examples/](./examples) for complete runnable demos:

- [Basic Usage](./examples/basic) - Standalone middleware usage
- [Gin Integration](./examples/gin) - Full-stack Gin + Eino + Middleware
- [Distributed Tracing](./examples/tracing) - Multi-service trace propagation
- [Grafana Dashboard](./examples/grafana) - Pre-built Grafana dashboards

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Adding a New Middleware

1. Create a new directory under project root (e.g., `autoscaler/`)
2. Add independent `go.mod` with minimal dependencies
3. Follow interface-first design
4. Include tests with >80% coverage
5. Update this README with usage examples

## License

This project is licensed under the Apache-2.0 License - see [LICENSE](./LICENSE) file for details.

## Related Projects

- [Eino](https://github.com/cloudwego/eino) - The core LLM application framework
- [Eino-Ext](https://github.com/cloudwego/eino-ext) - Official Eino component implementations

---

# 中文文档

为任何 Go 应用设计的生产级中间件工具包，提供限流、熔断、分布式追踪和完整的可观测性。支持 **HTTP (Gin, Hertz, net/http)**、**gRPC**，以及任何 Go 服务。

**使用场景**：微服务、API 网关、LLM 应用、分布式系统，或任何需要可靠性和可观测性的 Go 后端。

## 功能特性

- **限流器**：令牌桶（内存）和 Redis 分布式限流
- **熔断器**：三态熔断器（关闭 → 打开 → 半开），自动恢复
- **降级策略**：可插拔降级策略（固定话术、关键词匹配）
- **分布式追踪**：自动生成 trace ID，上下游传递，日志关联
- **Prometheus 指标**：完整的 LLM 性能指标（QPS、延迟分位数、TTFT、token 吞吐量等）

## 快速开始

### 安装

每个包可独立导入：

```bash
# Core packages (framework-agnostic)
go get github.com/yourusername/go-guardian/ratelimit
go get github.com/yourusername/go-guardian/circuitbreaker
go get github.com/yourusername/go-guardian/trace
go get github.com/yourusername/go-guardian/metrics
go get github.com/yourusername/go-guardian/logger

# Framework adapters
go get github.com/yourusername/go-guardian/middleware/gin
go get github.com/yourusername/go-guardian/middleware/grpc
go get github.com/yourusername/go-guardian/middleware/hertz

# Domain-specific extensions (optional)
go get github.com/yourusername/go-guardian/extensions/llm      # LLM metrics + fallback
go get github.com/yourusername/go-guardian/extensions/eino     # Eino callbacks
```

### 使用示例

详见 [Quick Start](#quick-start) 英文部分。

## 分层防护架构

```
客户端请求
    ↓
[Trace 中间件] ← 生成/透传 trace_id
    ↓
[全局限流] ← Redis 分布式限流
    ↓
[并发控制] ← 信号量/令牌桶
    ↓
[熔断器] ← LLM 调用快速失败
    ↓
[重试] ← 指数退避重试
    ↓
[降级策略] ← 固定话术/关键词匹配
    ↓
[Prometheus 指标采集] ← 全方位性能监控
```

## 与 eino-ext 的对比

| 维度 | eino-ext | llm-guard |
| --- | --- | --- |
| **聚焦点** | 组件实现（模型、工具、检索器） | 横切关注点（可靠性、可观测性） |
| **依赖** | 重（模型 SDK、向量数据库等） | 轻（Redis 可选、Prometheus 可选） |
| **集成点** | Eino Components | HTTP Middleware + 可选 Eino Callbacks |
| **使用场景** | "用什么模型/工具？" | "如何保护和监控模型？" |
| **框架** | Eino 专用 | 框架无关（任何 Go 框架均可） |

两者互补：`eino-ext` 提供构建块，`llm-guard` 提供运维保障。

## License

Apache-2.0 License
