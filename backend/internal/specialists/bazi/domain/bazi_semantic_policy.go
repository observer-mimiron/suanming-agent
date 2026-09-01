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

var baziCurrentPeriodRealizationValues = []string{"repair", "assist", "maintain", "disturb", "suppress"}
var baziStaticClaimSlots = []string{"main_axis", "strength", "tiaohou", "pattern_usage"}

// validateBaziStaticJudgmentPolicy enforces static eligibility before legacy projection.
func validateBaziStaticJudgmentPolicy(state baziCharterState, judgment baziStructuredStaticSynthesis) error {
	claims, err := NormalizeStaticClaims(judgment.Claims)
	if err != nil {
		return err
	}
	judgment.Claims = claims
	if !containsString([]string{"candidate", "established", "withheld"}, judgment.AxisStatus) {
		return baziViolationError(baziViolationMethodContract, "static.axis_status", "", "static axis status is outside the closed contract", nil, nil)
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
	if err := validateStaticPatternFields(state, judgment); err != nil {
		return err
	}
	return nil
}

// validateStaticPatternFields keeps displayed pattern fields inside the
// deterministic candidates for this chart without restoring the old tier gate.
func validateStaticPatternFields(state baziCharterState, judgment baziStructuredStaticSynthesis) error {
	name := normalizePatternCandidateName(judgment.PatternName)
	expectedName := normalizePatternCandidateName(stringValue(state.Input.Yongshen["geju_candidate"]))
	if expectedName != "" && name != "" && name != expectedName {
		return baziViolationError(baziViolationFactConflict, "static.pattern_name", "", "pattern name conflicts with deterministic month-command candidate", []string{"yongshen.geju_candidate"}, []string{expectedName})
	}
	if name == "" {
		return baziViolationError(baziViolationMethodContract, "static.pattern_name", "", "pattern name is required", nil, nil)
	}
	route := normalizePatternCandidateName(judgment.PatternRoute)
	if route == "" {
		return baziViolationError(baziViolationMethodContract, "static.pattern_route", "", "pattern route is required", nil, nil)
	}
	combination := stringValue(state.Input.Yongshen["geju_combination"])
	if combination == "" {
		return nil
	}
	for _, part := range strings.FieldsFunc(combination, func(r rune) bool {
		return r == '；' || r == ';' || r == '，' || r == ',' || r == '\n'
	}) {
		if normalizePatternCandidateName(part) == route {
			return nil
		}
	}
	return baziViolationError(baziViolationFactConflict, "static.pattern_route", "", "pattern route is outside deterministic chart candidates", []string{"yongshen.geju_combination"}, nil)
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
	if err := validateBaziModelTextSlots(state, dynamicJudgmentTextSlots(judgment)); err != nil {
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

// dynamicJudgmentTextSlots lists dynamic prose fields.
func dynamicJudgmentTextSlots(judgment baziStructuredDynamicSynthesis) []baziModelTextSlot {
	slots := make([]baziModelTextSlot, 0, len(judgment.PeriodClaims)+1)
	for index, claim := range judgment.PeriodClaims {
		slots = append(slots,
			baziModelTextSlot{Field: fmt.Sprintf("dynamic.period_claims[%d].verdict", index), AssertionID: "dynamic.period", Value: claim.Verdict},
		)
	}
	slots = append(slots,
		baziModelTextSlot{Field: "dynamic.liunian_claim.verdict", AssertionID: "dynamic.liunian", Value: judgment.LiunianClaim.Verdict},
	)
	return slots
}

// validateBaziModelTextSlots keeps machine identifiers out of prose at the DTO boundary.
func validateBaziModelTextSlots(state baziCharterState, slots []baziModelTextSlot) error {
	for _, slot := range slots {
		if token := baziPresentationReferenceToken(state, slot.Value); token != "" {
			return baziViolationError(baziViolationMethodContract, slot.Field, slot.AssertionID, "model prose must keep runtime identifiers in typed reference arrays", []string{token}, []string{"fact_refs", "relation_refs", "claim_refs"})
		}
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
