// Package runtime 包含 Manager 拥有的八字运行时合同。
//
// 本文件校验已投影 assertion 的事实引用和兼容展示边界；
// 不负责模型调用、命理裁断或最终 Markdown 渲染。
package runtime

import (
	"fmt"
	"sort"
	"strings"
)

const (
	baziEvidenceSupported = "supported"
	baziEvidenceWithheld  = "withheld_missing_evidence"
)

var requiredPatternComparisonDimensions = []string{
	"visibility",
	"hidden_stem_tier",
	"root_support",
	"season_support",
	"structural_closure",
	"counter_evidence",
}

func ensureStaticAssertions(state baziCharterState, in baziStaticSynthesis) baziStaticSynthesis {
	if len(in.Assertions) > 0 {
		return canonicalizeStaticAssertionEvidence(state, in)
	}
	profile := state.Input.RuleProfile
	claim := firstClaimRefByCategory(profile, "main_axis", "pattern_candidate")
	strengthClaim := firstClaimRefByCategory(profile, "strength")
	tiaohouClaim := firstVerdictRefByPrefix(profile, "qiongtong_")
	tierClaim := firstClaimRefByCategory(profile, "tier")
	in.Assertions = []baziAssertion{
		{
			ID:             "static.main_axis",
			Kind:           baziAssertionMainAxis,
			Subject:        "chart",
			Verdict:        firstNonEmptyTrim(in.MainAxis, in.PatternOutcome),
			FactRefs:       []baziFactRef{"chart.month_branch", "yongshen.geju_candidate"},
			ClaimRefs:      claim,
			EvidenceTopics: []string{"geju"},
			EvidenceStatus: evidenceStatusForTopics(state.EvidenceQuality, []string{"geju"}),
			Confidence:     in.ClaimStrength,
			Boundary:       firstNonEmptyTrim(in.CounterEvidence, profileBoundaryByRef(profile, claim)),
		},
		{
			ID:             "static.strength",
			Kind:           baziAssertionStrength,
			Subject:        "day_master",
			Verdict:        firstNonEmptyTrim(in.Strength.Conclusion, in.StrengthBalance),
			FactRefs:       []baziFactRef{"yongshen.strength", "yongshen.strength_evidence"},
			ClaimRefs:      strengthClaim,
			EvidenceStatus: baziEvidenceSupported,
			Confidence:     in.ClaimStrength,
			Boundary:       in.Strength.Boundary,
		},
		{
			ID:             "static.tiaohou",
			Kind:           baziAssertionTiaohou,
			Subject:        "chart",
			Verdict:        firstNonEmptyTrim(in.TiaohouAnchor, in.TiaohouConstraint),
			FactRefs:       []baziFactRef{"chart.day_gan", "chart.month_branch"},
			ClaimRefs:      tiaohouClaim,
			EvidenceTopics: []string{"tiaohou"},
			EvidenceStatus: evidenceStatusForTopics(state.EvidenceQuality, []string{"tiaohou"}),
			Confidence:     in.ClaimStrength,
			Boundary:       in.TiaohouConstraint,
		},
		{
			ID:             "static.tier",
			Kind:           baziAssertionTier,
			Subject:        "chart",
			Verdict:        in.TierJudgment,
			FactRefs:       []baziFactRef{"chart.month_branch", "yongshen.strength_evidence"},
			ClaimRefs:      tierClaim,
			EvidenceTopics: append([]string{}, state.EvidenceQuality.RequiredTopics...),
			EvidenceStatus: evidenceStatusForTopics(state.EvidenceQuality, state.EvidenceQuality.RequiredTopics),
			Confidence:     in.ClaimStrength,
			Boundary:       in.TierBasis,
		},
	}
	if strings.TrimSpace(in.TopicDirectAnswer) != "" {
		in.Assertions = append(in.Assertions, baziAssertion{
			ID:             "static.topic_answer",
			Kind:           baziAssertionTopicAnswer,
			Subject:        "question",
			Verdict:        in.TopicDirectAnswer,
			FactRefs:       []baziFactRef{"chart.month_branch"},
			ClaimRefs:      claim,
			EvidenceTopics: []string{"geju"},
			EvidenceStatus: evidenceStatusForTopics(state.EvidenceQuality, []string{"geju"}),
			Confidence:     in.ClaimStrength,
			Boundary:       firstNonEmptyTrim(in.TopicFocusAnswer, in.CounterEvidence),
		})
	}
	return canonicalizeStaticAssertionEvidence(state, in)
}

