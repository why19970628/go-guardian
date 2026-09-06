# Go Guardian

[![Go CI](https://github.com/why19970628/go-guardian/actions/workflows/ci.yml/badge.svg)](https://github.com/why19970628/go-guardian/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/why19970628/go-guardian)](https://goreportcard.com/report/github.com/why19970628/go-guardian)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GitHub release](https://img.shields.io/github/v/release/why19970628/go-guardian)](https://github.com/why19970628/go-guardian/releases)

[English](#english) | [中文](#中文)

---

## English

Production-ready middleware toolkit for Go applications. Provides rate limiting, circuit breaking, distributed tracing, and comprehensive observability. Works with **HTTP (Gin, Hertz, net/http)**, **gRPC**, and any Go service.

**Use Cases**: Microservices, API gateways, LLM applications, distributed systems, or any Go backend requiring reliability and observability.

### Why Go Guardian?

A universal Go middleware toolkit designed for **any Go application**:
- **Protocol-agnostic**: Core logic works with HTTP, gRPC, or direct function calls
- **Framework adapters**: Pre-built adapters for Gin, Hertz, gRPC, and more
- **Domain-specific extensions**: Generic + specialized (e.g., LLM metrics with TTFT/tokens/s)
- **Production-tested**: Battle-tested patterns from high-scale distributed systems
- **Zero lock-in**: Use individual packages independently
- **Minimal dependencies**: Each package has its own `go.mod`

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
- **Coming soon**: Kratos, go-zero

**Domain-Specific (Optional)**
- **LLM Applications**: TTFT (Time To First Token), token throughput, prompt/completion metrics, Eino callbacks
- **Database**: Query performance tracking (coming soon)
- **Cache**: Hit rate, latency tracking (coming soon)

### Installation

```bash
# Core packages (framework-agnostic)
go get github.com/why19970628/go-guardian/ratelimit
go get github.com/why19970628/go-guardian/circuitbreaker
go get github.com/why19970628/go-guardian/trace
go get github.com/why19970628/go-guardian/metrics
go get github.com/why19970628/go-guardian/logger

# Framework adapters
go get github.com/why19970628/go-guardian/middleware/gin
go get github.com/why19970628/go-guardian/middleware/grpc

# Domain-specific extensions (optional)
go get github.com/why19970628/go-guardian/extensions/llm/fallback
go get github.com/why19970628/go-guardian/extensions/eino/observability
```

Or install all at once:

```bash
go get github.com/why19970628/go-guardian/...
```

### Quick Start

#### Core Usage (Any Go Project)

```go
import (
    "github.com/why19970628/go-guardian/ratelimit"
    "github.com/why19970628/go-guardian/circuitbreaker"
    "github.com/why19970628/go-guardian/trace"
)

func processRequest(ctx context.Context, req *Request) error {
    // 1. Rate limiting
    limiter := ratelimit.NewTokenBucket(100, 200)
    if err := limiter.Allow(ctx); err != nil {
        return errors.New("rate limit exceeded")
    }
    
    // 2. Distributed tracing
    ctx = trace.WithTraceID(ctx, trace.GenerateTraceID())
    
    // 3. Circuit breaker for external calls
    breaker := circuitbreaker.New(5, 10*time.Second)
    return breaker.Call(func() error {
        return externalService.Call(ctx, req)
    })
}
```

#### HTTP (Gin)

```go
import (
    "github.com/gin-gonic/gin"
    ginmw "github.com/why19970628/go-guardian/middleware/gin"
    "github.com/why19970628/go-guardian/ratelimit"
)

r := gin.Default()

// Apply middlewares
limiter := ratelimit.NewTokenBucket(100, 200)
r.Use(ginmw.Trace())           // Distributed tracing
r.Use(ginmw.RateLimit(limiter)) // Rate limiting
r.Use(ginmw.Metrics())         // Prometheus metrics

r.POST("/api/users", handler)
```

#### gRPC

```go
import (
    "google.golang.org/grpc"
    grpcmw "github.com/why19970628/go-guardian/middleware/grpc"
    "github.com/why19970628/go-guardian/ratelimit"
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

#### LLM Application (Optional Extension)

```go
import (
    "github.com/why19970628/go-guardian/extensions/llm"
    "github.com/why19970628/go-guardian/extensions/eino/observability"
)

// Initialize LLM-specific metrics (TTFT, tokens/s, etc.)
llm.InitMetrics("my-llm-app")

// Register Eino callbacks (auto-metrics for ChatModel)
observability.RegisterCallbacks(logger)

// Fallback strategy for LLM failures
fallback := llm.NewKeywordFallback()
response, _ := fallback.Execute(ctx, &llm.Request{
    UserQuery: "Engine fault code",
})
```

### Project Structure

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
│   └── go.mod
├── metrics/                # Prometheus metrics (generic)
│   ├── http.go             # HTTP metrics
│   ├── grpc.go             # gRPC metrics
│   └── go.mod
├── logger/                 # Structured logging
│   ├── logger.go           # Zap wrapper with trace context
│   └── go.mod
├── middleware/             # Framework adapters
│   ├── gin/                # Gin middleware
│   ├── grpc/               # gRPC interceptors
│   └── hertz/              # Hertz middleware (coming soon)
├── extensions/             # Domain-specific extensions (optional)
│   ├── llm/                # LLM-specific (TTFT, fallback, etc.)
│   ├── eino/               # Eino framework integration
│   └── database/           # Database metrics (coming soon)
└── examples/               # Usage examples
    ├── http-gin/           # Gin microservice
    ├── grpc/               # gRPC service
    └── llm-app/            # LLM application with Eino
```

### Design Philosophy

- **Core packages** (ratelimit, circuitbreaker, trace, metrics, logger): Pure logic, no framework dependencies, works anywhere
- **Middleware packages** (middleware/*): Framework-specific adapters (Gin, gRPC, Hertz)
- **Extension packages** (extensions/*): Domain-specific features (LLM metrics, Eino callbacks, database tracking)

**Dependency Direction**: `middleware → core`, `extensions → core`. Core packages have zero dependency on frameworks or domains.

### Comparison with eino-ext

| Aspect | eino-ext | go-guardian |
| --- | --- | --- |
| **Focus** | Component implementations (models, tools, retrievers) | Cross-cutting concerns (reliability, observability) |
| **Dependencies** | Heavy (model SDKs, vector DBs, etc.) | Light (Redis optional, Prometheus optional) |
| **Integration Point** | Eino Components | HTTP Middleware + Optional Eino Callbacks |
| **Use Case** | "What model/tool to use?" | "How to protect and observe models?" |
| **Framework** | Eino-specific | Framework-agnostic (works with any Go framework) |

Both complement each other: `eino-ext` provides the building blocks, `go-guardian` provides the operational safeguards.

### Documentation

- [Quick Start Guide](./QUICKSTART.md) - Get started in 1 minute
- [Contributing Guidelines](./CONTRIBUTING.md) - How to contribute
- [Security Policy](./SECURITY.md) - Security best practices
- [Project Summary](./PROJECT_SUMMARY.md) - Architecture and roadmap

### Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](./CONTRIBUTING.md) before submitting PRs.

### License

Apache License 2.0 - see [LICENSE](./LICENSE) for details.

---

## 中文

为任何 Go 应用设计的生产级中间件工具包，提供限流、熔断、分布式追踪和完整的可观测性。支持 **HTTP (Gin, Hertz, net/http)**、**gRPC**，以及任何 Go 服务。

**使用场景**：微服务、API 网关、LLM 应用、分布式系统，或任何需要可靠性和可观测性的 Go 后端。

### 为什么选择 Go Guardian？

为**任何 Go 应用**设计的通用中间件工具包：
- **协议无关**：核心逻辑适用于 HTTP、gRPC 或直接函数调用
- **框架适配器**：为 Gin、Hertz、gRPC 等预构建适配器
- **领域扩展**：通用 + 专用（如 LLM 指标：TTFT / tokens/s）
- **生产验证**：来自大规模分布式系统的实战模式
- **零锁定**：独立使用单个包
- **最小依赖**：每个包有独立的 `go.mod`

### 功能特性

**核心中间件（框架无关）**
- **限流器**：令牌桶（内存）+ Redis 分布式限流
- **熔断器**：三态熔断器（CLOSED → OPEN → HALF_OPEN）自动恢复
- **分布式追踪**：自动生成 trace ID、传递和上下文关联
- **Prometheus 指标**：完整的可观测性（QPS、延迟分位数、错误率、并发数）
- **结构化日志**：基于 Zap，自动注入 trace context

**框架适配器**
- **HTTP**：Gin、Hertz、`net/http` 中间件
- **gRPC**：Unary 和 Stream 拦截器
- **即将支持**：Kratos、go-zero

**领域扩展（可选）**
- **LLM 应用**：TTFT（首 token 延迟）、token 吞吐量、prompt/completion 指标、Eino callbacks
- **数据库**：查询性能追踪（即将推出）
- **缓存**：命中率、延迟追踪（即将推出）

### 安装

```bash
# 核心包（框架无关）
go get github.com/why19970628/go-guardian/ratelimit
go get github.com/why19970628/go-guardian/circuitbreaker
go get github.com/why19970628/go-guardian/trace
go get github.com/why19970628/go-guardian/metrics
go get github.com/why19970628/go-guardian/logger

# 框架适配器
go get github.com/why19970628/go-guardian/middleware/gin
go get github.com/why19970628/go-guardian/middleware/grpc

# 领域扩展（可选）
go get github.com/why19970628/go-guardian/extensions/llm/fallback
go get github.com/why19970628/go-guardian/extensions/eino/observability
```

或一次性安装全部：

```bash
go get github.com/why19970628/go-guardian/...
```

### 快速开始

查看 [快速开始指南](./QUICKSTART.md) 了解详细用法。

### 文档

- [快速开始指南](./QUICKSTART.md) - 1 分钟上手
- [贡献指南](./CONTRIBUTING.md) - 如何贡献代码
- [安全策略](./SECURITY.md) - 安全最佳实践
- [项目总结](./PROJECT_SUMMARY.md) - 架构设计与路线图

### 贡献

欢迎贡献！提交 PR 前请阅读 [贡献指南](./CONTRIBUTING.md)。

### 许可证

Apache License 2.0 - 详见 [LICENSE](./LICENSE)。

### 联系方式

- GitHub: https://github.com/why19970628/go-guardian
- Issues: https://github.com/why19970628/go-guardian/issues
