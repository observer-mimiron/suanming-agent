// Package runtime 包含 Manager 拥有的八字确定性投影。
//
// 本文件负责图阶段摘要、模型输入视图和可复算事实投影；
// 不负责 Graph 拓扑、合同判定、恢复策略或最终答复渲染。
package runtime

import (
	"fmt"
	"strings"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

// baziSubjectContext 保留 runtime 调用名，实际年龄授权类型由 domain 所有。
type baziSubjectContext = bazidomain.SubjectContext

// mapValue 从动态载荷读取嵌套对象，类型不符时返回 nil，避免投影阶段 panic。
func mapValue(src map[string]any, key string) map[string]any {
	if src == nil {
		return nil
	}
	value, ok := src[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return nil
	}
}

// buildEvidenceStageSummary 将证据阶段状态压缩为可展示的阶段摘要。
func buildEvidenceStageSummary(state baziCharterState) string {
	if len(state.EvidenceBundle.Citations) == 0 {
		return "证据规划已完成，但本轮未拿到可用古籍引文。"
	}
	topics := len(state.EvidenceBundle.TopicBuckets)
	classics := make([]string, 0, len(state.EvidenceBundle.Citations))
	for _, citation := range state.EvidenceBundle.Citations {
		if citation.Classic == "" || containsString(classics, citation.Classic) {
			continue
		}
		classics = append(classics, citation.Classic)
		if len(classics) >= 3 {
			break
		}
	}
	if topics == 0 {
		if len(classics) > 0 {
			return fmt.Sprintf("证据检索已完成，已整理 %d 组古籍依据，主证来自《%s》。", len(state.EvidenceBundle.Citations), strings.Join(classics, "》《"))
		}
		return fmt.Sprintf("证据检索已完成，已整理 %d 组古籍依据。", len(state.EvidenceBundle.Citations))
	}
	if len(classics) > 0 {
		return fmt.Sprintf("证据检索已完成，已为 %d 个判题主题整理 %d 组古籍依据，主证来自《%s》。", topics, len(state.EvidenceBundle.Citations), strings.Join(classics, "》《"))
	}
	return fmt.Sprintf("证据检索已完成，已为 %d 个判题主题整理 %d 组古籍依据。", topics, len(state.EvidenceBundle.Citations))
}

// buildStaticStageSummary 将静态综合结果压缩为阶段提示文本。
func buildStaticStageSummary(state baziCharterState) string {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		return "静态综合未通过，本轮只保留可复算命盘事实。"
	}
	if len(state.StaticSynthesis.ReasoningSteps) > 1 {
		return fmt.Sprintf("静态综合已完成：%s。调候锚点为%s。先看%s，再看%s。关键限制是%s。", state.StaticSynthesis.MainAxis, state.StaticSynthesis.TiaohouAnchor, state.StaticSynthesis.ReasoningSteps[0], state.StaticSynthesis.ReasoningSteps[1], state.StaticSynthesis.CounterEvidence)
	}
	if len(state.StaticSynthesis.ReasoningSteps) > 0 {
		return fmt.Sprintf("静态综合已完成：%s。调候锚点为%s。当前已落实关键推演：%s。关键限制是%s。", state.StaticSynthesis.MainAxis, state.StaticSynthesis.TiaohouAnchor, state.StaticSynthesis.ReasoningSteps[0], state.StaticSynthesis.CounterEvidence)
	}
	return fmt.Sprintf("静态综合已完成，命局主轴收敛为：%s。调候锚点为%s。当前限制为：%s。", state.StaticSynthesis.MainAxis, state.StaticSynthesis.TiaohouAnchor, state.StaticSynthesis.CounterEvidence)
}

