// package domain This file belongs to the manager-owned runtime layer.
// It owns BaZi profile synthesis behavior for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	// baziSynthesisSourceFactsOnlyDegraded marks the honest fallback path: tool
	// facts may still be shown, but runtime code must not manufacture a reading.
	baziSynthesisSourceFactsOnlyDegraded = "facts_only_degraded"
	dynamicFlagStructureOnly             = "仅作结构观察"
)

// strengthEvidenceSummary 将确定性强弱证据压成事实摘要，不生成综合结论。
func strengthEvidenceSummary(yongshen map[string]any) string {
	strength := strings.TrimSpace(stringValue(yongshen["strength"]))
	evidence, _ := yongshen["strength_evidence"].(map[string]any)
	support := intValue(evidence["support_score"])
	pressure := intValue(evidence["pressure_score"])
	month := intValue(yongshen["month_score"])
	root := intValue(yongshen["root_count"])
	sameElement := intValue(yongshen["same_element"])
	generate := intValue(yongshen["generate_count"])
	if strength == "" {
		return "整体受力仍需保守判断。"
	}
	return fmt.Sprintf("日主%s；月令受力 %d，通根 %d 处，同类透干 %d，印星生扶 %d；扶身合计 %d，食伤泄身、财耗与官杀克合计 %d。", strength, month, root, sameElement, generate, support, pressure)
}

// anyToString 仅把动态事实载荷转换为展示文本。
func anyToString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

// buildProfileStaticSynthesis builds a facts-only degraded static artifact.
// Rule profile claims are materials for the model, not a deterministic author
// of final chart judgments, so this fallback intentionally withholds 主轴/层次.
func buildProfileStaticSynthesis(input baziCharterInput) baziStaticSynthesis {
	return buildFactsOnlyStaticSynthesis(input, "")
}

// buildProfileDynamicSynthesis builds a facts-only degraded dynamic artifact.
// The code may format 大运 facts, but it must not derive trend, 吉凶 or 应事.
func buildProfileDynamicSynthesis(input baziCharterInput, static baziStaticSynthesis) baziDynamicSynthesis {
	return buildFactsOnlyDynamicSynthesis(input, static, "")
}

func buildFactsOnlyStaticSynthesis(input baziCharterInput, reason string) baziStaticSynthesis {
	ruleProfile := strings.TrimSpace(input.RuleProfile.ID)
	strengthBalance := strengthEvidenceSummary(input.Yongshen)
	if strengthBalance == "" {
		strengthBalance = "工具未返回完整扶抑证据。"
	}
	patternBasis := staticPatternFactSummary(input)
	if patternBasis == "" {
		patternBasis = "工具未返回完整月令取格事实。"
	}
	recoveryReason := firstNonEmpty(reason, "模型静态综合不可用；本轮只展示可复算事实。")
	return baziStaticSynthesis{
		Source:                  baziSynthesisSourceFactsOnlyDegraded,
		RecoveryReason:          recoveryReason,
		RuleProfile:             ruleProfile,
		MainAxis:                "静态综合未通过，本轮不输出主轴裁断。",
		ClaimStrength:           "保守判断",
		SupportLevel:            "出现",
		LimitationLevel:         "明显",
		WordingCap:              "保守",
		ConsistencyFlags:        []string{"仅展示工具事实"},
		AxisLevel:               "结构可见",
		EffectOnTiaohou:         "中性",
		EffectOnCoreDisease:     "中性",
		EffectOnJiShenDirection: "中性",
		AxisCeiling:             "结构信号",
		ConflictReasons:         []string{"模型静态综合未通过，本轮不把规则材料投影成完整命理裁断。"},
		PatternBasis:            patternBasis,
		PatternOutcome:          "静态综合未通过，本轮不输出格局成败裁断。",
		CounterEvidence:         "静态综合未通过，本轮不输出反证裁断。",
		AxisConsistency:         "本轮只展示排盘与工具事实，不生成解释主轴。",
		TiaohouConstraint:       "静态综合未通过，本轮不输出调候裁断。",
		TiaohouAnchor:           "调候规则材料未被模型成功综合。",
		StrengthBalance:         strengthBalance,
		Strength: baziStrengthJudgment{
			Reasoning: strengthBalance,
			Boundary:  "这是工具事实摘要，不等同于完整强弱裁断。",
		},
		Usage: baziUsageLayers{
			Fuyi:     "静态综合未通过，本轮不输出扶抑喜忌。",
			Pattern:  "静态综合未通过，本轮不输出格局取用。",
			Tiaohou:  "静态综合未通过，本轮不输出调候用神。",
			Priority: "降级态只展示事实，不排序用神。",
		},
		PatternAndQingZhuo: "静态综合未通过，本轮不输出清浊成败裁断。",
		QiShiOrCongHua:     "静态综合未通过，本轮不输出气势从化裁断。",
		TierJudgment:       "静态综合未通过，本轮不输出层次裁断。",
		TierBasis:          "层次必须由模型综合事实与规则材料后给出；本轮综合未通过。",
		ReasoningSummary:   "静态综合未通过，本轮不输出主轴、层次或用神裁断。",
		ReasoningSteps: []string{
			"排盘、强弱证据、月令取格候选仍来自工具事实。",
			"模型静态综合未通过，因此 runtime 不把 profile 材料拼成完整裁断。",
		},
		Citations: profileCitations(input.RuleProfile),
	}
}

