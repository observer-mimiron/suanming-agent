package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/mcp"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func validateStaticStage(state baziCharterState) error {
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return fmt.Errorf("missing static synthesis main axis")
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternBasis) == "" {
		return fmt.Errorf("missing static synthesis pattern basis")
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternOutcome) == "" {
		return fmt.Errorf("missing static synthesis pattern outcome")
	}
	if strings.TrimSpace(state.StaticSynthesis.CounterEvidence) == "" {
		return fmt.Errorf("missing static synthesis counter evidence")
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisConsistency) == "" {
		return fmt.Errorf("missing static synthesis axis consistency")
	}
	if strings.TrimSpace(state.StaticSynthesis.TiaohouAnchor) == "" {
		return fmt.Errorf("missing static synthesis tiaohou anchor")
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternAndQingZhuo) == "" {
		return fmt.Errorf("missing static synthesis pattern and qingzhuo")
	}
	if strings.TrimSpace(state.StaticSynthesis.TierJudgment) == "" {
		return fmt.Errorf("missing static synthesis tier judgment")
	}
	if strings.TrimSpace(state.StaticSynthesis.TierBasis) == "" {
		return fmt.Errorf("missing static synthesis tier basis")
	}
	if strings.TrimSpace(state.StaticSynthesis.ReasoningSummary) == "" {
		return fmt.Errorf("missing static synthesis reasoning summary")
	}
	if len(state.StaticSynthesis.ReasoningSteps) == 0 {
		return fmt.Errorf("missing static synthesis reasoning steps")
	}
	if strings.TrimSpace(state.StaticSynthesis.ClaimStrength) == "" {
		return fmt.Errorf("missing static synthesis claim strength")
	}
	if strings.TrimSpace(state.StaticSynthesis.SupportLevel) == "" {
		return fmt.Errorf("missing static synthesis support level")
	}
	if strings.TrimSpace(state.StaticSynthesis.LimitationLevel) == "" {
		return fmt.Errorf("missing static synthesis limitation level")
	}
	if strings.TrimSpace(state.StaticSynthesis.WordingCap) == "" {
		return fmt.Errorf("missing static synthesis wording cap")
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisLevel) == "" {
		return fmt.Errorf("missing static synthesis axis level")
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnTiaohou) == "" {
		return fmt.Errorf("missing static synthesis effect on tiaohou")
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnCoreDisease) == "" {
		return fmt.Errorf("missing static synthesis effect on core disease")
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnJiShenDirection) == "" {
		return fmt.Errorf("missing static synthesis effect on ji-shen direction")
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisCeiling) == "" {
		return fmt.Errorf("missing static synthesis axis ceiling")
	}
	if len(state.StaticSynthesis.ConflictReasons) == 0 {
		return fmt.Errorf("missing static synthesis conflict reasons")
	}
	return nil
}

func validateEvidenceBundlePreconditions(state baziCharterState) error {
	if state.EvidencePlan.NeedRetrieval && len(state.EvidencePlan.QueryPackets) == 0 {
		return fmt.Errorf("missing query packets for retrieval-required state")
	}
	return nil
}

func validateDynamicPreconditions(state baziCharterState) error {
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return fmt.Errorf("dynamic stage requires static synthesis first")
	}
	return nil
}

func validateDynamicStage(state baziCharterState) error {
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return fmt.Errorf("missing dynamic synthesis current trend")
	}
	if len(state.DynamicSynthesis.DayunPath) == 0 {
		return fmt.Errorf("missing dynamic synthesis dayun path")
	}
	if strings.TrimSpace(state.DynamicSynthesis.LiunianFocus) == "" {
		return fmt.Errorf("missing dynamic synthesis liunian focus")
	}
	if strings.TrimSpace(state.DynamicSynthesis.WindowLevel) == "" {
		return fmt.Errorf("missing dynamic synthesis window level")
	}
	if strings.TrimSpace(state.DynamicSynthesis.ReasoningSummary) == "" {
		return fmt.Errorf("missing dynamic synthesis reasoning summary")
	}
	if len(state.DynamicSynthesis.ReasoningSteps) == 0 {
		return fmt.Errorf("missing dynamic synthesis reasoning steps")
	}
	if strings.TrimSpace(state.DynamicSynthesis.ClaimStrength) == "" {
		return fmt.Errorf("missing dynamic synthesis claim strength")
	}
	if strings.TrimSpace(state.DynamicSynthesis.SupportLevel) == "" {
		return fmt.Errorf("missing dynamic synthesis support level")
	}
	if strings.TrimSpace(state.DynamicSynthesis.LimitationLevel) == "" {
		return fmt.Errorf("missing dynamic synthesis limitation level")
	}
	if strings.TrimSpace(state.DynamicSynthesis.WordingCap) == "" {
		return fmt.Errorf("missing dynamic synthesis wording cap")
	}
	return nil
}

func validateCharterConsistency(state baziCharterState) error {
	if err := validateStaticDecisionConsistency(state.StaticSynthesis); err != nil {
		return err
	}
	if err := validateStaticAxisVerdictConsistency(state.StaticSynthesis); err != nil {
		return err
	}
	if err := validateStaticAxisAgainstChartFacts(state); err != nil {
		return err
	}
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return nil
	}
	if err := validateDynamicDecisionConsistency(state.DynamicSynthesis); err != nil {
		return err
	}
	if err := validateCurrentDayunLineConsistency(state.DynamicSynthesis); err != nil {
		return err
	}
	return validateDynamicAgainstStaticCeiling(state.StaticSynthesis, state.DynamicSynthesis)
}

// validateCurrentDayunLineConsistency 约束当前大运总述与当前大运条目保持同线，
// 避免同一步运同时被写成“承托主轴”和“偏压/压制主轴”的相反口径。
func validateCurrentDayunLineConsistency(d baziDynamicSynthesis) error {
	currentTrend := strings.TrimSpace(d.CurrentTrend)
	if currentTrend == "" || len(d.DayunPath) == 0 {
		return nil
	}

	currentDayun := strings.TrimSpace(d.DayunPath[0])
	if currentDayun == "" {
		return nil
	}

	supportNeedles := []string{"承托", "承接", "托住", "助起", "配合"}
	pressureNeedles := []string{"偏压", "压制", "受压", "克制", "阻滞"}

	trendSupports := containsAnyText([]string{currentTrend}, supportNeedles)
	trendPressures := containsAnyText([]string{currentTrend}, pressureNeedles)
	dayunSupports := containsAnyText([]string{currentDayun}, supportNeedles)
	dayunPressures := containsAnyText([]string{currentDayun}, pressureNeedles)

	if trendSupports && dayunPressures && !dayunSupports {
		return fmt.Errorf("current dayun path contradicts current trend direction")
	}
	if trendPressures && dayunSupports && !dayunPressures {
		return fmt.Errorf("current dayun path contradicts current trend direction")
	}
	return nil
}

