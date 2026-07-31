package runtime

import (
	"fmt"
	"strings"
)

// renderBaziFinalReply 改为由程序直接消费上游结构化结论并渲染最终 markdown。
// 这样最终成文不再依赖自由文本 writer 自行排版，从根上消除标题、加粗结论、
// 编号步骤等展示合同漂移导致的整轮失败。
func renderBaziFinalReply(plan baziAnalysisPlan, state baziCharterState, question string) string {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		return renderFactsOnlyDegradedTemplate(state)
	}
	switch strings.TrimSpace(plan.WriterTemplate) {
	case "topic":
		return renderTopicTemplate(plan, state, question)
	case "year":
		return renderYearTemplate(state, question)
	default:
		return renderFullTemplate(state)
	}
}

func renderFactsOnlyDegradedTemplate(state baziCharterState) string {
	var b strings.Builder
	writeHeading(&b, "总览结论")
	writeConclusion(&b, "模型综合未通过，本轮只展示可复算事实；不输出主轴、层次、大运吉凶或具体应事。")
	writeBullets(&b, []string{
		"**输出范围**：排盘、强弱证据摘要、大运日期边界、十神与已计算关系。",
		"**静态状态**：静态综合未通过，本轮未输出主轴与层次裁断。",
		"**动态状态**：动态综合未通过时，本轮仅展示大运与流年事实。",
	})

	writeHeading(&b, "命盘事实")
	writeConclusion(&b, "以下为工具返回事实，不是模型综合裁断。")
	writeBullets(&b, buildStaticFactBullets(state))

	writeHeading(&b, "大运事实")
	writeConclusion(&b, "大运干支、年龄与日期边界由工具计算；本节不判趋势。")
	factsDynamic := buildFactsOnlyDynamicSynthesis(state.Input, state.StaticSynthesis, state.DynamicSynthesis.RecoveryReason)
	writeDayunAnalysis(&b, attachDayunPeriodLabels(factsDynamic.DayunPath, dayunPeriods(state.Input.Dayun)))

	writeHeading(&b, "流年事实")
	writeConclusion(&b, "流年仅展示工具事实，不展开应期。")
	writeBullets(&b, buildLiunianFactBullets(state))

	writeHeading(&b, "说明")
	writeConclusion(&b, "这不是完整八字解读；需要模型静态与动态综合通过后，才输出主轴、用神、层次和岁运判断。")
	return strings.TrimSpace(b.String())
}

func buildStaticFactBullets(state baziCharterState) []string {
	items := []string{}
	if pillars := pillarFactSummary(state.Input.BaziResult["pillars"]); pillars != "" {
		items = append(items, "**四柱**："+pillars)
	}
	if dayGan := strings.TrimSpace(stringValue(state.Input.BaziResult["dayGan"])); dayGan != "" {
		items = append(items, "**日主**："+dayGan)
	}
	if strength := strings.TrimSpace(state.StaticSynthesis.StrengthBalance); strength != "" {
		items = append(items, "**扶抑证据**："+strength)
	} else if strength := strengthEvidenceSummary(state.Input.Yongshen); strength != "" {
		items = append(items, "**扶抑证据**："+strength)
	}
	if pattern := staticPatternFactSummary(state.Input); pattern != "" {
		items = append(items, "**结构事实**："+pattern)
	}
	if len(items) == 0 {
		return []string{"**事实状态**：工具未返回可展示的静态事实。"}
	}
	return items
}