// buildDynamicStageSummary 将动态综合结果压缩为阶段提示文本。
func buildDynamicStageSummary(state baziCharterState) string {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return "动态综合未通过，本轮只保留已计算的大运与流年事实。"
	}
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return "动态综合已完成。"
	}
	if strings.TrimSpace(state.DynamicSynthesis.LiunianFocus) != "" {
		return fmt.Sprintf("动态综合已完成：%s。当前判定为%s，流年焦点是%s。", state.DynamicSynthesis.CurrentTrend, state.DynamicSynthesis.WindowLevel, state.DynamicSynthesis.LiunianFocus)
	}
	if len(state.DynamicSynthesis.ReasoningSteps) > 1 {
		return fmt.Sprintf("动态综合已完成：%s。当前判定为%s。先看%s，再看%s。", state.DynamicSynthesis.CurrentTrend, state.DynamicSynthesis.WindowLevel, state.DynamicSynthesis.ReasoningSteps[0], state.DynamicSynthesis.ReasoningSteps[1])
	}
	if len(state.DynamicSynthesis.ReasoningSteps) > 0 {
		return fmt.Sprintf("动态综合已完成：%s。当前判定为%s，关键触发为%s。", state.DynamicSynthesis.CurrentTrend, state.DynamicSynthesis.WindowLevel, state.DynamicSynthesis.ReasoningSteps[0])
	}
	return fmt.Sprintf("动态综合已完成：%s。当前判定为%s。", state.DynamicSynthesis.CurrentTrend, state.DynamicSynthesis.WindowLevel)
}

// buildAnalysisPlannerPayload 生成分析规划模型只需读取的确定性输入视图。
func buildAnalysisPlannerPayload(question string, chartFacts baziCharterInput) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart":          buildCoreChartView(chartFacts),
			"dynamic_facts_ready": hasDynamicSystemFacts(chartFacts),
		},
		"question": question,
	}
}

// buildEvidencePlannerPayload 生成证据规划模型需要的图表、动态事实和来源范围。
func buildEvidencePlannerPayload(question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) map[string]any {
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	input := map[string]any{
		"core_chart":          buildCoreChartView(chartFacts),
		"dynamic_facts_ready": hasDynamicSystemFacts(chartFacts),
	}
	if stage == "dynamic" {
		input["dynamic_facts"] = buildDynamicFactsView(chartFacts)
	}
	return map[string]any{
		"input":             input,
		"question":          question,
		"analysis_plan":     analysisPlan,
		"authority_sources": stageAuthoritySources(stage),
	}
}

// buildStaticSynthesisPayload 生成静态综合模型的最小结构化输入。
func buildStaticSynthesisPayload(state baziCharterState) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart":      buildCoreChartView(state.Input),
			"fact_capsule":    buildBaziFactCapsulePromptView(state, false),
			"subject_context": buildBaziSubjectContext(state.Input),
		},
		"analysis_plan":    state.AnalysisPlan,
		"evidence_plan":    state.EvidencePlan,
		"evidence_bundle":  buildEvidenceBundleView(state.EvidenceBundle, true),
		"evidence_quality": state.EvidenceQuality,
		"question":         state.Input.UserQuestion,
	}
}

// buildDynamicSynthesisPayload 生成动态综合模型的最小结构化输入。
func buildDynamicSynthesisPayload(state baziCharterState) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart":      buildCoreChartView(state.Input),
			"dynamic_facts":   buildDynamicFactsView(state.Input),
			"fact_capsule":    buildBaziFactCapsulePromptView(state, true),
			"subject_context": buildBaziSubjectContext(state.Input),
		},
		"analysis_plan":    state.AnalysisPlan,
		"evidence_bundle":  buildEvidenceBundleView(state.EvidenceBundle, true),
		"evidence_quality": state.EvidenceQuality,
		"static_synthesis": state.StaticSynthesis,
		"question":         state.Input.UserQuestion,
	}
}

