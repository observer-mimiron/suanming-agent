package orchestrator

import (
	"context"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
	qimenTools "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)


// executeRoute 直接从 ApprovedRoute 字段调度执行。
func (o *Orchestrator) executeRoute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, profilePatch map[string]any, userQuestion string, rawBazi []string) (string, string, error) {
	// 当会话已有资料时，重新分类误判的 collect_profile
	taskIntent := route.TaskIntent
	if taskIntent == "collect_profile" && (st.HasBaziResult() || len(st.Profile) > 0) && !containsBirthTime(userQuestion) {
		taskIntent = "amend_profile"
	}

	if route.NeedsClarification {
		return o.executeClarificationRoute(ctx, sink, st, profilePatch, userQuestion, route.ClarificationQuestion)
	}

	// 奇门主领域路径：当 supervisor + 策略门批准奇门作为
	// 择时或跨领域任务的主领域时，将 qimen_dunjia 作为
	// 主线步骤执行，而非作为辅助补充。
	if route.PrimaryDomain == "qimen" && (taskIntent == "timing_followup" || taskIntent == "cross_domain_consult") {
		return o.executeQimenPrimaryRoute(ctx, sink, st, profilePatch, userQuestion)
	}

	// 紫微主领域路径：当 supervisor + 策略门批准紫微作为
	// 主领域时，将 ziwei_calc 作为主线步骤执行。
	if route.PrimaryDomain == "ziwei" {
		return o.executeZiweiPrimaryRoute(ctx, sink, st, profilePatch, userQuestion)
	}

	switch taskIntent {
	case "direct_bazi":
		return o.executeDirectBaziRoute(ctx, sink, st, profilePatch, rawBazi)
	case "collect_profile":
		return o.executeCollectProfileRoute(ctx, sink, st, profilePatch, userQuestion)
	case "amend_profile":
		return o.executeAmendProfileRoute(ctx, sink, st, profilePatch, userQuestion)
	case "timing_followup", "cross_domain_consult":
		return o.executeFollowupRoute(ctx, sink, st, userQuestion, true)
	default:
		return o.executeFollowupRoute(ctx, sink, st, userQuestion, st.NeedsQimen)
	}
}

// executeClarificationRoute 在路由需要时询问澄清问题，否则回退到询问缺失的资料字段。
func (o *Orchestrator) executeClarificationRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string, clarificationQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// 当路由明确要求澄清时，不直接从现有命盘回答
	// 而是询问已批准的澄清问题。
	if candidate.HasBaziResult() || candidate.IsProfileComplete() {
		if strings.TrimSpace(clarificationQuestion) == "" {
			clarificationQuestion = "请确认一下您的需求，我再为您详细分析。"
		}
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": clarificationQuestion}})
		*st = *candidate
		return "clarification", clarificationQuestion, nil
	}

	if strings.TrimSpace(clarificationQuestion) != "" {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": clarificationQuestion}})
		*st = *candidate
		return "ask_missing_profile", clarificationQuestion, nil
	}

	text, err := o.handleAsk(ctx, sink, candidate)
	if err == nil {
		*st = *candidate
	}
	return "ask_missing_profile", text, err
}