func buildLiunianFactBullets(state baziCharterState) []string {
	items := []string{}
	if ganZhi := strings.TrimSpace(stringValue(state.Input.Liunian["liunian_ganzhi"])); ganZhi != "" {
		items = append(items, "**流年干支**："+ganZhi)
	}
	if tenGod := strings.TrimSpace(stringValue(state.Input.Liunian["liunian_shi_shen"])); tenGod != "" {
		items = append(items, "**流年干十神**："+tenGod)
	}
	if current := mapValue(state.Input.Liunian, "current_dayun"); len(current) > 0 {
		if ganZhi := strings.TrimSpace(stringValue(current["ganZhi"])); ganZhi != "" {
			items = append(items, "**当前大运**："+ganZhi)
		}
	}
	for _, relation := range relationTextList(state.Input.Liunian["liunian_chonghe"]) {
		items = append(items, "**关系事实**："+relation)
	}
	if len(items) == 0 {
		return []string{"**事实状态**：工具未返回可展示的流年事实。"}
	}
	return items
}

func dayunPeriods(dayun map[string]any) []map[string]any {
	raw := dayun["dayun_analyzed"]
	if wrapper, ok := raw.(map[string]any); ok {
		raw = wrapper["dayun_analyzed"]
	}
	return anyMapSlice(raw)
}

func anyMapSlice(raw any) []map[string]any {
	switch items := raw.(type) {
	case []map[string]any:
		return items
	case []map[string]string:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			value := make(map[string]any, len(item))
			for key, field := range item {
				value[key] = field
			}
			out = append(out, value)
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			switch value := item.(type) {
			case map[string]any:
				out = append(out, value)
			case map[string]string:
				converted := make(map[string]any, len(value))
				for key, field := range value {
					converted[key] = field
				}
				out = append(out, converted)
			}
		}
		return out
	default:
		return nil
	}
}

func renderFullTemplate(state baziCharterState) string {
	var b strings.Builder
	writeHeading(&b, "总览结论")
	writeConclusion(&b, buildOverviewConclusion(state))
	writeHighlightBlock(&b,
		"◎ 主轴",
		buildOverviewAxisSummary(state),
	)
	writeHighlightBlock(&b,
		"▲ 限制",
		buildOverviewLimitationSummary(state),
	)
	writeHighlightBlock(&b,
		"◇ 读法",
		"格局、扶抑与调候分层看；岁运只说明对主线的承接与扰动。",
	)

	writeHeading(&b, "强弱视角")
	writeConclusion(&b, buildStrengthConclusion(state))
	writeBullets(&b, []string{
		"**依据**：" + fallbackText(state.StaticSynthesis.Strength.Reasoning, fallbackText(state.StaticSynthesis.StrengthBalance, "未取得足够的扶抑证据。")),
		"**扶抑喜忌**：" + fallbackText(state.StaticSynthesis.Usage.Fuyi, "上游未提供扶抑喜忌。"),
		"**解释**：" + fallbackText(state.StaticSynthesis.Strength.Boundary, "扶抑只处理日主受力；格局取用与调候另行判断。"),
	})

	writeHeading(&b, "调候视角")
	writeConclusion(&b, buildTiaohouConclusion(state))
	writeBullets(&b, []string{
		"**依据**：" + fallbackText(state.StaticSynthesis.TiaohouAnchor, "未命中逐月调候规则。"),
		"**解释**：" + fallbackText(state.StaticSynthesis.TiaohouConstraint, "调候与扶抑、格局用神分层处理。"),
	})

	writeHeading(&b, "格局视角")
	writeConclusion(&b, buildPatternConclusion(state))
	writeBullets(&b, []string{
		"**规则口径**：" + ruleProfileLabel(state),
		"**依据**：" + fallbackText(state.StaticSynthesis.PatternBasis, "未取得格局主证。"),
		"**取用分层**：" + buildUseGodSummary(state),
		"**限制**：" + buildLimitationText(state),
	})

	writeHeading(&b, "大运验证")
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeConclusion(&b, "动态综合未通过，本轮仅展示已计算的大运事实，不判断趋势。")
		writeDayunAnalysis(&b, factsOnlyDayunPeriods(state))
	} else {
		writeConclusion(&b, buildDayunConclusion(state))
		writeDayunAnalysis(&b, renderedDayunPeriods(state))
	}

	writeHeading(&b, "流年应期")
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeConclusion(&b, "动态综合未通过，本轮仅展示流年事实，不展开应期。")
		writeBullets(&b, buildLiunianFactBullets(state))
	} else {
		writeConclusion(&b, buildLiunianConclusion(state))
		writeBullets(&b, []string{
			"**年性**：" + fallbackText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "上游未提供流年等级。"),
			"**依据**：" + joinOrDefault(state.DynamicSynthesis.TriggerSignals, "上游未提供触发点。"),
			"**限制**：" + buildDynamicConstraintText(state),
		})
	}

	writeHeading(&b, "综合判定")
	writeConclusion(&b, fallbackText(state.StaticSynthesis.TierJudgment, "上游未提供层次裁断。"))
	writeBullets(&b, []string{
		"**解释**：" + fallbackText(state.StaticSynthesis.TierBasis, "尚无足够的层次依据。"),
		"**岁运兑现**：" + buildTierRealizationText(state),
	})

	writeHeading(&b, "命格总结")
	writeBullets(&b, []string{
		"**最大优点**：" + joinOrDefault(state.StaticSynthesis.Advantages, "上游未提供优势裁断。"),
		"**最大风险**：" + joinOrDefault(state.StaticSynthesis.Risks, buildLimitationText(state)),
		"**用力方向**：" + buildProfileActionDirection(state),
		"**务实建议**：" + buildProfilePracticalAdvice(state),
	})
	return strings.TrimSpace(b.String())
}