// canonicalizeStaticAssertionEvidence derives evidence status from the runtime
// retrieval audit instead of trusting the model to restate a deterministic
// coverage result. The model still owns the verdict text and cited topics; the
// runtime owns whether those topics were actually covered.
func canonicalizeStaticAssertionEvidence(state baziCharterState, in baziStaticSynthesis) baziStaticSynthesis {
	for i := range in.Assertions {
		required := requiredTopicsForStaticAssertion(in.Assertions[i].Kind, state.EvidenceQuality.RequiredTopics)
		if len(required) == 0 {
			continue
		}
		in.Assertions[i].EvidenceTopics = mergeEvidenceTopics(in.Assertions[i].EvidenceTopics, required)
		in.Assertions[i].EvidenceStatus = evidenceStatusForTopics(state.EvidenceQuality, required)
	}
	return in
}

func mergeEvidenceTopics(existing, required []string) []string {
	merged := make([]string, 0, len(existing)+len(required))
	for _, topic := range append(append([]string{}, existing...), required...) {
		topic = strings.TrimSpace(topic)
		if topic != "" && !containsString(merged, topic) {
			merged = append(merged, topic)
		}
	}
	return merged
}

func ensureDynamicAssertions(state baziCharterState, in baziDynamicSynthesis) baziDynamicSynthesis {
	if len(in.Assertions) > 0 {
		return in
	}
	dynamicClaim := firstClaimRefByCategory(state.Input.RuleProfile, "dynamic_framework")
	periods := dayunPeriods(state.Input.Dayun)
	if len(in.DayunJudgments) > 0 {
		for _, judgment := range in.DayunJudgments {
			index, ok := dayunIndexByGanZhi(periods, judgment.GanZhi)
			if !ok {
				continue
			}
			in.Assertions = append(in.Assertions, dayunAssertionFromParts(index, judgment.GanZhi, judgment.Trend, judgment.Interpretation, dynamicClaim))
		}
	}
	if strings.TrimSpace(in.LiunianFocus) != "" {
		currentIndex := currentDayunIndexForInput(state.Input)
		factRefs := []baziFactRef{"liunian.gan_zhi", "liunian.relations"}
		if currentIndex >= 0 {
			factRefs = append(factRefs,
				baziFactRef(fmt.Sprintf("dayun[%d].gan_zhi", currentIndex)),
				baziFactRef(fmt.Sprintf("dayun[%d].relations", currentIndex)),
			)
		}
		in.Assertions = append(in.Assertions, baziAssertion{
			ID:         "dynamic.liunian",
			Kind:       baziAssertionLiunian,
			Subject:    "liunian",
			Verdict:    in.LiunianFocus,
			FactRefs:   factRefs,
			ClaimRefs:  dynamicClaim,
			Confidence: in.ClaimStrength,
			Boundary:   firstNonEmptyTrim(strings.Join(in.Risks, "；"), "具体应事不作展开。"),
		})
	}
	return in
}

// dayunIndexByGanZhi resolves a model judgment to the calculated period it
// actually names. DayunJudgments is sparse, so its slice position must never
// be treated as a full-catalog index.
func dayunIndexByGanZhi(periods []map[string]any, ganZhi string) (int, bool) {
	ganZhi = strings.TrimSpace(ganZhi)
	if ganZhi == "" {
		return 0, false
	}
	for index, period := range periods {
		if strings.TrimSpace(stringValue(period["ganZhi"])) == ganZhi {
			return index, true
		}
	}
	return 0, false
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
	if err := validateMainAxisAssertionConsistency(static); err != nil {
		return err
	}
	if err := validatePatternAdjudication(state, static.PatternAdjudication); err != nil {
		return err
	}
	if err := validateStaticAssertionEvidenceTopics(state, static.Assertions); err != nil {
		return err
	}
	if err := validateStaticTierWithheldBoundary(state, static); err != nil {
		return err
	}
	return validateBaziAssertions(state, static.Assertions)
}