func validateStaticAxisVerdictConsistency(s baziStaticSynthesis) error {
	if strings.TrimSpace(s.AxisLevel) == "" &&
		strings.TrimSpace(s.EffectOnTiaohou) == "" &&
		strings.TrimSpace(s.EffectOnCoreDisease) == "" &&
		strings.TrimSpace(s.EffectOnJiShenDirection) == "" &&
		strings.TrimSpace(s.AxisCeiling) == "" &&
		len(s.ConflictReasons) == 0 {
		return nil
	}
	if err := validateAllowedValue("static axis level", s.AxisLevel, []string{"结构可见", "方向成立", "主轴成立", "可以拔高"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static effect on tiaohou", s.EffectOnTiaohou, []string{"支持", "中性", "冲突"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static effect on core disease", s.EffectOnCoreDisease, []string{"缓解", "中性", "放大"}); err != nil {
		return err
	}
	// `effect_on_jishen_direction` 的主合同已收口为 `缓解/中性/放大`，
	// 但旧测试样例和历史输出里仍可能出现同义的 `抑制`，这里保留兼容以避免被误杀。
	if err := validateAllowedValue("static effect on ji-shen direction", s.EffectOnJiShenDirection, []string{"缓解", "抑制", "中性", "放大"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static axis ceiling", s.AxisCeiling, []string{"结构信号", "受限路线", "可作主轴", "可以拔高"}); err != nil {
		return err
	}
	if len(s.ConflictReasons) == 0 {
		return fmt.Errorf("static conflict reasons required")
	}
	if err := validateAxisVerdictAgainstConflict(s); err != nil {
		return err
	}
	if s.AxisCeiling == "结构信号" &&
		containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"主轴", "贵格", "化杀为权"}) {
		return fmt.Errorf("static synthesis promotes structure signal beyond axis ceiling")
	}
	if s.AxisCeiling == "受限路线" &&
		containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"纯主轴贵格", "可以拔高", "化杀为权"}) {
		return fmt.Errorf("static synthesis promotes restricted route beyond axis ceiling")
	}
	return nil
}

func validateAxisVerdictAgainstConflict(s baziStaticSynthesis) error {
	conflictCount := 0
	if s.EffectOnTiaohou == "冲突" {
		conflictCount++
	}
	if s.EffectOnCoreDisease == "放大" {
		conflictCount++
	}
	if s.EffectOnJiShenDirection == "放大" {
		conflictCount++
	}
	if conflictCount == 0 {
		return nil
	}
	switch s.AxisCeiling {
	case "结构信号", "受限路线":
	default:
		return fmt.Errorf("static axis ceiling is too high for a conflict-amplifying route")
	}
	if s.AxisCeiling == "结构信号" && (s.AxisLevel == "主轴成立" || s.AxisLevel == "可以拔高") {
		return fmt.Errorf("static axis level exceeds structure-signal ceiling")
	}
	if s.AxisCeiling == "受限路线" && s.AxisLevel == "可以拔高" {
		return fmt.Errorf("static axis level exceeds restricted-route ceiling")
	}
	return nil
}

func validateDynamicAgainstStaticCeiling(s baziStaticSynthesis, d baziDynamicSynthesis) error {
	if s.AxisCeiling != "结构信号" && s.AxisCeiling != "受限路线" {
		return nil
	}
	if containsAnyText([]string{d.CurrentTrend, d.LiunianFocus, d.ReasoningSummary}, []string{
		"大成",
		"权位显赫",
		"化杀为权",
		"贵格已成",
		"一飞冲天",
	}) {
		return fmt.Errorf("dynamic synthesis escalates beyond static axis ceiling")
	}
	return nil
}

func validateStaticDecisionConsistency(s baziStaticSynthesis) error {
	if err := validateAllowedValue("static claim strength", s.ClaimStrength, []string{"保守判断", "倾向成立", "明确成立", "封顶判断"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static support level", s.SupportLevel, []string{"出现", "有根", "有气", "得力", "成势"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static limitation level", s.LimitationLevel, []string{"轻微", "明显", "核心硬伤"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static wording cap", s.WordingCap, []string{"保守", "中性", "明确", "封顶"}); err != nil {
		return err
	}
	if containsString(s.ConsistencyFlags, "方向成立但力度受限") &&
		!containsAnyText([]string{s.PatternOutcome, s.CounterEvidence, s.TierBasis}, []string{
			"力度受限",
			"条件受限",
			"受限",
			"不足以",
			"不算强救",
			"不够强",
			"药力不够",
			"药力有限",
			"层次受限",
			"难以拔高",
			"不能拔高",
			"难入上等",
			"难以进入上等",
		}) {
		return fmt.Errorf("static consistency flag requires visible limitation text")
	}
	if containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"一飞冲天"}) &&
		!allowsFlourishByWordingCap(s.WordingCap, "一飞冲天") {
		return fmt.Errorf("static synthesis overstates wording beyond wording cap")
	}
	if containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"可享清福", "福泽深厚", "贵人众多"}) &&
		!allowsFlourishByWordingCap(s.WordingCap, "positive_flourish") {
		return fmt.Errorf("static synthesis overstates wording beyond wording cap")
	}
	return nil
}

// validateStaticAxisAgainstChartFacts 在静态综合之后追加一层“跨流派裁决”校验，
// 防止低优先级的漂亮结构覆盖高优先级的调候硬约束与喜忌病点。
func validateStaticAxisAgainstChartFacts(state baziCharterState) error {
	axisTexts := []string{
		state.StaticSynthesis.MainAxis,
		state.StaticSynthesis.PatternBasis,
		state.StaticSynthesis.PatternOutcome,
		state.StaticSynthesis.CounterEvidence,
		state.StaticSynthesis.AxisConsistency,
		state.StaticSynthesis.TiaohouConstraint,
		state.StaticSynthesis.StrengthBalance,
		state.StaticSynthesis.TierBasis,
		state.StaticSynthesis.ReasoningSummary,
	}
	axisTexts = append(axisTexts, state.StaticSynthesis.ReasoningSteps...)
	if !containsAnyText(axisTexts, []string{"杀印相生", "化杀为权", "印化杀"}) {
		return nil
	}

	chartFacts := []string{
		anyToString(state.Input.Yongshen["strength"]),
		anyToString(state.Input.Yongshen["tiao_hou"]),
		state.StaticSynthesis.TiaohouConstraint,
		state.StaticSynthesis.TiaohouAnchor,
		state.StaticSynthesis.StrengthBalance,
		state.StaticSynthesis.CounterEvidence,
	}
	if !containsString(anySliceToStrings(state.Input.Yongshen["ji_shen"]), "水") {
		return nil
	}
	if !containsAnyText(chartFacts, []string{"需火调候", "火为第一调候", "调候为第一硬约束", "寒木待火", "寒冬木冷"}) {
		return nil
	}
	if !containsAnyText(chartFacts, []string{"忌水", "印水偏旺", "印旺", "水多", "寒湿", "印比偏旺"}) {
		return nil
	}
	if containsAnyText(axisTexts, []string{
		"方向成立但力度受限",
		"只可作结构信号",
		"不宜拔高",
		"不能拔高",
		"层次受限",
		"有其路数",
		"可见但",
		"只可作方向成立",
		"力度受限",
	}) {
		return nil
	}
	return fmt.Errorf("static main axis amplifies ji-shen or core disease direction without downgrade")
}