func buildUseGodSummary(state baziCharterState) string {
	parts := []string{}
	if verdict := strings.TrimSpace(state.StaticSynthesis.Usage.Fuyi); verdict != "" {
		parts = append(parts, verdict)
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Pattern); usage != "" {
		parts = append(parts, usage)
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Tiaohou); usage != "" {
		parts = append(parts, usage)
	}
	if len(parts) == 0 {
		return "上游未提供取用分层。"
	}
	for i := range parts {
		parts[i] = strings.TrimRight(strings.TrimSpace(parts[i]), "。；")
	}
	return strings.Join(parts, "；") + "。"
}

func ruleProfileLabel(state baziCharterState) string {
	if label := strings.TrimSpace(state.StaticSynthesis.RuleProfile); label != "" {
		return label
	}
	if label := strings.TrimSpace(state.Input.RuleProfile.ID); label != "" {
		return label
	}
	return "未启用运行时规则 profile"
}

func buildOverviewConclusion(state baziCharterState) string {
	return fallbackText(state.StaticSynthesis.ReasoningSummary, state.StaticSynthesis.MainAxis)
}

func buildProfileActionDirection(state baziCharterState) string {
	_ = state
	return "上游未提供行动方向。"
}

func buildProfilePracticalAdvice(state baziCharterState) string {
	_ = state
	return "上游未提供务实建议。"
}

func buildOverviewAxisSummary(state baziCharterState) string {
	return fallbackText(state.StaticSynthesis.MainAxis, "上游未提供主轴裁断")
}

func buildOverviewLimitationSummary(state baziCharterState) string {
	return fallbackText(state.StaticSynthesis.CounterEvidence, "上游未提供限制裁断")
}

