// Package runtime 包含 Manager 拥有的八字最终渲染。
//
// 本文件只把已验证的静态和动态槽位转成用户可见 Markdown；
// 不重判命理事实、不改写层次资格，也不承担模型修复或图调度。
package runtime

import (
	"fmt"
	"regexp"
	"strings"
)

var baziInternalReferencePath = regexp.MustCompile(`(?:dayun\[[0-9]+\](?:\.[A-Za-z0-9_]+)+|(?:liunian|yongshen|evidence_quality)(?:\.[A-Za-z0-9_]+)+|\b(?:support_score|pressure_score|tier_status)\b)`)

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

	writeHeading(&b, "强弱事实")
	writeConclusion(&b, "以下仅展示工具计算的扶抑证据，不作喜忌或格局裁断。")
	writeBullets(&b, factsOnlyStrengthBullets(state))

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

// factsOnlyStrengthBullets 只展示确定性扶抑证据，保证降级答复不伪装为强弱裁断。
func factsOnlyStrengthBullets(state baziCharterState) []string {
	strength := strings.TrimSpace(state.StaticSynthesis.StrengthBalance)
	if strength == "" {
		strength = strengthEvidenceSummary(state.Input.Yongshen)
	}
	if strength == "" {
		return []string{"**强弱证据**：工具未返回可展示的扶抑证据。"}
	}
	return []string{"**强弱证据**：" + strength}
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

// renderFullTemplate 以九级本命层次为第一层，再并列全程运路和当前阶段。
// 三层分别消费独立投影，避免岁运结论覆盖本命评级。
func renderFullTemplate(state baziCharterState) string {
	var b strings.Builder
	writeHeading(&b, "总览结论")
	writeConclusion(&b, buildOverviewAxisSummary(state))
	writeHighlightBlock(&b,
		"▲ 限制",
		buildOverviewLimitationSummary(state),
	)

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
	writeBullets(&b, strengthBullets)

	writeHeading(&b, "调候视角")
	writeConclusion(&b, buildTiaohouConclusion(state))
	writeBullets(&b, []string{
		labeledBullet("依据", state.StaticSynthesis.TiaohouConstraint),
	})

	writeHeading(&b, "格局视角")
	writeConclusion(&b, withoutAxisEcho(state, state.StaticSynthesis.PatternOutcome, "格局取用与总览主轴一致，不再重复表述。"))
	writeBullets(&b, []string{
		"**规则口径**：" + ruleProfileLabel(state),
		labeledBullet("依据", buildPatternEvidence(state)),
	})

	if state.AnalysisPlan.NeedLifetimeDayun {
		writeHeading(&b, "全程运路")
		writeConclusion(&b, buildLifetimeDayunConclusion(state))
		writeLifetimeDayunGroups(&b, state)
	}

	if state.AnalysisPlan.NeedLifetimeDayun {
		writeHeading(&b, "当前大运")
	} else {
		writeHeading(&b, "大运验证")
	}
	if isMinorBaziSubject(state) {
		writeConclusion(&b, buildMinorDayunConclusion(state))
		writeBullets(&b, buildMinorDayunBullets(state))
	} else if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		writeConclusion(&b, "当前大运仅保留可复算事实，暂不判断趋势。")
		writeBullets(&b, factsOnlyCurrentDayunBullets(state))
	} else {
		writeConclusion(&b, buildDayunConclusion(state))
		// 当前阶段只消费 runtime 已绑定的单一步大运；全量 DayunPath 属于全程运路，
		// 不能借旧兼容字段再次渲染，否则会让两个章节的所有权重新混在一起。
		writeBullets(&b, factsOnlyCurrentDayunBullets(state))
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

	writeHeading(&b, "评判标准")
	if state.AnalysisPlan.NeedLifetimeDayun {
		writeConclusion(&b, "本命命格层次仍按现有九级标准评价；再逐步观察全部大运对该结构的补、助、损、破；当前大运与流年只说明所处阶段和短期触发。")
	} else {
		writeConclusion(&b, tierAssessmentStandardText())
	}

	writeHeading(&b, "综合判定")
	if state.AnalysisPlan.NeedLifetimeDayun {
		writeSubheading(&b, "本命命格层次")
		writeConclusion(&b, strings.TrimPrefix(firstDisplayText(state.StaticSynthesis.TierJudgment, "本命命格层次暂未形成稳定裁断。"), "命格基础层次："))
		writeBullets(&b, []string{labeledBullet("判定依据", state.StaticSynthesis.TierBasis), labeledBullet("本命格局", state.StaticSynthesis.PatternOutcome)})

		writeSubheading(&b, "全程运路")
		writeConclusion(&b, lifetimeTrajectorySummary(state.LifetimeSynthesis.Trajectory))
		writeBullets(&b, []string{labeledBullet("全程结论", buildLifetimeDayunConclusion(state))})

		writeSubheading(&b, "当前阶段")
		writeConclusion(&b, buildDayunConclusion(state))
		writeBullets(&b, factsOnlyCurrentDayunBullets(state))
	} else {
		writeConclusion(&b, buildCombinedAssessmentConclusion(state))
		writeBullets(&b, []string{labeledBullet("判定依据", state.StaticSynthesis.TierBasis), labeledBullet("岁运兑现", buildTierRealizationText(state))})
	}

	return strings.TrimSpace(b.String())
}

// buildCombinedAssessmentConclusion 只并列已接受的本命九级层次、全程和当前判断，三层互不改写。
func buildCombinedAssessmentConclusion(state baziCharterState) string {
	if !state.AnalysisPlan.NeedLifetimeDayun {
		baseline := firstDisplayText(state.StaticSynthesis.TierJudgment, "本命基础层次暂未形成稳定裁断。")
		if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
			return baseline + "；当前岁运仅保留可复算事实，暂不纳入走势评价。"
		}
		trend := withoutAxisEcho(state, firstDisplayText(state.DynamicSynthesis.CurrentTrend, "当前岁运走势暂未形成稳定裁断。"), "当前岁运按已绑定事实说明承接与扰动。")
		return baseline + "；当前岁运走势：" + trend
	}
	baseline := strings.TrimPrefix(firstDisplayText(state.StaticSynthesis.TierJudgment, "本命命格层次暂未形成稳定裁断。"), "命格基础层次：")
	lifetime := lifetimeTrajectorySummary(state.LifetimeSynthesis.Trajectory)
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return "本命命格层次：" + baseline + "；全程运路：" + lifetime + "；当前岁运仅保留可复算事实。"
	}
	trend := withoutAxisEcho(state, firstDisplayText(state.DynamicSynthesis.CurrentTrend, "当前岁运走势暂未形成稳定裁断。"), "当前岁运按已绑定事实说明承接与扰动。")
	return "本命命格层次：" + baseline + "；全程运路：" + lifetime + "；当前阶段：" + trend
}

