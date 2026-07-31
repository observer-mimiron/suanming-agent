package runtime

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type baziValidationError struct {
	Violation baziValidationViolation
}

func (e baziValidationError) Error() string {
	if strings.TrimSpace(e.Violation.Message) == "" {
		return string(e.Violation.Code)
	}
	return fmt.Sprintf("%s: %s", e.Violation.Code, e.Violation.Message)
}

func baziViolationError(code baziViolationCode, field, assertionID, message string, missing, allowed []string) error {
	return baziValidationError{Violation: baziValidationViolation{
		Code:        code,
		Field:       field,
		Message:     message,
		AssertionID: assertionID,
		MissingRefs: filterNonEmpty(missing),
		AllowedRefs: filterNonEmpty(allowed),
	}}
}

func baziViolationFromError(err error) (baziValidationViolation, bool) {
	var validationErr baziValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Violation, true
	}
	return baziValidationViolation{}, false
}

func ensureStaticAssertions(state baziCharterState, in baziStaticSynthesis) baziStaticSynthesis {
	if len(in.Assertions) > 0 {
		return in
	}
	profile := state.Input.RuleProfile
	claim := firstClaimRefByCategory(profile, "main_axis", "pattern_candidate")
	strengthClaim := firstClaimRefByCategory(profile, "strength")
	tiaohouClaim := firstVerdictRefByPrefix(profile, "qiongtong_")
	tierClaim := firstClaimRefByCategory(profile, "tier")
	in.Assertions = []baziAssertion{
		{
			ID:         "static.main_axis",
			Kind:       baziAssertionMainAxis,
			Subject:    "chart",
			Verdict:    firstNonEmptyTrim(in.MainAxis, in.PatternOutcome),
			FactRefs:   []baziFactRef{"chart.month_branch", "yongshen.geju_candidate"},
			ClaimRefs:  claim,
			Confidence: in.ClaimStrength,
			Boundary:   firstNonEmptyTrim(in.CounterEvidence, profileBoundaryByRef(profile, claim)),
		},
		{
			ID:         "static.strength",
			Kind:       baziAssertionStrength,
			Subject:    "day_master",
			Verdict:    firstNonEmptyTrim(in.Strength.Conclusion, in.StrengthBalance),
			FactRefs:   []baziFactRef{"yongshen.strength", "yongshen.strength_evidence"},
			ClaimRefs:  strengthClaim,
			Confidence: in.ClaimStrength,
			Boundary:   in.Strength.Boundary,
		},
		{
			ID:         "static.tiaohou",
			Kind:       baziAssertionTiaohou,
			Subject:    "chart",
			Verdict:    firstNonEmptyTrim(in.TiaohouAnchor, in.TiaohouConstraint),
			FactRefs:   []baziFactRef{"chart.day_gan", "chart.month_branch"},
			ClaimRefs:  tiaohouClaim,
			Confidence: in.ClaimStrength,
			Boundary:   in.TiaohouConstraint,
		},
		{
			ID:         "static.tier",
			Kind:       baziAssertionTier,
			Subject:    "chart",
			Verdict:    in.TierJudgment,
			FactRefs:   []baziFactRef{"chart.month_branch", "yongshen.strength_evidence"},
			ClaimRefs:  tierClaim,
			Confidence: in.ClaimStrength,
			Boundary:   in.TierBasis,
		},
	}
	if strings.TrimSpace(in.TopicDirectAnswer) != "" {
		in.Assertions = append(in.Assertions, baziAssertion{
			ID:         "static.topic_answer",
			Kind:       baziAssertionTopicAnswer,
			Subject:    "question",
			Verdict:    in.TopicDirectAnswer,
			FactRefs:   []baziFactRef{"chart.month_branch"},
			ClaimRefs:  claim,
			Confidence: in.ClaimStrength,
			Boundary:   firstNonEmptyTrim(in.TopicFocusAnswer, in.CounterEvidence),
		})
	}
	return in
}

