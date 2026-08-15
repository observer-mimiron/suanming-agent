// Package presentation 包含八字已验收结果的用户可见投影。
//
// 本文件负责生成完整报告的视角章节、总览和限制说明；
// 不改变上游层次、主轴、岁运或证据裁断，也不读取 runtime 状态。
package presentation

import (
	"regexp"
	"strings"
)

var classicalChapterHeadingPattern = regexp.MustCompile(`^[0-9一二三四五六七八九十百]+[、.．]\s*论`)

// writeFinalOverview 在报告末尾收束本命主轴、格局评价、限制、发挥方向与阶段走势。
// 这些内容都来自已验证槽位，展示层不新增性格、事业或婚姻断语。
func writeFinalOverview(b *strings.Builder, state FinalReplyInput) {
	writeHeading(b, "总览结论")

	writeSubheading(b, "本命总断")
	writeConclusion(b, buildOverviewAxisSummary(state))
	writeBullets(b, []string{
		labeledBullet("格局评价", buildOverviewTierSummary(state)),
		labeledBullet("可发挥之处", buildSummaryAdvantages(state)),
		labeledBullet("主要限制", buildSummaryRisks(state)),
		labeledBullet("发挥取向", buildProfileActionDirection(state)),
	})

	if state.AnalysisPlan.NeedLifetimeDayun {
		writeSubheading(b, "全程走势")
		writeConclusion(b, withoutAxisEcho(state, buildLifetimeDayunConclusion(state), "全程走势只按各运对本命结构的承接与变化观察。"))
	}

	writeSubheading(b, "当前阶段")
	switch {
	case isMinorBaziSubject(state):
		writeConclusion(b, buildMinorDayunConclusion(state))
	case state.DynamicSynthesis.FactsOnly:
		writeConclusion(b, "当前阶段仅保留可复算岁运事实，暂不判断趋势。")
	default:
		writeConclusion(b, buildDayunConclusion(state))
	}
}

// buildCombinedAssessmentConclusion 只并列已接受的本命格局评价、全程和当前判断，三层互不改写。
func buildCombinedAssessmentConclusion(state FinalReplyInput) string {
	if !state.AnalysisPlan.NeedLifetimeDayun {
		baseline := firstDisplayText(state.StaticSynthesis.TierJudgment, "格局暂不立评（仅作结构观察）。")
		if state.DynamicSynthesis.FactsOnly {
			return baseline + "；当前岁运仅保留可复算事实，暂不纳入走势评价。"
		}
		trend := withoutAxisEcho(state, firstDisplayText(state.DynamicSynthesis.CurrentTrend, "当前岁运走势暂未形成稳定裁断。"), "当前岁运按已绑定事实说明承接与扰动。")
		return baseline + "；当前岁运走势：" + trend
	}
	baseline := firstDisplayText(state.StaticSynthesis.TierJudgment, "格局暂不立评（仅作结构观察）。")
	lifetime := lifetimeTrajectorySummary(state.LifetimeSynthesis.Trajectory)
	if state.DynamicSynthesis.FactsOnly {
		return "本命格局评价：" + baseline + "；全程运路：" + lifetime + "；当前岁运仅保留可复算事实。"
	}
	trend := withoutAxisEcho(state, firstDisplayText(state.DynamicSynthesis.CurrentTrend, "当前岁运走势暂未形成稳定裁断。"), "当前岁运按已绑定事实说明承接与扰动。")
	return "本命格局评价：" + baseline + "；全程运路：" + lifetime + "；当前阶段：" + trend
}

// buildLifetimeDayunConclusion projects only the dedicated all-life DTO.

