// Package presentation 包含八字已验收结果的用户可见投影。
//
// 本文件负责把已验收的命盘、流年和大运事实整理为受限展示片段；
// 不读取原始工具载荷、不判断命理趋势或改变动态合同边界。
package presentation

import "strings"

// factsOnlyStrengthBullets 只展示确定性扶抑证据，保证降级答复不伪装为强弱裁断。
func factsOnlyStrengthBullets(state FinalReplyInput) []string {
	strength := strings.TrimSpace(state.StaticSynthesis.StrengthBalance)
	if strength == "" {
		strength = state.Facts.StrengthEvidence
	}
	if strength == "" {
		return []string{"**强弱证据**：工具未返回可展示的扶抑证据。"}
	}
	return []string{"**强弱证据**：" + strength}
}

func buildStaticFactBullets(state FinalReplyInput) []string {
	items := []string{}
	if pillars := strings.TrimSpace(state.Facts.PillarsSummary); pillars != "" {
		items = append(items, "**四柱**："+pillars)
	}
	if dayGan := strings.TrimSpace(state.Facts.DayMaster); dayGan != "" {
		items = append(items, "**日主**："+dayGan)
	}
	if strength := strings.TrimSpace(state.StaticSynthesis.StrengthBalance); strength != "" {
		items = append(items, "**扶抑证据**："+strength)
	} else if strength := strings.TrimSpace(state.Facts.StrengthEvidence); strength != "" {
		items = append(items, "**扶抑证据**："+strength)
	}
	if pattern := strings.TrimSpace(state.Facts.PatternSummary); pattern != "" {
		items = append(items, "**结构事实**："+pattern)
	}
	if len(items) == 0 {
		return []string{"**事实状态**：工具未返回可展示的静态事实。"}
	}
	return items
}

func buildLiunianFactBullets(state FinalReplyInput) []string {
	items := []string{}
	if ganZhi := strings.TrimSpace(state.Facts.LiunianGanZhi); ganZhi != "" {
		items = append(items, "**流年干支**："+ganZhi)
	}
	if tenGod := strings.TrimSpace(state.Facts.LiunianTenGod); tenGod != "" {
		items = append(items, "**流年干十神**："+tenGod)
	}
	if ganZhi := strings.TrimSpace(state.Facts.CurrentDayunGanZhi); ganZhi != "" {
		items = append(items, "**当前大运**："+ganZhi)
	}
	for _, relation := range state.Facts.LiunianRelations {
		items = append(items, "**关系事实**："+relation)
	}
	if len(items) == 0 {
		return []string{"**事实状态**：工具未返回可展示的流年事实。"}
	}
	return items
}

// renderLifetimeDayunBullets 以紧凑条目展示每步大运，避免当前运覆盖全程判断。
func renderLifetimeDayunBullets(state FinalReplyInput) []string {
	if state.LifetimeSynthesis.Status != "accepted" {
		return []string{"**状态**：全程运路未通过完整合同，未以事实目录冒充综合判断。"}
	}
	items := make([]string, 0, len(state.LifetimeSynthesis.PeriodClaims))
	for _, claim := range state.LifetimeSynthesis.PeriodClaims {
		items = append(items, "**"+lifetimePeriodLabel(state, claim.PeriodRef)+"｜"+lifetimePeriodEffectLabel(claim.PeriodEffect)+"**："+lifetimePeriodStemTenGod(state, claim.PeriodRef)+"；"+lifetimePeriodEffectSummary(claim.PeriodEffect))
	}
	return items
}

// lifetimePeriodStemTenGod 从工具结果展示运干十神，避免模型断语误标
// 已知的大运天干或十神事实。

// lifetimePeriodStemTenGod 从工具结果展示运干十神，避免模型断语误标
// 已知的大运天干或十神事实。
func lifetimePeriodStemTenGod(state FinalReplyInput, ref string) string {
	period, ok := periodByRef(state.Facts.DayunPeriods, ref)
	if !ok {
		return "工具未返回该步大运的运干十神。"
	}
	ganZhi := strings.TrimSpace(period.GanZhi)
	tenGod := strings.TrimSpace(period.TenGod)
	if ganZhi == "" || tenGod == "" {
		return "工具未返回该步大运的运干十神。"
	}
	runes := []rune(ganZhi)
	if len(runes) == 0 {
		return "工具未返回该步大运的运干十神。"
	}
	return string(runes[0]) + "为" + tenGod
}

// lifetimePeriodEffectSummary 将已接受的枚举转换为有边界的结构说明。
// 它不复用模型自由断语，避免已确定的运干或十神事实被不一致地重述。

// lifetimePeriodEffectSummary 将已接受的枚举转换为有边界的结构说明。
// 它不复用模型自由断语，避免已确定的运干或十神事实被不一致地重述。
func lifetimePeriodEffectSummary(effect string) string {
	return map[string]string{
		"complete_pattern":  "此运可补足本命结构。",
		"support_use":       "此运有助于发挥本命用神。",
		"carry_balance":     "此运侧重维持结构平衡。",
		"damage_use":        "此运会削弱原有承接。",
		"break_pattern":     "此运会扰动原有结构。",
		"transform_pattern": "此运体现结构转化。",
		"undetermined":      "本运仅保留已计算事实，不扩展趋势判断。",
	}[strings.TrimSpace(effect)]
}

