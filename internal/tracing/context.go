// Package tracing 暂与 tracing.go 共享包注释，本文件提供上下文中的 Trace 存取方法。

package tracing

import "context"

type ctxKey string

const traceKey ctxKey = "tracing.trace"

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

// contextWithTrace 将 TurnTrace 存入 context，供下游代码通过 TraceFromContext 取回。
func contextWithTrace(ctx context.Context, t *TurnTrace) context.Context {
	return context.WithValue(ctx, traceKey, t)
}
