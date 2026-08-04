// This file belongs to the manager-owned runtime layer.
// It owns BaZi final rendering for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
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
	writeConclusion(&b, "本轮只列可复算命盘事实；暂不作主轴、层次、大运吉凶或具体应事。")
	writeBullets(&b, []string{
		"**输出范围**：排盘、强弱证据摘要、大运日期边界、十神与已计算关系。",
		"**静态状态**：本轮暂不输出主轴与层次裁断。",
		"**动态状态**：动态裁断受限时，仅展示大运与流年事实。",
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
	writeConclusion(&b, "这不是完整八字解读；需要静态与动态综合稳定后，才输出主轴、用神、层次和岁运判断。")
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
		labeledBullet("依据", firstDisplayText(state.StaticSynthesis.Strength.Reasoning, state.StaticSynthesis.StrengthBalance)),
		labeledBullet("扶抑喜忌", state.StaticSynthesis.Usage.Fuyi),
		labeledBullet("解释", state.StaticSynthesis.Strength.Boundary),
	})

	writeHeading(&b, "调候视角")
	writeConclusion(&b, buildTiaohouConclusion(state))
	writeBullets(&b, []string{
		labeledBullet("依据", state.StaticSynthesis.TiaohouAnchor),
		labeledBullet("解释", state.StaticSynthesis.TiaohouConstraint),
	})

	writeHeading(&b, "格局视角")
	writeConclusion(&b, buildPatternConclusion(state))
	writeBullets(&b, []string{
		"**规则口径**：" + ruleProfileLabel(state),
		labeledBullet("依据", state.StaticSynthesis.PatternBasis),
		labeledBullet("取用分层", buildUseGodSummary(state)),
		labeledBullet("限制", buildLimitationText(state)),
	})

	writeHeading(&b, "大运验证")
	if isMinorBaziSubject(state) {
		writeConclusion(&b, buildMinorDayunConclusion(state))
		writeBullets(&b, buildMinorDayunBullets(state))
	} else if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeConclusion(&b, buildFactsOnlyDayunConclusion(state))
		writeDayunAnalysis(&b, factsOnlyDayunPeriods(state))
	} else {
		writeConclusion(&b, buildDayunConclusion(state))
		writeDayunAnalysis(&b, renderedDayunPeriods(state))
	}

	writeHeading(&b, "流年应期")
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeConclusion(&b, "流年只展示干支、十神和已计算关系，不展开现实应事。")
		writeBullets(&b, buildLiunianFactBullets(state))
	} else {
		writeConclusion(&b, buildLiunianConclusion(state))
		writeBullets(&b, []string{
			labeledBullet("年性", renderWindowLevel(state.DynamicSynthesis.WindowLevel)),
			labeledBullet("依据", joinOrDefault(state.DynamicSynthesis.TriggerSignals, "")),
			labeledBullet("限制", buildDynamicConstraintText(state)),
		})
	}

	writeHeading(&b, "综合判定")
	writeConclusion(&b, state.StaticSynthesis.TierJudgment)
	writeBullets(&b, []string{
		labeledBullet("解释", state.StaticSynthesis.TierBasis),
		labeledBullet("岁运兑现", buildTierRealizationText(state)),
	})

	writeHeading(&b, "命格总结")
	writeBullets(&b, []string{
		labeledBullet("最大优点", joinOrDefault(state.StaticSynthesis.Advantages, "")),
		labeledBullet("最大风险", joinOrDefault(state.StaticSynthesis.Risks, buildLimitationText(state))),
		labeledBullet("用力方向", buildProfileActionDirection(state)),
		labeledBullet("务实建议", buildProfilePracticalAdvice(state)),
	})
	return strings.TrimSpace(b.String())
}

