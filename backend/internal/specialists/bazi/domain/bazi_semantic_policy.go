// Package domain 包含八字语义范围与安全边界规则。
//
// 本文件用类型化槽位和事实胶囊校验模型裁断资格；不扫描最终中文文案，
// 不渲染展示文本，也不负责图节点调度。
package domain

import (
	"fmt"
	"sort"
	"strings"
)

var baziTierStatusValues = []string{"rated", "provisional", "withheld"}
var baziStructuredConfidenceValues = []string{"保守判断", "倾向成立", "明确成立"}
var baziTierDimensionStateValues = []string{"missing", "limited", "mixed", "usable", "strong"}
var baziTierDiseaseStateValues = []string{"unresolved", "light", "moderate", "heavy", "critical"}
var baziCurrentPeriodRealizationValues = []string{"repair", "assist", "maintain", "disturb", "suppress"}
var baziTierProseMarkers = []string{"层次", "等级", "第1级", "第2级", "第3级", "第4级", "第5级", "第6级", "第7级", "第8级", "第9级", "中格", "中上", "上格", "上上", "下等"}
var baziStaticClaimSlots = []string{"main_axis", "strength", "tiaohou", "pattern_usage"}

type baziTierBounds struct {
	Min int
	Max int
}

type baziNamedTierDimension struct {
	Name    string
	Value   baziTierDimension
	Disease bool
}

// validateBaziStaticJudgmentPolicy enforces static eligibility before legacy projection.
func validateBaziStaticJudgmentPolicy(state baziCharterState, judgment baziStructuredStaticSynthesis) error {
	claims, err := NormalizeStaticClaims(judgment.Claims)
	if err != nil {
		return err
	}
	judgment.Claims = claims
	facts := buildBaziFactCapsule(state)
	if !containsString([]string{"candidate", "established", "withheld"}, judgment.AxisStatus) {
		return baziViolationError(baziViolationMethodContract, "static.axis_status", "", "static axis status is outside the closed contract", nil, nil)
	}
	if !containsString([]string{"none", "withheld"}, judgment.NatalRiskStatus) {
		return baziViolationError(baziViolationMethodContract, "static.natal_risk_status", "", "static natal risk status is outside the closed contract", nil, nil)
	}
	for index, claim := range judgment.Claims {
		if strings.TrimSpace(claim.Verdict) == "" {
			return baziViolationError(baziViolationMethodContract, fmt.Sprintf("static.claims[%d].verdict", index), "", "static claim verdict is required", nil, nil)
		}
		if !containsString([]string{"established", "candidate", "limited", "withheld"}, claim.Status) {
			return baziViolationError(baziViolationMethodContract, fmt.Sprintf("static.claims[%d].status", index), "", "static claim status is outside the closed contract", nil, []string{"established", "candidate", "limited", "withheld"})
		}
	}
	if len(judgment.Claims) > 0 && judgment.AxisStatus == "established" && judgment.Claims[0].Status != "established" {
		return baziViolationError(baziViolationMethodContract, "static.claims[0].status", "static.main_axis", "established axis requires an established main-axis claim", nil, []string{"established"})
	}
	if len(judgment.Claims) > 0 && judgment.AxisStatus == "candidate" && judgment.Claims[0].Status == "established" {
		return baziViolationError(baziViolationMethodContract, "static.claims[0].status", "static.main_axis", "candidate axis cannot use an established main-axis claim", nil, []string{"candidate", "limited", "withheld"})
	}
	if err := validateStaticMainAxisPattern(state, judgment); err != nil {
		return err
	}
	if err := validateBaziTierAssessment(facts, judgment.AxisStatus, judgment.TierAssessment); err != nil {
		return err
	}
	if !facts.OfficialVisible && judgment.NatalRiskStatus != "withheld" {
		return baziViolationError(baziViolationFactConflict, "static.natal_risk_status", "", "natal official conflict is unavailable when official is not visible", nil, nil)
	}
	return nil
}

