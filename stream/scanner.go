package stream

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// InitialScannerBufferSize 初始缓冲区大小（64KB）
	InitialScannerBufferSize = 64 << 10
	// MaxScannerBufferSize 最大缓冲区大小（128MB）
	MaxScannerBufferSize = 128 << 20
	// DefaultStreamingTimeout 默认流式超时
	DefaultStreamingTimeout = 30 * time.Second
	// DefaultPingInterval 默认心跳间隔
	DefaultPingInterval = 10 * time.Second
	// StreamWriteTimeout 单次写入超时
	StreamWriteTimeout = 30 * time.Second
	// MaxPingDuration 最大心跳持续时间
	MaxPingDuration = 30 * time.Minute
)

// DataHandler 数据处理回调
// data: 当前行数据
// result: 用于记录错误、停止流或标记完成
type DataHandler func(data string, result *Result)

// ScannerConfig StreamScanner 配置
type ScannerConfig struct {
	Logger            *zap.SugaredLogger // 日志
	StreamingTimeout  time.Duration      // 流式超时
	PingInterval      time.Duration      // 心跳间隔
	PingEnabled       bool               // 是否启用心跳
	MaxBufferSize     int                // 最大缓冲区大小
	CopyResponseHeader func(*gin.Context, *http.Response) // 复制响应头回调（可选）
}

// DefaultScannerConfig 默认配置
func DefaultScannerConfig(logger *zap.SugaredLogger) ScannerConfig {
	return ScannerConfig{
		Logger:           logger,
		StreamingTimeout: DefaultStreamingTimeout,
		PingInterval:     DefaultPingInterval,
		PingEnabled:      true,
		MaxBufferSize:    MaxScannerBufferSize,
	}
}

// StreamScanner 并发安全的流式扫描器
type StreamScanner struct {
	config      ScannerConfig
	status      *Status
	ctx         context.Context
	cancel      context.CancelFunc
	stopChan    chan bool
	stopOnce    sync.Once
	cleanupOnce sync.Once
	wg          sync.WaitGroup
	writeMutex  sync.Mutex
}

// NewStreamScanner 创建流式扫描器
func NewStreamScanner(config ScannerConfig) *StreamScanner {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamScanner{
		config:   config,
		status:   NewStatus(),
		ctx:      ctx,
		cancel:   cancel,
		stopChan: make(chan bool, 3),
	}
}

// Scan 执行流式扫描
// c: Gin 上下文
// resp: HTTP 响应
// dataHandler: 数据处理回调
func (s *StreamScanner) Scan(c *gin.Context, resp *http.Response, dataHandler DataHandler) {
	if resp == nil || dataHandler == nil {
		s.config.Logger.Errorw("StreamScanner.Scan: resp or dataHandler is nil")
		return
	}

	defer s.cleanup(resp)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 复制自定义响应头
	if s.config.CopyResponseHeader != nil {
		s.config.CopyResponseHeader(c, resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, InitialScannerBufferSize), s.config.MaxBufferSize)
	scanner.Split(bufio.ScanLines)

	ticker := time.NewTicker(s.config.StreamingTimeout)
	defer ticker.Stop()

	var pingTicker *time.Ticker
	if s.config.PingEnabled {
		pingTicker = time.NewTicker(s.config.PingInterval)
		defer pingTicker.Stop()
	}

	s.config.Logger.Debugw("StreamScanner started",
		"streaming_timeout", s.config.StreamingTimeout,
		"ping_enabled", s.config.PingEnabled,
		"ping_interval", s.config.PingInterval,
	)

	// 启动心跳 goroutine
	if s.config.PingEnabled && pingTicker != nil {
		s.wg.Add(1)
		gopool.Go(func() {
			defer s.wg.Done()
			s.pingLoop(c, pingTicker)
		})
	}

	// 启动数据处理 goroutine
	dataChan := make(chan string, 10)
	s.wg.Add(1)
	gopool.Go(func() {
		defer s.wg.Done()
		s.dataLoop(c, dataChan, dataHandler)
	})

	// 主扫描循环
	for scanner.Scan() {
		select {
		case <-s.stopChan:
			s.config.Logger.Debugw("StreamScanner stopped by stopChan")
			return
		case <-c.Request.Context().Done():
			s.config.Logger.Debugw("StreamScanner stopped by client disconnect")
			s.status.SetEndReason(EndReasonClientGone, c.Request.Context().Err())
			return
		case dataChan <- scanner.Text():
			ticker.Reset(s.config.StreamingTimeout)
		}
	}

	// 扫描结束或错误
	if err := scanner.Err(); err != nil {
		s.config.Logger.Errorw("StreamScanner error", "error", err)
		s.status.SetEndReason(EndReasonScannerErr, err)
	} else {
		s.status.SetEndReason(EndReasonEOF, nil)
	}

	close(dataChan)
}