func validateDynamicDecisionConsistency(d baziDynamicSynthesis) error {
	if err := validateAllowedValue("dynamic claim strength", d.ClaimStrength, []string{"保守判断", "倾向成立", "明确成立", "封顶判断"}); err != nil {
		return err
	}
	if err := validateAllowedValue("dynamic support level", d.SupportLevel, []string{"出现", "有根", "有气", "得力", "成势"}); err != nil {
		return err
	}
	if err := validateAllowedValue("dynamic limitation level", d.LimitationLevel, []string{"轻微", "明显", "核心硬伤"}); err != nil {
		return err
	}
	if err := validateAllowedValue("dynamic wording cap", d.WordingCap, []string{"保守", "中性", "明确", "封顶"}); err != nil {
		return err
	}
	if containsAnyText([]string{d.CurrentTrend}, []string{"一路顺", "明显起飞", "全面起飞"}) &&
		containsAnyText(d.DayunPath, []string{"吉中有阻", "承压", "放大病点", "限制", "有阻"}) {
		return fmt.Errorf("dynamic current trend conflicts with mixed dayun path")
	}
	if d.WindowLevel == "窗口年" && d.WordingCap != "封顶" &&
		containsAnyText([]string{d.LiunianFocus, d.ReasoningSummary}, []string{"关键翻身", "一飞冲天", "彻底起势"}) {
		return fmt.Errorf("dynamic liunian focus overstates a window year")
	}
	if d.WindowLevel == "承压年" &&
		containsAnyText([]string{d.LiunianFocus, d.ReasoningSummary}, []string{"明显起飞", "一飞冲天", "高歌猛进"}) {
		return fmt.Errorf("dynamic liunian focus conflicts with pressure-year judgment")
	}
	if containsString(d.ConsistencyFlags, "吉中有阻") &&
		!containsAnyText(append([]string{d.CurrentTrend, d.LiunianFocus}, d.DayunPath...), []string{"吉中有阻", "有阻", "并存", "限制"}) {
		return fmt.Errorf("dynamic consistency flag 吉中有阻 requires visible mixed wording")
	}
	if containsString(d.ConsistencyFlags, "机会伴随强变动") &&
		!containsAnyText([]string{d.CurrentTrend, d.LiunianFocus, d.ReasoningSummary}, []string{
			"机会伴随强变动",
			"吉中带险",
			"变动",
			"不宜激进",
			"可主动求变",
			"求变",
			"起伏",
			"波折",
			"反复",
			"不算稳",
			"不稳",
			"扰动",
			"宜边走边看",
		}) {
		return fmt.Errorf("dynamic consistency flag 机会伴随强变动 requires visible volatility wording")
	}
	return nil
}

func validateFinalWriterOutput(plan baziAnalysisPlan, state baziCharterState, output string) error {
	output = strings.TrimSpace(output)
	if output == "" {
		return fmt.Errorf("final writer produced empty output")
	}
	switch plan.WriterTemplate {
	case "full":
		if err := validateOrderedHeadings(output, []string{
			"## 总览结论",
			"## 强弱视角",
			"## 调候视角",
			"## 格局视角",
			"## 大运验证",
			"## 流年应期",
			"## 综合判定",
			"## 命格总结",
		}); err != nil {
			return err
		}
		if !strings.Contains(output, "## 命格总结") {
			return fmt.Errorf("full writer output must preserve 命格总结 section")
		}
		if strings.Count(output, "**结论：") < 7 {
			return fmt.Errorf("full writer output must expose bold conclusion lines")
		}
		overviewSection := sectionContent(output, "## 总览结论", "## 强弱视角")
		if overviewSection == "" {
			return fmt.Errorf("full writer output missing 总览结论 section body")
		}
		if !containsAnyText([]string{overviewSection}, []string{"◎ 主轴", "▲ 限制", "◇ 读法", "古法提要"}) {
			return fmt.Errorf("full writer output must expose layered overview summary blocks")
		}
		gejuSection := sectionContent(output, "## 格局视角", "## 大运验证")
		if gejuSection == "" {
			return fmt.Errorf("full writer output missing 格局视角 section body")
		}
		if !containsAnyText([]string{gejuSection}, []string{"**本轮总纲**"}) {
			return fmt.Errorf("full writer output must expose methodology outline in 格局视角")
		}
		if !containsAnyText([]string{gejuSection}, []string{"1.", "2."}) {
			return fmt.Errorf("full writer output must preserve numbered reasoning steps in 格局视角")
		}
		if !containsAnyText([]string{gejuSection}, []string{"**主证依据**"}) {
			return fmt.Errorf("full writer output must expose primary evidence in 格局视角")
		}
		if !containsAnyText([]string{gejuSection}, []string{"**为何成立**", "**限制在哪里**"}) {
			return fmt.Errorf("full writer output must expose both support and limitation in 格局视角")
		}
		if !containsAnyText([]string{gejuSection}, []string{"### 证据矩阵", "#### 格局主轴", "#### 调候边界", "#### 扶抑受力", "#### 反证与限制"}) {
			return fmt.Errorf("full writer output must expose evidence matrix in 格局视角")
		}
		tierSection := sectionContent(output, "## 综合判定", "## 命格总结")
		if tierSection == "" {
			return fmt.Errorf("full writer output missing 综合判定 section body")
		}
		if !containsAnyText([]string{tierSection}, []string{"**层次依据**"}) {
			return fmt.Errorf("full writer output must expose tier basis in 综合判定")
		}
	case "topic":
		if err := validateOrderedHeadings(output, []string{
			"## 直接回答",
			"## 命盘依据",
			"## 建议",
		}); err != nil {
			return err
		}
		if strings.Count(output, "**结论：") < 3 {
			return fmt.Errorf("topic writer output must expose bold conclusion lines")
		}
	case "year":
		if err := validateOrderedHeadings(output, []string{
			"## 年度判断",
			"## 作用机制",
			"## 重点应期",
			"## 建议",
		}); err != nil {
			return err
		}
		if strings.Count(output, "**结论：") < 1 {
			return fmt.Errorf("year writer output must expose bold conclusion line")
		}
	}
	if containsAnyText([]string{output}, []string{"贵人众多", "福泽深厚", "可享清福"}) &&
		!allowsFlourishByWordingCap(state.StaticSynthesis.WordingCap, "positive_flourish") {
		return fmt.Errorf("final writer output contains unsupported positive flourish claims")
	}
	if strings.TrimSpace(state.StaticSynthesis.CounterEvidence) != "" &&
		!containsAnyText([]string{output}, []string{"受限", "限制", "不足", "难以"}) {
		return fmt.Errorf("final writer output dropped limitation signals")
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "机会伴随强变动") &&
		!containsAnyText([]string{output}, []string{"机会伴随强变动", "吉中带险", "不宜激进"}) {
		return fmt.Errorf("final writer output dropped dynamic volatility constraint")
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "吉中有阻") &&
		!containsAnyText([]string{output}, []string{"吉中有阻", "有阻", "并存"}) {
		return fmt.Errorf("final writer output dropped mixed dynamic constraint")
	}
	if state.DynamicSynthesis.WindowLevel == "窗口年" && state.DynamicSynthesis.WordingCap != "封顶" &&
		containsAnyText([]string{output}, []string{"关键翻身年", "一飞冲天", "彻底翻身"}) {
		return fmt.Errorf("final writer output overstates a window year")
	}
	return nil
}

