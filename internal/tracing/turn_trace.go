// Package tracing 暂与 tracing.go 共享包注释，本文件定义 TurnTrace（单次对话的完整追踪）及其面向用户的 TraceDigest。

package tracing

import (
	"context"
	"time"
)

// SpanKind 遵循 OpenInference 语义约定的 Span 类型。
type SpanKind string

const (
	KindAgent     SpanKind = "AGENT"
	KindChain     SpanKind = "CHAIN"
	KindTool      SpanKind = "TOOL"
	KindRetriever SpanKind = "RETRIEVER"
	KindLLM       SpanKind = "LLM"
)

// TraceSpan 表示一次追踪中的一个工作单元。
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

// TurnTrace 是一次对话轮次的顶层追踪记录。
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
	otelCtx     context.Context
	otelRoot    otelSpanBridge
	otelBridge  otelBridge
}

// AddSpan 向追踪追加一个子 span。
func (t *TurnTrace) AddSpan(s TraceSpan) { t.Spans = append(t.Spans, s) }

// BuildDigest 从 TurnTrace 生成 legacy TraceDigest。
// 它保留原有步骤摘要契约用于兼容旧消费方，但不再是新的产品展示模型。
func (t *TurnTrace) BuildDigest() TraceDigest {
	steps := make([]TraceStepDigest, 0, len(t.Spans))

	for _, s := range t.Spans {
		// 跳过根 agent span 自身——它不是一个步骤
		if s.Kind == KindAgent {
			continue
		}
		// `sse_emit` 仍保留在原始 trace 里做排障，但不进入用户可见的处理过程，
		// 否则流式 chunk 会把时间线膨胀成上千步。
		if s.Name == "sse_emit" {
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
	return TraceDigest{
		TraceID:  t.TraceID,
		TurnType: t.TurnType,
		TotalMs:  traceTotalMs(t),
		Status:   normalizeStatus(t.Status),
		Steps:    steps,
	}
}

// stepLabel 根据 span 名称返回用户可读的中文标签。
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
		"supervisor_model":     "路由模型",
		"policy_gate":          "策略校验",
		"preflight":            "执行前校验",
		"prefill":              "预填充复用",
		"domain_dispatch":      "领域调度",
		"specialist_bazi":      "八字分析",
		"specialist_qimen":     "奇门分析",
		"contract_gate":        "结果验收",
		"sse_emit":             "SSE 输出",
	}
	if label, ok := m[name]; ok {
		return label
	}
	return name
}

// TraceDigest 是旧版 trace 摘要契约。
// 新的产品过程展示应改用 ProcessDigest；调试展示应改用 DebugTraceDigest。
type TraceDigest struct {
	TraceID  string            `json:"trace_id"`
	TurnType string            `json:"turn_type"`
	TotalMs  int64             `json:"total_ms"`
	Status   string            `json:"status"`
	Steps    []TraceStepDigest `json:"steps"`
}

// TraceStepDigest 是 TraceDigest 中的一个用户可见步骤。
type TraceStepDigest struct {
	Label  string         `json:"label"`
	Kind   SpanKind       `json:"kind"`
	Status string         `json:"status"`
	Ms     int64          `json:"ms"`
	Meta   map[string]any `json:"meta,omitempty"`
}
