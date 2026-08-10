// Package runtime 包含 Manager 拥有的八字最终渲染。
//
// 本文件负责把已验证的命盘、流年和大运事实整理为受限展示片段；
// 不负责生成事实、判断命理趋势或改变动态合同边界。
package runtime

import (
	"fmt"
	"strings"
)

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

// renderFullTemplate 按本命视角、全程运路、当前应期与末尾总览组织完整报告。
// 总览放在证据之后作收束；每层仍只消费所属投影，不重判命理结论。

// renderLifetimeDayunBullets keeps every period separate instead of letting current dynamic overwrite it.
func renderLifetimeDayunBullets(state baziCharterState) []string {
	if state.LifetimeSynthesis.Status != "accepted" {
		return []string{"**状态**：全程运路未通过完整合同，未以事实目录冒充综合判断。"}
	}
	items := make([]string, 0, len(state.LifetimeSynthesis.PeriodClaims))
	for _, claim := range state.LifetimeSynthesis.PeriodClaims {
		items = append(items, "**"+lifetimePeriodLabel(state, claim.PeriodRef)+"｜"+lifetimePeriodEffectLabel(claim.PeriodEffect)+"**："+lifetimePeriodStemTenGod(state, claim.PeriodRef)+"；"+lifetimePeriodEffectSummary(claim.PeriodEffect))
	}
	return items
}

// writeLifetimeDayunGroups keeps full coverage while listing periods in chronological order.

// writeLifetimeDayunGroups keeps full coverage while listing periods in chronological order.
func writeLifetimeDayunGroups(b *strings.Builder, state baziCharterState) {
	if state.LifetimeSynthesis.Status != "accepted" {
		writeBullets(b, renderLifetimeDayunBullets(state))
		return
	}
	for _, claim := range state.LifetimeSynthesis.PeriodClaims {
		b.WriteString("\n**")
		b.WriteString(lifetimePeriodLabel(state, claim.PeriodRef))
		b.WriteString("**\n")
		writeBullets(b, []string{
			labeledBullet("定位", lifetimePeriodEffectLabel(claim.PeriodEffect)+"；"+lifetimePeriodStemTenGod(state, claim.PeriodRef)),
			labeledBullet("说明", lifetimePeriodEffectSummary(claim.PeriodEffect)),
		})
	}
}

// lifetimePeriodStemTenGod 从工具结果展示运干十神，避免模型断语误标
// 已知的大运天干或十神事实。

// lifetimePeriodStemTenGod 从工具结果展示运干十神，避免模型断语误标
// 已知的大运天干或十神事实。
func lifetimePeriodStemTenGod(state baziCharterState, ref string) string {
	periods := dayunPeriods(state.Input.Dayun)
	index, ok := dynamicPeriodIndex(ref, periods)
	if !ok {
		return "工具未返回该步大运的运干十神。"
	}
	ganZhi := strings.TrimSpace(stringValue(periods[index]["ganZhi"]))
	tenGod := strings.TrimSpace(stringValue(periods[index]["tenGod"]))
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
		"complete_pattern":  "此运对本命结构形成补全作用，具体兑现仍须结合流年观察。",
		"support_use":       "此运对本命用神形成助力，具体力度仍以已计算关系为准。",
		"carry_balance":     "此运以平衡承接为主，顺逆仍须结合原局限制观察。",
		"damage_use":        "此运会削弱既有承接条件，宜以已计算关系保守观察。",
		"break_pattern":     "此运对既有结构的扰动较大，不预设具体现实应事。",
		"transform_pattern": "此运体现结构转化，是否成局仍须结合已声明事实判断。",
		"undetermined":      "本运仅保留已计算事实，不扩展趋势判断。",
	}[strings.TrimSpace(effect)]
}

// writeClassicalReferences 只展示可读的短引文，并过滤检索元数据与残句。
// 引文用于说明取法，不自动生成命理结论；宁缺毋滥，避免把检索卡片当古籍正文。

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