func validateOrderedHeadings(output string, headings []string) error {
	last := -1
	for _, heading := range headings {
		index := strings.Index(output, heading)
		if index < 0 {
			return fmt.Errorf("final writer output missing heading: %s", heading)
		}
		if index <= last {
			return fmt.Errorf("final writer output has heading out of order: %s", heading)
		}
		last = index
	}
	return nil
}

func sectionContent(output, heading, nextHeading string) string {
	start := strings.Index(output, heading)
	if start < 0 {
		return ""
	}
	start += len(heading)
	end := len(output)
	if nextHeading != "" {
		if next := strings.Index(output[start:], nextHeading); next >= 0 {
			end = start + next
		}
	}
	return strings.TrimSpace(output[start:end])
}

func validateAllowedValue(name, value string, allowed []string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("missing %s", name)
	}
	if !containsString(allowed, value) {
		return fmt.Errorf("invalid %s: %s", name, value)
	}
	return nil
}

func containsAnyText(texts []string, needles []string) bool {
	for _, text := range texts {
		for _, needle := range needles {
			if needle != "" && strings.Contains(text, needle) {
				return true
			}
		}
	}
	return false
}

func allowsFlourishByWordingCap(wordingCap, flourishClass string) bool {
	wordingCap = strings.TrimSpace(wordingCap)
	switch flourishClass {
	case "positive_flourish":
		return wordingCap == "明确" || wordingCap == "封顶"
	case "一飞冲天":
		return wordingCap == "封顶"
	default:
		return false
	}
}

func anySliceToStrings(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(anyToString(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		if text := strings.TrimSpace(anyToString(value)); text != "" {
			return []string{text}
		}
		return nil
	}
}

func anyToString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}

func shouldUseBaziCharterGraph(plan ExecutionPlan) bool {
	if plan.Route.PrimaryDomain != "bazi" {
		return false
	}
	switch plan.FollowupMode {
	case "", followupModeRerunSpecialist:
		// 非 follow-up，或 manager 已明确要求继续走 specialist 主链，才允许进入八字 inner graph。
	default:
		return false
	}
	return len(plan.Domains) == 1 && plan.Domains[0] == "bazi"
}

func (e *Executor) runBaziAuthorityFirstGraph(ctx context.Context, sink EventSink, st *state.SessionState, question string) (string, error) {
	if st == nil || !st.HasBaziResult() {
		err := fmt.Errorf("pure bazi charter graph requires bazi result")
		annotateBaziGraphError(ctx, "bootstrap", err)
		return "", err
	}

	chartState := baziCharterState{
		Input: baziCharterInput{
			UserQuestion: question,
			BaziResult:   st.BaziResult,
			Yongshen:     mapValue(st.BaziResult, "yongshen"),
			Dayun:        mapValue(st.BaziResult, "dayun_analyzed"),
			Liunian:      mapValue(st.BaziResult, "liunian"),
		},
	}

	analysisPlan, err := e.runBaziAnalysisPlanner(ctx, st, question, chartState.Input)
	if err != nil {
		annotateBaziGraphError(ctx, "analysis_planner", err)
		analysisPlan = defaultBaziAnalysisPlan(question)
	}
	analysisPlan = normalizeBaziAnalysisPlan(analysisPlan)
	chartState.AnalysisPlan = analysisPlan
	emitBaziStageThinking(ctx, sink, "bazi_graph", analysisPlan.StageSummary)

	plan, bundle, quality, err := e.runBaziEvidenceStage(ctx, st, question, chartState.Input, analysisPlan)
	if err != nil {
		annotateBaziGraphError(ctx, "evidence_stage", err)
		return "", err
	}
	chartState.EvidencePlan = plan
	chartState.EvidenceBundle = bundle
	chartState.EvidenceQuality = quality
	chartState, err = e.maybeReflectOnBaziEvidence(ctx, st, chartState)
	if err != nil {
		annotateBaziGraphError(ctx, "evidence_reflection", err)
		return "", err
	}
	if err := validateEvidenceBundlePreconditions(chartState); err != nil {
		annotateBaziGraphError(ctx, "evidence_validation", err)
		return "", err
	}
	emitBaziStageThinking(ctx, sink, "bazi_graph", buildEvidenceStageSummary(chartState))

	chartState.StaticSynthesis, err = e.runStaticSynthesisWithFeedback(chartState, func(payload map[string]any) (baziStaticSynthesis, error) {
		return runBaziInnerAgentJSON[baziStaticSynthesis](ctx, e.builder, baziStaticSynthesisConfig(), st, buildBaziCharterPrompt("静态综合", question, payload))
	})
	if err != nil {
		annotateBaziGraphError(ctx, "static_synthesis", err)
		return "", err
	}
	chartState.StaticSynthesis = normalizeStaticSynthesis(chartState.StaticSynthesis)
	emitBaziStageThinking(ctx, sink, "bazi_graph", buildStaticStageSummary(chartState))
	emitBaziReasoningSteps(ctx, sink, "静态推演", chartState.StaticSynthesis.ReasoningSteps)

	if chartState.AnalysisPlan.NeedDynamic {
		chartState, err = e.supplementDynamicEvidenceIfNeeded(ctx, st, question, chartState)
		if err != nil {
			annotateBaziGraphError(ctx, "dynamic_evidence", err)
			return "", err
		}
		chartState.DynamicSynthesis, err = e.runDynamicSynthesis(ctx, st, chartState, question)
		if err != nil {
			annotateBaziGraphError(ctx, "dynamic_synthesis", err)
			return "", err
		}
		chartState.DynamicSynthesis = normalizeDynamicSynthesis(chartState.DynamicSynthesis)
		if err := validateDynamicStage(chartState); err != nil {
			chartState.DynamicSynthesis = recoverDynamicSynthesis(chartState, chartState.DynamicSynthesis, err)
		}
		if err := validateCharterConsistency(chartState); err != nil {
			chartState.DynamicSynthesis = recoverDynamicSynthesis(chartState, chartState.DynamicSynthesis, err)
			if recoverErr := validateDynamicStage(chartState); recoverErr != nil {
				annotateBaziGraphError(ctx, "dynamic_validation", recoverErr)
				return "", recoverErr
			}
			if recoverErr := validateCharterConsistency(chartState); recoverErr != nil {
				annotateBaziGraphError(ctx, "dynamic_consistency", recoverErr)
				return "", recoverErr
			}
		}
		emitBaziStageThinking(ctx, sink, "bazi_graph", buildDynamicStageSummary(chartState))
		emitBaziReasoningSteps(ctx, sink, "动态推演", chartState.DynamicSynthesis.ReasoningSteps)
	}

	return e.runFinalWriter(ctx, st, chartState, question)
}

