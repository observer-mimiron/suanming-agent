// Package runtime 包含 Manager 拥有的八字最终渲染。
//
// 本文件负责把已确认的主题模式和上游槽位组织成主题答复；
// 不负责重新路由问题、扩展分析范围或生成领域事实。
package runtime

import "strings"

func buildTopicDirectConclusion(plan baziAnalysisPlan, state baziCharterState, question string) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TopicDirectAnswer); text != "" {
		return text
	}
	switch normalizedTopicMode(plan.TopicMode) {
	case "timing_reason":
		return firstDisplayText(state.DynamicSynthesis.CurrentTrend, "本轮未形成动态裁断。")
	}
	// 普通结构追问没有单独 topic claim 时，直接回答必须仍来自已通过
	// 静态合同的命局结论；不能把“缺少专用字段”展示成“没有裁断”。
	if text := firstDisplayText(
		state.StaticSynthesis.PatternOutcome,
		state.StaticSynthesis.MainAxis,
		state.StaticSynthesis.TopicFocusAnswer,
	); text != "" {
		return text
	}
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		return text
	}
	return "本轮未形成这次追问的直接裁断。"
}

func buildTopicDirectParagraph(state baziCharterState) string {
	parts := make([]string, 0, 3)
	if text := strings.TrimSpace(state.StaticSynthesis.TopicFocusAnswer); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.DynamicSynthesis.LiunianFocus); text != "" {
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func buildTopicFocusAnswer(plan baziAnalysisPlan, state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TopicFocusAnswer); text != "" {
		return text
	}
	return ""
}

func normalizedTopicMode(mode string) string {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "explain_term", "timing_reason", "conservative_reason":
		return mode
	default:
		return "analysis"
	}
}

func buildTopicFrameText(state baziCharterState) string {
	return firstDisplayText(state.StaticSynthesis.MainAxis, "本轮未形成结构框架裁断。")
}

func buildTopicRouteText(state baziCharterState) string {
	return firstDisplayText(state.StaticSynthesis.PatternOutcome, "本轮未形成结构路线裁断。")
}

func buildTopicExplainPosition(state baziCharterState) string {
	parts := filterNonEmpty([]string{
		buildTopicFrameText(state),
		buildTopicRouteText(state),
	})
	if len(parts) == 0 {
		return "本轮未形成术语位置说明。"
	}
	return strings.Join(parts, " ")
}

func buildTopicExplainBoundary(state baziCharterState) string {
	return firstNonEmptyTrim(
		state.StaticSynthesis.CounterEvidence,
		"本轮未形成术语解释边界。",
	)
}

func buildTopicAdviceConclusion(state baziCharterState) string {
	if context := buildBaziSubjectContext(state.Input); context.AgeBand == "infant" || context.AgeBand == "child" {
		return "本轮只解释命理结构和成长节奏，不给出成人现实领域建议。"
	}
	return "本轮只解释命理结构，不替代现实决策。"
}
