// Package distributor 多 LLM 提供商渠道分发器
//
// 提供智能渠道选择、用户亲和性、健康检查等功能
package distributor

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Provider LLM 提供商
type Provider string

const (
	ProviderOpenAI   Provider = "openai"
	ProviderClaude   Provider = "claude"
	ProviderGemini   Provider = "gemini"
	ProviderDeepSeek Provider = "deepseek"
	ProviderDoubao   Provider = "doubao"
	ProviderCustom   Provider = "custom"
)

// Channel 渠道配置
type Channel struct {
	ID       string   // 渠道 ID
	Name     string   // 渠道名称
	Provider Provider // 提供商
	BaseURL  string   // API 基础地址
	APIKey   string   // API 密钥
	Models   []string // 支持的模型列表（空表示全部）
	Priority int      // 优先级（越小越高，用于 priority 策略）
	Weight   int      // 权重（用于 weighted 策略）
	Enabled  bool     // 是否启用
	Healthy  bool     // 是否健康（健康检查更新）
	mu       sync.RWMutex
}

// IsHealthy 检查渠道是否健康
func (c *Channel) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Enabled && c.Healthy
}

// SetHealthy 设置渠道健康状态
func (c *Channel) SetHealthy(healthy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Healthy = healthy
}

// SupportsModel 检查渠道是否支持指定模型
func (c *Channel) SupportsModel(model string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.Models) == 0 {
		return true // 空列表表示支持所有模型
	}
	for _, m := range c.Models {
		if m == model || m == "*" {
			return true
		}
	}
	return false
}

// Strategy 选择策略
type Strategy string

const (
	StrategyPriority Strategy = "priority" // 优先级策略（选择优先级最高的）
	StrategyRandom   Strategy = "random"   // 随机策略
	StrategyWeighted Strategy = "weighted" // 加权随机策略
)

// Distributor 渠道分发器
type Distributor struct {
	channels           []*Channel
	defaultStrategy    Strategy          // 默认选择策略
	enableUserAffinity bool              // 是否启用用户亲和性
	pinnedChan         map[string]string // key: userID, value: channelID（用户亲和性）
	mu                 sync.RWMutex
	rand               *rand.Rand
	healthCheck        HealthCheckFunc
	healthTicker       *time.Ticker
	stopChan           chan struct{}
	wg                 sync.WaitGroup
}

// HealthCheckFunc 健康检查函数
type HealthCheckFunc func(ctx context.Context, channel *Channel) error

// Config 分发器配置
type Config struct {
	Channels           []*Channel      // 渠道列表
	DefaultStrategy    Strategy        // 默认选择策略（默认 priority）
	EnableUserAffinity bool            // 是否启用用户亲和性（默认 false）
	HealthCheckFunc    HealthCheckFunc // 健康检查函数（可选）
	HealthCheckPeriod  time.Duration   // 健康检查周期（默认 30s）
}

// NewDistributor 创建渠道分发器
func NewDistributor(cfg Config) *Distributor {
	if cfg.HealthCheckPeriod == 0 {
		cfg.HealthCheckPeriod = 30 * time.Second
	}
	if cfg.DefaultStrategy == "" {
		cfg.DefaultStrategy = StrategyPriority
	}

	d := &Distributor{
		channels:           cfg.Channels,
		defaultStrategy:    cfg.DefaultStrategy,
		enableUserAffinity: cfg.EnableUserAffinity,
		pinnedChan:         make(map[string]string),
		rand:               rand.New(rand.NewSource(time.Now().UnixNano())),
		healthCheck:        cfg.HealthCheckFunc,
		stopChan:           make(chan struct{}),
	}

	// 启动健康检查
	if d.healthCheck != nil {
		d.healthTicker = time.NewTicker(cfg.HealthCheckPeriod)
		d.wg.Add(1)
		go d.healthCheckLoop()
	}

	return d
}