func annotateBaziGraphError(ctx context.Context, stage string, err error) {
	if err == nil {
		return
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.graph.error_stage": stage,
		"bazi.graph.error":       err.Error(),
	})
}

// supplementDynamicEvidenceIfNeeded 在首轮完整看盘这类“静态主判 + 动态补证”场景中，
// 追加一次动态证据规划与检索，避免 dynamic_synthesis 只拿到静态格局证据。
func (e *Executor) supplementDynamicEvidenceIfNeeded(ctx context.Context, st *state.SessionState, question string, chartState baziCharterState) (baziCharterState, error) {
	if !chartState.AnalysisPlan.NeedDynamic {
		return chartState, nil
	}
	if chartState.AnalysisPlan.RetrievalStage == "dynamic" {
		return chartState, nil
	}
	// 首轮完整看盘优先消费系统已就绪的 dayun_analyzed / liunian 字段，
	// 不再默认补第二轮动态检索。只有动态基础事实缺失时才补证。
	if hasDynamicSystemFacts(chartState.Input) {
		return chartState, nil
	}

	dynamicPlan := chartState.AnalysisPlan
	dynamicPlan.RetrievalStage = "dynamic"
	dynamicPlan.StageSummary = "已为大运验证与流年应期补充古籍依据。"

	plan, bundle, _, err := e.runBaziEvidenceStage(ctx, st, question, chartState.Input, dynamicPlan)
	if err != nil {
		return chartState, err
	}
	chartState.EvidencePlan = plan
	chartState.EvidenceBundle = mergeEvidenceBundles(chartState.EvidenceBundle, bundle)
	chartState.EvidenceQuality = evaluateEvidenceBundleQuality(chartState.EvidenceBundle)
	return chartState, nil
}

func emitBaziStageThinking(ctx context.Context, sink EventSink, agent, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = emitEventWithTrace(ctx, sink, Event{
		Type: "thinking",
		Data: map[string]any{
			"text":  text,
			"agent": agent,
		},
	}, map[string]any{
		"phase": "bazi_graph",
	})
}

// emitBaziReasoningSteps 只把产品化后的推演步骤发给前端，不暴露原始自由思维。
// 这些步骤来自上游结构化 synthesis 字段，属于可展示的“分析过程摘要”。
func emitBaziReasoningSteps(ctx context.Context, sink EventSink, label string, steps []string) {
	limit := 4
	count := 0
	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		emitBaziStageThinking(ctx, sink, "bazi_graph", fmt.Sprintf("%s：%s", label, step))
		count++
		if count >= limit {
			return
		}
	}
}

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

func buildStaticStageSummary(state baziCharterState) string {
	if len(state.StaticSynthesis.ReasoningSteps) > 1 {
		return fmt.Sprintf("静态综合已完成：%s。调候锚点为%s。先看%s，再看%s。关键限制是%s。", state.StaticSynthesis.MainAxis, state.StaticSynthesis.TiaohouAnchor, state.StaticSynthesis.ReasoningSteps[0], state.StaticSynthesis.ReasoningSteps[1], state.StaticSynthesis.CounterEvidence)
	}
	if len(state.StaticSynthesis.ReasoningSteps) > 0 {
		return fmt.Sprintf("静态综合已完成：%s。调候锚点为%s。当前已落实关键推演：%s。关键限制是%s。", state.StaticSynthesis.MainAxis, state.StaticSynthesis.TiaohouAnchor, state.StaticSynthesis.ReasoningSteps[0], state.StaticSynthesis.CounterEvidence)
	}
	return fmt.Sprintf("静态综合已完成，命局主轴收敛为：%s。调候锚点为%s。当前限制为：%s。", state.StaticSynthesis.MainAxis, state.StaticSynthesis.TiaohouAnchor, state.StaticSynthesis.CounterEvidence)
}

func buildDynamicStageSummary(state baziCharterState) string {
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

func buildAnalysisPlannerPayload(question string, chartFacts baziCharterInput) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart":          buildCoreChartView(chartFacts),
			"dynamic_facts_ready": hasDynamicSystemFacts(chartFacts),
		},
		"question": question,
	}
}

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

func buildStaticSynthesisPayload(state baziCharterState) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart": buildCoreChartView(state.Input),
		},
		"analysis_plan":    state.AnalysisPlan,
		"evidence_plan":    state.EvidencePlan,
		"evidence_bundle":  buildEvidenceBundleView(state.EvidenceBundle, true),
		"evidence_quality": state.EvidenceQuality,
		"question":         state.Input.UserQuestion,
	}
}

func buildDynamicSynthesisPayload(state baziCharterState) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart":    buildCoreChartView(state.Input),
			"dynamic_facts": buildDynamicFactsView(state.Input),
		},
		"analysis_plan":    state.AnalysisPlan,
		"evidence_bundle":  buildEvidenceBundleView(state.EvidenceBundle, true),
		"evidence_quality": state.EvidenceQuality,
		"static_synthesis": state.StaticSynthesis,
		"question":         state.Input.UserQuestion,
	}
}

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