// validatePatternAdjudication enforces a complete comparison process without
// selecting a chart-specific pattern on behalf of the synthesis model.
func validatePatternAdjudication(state baziCharterState, adjudication baziPatternAdjudication) error {
	if strings.TrimSpace(stringValue(state.Input.Yongshen["geju_candidate"])) == "" {
		return nil
	}
	if strings.TrimSpace(adjudication.MonthCommandCandidateID) == "" || len(adjudication.Candidates) == 0 {
		return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication", "", "pattern adjudication must include the month-command candidate", nil, nil)
	}

	candidates := make(map[string]baziPatternCandidate, len(adjudication.Candidates))
	for _, candidate := range adjudication.Candidates {
		id := strings.TrimSpace(candidate.ID)
		if id == "" || candidates[id].ID != "" {
			return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.candidates", id, "pattern candidate ids must be non-empty and unique", nil, nil)
		}
		if !containsString([]string{"month_command", "visible_stem", "hidden_combination", "transformation", "other_evidence_backed"}, strings.TrimSpace(candidate.Origin)) {
			return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.candidates.origin", id, "pattern candidate origin is outside the closed set", nil, nil)
		}
		if !containsString([]string{"pattern_foundation", "selected_axis", "selected_usage", "secondary_signal", "rejected"}, strings.TrimSpace(candidate.Role)) {
			return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.candidates.role", id, "pattern candidate role is outside the closed set", nil, nil)
		}
		candidates[id] = candidate
	}

	month, ok := candidates[strings.TrimSpace(adjudication.MonthCommandCandidateID)]
	if !ok || month.Origin != "month_command" {
		return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.month_command_candidate_id", "", "month-command candidate id must reference a month_command candidate", nil, nil)
	}
	toolCandidate := normalizePatternCandidateName(stringValue(state.Input.Yongshen["geju_candidate"]))
	modelCandidate := normalizePatternCandidateName(month.Name)
	if toolCandidate != "" && modelCandidate != "" && !strings.Contains(toolCandidate, modelCandidate) && !strings.Contains(modelCandidate, toolCandidate) {
		return baziViolationError(baziViolationFactConflict, "static.pattern_adjudication.month_command_candidate", month.ID, "month-command candidate conflicts with deterministic tool candidate", nil, nil)
	}
	if month.Role == "rejected" {
		if len(month.RejectionReasons) == 0 || onlyContainsString(month.RejectionReasons, "not_transparent") {
			return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.month_command_rejection", month.ID, "month-command candidate cannot be rejected solely because its main qi is not transparent", nil, nil)
		}
		if err := requirePatternComparisonDimensions(month); err != nil {
			return err
		}
	}

	if len(adjudication.SelectedAxisCandidateIDs) == 0 {
		return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.selected_axis_candidate_ids", "", "pattern adjudication must identify the selected axis candidates", nil, nil)
	}
	for _, selectedID := range adjudication.SelectedAxisCandidateIDs {
		candidate, ok := candidates[strings.TrimSpace(selectedID)]
		if !ok {
			return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.selected_axis_candidate_ids", selectedID, "selected axis references an unknown candidate", nil, nil)
		}
		if candidate.Role != "selected_axis" && candidate.Role != "selected_usage" && candidate.Role != "pattern_foundation" {
			return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.candidates.role", candidate.ID, "selected axis candidate has a non-selected role", nil, nil)
		}
		if err := requirePatternComparisonDimensions(candidate); err != nil {
			return err
		}
		if candidate.Origin == "hidden_combination" {
			if !containsString(state.EvidenceQuality.CoveredTopics, "geju") {
				return baziViolationError(baziViolationEvidenceTopicMissing, "static.pattern_adjudication.candidates.evidence_topics", candidate.ID, "hidden combination cannot lead without covered geju authority evidence", []string{"geju"}, state.EvidenceQuality.CoveredTopics)
			}
		}
	}
	return nil
}

