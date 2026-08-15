// Package presentation 包含八字已验收结果的用户可见投影。
//
// 本文件负责按上游模板组织八字 Markdown 报告；
// 不重新裁断命理语义、不校验合同或调度图执行。
package presentation

import "strings"

func renderPresentationFactsOnlyDegradedTemplate(state FinalReplyInput) string {
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

	writeHeading(&b, "强弱事实")
	writeConclusion(&b, "以下仅展示工具计算的扶抑证据，不作喜忌或格局裁断。")
	writeBullets(&b, factsOnlyStrengthBullets(state))

	writeHeading(&b, "大运事实")
	writeConclusion(&b, "大运干支、年龄与日期边界由工具计算；本节不判趋势。")
	factsDynamic := state.FactsOnlyDynamicSynthesis
	writeDayunAnalysis(&b, attachDayunPeriodLabels(factsDynamic.DayunPath, state.Facts.DayunPeriods))

	writeHeading(&b, "流年事实")
	writeConclusion(&b, "流年仅展示工具事实，不展开应期。")
	writeBullets(&b, buildLiunianFactBullets(state))

	writeHeading(&b, "说明")
	writeConclusion(&b, "这不是完整八字解读；需要静态与动态综合稳定后，才输出主轴、用神、层次和岁运判断。")
	return strings.TrimSpace(b.String())
}

// factsOnlyStrengthBullets 只展示确定性扶抑证据，保证降级答复不伪装为强弱裁断。

// renderFullTemplate 先给出本命总览，再展开静态、全程和当前各自的已验收结论。
// 每层仍只消费所属投影，不重判命理结论。
func renderPresentationFullTemplate(state FinalReplyInput) string {
	var b strings.Builder
	writeFinalOverview(&b, state)

	writeHeading(&b, "强弱视角")
	writeConclusion(&b, buildStrengthConclusion(state))
	strengthBullets := []string{}
	seenStrength := map[string]struct{}{}
	for _, item := range []struct {
		label string
		value string
	}{
		{label: "依据", value: firstDisplayText(state.StaticSynthesis.Strength.Reasoning, state.StaticSynthesis.StrengthBalance)},
	} {
		value := strings.TrimSpace(item.value)
		key := displayFingerprint(value)
		if key == "" {
			continue
		}
		if _, exists := seenStrength[key]; exists {
			continue
		}
		seenStrength[key] = struct{}{}
		strengthBullets = append(strengthBullets, labeledBullet(item.label, value))
	}
	strengthBullets = append(strengthBullets, labeledBullet("说明", "扶抑只说明日主受力，不自动等同于格局取用或调候用神。"))
	writeBullets(&b, strengthBullets)

	writeHeading(&b, "调候视角")
	writeConclusion(&b, buildPresentationTiaohouConclusion(state))
	writeBullets(&b, []string{
		labeledBullet("依据", state.StaticSynthesis.TiaohouConstraint),
	})

	writeHeading(&b, "格局视角")
	writeConclusion(&b, withoutAxisEcho(state, state.StaticSynthesis.PatternOutcome, "格局取用与总览主轴一致，不再重复表述。"))
	writeBullets(&b, []string{
		"**规则口径**：" + ruleProfileLabel(state),
		labeledBullet("依据", buildPresentationPatternEvidence(state)),
	})

	writeSubheading(&b, "格局评价")
	writeBullets(&b, []string{
		labeledBullet("判读口径", tierAssessmentStandardText()),
	})
	writeConclusion(&b, firstDisplayText(state.StaticSynthesis.TierJudgment, "格局暂不立评（仅作结构观察）。"))
	writeBullets(&b, []string{
		labeledBullet("判定依据", state.StaticSynthesis.TierBasis),
	})
	writeClassicalReferences(&b, state)
	writeHighlightBlock(&b, "断语所限", buildOverviewLimitationSummary(state))

	if state.AnalysisPlan.NeedLifetimeDayun {
		writeHeading(&b, "全程运路")
		writeConclusion(&b, withoutAxisEcho(state, buildLifetimeDayunConclusion(state), "全程运路只说明各运对本命结构的承接与变化。"))
		writeBullets(&b, renderLifetimeDayunBullets(state))
	}

	writeHeading(&b, "当前应期")
	writeSubheading(&b, "当前大运")
	if isMinorBaziSubject(state) {
		writeConclusion(&b, buildMinorDayunConclusion(state))
		writeBullets(&b, buildMinorDayunBullets(state))
	} else if state.DynamicSynthesis.FactsOnly || limitsFortuneProse(state) {
		writeConclusion(&b, buildDayunConclusion(state))
		writeBullets(&b, factsOnlyCurrentDayunBullets(state))
	} else {
		writeConclusion(&b, buildDayunConclusion(state))
		// 当前阶段只消费 runtime 已绑定的单一步大运；全量 DayunPath 属于全程运路，
		// 不能借旧兼容字段再次渲染，否则会让两个章节的所有权重新混在一起。
		writeBullets(&b, factsOnlyCurrentDayunBullets(state))
	}

	writeSubheading(&b, "流年应期")
	if state.DynamicSynthesis.FactsOnly || limitsFortuneProse(state) {
		writeConclusion(&b, buildLiunianConclusion(state))
		writeBullets(&b, buildLiunianFactBullets(state))
	} else {
		writeConclusion(&b, buildLiunianConclusion(state))
		writeBullets(&b, []string{
			labeledBullet("年性", renderWindowLevel(state.DynamicSynthesis.WindowLevel)),
			labeledBullet("依据", joinOrDefault(state.DynamicSynthesis.TriggerSignals, "")),
			labeledBullet("限制", buildDynamicConstraintText(state)),
		})
	}

	return strings.TrimSpace(b.String())
}

