package tracing

import "time"

// ProcessDigest 是面向产品 UI 的处理过程投影。
// 它来自原始 TurnTrace，但会按产品阶段聚合并裁掉低层调试噪音。
type ProcessDigest struct {
	TraceID  string               `json:"trace_id"`
	TurnType string               `json:"turn_type"`
	TotalMs  int64                `json:"total_ms"`
	Status   string               `json:"status"`
	Phases   []ProcessPhaseDigest `json:"phases"`
}

// ProcessPhaseDigest 表示一个可直接展示给用户的过程阶段。
type ProcessPhaseDigest struct {
	Key      string         `json:"key"`
	Label    string         `json:"label"`
	Status   string         `json:"status"`
	Ms       int64          `json:"ms"`
	Summary  string         `json:"summary,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
	DebugRef []string       `json:"debug_ref,omitempty"`
}

type processPhaseSpec struct {
	key   string
	label string
}

var processPhaseSpecs = map[string]processPhaseSpec{
	"supervisor_decision": {key: "route", label: "路由判断"},
	"supervisor_model":    {key: "route", label: "路由判断"},
	"policy_gate":         {key: "policy", label: "资料与策略校验"},
	"preflight":           {key: "policy", label: "资料与策略校验"},
	"prefill":             {key: "prepare", label: "排盘与资料准备"},
	"bazi_calc":           {key: "prepare", label: "排盘与资料准备"},
	"qimen_dunjia":        {key: "prepare", label: "排盘与资料准备"},
	"ziwei_calc":          {key: "prepare", label: "排盘与资料准备"},
	"knowledge_search":    {key: "prepare", label: "排盘与资料准备"},
	"domain_dispatch":     {key: "execute", label: "领域执行"},
	"specialist_bazi":     {key: "execute", label: "领域执行"},
	"specialist_qimen":    {key: "execute", label: "领域执行"},
	"specialist_ziwei":    {key: "execute", label: "领域执行"},
	"contract_gate":       {key: "answer", label: "结果验收与生成"},
	"llm_generate":        {key: "answer", label: "结果验收与生成"},
}

type phaseAccumulator struct {
	digest ProcessPhaseDigest
}

// BuildProcessDigest 把原始 TurnTrace 投影成面向产品展示的阶段时间线。
func (t *TurnTrace) BuildProcessDigest() ProcessDigest {
	phases := make([]ProcessPhaseDigest, 0, len(t.Spans))
	phaseIndex := map[string]int{}

	for _, s := range t.Spans {
		if s.Kind == KindAgent || s.Name == "sse_emit" {
			continue
		}
		spec, ok := processPhaseSpecs[s.Name]
		if !ok {
			continue
		}
		idx, exists := phaseIndex[spec.key]
		if !exists {
			phaseIndex[spec.key] = len(phases)
			phases = append(phases, ProcessPhaseDigest{
				Key:      spec.key,
				Label:    spec.label,
				Status:   normalizeStatus(s.Status),
				Meta:     map[string]any{},
				DebugRef: []string{s.Name},
			})
			idx = len(phases) - 1
		} else {
			phases[idx].DebugRef = append(phases[idx].DebugRef, s.Name)
			phases[idx].Status = worseStatus(phases[idx].Status, normalizeStatus(s.Status))
		}
		phases[idx].Ms += s.DurationMs
		collectUserMeta(&phases[idx], s)
	}

	for i := range phases {
		phases[i].Summary = buildPhaseSummary(phases[i])
		if len(phases[i].Meta) == 0 {
			phases[i].Meta = nil
		}
	}

	return ProcessDigest{
		TraceID:  t.TraceID,
		TurnType: t.TurnType,
		TotalMs:  traceTotalMs(t),
		Status:   normalizeStatus(t.Status),
		Phases:   phases,
	}
}

func collectUserMeta(phase *ProcessPhaseDigest, span TraceSpan) {
	if phase == nil {
		return
	}
	switch span.Kind {
	case KindRetriever:
		if v, ok := span.Attributes["hits"]; ok {
			phase.Meta["hits"] = v
		}
	case KindLLM:
		if v, ok := span.Attributes["model"]; ok {
			phase.Meta["model"] = v
		}
		if v, ok := span.Attributes["output_tokens"]; ok {
			phase.Meta["output_tokens"] = v
		}
	}
	if v, ok := span.Attributes["artifact_present"]; ok {
		phase.Meta["artifact_present"] = v
	}
	if v, ok := span.Attributes["guardrail_result"]; ok {
		phase.Meta["guardrail_result"] = v
	}
}

func buildPhaseSummary(phase ProcessPhaseDigest) string {
	switch phase.Key {
	case "route":
		return "已完成路由判断，正在按当前主领域组织本轮解读。"
	case "policy":
		return "已完成资料与策略校验，确认本轮是否需要澄清或补资料。"
	case "prepare":
		if _, ok := phase.Meta["hits"]; ok {
			return "已完成排盘与资料准备，并命中相关依据资料。"
		}
		return "已完成排盘与资料准备。"
	case "execute":
		return "已完成领域执行，正在汇总本轮分析结果。"
	case "answer":
		if result, ok := phase.Meta["guardrail_result"]; ok && result == "blocked" {
			return "结果验收未通过，已阻止不完整结论直接展示。"
		}
		return "已完成结果验收与生成。"
	default:
		return phase.Label
	}
}

func traceTotalMs(t *TurnTrace) int64 {
	if t == nil {
		return 0
	}
	var totalMs int64
	if !t.EndedAt.IsZero() {
		totalMs = t.EndedAt.Sub(t.StartedAt).Milliseconds()
	} else {
		totalMs = time.Since(t.StartedAt).Milliseconds()
	}
	if totalMs < 0 {
		totalMs = 0
	}
	return totalMs
}

func normalizeStatus(status string) string {
	if status == "" {
		return "ok"
	}
	return status
}

func worseStatus(current, next string) string {
	if statusSeverity(next) > statusSeverity(current) {
		return next
	}
	return current
}

func statusSeverity(status string) int {
	switch normalizeStatus(status) {
	case "error":
		return 4
	case "fallback":
		return 3
	case "degraded":
		return 2
	default:
		return 1
	}
}