func buildFactsOnlyDynamicSynthesis(input baziCharterInput, static baziStaticSynthesis, reason string) baziDynamicSynthesis {
	current := mapValue(input.Liunian, "current_dayun")
	dayunName := currentDayunName(input, current)
	liunian := strings.TrimSpace(stringValue(input.Liunian["liunian_ganzhi"]))
	dayunRelations := relationTextForCurrentDayun(input.Dayun, dayunName)
	liunianRelations := relationTextList(input.Liunian["liunian_chonghe"])
	dayunLines, currentDayunLine, currentDayunIndex := buildDayunFactLines(input, dayunName)
	if currentDayunLine == "" {
		if dayunName == "" {
			currentDayunLine = currentDayunUnavailableLine(input)
		} else {
			currentDayunLine = fmt.Sprintf("当前%s大运：未在完整大运序列中匹配，以下仅列各步大运事实。", dayunName)
		}
	}
	liunianLine := buildLiunianFactLine(input, liunian, liunianRelations)
	recoveryReason := firstNonEmpty(reason, "动态裁断受授权边界限制；本轮只展示大运与流年事实。")
	out := baziDynamicSynthesis{
		Source:            baziSynthesisSourceFactsOnlyDegraded,
		RecoveryReason:    recoveryReason,
		CurrentTrend:      "受授权边界限制，本轮只展示可复算大运事实，不判吉凶趋势。",
		ClaimStrength:     "保守判断",
		SupportLevel:      "出现",
		LimitationLevel:   "明显",
		WordingCap:        "保守",
		ConsistencyFlags:  []string{dynamicFlagStructureOnly},
		DayunPath:         dayunLines,
		CurrentDayunIndex: currentDayunIndex,
		LiunianFocus:      "流年只展示干支、十神和已计算关系，不展开现实应事。",
		WindowLevel:       "仅事实",
		TriggerSignals:    append(dayunRelations, liunianRelations...),
		KeyWindows:        []string{"只保留大运、流年边界和关系事实。"},
		Risks:             []string{"本轮不输出风险判断。"},
		ReasoningSummary:  "只格式化大运干支、十神、日期边界和已计算关系；不生成趋势或应事。",
		ReasoningSteps: []string{
			"静态来源：" + firstNonEmpty(static.Source, "unknown"),
			"当前大运事实：" + periodHeadline(currentDayunLine),
			"流年事实：" + periodHeadline(liunianLine),
		},
	}
	return out
}