// NormalizeStaticClaims 校验四个静态裁断槽位齐全且唯一，并返回固定业务顺序。
// Schema 只能限定单项枚举，不能保证数组里四个槽位各出现一次；这里收口该语义合同，
// 避免模型调整数组顺序时被错误地当成主轴裁断。
func NormalizeStaticClaims(claims []StructuredStaticClaim) ([]StructuredStaticClaim, error) {
	if len(claims) != len(baziStaticClaimSlots) {
		return nil, baziViolationError(baziViolationMethodContract, "static.claims", "", "static synthesis requires four named claim slots", nil, baziStaticClaimSlots)
	}
	bySlot := make(map[string]StructuredStaticClaim, len(claims))
	for index, claim := range claims {
		slot := strings.TrimSpace(claim.Slot)
		if !containsString(baziStaticClaimSlots, slot) {
			return nil, baziViolationError(baziViolationMethodContract, fmt.Sprintf("static.claims[%d].slot", index), "", "static claim slot is outside the closed contract", nil, baziStaticClaimSlots)
		}
		if _, exists := bySlot[slot]; exists {
			return nil, baziViolationError(baziViolationMethodContract, fmt.Sprintf("static.claims[%d].slot", index), "", "static claim slots must be unique", nil, baziStaticClaimSlots)
		}
		bySlot[slot] = claim
	}
	ordered := make([]StructuredStaticClaim, 0, len(baziStaticClaimSlots))
	for _, slot := range baziStaticClaimSlots {
		claim, ok := bySlot[slot]
		if !ok {
			return nil, baziViolationError(baziViolationMethodContract, "static.claims", "", "static synthesis is missing a required claim slot", []string{slot}, baziStaticClaimSlots)
		}
		ordered = append(ordered, claim)
	}
	return ordered, nil
}

// validateStaticMainAxisPattern prevents a free-text main axis from replacing
// the deterministic month-command pattern with a different named pattern.
// The model may still describe the usage route (for example, 伤官佩印), but
// the principal pattern frame must remain the tool candidate.
func validateStaticMainAxisPattern(state baziCharterState, judgment baziStructuredStaticSynthesis) error {
	if len(judgment.Claims) == 0 {
		return nil
	}
	expected := normalizePatternCandidateName(stringValue(state.Input.Yongshen["geju_candidate"]))
	if expected == "" {
		return nil
	}
	axis := strings.ReplaceAll(strings.TrimSpace(judgment.Claims[0].Verdict), " ", "")
	if axis == "" {
		return nil
	}
	principal := map[string][]string{
		"正官格": {"正官格"}, "七杀格": {"七杀格"}, "正财格": {"正财格"}, "偏财格": {"偏财格"},
		"正印格": {"正印格"}, "偏印格": {"偏印格"}, "食神格": {"食神格"}, "伤官格": {"伤官格"},
		"建禄格": {"建禄格", "建禄"}, "月劫格": {"月劫格", "月劫"}, "月刃格": {"月刃格", "月刃"},
	}
	for candidate, markers := range principal {
		if candidate == expected {
			continue
		}
		for _, marker := range markers {
			if !strings.Contains(axis, marker) {
				continue
			}
			return baziViolationError(
				baziViolationFactConflict,
				"static.claims[0].verdict",
				"static.main_axis",
				"main-axis pattern conflicts with deterministic month-command candidate",
				[]string{"yongshen.geju_candidate"},
				[]string{expected},
			)
		}
	}
	return nil
}

