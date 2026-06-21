package tracing

import "context"

type otelBridge interface {
	StartRoot(ctx context.Context, name string, kind SpanKind) (context.Context, otelSpanBridge)
	StartChild(ctx context.Context, name string, kind SpanKind) otelSpanBridge
}

type otelSpanBridge interface {
	SetAttribute(key string, value any)
	SetStatus(status string)
	RecordError(err error)
	End()
}