func buildUseGodSummary(state baziCharterState) string {
	parts := []string{}
	if verdict := strings.TrimSpace(state.StaticSynthesis.Usage.Fuyi); verdict != "" {
		parts = append(parts, conciseDisplayText(verdict, 96))
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Pattern); usage != "" {
		parts = append(parts, conciseDisplayText(usage, 96))
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Tiaohou); usage != "" {
		parts = append(parts, conciseDisplayText(usage, 96))
	}
	if len(parts) == 0 {
		return ""
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
	return "未选择专门规则口径"
}

func buildOverviewConclusion(state baziCharterState) string {
	axis := conciseDisplayText(firstDisplayText(state.StaticSynthesis.MainAxis, state.StaticSynthesis.PatternOutcome), 96)
	tier := conciseTierJudgmentText(state.StaticSynthesis.TierJudgment, 64)
	if axis != "" && tier != "" && !strings.Contains(axis, tier) {
		return strings.TrimRight(axis, "。；") + "；" + strings.TrimLeft(tier, "；")
	}
	return firstDisplayText(axis, conciseDisplayText(state.StaticSynthesis.ReasoningSummary, 160))
}

// conciseTierJudgmentText keeps the tier verdict visible in the overview. A
// legacy no-tier phrase is converted to the current bounded-grading contract.
func conciseTierJudgmentText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if strings.Contains(text, "证据不足") && strings.Contains(text, "暂不定级") {
		return "命格层次中等（保守定位）。"
	}
	return conciseDisplayText(text, maxRunes)
}

func buildProfileActionDirection(state baziCharterState) string {
	parts := []string{}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Pattern); usage != "" {
		parts = append(parts, "围绕格局取用："+conciseDisplayText(usage, 96))
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Tiaohou); usage != "" {
		parts = append(parts, "兼顾调候约束："+conciseDisplayText(usage, 96))
	}
	return strings.Join(filterNonEmpty(parts), "；")
}

func buildProfilePracticalAdvice(state baziCharterState) string {
	if context := buildBaziSubjectContext(state.Input); context.AgeBand == "infant" || context.AgeBand == "child" {
		return "只按结构观察成长节奏，不把命理信号转成成人现实领域判断。"
	}
	return ""
}

func buildOverviewAxisSummary(state baziCharterState) string {
	return firstDisplayText(conciseDisplayText(state.StaticSynthesis.MainAxis, 140), "本轮未形成主轴裁断")
}

func buildOverviewLimitationSummary(state baziCharterState) string {
	return firstDisplayText(conciseDisplayText(buildLimitationText(state), 140), "本轮未形成限制裁断")
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
			"**命盘依据**：" + firstDisplayText(state.StaticSynthesis.AxisConsistency, state.StaticSynthesis.PatternOutcome, "本轮未形成命盘依据。"),
			"**边界**：" + buildTopicExplainBoundary(state),
		})
	case "timing_reason":
		writeConclusion(&b, "本节仅展示上游提供的动态裁断。")
		writeBullets(&b, []string{
			"**原局裁断**：" + firstDisplayText(state.StaticSynthesis.MainAxis, "本轮未形成原局裁断。"),
			"**岁运机制**：" + buildTierRealizationText(state),
			"**限制**：" + buildDynamicConstraintText(state),
		})
	case "conservative_reason":
		writeConclusion(&b, "本节仅展示上游提供的裁断与反证。")
		writeBullets(&b, []string{
			labeledBullet("裁断", firstDisplayText(state.StaticSynthesis.PatternOutcome, "本轮未形成格局裁断。")),
			"**限制面**：" + buildLimitationText(state),
			"**层次依据**：" + firstDisplayText(state.StaticSynthesis.TierBasis, "本轮未形成层次依据。"),
		})
	default:
		writeConclusion(&b, "本节仅展示上游提供的命盘裁断。")
		writeBullets(&b, []string{
			"**主轴**：" + firstDisplayText(state.StaticSynthesis.MainAxis, "本轮未形成主轴裁断。"),
			labeledBullet("裁断", firstDisplayText(state.StaticSynthesis.PatternOutcome, "本轮未形成格局裁断。")),
			"**限制**：" + buildTopicConstraintText(state),
		})
	}

	writeHeading(&b, "建议")
	switch topicMode {
	case "explain_term":
		writeConclusion(&b, "这类术语解释只回答结构含义，不额外生成行动建议。")
	case "timing_reason":
		writeConclusion(&b, "本轮只解释岁运机制，现实行动仍需结合当下条件判断。")
	case "conservative_reason":
		writeConclusion(&b, "本轮以保守解释为主，不把命理结构转成具体行动指令。")
	default:
		writeConclusion(&b, buildTopicAdviceConclusion(state))
	}
	return strings.TrimSpace(b.String())
}

