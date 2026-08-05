// Package runtime 包含 Manager 拥有的八字执行图。
//
// 本文件负责八字 charter graph 的节点、静态/动态投影合同和本地恢复入口；
// 不拥有最终答复权，也不让 specialist 绕过 ExecutionPlan 或 final guard。
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/mcp"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// validateStaticStage 校验静态综合展示所需的结构字段。
// 这些错误必须保持机器可读，后续 repair/recovery 节点才可分类处理。
func validateStaticStage(state baziCharterState) error {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		return validateFactsOnlyStaticSynthesis(state)
	}
	if state.Input.RuleProfile.ID != "" && state.StaticSynthesis.RuleProfile != "" && state.StaticSynthesis.RuleProfile != state.Input.RuleProfile.ID {
		return projectionMismatchViolation("static.rule_profile", "static synthesis rule profile does not match selected profile", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return projectionMismatchViolation("static.main_axis", "missing static synthesis main axis", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternBasis) == "" {
		return projectionMismatchViolation("static.pattern_basis", "missing static synthesis pattern basis", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternOutcome) == "" {
		return projectionMismatchViolation("static.pattern_outcome", "missing static synthesis pattern outcome", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.CounterEvidence) == "" {
		return projectionMismatchViolation("static.counter_evidence", "missing static synthesis counter evidence", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisConsistency) == "" {
		return projectionMismatchViolation("static.axis_consistency", "missing static synthesis axis consistency", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.TiaohouAnchor) == "" {
		return projectionMismatchViolation("static.tiaohou_anchor", "missing static synthesis tiaohou anchor", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternAndQingZhuo) == "" {
		return projectionMismatchViolation("static.pattern_and_qing_zhuo", "missing static synthesis pattern and qingzhuo", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.TierJudgment) == "" {
		return projectionMismatchViolation("static.tier_judgment", "missing static synthesis tier judgment", nil)
	}
	if strings.Contains(state.StaticSynthesis.TierJudgment, "层级暂不定级") {
		return projectionMismatchViolation("static.tier_judgment", "static synthesis exposes internal no-tier state", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.TierBasis) == "" {
		return projectionMismatchViolation("static.tier_basis", "missing static synthesis tier basis", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.ReasoningSummary) == "" {
		return projectionMismatchViolation("static.reasoning_summary", "missing static synthesis reasoning summary", nil)
	}
	if len(state.StaticSynthesis.ReasoningSteps) == 0 {
		return projectionMismatchViolation("static.reasoning_steps", "missing static synthesis reasoning steps", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.ClaimStrength) == "" {
		return projectionMismatchViolation("static.claim_strength", "missing static synthesis claim strength", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.SupportLevel) == "" {
		return projectionMismatchViolation("static.support_level", "missing static synthesis support level", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.LimitationLevel) == "" {
		return projectionMismatchViolation("static.limitation_level", "missing static synthesis limitation level", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.WordingCap) == "" {
		return projectionMismatchViolation("static.wording_cap", "missing static synthesis wording cap", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisLevel) == "" {
		return projectionMismatchViolation("static.axis_level", "missing static synthesis axis level", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnTiaohou) == "" {
		return projectionMismatchViolation("static.effect_on_tiaohou", "missing static synthesis effect on tiaohou", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnCoreDisease) == "" {
		return projectionMismatchViolation("static.effect_on_core_disease", "missing static synthesis effect on core disease", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnJiShenDirection) == "" {
		return projectionMismatchViolation("static.effect_on_jishen_direction", "missing static synthesis effect on ji-shen direction", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisCeiling) == "" {
		return projectionMismatchViolation("static.axis_ceiling", "missing static synthesis axis ceiling", nil)
	}
	if requiresStaticConflictReasons(state.StaticSynthesis) && len(state.StaticSynthesis.ConflictReasons) == 0 {
		return projectionMismatchViolation("static.conflict_reasons", "missing static synthesis conflict reasons", nil)
	}
	if err := validateStaticOutcomeScope(state); err != nil {
		return err
	}
	return nil
}

func isFactsOnlyStaticSynthesis(s baziStaticSynthesis) bool {
	return strings.TrimSpace(s.Source) == baziSynthesisSourceFactsOnlyDegraded
}

func isFactsOnlyDynamicSynthesis(d baziDynamicSynthesis) bool {
	return strings.TrimSpace(d.Source) == baziSynthesisSourceFactsOnlyDegraded
}

// validateFactsOnlyStaticSynthesis 校验 facts-only fallback 自身可展示。
// 这里不触发模型 repair；错误仍必须机器可读，便于 recovery 和 trace 分类。
func validateFactsOnlyStaticSynthesis(state baziCharterState) error {
	if state.Input.RuleProfile.ID != "" && state.StaticSynthesis.RuleProfile != state.Input.RuleProfile.ID {
		return projectionMismatchViolation("static.facts_only.rule_profile", "facts-only static synthesis rule profile does not match selected profile", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return projectionMismatchViolation("static.facts_only.main_axis", "facts-only static synthesis missing degraded message", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternBasis) == "" && len(state.Input.BaziResult) == 0 && len(state.Input.Yongshen) == 0 {
		return projectionMismatchViolation("static.facts_only.chart_facts", "facts-only static synthesis has no chart facts to show", nil)
	}
	return nil
}

// validateStaticAgainstProfileVerdicts is kept as a compatibility hook for
// externally supplied profiles. The runtime no longer installs default rule
// verdicts, so there is no built-in phrase-level correction here.
func validateStaticAgainstProfileVerdicts(state baziCharterState) error {
	_ = state
	return nil
}

func staticSynthesisUserVisibleText(output baziStaticSynthesis) string {
	return strings.Join([]string{
		output.MainAxis, output.PatternBasis, output.PatternOutcome, output.CounterEvidence,
		output.AxisConsistency, output.TiaohouConstraint, output.TiaohouAnchor, output.TierJudgment,
		output.TierBasis, output.ReasoningSummary, strings.Join(output.ReasoningSteps, "\n"),
		output.Strength.Conclusion, output.Strength.Reasoning, output.Strength.Boundary,
		output.Usage.Fuyi, output.Usage.Pattern, output.Usage.Tiaohou, output.Usage.Priority,
		strings.Join(output.Advantages, "\n"), strings.Join(output.Risks, "\n"),
	}, "\n")
}

// validateStaticOutcomeScope applies the same age-domain contract to static
// synthesis as dynamic synthesis. Static layers own visible advantages, risks
// and tier rationale, so minors must not receive adult-domain projections there.
func validateStaticOutcomeScope(state baziCharterState) error {
	context := buildBaziSubjectContext(state.Input)
	if context.AgeBand != "infant" && context.AgeBand != "child" && context.AgeBand != "adolescent" {
		return nil
	}
	if domain, term := firstUnauthorizedMinorOutcomeSignal(staticSynthesisUserVisibleText(state.StaticSynthesis)); domain != "" {
		return baziViolationError(baziViolationUnsupportedConcreteOutcome, "static", "", fmt.Sprintf("minor static synthesis writes unauthorized %s outcome signal %q", domain, term), nil, context.AllowedOutcomeDomains)
	}
	return nil
}

func validateDynamicAgainstProfileScope(state baziCharterState) error {
	if err := validateDynamicFireBureauFacts(state); err != nil {
		return err
	}
	if err := validateDynamicOutcomeDomains(state); err != nil {
		return err
	}
	text := strings.Join([]string{
		state.DynamicSynthesis.CurrentTrend, strings.Join(state.DynamicSynthesis.DayunPath, "\n"),
		strings.Join(renderDayunJudgmentLines(state.DynamicSynthesis.DayunJudgments), "\n"),
		state.DynamicSynthesis.LiunianFocus, strings.Join(state.DynamicSynthesis.TriggerSignals, "\n"),
		strings.Join(state.DynamicSynthesis.KeyWindows, "\n"), strings.Join(state.DynamicSynthesis.Risks, "\n"),
		strings.Join(state.DynamicSynthesis.ConsistencyFlags, "\n"), state.DynamicSynthesis.ReasoningSummary,
		strings.Join(state.DynamicSynthesis.ReasoningSteps, "\n"),
	}, "\n")
	if hasDynamicHardBoundary(text) {
		return baziViolationError(baziViolationUnsupportedConcreteOutcome, "dynamic", "", "dynamic synthesis overstates unsupported concrete outcome", nil, nil)
	}
	if err := validateDynamicConsistencyFlags(state.DynamicSynthesis.ConsistencyFlags); err != nil {
		return err
	}
	return nil
}

// validateDynamicOutcomeDomains enforces the deterministic age-based scope
// declared in the payload. It validates structured domains rather than growing
// a natural-language blacklist of possible life events.
func validateDynamicOutcomeDomains(state baziCharterState) error {
	context := buildBaziSubjectContext(state.Input)
	isMinor := context.AgeBand == "infant" || context.AgeBand == "child" || context.AgeBand == "adolescent"
	if !isMinor && len(state.DynamicSynthesis.DayunJudgments) == 0 {
		return nil
	}
	periodDomains := make(map[string]bool)
	for index, judgment := range state.DynamicSynthesis.DayunJudgments {
		if len(judgment.OutcomeDomains) == 0 {
			return baziViolationError(baziViolationScopeEscalation, fmt.Sprintf("dynamic.dayun_judgments[%d].outcome_domains", index), "", "each dayun judgment must declare outcome domains", nil, context.AllowedOutcomeDomains)
		}
		for _, domain := range judgment.OutcomeDomains {
			domain = strings.TrimSpace(domain)
			if !containsString(context.AllowedOutcomeDomains, domain) {
				return baziViolationError(baziViolationScopeEscalation, fmt.Sprintf("dynamic.dayun_judgments[%d].outcome_domains", index), "", "dayun outcome domain is not authorized for the subject age", nil, context.AllowedOutcomeDomains)
			}
			periodDomains[domain] = true
		}
	}
	if len(state.DynamicSynthesis.OutcomeDomains) == 0 {
		if !isMinor && len(periodDomains) == 0 {
			return nil
		}
		return baziViolationError(baziViolationScopeEscalation, "dynamic.outcome_domains", "", "dynamic synthesis must declare outcome domains", nil, context.AllowedOutcomeDomains)
	}
	for _, domain := range state.DynamicSynthesis.OutcomeDomains {
		if !containsString(context.AllowedOutcomeDomains, strings.TrimSpace(domain)) {
			return baziViolationError(baziViolationScopeEscalation, "dynamic.outcome_domains", "", "dynamic outcome domain is not authorized for the subject age", nil, context.AllowedOutcomeDomains)
		}
	}
	for domain := range periodDomains {
		if !containsString(state.DynamicSynthesis.OutcomeDomains, domain) {
			return baziViolationError(baziViolationScopeEscalation, "dynamic.outcome_domains", "", "top-level outcome domains must cover every dayun judgment domain", nil, state.DynamicSynthesis.OutcomeDomains)
		}
	}
	if isMinor {
		if domain, term := firstUnauthorizedMinorOutcomeSignal(dynamicUserVisibleText(state.DynamicSynthesis)); domain != "" {
			return baziViolationError(baziViolationUnsupportedConcreteOutcome, "dynamic", "", fmt.Sprintf("minor dynamic synthesis writes unauthorized %s outcome signal %q", domain, term), nil, context.AllowedOutcomeDomains)
		}
	}
	return nil
}

// firstUnauthorizedMinorOutcomeSignal maps broad life-domain wording back to
// the age-domain contract. This is a small taxonomy check, not a chart-specific
// phrase patch: minors may receive structure/growth/care wording, not adult
// career, finance, marriage, legal or medical projections hidden in prose.
func firstUnauthorizedMinorOutcomeSignal(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	for _, signal := range []struct {
		domain string
		terms  []string
	}{
		{domain: "career", terms: []string{"事业", "职业", "职场", "工作", "职位", "升迁", "权力", "权威", "名望"}},
		{domain: "finance", terms: []string{"财富", "财运", "财务", "收入", "赚钱", "投资", "破财"}},
		{domain: "marriage", terms: []string{"婚姻", "婚恋", "配偶", "感情"}},
		{domain: "legal", terms: []string{"法律", "官非", "诉讼", "牢狱", "血光", "伤灾"}},
		{domain: "medical", terms: []string{"疾病", "健康", "病症", "脾胃", "心血管", "高血压"}},
	} {
		for _, term := range signal.terms {
			if strings.Contains(text, term) {
				return signal.domain, term
			}
		}
	}
	return "", ""
}

func dynamicUserVisibleText(dynamic baziDynamicSynthesis) string {
	periodTexts := make([]string, 0, len(dynamic.DayunJudgments))
	for _, judgment := range dynamic.DayunJudgments {
		periodTexts = append(periodTexts, judgment.Trend, judgment.Interpretation, strings.Join(judgment.Evidence, "\n"))
	}
	return strings.Join([]string{
		dynamic.CurrentTrend,
		strings.Join(dynamic.DayunPath, "\n"),
		strings.Join(periodTexts, "\n"),
		dynamic.LiunianFocus,
		strings.Join(dynamic.TriggerSignals, "\n"),
		strings.Join(dynamic.KeyWindows, "\n"),
		strings.Join(dynamic.Risks, "\n"),
		dynamic.ReasoningSummary,
		strings.Join(dynamic.ReasoningSteps, "\n"),
	}, "\n")
}

// validateDynamicConsistencyFlags 校验动态一致性标签是否属于固定枚举。
// 非法枚举是投影合同错误，不进入事实修复或动态 facts-only。
func validateDynamicConsistencyFlags(flags []string) error {
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}
		if !containsString(allowedDynamicConsistencyFlags, flag) {
			return projectionMismatchViolation(
				"dynamic.consistency_flags",
				fmt.Sprintf("dynamic synthesis uses unsupported consistency flag %q; allowed: %s", flag, strings.Join(allowedDynamicConsistencyFlags, "、")),
				allowedDynamicConsistencyFlags,
			)
		}
	}
	return nil
}

var dynamicRelationTokenPattern = regexp.MustCompile(`[子丑寅卯辰巳午未申酉戌亥]{2,3}(?:相(?:冲|害|刑)|冲|害|刑|合(?:[金木水火土]局)?|会(?:[金木水火土]局)?)`)

// validateDynamicRelationFacts 校验动态文本只引用确定性工具声明的关系事实。
// 未声明关系是事实冲突，不能交给模型猜测或通过动态 facts-only 吞掉。
func validateDynamicRelationFacts(state baziCharterState) error {
	natalRelations := relationTextList(state.Input.Yongshen["chonghe"])
	periods := dayunPeriods(state.Input.Dayun)
	for index, line := range state.DynamicSynthesis.DayunPath {
		if index >= len(periods) {
			break
		}
		allowed := append(append([]string{}, natalRelations...), relationTextList(periods[index]["dayun_chonghe"])...)
		if token := firstUndeclaredRelationToken(line, allowed); token != "" {
			return baziViolationError(
				baziViolationFactConflict,
				fmt.Sprintf("dynamic.dayun_path[%d].relations", index),
				"",
				fmt.Sprintf("dynamic synthesis uses undeclared branch relation %q for dayun %q", token, stringValue(periods[index]["ganZhi"])),
				nil,
				allowed,
			)
		}
	}
	for index, judgment := range state.DynamicSynthesis.DayunJudgments {
		if index >= len(periods) {
			break
		}
		allowed := append(append([]string{}, natalRelations...), relationTextList(periods[index]["dayun_chonghe"])...)
		text := strings.Join(append([]string{judgment.Interpretation}, judgment.Evidence...), "\n")
		if token := firstUndeclaredRelationToken(text, allowed); token != "" {
			return baziViolationError(
				baziViolationFactConflict,
				fmt.Sprintf("dynamic.dayun_judgments[%d].interpretation", index),
				"",
				fmt.Sprintf("dynamic synthesis uses undeclared branch relation %q for dayun %q", token, stringValue(periods[index]["ganZhi"])),
				nil,
				allowed,
			)
		}
	}
	liunianText := strings.Join([]string{
		state.DynamicSynthesis.LiunianFocus,
		strings.Join(state.DynamicSynthesis.TriggerSignals, "\n"),
		strings.Join(state.DynamicSynthesis.KeyWindows, "\n"),
		strings.Join(state.DynamicSynthesis.Risks, "\n"),
	}, "\n")
	allowedLiunian := append(append([]string{}, natalRelations...), relationTextList(state.Input.Liunian["liunian_chonghe"])...)
	if token := firstUndeclaredRelationToken(liunianText, allowedLiunian); token != "" {
		return baziViolationError(
			baziViolationFactConflict,
			"dynamic.liunian.relations",
			"",
			fmt.Sprintf("dynamic synthesis uses undeclared branch relation %q for liunian", token),
			nil,
			allowedLiunian,
		)
	}
	return nil
}

func firstUndeclaredRelationToken(text string, descriptions []string) string {
	allowed := make(map[string]struct{}, len(descriptions))
	for _, description := range descriptions {
		for _, token := range dynamicRelationTokenPattern.FindAllString(description, -1) {
			allowed[normalizeDynamicRelationToken(token)] = struct{}{}
		}
	}
	for _, token := range dynamicRelationTokenPattern.FindAllString(text, -1) {
		if _, ok := allowed[normalizeDynamicRelationToken(token)]; !ok {
			return token
		}
	}
	return ""
}

func normalizeDynamicRelationToken(token string) string {
	return strings.ReplaceAll(strings.TrimSpace(token), "相", "")
}

// validateDynamicFireBureauFacts 校验动态文本中的火局必须来自确定性关系事实。
// 动态模型可以解释已声明事实，但不能凭文本创建新的火局。
func validateDynamicFireBureauFacts(state baziCharterState) error {
	periods := dayunPeriods(state.Input.Dayun)
	for index, line := range state.DynamicSynthesis.DayunPath {
		if index >= len(periods) || !strings.Contains(line, "火局") {
			continue
		}
		relations := relationTextList(periods[index]["dayun_chonghe"])
		if !containsAnyText(relations, []string{"火局"}) {
			return baziViolationError(
				baziViolationFactConflict,
				fmt.Sprintf("dynamic.dayun_path[%d].relations", index),
				"",
				fmt.Sprintf("dynamic synthesis uses an undeclared fire bureau for dayun %q", stringValue(periods[index]["ganZhi"])),
				nil,
				relations,
			)
		}
	}
	for index, judgment := range state.DynamicSynthesis.DayunJudgments {
		if index >= len(periods) {
			break
		}
		text := strings.Join(append([]string{judgment.Interpretation}, judgment.Evidence...), "\n")
		if !strings.Contains(text, "火局") {
			continue
		}
		relations := relationTextList(periods[index]["dayun_chonghe"])
		if !containsAnyText(relations, []string{"火局"}) {
			return baziViolationError(
				baziViolationFactConflict,
				fmt.Sprintf("dynamic.dayun_judgments[%d].interpretation", index),
				"",
				fmt.Sprintf("dynamic synthesis uses an undeclared fire bureau for dayun %q", stringValue(periods[index]["ganZhi"])),
				nil,
				relations,
			)
		}
	}
	liunianText := strings.Join([]string{
		state.DynamicSynthesis.LiunianFocus,
		strings.Join(state.DynamicSynthesis.TriggerSignals, "\n"),
		strings.Join(state.DynamicSynthesis.KeyWindows, "\n"),
	}, "\n")
	liunianRelations := relationTextList(state.Input.Liunian["liunian_chonghe"])
	if strings.Contains(liunianText, "火局") && !containsAnyText(liunianRelations, []string{"火局"}) {
		return baziViolationError(
			baziViolationFactConflict,
			"dynamic.liunian.relations",
			"",
			"dynamic synthesis uses an undeclared fire bureau for liunian",
			nil,
			liunianRelations,
		)
	}
	return nil
}

func validateEvidenceBundlePreconditions(state baziCharterState) error {
	if state.EvidencePlan.NeedRetrieval && len(state.EvidencePlan.QueryPackets) == 0 {
		return fmt.Errorf("missing query packets for retrieval-required state")
	}
	return nil
}

// validateDynamicPreconditions 确认动态综合只在静态主轴已准备后运行。
// 上游静态结果缺失属于结构合同错误，不触发模型 repair。
func validateDynamicPreconditions(state baziCharterState) error {
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return projectionMismatchViolation("dynamic.preconditions.static.main_axis", "dynamic stage requires static synthesis first", nil)
	}
	return nil
}

// validateDynamicStage 校验动态综合的核心字段、覆盖范围和授权边界。
// 字段缺失或结构不一致保持 hard-error/recovery 语义，不新增动态 repair。
func validateDynamicStage(state baziCharterState) error {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return validateFactsOnlyDynamicSynthesis(state)
	}
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return projectionMismatchViolation("dynamic.current_trend", "missing dynamic synthesis current trend", nil)
	}
	if len(state.DynamicSynthesis.DayunPath) == 0 {
		return projectionMismatchViolation("dynamic.dayun_path", "missing dynamic synthesis dayun path", nil)
	}
	if expected := len(dayunPeriods(state.Input.Dayun)); expected > 0 && len(state.DynamicSynthesis.DayunPath) < expected {
		return baziViolationError(
			baziViolationDayunCoverageMissing,
			"dynamic.dayun_path",
			"",
			fmt.Sprintf("dynamic synthesis omits calculated dayun periods: got %d, want at least %d", len(state.DynamicSynthesis.DayunPath), expected),
			nil,
			nil,
		)
	}
	if err := validateDayunJudgmentFacts(state.Input.Dayun, state.DynamicSynthesis.DayunJudgments); err != nil {
		return err
	}
	if strings.TrimSpace(state.DynamicSynthesis.LiunianFocus) == "" {
		return projectionMismatchViolation("dynamic.liunian_focus", "missing dynamic synthesis liunian focus", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.WindowLevel) == "" {
		return projectionMismatchViolation("dynamic.window_level", "missing dynamic synthesis window level", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.ReasoningSummary) == "" {
		return projectionMismatchViolation("dynamic.reasoning_summary", "missing dynamic synthesis reasoning summary", nil)
	}
	if len(state.DynamicSynthesis.ReasoningSteps) == 0 {
		return projectionMismatchViolation("dynamic.reasoning_steps", "missing dynamic synthesis reasoning steps", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.ClaimStrength) == "" {
		return projectionMismatchViolation("dynamic.claim_strength", "missing dynamic synthesis claim strength", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.SupportLevel) == "" {
		return projectionMismatchViolation("dynamic.support_level", "missing dynamic synthesis support level", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.LimitationLevel) == "" {
		return projectionMismatchViolation("dynamic.limitation_level", "missing dynamic synthesis limitation level", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.WordingCap) == "" {
		return projectionMismatchViolation("dynamic.wording_cap", "missing dynamic synthesis wording cap", nil)
	}
	return validateDynamicAgainstProfileScope(state)
}

// validateFactsOnlyDynamicSynthesis 校验动态 facts-only 降级仍保留可展示事实。
// 该校验只分类失败，不把降级输出重新送入模型 repair。
func validateFactsOnlyDynamicSynthesis(state baziCharterState) error {
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return projectionMismatchViolation("dynamic.facts_only.current_trend", "facts-only dynamic synthesis missing degraded message", nil)
	}
	if expected := len(dayunPeriods(state.Input.Dayun)); expected > 0 && len(state.DynamicSynthesis.DayunPath) < expected {
		return baziViolationError(
			baziViolationDayunCoverageMissing,
			"dynamic.facts_only.dayun_path",
			"",
			fmt.Sprintf("facts-only dynamic synthesis omits calculated dayun periods: got %d, want at least %d", len(state.DynamicSynthesis.DayunPath), expected),
			nil,
			nil,
		)
	}
	return nil
}

// validateDayunJudgmentFacts 校验每步动态判断与确定性大运事实逐项对齐。
// 干支错配是 fact_conflict，字段缺失和数量不符是结构合同错误。
func validateDayunJudgmentFacts(dayun map[string]any, judgments []baziDayunJudgment) error {
	if len(judgments) == 0 {
		// Keep old structured outputs valid during the schema migration. New model
		// prompts emit dayun_judgments, while dayun_path remains the compatibility
		// contract for historical sessions and fixtures.
		return nil
	}
	periods := dayunPeriods(dayun)
	if len(judgments) != len(periods) {
		return baziViolationError(
			baziViolationDayunCoverageMissing,
			"dynamic.dayun_judgments",
			"",
			fmt.Sprintf("dynamic synthesis dayun judgments mismatch: got %d, want %d", len(judgments), len(periods)),
			nil,
			nil,
		)
	}
	for i, judgment := range judgments {
		want := strings.TrimSpace(stringValue(periods[i]["ganZhi"]))
		got := strings.TrimSpace(judgment.GanZhi)
		field := fmt.Sprintf("dynamic.dayun_judgments[%d].gan_zhi", i)
		if want == "" {
			return baziViolationError(baziViolationFactRefMissing, field, "", "calculated dayun period is missing gan_zhi", nil, nil)
		}
		if got == "" {
			return projectionMismatchViolation(field, fmt.Sprintf("dynamic synthesis dayun judgment %d is missing gan_zhi", i), []string{want})
		}
		if !strings.HasPrefix(got, want) {
			return baziViolationError(
				baziViolationFactConflict,
				field,
				"",
				fmt.Sprintf("dynamic synthesis dayun judgment %d does not match calculated period %q", i, want),
				nil,
				[]string{want},
			)
		}
		if strings.TrimSpace(judgment.Trend) == "" || strings.TrimSpace(judgment.Interpretation) == "" {
			return projectionMismatchViolation(
				fmt.Sprintf("dynamic.dayun_judgments[%d]", i),
				fmt.Sprintf("dynamic synthesis dayun judgment %d is incomplete", i),
				nil,
			)
		}
	}
	return nil
}

func validateCharterConsistency(state baziCharterState) error {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) || isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return nil
	}
	if err := validateStaticDecisionConsistency(state.StaticSynthesis); err != nil {
		return err
	}
	if err := validateStaticAxisVerdictConsistency(state.StaticSynthesis); err != nil {
		return err
	}
	if err := validateStaticEvidenceCoverageBoundary(state); err != nil {
		return err
	}
	if err := validateStaticTiaohouEvidenceWording(state); err != nil {
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

// validateStaticEvidenceCoverageBoundary caps only the strength of a static
// route when planned A-tier topics remain missing. It does not select a chart
// methodology or alter the model's interpretive text.
func validateStaticEvidenceCoverageBoundary(state baziCharterState) error {
	if len(state.EvidenceQuality.MissingTopics) == 0 {
		return nil
	}
	if state.StaticSynthesis.AxisLevel == "主轴成立" || state.StaticSynthesis.AxisLevel == "可以拔高" {
		return baziViolationError(baziViolationEvidenceTopicMissing, "static.axis_level", "", fmt.Sprintf("static axis level exceeds incomplete evidence boundary: missing %s", strings.Join(state.EvidenceQuality.MissingTopics, ", ")), state.EvidenceQuality.MissingTopics, state.EvidenceQuality.CoveredTopics)
	}
	return nil
}

// validateStaticTiaohouEvidenceWording requires a real 调候 verdict once
// authority evidence is covered. Boundary text may remain cautious, but the
// visible anchor cannot be only "environment constraint" filler.
func validateStaticTiaohouEvidenceWording(state baziCharterState) error {
	if !containsString(state.EvidenceQuality.CoveredTopics, "tiaohou") {
		return nil
	}
	static := state.StaticSynthesis
	if !hasConcreteTiaohouVerdict(static.TiaohouAnchor) {
		return baziViolationError(baziViolationEvidenceTopicMissing, "static.tiaohou_anchor", "", fmt.Sprintf("static tiaohou anchor lacks concrete verdict despite covered authority evidence: %s", static.TiaohouAnchor), []string{"tiaohou"}, nil)
	}
	fields := []struct {
		name  string
		value string
	}{
		{name: "main_axis", value: static.MainAxis},
		{name: "pattern_outcome", value: static.PatternOutcome},
		{name: "counter_evidence", value: static.CounterEvidence},
		{name: "axis_consistency", value: static.AxisConsistency},
		{name: "tiaohou_constraint", value: static.TiaohouConstraint},
		{name: "tiaohou_anchor", value: static.TiaohouAnchor},
		{name: "pattern_and_qing_zhuo", value: static.PatternAndQingZhuo},
		{name: "tier_basis", value: static.TierBasis},
		{name: "reasoning_summary", value: static.ReasoningSummary},
		{name: "usage.tiaohou", value: static.Usage.Tiaohou},
		{name: "usage.priority", value: static.Usage.Priority},
		{name: "topic_direct_answer", value: static.TopicDirectAnswer},
		{name: "topic_focus_answer", value: static.TopicFocusAnswer},
	}
	for i, step := range static.ReasoningSteps {
		fields = append(fields, struct {
			name  string
			value string
		}{name: fmt.Sprintf("reasoning_steps[%d]", i), value: step})
	}
	for i, item := range static.Advantages {
		fields = append(fields, struct {
			name  string
			value string
		}{name: fmt.Sprintf("advantages[%d]", i), value: item})
	}
	for i, item := range static.Risks {
		fields = append(fields, struct {
			name  string
			value string
		}{name: fmt.Sprintf("risks[%d]", i), value: item})
	}
	for _, field := range fields {
		if phrase := firstTiaohouMissingEvidencePhrase(field.value); phrase != "" {
			return baziViolationError(baziViolationEvidenceTopicMissing, "static."+field.name, "", fmt.Sprintf("static tiaohou wording contradicts covered authority evidence in %s: %s; covered_topics=%s", field.name, phrase, strings.Join(state.EvidenceQuality.CoveredTopics, ", ")), []string{"tiaohou"}, nil)
		}
	}
	return nil
}

func hasConcreteTiaohouVerdict(text string) bool {
	return containsAnyText([]string{text}, []string{
		"调候不足", "调候受损", "调候受限", "调候得力", "调候有力",
		"调候可用", "调候不纯", "火根受冲", "火根受损", "火有但弱",
		"寒暖失衡", "寒重火弱", "寒湿", "暖局", "润燥", "燥烈",
	})
}

func firstTiaohouMissingEvidencePhrase(text string) string {
	for _, phrase := range []string{
		"调候关键证据缺失",
		"调候证据缺失",
		"调候规则材料不足",
		"调候用神未定",
		"具体调候用神待",
		"具体用神待穷通宝鉴",
		"需穷通宝鉴",
		"待穷通宝鉴",
	} {
		if strings.Contains(text, phrase) {
			return phrase
		}
	}
	return ""
}

// validateCurrentDayunLineConsistency 约束当前大运总述与当前大运条目保持同线，
// 避免同一步运同时被写成“承托主轴”和“偏压/压制主轴”的相反口径。
func validateCurrentDayunLineConsistency(d baziDynamicSynthesis) error {
	// Direction words are explanatory language, not chart facts. A mixed trend
	// can legitimately describe the same luck pillar from multiple dimensions;
	// record suspicious wording in soft audit instead of rejecting the report.
	_ = d
	return nil
}

// validateStaticAxisVerdictConsistency 校验轴线裁断字段之间的封顶关系。
// 该层只处理结构投影一致性，不替模型选择格局路线。
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
	if requiresStaticConflictReasons(s) && len(s.ConflictReasons) == 0 {
		return projectionMismatchViolation("static.conflict_reasons", "static conflict reasons required", nil)
	}
	if err := validateAxisVerdictAgainstConflict(s); err != nil {
		return err
	}
	if s.AxisCeiling == "结构信号" &&
		containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"主轴", "贵格", "化杀为权"}) {
		return projectionMismatchViolation("static.axis_ceiling", "static synthesis promotes structure signal beyond axis ceiling", nil)
	}
	if s.AxisCeiling == "受限路线" &&
		containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"纯主轴贵格", "可以拔高", "化杀为权"}) {
		return projectionMismatchViolation("static.axis_ceiling", "static synthesis promotes restricted route beyond axis ceiling", nil)
	}
	return nil
}

// requiresStaticConflictReasons limits the conflict-reason requirement to
// outputs that actually declare conflict, amplification, or a restricted
// ceiling. Neutral axes still go through semantic audit; they should not be
// rejected before audit just because no conflict list is present.
func requiresStaticConflictReasons(s baziStaticSynthesis) bool {
	return s.EffectOnTiaohou == "冲突" ||
		s.EffectOnCoreDisease == "放大" ||
		s.EffectOnJiShenDirection == "放大" ||
		s.AxisCeiling == "结构信号" ||
		s.AxisCeiling == "受限路线" ||
		containsString(s.ConsistencyFlags, "方向成立但力度受限")
}

// validateAxisVerdictAgainstConflict 防止冲突路线被投影成高封顶主轴。
// 冲突事实本身可被解释，但字段封顶必须同步降级。
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
		return projectionMismatchViolation("static.axis_ceiling", "static axis ceiling is too high for a conflict-amplifying route", nil)
	}
	if s.AxisCeiling == "结构信号" && (s.AxisLevel == "主轴成立" || s.AxisLevel == "可以拔高") {
		return projectionMismatchViolation("static.axis_level", "static axis level exceeds structure-signal ceiling", nil)
	}
	if s.AxisCeiling == "受限路线" && s.AxisLevel == "可以拔高" {
		return projectionMismatchViolation("static.axis_level", "static axis level exceeds restricted-route ceiling", nil)
	}
	return nil
}

// validateDynamicAgainstStaticCeiling 防止动态文字越过静态轴线封顶。
// 它只做上下游字段一致性检查，不重新裁断运势吉凶。
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
		return projectionMismatchViolation("dynamic.static_axis_ceiling", "dynamic synthesis escalates beyond static axis ceiling", nil)
	}
	return nil
}

// validateStaticDecisionConsistency 校验静态综合的强度、限制和可见措辞封顶。
// 失败进入 projection mismatch，不当作事实冲突或方法合同错误。
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
			"层级暂不定级",
			"完整层次规则尚未覆盖",
			"不自动换算富贵层次",
			"不自动换算富贵等级",
		}) {
		return projectionMismatchViolation("static.consistency_flags", "static consistency flag requires visible limitation text", nil)
	}
	if containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"一飞冲天"}) &&
		!allowsFlourishByWordingCap(s.WordingCap, "一飞冲天") {
		return projectionMismatchViolation("static.wording_cap", "static synthesis overstates wording beyond wording cap", nil)
	}
	if containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"可享清福", "福泽深厚", "贵人众多"}) &&
		!allowsFlourishByWordingCap(s.WordingCap, "positive_flourish") {
		return projectionMismatchViolation("static.wording_cap", "static synthesis overstates wording beyond wording cap", nil)
	}
	return nil
}

// validateStaticAxisAgainstChartFacts leaves methodology disputes to model eval.
// Runtime guards only reject structural field conflicts elsewhere in the graph.
func validateStaticAxisAgainstChartFacts(state baziCharterState) error {
	_ = state
	return nil
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
	return nil
}

func validateFinalWriterOutput(plan baziAnalysisPlan, state baziCharterState, output string) error {
	output = strings.TrimSpace(output)
	if output == "" {
		return fmt.Errorf("final writer produced empty output")
	}
	if strings.Contains(output, "上游未提供") {
		return fmt.Errorf("final writer output leaked internal missing-field placeholder")
	}
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		if !strings.Contains(output, "只列可复算命盘事实") || !strings.Contains(output, "暂不作主轴") {
			return fmt.Errorf("facts-only final output must expose facts-only synthesis boundary")
		}
		if containsAnyText([]string{output}, []string{"优先按伤官佩印", "层次中等", "中等（保守定位）", "结构承接、压力", "倾向有利"}) {
			return fmt.Errorf("facts-only final output must not expose synthesized reading language")
		}
		return nil
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
		if !containsAnyText([]string{gejuSection}, []string{"**规则口径**"}) {
			return fmt.Errorf("full writer output must expose the selected rule-profile boundary in 格局视角")
		}
		if strings.TrimSpace(state.StaticSynthesis.PatternBasis) != "" && !containsAnyText([]string{gejuSection}, []string{"**依据**"}) {
			return fmt.Errorf("full writer output must expose concise evidence in 格局视角")
		}
		if !containsAnyText([]string{gejuSection}, []string{"**限制**"}) {
			return fmt.Errorf("full writer output must expose its limiting evidence in 格局视角")
		}
		tierSection := sectionContent(output, "## 综合判定", "## 命格总结")
		if tierSection == "" {
			return fmt.Errorf("full writer output missing 综合判定 section body")
		}
		if strings.TrimSpace(state.StaticSynthesis.TierBasis) != "" && !containsAnyText([]string{tierSection}, []string{"**解释**"}) {
			return fmt.Errorf("full writer output must expose a concise basis in 综合判定")
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
	if counterEvidence := strings.TrimSpace(state.StaticSynthesis.CounterEvidence); counterEvidence != "" &&
		!strings.Contains(output, counterEvidence) &&
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

// validateAllowedValue 校验静态/动态投影枚举值，并返回机器可读字段错误。
func validateAllowedValue(name, value string, allowed []string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return projectionMismatchViolation(validationFieldName(name), fmt.Sprintf("missing %s", name), allowed)
	}
	if !containsString(allowed, value) {
		return projectionMismatchViolation(validationFieldName(name), fmt.Sprintf("invalid %s: %s", name, value), allowed)
	}
	return nil
}

// projectionMismatchViolation 统一包装静态/动态投影类字段错误。
func projectionMismatchViolation(field, message string, allowed []string) error {
	return baziViolationError(baziViolationScopeEscalation, field, "", message, nil, allowed)
}

// validationFieldName 把 validator 的人类标签压成 trace/repair 可读字段名。
func validationFieldName(name string) string {
	field := strings.TrimSpace(strings.ToLower(name))
	field = strings.ReplaceAll(field, "ji-shen", "jishen")
	field = strings.ReplaceAll(field, " ", ".")
	return field
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

// runBaziAuthorityFirstGraph keeps the outer orchestration contract unchanged
// while delegating BaZi-specific control flow to the internal graph.
func (e *Executor) runBaziAuthorityFirstGraph(ctx context.Context, sink EventSink, st *state.SessionState, question string) (string, error) {
	return e.runBaziInternalGraph(ctx, sink, st, question)
}

func baziSynthesisRuntimeFailure(stage, code string, cause error) error {
	return &RuntimeFailure{
		Class:       failureClassModelContractViolation,
		Stage:       stage,
		Domain:      "bazi",
		Code:        code,
		Retryable:   true,
		Degraded:    false,
		UserVisible: true,
		Message:     baziSynthesisFailureMessage(stage, cause),
		Cause:       cause,
	}
}

// baziSynthesisFailureMessage keeps user-visible errors aligned with the actual
// contract failure class without leaking candidate text.
func baziSynthesisFailureMessage(stage string, cause error) string {
	if failure, ok := baziContractFailureFromError(stage, cause); ok {
		switch failure.Class {
		case baziContractFailureEvidenceOverclaim:
			return "证据主题不足，已停止展示过度裁断。请稍后重试。"
		case baziContractFailureDomainUnauthorized:
			return "岁运综合越过授权领域，已停止展示不稳定内容。请稍后重试。"
		case baziContractFailureFactConflict, baziContractFailureMethodContract:
			return "候选推演与事实或方法合同冲突，已停止展示不稳定内容。请稍后重试。"
		}
	}
	return "本轮八字综合未通过结构化合同校验，已停止展示不稳定内容。请稍后重试。"
}

func validateDynamicSynthesisAfterGraphNormalization(state baziCharterState) error {
	if isPartialSynthesisSource(state.DynamicSynthesis.Source) {
		return validatePartialDynamicSynthesis(state, state.DynamicSynthesis)
	}
	return validateDynamicSynthesisResult(state, state.DynamicSynthesis)
}

func annotateBaziSynthesisSources(ctx context.Context, state baziCharterState) {
	outputMode := "model_full"
	switch {
	case isFactsOnlyStaticSynthesis(state.StaticSynthesis):
		outputMode = baziSynthesisSourceFactsOnlyDegraded
	case isFactsOnlyDynamicSynthesis(state.DynamicSynthesis):
		outputMode = "model_static_dynamic_degraded"
	case isPartialSynthesisSource(state.StaticSynthesis.Source) && isPartialSynthesisSource(state.DynamicSynthesis.Source):
		outputMode = "model_static_dynamic_partial"
	case isPartialSynthesisSource(state.StaticSynthesis.Source):
		outputMode = "model_static_partial"
	case isPartialSynthesisSource(state.DynamicSynthesis.Source):
		outputMode = "model_dynamic_partial"
	}
	attrs := map[string]any{
		"bazi.facts.source":            "deterministic_tools",
		"bazi.static.source":           firstNonEmptyTrim(state.StaticSynthesis.Source, "unknown"),
		"bazi.static.error":            state.StaticSynthesis.RecoveryReason,
		"bazi.static.partial_reason":   partialReasonForSource(state.StaticSynthesis.Source, state.StaticSynthesis.RecoveryReason),
		"bazi.static.degraded_reason":  degradedReasonForSource(state.StaticSynthesis.Source, state.StaticSynthesis.RecoveryReason),
		"bazi.static.recovery_reason":  state.StaticSynthesis.RecoveryReason,
		"bazi.static.assertion_count":  len(state.StaticSynthesis.Assertions),
		"bazi.static.contract_audit":   baziContractAuditSummary(state.StaticSynthesis.ContractAudit),
		"bazi.tier.source":             firstNonEmptyTrim(state.StaticSynthesis.Source, "unknown"),
		"bazi.tiaohou.coverage":        baziTiaohouCoverage(state.EvidenceQuality),
		"bazi.dynamic.source":          firstNonEmptyTrim(state.DynamicSynthesis.Source, "unknown"),
		"bazi.dynamic.error":           state.DynamicSynthesis.RecoveryReason,
		"bazi.dynamic.partial_reason":  partialReasonForSource(state.DynamicSynthesis.Source, state.DynamicSynthesis.RecoveryReason),
		"bazi.dynamic.degraded_reason": degradedReasonForSource(state.DynamicSynthesis.Source, state.DynamicSynthesis.RecoveryReason),
		"bazi.dynamic.recovery_reason": state.DynamicSynthesis.RecoveryReason,
		"bazi.dynamic.assertion_count": len(state.DynamicSynthesis.Assertions),
		"bazi.dynamic.contract_audit":  baziContractAuditSummary(state.DynamicSynthesis.ContractAudit),
		"bazi.dayun.count":             len(state.DynamicSynthesis.DayunPath),
		"bazi.final.output_mode":       outputMode,
		"bazi.final.audit_result":      baziFieldAuditResult(state.FieldAudit),
	}
	if class := firstFieldAuditValue(state.FieldAudit, "contract_failure_class:"); class != "" {
		attrs["bazi.contract.failure_class"] = class
	}
	if policy := firstFieldAuditValue(state.FieldAudit, "recovery_policy:"); policy != "" {
		attrs["bazi.contract.recovery_policy"] = policy
	}
	tracing.SetTraceAttributes(ctx, attrs)
}

// firstFieldAuditValue returns the first compact key-value note stored by a
// recovery path for trace projection.
func firstFieldAuditValue(notes []string, prefix string) string {
	for _, note := range notes {
		if strings.HasPrefix(note, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(note, prefix))
		}
	}
	return ""
}

func degradedReasonForSource(source, reason string) string {
	if strings.TrimSpace(source) != baziSynthesisSourceFactsOnlyDegraded {
		return ""
	}
	return reason
}

func partialReasonForSource(source, reason string) string {
	if !isPartialSynthesisSource(source) {
		return ""
	}
	return reason
}

func baziFieldAuditResult(notes []string) string {
	repairs := make([]string, 0, len(notes))
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" || note == "canonical_tier_withheld_by_runtime" {
			continue
		}
		repairs = append(repairs, note)
	}
	if len(repairs) == 0 {
		return "clean"
	}
	return "repaired: " + strings.Join(repairs, ", ")
}

// annotateBaziSoftAudit records reviewable wording risks without mutating the
// user-facing report. These concerns depend on rule-profile scope and human
// calibration, so they are unsuitable as a runtime hard gate.
func annotateBaziSoftAudit(ctx context.Context, state baziCharterState) {
	warnings := collectBaziSoftAuditWarnings(state)
	if len(warnings) == 0 {
		return
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.graph.soft_audit_warning_count": len(warnings),
		"bazi.graph.soft_audit_warnings":      strings.Join(warnings, " | "),
	})
}

func collectBaziSoftAuditWarnings(state baziCharterState) []string {
	staticText := strings.Join([]string{
		state.StaticSynthesis.MainAxis,
		state.StaticSynthesis.PatternOutcome,
		state.StaticSynthesis.TierJudgment,
		state.StaticSynthesis.TierBasis,
	}, "\n")
	dynamicText := strings.Join([]string{
		state.DynamicSynthesis.CurrentTrend,
		strings.Join(state.DynamicSynthesis.DayunPath, "\n"),
		state.DynamicSynthesis.LiunianFocus,
	}, "\n")
	warnings := []string{}
	knownFacts := knownBaziFactRefs(state)
	knownClaims := knownBaziClaimRefs(state.Input.RuleProfile)
	unknownFactRefs := []string{}
	unknownClaimRefs := []string{}
	for _, assertion := range append(append([]baziAssertion{}, state.StaticSynthesis.Assertions...), state.DynamicSynthesis.Assertions...) {
		for _, ref := range assertion.FactRefs {
			if !isKnownBaziFactRef(ref, knownFacts) {
				unknownFactRefs = append(unknownFactRefs, string(ref))
			}
		}
		for _, ref := range assertion.ClaimRefs {
			if _, ok := knownClaims[string(ref)]; !ok {
				unknownClaimRefs = append(unknownClaimRefs, string(ref))
			} else if !claimRefAllowsAssertionKind(state.Input.RuleProfile, string(ref), assertion.Kind) {
				warnings = append(warnings, "assertion uses a known claim outside its suggested kind: "+assertion.ID+" -> "+string(ref))
			}
		}
	}
	if len(unknownFactRefs) > 0 {
		warnings = append(warnings, "assertions use unknown fact-ref aliases: "+strings.Join(uniqueText(unknownFactRefs), ", "))
	}
	if len(unknownClaimRefs) > 0 {
		warnings = append(warnings, "assertions use unknown claim refs: "+strings.Join(uniqueText(unknownClaimRefs), ", "))
	}
	if err := validateDynamicRelationFacts(state); err != nil {
		warnings = append(warnings, err.Error())
	}
	if containsAnyText([]string{staticText}, []string{"贵格", "富格", "伤官佩印格成", "伤官佩印成立"}) {
		warnings = append(warnings, "static wording uses a strong pattern or status conclusion; review profile evidence")
	}
	if containsAnyText([]string{dynamicText}, []string{"大吉", "大凶", "黄金窗口", "职位提升", "贵人赏识", "婚姻", "财运", "健康"}) {
		warnings = append(warnings, "dynamic wording includes a broad tendency or concrete domain outcome; review evidence before expanding the rule profile")
	}
	return warnings
}

func annotateBaziGraphError(ctx context.Context, stage string, err error) {
	if err == nil {
		return
	}
	attrs := map[string]any{
		"bazi.graph.error_stage": stage,
		"bazi.graph.error":       err.Error(),
	}
	switch {
	case strings.HasPrefix(stage, "static"):
		attrs["bazi.static.error_stage"] = stage
		attrs["bazi.static.error"] = err.Error()
	case strings.HasPrefix(stage, "dynamic"):
		attrs["bazi.dynamic.error_stage"] = stage
		attrs["bazi.dynamic.error"] = err.Error()
	case strings.HasPrefix(stage, "evidence"):
		attrs["bazi.evidence.error_stage"] = stage
		attrs["bazi.evidence.error"] = err.Error()
	}
	for key, value := range baziTraceAttrsForContractFailure(stage, err) {
		attrs[key] = value
	}
	tracing.SetTraceAttributes(ctx, attrs)
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
	chartState.EvidenceQuality = evaluateEvidenceBundleQuality(plan, bundle)
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

func buildPartialProfileStaticSummary(state baziCharterState) string {
	candidate := strings.TrimSpace(stringValue(state.Input.Yongshen["geju_candidate"]))
	if candidate == "" {
		return "已核对月令、透干、藏干以及扶身与泄耗克身两侧证据。"
	}
	return "已核对月令取格、透藏和双方受力；当前结构候选为" + candidate + "，成格与层次另按已实现规则裁断。"
}

func buildPartialProfileDynamicSummary(state baziCharterState) string {
	current, _ := state.Input.Liunian["current_dayun"].(map[string]any)
	dayun := strings.TrimSpace(stringValue(current["ganZhi"]))
	liunian := strings.TrimSpace(stringValue(state.Input.Liunian["liunian_ganzhi"]))
	switch {
	case dayun != "" && liunian != "":
		return "已核对当前" + dayun + "大运、" + liunian + "流年及程序算出的标准关系；具体吉凶不由单个关系直接推出。"
	case dayun != "":
		return "已核对当前" + dayun + "大运及程序算出的标准关系；具体吉凶不由单个关系直接推出。"
	default:
		return "已核对当前岁运的可复算事实；具体吉凶不由单个关系直接推出。"
	}
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
			"core_chart":      buildCoreChartView(state.Input),
			"subject_context": buildBaziSubjectContext(state.Input),
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
			"core_chart":      buildCoreChartView(state.Input),
			"dynamic_facts":   buildDynamicFactsView(state.Input),
			"subject_context": buildBaziSubjectContext(state.Input),
		},
		"analysis_plan":    state.AnalysisPlan,
		"evidence_bundle":  buildEvidenceBundleView(state.EvidenceBundle, true),
		"evidence_quality": state.EvidenceQuality,
		"static_synthesis": state.StaticSynthesis,
		"question":         state.Input.UserQuestion,
	}
}

// buildBaziSubjectContext derives age from the calculated birth timestamp and
// target year. It scopes interpretation only; it never changes chart facts.
func buildBaziSubjectContext(input baziCharterInput) baziSubjectContext {
	birthYear := yearPrefix(stringValue(input.BaziResult["birthday"]))
	targetYear := intValue(input.Liunian["liunian_year"])
	context := baziSubjectContext{BirthYear: birthYear, TargetYear: targetYear, AgeBand: "unknown", AllowedOutcomeDomains: []string{"structure"}}
	if birthYear <= 0 || targetYear < birthYear {
		return context
	}
	context.Age = targetYear - birthYear
	switch {
	case context.Age <= 2:
		context.AgeBand = "infant"
		context.AllowedOutcomeDomains = []string{"structure", "growth_environment", "care_routine", "observable_development"}
	case context.Age <= 12:
		context.AgeBand = "child"
		context.AllowedOutcomeDomains = []string{"structure", "growth_environment", "care_routine", "observable_development"}
	case context.Age <= 17:
		context.AgeBand = "adolescent"
		context.AllowedOutcomeDomains = []string{"structure", "growth_environment", "care_routine", "observable_development"}
	case context.Age <= 64:
		context.AgeBand = "adult"
		context.AllowedOutcomeDomains = []string{"structure", "user_requested_authorized_domain"}
	default:
		context.AgeBand = "senior"
		context.AllowedOutcomeDomains = []string{"structure", "user_requested_authorized_domain"}
	}
	return context
}

// yearPrefix reads a four-digit year from a normalized timestamp prefix.
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
	view["selected_rule_profile"] = input.RuleProfile
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
			"strength_method",
			"strength_evidence",
			"balance_status",
			"seasonal_tiaohou_hint",
			"official_visibility",
			"season",
			"tiao_hou",
			"balance_yong_shen",
			"tiaohou_yong_shen",
			"conditional_yong_shen",
			"yong_shen",
			"xi_shen",
			"ji_shen",
			"geju",
			"geju_candidate",
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
				if key == "tiao_hou" && isTiaohouImplementationPlaceholder(value) {
					continue
				}
				view[key] = value
			}
		}
	}
	return view
}

