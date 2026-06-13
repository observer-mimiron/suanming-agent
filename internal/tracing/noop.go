package tracing

import "context"

type noopSpan struct{}

func (noopSpan) End()                        {}
func (noopSpan) SetKind(_ SpanKind)          {}
func (noopSpan) SetStatus(_ string)          {}
func (noopSpan) SetAttribute(_ string, _ any) {}
func (noopSpan) RecordError(_ error)         {}

type noopTrace struct{}

func (noopTrace) StartSpan(_ string) Span { return noopSpan{} }
func (noopTrace) SetStatus(_ string)      {}
func (noopTrace) End()                    {}

type noopTracer struct{}

func (noopTracer) StartTrace(ctx context.Context, _ string) (context.Context, Trace) {
	return ctx, noopTrace{}
}

// NewNoopTracer returns a Tracer whose methods are no-ops.
// Always returns non-nil handles so callers never need nil checks.
func NewNoopTracer() Tracer {
	return noopTracer{}
}
