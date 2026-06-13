package tracing

import "context"

type ctxKey string

const traceKey ctxKey = "tracing.trace"

// TraceFromContext returns the TurnTrace stored in context, or nil.
func TraceFromContext(ctx context.Context) *TurnTrace {
	if t, ok := ctx.Value(traceKey).(*TurnTrace); ok {
		return t
	}
	return nil
}

// TraceIDFromContext returns the trace ID from context, or "".
func TraceIDFromContext(ctx context.Context) string {
	if t := TraceFromContext(ctx); t != nil {
		return t.TraceID
	}
	return ""
}

// contextWithTrace stores a TurnTrace in the context.
func contextWithTrace(ctx context.Context, t *TurnTrace) context.Context {
	return context.WithValue(ctx, traceKey, t)
}
