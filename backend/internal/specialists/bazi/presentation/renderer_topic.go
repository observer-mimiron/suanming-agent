// Package presentation 包含八字已验收结果的用户可见投影。
//
// 本文件负责把已确认的主题模式和上游槽位组织成主题答复；
// 不重新路由问题、不扩展分析范围或生成领域事实。
package presentation

import "strings"

func buildPresentationTopicDirectConclusion(state FinalReplyInput) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TopicDirectAnswer); text != "" {
		return text
	}
	switch normalizedTopicMode(state.AnalysisPlan.TopicMode) {
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

func buildPresentationTopicDirectParagraph(state FinalReplyInput) string {
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

func buildPresentationTopicFocusAnswer(state FinalReplyInput) string {
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

func buildPresentationTopicFrameText(state FinalReplyInput) string {
	return firstDisplayText(state.StaticSynthesis.MainAxis, "本轮未形成结构框架裁断。")
}

func buildPresentationTopicRouteText(state FinalReplyInput) string {
	return firstDisplayText(state.StaticSynthesis.PatternOutcome, "本轮未形成结构路线裁断。")
}

func buildPresentationTopicExplainPosition(state FinalReplyInput) string {
	parts := filterNonEmpty([]string{
		buildPresentationTopicFrameText(state),
		buildPresentationTopicRouteText(state),
	})
	if len(parts) == 0 {
		return "本轮未形成术语位置说明。"
	}
	return strings.Join(parts, " ")
}

func buildPresentationTopicExplainBoundary(state FinalReplyInput) string {
	if text := strings.TrimSpace(state.StaticSynthesis.CounterEvidence); text != "" {
		return text
	}
	return "本轮未形成术语解释边界。"
}

func buildPresentationTopicAdviceConclusion(state FinalReplyInput) string {
	if state.Facts.SubjectAgeBand == "infant" || state.Facts.SubjectAgeBand == "child" {
		return "本轮只解释命理结构和成长节奏，不给出成人现实领域建议。"
	}
	return "本轮只解释命理结构，不替代现实决策。"
}