func renderTopicTemplate(plan baziAnalysisPlan, state baziCharterState, question string) string {
	var b strings.Builder
	topicMode := normalizedTopicMode(plan.TopicMode)
	writeHeading(&b, "直接回答")
	writeConclusion(&b, buildTopicDirectConclusion(plan, state, question))
	if focus := buildTopicFocusAnswer(plan, state); focus != "" {
		writeBullets(&b, []string{
			"**这次追问的关键**：" + focus,
		})
	} else {
		writeParagraphs(&b, []string{
			buildTopicDirectParagraph(state),
		})
	}

	writeHeading(&b, "命盘依据")
	switch topicMode {
	case "explain_term":
		writeConclusion(&b, "本节仅展示上游对该术语或句子的结构化说明。")
		writeBullets(&b, []string{
			"**结构落点**：" + buildTopicExplainPosition(state),
			"**命盘依据**：" + fallbackText(state.StaticSynthesis.AxisConsistency, fallbackText(state.StaticSynthesis.PatternOutcome, "上游未提供命盘依据。")),
			"**边界**：" + buildTopicExplainBoundary(state),
		})
	case "timing_reason":
		writeConclusion(&b, "本节仅展示上游提供的动态裁断。")
		writeBullets(&b, []string{
			"**原局裁断**：" + fallbackText(state.StaticSynthesis.MainAxis, "上游未提供原局裁断。"),
			"**岁运机制**：" + buildTierRealizationText(state),
			"**限制**：" + buildDynamicConstraintText(state),
		})
	case "conservative_reason":
		writeConclusion(&b, "本节仅展示上游提供的裁断与反证。")
		writeBullets(&b, []string{
			"**上游裁断**：" + fallbackText(state.StaticSynthesis.PatternOutcome, "上游未提供格局裁断。"),
			"**限制面**：" + buildLimitationText(state),
			"**层次依据**：" + fallbackText(state.StaticSynthesis.TierBasis, "上游未提供层次依据。"),
		})
	default:
		writeConclusion(&b, "本节仅展示上游提供的命盘裁断。")
		writeBullets(&b, []string{
			"**主轴**：" + fallbackText(state.StaticSynthesis.MainAxis, "上游未提供主轴裁断。"),
			"**上游裁断**：" + fallbackText(state.StaticSynthesis.PatternOutcome, "上游未提供格局裁断。"),
			"**限制**：" + buildTopicConstraintText(state),
		})
	}

	writeHeading(&b, "建议")
	switch topicMode {
	case "explain_term":
		writeConclusion(&b, "上游未提供行动建议。")
	case "timing_reason":
		writeConclusion(&b, "上游未提供行动建议。")
	case "conservative_reason":
		writeConclusion(&b, "上游未提供行动建议。")
	default:
		writeConclusion(&b, buildTopicAdviceConclusion(state))
	}
	return strings.TrimSpace(b.String())
}

func renderYearTemplate(state baziCharterState, question string) string {
	var b strings.Builder
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeHeading(&b, "年度判断")
		writeConclusion(&b, "动态综合未通过，本轮不输出年度趋势与应期判断。")
		writeParagraphs(&b, []string{"原局参考：" + fallbackText(state.StaticSynthesis.MainAxis, "静态综合未提供主轴裁断。")})

		writeHeading(&b, "作用机制")
		writeConclusion(&b, "本轮仅保留工具计算的流年与当前大运事实。")

		writeHeading(&b, "重点应期")
		writeConclusion(&b, "动态综合未通过，本轮不展开具体应期。")
		writeBullets(&b, buildLiunianFactBullets(state))

		writeHeading(&b, "建议")
		writeConclusion(&b, "本轮不依据未通过的动态综合给出行动建议。")
		return strings.TrimSpace(b.String())
	}
	writeHeading(&b, "年度判断")
	writeConclusion(&b, conclusionOrDefault("上游未提供年度裁断。", buildLiunianConclusion(state)))
	writeParagraphs(&b, []string{
		buildTopicDirectParagraph(state),
	})

	writeHeading(&b, "作用机制")
	writeConclusion(&b, "本节仅展示上游提供的动态推盘步骤。")
	writeSteps(&b, ensureSteps(state.DynamicSynthesis.ReasoningSteps, []string{
		"上游未提供动态推盘步骤。",
	}))

	writeHeading(&b, "重点应期")
	writeConclusion(&b, "本节仅展示上游提供的流年信息。")
	writeBullets(&b, []string{
		"**年性**：" + fallbackText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "上游未提供流年等级。"),
		"**触发点**：" + joinOrDefault(state.DynamicSynthesis.TriggerSignals, "本轮未给出更细触发点。"),
		"**应事领域**：" + fallbackText(state.DynamicSynthesis.LiunianFocus, "上游未提供应事领域。"),
		"**限制**：" + buildDynamicConstraintText(state),
	})

	writeHeading(&b, "建议")
	writeConclusion(&b, buildTopicAdviceConclusion(state))
	return strings.TrimSpace(b.String())
}