func renderYearTemplate(state baziCharterState, question string) string {
	var b strings.Builder
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeHeading(&b, "年度判断")
		writeConclusion(&b, "受授权边界限制，本轮只展示可复算年度事实，不判断现实应事。")
		writeParagraphs(&b, []string{"原局参考：" + fallbackText(state.StaticSynthesis.MainAxis, "静态综合未提供主轴裁断。")})

		writeHeading(&b, "作用机制")
		writeConclusion(&b, "本轮仅保留工具计算的流年与当前大运事实。")

		writeHeading(&b, "重点应期")
		writeConclusion(&b, "流年只展示干支、十神和已计算关系，不展开具体应期。")
		writeBullets(&b, buildLiunianFactBullets(state))

		writeHeading(&b, "建议")
		writeConclusion(&b, "本轮不依据未通过的动态综合给出行动建议。")
		return strings.TrimSpace(b.String())
	}
	writeHeading(&b, "年度判断")
	writeConclusion(&b, conclusionOrDefault("本轮未形成年度裁断。", buildLiunianConclusion(state)))
	writeParagraphs(&b, []string{
		buildTopicDirectParagraph(state),
	})

	writeHeading(&b, "作用机制")
	writeConclusion(&b, "本节仅展示上游提供的动态推盘步骤。")
	writeSteps(&b, ensureSteps(state.DynamicSynthesis.ReasoningSteps, []string{
		"本轮未形成动态推盘步骤。",
	}))

	writeHeading(&b, "重点应期")
	writeConclusion(&b, "本节仅展示上游提供的流年信息。")
	writeBullets(&b, []string{
		"**年性**：" + firstDisplayText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "本轮未形成流年等级。"),
		"**触发点**：" + joinOrDefault(state.DynamicSynthesis.TriggerSignals, "本轮未给出更细触发点。"),
		"**应事领域**：" + firstDisplayText(state.DynamicSynthesis.LiunianFocus, "本轮未形成应事领域。"),
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
		text = cleanUserVisibleText(text)
		if index := strings.Index(text, "；"); index > 0 {
			return text[:index] + "。"
		}
		return text
	}
	return ""
}

func buildTiaohouConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouAnchor); text != "" {
		return conciseDisplayText(text, 120)
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouConstraint); text != "" {
		return conciseDisplayText(text, 120)
	}
	return ""
}

func buildPatternConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.MainAxis); text != "" {
		return conciseDisplayText(text, 150)
	}
	return ""
}

func buildDayunConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		return conciseDisplayText(text, 120)
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
		return conciseDisplayText(text, 120)
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
		return firstDisplayText(state.DynamicSynthesis.CurrentTrend, "本轮未形成动态裁断。")
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
	if text := strings.TrimSpace(state.StaticSynthesis.PatternOutcome); text != "" {
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

func buildTierRealizationText(state baziCharterState) string {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return "动态裁断受授权边界限制，本轮不作岁运趋势裁断。"
	}
	return firstDisplayText(state.DynamicSynthesis.CurrentTrend, "本轮未形成岁运兑现裁断。")
}

