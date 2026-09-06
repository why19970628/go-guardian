// Package adaptor 协议转换器 - 支持多种 LLM 协议互转
package adaptor

import (
	"encoding/json"
	"fmt"
)

// Protocol LLM 协议类型
type Protocol string

const (
	ProtocolOpenAI Protocol = "openai"
	ProtocolClaude Protocol = "claude"
	ProtocolGemini Protocol = "gemini"
)

// Message 通用消息格式
type Message struct {
	Role    string `json:"role"`    // user / assistant / system
	Content string `json:"content"` // 消息内容
}

// ChatRequest 通用聊天请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse 通用聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content string `json:"content"`
	Usage   Usage  `json:"usage,omitempty"`
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Adaptor 协议转换器接口
type Adaptor interface {
	// ConvertRequest 转换请求（通用格式 -> 目标协议）
	ConvertRequest(req ChatRequest) ([]byte, error)

	// ConvertResponse 转换响应（目标协议 -> 通用格式）
	ConvertResponse(data []byte) (ChatResponse, error)

	// Protocol 返回目标协议类型
	Protocol() Protocol
}

// NewAdaptor 创建协议转换器
func NewAdaptor(protocol Protocol) (Adaptor, error) {
	switch protocol {
	case ProtocolOpenAI:
		return &OpenAIAdaptor{}, nil
	case ProtocolClaude:
		return &ClaudeAdaptor{}, nil
	case ProtocolGemini:
		return &GeminiAdaptor{}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// OpenAIAdaptor OpenAI 协议转换器
type OpenAIAdaptor struct{}

func (a *OpenAIAdaptor) Protocol() Protocol {
	return ProtocolOpenAI
}

func (a *OpenAIAdaptor) ConvertRequest(req ChatRequest) ([]byte, error) {
	// OpenAI 格式即通用格式，直接序列化
	return json.Marshal(req)
}

func (a *OpenAIAdaptor) ConvertResponse(data []byte) (ChatResponse, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return ChatResponse{}, fmt.Errorf("failed to unmarshal openai response: %w", err)
	}

	result := ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: resp.Usage,
	}

	if len(resp.Choices) > 0 {
		result.Content = resp.Choices[0].Message.Content
	}

	return result, nil
}

// ClaudeAdaptor Claude 协议转换器
type ClaudeAdaptor struct{}

func (a *ClaudeAdaptor) Protocol() Protocol {
	return ProtocolClaude
}

func (a *ClaudeAdaptor) ConvertRequest(req ChatRequest) ([]byte, error) {
	// Claude 格式转换
	claudeReq := map[string]interface{}{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"messages":    req.Messages,
		"stream":      req.Stream,
	}

	if req.Temperature > 0 {
		claudeReq["temperature"] = req.Temperature
	}

	return json.Marshal(claudeReq)
}

func (a *ClaudeAdaptor) ConvertResponse(data []byte) (ChatResponse, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return ChatResponse{}, fmt.Errorf("failed to unmarshal claude response: %w", err)
	}

	result := ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// 合并 content 块
	for _, block := range resp.Content {
		if block.Type == "text" {
			result.Content += block.Text
		}
	}

	return result, nil
}

// GeminiAdaptor Gemini 协议转换器
type GeminiAdaptor struct{}

func (a *GeminiAdaptor) Protocol() Protocol {
	return ProtocolGemini
}

func (a *GeminiAdaptor) ConvertRequest(req ChatRequest) ([]byte, error) {
	// Gemini 格式转换
	var contents []map[string]interface{}
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]string{
				{"text": msg.Content},
			},
		})
	}

	geminiReq := map[string]interface{}{
		"contents": contents,
	}

	if req.Temperature > 0 || req.MaxTokens > 0 {
		config := map[string]interface{}{}
		if req.Temperature > 0 {
			config["temperature"] = req.Temperature
		}
		if req.MaxTokens > 0 {
			config["maxOutputTokens"] = req.MaxTokens
		}
		geminiReq["generationConfig"] = config
	}

	return json.Marshal(geminiReq)
}

func (a *GeminiAdaptor) ConvertResponse(data []byte) (ChatResponse, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return ChatResponse{}, fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	result := ChatResponse{
		Model: "gemini",
		Usage: Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}

	// 合并 content 块
	if len(resp.Candidates) > 0 {
		for _, part := range resp.Candidates[0].Content.Parts {
			result.Content += part.Text
		}
	}

	return result, nil
}