func buildStrengthConclusion(state baziCharterState) string {
	if conclusion := strings.TrimSpace(state.StaticSynthesis.Strength.Conclusion); conclusion != "" {
		if strings.HasPrefix(conclusion, "日主") {
			return conclusion + "。"
		}
		return "日主" + conclusion + "。"
	}
	if text := strings.TrimSpace(state.StaticSynthesis.StrengthBalance); text != "" {
		if index := strings.Index(text, "；"); index > 0 {
			return text[:index] + "。"
		}
		return text
	}
	return ""
}

func buildTiaohouConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouConstraint); text != "" {
		return text
	}
	return ""
}

func buildPatternConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.MainAxis); text != "" {
		return text
	}
	return ""
}

func buildDayunConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		return text
	}
	if periods := renderedDayunPeriods(state); len(periods) > 0 {
		return periodHeadline(periods[0])
	}
	return ""
}

func renderedDayunPeriods(state baziCharterState) []string {
	dynamic := state.DynamicSynthesis
	lines := dynamic.DayunPath
	if dynamic.Source != "recovered" && len(dynamic.DayunJudgments) > 0 {
		lines = renderDayunJudgmentLines(dynamic.DayunJudgments)
	}
	return attachDayunPeriodLabels(lines, dayunPeriods(state.Input.Dayun))
}

func attachDayunPeriodLabels(lines []string, periods []map[string]any) []string {
	lines = filterNonEmpty(lines)
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if i >= len(periods) {
			out = append(out, line)
			continue
		}
		label := dayunPeriodDisplayLabel(periods[i])
		if label == "" {
			out = append(out, line)
			continue
		}
		out = append(out, replaceDayunHeading(line, label))
	}
	return out
}

func dayunPeriodDisplayLabel(period map[string]any) string {
	ganZhi := strings.TrimSpace(stringValue(period["ganZhi"]))
	if ganZhi == "" {
		return ""
	}
	parts := []string{}
	startAge := strings.TrimSpace(anyToString(period["startAge"]))
	endAge := strings.TrimSpace(anyToString(period["endAge"]))
	if startAge != "" && endAge != "" && startAge != "<nil>" && endAge != "<nil>" {
		parts = append(parts, startAge+"-"+endAge+"岁")
	}
	startAt := shortPeriodTime(period["startAt"])
	endAt := shortPeriodTime(period["endAtExclusive"])
	if startAt != "" && endAt != "" {
		parts = append(parts, startAt+"至"+endAt+"前")
	}
	if len(parts) == 0 {
		return ganZhi + "运"
	}
	return ganZhi + "运（" + strings.Join(parts, "；") + "）"
}

func shortPeriodTime(raw any) string {
	text := strings.TrimSpace(anyToString(raw))
	if text == "" || text == "<nil>" {
		return ""
	}
	if len(text) >= len("2006-01-02 15:04") {
		return text[:len("2006-01-02 15:04")]
	}
	return text
}

func replaceDayunHeading(line, label string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if !strings.HasPrefix(line, "###") {
		return "### " + label + "\n**解读**：" + line
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "###"))
	heading := rest
	body := ""
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		heading = strings.TrimSpace(rest[:idx])
		body = strings.TrimSpace(rest[idx+1:])
	}
	suffix := ""
	if idx := strings.Index(heading, "："); idx >= 0 {
		suffix = heading[idx:]
	}
	if body == "" {
		return "### " + label + suffix
	}
	return "### " + label + suffix + "\n" + body
}

