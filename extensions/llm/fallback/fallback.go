// Package fallback provides LLM-specific fallback strategies for graceful degradation.
//
// When LLM calls fail, these strategies provide alternative responses:
//   - CannedResponse: Fixed template response
//   - KeywordMatch: Rule-based responses for common queries
//
// Part of go-guardian extensions (optional, LLM-specific).
package fallback

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// Strategy 降级策略接口
type Strategy interface {
	// Execute 执行降级逻辑
	Execute(ctx context.Context, req *Request) (*schema.Message, error)
}

// Request 降级请求
type Request struct {
	UserQuery string   // 用户原始查询
	Brand     string   // 车型品牌
	Series    string   // 车型系列
	LastError error    // 上游失败的最后错误
}

// CannedResponse 固定话术降级策略（最简单兜底）
type CannedResponse struct {
	template string
}

// NewCannedResponse 创建固定话术降级
func NewCannedResponse() *CannedResponse {
	return &CannedResponse{
		template: `抱歉，AI 技师暂时无法回答您的问题。

您可以：
1. 稍后重试
2. 联系人工技师获取专业帮助
3. 前往汽车大师 App 提问

我们会尽快恢复服务，感谢您的理解。`,
	}
}

// Execute 返回固定话术
func (c *CannedResponse) Execute(_ context.Context, req *Request) (*schema.Message, error) {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: c.template,
	}, nil
}

// KeywordMatch 关键词匹配降级策略（简单规则兜底）
type KeywordMatch struct {
	rules map[string]string // 关键词 -> 固定回复
}

// NewKeywordMatch 创建关键词匹配降级
func NewKeywordMatch() *KeywordMatch {
	return &KeywordMatch{
		rules: map[string]string{
			"故障码": "您提到了故障码问题。建议您：\n1. 使用 OBD 诊断仪读取完整故障码\n2. 记录故障码后联系 4S 店或专业维修厂\n3. 不要自行清除故障码，可能影响后续诊断",
			"异响":  "车辆异响需要现场判断。建议您：\n1. 记录异响出现的时机（启动/行驶/制动等）\n2. 注意异响部位（发动机舱/底盘/车内等）\n3. 尽快到维修厂现场检查",
			"打不着": "启动困难常见原因：\n1. 电瓶亏电（检查电压是否正常）\n2. 燃油系统故障（检查油泵/油压）\n3. 点火系统故障（检查火花塞/点火线圈）\n建议先检查电瓶，必要时联系救援",
			"漏油":  "漏油问题需立即处理。建议您：\n1. 停车检查漏油位置和程度\n2. 不要继续长距离行驶\n3. 联系维修厂或救援拖车\n漏油可能导致严重故障，请尽快处理",
		},
	}
}

// Execute 匹配关键词返回规则回复，无匹配则返回通用话术
func (k *KeywordMatch) Execute(_ context.Context, req *Request) (*schema.Message, error) {
	query := req.UserQuery
	for keyword, reply := range k.rules {
		if contains(query, keyword) {
			return &schema.Message{
				Role:    schema.Assistant,
				Content: reply,
			}, nil
		}
	}
	// 无匹配，返回通用话术
	return NewCannedResponse().Execute(context.Background(), req)
}

// contains 简单子串匹配（生产环境可换更强的分词 + 相似度）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// NoopStrategy 空降级策略，直接返回错误
type NoopStrategy struct{}

func (NoopStrategy) Execute(_ context.Context, req *Request) (*schema.Message, error) {
	return nil, fmt.Errorf("降级策略未启用: %w", req.LastError)
}
