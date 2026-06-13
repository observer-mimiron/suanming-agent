package tracing

import "context"

// Span represents a named unit of work within a trace.
type Span interface {
	End()
	SetKind(kind SpanKind)
	SetStatus(status string)
	SetAttribute(key string, value any)
	RecordError(err error)
}

// Trace represents the root of a traced operation.
type Trace interface {
	StartSpan(name string) Span
	SetStatus(status string)
	End()
}

// Tracer is the factory for creating traces.
type Tracer interface {
	StartTrace(ctx context.Context, name string) (context.Context, Trace)
}
