# Go Guardian - 快速参考

## 一分钟上手

### 通用 Go 服务

```go
import (
    "github.com/wanghuayang820310/go-guardian/ratelimit"
    "github.com/wanghuayang820310/go-guardian/circuitbreaker"
    "github.com/wanghuayang820310/go-guardian/trace"
)

// 1. 初始化组件
limiter := ratelimit.NewTokenBucket(100, 200)
breaker := circuitbreaker.New(5, 10*time.Second)

// 2. 使用
func handleRequest(ctx context.Context, req *Request) error {
    // 限流
    if err := limiter.Allow(ctx); err != nil {
        return err
    }
    
    // 追踪
    ctx = trace.WithTraceID(ctx, trace.GenerateTraceID())
    
    // 熔断
    return breaker.Call(func() error {
        return service.Process(ctx, req)
    })
}
```

### HTTP (Gin)

```go
import (
    "github.com/gin-gonic/gin"
    ginmw "github.com/wanghuayang820310/go-guardian/middleware/gin"
)

r := gin.Default()
r.Use(ginmw.Trace())
r.Use(ginmw.RateLimit(limiter))
r.POST("/api", handler)
```

### gRPC

```go
import (
    "google.golang.org/grpc"
    grpcmw "github.com/wanghuayang820310/go-guardian/middleware/grpc"
)

server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcmw.UnaryTrace()),
    grpc.UnaryInterceptor(grpcmw.UnaryRateLimit(limiter)),
)
```

### LLM 应用 (Eino)

```go
import (
    "github.com/wanghuayang820310/go-guardian/extensions/llm"
    "github.com/wanghuayang820310/go-guardian/extensions/eino/observability"
)

// 自动监控所有 ChatModel 调用
observability.Register(logger)

// 降级策略
fallback := llm.NewKeywordFallback()
response, _ := fallback.Execute(ctx, &llm.Request{
    UserQuery: "发动机故障码",
})
```

## 包导入速查

| 功能 | 导入路径 | 用途 |
| --- | --- | --- |
| 限流 | `go-guardian/ratelimit` | 令牌桶 / Redis 分布式限流 |
| 熔断 | `go-guardian/circuitbreaker` | 三态熔断器 |
| 追踪 | `go-guardian/trace` | Trace ID 生成与传递 |
| 指标 | `go-guardian/metrics` | Prometheus 指标 |
| 日志 | `go-guardian/logger` | Zap + trace context |
| Gin 适配 | `go-guardian/middleware/gin` | Gin 中间件 |
| gRPC 适配 | `go-guardian/middleware/grpc` | gRPC 拦截器 |
| LLM 扩展 | `go-guardian/extensions/llm` | TTFT、降级策略 |
| Eino 集成 | `go-guardian/extensions/eino/observability` | Eino callbacks |

## 常见场景

### 场景1：微服务网关

```go
r := gin.Default()
r.Use(ginmw.Trace())                    // 追踪
r.Use(ginmw.RateLimit(redisLimiter))    // 分布式限流
r.Use(ginmw.Metrics())                  // 指标
r.Use(ginmw.Recovery())                 // 恢复
```

### 场景2：gRPC 服务

```go
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpcmw.UnaryTrace(),
        grpcmw.UnaryRateLimit(limiter),
        grpcmw.UnaryCircuitBreaker(breaker),
        grpcmw.UnaryMetrics(),
    ),
)
```

### 场景3：分布式系统

```go
// 上游服务 A
ctx = trace.WithTraceID(ctx, trace.GenerateTraceID())
grpcClient.Call(ctx, req) // trace ID 自动传递

// 下游服务 B
traceID := trace.GetTraceID(ctx) // 自动获取
logger.Infow("Processing", "trace_id", traceID)
```

### 场景4：LLM 应用

```go
// 初始化
llm.InitMetrics("my-llm-app")
observability.Register(logger)

// 使用（自动埋点）
agent := eino.NewAgent(...)
response := agent.Run(ctx, input) // 自动采集 TTFT、tokens/s

// 降级
if err != nil {
    fallback.Execute(ctx, &llm.Request{...})
}
```

## Prometheus 指标速查

| 指标名 | 类型 | 说明 |
| --- | --- | --- |
| `guardian_requests_total` | Counter | 请求总数 |
| `guardian_request_duration_seconds` | Histogram | 请求延迟 |
| `guardian_concurrent_requests` | Gauge | 并发数 |
| `guardian_circuit_breaker_state` | Gauge | 熔断器状态 (0/1/2) |
| `guardian_ratelimit_rejects_total` | Counter | 限流拒绝次数 |
| `guardian_llm_ttft_seconds` | Histogram | LLM 首 token 延迟 |
| `guardian_llm_tokens_per_second` | Histogram | LLM 吞吐量 |

## 故障排查

### 限流生效了吗？

```promql
rate(guardian_ratelimit_rejects_total[5m])
```

### 熔断器是否打开？

```promql
guardian_circuit_breaker_state{service="my-service"} == 1
```

### 追踪是否工作？

```bash
curl -H "X-Trace-ID: test-123" http://localhost:8080/api
# 检查日志是否包含 trace_id=test-123
```

### LLM 性能如何？

```promql
# TTFT P95
histogram_quantile(0.95, rate(guardian_llm_ttft_seconds_bucket[5m]))

# 吞吐量中位数
histogram_quantile(0.50, rate(guardian_llm_tokens_per_second_bucket[5m]))
```

## 最佳实践

1. **分布式部署**：使用 Redis 限流器，而非内存令牌桶
2. **链路追踪**：在入口处生成 trace ID，所有下游自动传递
3. **指标采集**：暴露 `/metrics` 端点，Prometheus 定期拉取
4. **熔断阈值**：根据 P99 延迟设置超时，根据错误率设置熔断
5. **降级策略**：LLM 应用必须有 fallback，避免全链路阻塞

## 项目地址

- GitHub: `https://github.com/wanghuayang820310/go-guardian`
- 文档: `https://github.com/wanghuayang820310/go-guardian/blob/main/README.md`
- 示例: `https://github.com/wanghuayang820310/go-guardian/tree/main/examples`

## 贡献

欢迎贡献！请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md)