// executeCollectProfileRoute 处理 collect_profile：重置候选会话的资料/命盘，询问缺失字段，或在资料完整时进行完整解读。
func (o *Orchestrator) executeCollectProfileRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.Profile = make(map[string]any)
	candidate.BaziResult = nil
	for k, v := range profilePatch {
		candidate.Profile[k] = v
	}
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	if !candidate.IsProfileComplete() {
		text, err := o.handleAsk(ctx, sink, candidate)
		if err == nil && !st.HasBaziResult() && len(st.Profile) == 0 {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	text, err := o.handleFullReading(ctx, sink, candidate)
	if err == nil && candidate.HasBaziResult() {
		*st = *candidate
	}
	return "full_reading", text, err
}

// executeAmendProfileRoute 处理 amend_profile：合并资料补丁而不清空命盘，除非影响命盘的字段发生了变化。
func (o *Orchestrator) executeAmendProfileRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	changed := candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	if changed && profileChangesAffectChart(profilePatch) {
		candidate.BaziResult = nil
	}
	if !candidate.IsProfileComplete() && !candidate.HasBaziResult() {
		text, err := o.handleAsk(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	if candidate.BaziResult == nil {
		text, err := o.handleFullReading(ctx, sink, candidate)
		if err == nil && candidate.HasBaziResult() {
			*st = *candidate
		}
		return "full_reading", text, err
	}
	text, err := o.handleFollowupReading(ctx, sink, candidate)
	if err == nil {
		*st = *candidate
	}
	return "followup_reading", text, err
}

// executeDirectBaziRoute 处理 direct_bazi：将性别合并到候选会话中，并运行直接八字输入分析。
func (o *Orchestrator) executeDirectBaziRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, rawBazi []string) (string, string, error) {
	candidate := st.Clone()
	if g, ok := profilePatch["gender"]; ok {
		candidate.Profile["gender"] = g
	}
	text, err := o.handleBaziInput(ctx, sink, candidate, rawBazi)
	if err == nil && candidate.HasBaziResult() {
		*st = *candidate
	}
	return "direct_bazi", text, err
}

// executeFollowupRoute 处理所有跟进任务意图：有命盘时复用，否则回退到资料收集/完整解读。
func (o *Orchestrator) executeFollowupRoute(ctx context.Context, sink EventSink, st *state.SessionState, userQuestion string, needsQimen bool) (string, string, error) {
	candidate := st.Clone()
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	candidate.NeedsQimen = needsQimen

	if needsQimen {
		return o.executeParallelFortuneRoute(ctx, sink, st, candidate, userQuestion)
	}

	if candidate.HasBaziResult() {
		text, err := o.handleFollowupReading(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "followup_reading", text, err
	}
	if !candidate.IsProfileComplete() {
		text, err := o.handleAsk(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	text, err := o.handleFullReading(ctx, sink, candidate)
	if err == nil && candidate.HasBaziResult() {
		*st = *candidate
	}
	return "full_reading", text, err
}

// executeQimenPrimaryRoute 处理奇门作为主执行领域的情况。
// 无条件调用 qimen_dunjia（不受 NeedsQimen 标志控制），并将奇门盘作为解读的主要数据来源。
// 当存在现有 BaziResult 时作为背景上下文复用，但奇门主领域不依赖八字命盘即可运行。
func (o *Orchestrator) executeQimenPrimaryRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// 1. 执行 qimen_dunjia 作为主领域工具——始终执行，不受 NeedsQimen 标志控制。
	var qimenData map[string]any
	if qimenTool, ok := o.tools.Get("qimen_dunjia"); ok {
		qmSpan := tracing.SpanFromContext(ctx, "qimen_dunjia", tracing.KindTool)
		now := resolveQimenTime(time.Now())
		qimenParams := qimenTools.ResolveTime(now)
		sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
			"tool": "qimen_dunjia", "params": qimenParams,
		}})
		qimenResult, qimenErr := qimenTool.Execute(ctx, qimenParams)
		if qimenErr == nil {
			if qm, ok2 := qimenResult.(map[string]any); ok2 {
				qimenData = qm
				if !st.HasQimenResult() {
					sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
						"type": "qimen-chart", "payload": qm,
					}})
				}
				candidate.QimenResult = qm
			}
		} else {
			qmSpan.SetStatus("fallback")
			qmSpan.RecordError(qimenErr)
			sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
				"agent": "orchestrator", "text": "奇门排盘失败，改按八字继续分析。",
			}})
		}
		qmSpan.End()
	} else {
		qmSpan := tracing.SpanFromContext(ctx, "qimen_dunjia", tracing.KindChain)
		qmSpan.SetStatus("degraded")
		qmSpan.SetAttribute("degrade_reason", "tool_not_registered")
		qmSpan.End()
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "奇门排盘工具未注册，改按八字继续分析。",
		}})
	}

	// 2. 通过复用现有基础设施生成回答。
	if candidate.HasBaziResult() {
		// 复用现有八字命盘作为背景上下文，奇门数据作为主要数据源。
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "复用已有命盘，结合奇门盘分析...",
		}})
		passages := o.runKnowledgeSearch(ctx, sink, candidate, qimenData)
		text, err := o.streamInterpretation(ctx, sink, candidate, passages, qimenData, false)
		if err == nil {
			*st = *candidate
		}
		return "qimen_primary_reading", text, err
	}

	// 无八字命盘——奇门主领域仍可生成回答，因为
	// 奇门使用当前时间，而非出生时间。
	if candidate.IsProfileComplete() {
		// 计算八字作为背景上下文，然后以奇门为主生成回答。
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "开始排盘并结合奇门盘分析...",
		}})
		if baziTool, ok := o.tools.Get("bazi_calc"); ok {
			baziSpan := tracing.SpanFromContext(ctx, "bazi_calc", tracing.KindTool)
			baziResult, baziErr := baziTool.Execute(ctx, candidate.Profile)
			if baziErr == nil {
				if data, ok := baziResult.(map[string]any); ok {
					candidate.BaziResult = data
					sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
						"type": "bazi-chart", "payload": data,
					}})
				}
			} else {
				baziSpan.RecordError(baziErr)
			}
			baziSpan.End()
		}
		passages := o.runKnowledgeSearch(ctx, sink, candidate, qimenData)
		text, err := o.streamInterpretation(ctx, sink, candidate, passages, qimenData, false)
		if err == nil && candidate.HasBaziResult() {
			*st = *candidate
		}
		return "qimen_primary_reading", text, err
	}

	// 无资料且无命盘——奇门盘已在上面发出。
	// 直接基于奇门盘回答，无需出生信息。
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "基于奇门盘分析当前时机...",
	}})
	passages := o.runKnowledgeSearch(ctx, sink, candidate, qimenData)
	text, err := o.streamInterpretation(ctx, sink, candidate, passages, qimenData, true)
	return "qimen_primary_reading", text, err
}