// buildLifetimeDayunConclusion projects only the dedicated all-life DTO.
func buildLifetimeDayunConclusion(state FinalReplyInput) string {
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

// lifetimeTrajectorySummary is the compact full-life verdict used in the combined section.
func lifetimeTrajectorySummary(trajectory string) string {
	labels := map[string]string{"smooth_realization": "整体承接较顺", "volatile_realization": "起伏中有兑现", "realization_with_breaks": "可成亦有破损", "mostly_constrained": "受制阶段较多", "withheld": "暂缓判断"}
	return firstDisplayText(labels[trajectory], "暂缓判断")
}

// renderLifetimeDayunBullets keeps every period separate instead of letting current dynamic overwrite it.

// writeClassicalReferences 只展示可读的短引文，并过滤检索元数据与残句。
// 引文用于说明取法，不自动生成命理结论；宁缺毋滥，避免把检索卡片当古籍正文。
func writeClassicalReferences(b *strings.Builder, citations []Citation) {
	lines := make([]string, 0, 2)
	seen := map[string]struct{}{}
	for _, citation := range citations {
		classic := strings.TrimSpace(citation.Classic)
		if classic == "" {
			continue
		}
		if !strings.HasPrefix(classic, "《") {
			classic = "《" + classic + "》"
		}
		for _, quote := range filterNonEmpty(citation.Quotes) {
			quote = classicalQuoteForDisplay(quote)
			if quote == "" {
				continue
			}
			line := "**" + classic + "**：" + quote
			if _, exists := seen[line]; exists {
				continue
			}
			seen[line] = struct{}{}
			lines = append(lines, line)
			break
		}
		if len(lines) == 2 {
			break
		}
	}
	if len(lines) == 0 {
		return
	}
	writeSubheading(b, "古籍参照")
	writeBullets(b, lines)
}

// classicalQuoteForDisplay 过滤元数据、章节标题和残句，只保留可读原文。

// classicalQuoteForDisplay 过滤元数据、章节标题和残句，只保留可读原文。
func classicalQuoteForDisplay(raw string) string {
	quote := strings.TrimSpace(raw)
	if quote == "" || len([]rune(quote)) < 4 || len([]rune(quote)) > 160 {
		return ""
	}
	if strings.ContainsAny(quote, "…>|") || strings.Contains(quote, "...") {
		return ""
	}
	for _, marker := range []string{"⭐", "清·", "民国·", "tags:", "tags：", "作者："} {
		if strings.Contains(quote, marker) {
			return ""
		}
	}
	if classicalChapterHeadingPattern.MatchString(quote) {
		return ""
	}
	return quote
}

// buildUseGodSummary combines only strength and seasonal lenses. Pattern text
// belongs to the overview and must not create a second main-axis rendering.
func buildUseGodSummary(state FinalReplyInput) string {
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

func ruleProfileLabel(state FinalReplyInput) string {
	if label := strings.TrimSpace(state.StaticSynthesis.RuleProfile); label != "" {
		return label
	}
	if label := strings.TrimSpace(state.Facts.RuleProfileID); label != "" {
		return label
	}
	return "本轮未启用专门规则裁断；检索材料只作依据，不直接生成规则结论"
}

func buildOverviewConclusion(state FinalReplyInput) string {
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) != "" {
		return "先看本命主轴与限制，再按评判标准给出基础层次；岁运部分只说明当前承接。"
	}
	return conciseDisplayText(state.StaticSynthesis.ReasoningSummary, 160)
}

// tierAssessmentStandardText explains the fixed nine-level lens before showing
// the selected level, so the rank is not presented as an unexplained label.

// tierAssessmentStandardText 说明格局评价取法，避免把内部证据量表误作古籍定级。
func tierAssessmentStandardText() string {
	return "按月令用神、成败救应、用神纯杂、有情有力、藏透与位置配合观察；不等同于财富、地位或人格价值。"
}

// conciseTierJudgmentText keeps the tier verdict visible without inventing a rank.

// conciseTierJudgmentText keeps the tier verdict visible without inventing a rank.
func conciseTierJudgmentText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	return conciseDisplayText(text, maxRunes)
}

func buildProfileActionDirection(state FinalReplyInput) string {
	parts := []string{}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Fuyi); usage != "" {
		parts = append(parts, "扶抑方向："+strings.TrimRight(conciseDisplayText(usage, 96), "。；"))
	}
	if usage := strings.TrimSpace(state.StaticSynthesis.Usage.Tiaohou); usage != "" {
		parts = append(parts, "调候方向："+strings.TrimRight(conciseDisplayText(usage, 96), "。；"))
	}
	if len(parts) == 0 {
		return "按总览主轴的已验证条件推进，并持续核对限制。"
	}
	return strings.Join(filterNonEmpty(parts), "；") + "。"
}

func buildProfilePracticalAdvice(state FinalReplyInput) string {
	if state.Facts.SubjectAgeBand == "infant" || state.Facts.SubjectAgeBand == "child" {
		return "只按结构观察成长节奏，不把命理信号转成成人现实领域判断。"
	}
	return ""
}

// buildSummaryAdvantages drops an exact main-axis echo so the summary adds a
// distinct advantage instead of restating the overview's only axis sentence.

// buildSummaryAdvantages drops an exact main-axis echo so the summary adds a
// distinct advantage instead of restating the overview's only axis sentence.
func buildSummaryAdvantages(state FinalReplyInput) string {
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

// buildSummaryRisks keeps the summary on independent restrictions instead of
// replaying the overview axis through a model-supplied risk sentence.
func buildSummaryRisks(state FinalReplyInput) string {
	items := make([]string, 0, len(state.StaticSynthesis.Risks))
	for _, item := range state.StaticSynthesis.Risks {
		if !isAxisEcho(state.StaticSynthesis.MainAxis, item) {
			items = append(items, item)
		}
	}
	return joinOrDefault(items, buildPresentationLimitationText(state))
}

// isAxisEcho detects only the same full sentence or a direct containment; it
// deliberately does not judge whether two different Chinese explanations agree.

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

// withoutAxisEcho replaces an exact repeated axis sentence with the local
// section's narrower purpose. It never chooses a new axis or interpretation.
func withoutAxisEcho(state FinalReplyInput, text, fallback string) string {
	if isAxisEcho(state.StaticSynthesis.MainAxis, text) {
		return fallback
	}
	return text
}

// displayFingerprint normalizes presentation punctuation for exact-echo control.

// displayFingerprint normalizes presentation punctuation for exact-echo control.
func displayFingerprint(text string) string {
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "，", "", "。", "", "；", "", "：", "", ",", "", ".", "", ";", "", ":", "")
	return replacer.Replace(strings.TrimSpace(text))
}