// requirePatternComparisonDimensions requires every comparison axis promised
// by the methodology contract before a route can displace stronger visibility.
func requirePatternComparisonDimensions(candidate baziPatternCandidate) error {
	missing := make([]string, 0)
	completed := comparisonDimensionNames(candidate.ComparisonDimensions)
	for _, dimension := range requiredPatternComparisonDimensions {
		if !containsString(completed, dimension) {
			missing = append(missing, dimension)
		}
	}
	if len(missing) > 0 {
		return baziViolationError(baziViolationMethodContract, "static.pattern_adjudication.candidates.comparison_dimensions", candidate.ID, "pattern candidate lacks the required comparison dimensions", missing, requiredPatternComparisonDimensions)
	}
	return nil
}

// comparisonDimensionNames normalizes the two JSON shapes accepted from the
// model. The contract cares that each named comparison was performed, not
// whether the model encoded its explanation as an array or keyed object.
func comparisonDimensionNames(raw any) []string {
	seen := make(map[string]bool)
	appendName := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	switch value := raw.(type) {
	case []string:
		for _, item := range value {
			appendName(item)
		}
	case []any:
		for _, item := range value {
			if name, ok := item.(string); ok {
				appendName(name)
			}
		}
	case map[string]any:
		for name := range value {
			appendName(name)
		}
	case map[string]string:
		for name := range value {
			appendName(name)
		}
	}
	result := make([]string, 0, len(seen))
	for _, dimension := range requiredPatternComparisonDimensions {
		if seen[dimension] {
			result = append(result, dimension)
		}
	}
	return result
}

// validateStaticAssertionEvidenceTopics binds each static judgment to the
// authority topic that can support it and requires withholding on missing topics.
func validateStaticAssertionEvidenceTopics(state baziCharterState, assertions []baziAssertion) error {
	if len(state.EvidenceQuality.RequiredTopics) == 0 {
		return nil
	}
	for _, assertion := range assertions {
		required := requiredTopicsForStaticAssertion(assertion.Kind, state.EvidenceQuality.RequiredTopics)
		if len(required) == 0 {
			continue
		}
		missingDeclarations := make([]string, 0)
		for _, topic := range required {
			if !containsString(assertion.EvidenceTopics, topic) {
				missingDeclarations = append(missingDeclarations, topic)
			}
		}
		if len(missingDeclarations) > 0 {
			return baziViolationError(baziViolationEvidenceTopicMissing, "static.assertions.evidence_topics", assertion.ID, "static assertion omits required authority topics", missingDeclarations, required)
		}
		wantStatus := evidenceStatusForTopics(state.EvidenceQuality, required)
		if strings.TrimSpace(assertion.EvidenceStatus) != wantStatus {
			return baziViolationError(baziViolationEvidenceTopicMissing, "static.assertions.evidence_status", assertion.ID, "static assertion evidence status does not match topic coverage", []string{wantStatus}, []string{baziEvidenceSupported, baziEvidenceWithheld})
		}
	}
	return nil
}

// validateStaticTierWithheldBoundary keeps typed V2 state separate from the
// legacy text-only compatibility path.
func validateStaticTierWithheldBoundary(state baziCharterState, static baziStaticSynthesis) error {
	if static.TierAssessment.Status != "" {
		return validateTypedTierPresentation(static.TierAssessment)
	}
	if len(state.EvidenceQuality.RequiredTopics) == 0 {
		return nil
	}
	for _, assertion := range static.Assertions {
		if assertion.Kind != baziAssertionTier || assertion.EvidenceStatus != baziEvidenceWithheld {
			continue
		}
		checks := []struct {
			field string
			text  string
		}{
			{field: "static.tier_judgment", text: static.TierJudgment},
			{field: "static.tier_basis", text: static.TierBasis},
			{field: "static.assertions.tier.verdict", text: assertion.Verdict},
			{field: "static.assertions.tier.boundary", text: assertion.Boundary},
		}
		for _, check := range checks {
			if !withheldTierTextIsSafe(check.text) {
				return baziViolationError(
					baziViolationEvidenceTopicMissing,
					check.field,
					assertion.ID,
					"tier assertion is withheld_missing_evidence and must defer ranking",
					state.EvidenceQuality.MissingTopics,
					[]string{"命格层次暂不定级", "不作高低定级"},
				)
			}
		}
	}
	return nil
}