// executeZiweiPrimaryRoute 处理紫微斗数作为主领域的情况。
// 执行 ziwei_calc 工具，并以紫微命盘为主上下文生成解读。
func (o *Orchestrator) executeZiweiPrimaryRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// 必须有完整资料才能排紫微斗数命盘。
	if !candidate.IsProfileComplete() {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "需要出生信息才能排紫微斗数命盘，请提供出生年月日时和性别。",
		}})
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{
			"content": "需要出生信息（年月日时+性别）才能排紫微斗数命盘，请告诉我你的出生信息。",
		}})
		return "ask_missing_profile", "需要出生信息才能排紫微斗数命盘", nil
	}

	// 执行 ziwei_calc 工具。
	var ziweiData map[string]any
	if ziweiTool, ok := o.tools.Get("ziwei_calc"); ok {
		zwSpan := tracing.SpanFromContext(ctx, "ziwei_calc", tracing.KindTool)
		sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
			"tool": "ziwei_calc", "params": candidate.Profile,
		}})
		ziweiResult, ziweiErr := ziweiTool.Execute(ctx, candidate.Profile)
		if ziweiErr == nil {
			if data, ok := ziweiResult.(map[string]any); ok {
				candidate.ZiWeiResult = data
				sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
					"type": "ziwei-chart", "payload": data,
				}})
				ziweiData = data
			}
		} else {
			zwSpan.RecordError(ziweiErr)
			zwSpan.SetStatus("error")
			zwSpan.End()
			sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
				"agent": "orchestrator", "text": "紫微斗数排盘失败：" + ziweiErr.Error(),
			}})
			return "ziwei_primary_reading", "", ziweiErr
		}
		zwSpan.End()
	} else {
		span := tracing.SpanFromContext(ctx, "ziwei_calc", tracing.KindChain)
		span.SetStatus("degraded")
		span.SetAttribute("degrade_reason", "tool_not_registered")
		span.End()
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "紫微斗数排盘工具未注册。",
		}})
		return "ziwei_primary_reading", "", nil
	}

	// 同时计算八字作为背景上下文（如可用）。
	if baziTool, ok := o.tools.Get("bazi_calc"); ok && !candidate.HasBaziResult() {
		baziSpan := tracing.SpanFromContext(ctx, "bazi_calc", tracing.KindTool)
		baziResult, baziErr := baziTool.Execute(ctx, candidate.Profile)
		if baziErr == nil {
			if data, ok := baziResult.(map[string]any); ok {
				candidate.BaziResult = data
				sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
					"type": "bazi-chart", "payload": data,
				}})
			}
		}
		baziSpan.End()
	}

	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "结合紫微斗数命盘分析中...",
	}})
	passages := o.runKnowledgeSearch(ctx, sink, candidate, ziweiData)
	text, err := o.streamInterpretation(ctx, sink, candidate, passages, ziweiData, false)
	if err == nil {
		*st = *candidate
	}
	return "ziwei_primary_reading", text, err
}


// executeParallelFortuneRoute 处理带奇门辅助的 fortune_followup 情况。
// 如果缺少命盘则根据候选资料计算八字命盘，然后无条件运行当前时间的 qimen_dunjia，
// 最后将两者合并为结合知识搜索上下文的综合解读。
func (o *Orchestrator) executeParallelFortuneRoute(ctx context.Context, sink EventSink, st *state.SessionState, candidate *state.SessionState, userQuestion string) (string, string, error) {
	if !candidate.IsProfileComplete() && !candidate.HasBaziResult() {
		text, err := o.handleAsk(ctx, sink, candidate)
		if err == nil { *st = *candidate }
		return "ask_missing_profile", text, err
	}
	if !candidate.HasBaziResult() && candidate.IsProfileComplete() {
		if baziTool, ok := o.tools.Get("bazi_calc"); ok {
			r, err := baziTool.Execute(ctx, candidate.Profile)
			if err == nil {
				if data, ok := r.(map[string]any); ok {
					candidate.BaziResult = data
					sink.Emit(ctx, Event{Type: "component", Data: map[string]any{"type": "bazi-chart", "payload": data}})
				}
			}
		}
	}
	var qimenData map[string]any
	if qimenTool, ok := o.tools.Get("qimen_dunjia"); ok {
		qmSpan := tracing.SpanFromContext(ctx, "qimen_dunjia", tracing.KindTool)
		defer qmSpan.End()
		now := resolveQimenTime(time.Now())
		params := qimenTools.ResolveTime(now)
		sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{"tool": "qimen_dunjia", "params": params}})
		r, err := qimenTool.Execute(ctx, params)
		if err == nil {
			if qm, ok := r.(map[string]any); ok {
				qimenData = qm
				if !st.HasQimenResult() {
					sink.Emit(ctx, Event{Type: "component", Data: map[string]any{"type": "qimen-chart", "payload": qm}})
				}
				candidate.QimenResult = qm
			}
		}
	}
	passages := o.runKnowledgeSearch(ctx, sink, candidate, qimenData)
	text, err := o.streamInterpretation(ctx, sink, candidate, passages, qimenData, false)
	if err == nil { *st = *candidate }
	return "parallel_fortune", text, err
}