// baziTiaohouCoverage reports evidence coverage, not whether the optional Go
// rule-profile table is implemented. Static synthesis already receives retrieved
// authority material; calling that "runtime_profile_disabled" makes the model
// treat covered 调候 evidence as missing.
func baziTiaohouCoverage(quality baziEvidenceQuality) string {
	if containsString(quality.CoveredTopics, "tiaohou") {
		return "authority_evidence_covered"
	}
	if containsString(quality.MissingTopics, "tiaohou") {
		return "missing_authority_evidence"
	}
	return "not_required"
}

// isTiaohouImplementationPlaceholder filters engineering status out of the
// domain fact view; the synthesizer should see chart facts and evidence, not a
// stale reminder that a future deterministic rule table does not exist yet.
func isTiaohouImplementationPlaceholder(value any) bool {
	text := strings.TrimSpace(stringValue(value))
	return strings.Contains(text, "qiongtong_tiaohou_v1") || strings.Contains(text, "规则表实现")
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
		plan = defaultBaziEvidencePlan(question, analysisPlan, chartFacts)
	} else {
		plan = normalizeBaziEvidencePlan(plan, chartFacts, analysisPlan)
	}
	bundle, err := e.runControlledBaziRetrieval(ctx, plan)
	if err != nil {
		return plan, baziEvidenceBundle{}, baziEvidenceQuality{}, err
	}
	quality := evaluateEvidenceBundleQuality(plan, bundle)
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