// writeFinalOverview 在报告末尾收束本命主轴、层次、限制、发挥方向与阶段走势。
// 这些内容都来自已验证槽位，展示层不新增性格、事业或婚姻断语。

func renderPresentationTopicTemplate(state FinalReplyInput) string {
	var b strings.Builder
	topicMode := normalizedTopicMode(state.AnalysisPlan.TopicMode)
	writeHeading(&b, "直接回答")
	writeConclusion(&b, buildPresentationTopicDirectConclusion(state))
	if focus := buildPresentationTopicFocusAnswer(state); focus != "" {
		writeBullets(&b, []string{
			"**这次追问的关键**：" + focus,
		})
	} else {
		writeParagraphs(&b, []string{
			buildPresentationTopicDirectParagraph(state),
		})
	}

	writeHeading(&b, "命盘依据")
	switch topicMode {
	case "explain_term":
		writeConclusion(&b, "本节仅展示上游对该术语或句子的结构化说明。")
		writeBullets(&b, []string{
			"**结构落点**：" + buildPresentationTopicExplainPosition(state),
			"**命盘依据**：" + firstDisplayText(state.StaticSynthesis.AxisConsistency, state.StaticSynthesis.PatternOutcome, "本轮未形成命盘依据。"),
			"**边界**：" + buildPresentationTopicExplainBoundary(state),
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
			"**限制面**：" + buildPresentationLimitationText(state),
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
		writeConclusion(&b, buildPresentationTopicAdviceConclusion(state))
	}
	return strings.TrimSpace(b.String())
}

func renderPresentationYearTemplate(state FinalReplyInput) string {
	var b strings.Builder
	if state.DynamicSynthesis.FactsOnly || limitsFortuneProse(state) {
		writeHeading(&b, "年度判断")
		conclusion := "受授权边界限制，本轮只展示可复算年度事实，不判断现实应事。"
		if limitsFortuneProse(state) {
			conclusion = "格局评价尚未确定，本轮只展示可复算年度事实。"
		}
		writeConclusion(&b, conclusion)
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
		buildPresentationTopicDirectParagraph(state),
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
	writeConclusion(&b, buildPresentationTopicAdviceConclusion(state))
	return strings.TrimSpace(b.String())
}
