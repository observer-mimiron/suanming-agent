// Package tracing 提供可观测性追踪基础设施，包含 Span、Trace、Tracer 三个核心接口，以及 OpenInference 语义的 Span 类型常量。
// 支持 Gin 中间件自动埋点、Eino 回调钩子接入、以及可选的 JSON 文件持久化。

package tracing

import "context"

// Span 表示追踪中的一个命名工作单元。
type Span interface {
	End()
	SetKind(kind SpanKind)
	SetStatus(status string)
	SetAttribute(key string, value any)
	RecordError(err error)
}

// Trace 表示一个被追踪操作的根节点。
type Trace interface {
	StartSpan(name string) Span
	SetStatus(status string)
	End()
}

// Tracer 是创建 Trace 的工厂接口。
type Tracer interface {
	StartTrace(ctx context.Context, name string) (context.Context, Trace)
}
