// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件向八字用例层提供窄的确定性领域辅助函数；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

import "strings"

// BuildFactsOnlyDynamicSynthesis builds the deterministic dynamic fallback.
func BuildFactsOnlyDynamicSynthesis(input CharterInput, static StaticSynthesis, reason string) DynamicSynthesis {
	return buildFactsOnlyDynamicSynthesis(input, static, reason)
}

// BuildFactsOnlyStaticSynthesis builds the deterministic static fallback.
func BuildFactsOnlyStaticSynthesis(input CharterInput, reason string) StaticSynthesis {
	return buildFactsOnlyStaticSynthesis(input, reason)
}

// CurrentDayunIndexForInput returns the current calculated period index.
func CurrentDayunIndexForInput(input CharterInput) int { return currentDayunIndexForInput(input) }

// PeriodHeadline extracts a stable display headline.
func PeriodHeadline(line string) string { return periodHeadline(line) }

// ViolationFromError extracts a validation violation from an error.
func ViolationFromError(err error) (ValidationViolation, bool) { return baziViolationFromError(err) }

// HasUnsupportedConcreteOutcome reports whether text crosses the deterministic outcome boundary.
func HasUnsupportedConcreteOutcome(text string) bool { return containsUnsupportedConcreteOutcome(text) }

// StaticPatternFactSummary summarizes deterministic pattern facts.
func StaticPatternFactSummary(input CharterInput) string { return staticPatternFactSummary(input) }

// StrengthEvidenceSummary returns the deterministic strength evidence wording.
func StrengthEvidenceSummary(yongshen map[string]any) string {
	return strengthEvidenceSummary(yongshen)
}

// EnsureDynamicAssertions fills deterministic assertion references for a dynamic synthesis.
func EnsureDynamicAssertions(state CharterState, synthesis DynamicSynthesis) DynamicSynthesis {
	return ensureDynamicAssertions(state, synthesis)
}

// StringValue returns a raw string fact without coercion.
func StringValue(raw any) string { return stringValue(raw) }

// RelationTextList returns user-readable descriptions of calculated relations.
func RelationTextList(raw any) []string { return relationTextList(raw) }

// MapValue returns a nested calculated object without coercion.
func MapValue(src map[string]any, key string) map[string]any { return mapValue(src, key) }

// PillarFactSummary returns the deterministic four-pillars summary.
func PillarFactSummary(raw any) string { return pillarFactSummary(raw) }

// FirstUnauthorizedMinorOutcomeSignal identifies an age-scope violation.
func FirstUnauthorizedMinorOutcomeSignal(text string) (string, string) {
	return firstUnauthorizedMinorOutcomeSignal(text)
}

// TierAssessmentJudgment 返回面向用户的格局评价状态，不暴露内部证据等级。
func TierAssessmentJudgment(assessment TierAssessment) string {
	if assessment.Status == "withheld" {
		return "格局暂不立评（仅作结构观察）"
	}
	if assessment.Status == "rated" {
		return "格局评价已定"
	}
	return "格局判断暂定"
}

// TierAssessmentBasis returns the deterministic tier basis.
func TierAssessmentBasis(assessment TierAssessment) string {
	if assessment.Status == "withheld" {
		return "关键证据链尚未闭合，因此本轮只作结构观察。"
	}
	if assessment.Status == "provisional" {
		reasons := provisionalTierReasons(assessment.Dimensions)
		if len(reasons) == 0 {
			return "格局评价依据已验收的结构与限制维度综合确定；尚有维度待继续核对，本轮结论暂定。"
		}
		return "格局评价依据已验收的结构与限制维度综合确定；" + strings.Join(reasons, "、") + "，本轮结论暂定。"
	}
	return "格局评价依据已验收的结构、证据与限制维度综合确定。"
}

// provisionalTierReasons 把已验收的维度状态投影为暂定原因，不重新评估格局等级。
func provisionalTierReasons(dimensions TierDimensions) []string {
	reasons := make([]string, 0, 4)
	for _, states := range [][]string{{"missing", "unresolved"}, {"limited"}, {"mixed"}} {
		for _, dimension := range baziTierDimensionEntries(dimensions) {
			if len(reasons) == 4 {
				return reasons
			}
			if !containsString(states, dimension.Value.State) {
				continue
			}
			if reason := provisionalTierReason(dimension); reason != "" {
				reasons = append(reasons, reason)
			}
		}
	}
	return reasons
}

// provisionalTierReason 只翻译现有状态，避免展示层用自然语言补造证据。
func provisionalTierReason(dimension baziNamedTierDimension) string {
	if dimension.Disease && dimension.Value.State == "unresolved" {
		return "病药关系未明"
	}
	labels := map[string]map[string]string{
		"main_axis":  {"missing": "主轴证据缺位", "limited": "主轴承接受限", "mixed": "主轴条件并见"},
		"youqing":    {"missing": "有情条件缺位", "limited": "有情条件受限", "mixed": "有情条件并见"},
		"youli":      {"missing": "有力条件缺位", "limited": "有力条件受限", "mixed": "有力条件并见"},
		"qingzhuo":   {"missing": "清浊证据缺位", "limited": "清浊判断受限", "mixed": "清浊关系并见"},
		"remedy":     {"missing": "用药条件缺位", "limited": "用药条件受限", "mixed": "用药条件并见"},
		"rescue":     {"missing": "救应条件缺位", "limited": "救应条件受限", "mixed": "救应条件并见"},
		"tiaohou":    {"missing": "调候条件待核", "limited": "调候条件受限", "mixed": "调候条件并见"},
		"hezhizhang": {"missing": "何知章印证缺位", "limited": "何知章印证受限", "mixed": "何知章印证并见"},
	}
	return labels[dimension.Name][dimension.Value.State]
}