// pingLoop 心跳循环
func (s *StreamScanner) pingLoop(c *gin.Context, pingTicker *time.Ticker) {
	defer func() {
		if r := recover(); r != nil {
			s.config.Logger.Errorw("ping goroutine panic", "panic", r)
			s.status.SetEndReason(EndReasonPanic, fmt.Errorf("ping panic: %v", r))
			s.stop()
		}
		s.config.Logger.Debugw("ping goroutine exited")
	}()

	pingTimeout := time.NewTimer(MaxPingDuration)
	defer pingTimeout.Stop()

	for {
		select {
		case <-pingTicker.C:
			var err error
			func() {
				s.writeMutex.Lock()
				defer s.writeMutex.Unlock()
				s.extendWriteDeadline(c)
				err = s.writePing(c)
			}()
			if err != nil {
				s.config.Logger.Errorw("ping data error", "error", err)
				s.status.SetEndReason(EndReasonPingFail, err)
				return
			}
			s.config.Logger.Debugw("ping data sent")

		case <-s.ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-c.Request.Context().Done():
			return
		case <-pingTimeout.C:
			s.config.Logger.Errorw("ping goroutine max duration reached")
			return
		}
	}
}

// dataLoop 数据处理循环
func (s *StreamScanner) dataLoop(c *gin.Context, dataChan <-chan string, dataHandler DataHandler) {
	defer func() {
		if r := recover(); r != nil {
			s.config.Logger.Errorw("data handler goroutine panic", "panic", r)
			s.status.SetEndReason(EndReasonPanic, fmt.Errorf("handler panic: %v", r))
		}
		s.stop()
	}()

	result := NewResult(s.status)
	for data := range dataChan {
		result.Reset()
		func() {
			s.writeMutex.Lock()
			defer s.writeMutex.Unlock()
			s.extendWriteDeadline(c)
			dataHandler(data, result)
		}()

		if result.IsStopped() {
			s.config.Logger.Debugw("data handler stopped stream")
			return
		}
	}
	s.config.Logger.Debugw("data handler goroutine exited")
}

// stop 停止扫描器
func (s *StreamScanner) stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
}

// cleanup 清理资源
func (s *StreamScanner) cleanup(resp *http.Response) {
	s.cleanupOnce.Do(func() {
		s.cancel()
		s.stop()
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
		s.wg.Wait()
		s.config.Logger.Debugw("StreamScanner cleanup complete", "status", s.status.Summary())
	})
}

// extendWriteDeadline 延长写入截止时间
func (s *StreamScanner) extendWriteDeadline(c *gin.Context) {
	if c == nil || c.Writer == nil {
		return
	}
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(StreamWriteTimeout))
}

// writePing 写入心跳数据
func (s *StreamScanner) writePing(c *gin.Context) error {
	if c == nil || c.Writer == nil {
		return fmt.Errorf("context or writer is nil")
	}
	if c.Request.Context().Err() != nil {
		return fmt.Errorf("request context done: %w", c.Request.Context().Err())
	}
	_, err := c.Writer.Write([]byte(": PING\n\n"))
	if err != nil {
		return fmt.Errorf("write ping data failed: %w", err)
	}
	c.Writer.Flush()
	return nil
}

// Status 获取流式状态
func (s *StreamScanner) Status() *Status {
	return s.status
}