func buildCoreChartView(input baziCharterInput) map[string]any {
	view := map[string]any{}
	if len(input.BaziResult) > 0 {
		if pillars, ok := input.BaziResult["pillars"]; ok && pillars != nil {
			view["pillars"] = pillars
			if monthPillar := extractMonthPillar(pillars); len(monthPillar) > 0 {
				view["month_pillar"] = monthPillar
			}
		}
		if dayGan, ok := input.BaziResult["dayGan"].(string); ok && dayGan != "" {
			view["day_master"] = dayGan
		}
		if dayWx, ok := input.BaziResult["dayGanWuxing"]; ok && dayWx != nil {
			view["day_master_wuxing"] = dayWx
		}
		if wx, ok := input.BaziResult["wuxing"]; ok && wx != nil {
			view["wuxing"] = wx
		}
	}
	if len(input.Yongshen) > 0 {
		for _, key := range []string{
			"day_master",
			"day_master_wuxing",
			"strength",
			"season",
			"tiao_hou",
			"balance_yong_shen",
			"tiaohou_yong_shen",
			"conditional_yong_shen",
			"yong_shen",
			"xi_shen",
			"ji_shen",
			"geju",
			"geju_status",
			"geju_detail",
			"geju_basis",
			"geju_qing_zhuo",
			"geju_qing_zhuo_reason",
			"geju_combination",
			"chonghe",
			"shi_shen_power",
		} {
			if value, ok := input.Yongshen[key]; ok && value != nil {
				view[key] = value
			}
		}
	}
	return view
}

func buildDynamicFactsView(input baziCharterInput) map[string]any {
	view := map[string]any{}
	if dayun := buildDayunFactsView(input.Dayun); len(dayun) > 0 {
		view["dayun"] = dayun
	}
	if liunian := buildLiunianFactsView(input.Liunian); len(liunian) > 0 {
		view["liunian"] = liunian
	}
	return view
}

func buildDayunFactsView(dayun map[string]any) map[string]any {
	if len(dayun) == 0 {
		return nil
	}
	view := map[string]any{}
	for _, key := range []string{"dayun_analyzed", "current_dayun", "periods"} {
		if value, ok := dayun[key]; ok && value != nil {
			view[key] = value
		}
	}
	if len(view) == 0 {
		return nil
	}
	return view
}

func buildLiunianFactsView(liunian map[string]any) map[string]any {
	if len(liunian) == 0 {
		return nil
	}
	view := map[string]any{}
	for _, key := range []string{
		"liunian_year",
		"liunian_ganzhi",
		"liunian_stem",
		"liunian_branch",
		"liunian_shi_shen",
		"current_dayun",
		"liunian_chonghe",
	} {
		if value, ok := liunian[key]; ok && value != nil {
			view[key] = value
		}
	}
	if len(view) == 0 {
		return nil
	}
	return view
}

func extractMonthPillar(raw any) map[string]any {
	switch pillars := raw.(type) {
	case []map[string]any:
		if len(pillars) > 1 {
			return copyAnyMap(pillars[1])
		}
	case []interface{}:
		if len(pillars) > 1 {
			if pillar, ok := pillars[1].(map[string]any); ok {
				return copyAnyMap(pillar)
			}
			if pillar, ok := pillars[1].(map[string]interface{}); ok {
				view := make(map[string]any, len(pillar))
				for k, v := range pillar {
					view[k] = v
				}
				return view
			}
		}
	}
	return nil
}

func copyAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (e *Executor) runBaziAnalysisPlanner(ctx context.Context, st *state.SessionState, question string, chartFacts baziCharterInput) (baziAnalysisPlan, error) {
	payload := buildAnalysisPlannerPayload(question, chartFacts)
	return runBaziInnerAgentJSON[baziAnalysisPlan](ctx, e.builder, baziAnalysisPlannerConfig(), st, buildBaziCharterPrompt("分析模式判定", question, payload))
}

func defaultBaziAnalysisPlan(question string) baziAnalysisPlan {
	return baziAnalysisPlan{
		Mode:           "static_full",
		RetrievalStage: "static",
		NeedDynamic:    true,
		FocusTopics:    []string{"命局主轴", "命格层次", "大运验证", "流年应期"},
		WriterTemplate: "full",
		TopicMode:      "analysis",
		StageSummary:   "已判定本轮以命局主轴分析为主。",
	}
}

func normalizeBaziAnalysisPlan(plan baziAnalysisPlan) baziAnalysisPlan {
	plan.Mode = strings.TrimSpace(plan.Mode)
	plan.RetrievalStage = strings.TrimSpace(plan.RetrievalStage)
	plan.WriterTemplate = strings.TrimSpace(plan.WriterTemplate)
	plan.TopicMode = normalizeByAlias(plan.TopicMode, map[string]string{
		"":                    "",
		"analysis":            "analysis",
		"general_analysis":    "analysis",
		"普通分析":                "analysis",
		"explain_term":        "explain_term",
		"term_explain":        "explain_term",
		"解释术语":                "explain_term",
		"conservative_reason": "conservative_reason",
		"保守原因":                "conservative_reason",
		"timing_reason":       "timing_reason",
		"岁运原因":                "timing_reason",
	})
	if plan.WriterTemplate == "topic" && plan.TopicMode == "" {
		plan.TopicMode = "analysis"
	}
	return plan
}

func (e *Executor) runBaziEvidenceStage(ctx context.Context, st *state.SessionState, question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) (baziEvidencePlan, baziEvidenceBundle, baziEvidenceQuality, error) {
	plan, err := e.runBaziEvidencePlanner(ctx, st, question, chartFacts, analysisPlan)
	if err != nil {
		plan = defaultBaziEvidencePlan(question, analysisPlan)
	}
	bundle, err := e.runControlledBaziRetrieval(ctx, plan)
	if err != nil {
		return plan, baziEvidenceBundle{}, baziEvidenceQuality{}, err
	}
	quality := evaluateEvidenceBundleQuality(bundle)
	return plan, bundle, quality, nil
}

func (e *Executor) runBaziEvidencePlanner(ctx context.Context, st *state.SessionState, question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) (baziEvidencePlan, error) {
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	payload := buildEvidencePlannerPayload(question, chartFacts, analysisPlan)
	return runBaziInnerAgentJSON[baziEvidencePlan](ctx, e.builder, baziEvidencePlannerConfig(), st, buildBaziCharterPrompt("证据规划", question, payload))
}