// validateBaziTierAssessment verifies model-selected grade bounds without
// calculating a grade. The model chooses a level; runtime only rejects levels
// that exceed the evidenced ceiling or contradict a fixed state.
func validateBaziTierAssessment(facts BaziFactCapsule, axisStatus string, assessment baziTierAssessment) error {
	if !containsString(baziTierStatusValues, assessment.Status) {
		return baziViolationError(baziViolationMethodContract, "static.tier_assessment.status", "", "tier status is outside the closed contract", nil, baziTierStatusValues)
	}
	if !containsString(baziStructuredConfidenceValues, assessment.Confidence) {
		return baziViolationError(baziViolationMethodContract, "static.tier_assessment.confidence", "", "tier confidence is outside the closed contract", nil, baziStructuredConfidenceValues)
	}
	for _, dimension := range baziTierDimensionEntries(assessment.Dimensions) {
		allowed := baziTierDimensionStateValues
		if dimension.Disease {
			allowed = baziTierDiseaseStateValues
		}
		if !containsString(allowed, dimension.Value.State) {
			return baziViolationError(baziViolationMethodContract, "static.tier_assessment.dimensions."+dimension.Name+".state", "", "tier dimension state is outside the closed contract", nil, allowed)
		}
		if assessment.Status != "withheld" && !tierDimensionHasGround(dimension.Value) {
			return baziViolationError(baziViolationEvidenceTopicMissing, "static.tier_assessment.dimensions."+dimension.Name, "", "each rated tier dimension requires a fact, relation, rule or evidence reference", nil, nil)
		}
	}

	if assessment.Status == "withheld" {
		if assessment.Level != 0 {
			return baziViolationError(baziViolationMethodContract, "static.tier_assessment.level", "", "withheld tier must use level 0", nil, []string{"0"})
		}
		if axisStatus != "withheld" && facts.CoreFactsReady {
			return baziViolationError(baziViolationMethodContract, "static.tier_assessment.status", "", "tier may be withheld only when the static axis or core facts cannot be established", nil, []string{"rated", "provisional"})
		}
		return nil
	}

	if !facts.CoreFactsReady || axisStatus == "withheld" {
		return baziViolationError(baziViolationEvidenceTopicMissing, "static.tier_assessment", "", "tier requires an established core chart and static axis", nil, nil)
	}
	if assessment.Level < 1 || assessment.Level > 9 {
		return baziViolationError(baziViolationMethodContract, "static.tier_assessment.level", "", "rated tier must be in the nine-level range", nil, nil)
	}
	if assessment.Status == "rated" && !facts.TierEvidenceComplete {
		return baziViolationError(baziViolationEvidenceTopicMissing, "static.tier_assessment.status", "", "a fully rated tier requires independent qingzhuo, disease-remedy, rescue, break-risk and He Zhi Zhang evidence", facts.TierEvidenceMissing, baziTierEvidenceTopics)
	}
	bounds := baziTierBoundsFor(facts, axisStatus, assessment)
	if assessment.Level < bounds.Min || assessment.Level > bounds.Max {
		return baziViolationError(
			baziViolationMethodContract,
			"static.tier_assessment.level",
			"",
			fmt.Sprintf("tier level %d is outside the evidenced band %d-%d", assessment.Level, bounds.Min, bounds.Max),
			nil,
			[]string{fmt.Sprintf("%d-%d", bounds.Min, bounds.Max)},
		)
	}
	return nil
}

// baziTierBoundsFor supplies ceilings and narrow floors from the selected
// typed dimensions. It never turns facts into a concrete level by itself.
func baziTierBoundsFor(facts BaziFactCapsule, axisStatus string, assessment baziTierAssessment) baziTierBounds {
	bounds := baziTierBounds{Min: 1, Max: 9}
	if !facts.CoreFactsReady || axisStatus == "withheld" {
		return baziTierBounds{Min: 0, Max: 0}
	}
	if axisStatus == "candidate" {
		bounds.Max = minInt(bounds.Max, 6)
	}
	for _, dimension := range baziTierDimensionEntries(assessment.Dimensions) {
		bounds.Max = minInt(bounds.Max, tierDimensionCap(dimension))
	}
	if assessment.Status == "provisional" {
		bounds.Max = minInt(bounds.Max, 6)
	}
	if tierCanSupportUpperBand(assessment.Dimensions) && assessment.Status == "rated" {
		bounds.Min = 7
	}
	return bounds
}

// tierDimensionCap makes core axes stricter than adjustment lenses. In
// particular, unresolved 调候 alone cannot collapse an otherwise usable chart.
func tierDimensionCap(dimension baziNamedTierDimension) int {
	if dimension.Disease {
		switch dimension.Value.State {
		case "light":
			return 9
		case "moderate":
			return 7
		case "heavy":
			return 5
		case "critical":
			return 3
		default:
			return 6
		}
	}
	if dimension.Name == "main_axis" {
		switch dimension.Value.State {
		case "strong":
			return 9
		case "usable":
			return 8
		case "mixed":
			return 6
		case "limited":
			return 5
		default:
			return 2
		}
	}
	if dimension.Name == "tiaohou" || dimension.Name == "hezhizhang" {
		switch dimension.Value.State {
		case "strong", "usable":
			return 9
		case "mixed":
			return 8
		case "limited":
			return 7
		default:
			return 6
		}
	}
	switch dimension.Value.State {
	case "strong":
		return 9
	case "usable":
		return 8
	case "mixed":
		return 7
	case "limited":
		return 6
	default:
		return 4
	}
}