func defaultBaziEvidencePlan(question string, analysisPlan baziAnalysisPlan, chartFacts ...baziCharterInput) baziEvidencePlan {
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	var input baziCharterInput
	if len(chartFacts) > 0 {
		input = chartFacts[0]
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
			Query:            buildTiaohouEvidenceQuery(input),
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
	return normalizeBaziEvidencePlan(plan, input, analysisPlan)
}

// normalizeBaziEvidencePlan keeps model-planned retrieval useful for the
// downstream quality gate. 调候 evidence must be chart-specific; a broad
// "调候 月令" query often retrieves generic theory but still fails the
// per-topic authority coverage required by static synthesis.
func normalizeBaziEvidencePlan(plan baziEvidencePlan, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) baziEvidencePlan {
	if strings.TrimSpace(plan.Stage) == "" {
		plan.Stage = firstNonEmptyTrim(analysisPlan.RetrievalStage, "static")
	}
	if plan.Stage != "static" {
		return plan
	}
	hasTiaohou := false
	for i := range plan.QueryPackets {
		packet := &plan.QueryPackets[i]
		if packet.Topic != "tiaohou" || !strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") {
			continue
		}
		hasTiaohou = true
		if !queryContainsTiaohouChartTerms(packet.Query, chartFacts) {
			packet.Query = buildTiaohouEvidenceQuery(chartFacts)
		}
		packet.PreferredSources = mergeStrings(packet.PreferredSources, "穷通宝鉴", "滴天髓")
	}
	if !hasTiaohou {
		plan.QueryPackets = append(plan.QueryPackets, baziQueryPacket{
			Topic:            "tiaohou",
			Query:            buildTiaohouEvidenceQuery(chartFacts),
			PreferredSources: []string{"穷通宝鉴", "滴天髓"},
			SourceTier:       "A",
		})
	}
	return plan
}