func defaultBaziEvidencePlan(question string, analysisPlan baziAnalysisPlan) baziEvidencePlan {
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	sources := stageAuthoritySources(stage)
	plan := baziEvidencePlan{
		NeedRetrieval:     true,
		Stage:             stage,
		RecommendedSource: append([]string{}, sources.Primary...),
		AllowReflection:   true,
	}
	if stage == "dynamic" {
		plan.EvidenceGaps = []string{"当前大运如何兑现静态主轴", "目标流年触发点补证"}
		plan.QueryPackets = []baziQueryPacket{
			{
				Topic:            "dayun",
				Query:            "三命通会 大运 行运 岁运并临",
				PreferredSources: []string{"三命通会", "滴天髓"},
				SourceTier:       "A",
			},
			{
				Topic:            "liunian",
				Query:            "三命通会 流年 应期 太岁",
				PreferredSources: []string{"三命通会", "子平真诠"},
				SourceTier:       "A",
			},
		}
		return plan
	}

	plan.EvidenceGaps = []string{"取格依据补证", "调候与格局交叉验证", "病药救应补证", "同结构命例校验补证"}
	plan.QueryPackets = []baziQueryPacket{
		{
			Topic:            "geju",
			Query:            "子平真诠 格局 月令 取格",
			PreferredSources: []string{"子平真诠", "渊海子平"},
			SourceTier:       "A",
		},
		{
			Topic:            "tiaohou",
			Query:            "穷通宝鉴 调候 月令 寒暖燥湿",
			PreferredSources: []string{"穷通宝鉴", "滴天髓"},
			SourceTier:       "A",
		},
		{
			Topic:            "bingyao",
			Query:            "滴天髓 病药 救应 制化",
			PreferredSources: []string{"滴天髓", "子平真诠"},
			SourceTier:       "A",
		},
		{
			Topic:            "geju",
			Query:            "子平真诠 格局 命例 举例",
			PreferredSources: []string{"子平真诠", "格局论命"},
			SourceTier:       "B",
		},
	}
	return plan
}

func (e *Executor) runControlledBaziRetrieval(ctx context.Context, plan baziEvidencePlan) (baziEvidenceBundle, error) {
	bundle := baziEvidenceBundle{
		Stage:        plan.Stage,
		TopicBuckets: map[string][]baziCitation{},
	}
	if !plan.NeedRetrieval || len(plan.QueryPackets) == 0 {
		return bundle, nil
	}
	if e == nil || e.builder == nil || e.builder.reg == nil {
		return bundle, fmt.Errorf("knowledge search registry unavailable")
	}

	searchTool, ok := e.builder.reg.Get("knowledge_search")
	if !ok {
		return bundle, fmt.Errorf("knowledge_search tool not registered")
	}

	type retrievalResult struct {
		index     int
		packet    baziQueryPacket
		citations []baziCitation
		err       error
	}
	results := make([]retrievalResult, len(plan.QueryPackets))
	var wg sync.WaitGroup
	for i, packet := range plan.QueryPackets {
		wg.Add(1)
		go func(index int, packet baziQueryPacket) {
			defer wg.Done()
			retrievalSpan := tracing.SpanFromContext(ctx, "knowledge_search", tracing.KindRetriever)
			retrievalSpan.SetAttribute("query", packet.Query)
			retrievalSpan.SetAttribute("topic", packet.Topic)
			retrievalSpan.SetAttribute("source_tier", packet.SourceTier)
			defer retrievalSpan.End()
			result, err := searchTool.Execute(ctx, map[string]any{
				"query": packet.Query,
				"top_k": float64(3),
			})
			if err != nil {
				retrievalSpan.RecordError(err)
				results[index] = retrievalResult{index: index, packet: packet, err: err}
				return
			}
			citations := citationsFromKnowledgeResult(result, packet)
			retrievalSpan.SetAttribute("hits", len(citations))
			results[index] = retrievalResult{
				index:     index,
				packet:    packet,
				citations: citations,
			}
		}(i, packet)
	}
	wg.Wait()
	for _, result := range results {
		if result.err != nil {
			return bundle, result.err
		}
		if len(result.citations) == 0 {
			continue
		}
		bundle.TopicBuckets[result.packet.Topic] = mergeCitations(bundle.TopicBuckets[result.packet.Topic], result.citations...)
		bundle.Citations = mergeCitations(bundle.Citations, result.citations...)
	}

	return bundle, nil
}

func hasDynamicSystemFacts(input baziCharterInput) bool {
	return len(input.Dayun) > 0 && len(input.Liunian) > 0
}

func citationsFromKnowledgeResult(result any, packet baziQueryPacket) []baziCitation {
	rm, ok := result.(map[string]any)
	if !ok {
		return nil
	}
	rawPassages, ok := rm["passages"]
	if !ok || rawPassages == nil {
		return nil
	}

	var out []baziCitation
	switch passages := rawPassages.(type) {
	case []mcp.Passage:
		for _, passage := range passages {
			out = mergeCitations(out, citationFromPassage(passage.Source, passage.Content, packet))
		}
	case []any:
		for _, raw := range passages {
			pm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			source, _ := pm["source"].(string)
			content, _ := pm["content"].(string)
			out = mergeCitations(out, citationFromPassage(source, content, packet))
		}
	}

	if preferred := filterPreferredCitations(out, packet.PreferredSources); len(preferred) > 0 {
		return preferred
	}
	return out
}