// writeClassicalReferences 只展示可读的短引文，并过滤检索元数据与残句。
// 引文用于说明取法，不自动生成命理结论；宁缺毋滥，避免把检索卡片当古籍正文。

func lifetimePeriodLabel(state FinalReplyInput, ref string) string {
	if period, ok := periodByRef(state.Facts.DayunPeriods, ref); ok {
		if label := strings.TrimSpace(period.Label); label != "" {
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

func renderedDayunPeriods(state FinalReplyInput) []string {
	return attachDayunPeriodLabels(state.DynamicSynthesis.DayunPath, state.Facts.DayunPeriods)
}

func attachDayunPeriodLabels(lines []string, periods []DayunPeriod) []string {
	lines = filterNonEmpty(lines)
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if i >= len(periods) {
			out = append(out, line)
			continue
		}
		label := strings.TrimSpace(periods[i].Label)
		if label == "" {
			out = append(out, line)
			continue
		}
		out = append(out, replaceDayunHeading(line, label))
	}
	return out
}

// periodByRef resolves a display-period reference without reaching back into runtime state.
func periodByRef(periods []DayunPeriod, ref string) (DayunPeriod, bool) {
	for _, period := range periods {
		if strings.TrimSpace(period.Ref) == strings.TrimSpace(ref) {
			return period, true
		}
	}
	return DayunPeriod{}, false
}

// periodHeadline extracts the visible heading from an already-rendered period line.
func periodHeadline(line string) string {
	headline := strings.Split(strings.TrimSpace(line), "\n")[0]
	headline = strings.ReplaceAll(headline, "**", "")
	headline = strings.TrimSpace(strings.TrimPrefix(headline, "###"))
	return strings.TrimSpace(headline)
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

// buildFactsOnlyDayunConclusion explains a dynamic fallback as a scope boundary,
// not as an internal model failure. The facts still come from deterministic tools.
func buildFactsOnlyDayunConclusion(state FinalReplyInput) string {
	if isMinorBaziSubject(state) {
		return "受主体年龄与授权边界限制，本轮只展示可复算大运事实与成长节奏观察。"
	}
	return "受授权边界限制，本轮只展示可复算大运事实，不判断吉凶趋势。"
}

// buildMinorDayunConclusion keeps child and adolescent readings on growth
// cadence even when the dynamic model returns a full luck-cycle analysis.

// buildMinorDayunConclusion keeps child and adolescent readings on growth
// cadence even when the dynamic model returns a full luck-cycle analysis.
func buildMinorDayunConclusion(state FinalReplyInput) string {
	if limitsFortuneProse(state) {
		return buildDayunConclusion(state)
	}
	if state.DynamicSynthesis.FactsOnly {
		return buildFactsOnlyDayunConclusion(state)
	}
	return "本轮按未成年人边界，只展示大运事实与成长节奏观察，不展开成人现实应事。"
}

// isMinorBaziSubject keeps child-specific presentation at the renderer edge.
// It does not change chart facts or synthesize a new reading.

// isMinorBaziSubject keeps child-specific presentation at the renderer edge.
// It does not change chart facts or synthesize a new reading.
func isMinorBaziSubject(state FinalReplyInput) bool {
	switch state.Facts.SubjectAgeBand {
	case "infant", "child", "adolescent":
		return true
	default:
		return false
	}
}

// buildMinorFactsOnlyDayunBullets prevents a facts-only child reading from
// dumping the full adult luck-cycle table while keeping current/near facts visible.

// buildMinorFactsOnlyDayunBullets prevents a facts-only child reading from
// dumping the full adult luck-cycle table while keeping current/near facts visible.
func buildMinorFactsOnlyDayunBullets(state FinalReplyInput) []string {
	dynamic := state.FactsOnlyDynamicSynthesis
	periods := attachDayunPeriodLabels(dynamic.DayunPath, state.Facts.DayunPeriods)
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

// buildMinorDayunBullets caps child display to current and near-term periods.
// It may show model wording already validated upstream, but never the full adult table.
func buildMinorDayunBullets(state FinalReplyInput) []string {
	if state.DynamicSynthesis.FactsOnly || limitsFortuneProse(state) {
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

// currentDayunFactText returns the current period fact without presenting it as
// a trend verdict. Pre-start charts use the explicit boundary line.
func currentDayunFactText(dynamic DynamicSynthesis, periods []string) string {
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

// dayunPreviewText lists only near-term period labels for child facts-only
// output; it is a display cap, not a chart-specific branch.
func dayunPreviewText(dynamic DynamicSynthesis, periods []string, limit int) string {
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

// factsOnlyCurrentDayunBullets keeps a dynamic fallback scoped to the period
// it can safely identify. The complete life-cycle directory remains available
// to tools, but is not a substitute for a missing dynamic interpretation.
func factsOnlyCurrentDayunBullets(state FinalReplyInput) []string {
	dynamic := state.FactsOnlyDynamicSynthesis
	periods := attachDayunPeriodLabels(dynamic.DayunPath, state.Facts.DayunPeriods)
	if current := currentDayunFactText(dynamic, periods); current != "" {
		return []string{"**当前大运事实**：" + current}
	}
	return []string{"**当前大运事实**：工具未能定位当前大运。"}
}
