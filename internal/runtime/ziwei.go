package runtime

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// executeZiweiPrimaryRoute 处理紫微斗数作为主领域的情况。
// 执行 ziwei_calc 工具，并以紫微命盘为主上下文生成解读。
func (e *Executor) executeZiweiPrimaryRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	if !candidate.IsProfileComplete() {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "需要出生信息才能排紫微斗数命盘，请提供出生年月日时和性别。",
		}})
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{
			"content": "需要出生信息（年月日时+性别）才能排紫微斗数命盘，请告诉我你的出生信息。",
		}})
		return "ask_missing_profile", "需要出生信息才能排紫微斗数命盘", nil
	}

	if ziweiTool, ok := e.tools.Get("ziwei_calc"); ok {
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

	if _, ok := e.tools.Get("bazi_calc"); ok && !candidate.HasBaziResult() {
		if data, baziErr := e.runBaziCalc(ctx, sink, candidate.Profile); baziErr == nil {
			candidate.BaziResult = data
			emitChartComponent(ctx, sink, "bazi-chart", data)
		}
	}

	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "结合紫微斗数命盘分析中...",
	}})
	text, err := e.answerWithKnowledge(ctx, sink, candidate, "ziwei", "")
	if err == nil {
		*st = *candidate
	}
	return "ziwei_primary_reading", text, err
}