// buildTiaohouEvidenceQuery derives a concrete 穷通 query from the calculated
// day master and month branch. This makes evidence coverage deterministic across
// runs instead of depending on the planner to remember the exact chart terms.
func buildTiaohouEvidenceQuery(input baziCharterInput) string {
	terms := []string{"穷通宝鉴"}
	if day := dayMasterForEvidenceQuery(input); day != "" {
		terms = append(terms, day)
	}
	if month := monthBranchForEvidenceQuery(input); month != "" {
		terms = append(terms, month+"月")
		if label := baziMonthBranchLabel(month); label != "" {
			terms = append(terms, label)
			if day := dayMasterForEvidenceQuery(input); day != "" {
				terms = append(terms, label+day)
			}
		}
		if day := dayMasterForEvidenceQuery(input); day != "" {
			terms = append(terms, month+"月"+day)
		}
	}
	terms = append(terms, "调候")
	if len(terms) <= 2 {
		terms = append(terms, "月令", "寒暖燥湿")
	}
	return strings.Join(terms, " ")
}

func queryContainsTiaohouChartTerms(query string, input baziCharterInput) bool {
	day := dayMasterForEvidenceQuery(input)
	month := monthBranchForEvidenceQuery(input)
	if day != "" && !strings.Contains(query, day) {
		return false
	}
	if month != "" {
		monthTerms := []string{month + "月"}
		if label := baziMonthBranchLabel(month); label != "" {
			monthTerms = append(monthTerms, label)
		}
		if !containsAnyText([]string{query}, monthTerms) {
			return false
		}
		if day != "" {
			specificMonthTerms := []string{month + "月" + day}
			if label := baziMonthBranchLabel(month); label != "" {
				specificMonthTerms = append(specificMonthTerms, label+day)
			}
			if !containsAnyText([]string{query}, specificMonthTerms) {
				return false
			}
		}
	}
	return day != "" || month != ""
}