// validateTypedTierPresentation verifies typed compatibility state without
// scanning renderer text or recalculating a chart grade.
func validateTypedTierPresentation(assessment baziTierAssessment) error {
	if assessment.Status == "withheld" {
		if assessment.Level != 0 {
			return baziViolationError(baziViolationMethodContract, "static.tier_assessment", "", "withheld tier must render as a non-ranked boundary", nil, nil)
		}
		return nil
	}
	if assessment.Level < 1 || assessment.Level > 9 {
		return baziViolationError(baziViolationMethodContract, "static.tier_assessment", "", "tier projection must retain the selected nine-level value", nil, nil)
	}
	if assessment.Status == "provisional" && (assessment.Level < 3 || assessment.Level > 6) {
		return baziViolationError(baziViolationMethodContract, "static.tier_assessment", "", "provisional tier must remain in the 3-6 band and show its boundary", nil, nil)
	}
	return nil
}

// withheldTierTextIsSafe accepts only an explicit deferred rank when required topics are missing.
func withheldTierTextIsSafe(text string) bool {
	text = strings.TrimSpace(text)
	return strings.Contains(text, "暂不定级") || strings.Contains(text, "不作高低定级")
}

// containsHighTierAssertion detects positive high-rank language while allowing
// explicit cap wording such as “不上推中上或上等”.
func containsHighTierAssertion(text string) bool {
	positivePatterns := []string{
		"命格层次上等", "命格层次中上", "命格层次中等偏上",
		"层次上等", "层次中上", "层次中等偏上",
		"可以拔高", "可拔高", "足以拔高",
	}
	if containsAnyText([]string{text}, positivePatterns) {
		return true
	}
	highMarkers := []string{"上等", "中上", "中等偏上", "可以拔高", "可拔高", "拔高"}
	capMarkers := []string{"不上推", "不拔高", "不能拔高", "不宜拔高", "不足以拔高", "不进入", "不升至", "封顶"}
	for _, marker := range highMarkers {
		if strings.Contains(text, marker) && !containsAnyText([]string{text}, capMarkers) {
			return true
		}
	}
	return false
}

// requiredTopicsForStaticAssertion maps methodology claims to authority topics;
// tier judgments consume every A-tier topic because they synthesize all lenses.
func requiredTopicsForStaticAssertion(kind baziAssertionKind, requiredTopics []string) []string {
	switch kind {
	case baziAssertionMainAxis, baziAssertionPatternUsage, baziAssertionTopicAnswer:
		if containsString(requiredTopics, "geju") {
			return []string{"geju"}
		}
	case baziAssertionTiaohou:
		if containsString(requiredTopics, "tiaohou") {
			return []string{"tiaohou"}
		}
	}
	return nil
}

// evidenceStatusForTopics returns withheld when any referenced authority topic
// is missing, otherwise supported.
func evidenceStatusForTopics(quality baziEvidenceQuality, topics []string) string {
	for _, topic := range topics {
		if containsString(quality.MissingTopics, topic) || !containsString(quality.CoveredTopics, topic) {
			return baziEvidenceWithheld
		}
	}
	return baziEvidenceSupported
}

// normalizePatternCandidateName removes transport suffixes before comparing a
// model candidate with the deterministic month-command candidate.
func normalizePatternCandidateName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	value = strings.TrimSuffix(value, "候选")
	if index := strings.IndexAny(value, "(（"); index >= 0 {
		value = value[:index]
	}
	return value
}

// onlyContainsString reports whether a non-empty list consists solely of one
// normalized enum value.
func onlyContainsString(items []string, want string) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item) != want {
			return false
		}
	}
	return true
}

