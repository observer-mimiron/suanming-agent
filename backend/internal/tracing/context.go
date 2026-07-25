// Package tracing 暂与 tracing.go 共享包注释，本文件提供上下文中的 Trace 存取方法。

package tracing

import "context"

type ctxKey string

const traceKey ctxKey = "tracing.trace"

func expandTraceAttributeAliases(key string, value any) map[string]any {
	if key == "" {
		return nil
	}

	attrs := map[string]any{key: value}
	switch key {
	case "session_id":
		// Langfuse sessions are grouped from the top-level sessionId field, and the
		// native OTel integration maps this vendor-specific attribute into that field.
		attrs["langfuse.session.id"] = value
	case "user_id":
		attrs["langfuse.user.id"] = value
	case "input.value":
		attrs["langfuse.trace.input"] = value
	case "output.value":
		attrs["langfuse.trace.output"] = value
	}
	return attrs
}

// TraceFromContext 从 context 中取出 TurnTrace，不存在则返回 nil。
func TraceFromContext(ctx context.Context) *TurnTrace {
	if t, ok := ctx.Value(traceKey).(*TurnTrace); ok {
		return t
	}
	return nil
}

// TraceIDFromContext 从 context 中返回追踪 ID，不存在则返回空字符串。
func TraceIDFromContext(ctx context.Context) string {
	if t := TraceFromContext(ctx); t != nil {
		return t.TraceID
	}
	return ""
}

// SetTraceAttribute writes a top-level trace attribute when a TurnTrace is present.
func SetTraceAttribute(ctx context.Context, key string, value any) {
	t := TraceFromContext(ctx)
	if t == nil || key == "" {
		return
	}
	if t.Attributes == nil {
		t.Attributes = map[string]any{}
	}
	for attrKey, attrValue := range expandTraceAttributeAliases(key, value) {
		t.Attributes[attrKey] = attrValue
		if t.otelRoot != nil {
			t.otelRoot.SetAttribute(attrKey, attrValue)
		}
	}
}

// SetTraceAttributes writes multiple top-level trace attributes when a TurnTrace is present.
func SetTraceAttributes(ctx context.Context, attrs map[string]any) {
	for k, v := range attrs {
		SetTraceAttribute(ctx, k, v)
	}
}

// contextWithTrace 将 TurnTrace 存入 context，供下游代码通过 TraceFromContext 取回。
func contextWithTrace(ctx context.Context, t *TurnTrace) context.Context {
	return context.WithValue(ctx, traceKey, t)
}