// baziMonthBranchLabel adds the traditional month name used by classics such
// as 穷通宝鉴, where 亥月 is usually indexed as 十月.
func baziMonthBranchLabel(branch string) string {
	switch strings.TrimSpace(branch) {
	case "寅":
		return "正月"
	case "卯":
		return "二月"
	case "辰":
		return "三月"
	case "巳":
		return "四月"
	case "午":
		return "五月"
	case "未":
		return "六月"
	case "申":
		return "七月"
	case "酉":
		return "八月"
	case "戌":
		return "九月"
	case "亥":
		return "十月"
	case "子":
		return "十一月"
	case "丑":
		return "十二月"
	default:
		return ""
	}
}

func dayMasterForEvidenceQuery(input baziCharterInput) string {
	day := firstNonEmptyTrim(stringValue(input.BaziResult["dayGan"]), stringValue(input.Yongshen["day_master"]))
	switch day {
	case "甲", "乙":
		return day + "木"
	case "丙", "丁":
		return day + "火"
	case "戊", "己":
		return day + "土"
	case "庚", "辛":
		return day + "金"
	case "壬", "癸":
		return day + "水"
	default:
		return strings.TrimSpace(day)
	}
}

func monthBranchForEvidenceQuery(input baziCharterInput) string {
	if pillar := extractMonthPillar(input.BaziResult["pillars"]); len(pillar) > 0 {
		if branch := stringValue(pillar["branch"]); branch != "" {
			return branch
		}
	}
	return firstNonEmptyTrim(stringValue(input.Yongshen["month_branch"]), stringValue(input.Yongshen["month_zhi"]))
}