func currentDayunUnavailableLine(input baziCharterInput) string {
	targetAt, targetOK := parseProfileTime(input.Liunian["liunian_target_at"])
	firstStart := time.Time{}
	for _, period := range DayunPeriods(input.Dayun) {
		startAt, ok := parseProfileTime(period["startAt"])
		if ok && (firstStart.IsZero() || startAt.Before(firstStart)) {
			firstStart = startAt
		}
	}
	if targetOK && !firstStart.IsZero() && targetAt.Before(firstStart) {
		return "当前尚未交入第一步大运（起运时间：" + firstStart.Format("2006-01-02 15:04:05") + "）；以下列出后续各步大运。"
	}
	return "当前大运事实未能定位；以下仅列各步大运的结构观察。"
}

// currentDayunName first trusts the calendar tool's selection, then uses the
// annotated period boundaries to recover from a missing cache. It never picks
// an arbitrary period when the chart has not yet entered a luck cycle.
func currentDayunName(input baziCharterInput, current map[string]any) string {
	if ganZhi := strings.TrimSpace(stringValue(current["ganZhi"])); ganZhi != "" {
		return ganZhi
	}
	targetAt, ok := parseProfileTime(input.Liunian["liunian_target_at"])
	if !ok {
		return ""
	}
	for _, period := range DayunPeriods(input.Dayun) {
		startAt, startOK := parseProfileTime(period["startAt"])
		endAt, endOK := parseProfileTime(period["endAtExclusive"])
		if !startOK || !endOK || targetAt.Before(startAt) || !targetAt.Before(endAt) {
			continue
		}
		return strings.TrimSpace(stringValue(period["ganZhi"]))
	}
	return ""
}

// currentDayunIndexForInput returns the deterministic current-period index used
// by dynamic claims. It returns -1 when the chart has no safely bound period.
func currentDayunIndexForInput(input baziCharterInput) int {
	name := currentDayunName(input, mapValue(input.Liunian, "current_dayun"))
	if name == "" {
		return -1
	}
	for index, period := range DayunPeriods(input.Dayun) {
		if strings.TrimSpace(stringValue(period["ganZhi"])) == name {
			return index
		}
	}
	return -1
}

func staticPatternFactSummary(input baziCharterInput) string {
	parts := []string{}
	if pillars := pillarFactSummary(input.BaziResult["pillars"]); pillars != "" {
		parts = append(parts, pillars)
	}
	if dayGan := strings.TrimSpace(stringValue(input.BaziResult["dayGan"])); dayGan != "" {
		parts = append(parts, "日主："+dayGan)
	}
	if candidate := strings.TrimSpace(stringValue(input.Yongshen["geju_candidate"])); candidate != "" {
		parts = append(parts, "月令取格候选："+candidate)
	}
	if basis := strings.TrimSpace(stringValue(input.Yongshen["geju_basis"])); basis != "" {
		parts = append(parts, "取格依据："+basis)
	}
	if combo := strings.TrimSpace(stringValue(input.Yongshen["geju_combination"])); combo != "" {
		parts = append(parts, "组合事实："+combo)
	}
	return strings.Join(NonEmptyStrings(parts), "；")
}

func pillarFactSummary(raw any) string {
	pillars := profileMapSlice(raw)
	if len(pillars) == 0 {
		return ""
	}
	labels := []string{"年柱", "月柱", "日柱", "时柱"}
	parts := make([]string, 0, len(pillars))
	for i, pillar := range pillars {
		label := strings.TrimSpace(stringValue(pillar["name"]))
		if label == "" && i < len(labels) {
			label = labels[i]
		}
		stem := strings.TrimSpace(stringValue(pillar["stem"]))
		branch := strings.TrimSpace(stringValue(pillar["branch"]))
		ganZhi := strings.TrimSpace(stringValue(pillar["ganZhi"]))
		if ganZhi == "" {
			ganZhi = stem + branch
		}
		if label == "" || ganZhi == "" {
			continue
		}
		parts = append(parts, label+"："+ganZhi)
	}
	if len(parts) == 0 {
		return ""
	}
	return "四柱：" + strings.Join(parts, "，")
}

