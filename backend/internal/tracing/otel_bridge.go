// This file belongs to the trace projection layer.
// It owns OpenTelemetry bridge behavior for this package.
// It projects runtime evidence; it must not change execution decisions.
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