func buildOverviewAxisSummary(state FinalReplyInput) string {
	return firstDisplayText(conciseDisplayText(state.StaticSynthesis.MainAxis, 140), "本轮未形成主轴裁断")
}

// buildOverviewTierSummary 去掉上游格局评价字段可能携带的标题前缀，避免总览标签重复。
func buildOverviewTierSummary(state FinalReplyInput) string {
	text := conciseTierJudgmentText(state.StaticSynthesis.TierJudgment, 120)
	text = strings.TrimPrefix(text, "命格基础层次：")
	text = strings.TrimPrefix(text, "本命命格层次：")
	text = strings.TrimPrefix(text, "格局评价：")
	text = strings.TrimPrefix(text, "本命格局评价：")
	return firstDisplayText(text, "格局暂不立评（仅作结构观察）。")
}

func buildOverviewLimitationSummary(state FinalReplyInput) string {
	return firstDisplayText(conciseDisplayText(buildPresentationLimitationText(state), 140), "本轮未形成限制裁断")
}

func buildStrengthConclusion(state FinalReplyInput) string {
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

// buildTiaohouConclusion 优先展示已通过合同的调候 anchor，再处理证据边界提示。

// buildTiaohouConclusion 优先展示已通过合同的调候 anchor，再处理证据边界提示。
func buildPresentationTiaohouConclusion(state FinalReplyInput) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouAnchor); text != "" {
		return conciseDisplayText(text, 120)
	}
	if !state.Facts.FireEffectivenessKnown {
		return "调候有效性尚待确认；当前只按月令寒暖燥湿需求与火的出现位置观察。"
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouConstraint); text != "" {
		return conciseDisplayText(text, 120)
	}
	return ""
}

// buildPatternEvidence projects deterministic pattern prerequisites instead
// of replaying the model's free-text pattern verdict in a second section.

// buildPatternEvidence projects deterministic pattern prerequisites instead
// of replaying the model's free-text pattern verdict in a second section.
func buildPresentationPatternEvidence(state FinalReplyInput) string {
	parts := []string{}
	if state.Facts.MonthCommand != "" {
		parts = append(parts, "月令为"+state.Facts.MonthCommand+"，取格须与透干、通根、承接和反证同看")
	}
	if state.Facts.OfficialHidden && !state.Facts.OfficialVisible {
		parts = append(parts, "官星藏支未透")
	}
	if len(parts) == 0 {
		return "已按月令、透干、通根、承接和反证核对。"
	}
	return strings.Join(uniqueText(parts), "；") + "。"
}

func buildDayunConclusion(state FinalReplyInput) string {
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

func buildLiunianConclusion(state FinalReplyInput) string {
	if text := strings.TrimSpace(state.DynamicSynthesis.LiunianFocus); text != "" {
		return withoutAxisEcho(state, conciseDisplayText(text, 120), "流年只按当前岁运关系说明结构触发。")
	}
	if level := strings.TrimSpace(renderWindowLevel(state.DynamicSynthesis.WindowLevel)); level != "" {
		return "这一年更像" + level + "。"
	}
	return ""
}

func buildTierRealizationText(state FinalReplyInput) string {
	if state.DynamicSynthesis.FactsOnly {
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

func buildTopicConstraintText(state FinalReplyInput) string {
	parts := []string{buildPresentationLimitationText(state)}
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		parts = append(parts, text)
	}
	if len(state.DynamicSynthesis.ConsistencyFlags) > 0 {
		parts = append(parts, strings.Join(filterNonEmpty(state.DynamicSynthesis.ConsistencyFlags), "；"))
	}
	return strings.Join(filterNonEmpty(parts), " ")
}

func buildDynamicConstraintText(state FinalReplyInput) string {
	parts := make([]string, 0, 4)
	if len(state.DynamicSynthesis.ConsistencyFlags) > 0 {
		parts = append(parts, strings.Join(filterNonEmpty(state.DynamicSynthesis.ConsistencyFlags), "；"))
	}
	if len(state.DynamicSynthesis.Risks) > 0 {
		parts = append(parts, joinOrDefault(state.DynamicSynthesis.Risks, "风险仍需保守看待。"))
	}
	if len(parts) == 0 {
		return buildPresentationLimitationText(state)
	}
	return strings.Join(uniqueText(parts), " ")
}

func buildPresentationLimitationText(state FinalReplyInput) string {
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