// tierCanSupportUpperBand prevents a top-half label from contradicting an
// otherwise uniformly strong set of model-selected dimension states.
func tierCanSupportUpperBand(dimensions baziTierDimensions) bool {
	for _, dimension := range baziTierDimensionEntries(dimensions) {
		if dimension.Disease {
			if dimension.Value.State != "light" {
				return false
			}
			continue
		}
		if dimension.Value.State != "usable" && dimension.Value.State != "strong" {
			return false
		}
	}
	return true
}

// baziTierDimensionEntries keeps validation, bounds and reference catalog
// projection on the same fixed nine-dimension order.
func baziTierDimensionEntries(dimensions baziTierDimensions) []baziNamedTierDimension {
	return []baziNamedTierDimension{
		{Name: "main_axis", Value: dimensions.MainAxis},
		{Name: "youqing", Value: dimensions.YouQing},
		{Name: "youli", Value: dimensions.YouLi},
		{Name: "qingzhuo", Value: dimensions.QingZhuo},
		{Name: "disease", Value: dimensions.Disease, Disease: true},
		{Name: "remedy", Value: dimensions.Remedy},
		{Name: "rescue", Value: dimensions.Rescue},
		{Name: "tiaohou", Value: dimensions.Tiaohou},
		{Name: "hezhizhang", Value: dimensions.HeZhiZhang},
	}
}

// tierDimensionHasGround prevents status-only tier scoring while allowing a
// fact-only dimension when no selected rule profile exists for this chart.
func tierDimensionHasGround(dimension baziTierDimension) bool {
	return len(dimension.FactRefs) > 0 || len(dimension.ClaimRefs) > 0 || len(dimension.EvidenceTopics) > 0
}

// tierDimensionAssertions adapts typed tier dimensions to the shared catalog
// checker. Their verdict is runtime-private state, never user-facing prose.
func tierDimensionAssertions(assessment baziTierAssessment) []baziAssertion {
	assertions := make([]baziAssertion, 0, 9)
	for _, dimension := range baziTierDimensionEntries(assessment.Dimensions) {
		assertions = append(assertions, baziAssertion{
			ID:             "static.tier." + dimension.Name,
			Kind:           baziAssertionTier,
			Subject:        "chart",
			Verdict:        strings.TrimSpace(dimension.Value.State),
			FactRefs:       append([]baziFactRef{}, dimension.Value.FactRefs...),
			ClaimRefs:      append([]baziClaimRef{}, dimension.Value.ClaimRefs...),
			EvidenceTopics: append([]string{}, dimension.Value.EvidenceTopics...),
			Confidence:     assessment.Confidence,
			Boundary:       "层次维度仅作状态投影。",
		})
	}
	return assertions
}

// validateBaziDynamicJudgmentPolicy binds every dynamic model claim to the runtime-selected period.
func validateBaziDynamicJudgmentPolicy(state baziCharterState, judgment baziStructuredDynamicSynthesis) error {
	facts := buildBaziFactCapsule(state)
	if facts.CurrentPeriodRef == "" {
		return baziViolationError(baziViolationMethodContract, "dynamic.current_period_ref", "", "runtime cannot bind a current dayun", nil, nil)
	}
	if judgment.CurrentPeriodRef != facts.CurrentPeriodRef {
		return baziViolationError(baziViolationFactConflict, "dynamic.current_period_ref", "", fmt.Sprintf("dynamic current period must equal %s", facts.CurrentPeriodRef), []string{judgment.CurrentPeriodRef}, []string{facts.CurrentPeriodRef})
	}
	if len(judgment.PeriodClaims) != 1 || judgment.PeriodClaims[0].PeriodRef != facts.CurrentPeriodRef {
		return baziViolationError(baziViolationMethodContract, "dynamic.period_claims", "", "dynamic judgment must contain one claim for the current period", nil, []string{facts.CurrentPeriodRef})
	}
	if !containsString(baziCurrentPeriodRealizationValues, judgment.CurrentPeriodRealization) {
		return baziViolationError(baziViolationMethodContract, "dynamic.current_period_realization", "", "current period realization is outside the closed contract", nil, baziCurrentPeriodRealizationValues)
	}
	if err := validateBaziModelTextSlots(state, dynamicJudgmentTextSlots(judgment), true, false); err != nil {
		return err
	}
	return nil
}

