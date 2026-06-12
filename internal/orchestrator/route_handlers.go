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


// executeRoute dispatches execution directly from ApprovedRoute fields.
func (o *Orchestrator) executeRoute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, profilePatch map[string]any, userQuestion string, rawBazi []string) (string, string, error) {
	// Reclassify spurious collect_profile when session already has data.
	taskIntent := route.TaskIntent
	if taskIntent == "collect_profile" && (st.HasBaziResult() || len(st.Profile) > 0) && !containsBirthTime(userQuestion) {
		taskIntent = "amend_profile"
	}

	if route.NeedsClarification {
		return o.executeClarificationRoute(ctx, sink, st, profilePatch, userQuestion, route.ClarificationQuestion)
	}

	// Qimen-primary lane: when the supervisor + policy gate have approved qimen as
	// the primary domain for a timing or cross-domain task, execute qimen_dunjia as
	// the mainline step rather than as a secondary supplement.
	if route.PrimaryDomain == "qimen" && (taskIntent == "timing_followup" || taskIntent == "cross_domain_consult") {
		return o.executeQimenPrimaryRoute(ctx, sink, st, profilePatch, userQuestion)
	}

	// Ziwei-primary lane: when supervisor + policy gate have approved ziwei as
	// the primary domain, execute ziwei_calc as the mainline step.
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

// executeClarificationRoute asks the clarification question when the route
// requires it, or falls back to asking for missing profile fields.
func (o *Orchestrator) executeClarificationRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string, clarificationQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// When the route explicitly requires clarification, do not answer directly
	// from an existing chart. Ask the approved clarification question instead.
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

// executeCollectProfileRoute handles collect_profile: reset profile/chart on
// candidate, ask for missing fields, or run full reading when complete.
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

// executeAmendProfileRoute handles amend_profile: merge profile patch without
// wiping chart, unless chart-affecting fields changed.
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

// executeDirectBaziRoute handles direct_bazi: merge gender into candidate
// and run direct bazi input analysis.
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

// executeFollowupRoute handles all followup task intents: reuse chart if
// available, or fall through to profile collection / full reading.
func (o *Orchestrator) executeFollowupRoute(ctx context.Context, sink EventSink, st *state.SessionState, userQuestion string, needsQimen bool) (string, string, error) {
	candidate := st.Clone()
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	candidate.NeedsQimen = needsQimen

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

// executeQimenPrimaryRoute handles qimen as the primary execution lane.
// It invokes qimen_dunjia unconditionally (not gated by NeedsQimen flag) and
// uses the qimen chart as the main data source for interpretation. Existing
// BaziResult is reused as background context when available, but qimen primary
// does not require a bazi chart to function.
func (o *Orchestrator) executeQimenPrimaryRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// 1. Execute qimen_dunjia as primary domain tool — always, not gated by NeedsQimen.
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
				sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
					"type": "qimen-chart", "payload": qm,
				}})
				qimenData = qm
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

	// 2. Generate answer by reusing existing infrastructure.
	if candidate.HasBaziResult() {
		// Reuse existing bazi chart as background context, qimen data as primary.
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

	// No bazi chart — qimen primary can still produce an answer because
	// qimen uses the current time, not birth time.
	if candidate.IsProfileComplete() {
		// Compute bazi for background context, then answer with qimen as primary.
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

	// No profile and no chart — qimen chart was already emitted above.
	// Answer directly based on the qimen chart; no birth info needed.
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "基于奇门盘分析当前时机...",
	}})
	passages := o.runKnowledgeSearch(ctx, sink, candidate, qimenData)
	text, err := o.streamInterpretation(ctx, sink, candidate, passages, qimenData, true)
	return "qimen_primary_reading", text, err
}

// executeZiweiPrimaryRoute handles ziwei as the primary domain.
// Executes ziwei_calc tool and generates a reading with ziwei chart as primary context.
func (o *Orchestrator) executeZiweiPrimaryRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// Must have complete profile for ziwei paipan.
	if !candidate.IsProfileComplete() {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "需要出生信息才能排紫微斗数命盘，请提供出生年月日时和性别。",
		}})
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{
			"content": "需要出生信息（年月日时+性别）才能排紫微斗数命盘，请告诉我你的出生信息。",
		}})
		return "ask_missing_profile", "需要出生信息才能排紫微斗数命盘", nil
	}

	// Execute ziwei_calc tool.
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

	// Also compute bazi for background context if available.
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

