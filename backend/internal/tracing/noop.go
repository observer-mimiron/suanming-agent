// Package tracing 暂与 tracing.go 共享包注释，本文件提供空操作 Tracer 实现，使调用方无需空值检查。

package tracing

import "context"

type noopSpan struct{}

func (noopSpan) End()                         {}
func (noopSpan) SetKind(_ SpanKind)           {}
func (noopSpan) SetStatus(_ string)           {}
func (noopSpan) SetAttribute(_ string, _ any) {}
func (noopSpan) RecordError(_ error)          {}

type noopTrace struct{}

func (noopTrace) StartSpan(_ string) Span { return noopSpan{} }
func (noopTrace) SetStatus(_ string)      {}
func (noopTrace) End()                    {}

type noopTracer struct{}

func (noopTracer) StartTrace(ctx context.Context, _ string) (context.Context, Trace) {
	return ctx, noopTrace{}
}