func filterPreferredCitations(items []baziCitation, preferredSources []string) []baziCitation {
	if len(items) == 0 || len(preferredSources) == 0 {
		return nil
	}
	var filtered []baziCitation
	for _, item := range items {
		if containsString(preferredSources, item.Classic) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func citationFromPassage(source, content string, packet baziQueryPacket) baziCitation {
	classic := extractAuthorityClassic(source)
	if classic == "" && len(packet.PreferredSources) > 0 {
		classic = packet.PreferredSources[0]
	}
	if classic == "" {
		classic = source
	}
	return baziCitation{
		Classic: classic,
		Quotes:  []string{strings.TrimSpace(content)},
	}
}

func extractAuthorityClassic(source string) string {
	for _, classic := range allAuthorityClassicNames() {
		if strings.Contains(source, classic) {
			return classic
		}
	}
	return ""
}

func allAuthorityClassicNames() []string {
	static := stageAuthoritySources("static")
	dynamic := stageAuthoritySources("dynamic")

	var names []string
	for _, bucket := range [][]string{static.Primary, static.Secondary, dynamic.Primary, dynamic.Secondary} {
		for _, name := range bucket {
			if !containsString(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
}

func mergeCitations(base []baziCitation, adds ...baziCitation) []baziCitation {
	merged := append([]baziCitation{}, base...)
	for _, add := range adds {
		if strings.TrimSpace(add.Classic) == "" {
			continue
		}
		found := false
		for i := range merged {
			if merged[i].Classic != add.Classic {
				continue
			}
			merged[i].Quotes = mergeStrings(merged[i].Quotes, add.Quotes...)
			found = true
			break
		}
		if !found {
			merged = append(merged, add)
		}
	}
	return merged
}

func mergeEvidenceBundles(base, add baziEvidenceBundle) baziEvidenceBundle {
	merged := baziEvidenceBundle{
		Stage:        base.Stage,
		TopicBuckets: map[string][]baziCitation{},
		Citations:    mergeCitations(base.Citations, add.Citations...),
		Conflicts:    mergeStrings(base.Conflicts, add.Conflicts...),
	}
	if strings.TrimSpace(merged.Stage) == "" {
		merged.Stage = add.Stage
	}
	for topic, citations := range base.TopicBuckets {
		merged.TopicBuckets[topic] = mergeCitations(merged.TopicBuckets[topic], citations...)
	}
	for topic, citations := range add.TopicBuckets {
		merged.TopicBuckets[topic] = mergeCitations(merged.TopicBuckets[topic], citations...)
	}
	return merged
}

func mergeStrings(base []string, adds ...string) []string {
	merged := append([]string{}, base...)
	for _, add := range adds {
		add = strings.TrimSpace(add)
		if add == "" || containsString(merged, add) {
			continue
		}
		merged = append(merged, add)
	}
	return merged
}

func (e *Executor) maybeReflectOnBaziEvidence(ctx context.Context, st *state.SessionState, chartState baziCharterState) (baziCharterState, error) {
	if !chartState.EvidencePlan.AllowReflection {
		return chartState, nil
	}
	if !shouldReflectOnEvidence(chartState.EvidenceQuality) {
		return chartState, nil
	}

	bundle, err := e.runControlledBaziRetrieval(ctx, chartState.EvidencePlan)
	if err != nil {
		return chartState, err
	}
	chartState.EvidenceBundle = bundle
	chartState.EvidenceQuality = evaluateEvidenceBundleQuality(bundle)
	return chartState, nil
}

func (e *Executor) runStaticSynthesis(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (baziStaticSynthesis, error) {
	payload := buildStaticSynthesisPayload(chartState)
	return runBaziInnerAgentJSON[baziStaticSynthesis](ctx, e.builder, baziStaticSynthesisConfig(), st, buildBaziCharterPrompt("静态综合", question, payload))
}

func (e *Executor) runDynamicSynthesis(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (baziDynamicSynthesis, error) {
	if err := validateDynamicPreconditions(chartState); err != nil {
		return baziDynamicSynthesis{}, err
	}
	payload := buildDynamicSynthesisPayload(chartState)
	return runBaziInnerAgentJSON[baziDynamicSynthesis](ctx, e.builder, baziDynamicSynthesisConfig(), st, buildBaziCharterPrompt("动态综合", question, payload))
}

func (e *Executor) runFinalWriter(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (string, error) {
	output := renderBaziFinalReply(chartState.AnalysisPlan, chartState, question)
	if err := validateFinalWriterOutput(chartState.AnalysisPlan, chartState, output); err != nil {
		if strings.Contains(err.Error(), "unsupported positive flourish claims") {
			sanitized := sanitizeUnsupportedFlourish(output)
			if sanitized != output {
				if retryErr := validateFinalWriterOutput(chartState.AnalysisPlan, chartState, sanitized); retryErr == nil {
					return sanitized, nil
				}
			}
		}
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.final_writer.template":       chartState.AnalysisPlan.WriterTemplate,
			"bazi.final_writer.validation_err": err.Error(),
			"bazi.final_writer.output_preview": truncateTracePreview(output, 1200),
		})
		return "", err
	}
	return output, nil
}

func truncateTracePreview(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "...(truncated)"
}

func buildBaziCharterPrompt(stage, question string, payload any) string {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		body = []byte("{}")
	}
	return strings.TrimSpace(fmt.Sprintf(
		"当前阶段：%s\n用户问题：%s\n\n请依据本阶段职责完成分析并输出。\n\n输入数据：\n%s",
		stage,
		question,
		string(body),
	))
}

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

func runBaziInnerAgentText(ctx context.Context, builder *AgentBuilder, cfg specialists.Config, st *state.SessionState, userPrompt string) (string, error) {
	agent, err := builder.BuildEphemeralInnerAgent(ctx, cfg, st)
	if err != nil {
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":  cfg.Name,
			"bazi.inner_agent.stage": "build",
			"bazi.inner_agent.error": err.Error(),
		})
		return "", err
	}
	if agent == nil {
		err := fmt.Errorf("inner agent %s is not configured", cfg.Name)
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":  cfg.Name,
			"bazi.inner_agent.stage": "build",
			"bazi.inner_agent.error": err.Error(),
		})
		return "", err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})
	raw, err := collectInnerAgentMessage(runner.Query(ctx, userPrompt))
	if err != nil {
		wrapped := fmt.Errorf("run inner agent %s: %w", cfg.Name, err)
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":  cfg.Name,
			"bazi.inner_agent.stage": "run",
			"bazi.inner_agent.error": wrapped.Error(),
		})
		return "", wrapped
	}
	return strings.TrimSpace(stripMarkdownFence(raw)), nil
}

// runBaziInnerAgentJSON 以结构化 JSON 模式运行内层 agent 并反序列化为目标类型。
// 当 Config.UseJSONMode=true 时，BuildSpecialist 会使用带 response_format: json_object 的模型，
// 确保 LLM 输出合法 JSON。stripMarkdownFence 仅作为安全网清理可能的 markdown 包裹。
func runBaziInnerAgentJSON[T any](ctx context.Context, builder *AgentBuilder, cfg specialists.Config, st *state.SessionState, userPrompt string) (T, error) {
	var out T
	raw, err := runBaziInnerAgentText(ctx, builder, cfg, st, userPrompt)
	if err != nil {
		return out, err
	}
	clean := stripMarkdownFence(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":           cfg.Name,
			"bazi.inner_agent.stage":          "parse_json",
			"bazi.inner_agent.error":          err.Error(),
			"bazi.inner_agent.output_preview": truncateTracePreview(clean, 1200),
		})
		return out, fmt.Errorf("parse inner agent %s output: %w", cfg.Name, err)
	}
	return out, nil
}

func collectInnerAgentMessage(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var last string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil || event.Output.MessageOutput.Message == nil {
			continue
		}
		last = event.Output.MessageOutput.Message.Content
	}
	if strings.TrimSpace(last) == "" {
		return "", fmt.Errorf("inner agent produced empty output")
	}
	return last, nil
}

func stripMarkdownFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}
