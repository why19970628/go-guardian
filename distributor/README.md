# Distributor - 多 LLM 提供商渠道分发器

[![Go Reference](https://pkg.go.dev/badge/github.com/why19970628/go-guardian/distributor.svg)](https://pkg.go.dev/github.com/why19970628/go-guardian/distributor)

智能渠道选择、用户亲和性、健康检查，支持多种 LLM 提供商。

## 特性

✅ **三种选择策略**
- Priority（优先级）：选择优先级最高的渠道
- Random（随机）：均匀分配流量
- Weighted（加权）：按权重分配流量

✅ **用户亲和性**
- 同一用户固定同一渠道（保持会话连续性）
- 运行时可动态开关

✅ **健康检查**
- 定时探活（默认 30s）
- 自动摘除不健康渠道

✅ **运行时控制**
- 动态切换策略
- 动态启用/禁用用户亲和性

✅ **6 种提供商**
- OpenAI / Claude / Gemini / DeepSeek / Doubao / Custom

## 安装

\`\`\`bash
go get github.com/why19970628/go-guardian/distributor@latest
\`\`\`

## 快速开始

### 基础用法

\`\`\`go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/why19970628/go-guardian/distributor"
)

func main() {
    // 创建分发器
    d := distributor.NewDistributor(distributor.Config{
        Channels: []*distributor.Channel{
            {
                ID:       "openai-main",
                Name:     "OpenAI 主渠道",
                Provider: distributor.ProviderOpenAI,
                BaseURL:  "https://api.openai.com/v1",
                APIKey:   "sk-xxx",
                Models:   []string{"gpt-4", "gpt-3.5-turbo"},
                Priority: 1,
                Weight:   70,
                Enabled:  true,
                Healthy:  true,
            },
            {
                ID:       "deepseek-backup",
                Name:     "DeepSeek 备份",
                Provider: distributor.ProviderDeepSeek,
                BaseURL:  "https://api.deepseek.com/v1",
                APIKey:   "sk-xxx",
                Models:   []string{"deepseek-chat"},
                Priority: 2,
                Weight:   30,
                Enabled:  true,
                Healthy:  true,
            },
        },
        DefaultStrategy:    distributor.StrategyWeighted,
        EnableUserAffinity: true,
        HealthCheckPeriod:  30 * time.Second,
    })
    defer d.Stop()

    // 选择渠道（使用默认策略）
    ch, err := d.SelectChannelWithDefault("user123", "gpt-4")
    if err != nil {
        panic(err)
    }

    fmt.Printf("选中渠道: %s (%s)\n", ch.Name, ch.Provider)
}
\`\`\`

### 三种策略对比

\`\`\`go
// 1. 优先级策略（默认）
d := distributor.NewDistributor(distributor.Config{
    DefaultStrategy: distributor.StrategyPriority,
    // 总是选择 Priority 最小的健康渠道
})

// 2. 随机策略
d := distributor.NewDistributor(distributor.Config{
    DefaultStrategy: distributor.StrategyRandom,
    // 从健康渠道中均匀随机选择
})

// 3. 加权随机策略
d := distributor.NewDistributor(distributor.Config{
    DefaultStrategy: distributor.StrategyWeighted,
    // 按 Weight 分配流量（70% / 30%）
})
\`\`\`

### 用户亲和性

\`\`\`go
d := distributor.NewDistributor(distributor.Config{
    EnableUserAffinity: true,  // 开启用户亲和性
})

// 同一用户总是路由到同一渠道
ch1, _ := d.SelectChannelWithDefault("user123", "gpt-4")
ch2, _ := d.SelectChannelWithDefault("user123", "gpt-4")
// ch1.ID == ch2.ID

// 检查用户绑定
channelID, ok := d.GetPinnedChannel("user123")

// 解除绑定
d.UnpinUser("user123")

// 统计绑定用户数
count := d.CountPinnedUsers()
\`\`\`

### 运行时控制

\`\`\`go
// 动态切换策略
d.SetDefaultStrategy(distributor.StrategyRandom)
strategy := d.GetDefaultStrategy()

// 动态启用/禁用用户亲和性
d.EnableUserAffinity(false)  // 禁用（清空所有绑定）
enabled := d.IsUserAffinityEnabled()

// 临时覆盖策略（不改变默认策略）
ch, _ := d.SelectChannel("user123", "gpt-4", distributor.StrategyRandom)
\`\`\`

### 健康检查

\`\`\`go
d := distributor.NewDistributor(distributor.Config{
    HealthCheckFunc: func(ctx context.Context, ch *distributor.Channel) error {
        // 发起健康检查请求
        resp, err := http.Get(ch.BaseURL + "/health")
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != 200 {
            return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
        }
        return nil
    },
    HealthCheckPeriod: 30 * time.Second,
})
\`\`\`

## API 文档

### Distributor

\`\`\`go
type Distributor struct {
    // 私有字段
}

// NewDistributor 创建渠道分发器
func NewDistributor(cfg Config) *Distributor

// SelectChannel 选择渠道（指定策略）
func (d *Distributor) SelectChannel(userID, model string, strategy Strategy) (*Channel, error)

// SelectChannelWithDefault 选择渠道（使用默认策略）
func (d *Distributor) SelectChannelWithDefault(userID, model string) (*Channel, error)

// SetDefaultStrategy 设置默认选择策略
func (d *Distributor) SetDefaultStrategy(strategy Strategy)

// GetDefaultStrategy 获取默认选择策略
func (d *Distributor) GetDefaultStrategy() Strategy

// EnableUserAffinity 启用/禁用用户亲和性
func (d *Distributor) EnableUserAffinity(enable bool)

// IsUserAffinityEnabled 检查用户亲和性是否启用
func (d *Distributor) IsUserAffinityEnabled() bool

// GetChannelByID 根据 ID 获取渠道
func (d *Distributor) GetChannelByID(id string) (*Channel, error)

// ListChannels 列出所有渠道
func (d *Distributor) ListChannels() []*Channel

// PinUserToChannel 将用户绑定到指定渠道
func (d *Distributor) PinUserToChannel(userID, channelID string)

// UnpinUser 解除用户绑定
func (d *Distributor) UnpinUser(userID string)

// GetPinnedChannel 获取用户绑定的渠道 ID
func (d *Distributor) GetPinnedChannel(userID string) (string, bool)

// CountPinnedUsers 统计绑定用户数
func (d *Distributor) CountPinnedUsers() int

// Stop 停止分发器
func (d *Distributor) Stop()
\`\`\`

### Config

\`\`\`go
type Config struct {
    Channels           []*Channel      // 渠道列表
    DefaultStrategy    Strategy        // 默认选择策略（默认 priority）
    EnableUserAffinity bool            // 是否启用用户亲和性（默认 false）
    HealthCheckFunc    HealthCheckFunc // 健康检查函数（可选）
    HealthCheckPeriod  time.Duration   // 健康检查周期（默认 30s）
}
\`\`\`

### Channel

\`\`\`go
type Channel struct {
    ID       string              // 渠道 ID
    Name     string              // 渠道名称
    Provider Provider            // 提供商
    BaseURL  string              // API 基础地址
    APIKey   string              // API 密钥
    Models   []string            // 支持的模型列表（空表示全部）
    Priority int                 // 优先级（越小越高）
    Weight   int                 // 权重（用于加权策略）
    Enabled  bool                // 是否启用
    Healthy  bool                // 是否健康
}

// IsHealthy 检查渠道是否健康
func (c *Channel) IsHealthy() bool

// SetHealthy 设置渠道健康状态
func (c *Channel) SetHealthy(healthy bool)

// SupportsModel 检查渠道是否支持指定模型
func (c *Channel) SupportsModel(model string) bool
\`\`\`

### Strategy

\`\`\`go
type Strategy string

const (
    StrategyPriority Strategy = "priority" // 优先级策略
    StrategyRandom   Strategy = "random"   // 随机策略
    StrategyWeighted Strategy = "weighted" // 加权随机策略
)
\`\`\`

### Provider

\`\`\`go
type Provider string

const (
    ProviderOpenAI   Provider = "openai"
    ProviderClaude   Provider = "claude"
    ProviderGemini   Provider = "gemini"
    ProviderDeepSeek Provider = "deepseek"
    ProviderDoubao   Provider = "doubao"
    ProviderCustom   Provider = "custom"
)
\`\`\`

## 使用场景

### 多渠道备份（主备模式）

\`\`\`go
d := distributor.NewDistributor(distributor.Config{
    Channels: []*distributor.Channel{
        {ID: "main", Priority: 1, Enabled: true, Healthy: true},
        {ID: "backup", Priority: 2, Enabled: true, Healthy: true},
    },
    DefaultStrategy:    distributor.StrategyPriority,
    EnableUserAffinity: false,
})
// 主渠道健康时总是选主渠道，主渠道故障自动切换备份
\`\`\`

### 负载均衡（按权重分配）

\`\`\`go
d := distributor.NewDistributor(distributor.Config{
    Channels: []*distributor.Channel{
        {ID: "ch1", Weight: 70, Enabled: true, Healthy: true},  // 70% 流量
        {ID: "ch2", Weight: 30, Enabled: true, Healthy: true},  // 30% 流量
    },
    DefaultStrategy:    distributor.StrategyWeighted,
    EnableUserAffinity: false,
})
\`\`\`

### AI 对话（会话连续性）

\`\`\`go
d := distributor.NewDistributor(distributor.Config{
    Channels: []*distributor.Channel{
        {ID: "ch1", Priority: 1, Enabled: true, Healthy: true},
        {ID: "ch2", Priority: 2, Enabled: true, Healthy: true},
    },
    DefaultStrategy:    distributor.StrategyPriority,
    EnableUserAffinity: true,  // 同一用户固定同一渠道
})
\`\`\`

### A/B 测试

\`\`\`go
d := distributor.NewDistributor(distributor.Config{
    Channels: []*distributor.Channel{
        {ID: "group-a", Enabled: true, Healthy: true},
        {ID: "group-b", Enabled: true, Healthy: true},
    },
    DefaultStrategy:    distributor.StrategyRandom,  // 均匀分配
    EnableUserAffinity: false,
})
\`\`\`

## 最佳实践

### 1. 优先级设置

\`\`\`go
// 主渠道设置最小 Priority
{ID: "main", Priority: 1}
// 备份渠道设置更大 Priority
{ID: "backup", Priority: 10}
\`\`\`

### 2. 权重分配

\`\`\`go
// 权重总和不需要等于 100
{ID: "ch1", Weight: 7},  // 70%
{ID: "ch2", Weight: 3},  // 30%
\`\`\`

### 3. 模型过滤

\`\`\`go
// 空列表：支持所有模型
{Models: []string{}}

// 指定模型：只支持特定模型
{Models: []string{"gpt-4", "gpt-3.5-turbo"}}

// 通配符：支持所有模型
{Models: []string{"*"}}
\`\`\`

### 4. 健康检查

\`\`\`go
// 推荐：实现幂等的健康检查接口
HealthCheckFunc: func(ctx context.Context, ch *distributor.Channel) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", ch.BaseURL+"/health", nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
    }
    return nil
}
\`\`\`

## 与 New-API 对比

| 特性 | New-API | go-guardian/distributor |
|------|---------|-------------------------|
| 渠道选择策略 | ✅ 内置多种 | ✅ 3 种核心策略 |
| 用户亲和性 | ✅ Pin 机制 | ✅ 可开关 Pin |
| 健康检查 | ✅ | ✅ 可配置周期 |
| 运行时控制 | ❌ | ✅ 动态切换策略 |
| 接口抽象 | ⚠️ 耦合业务 | ✅ 清晰接口 |

## 许可证

Apache-2.0 License - 详见 [LICENSE](../LICENSE)