// buildLifetimeDayunConclusion projects only the dedicated all-life DTO.
func buildLifetimeDayunConclusion(state baziCharterState) string {
	lifetime := state.LifetimeSynthesis
	if !state.AnalysisPlan.NeedLifetimeDayun {
		return "本轮未请求全程大运综合。"
	}
	if lifetime.Status != "accepted" {
		return "全程运路暂缓判定，仅保留各步可复算事实。"
	}
	labels := map[string]string{"smooth_realization": "整体承接较顺", "volatile_realization": "起伏中有兑现", "realization_with_breaks": "可成亦有破损", "mostly_constrained": "受制阶段较多"}
	return firstDisplayText(lifetime.Summary, labels[lifetime.Trajectory], "全程运路暂未形成稳定裁断。")
}

// lifetimeTrajectorySummary is the compact full-life verdict used in the combined section.
func lifetimeTrajectorySummary(trajectory string) string {
	labels := map[string]string{"smooth_realization": "整体承接较顺", "volatile_realization": "起伏中有兑现", "realization_with_breaks": "可成亦有破损", "mostly_constrained": "受制阶段较多", "withheld": "暂缓判断"}
	return firstDisplayText(labels[trajectory], "暂缓判断")
}

// renderLifetimeDayunBullets keeps every period separate instead of letting current dynamic overwrite it.
func renderLifetimeDayunBullets(state baziCharterState) []string {
	if state.LifetimeSynthesis.Status != "accepted" {
		return []string{"**状态**：全程运路未通过完整合同，未以事实目录冒充综合判断。"}
	}
	items := make([]string, 0, len(state.LifetimeSynthesis.PeriodClaims))
	for _, claim := range state.LifetimeSynthesis.PeriodClaims {
		items = append(items, "**"+lifetimePeriodLabel(state, claim.PeriodRef)+"｜"+lifetimePeriodEffectLabel(claim.PeriodEffect)+"**："+conciseDisplayText(claim.Verdict, 100))
	}
	return items
}