// SelectChannel 选择渠道
// userID: 用户 ID（可选，用于亲和性）
// model: 模型名称
// strategy: 选择策略（传空字符串使用默认策略）
func (d *Distributor) SelectChannel(userID, model string, strategy Strategy) (*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 0. 使用默认策略（如果未指定）
	if strategy == "" {
		strategy = d.defaultStrategy
	}

	// 1. 检查用户亲和性
	if d.enableUserAffinity && userID != "" {
		if pinnedID, ok := d.pinnedChan[userID]; ok {
			for _, ch := range d.channels {
				if ch.ID == pinnedID && ch.IsHealthy() && ch.SupportsModel(model) {
					return ch, nil
				}
			}
		}
	}

	// 2. 过滤可用渠道
	var candidates []*Channel
	for _, ch := range d.channels {
		if ch.IsHealthy() && ch.SupportsModel(model) {
			candidates = append(candidates, ch)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available channel for model: %s", model)
	}

	// 3. 根据策略选择渠道
	var selected *Channel
	switch strategy {
	case StrategyPriority:
		selected = d.selectByPriority(candidates)
	case StrategyRandom:
		selected = d.selectByRandom(candidates)
	case StrategyWeighted:
		selected = d.selectByWeight(candidates)
	default:
		selected = d.selectByPriority(candidates)
	}

	// 4. 记录用户亲和性
	if d.enableUserAffinity && userID != "" && selected != nil {
		d.pinnedChan[userID] = selected.ID
	}

	return selected, nil
}

// SelectChannelWithDefault 使用默认策略选择渠道（简化调用）
func (d *Distributor) SelectChannelWithDefault(userID, model string) (*Channel, error) {
	return d.SelectChannel(userID, model, "")
}

// SetDefaultStrategy 设置默认选择策略（运行时可调整）
func (d *Distributor) SetDefaultStrategy(strategy Strategy) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.defaultStrategy = strategy
}

// GetDefaultStrategy 获取默认选择策略
func (d *Distributor) GetDefaultStrategy() Strategy {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.defaultStrategy
}

// EnableUserAffinity 启用/禁用用户亲和性（运行时可调整）
func (d *Distributor) EnableUserAffinity(enable bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enableUserAffinity = enable
	if !enable {
		// 禁用时清空亲和性记录
		d.pinnedChan = make(map[string]string)
	}
}

// IsUserAffinityEnabled 检查用户亲和性是否启用
func (d *Distributor) IsUserAffinityEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enableUserAffinity
}

// selectByPriority 按优先级选择（优先级最高的）
func (d *Distributor) selectByPriority(candidates []*Channel) *Channel {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	for _, ch := range candidates[1:] {
		if ch.Priority < best.Priority {
			best = ch
		}
	}
	return best
}

// selectByRandom 随机选择
func (d *Distributor) selectByRandom(candidates []*Channel) *Channel {
	if len(candidates) == 0 {
		return nil
	}
	return candidates[d.rand.Intn(len(candidates))]
}

// selectByWeight 加权随机选择
func (d *Distributor) selectByWeight(candidates []*Channel) *Channel {
	if len(candidates) == 0 {
		return nil
	}

	totalWeight := 0
	for _, ch := range candidates {
		totalWeight += ch.Weight
	}

	if totalWeight == 0 {
		return d.selectByRandom(candidates)
	}

	r := d.rand.Intn(totalWeight)
	for _, ch := range candidates {
		r -= ch.Weight
		if r < 0 {
			return ch
		}
	}

	return candidates[len(candidates)-1]
}

// healthCheckLoop 健康检查循环
func (d *Distributor) healthCheckLoop() {
	defer d.wg.Done()

	for {
		select {
		case <-d.healthTicker.C:
			d.runHealthCheck()
		case <-d.stopChan:
			return
		}
	}
}

// runHealthCheck 执行健康检查
func (d *Distributor) runHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	d.mu.RLock()
	channels := make([]*Channel, len(d.channels))
	copy(channels, d.channels)
	d.mu.RUnlock()

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}

		err := d.healthCheck(ctx, ch)
		ch.SetHealthy(err == nil)
	}
}

// Stop 停止分发器
func (d *Distributor) Stop() {
	close(d.stopChan)
	if d.healthTicker != nil {
		d.healthTicker.Stop()
	}
	d.wg.Wait()
}

// GetChannelByID 根据 ID 获取渠道
func (d *Distributor) GetChannelByID(id string) (*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, ch := range d.channels {
		if ch.ID == id {
			return ch, nil
		}
	}
	return nil, fmt.Errorf("channel not found: %s", id)
}

// ListChannels 列出所有渠道
func (d *Distributor) ListChannels() []*Channel {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*Channel, len(d.channels))
	copy(result, d.channels)
	return result
}

// PinUserToChannel 将用户绑定到指定渠道（用户亲和性）
func (d *Distributor) PinUserToChannel(userID, channelID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pinnedChan[userID] = channelID
}

// UnpinUser 解除用户绑定
func (d *Distributor) UnpinUser(userID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.pinnedChan, userID)
}

// GetPinnedChannel 获取用户绑定的渠道 ID
func (d *Distributor) GetPinnedChannel(userID string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	channelID, ok := d.pinnedChan[userID]
	return channelID, ok
}

// CountPinnedUsers 统计绑定用户数
func (d *Distributor) CountPinnedUsers() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.pinnedChan)
}
