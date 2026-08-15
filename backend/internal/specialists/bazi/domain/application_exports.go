// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件向八字用例层提供窄的确定性领域辅助函数；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

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
		return "格局评价依据已验收的结构与限制维度综合确定；当前命盘结构仍有保留，本轮结论暂定。"
	}
	return "格局评价依据已验收的结构、证据与限制维度综合确定。"
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

// EvaluateEvidenceQuality checks retrieval coverage for the active evidence plan.
func EvaluateEvidenceQuality(plan EvidencePlan, bundle EvidenceBundle) EvidenceQuality {
	return evaluateEvidenceBundleQuality(plan, bundle)
}

// EvidenceSupported is the accepted evidence status.
const EvidenceSupported = baziEvidenceSupported

// EvidenceWithheld is the missing-evidence status.
const EvidenceWithheld = baziEvidenceWithheld
