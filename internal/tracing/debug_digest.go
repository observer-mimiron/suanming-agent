package tracing

// DebugTraceDigest 是调试面板使用的原始 trace 投影。
// 它保留所有 span，包括 AGENT 根 span 与 `sse_emit` 一类低层事件。
type DebugTraceDigest struct {
	TraceID  string           `json:"trace_id"`
	TurnType string           `json:"turn_type"`
	TotalMs  int64            `json:"total_ms"`
	Status   string           `json:"status"`
	Steps    []DebugTraceStep `json:"steps"`
}

// DebugTraceStep 是调试轨迹中的一个原始步骤。
type DebugTraceStep struct {
	Name   string         `json:"name"`
	Label  string         `json:"label"`
	Kind   SpanKind       `json:"kind"`
	Status string         `json:"status"`
	Ms     int64          `json:"ms"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// BuildDebugDigest 从 TurnTrace 生成调试视图需要的原始步骤序列。
func (t *TurnTrace) BuildDebugDigest() DebugTraceDigest {
	steps := make([]DebugTraceStep, 0, len(t.Spans))
	for _, s := range t.Spans {
		steps = append(steps, DebugTraceStep{
			Name:   s.Name,
			Label:  stepLabel(s.Name),
			Kind:   s.Kind,
			Status: normalizeStatus(s.Status),
			Ms:     s.DurationMs,
			Meta:   cloneAttrs(s.Attributes),
		})
	}
	return DebugTraceDigest{
		TraceID:  t.TraceID,
		TurnType: t.TurnType,
		TotalMs:  traceTotalMs(t),
		Status:   normalizeStatus(t.Status),
		Steps:    steps,
	}
}

func cloneAttrs(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