func ensureDynamicAssertions(state baziCharterState, in baziDynamicSynthesis) baziDynamicSynthesis {
	if len(in.Assertions) > 0 {
		return in
	}
	dynamicClaim := firstClaimRefByCategory(state.Input.RuleProfile, "dynamic_framework")
	periods := dayunPeriods(state.Input.Dayun)
	if len(in.DayunJudgments) > 0 {
		for i, judgment := range in.DayunJudgments {
			if strings.TrimSpace(judgment.GanZhi) == "" && i < len(periods) {
				judgment.GanZhi = strings.TrimSpace(stringValue(periods[i]["ganZhi"]))
			}
			in.Assertions = append(in.Assertions, dayunAssertionFromParts(i, judgment.GanZhi, judgment.Trend, judgment.Interpretation, dynamicClaim))
		}
	} else {
		for i, line := range in.DayunPath {
			ganZhi := ""
			if i < len(periods) {
				ganZhi = strings.TrimSpace(stringValue(periods[i]["ganZhi"]))
			}
			in.Assertions = append(in.Assertions, dayunAssertionFromParts(i, ganZhi, periodHeadline(line), line, dynamicClaim))
		}
	}
	if strings.TrimSpace(in.LiunianFocus) != "" {
		in.Assertions = append(in.Assertions, baziAssertion{
			ID:         "dynamic.liunian",
			Kind:       baziAssertionLiunian,
			Subject:    "liunian",
			Verdict:    in.LiunianFocus,
			FactRefs:   []baziFactRef{"liunian.gan_zhi", "liunian.relations"},
			ClaimRefs:  dynamicClaim,
			Confidence: in.ClaimStrength,
			Boundary:   firstNonEmptyTrim(strings.Join(in.Risks, "；"), "具体应事不作展开。"),
		})
	}
	return in
}

func dayunAssertionFromParts(index int, ganZhi, trend, interpretation string, claimRefs []baziClaimRef) baziAssertion {
	return baziAssertion{
		ID:         fmt.Sprintf("dynamic.dayun.%d", index),
		Kind:       baziAssertionDayunPeriod,
		Subject:    fmt.Sprintf("dayun[%d]", index),
		Verdict:    firstNonEmptyTrim(strings.TrimSpace(trend), strings.TrimSpace(interpretation)),
		FactRefs:   []baziFactRef{baziFactRef(fmt.Sprintf("dayun[%d].gan_zhi", index)), baziFactRef(fmt.Sprintf("dayun[%d].relations", index))},
		ClaimRefs:  claimRefs,
		Confidence: "保守判断",
		Boundary:   "仅解释已计算的大运事实与当前 profile 授权维度，不展开具体生活事件。",
	}
}

func validateStaticAssertions(state baziCharterState) error {
	static := ensureStaticAssertions(state, projectStaticAssertionsToLegacy(state.StaticSynthesis))
	return validateBaziAssertions(state, static.Assertions)
}

func validateDynamicAssertions(state baziCharterState) error {
	dynamic := ensureDynamicAssertions(state, projectDynamicAssertionsToLegacy(state.DynamicSynthesis))
	if expected := len(dayunPeriods(state.Input.Dayun)); expected > 0 && countAssertionsByKind(dynamic.Assertions, baziAssertionDayunPeriod) < expected {
		return baziViolationError(baziViolationDayunCoverageMissing, "dynamic.assertions", "", fmt.Sprintf("dynamic assertions omit calculated dayun periods: got %d, want %d", countAssertionsByKind(dynamic.Assertions, baziAssertionDayunPeriod), expected), nil, nil)
	}
	if err := validateDayunAssertionsAgainstFacts(state, dynamic.Assertions); err != nil {
		return err
	}
	return validateBaziAssertions(state, dynamic.Assertions)
}

func validateBaziAssertions(state baziCharterState, assertions []baziAssertion) error {
	for _, assertion := range assertions {
		if strings.TrimSpace(assertion.ID) == "" {
			return baziViolationError(baziViolationScopeEscalation, "assertions", "", "assertion id is required", nil, nil)
		}
		if !allowedBaziAssertionKind(assertion.Kind) {
			return baziViolationError(baziViolationScopeEscalation, "assertions.kind", assertion.ID, "assertion kind is outside the closed set", nil, nil)
		}
		if strings.TrimSpace(assertion.Verdict) == "" {
			return baziViolationError(baziViolationScopeEscalation, "assertions.verdict", assertion.ID, "assertion verdict is required", nil, nil)
		}
		// Fact-ref paths are model-authored provenance metadata. Unknown aliases are
		// audited in the trace, while concrete period and chart contradictions are
		// validated below by deterministic checks. A transport spelling mismatch
		// must not discard an otherwise usable interpretation.
		if containsUnsupportedConcreteOutcome(assertion.Verdict) || containsUnsupportedConcreteOutcome(assertion.Boundary) {
			return baziViolationError(baziViolationUnsupportedConcreteOutcome, "assertions.verdict", assertion.ID, "assertion includes unsupported concrete life outcome", nil, nil)
		}
	}
	return nil
}