// baziModelTextSlot is a model-authored prose field that is still required for
// a bounded explanation. The semantic policy validates it before projection;
// the renderer never receives rejected prose.
type baziModelTextSlot struct {
	Field       string
	AssertionID string
	Value       string
}

// staticJudgmentTextSlots lists the remaining static model prose. Boundaries,
// limitations and reasoning are runtime-owned fact-capsule projections.
func staticJudgmentTextSlots(judgment baziStructuredStaticSynthesis) []baziModelTextSlot {
	return nil
}

// dynamicJudgmentTextSlots lists dynamic prose fields. Tier language is
// rejected here because dynamic judgment may not rewrite the natal grade.
func dynamicJudgmentTextSlots(judgment baziStructuredDynamicSynthesis) []baziModelTextSlot {
	slots := make([]baziModelTextSlot, 0, len(judgment.PeriodClaims)+len(judgment.Limitations)+len(judgment.ReasoningSteps)+2)
	for index, claim := range judgment.PeriodClaims {
		slots = append(slots,
			baziModelTextSlot{Field: fmt.Sprintf("dynamic.period_claims[%d].verdict", index), AssertionID: "dynamic.period", Value: claim.Verdict},
		)
	}
	slots = append(slots,
		baziModelTextSlot{Field: "dynamic.liunian_claim.verdict", AssertionID: "dynamic.liunian", Value: judgment.LiunianClaim.Verdict},
		baziModelTextSlot{Field: "dynamic.reasoning_summary", Value: judgment.ReasoningSummary},
	)
	for index, text := range judgment.Limitations {
		slots = append(slots, baziModelTextSlot{Field: fmt.Sprintf("dynamic.limitations[%d]", index), Value: text})
	}
	for index, text := range judgment.ReasoningSteps {
		slots = append(slots, baziModelTextSlot{Field: fmt.Sprintf("dynamic.reasoning_steps[%d]", index), Value: text})
	}
	return slots
}

// validateBaziModelTextSlots keeps machine identifiers and tier ownership out
// of prose at the DTO boundary. It is catalog-derived, not a final-text filter.
func validateBaziModelTextSlots(state baziCharterState, slots []baziModelTextSlot, forbidTier, forbidNatalOfficerRisk bool) error {
	for _, slot := range slots {
		if token := baziPresentationReferenceToken(state, slot.Value); token != "" {
			return baziViolationError(baziViolationMethodContract, slot.Field, slot.AssertionID, "model prose must keep runtime identifiers in typed reference arrays", []string{token}, []string{"fact_refs", "relation_refs", "claim_refs"})
		}
		if forbidTier && containsAnyText([]string{slot.Value}, baziTierProseMarkers) {
			return baziViolationError(baziViolationMethodContract, slot.Field, slot.AssertionID, "tier language belongs only in tier_assessment", nil, []string{"tier_assessment"})
		}
		_ = forbidNatalOfficerRisk
	}
	return nil
}

// baziPresentationReferenceToken derives prohibited prose tokens from the
// same per-turn catalog used for fact validation. New catalog fields therefore
// cannot silently become user-visible engineering labels.
func baziPresentationReferenceToken(state baziCharterState, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	catalog := buildBaziRuntimeCatalog(state)
	seen := map[string]struct{}{}
	tokens := []string{}
	add := func(token string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		if _, ok := seen[token]; ok {
			return
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	for _, refs := range []map[string]struct{}{catalog.Facts, catalog.Claims, catalog.Relations} {
		for ref := range refs {
			add(ref)
			if dot := strings.LastIndex(ref, "."); dot >= 0 {
				if terminal := ref[dot+1:]; strings.Contains(terminal, "_") {
					add(terminal)
				}
			}
		}
	}
	for _, ref := range baziDynamicPeriodRefs(state) {
		add(ref)
	}
	sort.Slice(tokens, func(left, right int) bool {
		if len(tokens[left]) == len(tokens[right]) {
			return tokens[left] < tokens[right]
		}
		return len(tokens[left]) > len(tokens[right])
	})
	for _, token := range tokens {
		if strings.Contains(text, token) {
			return token
		}
	}
	return ""
}
