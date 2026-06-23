package tracing

import "strconv"

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
// 连续的 sse_emit 事件会被合并为一条摘要，避免上百条噪音淹没排障信息。
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
	steps = compactSSEEmits(steps)
	return DebugTraceDigest{
		TraceID:  t.TraceID,
		TurnType: t.TurnType,
		TotalMs:  traceTotalMs(t),
		Status:   normalizeStatus(t.Status),
		Steps:    steps,
	}
}

// compactSSEEmits 将连续的 sse_emit 步骤合并为一条摘要条目。
// 例如 315 条 "SSE 输出" → 1 条 "SSE 输出 (text×315, tool_call×16, thinking×2, component×7, done×1)"
func compactSSEEmits(steps []DebugTraceStep) []DebugTraceStep {
	out := make([]DebugTraceStep, 0, len(steps))
	var emitRun []DebugTraceStep

	flush := func() {
		if len(emitRun) == 0 {
			return
		}
		if len(emitRun) == 1 {
			out = append(out, emitRun[0])
			emitRun = nil
			return
		}
		counts := map[string]int{}
		for _, s := range emitRun {
			if et, ok := s.Meta["event_type"].(string); ok {
				counts[et]++
			}
		}
		// 构建摘要标签
		label := "SSE 输出 ("
		first := true
		for et, n := range counts {
			if !first {
				label += ", "
			}
			label += et + "×" + strconv.Itoa(n)
			first = false
		}
		label += ")"
		out = append(out, DebugTraceStep{
			Name:   "sse_emit_batch",
			Label:  label,
			Kind:   KindChain,
			Status: "ok",
			Meta: map[string]any{
				"batch_count": len(emitRun),
				"breakdown":   counts,
			},
		})
		emitRun = nil
	}

	for _, s := range steps {
		if s.Name == "sse_emit" {
			emitRun = append(emitRun, s)
		} else {
			flush()
			out = append(out, s)
		}
	}
	flush()
	return out
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