func projectStaticAssertionsToLegacy(in baziStaticSynthesis) baziStaticSynthesis {
	for _, assertion := range in.Assertions {
		verdict := strings.TrimSpace(assertion.Verdict)
		if verdict == "" {
			continue
		}
		switch assertion.Kind {
		case baziAssertionMainAxis:
			if strings.TrimSpace(in.MainAxis) == "" {
				in.MainAxis = verdict
			}
		case baziAssertionStrength:
			if strings.TrimSpace(in.Strength.Conclusion) == "" {
				in.Strength.Conclusion = verdict
			}
			if strings.TrimSpace(in.StrengthBalance) == "" {
				in.StrengthBalance = verdict
			}
		case baziAssertionTiaohou:
			if strings.TrimSpace(in.TiaohouAnchor) == "" {
				in.TiaohouAnchor = verdict
			}
		case baziAssertionTier:
			if strings.TrimSpace(in.TierJudgment) == "" {
				in.TierJudgment = verdict
			}
		case baziAssertionTopicAnswer:
			if strings.TrimSpace(in.TopicDirectAnswer) == "" {
				in.TopicDirectAnswer = verdict
			}
		}
	}
	return in
}

func projectDynamicAssertionsToLegacy(in baziDynamicSynthesis) baziDynamicSynthesis {
	if len(in.DayunPath) == 0 {
		for _, assertion := range in.Assertions {
			if assertion.Kind != baziAssertionDayunPeriod || strings.TrimSpace(assertion.Verdict) == "" {
				continue
			}
			in.DayunPath = append(in.DayunPath, assertion.Verdict)
		}
	}
	if strings.TrimSpace(in.LiunianFocus) == "" {
		for _, assertion := range in.Assertions {
			if assertion.Kind == baziAssertionLiunian && strings.TrimSpace(assertion.Verdict) != "" {
				in.LiunianFocus = assertion.Verdict
				break
			}
		}
	}
	return in
}

func allowedBaziAssertionKind(kind baziAssertionKind) bool {
	switch kind {
	case baziAssertionMainAxis, baziAssertionStrength, baziAssertionTiaohou, baziAssertionPatternUsage, baziAssertionTier, baziAssertionDayunPeriod, baziAssertionLiunian, baziAssertionTopicAnswer:
		return true
	default:
		return false
	}
}

func knownBaziClaimRefs(profile baziRuleProfile) map[string]struct{} {
	out := map[string]struct{}{}
	for _, claim := range profile.Claims {
		if strings.TrimSpace(claim.ID) != "" {
			out[claim.ID] = struct{}{}
		}
		if strings.TrimSpace(claim.RuleID) != "" {
			out[claim.RuleID] = struct{}{}
		}
	}
	for _, verdict := range profile.Verdicts {
		if strings.TrimSpace(verdict.RuleID) != "" {
			out[verdict.RuleID] = struct{}{}
		}
	}
	return out
}

func claimRefAllowsAssertionKind(profile baziRuleProfile, ref string, kind baziAssertionKind) bool {
	category := ""
	for _, claim := range profile.Claims {
		if claim.ID == ref || claim.RuleID == ref {
			category = claim.Category
			break
		}
	}
	if category == "" {
		for _, verdict := range profile.Verdicts {
			if verdict.RuleID == ref && strings.HasPrefix(verdict.RuleID, "qiongtong_") {
				return kind == baziAssertionTiaohou
			}
			if verdict.RuleID == ref {
				return true
			}
		}
	}
	switch category {
	case "main_axis":
		return kind == baziAssertionMainAxis || kind == baziAssertionPatternUsage || kind == baziAssertionTopicAnswer
	case "pattern_candidate":
		return kind == baziAssertionPatternUsage || kind == baziAssertionTopicAnswer
	case "pattern_material":
		return kind == baziAssertionPatternUsage || kind == baziAssertionTopicAnswer
	case "strength":
		return kind == baziAssertionStrength
	case "fuyi_yongji":
		return kind == baziAssertionPatternUsage || kind == baziAssertionTopicAnswer
	case "dynamic_framework":
		return kind == baziAssertionDayunPeriod || kind == baziAssertionLiunian
	case "tier":
		return kind == baziAssertionTier
	case "":
		return true
	default:
		return true
	}
}