// validateMainAxisAssertionConsistency requires one machine-verifiable main
// axis and a non-empty legacy projection. It deliberately does not compare
// Chinese wording: semantic agreement belongs to the independent audit, while
// substring matching rejects harmless paraphrases and turns the renderer's
// compatibility field into a second semantic judge.
func validateMainAxisAssertionConsistency(static baziStaticSynthesis) error {
	var mainAxisAssertions []baziAssertion
	for _, assertion := range static.Assertions {
		if assertion.Kind == baziAssertionMainAxis {
			mainAxisAssertions = append(mainAxisAssertions, assertion)
		}
	}
	if len(mainAxisAssertions) != 1 {
		return baziViolationError(baziViolationScopeEscalation, "static.assertions", "", fmt.Sprintf("static synthesis requires exactly one main_axis assertion, got %d", len(mainAxisAssertions)), nil, nil)
	}
	field := canonicalJudgmentText(static.MainAxis)
	verdict := canonicalJudgmentText(mainAxisAssertions[0].Verdict)
	if field == "" || verdict == "" {
		return baziViolationError(baziViolationScopeEscalation, "static.main_axis", mainAxisAssertions[0].ID, "main_axis field and assertion verdict are both required", nil, nil)
	}
	return nil
}

// canonicalJudgmentText removes presentation punctuation before comparing two
// projections of the same structured judgment.
func canonicalJudgmentText(value string) string {
	replacer := strings.NewReplacer(
		" ", "", "\t", "", "\n", "", "，", "", "。", "",
		"；", "", "：", "", ",", "", ".", "", ";", "", ":", "",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

func validateDynamicAssertions(state baziCharterState) error {
	dynamic := ensureDynamicAssertions(state, projectDynamicAssertionsToLegacy(state.DynamicSynthesis))
	if err := validateDayunAssertionsAgainstFacts(state, dynamic.Assertions); err != nil {
		return err
	}
	if err := validateLiunianAssertionsAgainstCurrentDayun(state, dynamic.Assertions); err != nil {
		return err
	}
	return validateBaziAssertions(state, dynamic.Assertions)
}

// validateLiunianAssertionsAgainstCurrentDayun binds annual interpretation to the runtime-selected luck period.
func validateLiunianAssertionsAgainstCurrentDayun(state baziCharterState, assertions []baziAssertion) error {
	currentIndex := currentDayunIndexForInput(state.Input)
	if currentIndex < 0 {
		return nil
	}
	wantGanZhi := fmt.Sprintf("dayun[%d].gan_zhi", currentIndex)
	wantRelations := fmt.Sprintf("dayun[%d].relations", currentIndex)
	for _, assertion := range assertions {
		if assertion.Kind != baziAssertionLiunian {
			continue
		}
		for _, ref := range assertion.FactRefs {
			if string(ref) == wantGanZhi || string(ref) == wantRelations {
				goto next
			}
		}
		return baziViolationError(baziViolationFactRefMissing, "assertions.fact_refs", assertion.ID, "liunian assertion must reference the runtime-selected current period", []string{fmt.Sprintf("dayun[%d]", currentIndex)}, nil)
	next:
	}
	return nil
}

func validateBaziAssertions(state baziCharterState, assertions []baziAssertion) error {
	if err := validateBaziReferenceCatalog(state, assertions); err != nil {
		return err
	}
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
		if token := baziPresentationReferenceToken(state, assertion.Verdict); token != "" {
			return baziViolationError(baziViolationMethodContract, "assertions.presentation", assertion.ID, "user-visible assertion text must not expose runtime reference paths", []string{token}, []string{"fact_refs", "relation_refs", "claim_refs"})
		}
		if token := baziPresentationReferenceToken(state, assertion.Boundary); token != "" {
			return baziViolationError(baziViolationMethodContract, "assertions.presentation", assertion.ID, "user-visible assertion text must not expose runtime reference paths", []string{token}, []string{"fact_refs", "relation_refs", "claim_refs"})
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
