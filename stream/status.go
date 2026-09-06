// Package stream 流式处理基础设施
package stream

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// EndReason 流式结束原因
type EndReason string

const (
	EndReasonNone        EndReason = ""
	EndReasonDone        EndReason = "done"           // 正常完成
	EndReasonTimeout     EndReason = "timeout"        // 超时
	EndReasonClientGone  EndReason = "client_gone"    // 客户端断开
	EndReasonScannerErr  EndReason = "scanner_error"  // 扫描器错误
	EndReasonHandlerStop EndReason = "handler_stop"   // 处理器停止
	EndReasonEOF         EndReason = "eof"            // 流结束
	EndReasonPanic       EndReason = "panic"          // Panic
	EndReasonPingFail    EndReason = "ping_fail"      // 心跳失败
	EndReasonAgentError  EndReason = "agent_error"    // Agent 执行错误
)

const maxStreamErrorEntries = 20

// ErrorEntry 错误条目
type ErrorEntry struct {
	Message   string    // 错误消息
	Timestamp time.Time // 时间戳
}

// Status 流式状态跟踪
type Status struct {
	EndReason  EndReason    // 结束原因
	EndError   error        // 结束错误
	endOnce    sync.Once    // 保证只设置一次
	mu         sync.Mutex   // 保护并发访问
	Errors     []ErrorEntry // 错误列表（最多 20 条）
	ErrorCount int          // 总错误数
}

// NewStatus 创建流式状态
func NewStatus() *Status {
	return &Status{}
}

// SetEndReason 设置结束原因（只能设置一次）
func (s *Status) SetEndReason(reason EndReason, err error) {
	if s == nil {
		return
	}
	s.endOnce.Do(func() {
		s.EndReason = reason
		s.EndError = err
	})
}

// RecordError 记录错误（线程安全）
func (s *Status) RecordError(msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
	if len(s.Errors) < maxStreamErrorEntries {
		s.Errors = append(s.Errors, ErrorEntry{
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

// HasErrors 是否有错误
func (s *Status) HasErrors() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount > 0
}

// TotalErrorCount 总错误数
func (s *Status) TotalErrorCount() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount
}

// GetErrors 获取错误列表（副本）
func (s *Status) GetErrors() []ErrorEntry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ErrorEntry, len(s.Errors))
	copy(result, s.Errors)
	return result
}

// Summary 生成状态摘要
func (s *Status) Summary() string {
	if s == nil {
		return "stream status: nil"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("end_reason=%s", s.EndReason))

	if s.EndError != nil {
		sb.WriteString(fmt.Sprintf(", end_error=%s", s.EndError.Error()))
	}

	s.mu.Lock()
	errorCount := s.ErrorCount
	errorList := make([]ErrorEntry, len(s.Errors))
	copy(errorList, s.Errors)
	s.mu.Unlock()

	if errorCount > 0 {
		sb.WriteString(fmt.Sprintf(", total_errors=%d", errorCount))
		if len(errorList) > 0 {
			sb.WriteString(", recent_errors=[")
			for i, e := range errorList {
				if i > 0 {
					sb.WriteString("; ")
				}
				sb.WriteString(fmt.Sprintf("%s: %s", e.Timestamp.Format("15:04:05"), e.Message))
			}
			sb.WriteString("]")
		}
	}

	return sb.String()
}

// IsNormalEnd 是否正常结束
func (s *Status) IsNormalEnd() bool {
	if s == nil {
		return false
	}
	return s.EndReason == EndReasonDone || s.EndReason == EndReasonEOF
}