// buildFactsOnlyDayunConclusion explains a dynamic fallback as a scope boundary,
// not as an internal model failure. The facts still come from deterministic tools.
func buildFactsOnlyDayunConclusion(state baziCharterState) string {
	if isMinorBaziSubject(state) {
		return "受主体年龄与授权边界限制，本轮只展示可复算大运事实与成长节奏观察。"
	}
	return "受授权边界限制，本轮只展示可复算大运事实，不判断吉凶趋势。"
}

// buildMinorDayunConclusion keeps child and adolescent readings on growth
// cadence even when the dynamic model returns a full luck-cycle analysis.
func buildMinorDayunConclusion(state baziCharterState) string {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return buildFactsOnlyDayunConclusion(state)
	}
	return "本轮按未成年人边界，只展示大运事实与成长节奏观察，不展开成人现实应事。"
}

// isMinorBaziSubject keeps child-specific presentation at the renderer edge.
// It does not change chart facts or synthesize a new reading.
func isMinorBaziSubject(state baziCharterState) bool {
	context := buildBaziSubjectContext(state.Input)
	switch context.AgeBand {
	case "infant", "child", "adolescent":
		return true
	default:
		return false
	}
}

// buildMinorFactsOnlyDayunBullets prevents a facts-only child reading from
// dumping the full adult luck-cycle table while keeping current/near facts visible.
func buildMinorFactsOnlyDayunBullets(state baziCharterState) []string {
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, state.StaticSynthesis, "")
	periods := attachDayunPeriodLabels(dynamic.DayunPath, dayunPeriods(state.Input.Dayun))
	bullets := []string{}
	if current := currentDayunFactText(dynamic, periods); current != "" {
		bullets = append(bullets, "**当前阶段**："+current)
	}
	if preview := dayunPreviewText(dynamic, periods, 3); preview != "" {
		bullets = append(bullets, "**大运事实节选**："+preview)
	}
	if relations := joinOrDefault(dynamic.TriggerSignals, ""); relations != "" {
		bullets = append(bullets, "**已计算关系**："+conciseDisplayText(relations, 120))
	}
	if len(bullets) == 0 {
		return []string{"**大运事实**：工具未返回可展示的大运边界。"}
	}
	return bullets
}

// buildMinorDayunBullets caps child display to current and near-term periods.
// It may show model wording already validated upstream, but never the full adult table.
func buildMinorDayunBullets(state baziCharterState) []string {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return buildMinorFactsOnlyDayunBullets(state)
	}
	periods := renderedDayunPeriods(state)
	bullets := []string{}
	if trend := conciseDisplayText(state.DynamicSynthesis.CurrentTrend, 120); trend != "" {
		bullets = append(bullets, "**成长节奏**："+trend)
	}
	if preview := dayunPreviewText(state.DynamicSynthesis, periods, 3); preview != "" {
		bullets = append(bullets, "**大运事实节选**："+preview)
	}
	if relations := joinOrDefault(state.DynamicSynthesis.TriggerSignals, ""); relations != "" {
		bullets = append(bullets, "**已计算关系**："+conciseDisplayText(relations, 120))
	}
	if len(bullets) == 0 {
		return []string{"**大运事实**：工具未返回可展示的大运边界。"}
	}
	return bullets
}

// currentDayunFactText returns the current period fact without presenting it as
// a trend verdict. Pre-start charts use the explicit boundary line.
func currentDayunFactText(dynamic baziDynamicSynthesis, periods []string) string {
	if dynamic.CurrentDayunIndex >= 0 && dynamic.CurrentDayunIndex < len(periods) {
		return periodHeadline(periods[dynamic.CurrentDayunIndex])
	}
	if len(dynamic.ReasoningSteps) > 1 {
		return strings.TrimPrefix(periodHeadline(dynamic.ReasoningSteps[1]), "当前大运事实：")
	}
	return ""
}