// writeLifetimeDayunGroups keeps full coverage while grouping periods by their deterministic age boundary.
func writeLifetimeDayunGroups(b *strings.Builder, state baziCharterState) {
	if state.LifetimeSynthesis.Status != "accepted" {
		writeBullets(b, renderLifetimeDayunBullets(state))
		return
	}
	groups := []struct {
		label  string
		claims []baziLifetimeDayunClaim
	}{
		{label: "早期运程（29岁前）"},
		{label: "中期运程（30-59岁）"},
		{label: "后期运程（60岁后）"},
	}
	for _, claim := range state.LifetimeSynthesis.PeriodClaims {
		group := lifetimePeriodGroup(state, claim.PeriodRef)
		groups[group].claims = append(groups[group].claims, claim)
	}
	for _, group := range groups {
		if len(group.claims) == 0 {
			continue
		}
		writeSubheading(b, group.label)
		for _, claim := range group.claims {
			b.WriteString("\n#### ")
			b.WriteString(lifetimePeriodLabel(state, claim.PeriodRef))
			b.WriteString("\n")
			b.WriteString("**结构定位：")
			b.WriteString(lifetimePeriodEffectLabel(claim.PeriodEffect))
			b.WriteString("**\n")
			b.WriteString(conciseDisplayText(claim.Verdict, 100))
			b.WriteString("\n")
		}
	}
}

// lifetimePeriodGroup maps a deterministic age boundary to one presentation group.
func lifetimePeriodGroup(state baziCharterState, ref string) int {
	periods := dayunPeriods(state.Input.Dayun)
	index, ok := dynamicPeriodIndex(ref, periods)
	if !ok {
		return 1
	}
	startAge := intValue(periods[index]["startAge"])
	endAge := intValue(periods[index]["endAge"])
	if endAge > 0 && endAge <= 29 {
		return 0
	}
	if startAge >= 60 {
		return 2
	}
	return 1
}

func lifetimePeriodLabel(state baziCharterState, ref string) string {
	periods := dayunPeriods(state.Input.Dayun)
	if index, ok := dynamicPeriodIndex(ref, periods); ok {
		if label := dayunPeriodDisplayLabel(periods[index]); label != "" {
			return label
		}
	}
	return "大运"
}

func lifetimePeriodEffectLabel(effect string) string {
	labels := map[string]string{"complete_pattern": "补全结构", "support_use": "扶助用神", "carry_balance": "平衡承接", "damage_use": "损伤用神", "break_pattern": "破坏结构", "transform_pattern": "结构转化", "undetermined": "暂缓判断"}
	return firstDisplayText(labels[effect], "结构观察")
}

// buildUseGodSummary combines only strength and seasonal lenses. Pattern text
// belongs to the overview and must not create a second main-axis rendering.
func buildUseGodSummary(state baziCharterState) string {
	parts := []string{}
	if verdict := strings.TrimSpace(state.StaticSynthesis.Usage.Fuyi); verdict != "" {
		parts = append(parts, conciseDisplayText(verdict, 96))
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
	return "本轮未启用专门规则裁断；检索材料只作依据，不直接生成规则结论"
}

func buildOverviewConclusion(state baziCharterState) string {
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) != "" {
		return "先看本命主轴与限制，再按评判标准给出基础层次；岁运部分只说明当前承接。"
	}
	return conciseDisplayText(state.StaticSynthesis.ReasoningSummary, 160)
}

// tierAssessmentStandardText explains the fixed nine-level lens before showing
// the selected level, so the rank is not presented as an unexplained label.
func tierAssessmentStandardText() string {
	return "本命基础层次只评价命局结构，不等同于财富、地位或人格价值；本轮综合看主轴、有情、有力、清浊、病药、救应、调候与何知章印证，当前大运只说明承接，不改写本命层次。"
}

// conciseTierJudgmentText keeps the tier verdict visible without inventing a rank.
func conciseTierJudgmentText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	return conciseDisplayText(text, maxRunes)
}

