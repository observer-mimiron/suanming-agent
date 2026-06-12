package tracing

import "time"

// SpanKind follows OpenInference semantic conventions.
type SpanKind string

const (
	KindAgent     SpanKind = "AGENT"
	KindChain     SpanKind = "CHAIN"
	KindTool      SpanKind = "TOOL"
	KindRetriever SpanKind = "RETRIEVER"
	KindLLM       SpanKind = "LLM"
)

// TraceSpan represents a single unit of work within a trace.
type TraceSpan struct {
	SpanID        string         `json:"span_id"`
	ParentSpanID  string         `json:"parent_span_id,omitempty"`
	Name          string         `json:"name"`
	Kind          SpanKind       `json:"kind"`
	Status        string         `json:"status"` // "ok" | "degraded" | "fallback" | "error"
	StartedAt     time.Time      `json:"started_at"`
	EndedAt       time.Time      `json:"ended_at"`
	DurationMs    int64          `json:"duration_ms"`
	InputPreview  any            `json:"input_preview,omitempty"`
	OutputPreview any            `json:"output_preview,omitempty"`
	Error         string         `json:"error,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

// TurnTrace is the top-level trace for a single chat turn.
type TurnTrace struct {
	TraceID     string         `json:"trace_id"`
	SessionID   string         `json:"session_id"`
	TurnType    string         `json:"turn_type"`
	UserMessage string         `json:"user_message"`
	StartedAt   time.Time      `json:"started_at"`
	EndedAt     time.Time      `json:"ended_at"`
	Status      string         `json:"status"`
	Attributes  map[string]any `json:"attributes,omitempty"`
	Spans       []TraceSpan    `json:"spans"`
}

// AddSpan appends a child span to the trace.
func (t *TurnTrace) AddSpan(s TraceSpan) { t.Spans = append(t.Spans, s) }

// BuildDigest returns a user-facing TraceDigest from the TurnTrace.
// Only safe-to-show fields are included; no CoT, no raw prompts, no internal params.
func (t *TurnTrace) BuildDigest() TraceDigest {
	var totalMs int64
	if !t.EndedAt.IsZero() {
		totalMs = t.EndedAt.Sub(t.StartedAt).Milliseconds()
	} else {
		totalMs = time.Since(t.StartedAt).Milliseconds()
	}
	if totalMs < 0 {
		totalMs = 0
	}
	steps := make([]TraceStepDigest, 0, len(t.Spans))

	for _, s := range t.Spans {
		// Skip the root agent span itself — it's not a step
		if s.Kind == KindAgent {
			continue
		}

		meta := map[string]any{}
		if s.Kind == KindLLM {
			if v, ok := s.Attributes["model"]; ok {
				meta["model"] = v
			}
			if v, ok := s.Attributes["output_tokens"]; ok {
				meta["output_tokens"] = v
			}
		}
		if s.Kind == KindRetriever {
			if v, ok := s.Attributes["hits"]; ok {
				meta["hits"] = v
			}
		}
		steps = append(steps, TraceStepDigest{
			Label:  stepLabel(s.Name),
			Kind:   s.Kind,
			Status: s.Status,
			Ms:     s.DurationMs,
			Meta:   meta,
		})
	}
	if totalMs < 0 {
		totalMs = 0
	}

	return TraceDigest{
		TraceID:  t.TraceID,
		TurnType: t.TurnType,
		TotalMs:  totalMs,
		Status:   t.Status,
		Steps:    steps,
	}
}

// stepLabel returns the human-readable label for a span name.
func stepLabel(name string) string {
	m := map[string]string{
		"classify_and_extract": "意图识别",
		"ask_missing_profile":  "信息确认",
		"bazi_calc":            "八字排盘",
		"yongshen":             "用神分析",
		"dayun_analyzer":       "大运分析",
		"knowledge_search":     "知识检索",
		"qimen_dunjia":         "奇门遁甲",
		"llm_generate":         "命理解读",
		"reuse_bazi_result":    "复用命盘",
		"parse_direct_bazi":    "解析八字",
	"supervisor_decision":  "路由决策",
	"policy_gate":          "策略校验",
	"domain_dispatch":      "领域调度",
	"specialist_bazi":      "八字分析",
	"specialist_qimen":     "奇门分析",
	}
	if label, ok := m[name]; ok {
		return label
	}
	return name
}

// TraceDigest is a user-facing summary derived from TurnTrace.
// Only safe-to-show fields; no CoT, no raw prompts, no internal params.
type TraceDigest struct {
	TraceID  string            `json:"trace_id"`
	TurnType string            `json:"turn_type"`
	TotalMs  int64             `json:"total_ms"`
	Status   string            `json:"status"`
	Steps    []TraceStepDigest `json:"steps"`
}

// TraceStepDigest is a single user-visible step in the trace digest.
type TraceStepDigest struct {
	Label  string         `json:"label"`
	Kind   SpanKind       `json:"kind"`
	Status string         `json:"status"`
	Ms     int64          `json:"ms"`
	Meta   map[string]any `json:"meta,omitempty"`
}
