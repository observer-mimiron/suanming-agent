// Package domain 定义八字领域的输入、结果与断言类型。
//
// 本文件定义确定性事实、证据引用与受限模型裁断 DTO；
// renderer 只能消费投影结果，不得自行新增命理结论。
package domain

type baziCharterInput = CharterInput

// 以下别名保留 runtime 内未导出调用点；八字业务值对象由 domain 所有。
type baziRuleProfile = RuleProfile
type baziProfileClaim = ProfileClaim
type baziRuleProfileOverlay = RuleProfileOverlay
type baziProfileRuleVerdict = ProfileRuleVerdict
type baziCitation = Citation
type baziAssertionKind = AssertionKind
type baziFactRef = FactRef
type baziClaimRef = ClaimRef
type baziRelationRef = RelationRef
type baziAssertion = Assertion

const (
	baziAssertionMainAxis     = AssertionMainAxis
	baziAssertionStrength     = AssertionStrength
	baziAssertionTiaohou      = AssertionTiaohou
	baziAssertionPatternUsage = AssertionPatternUsage
	baziAssertionTier         = AssertionTier
	baziAssertionDayunPeriod  = AssertionDayunPeriod
	baziAssertionLiunian      = AssertionLiunian
	baziAssertionTopicAnswer  = AssertionTopicAnswer
)

// 以下别名保持 runtime 私有调用点稳定；结构化模型合同由八字领域拥有。
type baziStructuredClaim = StructuredClaim
type baziStructuredStaticClaim = StructuredStaticClaim
type baziTierDimension = TierDimension
type baziTierDimensions = TierDimensions
type baziTierAssessment = TierAssessment
type baziStructuredStaticSynthesis = StructuredStaticSynthesis
type baziStructuredPeriodClaim = StructuredPeriodClaim
type baziStructuredDynamicSynthesis = StructuredDynamicSynthesis
type baziPatternCandidate = PatternCandidate
type baziPatternAdjudication = PatternAdjudication

// 以下别名保留 runtime 的恢复与 trace 调用点；合同审计数据结构由 domain 所有。
type baziContractAuditFinding = ContractAuditFinding
type baziContractAudit = ContractAudit

// 以下别名保留 runtime 的恢复与 trace 调用点；violation 数据结构由 domain 所有。
type baziViolationCode = ViolationCode
type baziValidationViolation = ValidationViolation

const (
	baziViolationFactRefMissing             = ViolationFactRefMissing
	baziViolationUndeclaredFactClaim        = ViolationUndeclaredFactClaim
	baziViolationFactConflict               = ViolationFactConflict
	baziViolationClaimNotAuthorized         = ViolationClaimNotAuthorized
	baziViolationScopeEscalation            = ViolationScopeEscalation
	baziViolationDayunCoverageMissing       = ViolationDayunCoverageMissing
	baziViolationMethodContract             = ViolationMethodContract
	baziViolationEvidenceTopicMissing       = ViolationEvidenceTopicMissing
	baziViolationSemanticContract           = ViolationSemanticContract
	baziViolationUnsupportedConcreteOutcome = ViolationUnsupportedConcreteOutcome
	baziViolationRendererContract           = ViolationRendererContract
)

// 以下别名保持 runtime 私有调用点稳定；模型数据合同由八字领域拥有。
type baziAnalysisPlan = AnalysisPlan
type baziLifetimeDayunClaim = LifetimeDayunClaim
type baziLifetimeDayunSynthesis = LifetimeDayunSynthesis

// 以下别名保持 runtime 私有调用点稳定；已接受综合结果由八字领域拥有。
type baziCanonicalUnit = CanonicalUnit
type baziCanonicalDayunUnit = CanonicalDayunUnit
type baziCanonicalSynthesis = CanonicalSynthesis
type baziStrengthJudgment = StrengthJudgment
type baziUsageLayers = UsageLayers

// baziDayunJudgment 保持既有 runtime DTO 名称，同时由八字领域拥有其合同。
type baziDayunJudgment = DayunJudgment

// 以下别名保持 runtime 私有调用点稳定；已验收综合结果由八字领域拥有。
type baziStaticSynthesis = StaticSynthesis
type baziDynamicSynthesis = DynamicSynthesis

type baziCharterState = CharterState
