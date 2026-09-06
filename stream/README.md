# Stream - 并发安全的 SSE 流式处理

[![Go Reference](https://pkg.go.dev/badge/github.com/why19970628/go-guardian/stream.svg)](https://pkg.go.dev/github.com/why19970628/go-guardian/stream)

生产级 SSE（Server-Sent Events）流式处理框架，提供并发安全、心跳保活、错误跟踪等完整功能。

## 特性

✅ **并发安全**
- `sync.Mutex` 保护并发写入
- `sync.WaitGroup` 等待所有 goroutine 退出
- `sync.Once` 保证 cleanup 只执行一次

✅ **心跳保活**
- 可配置心跳间隔（默认 10s）
- 自动发送 `: PING\n\n` 保持连接
- 防止 CDN/代理超时断开

✅ **超时管理**
- 流式超时（默认 30s）
- 写入截止时间自动延长（30s）
- 心跳最大持续时间（30 分钟）

✅ **错误跟踪**
- 9 种结束原因：done/timeout/client_gone/panic/...
- 最多记录 20 条错误历史
- 线程安全的错误记录

✅ **缓冲区管理**
- 初始缓冲区：64KB
- 最大缓冲区：128MB（可配置）
- `bufio.Scanner` 自动扩容

## 安装

```bash
go get github.com/why19970628/go-guardian/stream@latest
```

## 快速开始

### 基础用法

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/why19970628/go-guardian/stream"
    "go.uber.org/zap"
)

func handler(c *gin.Context) {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // 1. 创建配置
    config := stream.DefaultScannerConfig(logger.Sugar())
    config.PingEnabled = true
    config.PingInterval = 10 * time.Second
    
    // 2. 创建扫描器
    scanner := stream.NewStreamScanner(config)
    
    // 3. 发起上游请求
    resp, err := http.Get("https://api.example.com/stream")
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 4. 扫描并处理流式数据
    scanner.Scan(c, resp, func(data string, result *stream.Result) {
        // 解析数据
        var chunk struct {
            Text string `json:"text"`
        }
        if err := json.Unmarshal([]byte(data), &chunk); err != nil {
            result.Error(err)
            return
        }
        
        // 写入 SSE
        if err := stream.ObjectData(c, map[string]string{
            "text": chunk.Text,
            "type": "text",
        }); err != nil {
            result.Stop(err)
            return
        }
        
        // 检测结束标记
        if chunk.Text == "[DONE]" {
            result.Done()
        }
    })
    
    // 5. 发送完成标记
    stream.DoneData(c)
    
    // 6. 检查状态
    status := scanner.Status()
    logger.Info("stream completed", zap.String("summary", status.Summary()))
}
```

### 自定义响应头

```go
config := stream.DefaultScannerConfig(logger.Sugar())
config.CopyResponseHeader = func(c *gin.Context, resp *http.Response) {
    // 复制自定义响应头
    for _, name := range []string{"X-Request-ID", "X-Custom-Header"} {
        if values := resp.Header.Values(name); len(values) > 0 {
            for _, value := range values {
                c.Writer.Header().Add(name, value)
            }
        }
    }
}
```

## API 文档

### StreamScanner

并发安全的流式扫描器。

```go
type StreamScanner struct {
    // 私有字段
}

// NewStreamScanner 创建流式扫描器
func NewStreamScanner(config ScannerConfig) *StreamScanner

// Scan 执行流式扫描
func (s *StreamScanner) Scan(c *gin.Context, resp *http.Response, dataHandler DataHandler)

// Status 获取流式状态
func (s *StreamScanner) Status() *Status
```

### ScannerConfig

```go
type ScannerConfig struct {
    Logger             *zap.SugaredLogger                          // 日志
    StreamingTimeout   time.Duration                               // 流式超时（默认 30s）
    PingInterval       time.Duration                               // 心跳间隔（默认 10s）
    PingEnabled        bool                                        // 是否启用心跳
    MaxBufferSize      int                                         // 最大缓冲区大小（默认 128MB）
    CopyResponseHeader func(*gin.Context, *http.Response)          // 复制响应头回调
}

// DefaultScannerConfig 默认配置
func DefaultScannerConfig(logger *zap.SugaredLogger) ScannerConfig
```

### DataHandler

数据处理回调函数。

```go
type DataHandler func(data string, result *Result)
```

**参数：**
- `data`: 当前行数据（去除 `data:` 前缀）
- `result`: 用于记录错误、停止流或标记完成

### Result

流式处理结果。

```go
type Result struct {
    // 私有字段
}

// Error 记录软错误（流继续处理）
func (r *Result) Error(err error)

// Stop 记录致命错误并停止流
func (r *Result) Stop(err error)

// Done 标记正常完成并停止流
func (r *Result) Done()

// IsStopped 是否已停止
func (r *Result) IsStopped() bool
```

### Status

流式状态跟踪。

```go
type Status struct {
    EndReason  EndReason    // 结束原因
    EndError   error        // 结束错误
    Errors     []ErrorEntry // 错误列表（最多 20 条）
    ErrorCount int          // 总错误数
}

// SetEndReason 设置结束原因（只能设置一次）
func (s *Status) SetEndReason(reason EndReason, err error)

// RecordError 记录错误（线程安全）
func (s *Status) RecordError(msg string)

// HasErrors 是否有错误
func (s *Status) HasErrors() bool

// TotalErrorCount 总错误数
func (s *Status) TotalErrorCount() int

// GetErrors 获取错误列表（副本）
func (s *Status) GetErrors() []ErrorEntry

// Summary 生成状态摘要
func (s *Status) Summary() string

// IsNormalEnd 是否正常结束
func (s *Status) IsNormalEnd() bool
```

### EndReason

流式结束原因枚举。

```go
type EndReason string

const (
    EndReasonNone        EndReason = ""              // 未结束
    EndReasonDone        EndReason = "done"          // 正常完成
    EndReasonTimeout     EndReason = "timeout"       // 超时
    EndReasonClientGone  EndReason = "client_gone"   // 客户端断开
    EndReasonScannerErr  EndReason = "scanner_error" // 扫描器错误
    EndReasonHandlerStop EndReason = "handler_stop"  // 处理器停止
    EndReasonEOF         EndReason = "eof"           // 流结束
    EndReasonPanic       EndReason = "panic"         // Panic
    EndReasonPingFail    EndReason = "ping_fail"     // 心跳失败
    EndReasonAgentError  EndReason = "agent_error"   // Agent 执行错误
)
```

### SSE 写入助手

```go
// StringData 写入 SSE 字符串数据
func StringData(c *gin.Context, str string) error

// ObjectData 写入 SSE JSON 对象
func ObjectData(c *gin.Context, object interface{}) error

// DoneData 写入 SSE 完成标记
func DoneData(c *gin.Context) error
```

## 高级用法

### 错误处理策略

```go
scanner.Scan(c, resp, func(data string, result *stream.Result) {
    // 软错误：记录但继续处理
    if err := validateData(data); err != nil {
        result.Error(err)
        // 继续处理下一个 chunk
        return
    }
    
    // 致命错误：停止流
    if err := writeData(c, data); err != nil {
        result.Stop(err)
        return
    }
    
    // 正常完成：标记完成
    if isLastChunk(data) {
        result.Done()
    }
})
```

### 监控指标

```go
status := scanner.Status()

// Prometheus 指标示例
streamEndReasonCounter.WithLabelValues(string(status.EndReason)).Inc()
streamErrorCounter.Add(float64(status.TotalErrorCount()))

if status.HasErrors() {
    logger.Warn("stream completed with errors",
        zap.String("reason", string(status.EndReason)),
        zap.Int("error_count", status.TotalErrorCount()),
    )
}
```

### 自定义超时

```go
config := stream.DefaultScannerConfig(logger.Sugar())
config.StreamingTimeout = 60 * time.Second  // 流式超时
config.PingInterval = 5 * time.Second       // 心跳间隔
```

## 工作原理

### 架构

```
┌─────────────────────────────────────────────────────────────┐
│                       StreamScanner                          │
├─────────────────────────────────────────────────────────────┤
│  主扫描循环 (Main Loop)                                     │
│  ├─ bufio.Scanner 逐行读取                                  │
│  ├─ 超时检测 (Ticker)                                       │
│  └─ 客户端断开检测 (Context.Done)                          │
├─────────────────────────────────────────────────────────────┤
│  心跳 Goroutine (Ping Loop)                                 │
│  ├─ 定时发送 ": PING\n\n"                                   │
│  ├─ 并发写入保护 (Mutex)                                    │
│  └─ WriteDeadline 延长                                      │
├─────────────────────────────────────────────────────────────┤
│  数据处理 Goroutine (Data Loop)                             │
│  ├─ 从 channel 读取数据                                     │
│  ├─ 调用 dataHandler 回调                                   │
│  └─ 检测停止信号 (result.IsStopped)                        │
├─────────────────────────────────────────────────────────────┤
│  清理机制 (Cleanup)                                          │
│  ├─ sync.Once 保证只执行一次                                │
│  ├─ WaitGroup 等待所有 goroutine 退出                       │
│  └─ 关闭 HTTP Response Body                                 │
└─────────────────────────────────────────────────────────────┘
```

### 并发安全保证

1. **写入保护**：`sync.Mutex` 保护所有 SSE 写入
2. **优雅关闭**：`sync.Once` + `sync.WaitGroup` 保证资源释放
3. **停止信号**：`stopChan` + `Context` 协同通知所有 goroutine

### 超时管理

- **StreamingTimeout**：主循环超时，防止长时间无数据
- **WriteDeadline**：单次写入超时（30s），防止慢客户端阻塞
- **PingInterval**：心跳间隔，保持连接活跃

## 最佳实践

### 1. 错误分级

```go
// 软错误：解析失败但不影响后续处理
if err := json.Unmarshal([]byte(data), &chunk); err != nil {
    result.Error(err)
    return
}

// 致命错误：写入失败需要立即停止
if err := stream.ObjectData(c, response); err != nil {
    result.Stop(err)
    return
}
```

### 2. 心跳配置

```go
// 短连接（< 1 分钟）：关闭心跳
config.PingEnabled = false

// 长连接（> 1 分钟）：启用心跳
config.PingEnabled = true
config.PingInterval = 10 * time.Second
```

### 3. 日志记录

```go
status := scanner.Status()
if !status.IsNormalEnd() {
    logger.Error("stream abnormal end", zap.String("summary", status.Summary()))
}
```

### 4. 资源清理

`StreamScanner.Scan` 内部已处理清理，无需手动调用 `Close()`：

```go
scanner.Scan(c, resp, dataHandler)
// resp.Body 自动关闭
// 所有 goroutine 自动退出
```

## 与 New-API 对比

本实现参考了 [New-API](https://github.com/QuantumNous/new-api) 的流式处理设计，主要改进：

| 特性 | New-API | go-guardian/stream |
|------|---------|-------------------|
| 并发安全 | ✅ | ✅ |
| 心跳保活 | ✅ | ✅ |
| 错误跟踪 | ✅ 20 条 | ✅ 20 条 |
| 缓冲区管理 | ✅ 64KB~128MB | ✅ 64KB~128MB |
| 接口抽象 | ❌ 耦合 Gin | ✅ 清晰接口 |
| 可测试性 | ⚠️ 一般 | ✅ 高（接口注入）|
| 文档完整性 | ⚠️ 缺失 | ✅ 完整 |

## 许可证

Apache-2.0 License - 详见 [LICENSE](../LICENSE)

## 贡献

欢迎提交 Issue 和 PR！详见 [CONTRIBUTING.md](../CONTRIBUTING.md)
