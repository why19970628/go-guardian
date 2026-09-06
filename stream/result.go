package stream

// Result 流式处理结果
// 提供给 dataHandler 回调，用于记录错误、停止流或标记完成
type Result struct {
	status  *Status
	stopped bool
}

// NewResult 创建流式结果
func NewResult(status *Status) *Result {
	return &Result{status: status}
}

// Error 记录软错误（流继续处理）
// 可在同一 chunk 内多次调用
func (r *Result) Error(err error) {
	if err == nil {
		return
	}
	r.status.RecordError(err.Error())
}

// Stop 记录致命错误并标记流在此 chunk 后停止
func (r *Result) Stop(err error) {
	if err != nil {
		r.status.RecordError(err.Error())
	}
	r.status.SetEndReason(EndReasonHandlerStop, err)
	r.stopped = true
}

// Done 标记处理正常完成（流在此 chunk 后停止）
// 例如：Dify "message_end" 事件
func (r *Result) Done() {
	r.status.SetEndReason(EndReasonDone, nil)
	r.stopped = true
}

// IsStopped 是否已停止
func (r *Result) IsStopped() bool {
	return r.stopped
}

// Reset 重置停止标志（供下一次 chunk 复用）
func (r *Result) Reset() {
	r.stopped = false
}