// dayunPreviewText lists only near-term period labels for child facts-only
// output; it is a display cap, not a chart-specific branch.
func dayunPreviewText(dynamic baziDynamicSynthesis, periods []string, limit int) string {
	if limit <= 0 || len(periods) == 0 {
		return ""
	}
	start := dynamic.CurrentDayunIndex
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(periods) {
		end = len(periods)
	}
	items := make([]string, 0, end-start)
	for _, period := range periods[start:end] {
		items = append(items, periodHeadline(period))
	}
	return strings.Join(filterNonEmpty(items), "；")
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
		parts = append(parts, conciseDisplayText(text, 120))
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TierBasis); text != "" {
		parts = append(parts, conciseDisplayText(text, 120))
	}
	if len(parts) == 0 {
		return "本轮未形成反证或限制。"
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
		return cleanUserVisibleText(fallback)
	}
	return cleanUserVisibleText(strings.Join(items, "；"))
}

// conciseDisplayText trims verbose model-safe boundary prose into one readable
// display sentence while preserving the upstream verdict's first-order meaning.
func conciseDisplayText(text string, maxRunes int) string {
	text = cleanUserVisibleText(text)
	if text == "" {
		return ""
	}
	clauses := splitDisplayClauses(text)
	if len(clauses) == 0 {
		return text
	}
	out := clauses[0]
	if len([]rune(out)) < maxRunes/2 && len(clauses) > 1 {
		out = strings.TrimRight(out, "。；") + "；" + clauses[1]
	}
	if maxRunes > 0 && len([]rune(out)) > maxRunes {
		runes := []rune(out)
		out = strings.TrimRight(string(runes[:maxRunes]), "，、；。 ") + "。"
	}
	if !strings.HasSuffix(out, "。") && !strings.HasSuffix(out, "！") && !strings.HasSuffix(out, "？") {
		out += "。"
	}
	return out
}

// splitDisplayClauses splits only on Chinese sentence-level separators so the
// renderer can cap repetition without parsing or re-adjudicating the reading.
func splitDisplayClauses(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '。', '；', ';', '\n':
			return true
		default:
			return false
		}
	})
	return filterNonEmpty(fields)
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
		return cleanUserVisibleText(fallback)
	}
	return cleanUserVisibleText(strings.TrimSpace(value))
}

func firstDisplayText(values ...string) string {
	for _, value := range values {
		if text := cleanUserVisibleText(value); text != "" {
			return text
		}
	}
	return ""
}

func labeledBullet(label, value string) string {
	value = cleanUserVisibleText(value)
	if value == "" {
		return ""
	}
	return "**" + strings.TrimSpace(label) + "**：" + value
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
		b.WriteString(cleanUserVisibleText(strings.TrimSpace(period)))
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

func cleanUserVisibleText(text string) string {
	text = sanitizeUnsupportedFlourish(text)
	text = sanitizeInternalBoundaryText(text)
	if text == "" || strings.Contains(text, "上游未提供") {
		return ""
	}
	return text
}

// sanitizeInternalBoundaryText removes engineering-oriented wording from
// user-visible renderer text without loosening any validation boundary.
func sanitizeInternalBoundaryText(text string) string {
	replacer := strings.NewReplacer(
		"证据不足，暂不定级", "命格层次中等（保守定位）",
		"暂不定级", "按保守标准定级",
		"调候规则未实现", "调候规则材料不足",
		"规则表未实现", "规则材料不足",
		"待规则表实现", "待规则材料补足",
		"待规则裁断", "待规则证据补足",
		"仅作结构观察", "只作结构说明",
		"证据不足", "证据还不够",
		"未启用运行时规则 profile", "未选择专门规则口径",
		"运行时规则 profile", "专门规则口径",
		"规则profile", "规则口径",
		"待profile裁断", "待规则证据补足",
		"动态综合未通过", "动态裁断受限",
		"模型动态综合不可用", "动态裁断受限",
		"runtime", "系统",
	)
	return replacer.Replace(strings.TrimSpace(text))
}