// ValidateStaticJudgment checks the static model DTO against deterministic facts.
func ValidateStaticJudgment(state CharterState, judgment StructuredStaticSynthesis) error {
	return validateBaziStaticJudgmentPolicy(state, judgment)
}

// ValidateDynamicJudgment checks the dynamic model DTO against deterministic facts.
func ValidateDynamicJudgment(state CharterState, judgment StructuredDynamicSynthesis) error {
	return validateBaziDynamicJudgmentPolicy(state, judgment)
}

// ValidateDynamicPreconditions checks whether calculated dynamic facts are available.
func ValidateDynamicPreconditions(state CharterState) error {
	return validateDynamicPreconditions(state)
}

// ValidateDynamicSynthesisAfterGraphNormalization rechecks normalized dynamic output at the Graph boundary.
func ValidateDynamicSynthesisAfterGraphNormalization(state CharterState) error {
	return validateDynamicSynthesisAfterGraphNormalization(state)
}

// TierDimensionAssertions returns the fixed assertions derived from tier dimensions.
func TierDimensionAssertions(assessment TierAssessment) []Assertion {
	return tierDimensionAssertions(assessment)
}

// ValidateStaticReferenceCatalog checks the static reference allow-list.
func ValidateStaticReferenceCatalog(state CharterState, assertions []Assertion) error {
	return validateStaticBaziReferenceCatalog(state, assertions)
}

// ValidateDynamicReferenceCatalog checks the dynamic reference allow-list.
func ValidateDynamicReferenceCatalog(state CharterState, assertions []Assertion) error {
	return validateDynamicBaziReferenceCatalog(state, assertions)
}

// StaticRuntimeCatalogView projects the static reference allow-list for a model input.
func StaticRuntimeCatalogView(state CharterState) map[string]any {
	return baziStaticRuntimeCatalogView(state)
}

// DynamicRuntimeCatalogView projects the dynamic reference allow-list for a model input.
func DynamicRuntimeCatalogView(state CharterState) map[string]any {
	return baziDynamicRuntimeCatalogView(state)
}

// FactCapsuleForState derives deterministic facts from a charter state.
func FactCapsuleForState(state CharterState) FactCapsule { return buildBaziFactCapsule(state) }

// DynamicPeriodRefs returns the runtime-selected dynamic period identifiers.
func DynamicPeriodRefs(state CharterState) []string { return baziDynamicPeriodRefs(state) }

// BuildFactCapsulePromptView projects deterministic facts into the bounded model input.
func BuildFactCapsulePromptView(state CharterState, includeDynamic bool) map[string]any {
	return buildBaziFactCapsulePromptView(state, includeDynamic)
}

// BuildRuntimeCatalog derives the current reference allow-list.
func BuildRuntimeCatalog(state CharterState) ReferenceCatalog { return buildBaziRuntimeCatalog(state) }

// LifetimeRuntimeCatalogView projects all calculated Dayun references for the lifetime synthesis input.
func LifetimeRuntimeCatalogView(state CharterState) map[string]any {
	view := baziRuntimeCatalogViewFor(state, buildBaziRuntimeCatalog(state))
	view["period_refs"] = baziDynamicPeriodRefs(state)
	return view
}

// IsFactsOnlyStaticSynthesis reports whether static output is the deterministic fallback.
func IsFactsOnlyStaticSynthesis(s StaticSynthesis) bool { return isFactsOnlyStaticSynthesis(s) }

// IsFactsOnlyDynamicSynthesis reports whether dynamic output is the deterministic fallback.
func IsFactsOnlyDynamicSynthesis(s DynamicSynthesis) bool { return isFactsOnlyDynamicSynthesis(s) }

// ContractAuditSummary returns the compact audit trace value.
func ContractAuditSummary(audit ContractAudit) string { return baziContractAuditSummary(audit) }

// ValidateEvidenceBundlePreconditions checks required authority coverage.
func ValidateEvidenceBundlePreconditions(state CharterState) error {
	return validateEvidenceBundlePreconditions(state)
}

// ValidateStaticSynthesisResult checks the accepted static projection.
func ValidateStaticSynthesisResult(state CharterState, synthesis StaticSynthesis) error {
	return validateStaticSynthesisResult(state, synthesis)
}

// FactsOnlySource identifies deterministic fallback output.
const FactsOnlySource = baziSynthesisSourceFactsOnlyDegraded

// KnownFactRefs returns the current deterministic fact allow-list.
func KnownFactRefs(state CharterState) map[string]struct{} { return knownBaziFactRefs(state) }

// KnownClaimRefs returns the selected rule-profile claim allow-list.
func KnownClaimRefs(profile RuleProfile) map[string]struct{} { return knownBaziClaimRefs(profile) }

// IsKnownFactRef reports whether a fact reference belongs to the current allow-list.
func IsKnownFactRef(ref FactRef, known map[string]struct{}) bool {
	return isKnownBaziFactRef(ref, known)
}

// ClaimRefAllowsAssertionKind checks the declared kind scope for a rule reference.
func ClaimRefAllowsAssertionKind(profile RuleProfile, ref string, kind AssertionKind) bool {
	return claimRefAllowsAssertionKind(profile, ref, kind)
}

// EvidenceSupported is the accepted evidence status.
const EvidenceSupported = baziEvidenceSupported

// EvidenceWithheld is the missing-evidence status.
const EvidenceWithheld = baziEvidenceWithheld