func knownBaziFactRefs(state baziCharterState) map[string]struct{} {
	out := map[string]struct{}{
		"chart.day_gan": {}, "chart.day_master": {}, "chart.day_master_wuxing": {}, "chart.month_branch": {},
		"chart.month_pillar": {}, "chart.pillars": {}, "chart.wuxing": {},
		"yongshen.balance_status": {}, "yongshen.balance_yong_shen": {}, "yongshen.conditional_yong_shen": {},
		"yongshen.day_master": {}, "yongshen.day_master_wuxing": {}, "yongshen.geju": {},
		"yongshen.geju_basis": {}, "yongshen.geju_candidate": {}, "yongshen.geju_combination": {},
		"yongshen.geju_detail": {}, "yongshen.geju_qing_zhuo": {}, "yongshen.geju_qing_zhuo_reason": {},
		"yongshen.geju_status": {}, "yongshen.ji_shen": {}, "yongshen.official_visibility": {},
		"yongshen.official_visibility.visible": {}, "yongshen.official_visibility.hidden": {},
		"yongshen.season": {}, "yongshen.seasonal_tiaohou_hint": {}, "yongshen.shi_shen_power": {},
		"yongshen.strength": {}, "yongshen.strength_evidence": {}, "yongshen.strength_method": {},
		"yongshen.tiao_hou": {}, "yongshen.tiaohou_yong_shen": {}, "yongshen.xi_shen": {}, "yongshen.yong_shen": {},
		"liunian.branch": {}, "liunian.current_dayun": {}, "liunian.gan_zhi": {}, "liunian.relations": {},
		"liunian.shen_sha": {}, "liunian.shi_shen": {}, "liunian.stem": {}, "liunian.year": {},
	}
	for i := range dayunPeriods(state.Input.Dayun) {
		out[fmt.Sprintf("dayun[%d].end_age", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].end_at_exclusive", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].gan_zhi", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].period_id", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].relations", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].sequence", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].start_age", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].start_at", i)] = struct{}{}
		out[fmt.Sprintf("dayun[%d].ten_god", i)] = struct{}{}
	}
	return out
}

func isKnownBaziFactRef(ref baziFactRef, known map[string]struct{}) bool {
	canonical := normalizeBaziFactRef(string(ref))
	if _, ok := known[canonical]; ok {
		return true
	}
	for base := range known {
		if isExpandableBaziFactRef(base) && (strings.HasPrefix(canonical, base+".") || strings.HasPrefix(canonical, base+"[")) {
			return true
		}
	}
	return false
}

func normalizeBaziFactRef(ref string) string {
	ref = strings.TrimSpace(ref)
	for _, prefix := range []string{"input.", "core_chart.", "chart_facts.", "static_facts.", "dynamic_facts."} {
		ref = strings.TrimPrefix(ref, prefix)
	}
	for _, prefix := range []string{"dayun.dayun_analyzed[", "dayun.periods[", "dayun_analyzed[", "periods["} {
		if strings.HasPrefix(ref, prefix) {
			ref = "dayun[" + strings.TrimPrefix(ref, prefix)
			break
		}
	}
	ref = strings.ReplaceAll(ref, ".ganZhi", ".gan_zhi")
	ref = strings.ReplaceAll(ref, ".tenGod", ".ten_god")
	ref = strings.ReplaceAll(ref, ".startAge", ".start_age")
	ref = strings.ReplaceAll(ref, ".endAge", ".end_age")
	ref = strings.ReplaceAll(ref, ".startAt", ".start_at")
	ref = strings.ReplaceAll(ref, ".endAtExclusive", ".end_at_exclusive")
	ref = strings.ReplaceAll(ref, ".dayun_chonghe", ".relations")
	ref = strings.ReplaceAll(ref, ".liunian_chonghe", ".relations")
	switch ref {
	case "day_master":
		return "chart.day_master"
	case "day_master_wuxing":
		return "chart.day_master_wuxing"
	case "liunian.liunian_ganzhi", "liunian_ganzhi":
		return "liunian.gan_zhi"
	case "liunian.liunian_shi_shen", "liunian_shi_shen":
		return "liunian.shi_shen"
	case "liunian.liunian_year", "liunian_year":
		return "liunian.year"
	case "liunian.liunian_stem", "liunian_stem":
		return "liunian.stem"
	case "liunian.liunian_branch", "liunian_branch":
		return "liunian.branch"
	case "liunian.liunian_chonghe", "liunian_chonghe":
		return "liunian.relations"
	}
	return ref
}

func isExpandableBaziFactRef(base string) bool {
	return strings.HasPrefix(base, "chart.month_pillar") ||
		strings.HasPrefix(base, "chart.pillars") ||
		strings.HasPrefix(base, "chart.wuxing") ||
		strings.HasPrefix(base, "yongshen.") ||
		strings.HasPrefix(base, "dayun[") ||
		strings.HasPrefix(base, "liunian.current_dayun") ||
		strings.HasPrefix(base, "liunian.relations")
}