func renderDayunJudgmentLines(judgments []baziDayunJudgment) []string {
	lines := make([]string, 0, len(judgments))
	for _, judgment := range judgments {
		ganZhi := strings.TrimSpace(judgment.GanZhi)
		trend := strings.TrimSpace(judgment.Trend)
		interpretation := strings.TrimSpace(judgment.Interpretation)
		if ganZhi == "" || trend == "" || interpretation == "" {
			continue
		}
		parts := []string{fmt.Sprintf("### %s：%s", ganZhi, trend), "**解读**：" + interpretation}
		for _, evidence := range filterNonEmpty(judgment.Evidence) {
			parts = append(parts, "- **依据**："+evidence)
		}
		lines = append(lines, strings.Join(parts, "\n"))
	}
	return lines
}

func buildLiunianConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.DynamicSynthesis.LiunianFocus); text != "" {
		return text
	}
	if level := strings.TrimSpace(renderWindowLevel(state.DynamicSynthesis.WindowLevel)); level != "" {
		return "这一年更像" + level + "。"
	}
	return ""
}

func buildTopicDirectConclusion(plan baziAnalysisPlan, state baziCharterState, question string) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TopicDirectAnswer); text != "" {
		return text
	}
	switch normalizedTopicMode(plan.TopicMode) {
	case "timing_reason":
		return fallbackText(state.DynamicSynthesis.CurrentTrend, "上游未提供动态裁断。")
	}
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		return text
	}
	return "上游未提供本次追问的直接裁断。"
}

func buildTopicDirectParagraph(state baziCharterState) string {
	parts := make([]string, 0, 3)
	if text := strings.TrimSpace(state.StaticSynthesis.TopicFocusAnswer); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.StaticSynthesis.PatternOutcome); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.DynamicSynthesis.LiunianFocus); text != "" {
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return "上游未提供补充说明。"
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
	return fallbackText(state.StaticSynthesis.MainAxis, "上游未提供结构框架裁断。")
}

func buildTopicRouteText(state baziCharterState) string {
	return fallbackText(state.StaticSynthesis.PatternOutcome, "上游未提供结构路线裁断。")
}

func buildTopicExplainPosition(state baziCharterState) string {
	parts := filterNonEmpty([]string{
		buildTopicFrameText(state),
		buildTopicRouteText(state),
	})
	if len(parts) == 0 {
		return "上游未提供术语位置说明。"
	}
	return strings.Join(parts, " ")
}

func buildTopicExplainBoundary(state baziCharterState) string {
	return firstNonEmptyTrim(
		state.StaticSynthesis.CounterEvidence,
		"上游未提供术语解释边界。",
	)
}

func buildTopicAdviceConclusion(state baziCharterState) string {
	return "上游未提供行动建议。"
}

func buildTierRealizationText(state baziCharterState) string {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return "动态综合未通过，本轮不作岁运趋势裁断。"
	}
	return fallbackText(state.DynamicSynthesis.CurrentTrend, "上游未提供岁运兑现裁断。")
}

// factsOnlyDayunPeriods renders deterministic period facts for a mixed result.
// It is used when static interpretation succeeded but dynamic synthesis did not.
func factsOnlyDayunPeriods(state baziCharterState) []string {
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, state.StaticSynthesis, "")
	return attachDayunPeriodLabels(dynamic.DayunPath, dayunPeriods(state.Input.Dayun))
}

func buildTopicConstraintText(state baziCharterState) string {
	parts := []string{buildLimitationText(state)}
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		parts = append(parts, text)
	}
	if len(state.DynamicSynthesis.ConsistencyFlags) > 0 {
		parts = append(parts, strings.Join(filterNonEmpty(state.DynamicSynthesis.ConsistencyFlags), "；"))
	}
	return strings.Join(filterNonEmpty(parts), " ")
}