func buildDayunFactLines(input baziCharterInput, currentGanZhi string) ([]string, string, int) {
	periods := DayunPeriods(input.Dayun)
	lines := make([]string, 0, len(periods))
	currentLine := ""
	currentIndex := -1
	for _, period := range periods {
		ganZhi := strings.TrimSpace(stringValue(period["ganZhi"]))
		if ganZhi == "" {
			continue
		}
		line := buildDayunFactLine(period)
		lines = append(lines, line)
		if ganZhi == currentGanZhi {
			currentLine = line
			currentIndex = len(lines) - 1
		}
	}
	return lines, currentLine, currentIndex
}

func buildDayunFactLine(period map[string]any) string {
	ganZhi := strings.TrimSpace(stringValue(period["ganZhi"]))
	if ganZhi == "" {
		return "### 大运事实缺失"
	}
	lines := []string{"### " + ganZhi + "运"}
	if tenGod := strings.TrimSpace(stringValue(period["tenGod"])); tenGod != "" {
		lines = append(lines, "- **运干十神**："+tenGod)
	}
	startAge := strings.TrimSpace(anyToString(period["startAge"]))
	endAge := strings.TrimSpace(anyToString(period["endAge"]))
	if startAge != "" && endAge != "" && startAge != "<nil>" && endAge != "<nil>" {
		lines = append(lines, "- **年龄边界**："+startAge+"-"+endAge+"岁")
	}
	startAt := ShortPeriodTime(period["startAt"])
	endAt := ShortPeriodTime(period["endAtExclusive"])
	if startAt != "" && endAt != "" {
		lines = append(lines, "- **日期边界**："+startAt+"至"+endAt+"前")
	}
	for _, relation := range relationTextList(period["dayun_chonghe"]) {
		lines = append(lines, "- **关系事实**："+relation)
	}
	if len(lines) == 1 {
		lines = append(lines, "- **事实状态**：工具未返回更多大运字段。")
	}
	return strings.Join(lines, "\n")
}

func buildLiunianFactLine(input baziCharterInput, ganZhi string, relations []string) string {
	ganZhi = strings.TrimSpace(ganZhi)
	if ganZhi == "" {
		ganZhi = "流年"
	}
	lines := []string{"### " + ganZhi + "流年"}
	if tenGod := strings.TrimSpace(stringValue(input.Liunian["liunian_shi_shen"])); tenGod != "" {
		lines = append(lines, "- **流年干十神**："+tenGod)
	}
	for _, relation := range relations {
		lines = append(lines, "- **关系事实**："+relation)
	}
	if len(lines) == 1 {
		lines = append(lines, "- **事实状态**：工具未返回更多流年字段。")
	}
	return strings.Join(lines, "\n")
}

func parseProfileTime(raw any) (time.Time, bool) {
	text := strings.TrimSpace(stringValue(raw))
	if text == "" {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", text, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func periodHeadline(line string) string {
	headline := strings.Split(strings.TrimSpace(line), "\n")[0]
	headline = strings.ReplaceAll(headline, "**", "")
	headline = strings.TrimSpace(strings.TrimPrefix(headline, "###"))
	return strings.TrimSpace(headline)
}

func profileCitations(profile baziRuleProfile) []baziCitation {
	var citations []baziCitation
	for _, claim := range profile.Claims {
		citations = append(citations, claim.Citations...)
	}
	for _, verdict := range profile.Verdicts {
		citations = append(citations, verdict.Citations...)
	}
	return citations
}

func relationTextForCurrentDayun(dayun map[string]any, name string) []string {
	for _, period := range DayunPeriods(dayun) {
		if strings.TrimSpace(stringValue(period["ganZhi"])) == name {
			return relationTextList(period["dayun_chonghe"])
		}
	}
	return nil
}

// profileMapSlice 仅为 profile 合成兼容工具返回的对象数组形状。
func profileMapSlice(raw any) []map[string]any {
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
			if value, ok := item.(map[string]any); ok {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}
