package runtime

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/state"
)

// executeQimenPrimaryRoute 处理奇门作为主执行领域的情况。
// 无条件调用 qimen_dunjia（不受 NeedsQimen 标志控制），并将奇门盘作为解读的主要数据来源。
// 当存在现有 BaziResult 时作为背景上下文复用，但奇门主领域不依赖八字命盘即可运行。
func (e *Executor) executeQimenPrimaryRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	qimenData, qimenRegistered, qimenErr := e.runCurrentQimen(ctx, sink)
	if qimenErr == nil && qimenData != nil {
		if !st.HasQimenResult() {
			emitChartComponent(ctx, sink, "qimen-chart", qimenData)
		}
		candidate.QimenResult = qimenData
	} else if qimenErr != nil {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "奇门排盘失败，改按八字继续分析。",
		}})
	} else if !qimenRegistered {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "奇门排盘工具未注册，改按八字继续分析。",
		}})
	}

	if candidate.HasBaziResult() {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "复用已有命盘，结合奇门盘分析...",
		}})
		text, err := e.answerWithKnowledge(ctx, sink, candidate, "qimen", "")
		if err == nil {
			*st = *candidate
		}
		return "qimen_primary_reading", text, err
	}

	if candidate.IsProfileComplete() {
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "开始排盘并结合奇门盘分析...",
		}})
		if baziData, baziErr := e.runBaziCalc(ctx, sink, candidate.Profile); baziErr == nil {
			candidate.BaziResult = baziData
			emitChartComponent(ctx, sink, "bazi-chart", baziData)
		}
		text, err := e.answerWithKnowledge(ctx, sink, candidate, "qimen", "")
		if candidate.HasBaziResult() {
			*st = *candidate
		}
		return "qimen_primary_reading", text, err
	}

	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "基于奇门盘分析当前时机...",
	}})
	text, err := e.answerWithKnowledge(ctx, sink, candidate, "qimen", "")
	return "qimen_primary_reading", text, err
}

// executeParallelFortuneRoute 处理带奇门辅助的 fortune_followup 情况。
// 如果缺少命盘则根据候选资料计算八字命盘，然后无条件运行当前时间的 qimen_dunjia，
// 最后将两者合并为结合知识搜索上下文的综合解读。
func (e *Executor) executeParallelFortuneRoute(ctx context.Context, sink EventSink, st *state.SessionState, candidate *state.SessionState, userQuestion string) (string, string, error) {
	if !candidate.IsProfileComplete() && !candidate.HasBaziResult() {
		text, err := e.handleAsk(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	if !candidate.HasBaziResult() && candidate.IsProfileComplete() {
		if baziData, baziErr := e.runBaziCalc(ctx, sink, candidate.Profile); baziErr == nil {
			candidate.BaziResult = baziData
			emitChartComponent(ctx, sink, "bazi-chart", baziData)
		}
	}

	if qimenData, _, qimenErr := e.runCurrentQimen(ctx, sink); qimenErr == nil && qimenData != nil {
		if !st.HasQimenResult() {
			emitChartComponent(ctx, sink, "qimen-chart", qimenData)
		}
		candidate.QimenResult = qimenData
	}

	text, err := e.answerWithKnowledge(ctx, sink, candidate, "qimen", "")
	if err == nil {
		*st = *candidate
	}
	return "parallel_fortune", text, err
}