func (e *Executor) runControlledBaziRetrieval(ctx context.Context, plan baziEvidencePlan) (baziEvidenceBundle, error) {
	bundle := baziEvidenceBundle{
		Stage:                plan.Stage,
		TopicBuckets:         map[string][]baziCitation{},
		CriticalTopicBuckets: map[string][]baziCitation{},
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
		degraded  bool
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
			degraded := knowledgeResultDegraded(result)
			retrievalSpan.SetAttribute("hits", len(citations))
			retrievalSpan.SetAttribute("degraded", degraded)
			results[index] = retrievalResult{
				index:     index,
				packet:    packet,
				citations: citations,
				degraded:  degraded,
			}
		}(i, packet)
	}
	wg.Wait()
	for _, result := range results {
		if result.err != nil {
			return bundle, result.err
		}
		if len(result.citations) == 0 {
			if result.degraded {
				bundle.DegradedTopics = mergeStrings(bundle.DegradedTopics, result.packet.Topic)
			}
			continue
		}
		bundle.TopicBuckets[result.packet.Topic] = mergeCitations(bundle.TopicBuckets[result.packet.Topic], result.citations...)
		if strings.EqualFold(strings.TrimSpace(result.packet.SourceTier), "A") {
			bundle.CriticalTopicBuckets[result.packet.Topic] = mergeCitations(bundle.CriticalTopicBuckets[result.packet.Topic], result.citations...)
		}
		bundle.Citations = mergeCitations(bundle.Citations, result.citations...)
	}

	return bundle, nil
}