func firstClaimRefByCategory(profile baziRuleProfile, categories ...string) []baziClaimRef {
	for _, category := range categories {
		for _, claim := range profile.Claims {
			if claim.Category != category {
				continue
			}
			if strings.TrimSpace(claim.ID) != "" {
				return []baziClaimRef{baziClaimRef(claim.ID)}
			}
			if strings.TrimSpace(claim.RuleID) != "" {
				return []baziClaimRef{baziClaimRef(claim.RuleID)}
			}
		}
	}
	return nil
}

func firstVerdictRefByPrefix(profile baziRuleProfile, prefix string) []baziClaimRef {
	for _, verdict := range profile.Verdicts {
		if strings.HasPrefix(verdict.RuleID, prefix) {
			return []baziClaimRef{baziClaimRef(verdict.RuleID)}
		}
	}
	return nil
}

func profileBoundaryByRef(profile baziRuleProfile, refs []baziClaimRef) string {
	if len(refs) == 0 {
		return ""
	}
	target := string(refs[0])
	for _, claim := range profile.Claims {
		if claim.ID == target || claim.RuleID == target {
			return claim.Boundary
		}
	}
	for _, verdict := range profile.Verdicts {
		if verdict.RuleID == target {
			return verdict.Boundary
		}
	}
	return ""
}

func containsUnsupportedConcreteOutcome(text string) bool {
	return containsAnyText([]string{text}, []string{
		"官非", "高血压", "心血管", "血光",
	})
}

func referencesQiongtongSequence(assertion baziAssertion) bool {
	text := strings.Join([]string{assertion.Verdict, assertion.Boundary}, "\n")
	return containsAnyText([]string{text}, []string{"《穷通宝鉴》", "先丙", "先甲", "先癸", "首需", "次喜", "调候用神"})
}

func assertionHasClaimPrefix(assertion baziAssertion, prefix string) bool {
	for _, ref := range assertion.ClaimRefs {
		if strings.HasPrefix(string(ref), prefix) {
			return true
		}
	}
	return false
}

func validateDayunAssertionsAgainstFacts(state baziCharterState, assertions []baziAssertion) error {
	periods := dayunPeriods(state.Input.Dayun)
	for _, assertion := range assertions {
		if assertion.Kind != baziAssertionDayunPeriod {
			continue
		}
		index, ok := dayunAssertionIndex(assertion)
		if !ok || index < 0 || index >= len(periods) {
			return baziViolationError(baziViolationFactRefMissing, "assertions.fact_refs", assertion.ID, "dayun assertion must reference one calculated period", nil, nil)
		}
		want := strings.TrimSpace(stringValue(periods[index]["ganZhi"]))
		if want == "" {
			continue
		}
		if mentioned := firstGanZhiInText(strings.Join([]string{assertion.Subject, assertion.Verdict, assertion.Boundary}, "\n")); mentioned != "" && mentioned != want {
			return baziViolationError(baziViolationFactConflict, "assertions.verdict", assertion.ID, fmt.Sprintf("dayun assertion mentions %s but references calculated period %s", mentioned, want), nil, []string{want})
		}
	}
	return nil
}

func dayunAssertionIndex(assertion baziAssertion) (int, bool) {
	for _, text := range append([]string{assertion.Subject, assertion.ID}, factRefsToStrings(assertion.FactRefs)...) {
		start := strings.Index(text, "dayun[")
		if start < 0 {
			continue
		}
		rest := text[start+len("dayun["):]
		end := strings.Index(rest, "]")
		if end <= 0 {
			continue
		}
		var index int
		if _, err := fmt.Sscanf(rest[:end], "%d", &index); err == nil {
			return index, true
		}
	}
	return 0, false
}

func factRefsToStrings(refs []baziFactRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, string(ref))
	}
	return out
}

func firstGanZhiInText(text string) string {
	stems := "甲乙丙丁戊己庚辛壬癸"
	branches := "子丑寅卯辰巳午未申酉戌亥"
	runes := []rune(text)
	for i := 0; i+1 < len(runes); i++ {
		if strings.ContainsRune(stems, runes[i]) && strings.ContainsRune(branches, runes[i+1]) {
			return string(runes[i : i+2])
		}
	}
	return ""
}

func countAssertionsByKind(assertions []baziAssertion, kind baziAssertionKind) int {
	count := 0
	for _, assertion := range assertions {
		if assertion.Kind == kind {
			count++
		}
	}
	return count
}

func sortedKeys(items map[string]struct{}) []string {
	out := make([]string, 0, len(items))
	for item := range items {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