func buildProfileActionDirection(state baziCharterState) string {
	parts := []string{}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Fuyi); usage != "" {
		parts = append(parts, "扶抑方向："+conciseDisplayText(usage, 96))
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Tiaohou); usage != "" {
		parts = append(parts, "调候方向："+conciseDisplayText(usage, 96))
	}
	if len(parts) == 0 {
		return "按总览主轴的已验证条件推进，并持续核对限制。"
	}
	return strings.Join(filterNonEmpty(parts), "；")
}

func buildProfilePracticalAdvice(state baziCharterState) string {
	if context := buildBaziSubjectContext(state.Input); context.AgeBand == "infant" || context.AgeBand == "child" {
		return "只按结构观察成长节奏，不把命理信号转成成人现实领域判断。"
	}
	return ""
}

// buildSummaryAdvantages drops an exact main-axis echo so the summary adds a
// distinct advantage instead of restating the overview's only axis sentence.
func buildSummaryAdvantages(state baziCharterState) string {
	items := make([]string, 0, len(state.StaticSynthesis.Advantages))
	for _, item := range state.StaticSynthesis.Advantages {
		if !isAxisEcho(state.StaticSynthesis.MainAxis, item) {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return "主轴的可用处仍以已验证的结构承接为准。"
	}
	return joinOrDefault(items, "")
}

// buildSummaryRisks keeps the summary on independent restrictions instead of
// replaying the overview axis through a model-supplied risk sentence.
func buildSummaryRisks(state baziCharterState) string {
	items := make([]string, 0, len(state.StaticSynthesis.Risks))
	for _, item := range state.StaticSynthesis.Risks {
		if !isAxisEcho(state.StaticSynthesis.MainAxis, item) {
			items = append(items, item)
		}
	}
	return joinOrDefault(items, buildLimitationText(state))
}

// isAxisEcho detects only the same full sentence or a direct containment; it
// deliberately does not judge whether two different Chinese explanations agree.
func isAxisEcho(axis, candidate string) bool {
	axis = displayFingerprint(axis)
	candidate = displayFingerprint(candidate)
	if axis == "" || candidate == "" {
		return false
	}
	return axis == candidate || strings.Contains(axis, candidate) || strings.Contains(candidate, axis)
}

// withoutAxisEcho replaces an exact repeated axis sentence with the local
// section's narrower purpose. It never chooses a new axis or interpretation.
func withoutAxisEcho(state baziCharterState, text, fallback string) string {
	if isAxisEcho(state.StaticSynthesis.MainAxis, text) {
		return fallback
	}
	return text
}

// displayFingerprint normalizes presentation punctuation for exact-echo control.
func displayFingerprint(text string) string {
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "，", "", "。", "", "；", "", "：", "", ",", "", ".", "", ";", "", ":", "")
	return replacer.Replace(strings.TrimSpace(text))
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
	if !buildBaziFactCapsule(state).FireEffectivenessKnown {
		return "调候有效性尚待确认；当前只按月令寒暖燥湿需求与火的出现位置观察。"
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouAnchor); text != "" {
		return conciseDisplayText(text, 120)
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouConstraint); text != "" {
		return conciseDisplayText(text, 120)
	}
	return ""
}

// buildPatternEvidence projects deterministic pattern prerequisites instead
// of replaying the model's free-text pattern verdict in a second section.
func buildPatternEvidence(state baziCharterState) string {
	facts := buildBaziFactCapsule(state)
	parts := []string{}
	if facts.MonthCommand != "" {
		parts = append(parts, "月令为"+facts.MonthCommand+"，取格须与透干、通根、承接和反证同看")
	}
	if facts.OfficialHidden && !facts.OfficialVisible {
		parts = append(parts, "官星藏支未透")
	}
	if len(parts) == 0 {
		return "已按月令、透干、通根、承接和反证核对。"
	}
	return strings.Join(uniqueText(parts), "；") + "。"
}

func buildDayunConclusion(state baziCharterState) string {
	role := renderCurrentPeriodRealization(state.DynamicSynthesis.CurrentPeriodRealization)
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		text = withoutAxisEcho(state, conciseDisplayText(text, 100), "当前大运只按已绑定事实说明对本命结构的承接。")
		if role != "" {
			return "当前大运承接：" + role + "；" + text
		}
		return text
	}
	if periods := renderedDayunPeriods(state); len(periods) > 0 {
		return periodHeadline(periods[0])
	}
	return ""
}

