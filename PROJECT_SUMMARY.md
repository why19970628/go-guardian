# Go Guardian - 项目总结

## 项目定位

**Go Guardian** 是一个通用的 Go 中间件工具包，为任何 Go 应用提供生产级的可靠性、可观测性和性能管理能力。

### 与其他项目的对比

| 项目 | 定位 | 适用场景 |
| --- | --- | --- |
| **go-guardian** | 通用中间件（限流、熔断、追踪、指标） | 任何 Go 项目 |
| **eino** | LLM 应用框架 | LLM 应用构建 |
| **eino-ext** | Eino 组件实现（模型、工具、检索器） | Eino 生态扩展 |

**互补关系**：
- `eino` 提供 LLM 应用框架
- `eino-ext` 提供模型/工具实现
- `go-guardian` 提供运维保障（可用于 Eino 应用，也可用于任何 Go 服务）

## 项目结构

```
go-guardian/
├── Core Packages (框架无关，零依赖)
│   ├── ratelimit/          # 限流器（令牌桶 + Redis）
│   ├── circuitbreaker/     # 熔断器（三态状态机）
│   ├── trace/              # 分布式追踪（trace ID 生成与传递）
│   ├── metrics/            # Prometheus 指标（通用）
│   └── logger/             # 结构化日志（Zap + trace context）
│
├── Middleware Adapters (框架适配器)
│   ├── middleware/gin/     # Gin 中间件
│   ├── middleware/grpc/    # gRPC 拦截器
│   └── middleware/hertz/   # Hertz 中间件（计划中）
│
└── Extensions (领域扩展，可选)
    ├── extensions/llm/     # LLM 特定（TTFT、tokens/s、fallback）
    ├── extensions/eino/    # Eino 框架集成（callbacks）
    └── extensions/database/# 数据库指标（计划中）
```

### 设计原则

1. **核心包 (core)**: 纯逻辑，无框架依赖，适用于任何 Go 项目
2. **适配器 (middleware)**: 框架特定的包装层（Gin、gRPC、Hertz 等）
3. **扩展 (extensions)**: 领域特定功能（LLM、数据库、缓存等），可选安装

**依赖方向**: `middleware → core`, `extensions → core`。核心包对框架和领域零依赖。

## 功能特性

### 核心能力（任何 Go 项目可用）

- **限流器**
  - 单机令牌桶（内存）
  - Redis 分布式限流（滑动窗口）
  - 接口抽象，可扩展其他实现

- **熔断器**
  - 三态状态机（CLOSED → OPEN → HALF_OPEN）
  - 连续失败达阈值触发熔断
  - 超时后自动探测恢复

- **分布式追踪**
  - 自动生成 trace ID（32 字符十六进制）
  - 上下游透传（HTTP header / gRPC metadata）
  - Context 传递，日志自动关联

- **Prometheus 指标**
  - HTTP: QPS、延迟分位数、状态码分布
  - gRPC: 调用次数、延迟、gRPC 状态码
  - 通用: 错误率、并发数、熔断器状态、限流拒绝次数

- **结构化日志**
  - Zap 封装
  - 自动注入 trace_id 和 caller (file:line)

### LLM 应用扩展（可选）

- **LLM 性能指标** (`extensions/llm/metrics`)
  - 首 token 延迟 (TTFT)
  - 生成吞吐量 (tokens/s)
  - 每 token 平均延迟 (ms/token)
  - Prompt/Completion 长度分布
  - Token 用量统计

- **降级策略** (`extensions/llm/fallback`)
  - 固定话术降级
  - 关键词匹配降级
  - 可插拔策略接口

- **Eino 集成** (`extensions/eino/observability`)
  - 自动监听 ChatModel 生命周期
  - 自动上报 Prometheus 指标
  - 无侵入式集成

## 使用场景

### 1. 通用微服务

```go
// HTTP API
r := gin.Default()
r.Use(ginmw.Trace())
r.Use(ginmw.RateLimit(limiter))
r.Use(ginmw.Metrics())

// gRPC Service
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcmw.UnaryTrace()),
    grpc.UnaryInterceptor(grpcmw.UnaryRateLimit(limiter)),
)
```