// buildBaziSubjectContext 把 runtime 命盘字段适配给 domain 年龄范围计算器；
// 它只限制解释范围，不修改命盘事实。
func buildBaziSubjectContext(input baziCharterInput) baziSubjectContext {
	calculated := bazidomain.BuildSubjectContext(bazidomain.SubjectContextInput{
		BirthYear:  yearPrefix(stringValue(input.BaziResult["birthday"])),
		TargetYear: intValue(input.Liunian["liunian_year"]),
	})
	return baziSubjectContext{
		BirthYear:             calculated.BirthYear,
		TargetYear:            calculated.TargetYear,
		Age:                   calculated.Age,
		AgeBand:               calculated.AgeBand,
		AllowedOutcomeDomains: append([]string{}, calculated.AllowedOutcomeDomains...),
	}
}

// yearPrefix 从标准时间字符串的前四位读取年份，异常输入返回零。
func yearPrefix(value string) int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return 0
	}
	var year int
	if _, err := fmt.Sscanf(value[:4], "%d", &year); err != nil {
		return 0
	}
	return year
}

// buildEvidenceBundleView 将内部证据 bundle 投影为模型可读视图。
func buildEvidenceBundleView(bundle baziEvidenceBundle, includeTopics bool) map[string]any {
	view := map[string]any{
		"stage":     bundle.Stage,
		"citations": append([]baziCitation{}, bundle.Citations...),
	}
	if len(bundle.Conflicts) > 0 {
		view["conflicts"] = append([]string{}, bundle.Conflicts...)
	}
	if includeTopics && len(bundle.TopicBuckets) > 0 {
		topics := make(map[string][]baziCitation, len(bundle.TopicBuckets))
		for topic, citations := range bundle.TopicBuckets {
			topics[topic] = append([]baziCitation{}, citations...)
		}
		view["topic_buckets"] = topics
	}
	return view
}

// buildCoreChartView 只暴露模型所需的核心命盘和规则事实。
func buildCoreChartView(input baziCharterInput) map[string]any {
	return bazidomain.BuildCoreChartView(bazidomain.ChartViewInput{
		BaziResult:          input.BaziResult,
		Yongshen:            input.Yongshen,
		SelectedRuleProfile: input.RuleProfile,
	})
}

// baziTiaohouCoverage 报告证据覆盖，不报告可选 Go 规则表是否实现。
// 静态综合已经收到检索到的权威材料，写成 runtime_profile_disabled 会让模型误判调候证据缺失。
func baziTiaohouCoverage(quality baziEvidenceQuality) string {
	return bazidomain.TiaohouCoverage(quality.CoveredTopics, quality.MissingTopics)
}

// isTiaohouImplementationPlaceholder 从领域事实视图中过滤工程状态占位文本，
// 让综合节点看到命盘事实和证据，而不是未来规则表尚未实现的过期提醒。
func isTiaohouImplementationPlaceholder(value any) bool {
	return bazidomain.IsTiaohouImplementationPlaceholder(value)
}

// buildDynamicFactsView 投影已由确定性工具计算的大运与流年事实。
func buildDynamicFactsView(input baziCharterInput) map[string]any {
	return bazidomain.BuildDynamicFactsView(bazidomain.ChartViewInput{
		Dayun: input.Dayun, Liunian: input.Liunian,
	})
}

// buildDayunFactsView 保留动态模型需要的大运目录和当前运绑定字段。
func buildDayunFactsView(dayun map[string]any) map[string]any {
	return bazidomain.BuildDayunFactsView(dayun)
}

// buildLiunianFactsView 保留动态模型需要的流年字段和当前大运引用。
func buildLiunianFactsView(liunian map[string]any) map[string]any {
	return bazidomain.BuildLiunianFactsView(liunian)
}

// extractMonthPillar 从不同解码形态的四柱数组中读取月柱。
func extractMonthPillar(raw any) map[string]any {
	return bazidomain.ExtractMonthPillar(raw)
}

// copyAnyMap 复制一层 map，避免投影视图修改原始事实容器。
func copyAnyMap(src map[string]any) map[string]any {
	return bazidomain.CopyAnyMap(src)
}