func renderedDayunPeriods(state baziCharterState) []string {
	return attachDayunPeriodLabels(state.DynamicSynthesis.DayunPath, dayunPeriods(state.Input.Dayun))
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
		return withoutAxisEcho(state, conciseDisplayText(text, 120), "流年只按当前岁运关系说明结构触发。")
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

func buildTierRealizationText(state baziCharterState) string {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return "动态裁断受授权边界限制，本轮不作岁运趋势裁断。"
	}
	role := renderCurrentPeriodRealization(state.DynamicSynthesis.CurrentPeriodRealization)
	trend := withoutAxisEcho(state, firstDisplayText(state.DynamicSynthesis.CurrentTrend, "本轮未形成岁运兑现裁断。"), "当前大运只按已绑定事实说明对本命结构的承接。")
	if role == "" {
		return trend
	}
	return "当前大运承接：" + role + "；" + trend
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

// factsOnlyCurrentDayunBullets keeps a dynamic fallback scoped to the period
// it can safely identify. The complete life-cycle directory remains available
// to tools, but is not a substitute for a missing dynamic interpretation.
func factsOnlyCurrentDayunBullets(state baziCharterState) []string {
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, state.StaticSynthesis, "")
	periods := attachDayunPeriodLabels(dynamic.DayunPath, dayunPeriods(state.Input.Dayun))
	if current := currentDayunFactText(dynamic, periods); current != "" {
		return []string{"**当前大运事实**：" + current}
	}
	return []string{"**当前大运事实**：工具未能定位当前大运。"}
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
		if !isAxisEcho(state.StaticSynthesis.MainAxis, text) {
			parts = append(parts, conciseDisplayText(text, 120))
		}
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

// renderCurrentPeriodRealization turns the dynamic enum into the separate
// current-dayun role shown beside, never inside, the natal base tier.
func renderCurrentPeriodRealization(value string) string {
	return map[string]string{
		"repair":   "修复",
		"assist":   "助成",
		"maintain": "维持",
		"disturb":  "扰动",
		"suppress": "压制",
	}[strings.TrimSpace(value)]
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

// writeSubheading creates a stable second-level section without changing the page-level outline.
func writeSubheading(b *strings.Builder, heading string) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString("### ")
	b.WriteString(heading)
	b.WriteString("\n")
}

// writeConclusion writes a sanitized conclusion, which is the final text sink
// for model-owned slot values.
func writeConclusion(b *strings.Builder, text string) {
	text = cleanUserVisibleText(text)
	if text == "" {
		text = "本轮未形成可展示结论。"
	}
	b.WriteString("**结论：")
	b.WriteString(strings.TrimSpace(text))
	b.WriteString("**\n")
}

// writeParagraphs writes only renderer-sanitized user-visible paragraphs.
func writeParagraphs(b *strings.Builder, paragraphs []string) {
	for _, paragraph := range filterNonEmpty(paragraphs) {
		paragraph = cleanUserVisibleText(paragraph)
		if paragraph == "" {
			continue
		}
		b.WriteString(paragraph)
		b.WriteString("\n")
	}
}

// writeBullets writes renderer-sanitized user-visible list items.
func writeBullets(b *strings.Builder, bullets []string) {
	for _, bullet := range filterNonEmpty(bullets) {
		bullet = cleanUserVisibleText(bullet)
		if bullet == "" {
			continue
		}
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

// writeSteps writes renderer-sanitized reasoning steps.
func writeSteps(b *strings.Builder, steps []string) {
	for i, step := range filterNonEmpty(steps) {
		step = cleanUserVisibleText(step)
		if step == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(step)))
	}
}

// writeHighlightBlock writes a sanitized short summary beneath the overview.
func writeHighlightBlock(b *strings.Builder, title, summary string, details ...string) {
	summary = cleanUserVisibleText(summary)
	if summary == "" {
		summary = "本轮未形成可展示结论。"
	}
	b.WriteString("> ")
	b.WriteString(strings.TrimSpace(title))
	b.WriteString("：")
	b.WriteString(strings.TrimSpace(summary))
	b.WriteString("\n")
	for _, detail := range filterNonEmpty(details) {
		detail = cleanUserVisibleText(detail)
		if detail == "" {
			continue
		}
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
	text = baziInternalReferencePath.ReplaceAllString(text, "已计算的结构事实")
	replacer := strings.NewReplacer(
		"暂不定级", "暂缓定级",
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
