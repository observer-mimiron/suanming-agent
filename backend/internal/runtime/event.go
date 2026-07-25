package runtime

import "context"

// Event 表示编排过程中发出的单个事件。
type Event struct {
	Type string
	Data any
}

// EventSink 接收来自编排器的事件。
// 处理器层将 SSE/HTTP 适配到此接口。
type EventSink interface {
	Emit(ctx context.Context, evt Event) error
}