func buildDynamicConstraintText(state baziCharterState) string {
	parts := make([]string, 0, 4)
	if len(state.DynamicSynthesis.ConsistencyFlags) > 0 {
		parts = append(parts, strings.Join(filterNonEmpty(state.DynamicSynthesis.ConsistencyFlags), "；"))
	}
	if len(state.DynamicSynthesis.Risks) > 0 {
		parts = append(parts, joinOrDefault(state.DynamicSynthesis.Risks, "风险仍需保守看待。"))
	}
	if len(parts) == 0 {
		return buildLimitationText(state)
	}
	return strings.Join(uniqueText(parts), " ")
}

func buildLimitationText(state baziCharterState) string {
	parts := make([]string, 0, 4)
	if text := strings.TrimSpace(state.StaticSynthesis.CounterEvidence); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TierBasis); text != "" {
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return "上游未提供反证或限制。"
	}
	return strings.Join(uniqueText(parts), " ")
}

func renderWindowLevel(level string) string {
	switch strings.TrimSpace(level) {
	case "窗口年":
		return "窗口年"
	case "扰动年":
		return "扰动年"
	case "转折年":
		return "转折年"
	case "承压年":
		return "承压年"
	default:
		return strings.TrimSpace(level)
	}
}
func ensureSteps(src []string, fallback []string) []string {
	if len(filterNonEmpty(src)) > 0 {
		return filterNonEmpty(src)
	}
	return filterNonEmpty(fallback)
}

func filterNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func joinOrDefault(items []string, fallback string) string {
	items = uniqueText(items)
	if len(items) == 0 {
		return sanitizeUnsupportedFlourish(fallback)
	}
	return sanitizeUnsupportedFlourish(strings.Join(items, "；"))
}

// uniqueText removes exact repeated fallback and risk lines while preserving
// their first occurrence and source order. It does not merge similar evidence.
func uniqueText(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range filterNonEmpty(items) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return sanitizeUnsupportedFlourish(fallback)
	}
	return sanitizeUnsupportedFlourish(strings.TrimSpace(value))
}

func conclusionOrDefault(fallback string, values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func writeHeading(b *strings.Builder, heading string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("## ")
	b.WriteString(heading)
	b.WriteString("\n")
}

func writeConclusion(b *strings.Builder, text string) {
	b.WriteString("**结论：")
	b.WriteString(strings.TrimSpace(text))
	b.WriteString("**\n")
}

func writeParagraphs(b *strings.Builder, paragraphs []string) {
	for _, paragraph := range filterNonEmpty(paragraphs) {
		b.WriteString(paragraph)
		b.WriteString("\n")
	}
}

func writeBullets(b *strings.Builder, bullets []string) {
	for _, bullet := range filterNonEmpty(bullets) {
		b.WriteString("- ")
		b.WriteString(strings.TrimSpace(bullet))
		b.WriteString("\n")
	}
}

// writeDayunAnalysis keeps each luck period as an independent Markdown block.
// The upstream synthesis owns the verdict and evidence; rendering must not
// flatten the full analysis into a single nested list item.
func writeDayunAnalysis(b *strings.Builder, periods []string) {
	periods = filterNonEmpty(periods)
	if len(periods) == 0 {
		b.WriteString("当前大运尚无可解释的关系。\n")
		return
	}
	for _, period := range periods {
		b.WriteString(strings.TrimSpace(period))
		b.WriteString("\n\n")
	}
}

func writeSteps(b *strings.Builder, steps []string) {
	for i, step := range filterNonEmpty(steps) {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(step)))
	}
}

func writeHighlightBlock(b *strings.Builder, title, summary string, details ...string) {
	b.WriteString("> ")
	b.WriteString(strings.TrimSpace(title))
	b.WriteString("：")
	b.WriteString(strings.TrimSpace(summary))
	b.WriteString("\n")
	for _, detail := range filterNonEmpty(details) {
		b.WriteString("> ")
		b.WriteString(strings.TrimSpace(detail))
		b.WriteString("\n")
	}
}

func sanitizeUnsupportedFlourish(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"贵人众多", "助力较多",
		"福泽深厚", "福分较厚",
		"可享清福", "后程较稳",
	)
	return replacer.Replace(strings.TrimSpace(text))
}