### 2. LLM 应用（Eino）

```go
// 初始化 LLM 指标
llm.InitMetrics("my-llm-app")

// 注册 Eino callbacks（自动监控所有 ChatModel 调用）
eino.RegisterCallbacks(logger)

// 降级策略
fallback := llm.NewKeywordFallback()
```

### 3. 纯 Go 服务（无框架）

```go
func processRequest(ctx context.Context, req *Request) error {
    // 限流
    if err := limiter.Allow(ctx); err != nil {
        return errors.New("rate limit exceeded")
    }
    
    // 追踪
    traceID := trace.GetTraceID(ctx)
    log.Printf("[%s] Processing", traceID)
    
    // 熔断
    return breaker.Call(func() error {
        return externalService.Call(ctx, req)
    })
}
```

## 与 ai-mechanic 的关系

`go-guardian` 是从 `ai-mechanic` 抽取出来的通用中间件：

| 能力 | ai-mechanic | go-guardian |
| --- | --- | --- |
| 限流/熔断/追踪 | 内置，AI Mechanic 专用 | 通用，任何 Go 项目可用 |
| LLM 指标 | 内置 | 可选扩展 (`extensions/llm`) |
| Eino 集成 | 硬编码 | 可选扩展 (`extensions/eino`) |
| 框架适配 | 只支持 Gin | 支持 Gin/gRPC/Hertz/任意框架 |

**迁移路径**：
1. `ai-mechanic` 的 `internal/{ratelimit,circuitbreaker,trace,metrics}` → `go-guardian` 核心包
2. `ai-mechanic` 改为依赖 `go-guardian`，删除 `internal/` 重复代码
3. 其他项目可直接使用 `go-guardian`

## 后续计划

### 已完成
- ✅ 核心包抽取（限流、熔断、追踪、指标、日志）
- ✅ Gin 中间件适配器
- ✅ gRPC 拦截器
- ✅ LLM 扩展（指标、降级）
- ✅ Eino 集成
- ✅ 项目文档和示例

### 计划中
- ⏳ Hertz 中间件适配器
- ⏳ Kratos 中间件适配器
- ⏳ go-zero 中间件适配器
- ⏳ 数据库扩展（查询性能、连接池指标）
- ⏳ 缓存扩展（命中率、延迟追踪）
- ⏳ 完整的示例项目（HTTP、gRPC、LLM 应用）
- ⏳ 单元测试覆盖率 >80%
- ⏳ 发布到 GitHub 并开源

## 发布清单

在发布到 GitHub 前需要完成：

1. **代码质量**
   - [ ] 所有包通过 `go build`
   - [ ] 单元测试覆盖率 >60%
   - [ ] `golangci-lint` 通过
   - [ ] 所有公开函数有 godoc 注释

2. **文档**
   - [x] README.md (中英文)
   - [x] CONTRIBUTING.md
   - [x] LICENSE (Apache 2.0)
   - [ ] 每个子包的 README
   - [ ] 完整的使用示例

3. **示例代码**
   - [ ] HTTP (Gin) 示例
   - [ ] gRPC 示例
   - [ ] LLM 应用示例
   - [ ] 纯 Go 服务示例

4. **CI/CD**
   - [ ] GitHub Actions 配置
   - [ ] 自动测试
   - [ ] 自动 lint
   - [ ] 覆盖率报告

5. **发布**
   - [ ] 创建 GitHub 仓库
   - [ ] 打 v0.1.0 tag
   - [ ] 发布 Release Notes
   - [ ] 更新所有导入路径（替换 `yourusername`）

## 总结

**Go Guardian** 是一个通用的 Go 中间件工具包，参考 `eino-ext` 的模块化设计，但专注于运维保障而非业务组件。它可以：

- 独立使用于任何 Go 项目
- 可选集成到 Eino LLM 应用
- 通过扩展包支持特定领域（LLM、数据库、缓存等）

**核心价值**：让开发者专注业务逻辑，运维保障开箱即用。