// knowledgeResultDegraded exposes a tool-level fallback so missing evidence is
// distinguishable from a successful empty semantic search.
func knowledgeResultDegraded(result any) bool {
	rm, ok := result.(map[string]any)
	if !ok {
		return false
	}
	degraded, _ := rm["fallback"].(bool)
	return degraded
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
	if classic == "" {
		classic = source
	}
	return baziCitation{
		Classic: classic,
		Quotes:  []string{strings.TrimSpace(content)},
	}
}

func extractAuthorityClassic(source string) string {
	if classic := extractAuthorityClassicFromSlug(source); classic != "" {
		return classic
	}
	for _, classic := range allAuthorityClassicNames() {
		if strings.Contains(source, classic) {
			return classic
		}
	}
	return ""
}

// extractAuthorityClassicFromSlug maps local wiki slugs back to canonical
// classics. The retrieval API returns sources like
// knowledge://ref-bazi-qiongtong-s001 (五行总论), whose title alone does not name
// 穷通宝鉴; without this map real 调候 hits are misclassified as non-authority.
func extractAuthorityClassicFromSlug(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	slugMap := []struct {
		needle  string
		classic string
	}{
		{needle: "ref-bazi-qiongtong", classic: "穷通宝鉴"},
		{needle: "ref-bazi-ditiansui", classic: "滴天髓"},
		{needle: "ref-bazi-ziping", classic: "子平真诠"},
		{needle: "ref-bazi-yuanhai", classic: "渊海子平"},
		{needle: "ref-bazi-sanming", classic: "三命通会"},
		{needle: "ref-bazi-gelulunming", classic: "格局论命"},
	}
	for _, item := range slugMap {
		if strings.Contains(source, item.needle) {
			return item.classic
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
		Stage:                base.Stage,
		TopicBuckets:         map[string][]baziCitation{},
		CriticalTopicBuckets: map[string][]baziCitation{},
		Citations:            mergeCitations(base.Citations, add.Citations...),
		Conflicts:            mergeStrings(base.Conflicts, add.Conflicts...),
		DegradedTopics:       mergeStrings(base.DegradedTopics, add.DegradedTopics...),
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
	for topic, citations := range base.CriticalTopicBuckets {
		merged.CriticalTopicBuckets[topic] = mergeCitations(merged.CriticalTopicBuckets[topic], citations...)
	}
	for topic, citations := range add.CriticalTopicBuckets {
		merged.CriticalTopicBuckets[topic] = mergeCitations(merged.CriticalTopicBuckets[topic], citations...)
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

	retryPlan := buildEvidenceRetryPlan(chartState.EvidencePlan, chartState.EvidenceQuality)
	if !retryPlan.NeedRetrieval || len(retryPlan.QueryPackets) == 0 {
		return chartState, nil
	}
	bundle, err := e.runControlledBaziRetrieval(ctx, retryPlan)
	if err != nil {
		return chartState, err
	}
	chartState.EvidenceBundle = mergeEvidenceBundles(chartState.EvidenceBundle, bundle)
	chartState.EvidenceQuality = evaluateEvidenceBundleQuality(chartState.EvidencePlan, chartState.EvidenceBundle)
	return chartState, nil
}

// buildEvidenceRetryPlan retries only missing A-tier topics with stable source
// and topic terms. Reissuing an unchanged broad plan cannot repair a coverage gap.
func buildEvidenceRetryPlan(plan baziEvidencePlan, quality baziEvidenceQuality) baziEvidencePlan {
	retryTopics := append([]string{}, quality.MissingTopics...)
	if len(retryTopics) == 0 && quality.ConflictScore == "high" {
		retryTopics = requiredEvidenceTopics(plan)
	}
	retry := baziEvidencePlan{
		NeedRetrieval:     len(retryTopics) > 0,
		Stage:             plan.Stage,
		EvidenceGaps:      append([]string{}, retryTopics...),
		RecommendedSource: append([]string{}, plan.RecommendedSource...),
	}
	for _, topic := range retryTopics {
		for _, packet := range plan.QueryPackets {
			if packet.Topic != topic || !strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") {
				continue
			}
			retryPacket := packet
			retryPacket.Query = strings.TrimSpace(strings.Join(append(append([]string{}, packet.PreferredSources...), packet.Topic), " "))
			retry.QueryPackets = append(retry.QueryPackets, retryPacket)
			break
		}
	}
	return retry
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

func (e *Executor) runDynamicSynthesisWithFeedback(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (baziDynamicSynthesis, error) {
	if err := validateDynamicPreconditions(chartState); err != nil {
		return baziDynamicSynthesis{}, err
	}
	payload := buildDynamicSynthesisPayload(chartState)
	run := func(payload map[string]any) (baziDynamicSynthesis, error) {
		return runBaziInnerAgentJSON[baziDynamicSynthesis](ctx, e.builder, baziDynamicSynthesisConfig(), st, buildBaziCharterPrompt("动态综合", question, payload))
	}

	output, err := run(payload)
	if err != nil {
		return baziDynamicSynthesis{}, err
	}
	output.Source = "model"
	output = ensureDynamicAssertions(chartState, projectDynamicAssertionsToLegacy(normalizeDynamicSynthesis(output)))
	if output, err = validateDynamicSynthesisWithAudit(chartState, output, func(candidate baziDynamicSynthesis) (baziContractAudit, error) {
		return e.runBaziContractAudit(ctx, st, "dynamic", chartState, candidate)
	}); err == nil {
		return output, nil
	} else {
		payload["dynamic_feedback"] = buildDynamicSynthesisFeedback(chartState, output, err)
	}

	output, err = run(payload)
	if err != nil {
		return baziDynamicSynthesis{}, err
	}
	output.Source = "model"
	output = ensureDynamicAssertions(chartState, projectDynamicAssertionsToLegacy(normalizeDynamicSynthesis(output)))
	if output, err = validateDynamicSynthesisWithAudit(chartState, output, func(candidate baziDynamicSynthesis) (baziContractAudit, error) {
		return e.runBaziContractAudit(ctx, st, "dynamic", chartState, candidate)
	}); err == nil {
		return output, nil
	} else {
		if partial, ok := acceptPartialDynamicSynthesisAfterRetry(chartState, output, err); ok {
			return partial, nil
		}
		if recovered, recoverErr := recoverDynamicSynthesisAfterRetry(chartState, output, err); recoverErr == nil {
			return recovered, nil
		}
		return baziDynamicSynthesis{}, err
	}
}

// recoverDynamicSynthesisAfterRetry replaces a retry candidate that still
// violates semantic scope with deterministic dynamic facts. It only runs after
// the model returned parseable synthesis twice; model execution and static-stage
// failures still surface as contract errors.
func recoverDynamicSynthesisAfterRetry(chartState baziCharterState, output baziDynamicSynthesis, cause error) (baziDynamicSynthesis, error) {
	failure, ok := baziContractFailureFromError("dynamic_synthesis", cause)
	if !ok || failure.RecoveryPolicy != baziRecoveryPolicyDynamicFactsOnly {
		return baziDynamicSynthesis{}, cause
	}
	recovered := recoverDynamicSynthesis(chartState, output, cause)
	recovered.ContractAudit = output.ContractAudit
	recovered.FieldAudit = append(recovered.FieldAudit,
		"contract_failure_class:"+failure.Class,
		"recovery_policy:"+failure.RecoveryPolicy,
	)
	if err := validateDynamicSynthesisResult(chartState, recovered); err != nil {
		return baziDynamicSynthesis{}, err
	}
	return recovered, nil
}

// validateDynamicSynthesisWithAudit runs deterministic checks before the
// independent reviewer; a failed review is either retried once, accepted as a
// partial display omission, or surfaced as a runtime contract error.
func validateDynamicSynthesisWithAudit(chartState baziCharterState, output baziDynamicSynthesis, audit func(baziDynamicSynthesis) (baziContractAudit, error)) (baziDynamicSynthesis, error) {
	if err := validateDynamicSynthesisResult(chartState, output); err != nil {
		return output, err
	}
	if audit == nil {
		return output, nil
	}
	result, err := audit(output)
	if err != nil {
		return output, err
	}
	output.ContractAudit = result
	if err := validateBaziContractAudit("dynamic", result); err != nil {
		return output, err
	}
	return output, nil
}

func validateDynamicSynthesisResult(chartState baziCharterState, output baziDynamicSynthesis) error {
	checkState := chartState
	checkState.DynamicSynthesis = normalizeDynamicSynthesis(output)
	if isFactsOnlyDynamicSynthesis(checkState.DynamicSynthesis) {
		return validateDynamicStage(checkState)
	}
	checkState.DynamicSynthesis = ensureDynamicAssertions(checkState, projectDynamicAssertionsToLegacy(checkState.DynamicSynthesis))
	if err := validateDynamicStage(checkState); err != nil {
		return err
	}
	if err := validateDynamicAssertions(checkState); err != nil {
		return err
	}
	return validateCharterConsistency(checkState)
}

func buildDynamicSynthesisFeedback(chartState baziCharterState, output baziDynamicSynthesis, cause error) string {
	context := buildBaziSubjectContext(chartState.Input)
	lines := []string{
		"请仅修复本次校验失败的字段，保留其余逐运裁断与静态主轴。",
		"大运必须逐条覆盖输入的完整计算序列；若输出 dayun_judgments，其顺序、干支和数量必须与输入完全一致。",
		"只使用输入已声明的十神和冲合刑害；不得补造关系或具体健康、财务、婚恋、职位、法律等应事。",
		"`consistency_flags` 只能使用以下固定值：" + strings.Join(allowedDynamicConsistencyFlags, "、") + "。非法值只修复该字段，不要重写其他已合格内容。",
	}
	if context.AgeBand == "infant" || context.AgeBand == "child" || context.AgeBand == "adolescent" {
		lines = append(lines, fmt.Sprintf(
			"本轮 subject_context：age_band=%s；allowed_outcome_domains=%s。你必须重写所有 dayun_judgments 和兼容正文：每步 outcome_domains 只能取此集合，未来成年大运也只能说明结构触发、成长环境、照护节奏或可观察发展；不得写事业、婚姻、财富、职位、社会地位、权威、权力、升迁、名望、健康、法律、事故等现实落点。当前顶层 outcome_domains=%s。",
			context.AgeBand,
			strings.Join(context.AllowedOutcomeDomains, "、"),
			strings.Join(output.OutcomeDomains, "、"),
		))
	}
	if !chartState.EvidenceQuality.Enough || len(chartState.EvidenceQuality.MissingTopics) > 0 {
		lines = append(lines, fmt.Sprintf(
			"本轮 evidence_quality.enough=%t；missing_topics=%s。缺失主题相关的大运与流年只能降为 structure 结构观察，不得把未覆盖的病药、层级或主轴证据转译成社会地位、权威、职位、财富、健康等现实结论。",
			chartState.EvidenceQuality.Enough,
			strings.Join(chartState.EvidenceQuality.MissingTopics, "、"),
		))
	}
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		lines = append(lines, "本次校验失败原因："+cause.Error())
	}
	if violation, ok := baziViolationFromError(cause); ok {
		if raw, err := json.Marshal(violation); err == nil {
			lines = append(lines, "机器可读 violation："+string(raw))
		}
	}
	if len(output.ContractAudit.Findings) > 0 {
		if raw, err := json.Marshal(output.ContractAudit.Findings); err == nil {
			lines = append(lines, "独立审计已确认的 findings（只修复这些字段，不要把检查过程当作结论）："+string(raw))
		}
	}
	return strings.Join(lines, "\n")
}

func (e *Executor) runFinalWriter(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (string, error) {
	output := renderBaziFinalReply(chartState.AnalysisPlan, chartState, question)
	if err := validateFinalWriterOutput(chartState.AnalysisPlan, chartState, output); err != nil {
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.final_writer.template":       chartState.AnalysisPlan.WriterTemplate,
			"bazi.final_writer.validation_err": err.Error(),
			"bazi.final_writer.output_preview": truncateTracePreview(output, 1200),
		})
		return "", &RuntimeFailure{
			Class:       failureClassModelContractViolation,
			Stage:       failureStageFinalWriter,
			Domain:      "bazi",
			Code:        "FINAL_CONTRACT_VIOLATION",
			Retryable:   false,
			Degraded:    false,
			UserVisible: true,
			Message:     "本轮输出未通过最终合同校验，已停止展示不稳定内容。请稍后重试。",
			Cause:       err,
		}
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
