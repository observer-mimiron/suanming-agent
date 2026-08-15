// Package adapter 负责把 runtime 提供的共享能力接到八字 Graph。
//
// 本文件只声明领域值对象和 repair/事件端口的窄兼容类型；
// 不导入 runtime，不拥有 Manager 会话，也不定义新的 Runner 接口。
package adapter

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/repair"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	baziapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/application"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

// AgentBuilder 由 runtime 注入内层 agent 构建能力。
type AgentBuilder interface {
	BuildEphemeralInnerAgent(context.Context, specialists.Config, *specialists.SessionView) (adk.Agent, error)
}

type baziCharterInput = bazidomain.CharterInput
type baziCharterState = bazidomain.CharterState
type baziCanonicalSynthesis = bazidomain.CanonicalSynthesis
type baziCanonicalUnit = bazidomain.CanonicalUnit
type baziStaticSynthesis = bazidomain.StaticSynthesis
type baziDynamicSynthesis = bazidomain.DynamicSynthesis
type baziLifetimeDayunSynthesis = bazidomain.LifetimeDayunSynthesis
type baziAnalysisPlan = bazidomain.AnalysisPlan
type baziEvidencePlan = bazidomain.EvidencePlan
type baziEvidenceBundle = bazidomain.EvidenceBundle
type baziEvidenceQuality = bazidomain.EvidenceQuality
type baziQueryPacket = bazidomain.QueryPacket
type baziCitation = bazidomain.Citation
type baziAssertion = bazidomain.Assertion
type baziAssertionKind = bazidomain.AssertionKind
type baziFactRef = bazidomain.FactRef
type baziClaimRef = bazidomain.ClaimRef
type baziRelationRef = bazidomain.RelationRef
type baziStructuredClaim = bazidomain.StructuredClaim
type baziStructuredStaticClaim = bazidomain.StructuredStaticClaim
type baziStructuredStaticSynthesis = bazidomain.StructuredStaticSynthesis
type baziStructuredPeriodClaim = bazidomain.StructuredPeriodClaim
type baziStructuredDynamicSynthesis = bazidomain.StructuredDynamicSynthesis
type baziTierAssessment = bazidomain.TierAssessment
type baziTierDimension = bazidomain.TierDimension
type baziTierDimensions = bazidomain.TierDimensions
type baziNamedTierDimension struct {
	Name    string
	Value   baziTierDimension
	Disease bool
}
type BaziFactCapsule = bazidomain.FactCapsule
type baziRuntimeCatalog = bazidomain.ReferenceCatalog
type baziContractAudit = bazidomain.ContractAudit
type baziContractFailure = bazidomain.ContractFailure
type baziCanonicalDayunUnit = bazidomain.CanonicalDayunUnit

const (
	baziAssertionMainAxis            = bazidomain.AssertionMainAxis
	baziAssertionStrength            = bazidomain.AssertionStrength
	baziAssertionTiaohou             = bazidomain.AssertionTiaohou
	baziAssertionPatternUsage        = bazidomain.AssertionPatternUsage
	baziAssertionTier                = bazidomain.AssertionTier
	baziAssertionDayunPeriod         = bazidomain.AssertionDayunPeriod
	baziAssertionLiunian             = bazidomain.AssertionLiunian
	baziViolationMethodContract      = bazidomain.ViolationMethodContract
	baziViolationScopeEscalation     = bazidomain.ViolationScopeEscalation
	baziViolationUndeclaredFactClaim = bazidomain.ViolationUndeclaredFactClaim
)

func baziStaticRuntimeCatalogView(state baziCharterState) map[string]any {
	return bazidomain.StaticRuntimeCatalogView(state)
}
func baziDynamicRuntimeCatalogView(state baziCharterState) map[string]any {
	return bazidomain.DynamicRuntimeCatalogView(state)
}
func buildBaziFactCapsule(state baziCharterState) BaziFactCapsule {
	return bazidomain.FactCapsuleForState(state)
}
func baziDynamicPeriodRefs(state baziCharterState) []string {
	return bazidomain.DynamicPeriodRefs(state)
}
func buildBaziFactCapsulePromptView(state baziCharterState, includeDynamic bool) map[string]any {
	return bazidomain.BuildFactCapsulePromptView(state, includeDynamic)
}
func buildDynamicFactsView(input baziCharterInput) map[string]any {
	return baziapplication.BuildDynamicFactsView(input)
}
func buildEvidenceBundleView(bundle baziEvidenceBundle, includeTopics bool) map[string]any {
	return baziapplication.BuildEvidenceBundleView(bundle, includeTopics)
}
func buildBaziRuntimeCatalog(state baziCharterState) baziRuntimeCatalog {
	return bazidomain.BuildRuntimeCatalog(state)
}
func mapValue(src map[string]any, key string) map[string]any { return bazidomain.MapValue(src, key) }
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func baziViolationError(code bazidomain.ViolationCode, field, assertionID, message string, missing, allowed []string) error {
	return bazidomain.NewValidationError(code, field, assertionID, message, missing, allowed)
}
func currentDayunIndexForInput(input baziCharterInput) int {
	return bazidomain.CurrentDayunIndexForInput(input)
}
func factRefsToStrings(refs []baziFactRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if value := string(ref); value != "" {
			out = append(out, value)
		}
	}
	return out
}
func baziTierDimensionEntries(dimensions baziTierDimensions) []baziNamedTierDimension {
	return []baziNamedTierDimension{{Name: "main_axis", Value: dimensions.MainAxis}, {Name: "youqing", Value: dimensions.YouQing}, {Name: "youli", Value: dimensions.YouLi}, {Name: "qingzhuo", Value: dimensions.QingZhuo}, {Name: "disease", Value: dimensions.Disease, Disease: true}, {Name: "remedy", Value: dimensions.Remedy}, {Name: "rescue", Value: dimensions.Rescue}, {Name: "tiaohou", Value: dimensions.Tiaohou}, {Name: "hezhizhang", Value: dimensions.HeZhiZhang}}
}
func strengthEvidenceSummary(yongshen map[string]any) string {
	return bazidomain.StrengthEvidenceSummary(yongshen)
}
func capsuleTiaohouDisplay(c BaziFactCapsule) string      { return bazidomain.TiaohouDisplay(c) }
func capsuleTierEvidenceDisplay(c BaziFactCapsule) string { return bazidomain.TierEvidenceDisplay(c) }
func staticPatternFactSummary(input baziCharterInput) string {
	return bazidomain.StaticPatternFactSummary(input)
}
func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
func isFactsOnlyStaticSynthesis(s baziStaticSynthesis) bool {
	return bazidomain.IsFactsOnlyStaticSynthesis(s)
}
func isFactsOnlyDynamicSynthesis(s baziDynamicSynthesis) bool {
	return bazidomain.IsFactsOnlyDynamicSynthesis(s)
}
func baziContractAuditSummary(audit baziContractAudit) string {
	return bazidomain.ContractAuditSummary(audit)
}

const baziSynthesisSourceFactsOnlyDegraded = bazidomain.FactsOnlySource

func knownBaziFactRefs(state baziCharterState) map[string]struct{} {
	return bazidomain.KnownFactRefs(state)
}
func knownBaziClaimRefs(profile bazidomain.RuleProfile) map[string]struct{} {
	return bazidomain.KnownClaimRefs(profile)
}
func isKnownBaziFactRef(ref baziFactRef, known map[string]struct{}) bool {
	return bazidomain.IsKnownFactRef(ref, known)
}
func claimRefAllowsAssertionKind(profile bazidomain.RuleProfile, ref string, kind baziAssertionKind) bool {
	return bazidomain.ClaimRefAllowsAssertionKind(profile, ref, kind)
}
func baziTraceAttrsForContractFailure(stage string, err error) map[string]any {
	failure, ok := baziapplication.ContractFailureFromError(stage, err)
	if !ok {
		return nil
	}
	return map[string]any{
		"bazi.contract.failure_class":   failure.Class,
		"bazi.contract.recovery_policy": failure.RecoveryPolicy,
		"bazi.contract.finding_code":    failure.FindingCode,
		"bazi.contract.finding_field":   failure.Field,
	}
}
func baziViolationFromError(err error) (bazidomain.ValidationViolation, bool) {
	return bazidomain.ViolationFromError(err)
}
func baziContractFailureFromError(stage string, err error) (baziContractFailure, bool) {
	return baziapplication.ContractFailureFromError(stage, err)
}
func repairFailureFromBaziContract(stage string, err error) (repair.Failure, bool) {
	return baziapplication.RepairFailureFromError(stage, err)
}
func repairClassFromBaziContract(class string) repair.Class {
	switch class {
	case bazidomain.ContractFailureEvidenceOverclaim:
		return repair.EvidenceOverclaim
	case bazidomain.ContractFailureDomainUnauthorized:
		return repair.DomainUnauthorized
	case bazidomain.ContractFailureProjectionMismatch:
		return repair.ProjectionMismatch
	case bazidomain.ContractFailureSchemaError:
		return repair.SchemaError
	case bazidomain.ContractFailureFactConflict:
		return repair.FactConflict
	case bazidomain.ContractFailureMethodContract:
		return repair.MethodContract
	default:
		return repair.Unknown
	}
}
func repairFallbackFromBaziRecoveryPolicy(policy string) string {
	if policy == bazidomain.RecoveryPolicyStaticFactsOnly || policy == bazidomain.RecoveryPolicyDynamicFactsOnly || policy == bazidomain.RecoveryPolicyFullFactsOnly {
		return "facts_only"
	}
	return ""
}
func recordGraphFailure(_ context.Context, failure *graphFailure, domain, stage string, err error) error {
	if failure == nil || err == nil {
		return nil
	}
	failure.FailureClass = "invariant_failure"
	failure.FailureStage = stage
	failure.FailureCode = "BAZI_GRAPH_NODE_FAILED"
	failure.Domain = domain
	failure.Retryable = true
	failure.Message = err.Error()
	return nil
}
func graphFailureError(failure graphFailure) error {
	if failure.FailureClass == "" && failure.FailureCode == "" && failure.Message == "" {
		return nil
	}
	return fmt.Errorf("%s: %s", firstNonEmptyTrim(failure.FailureCode, failure.FailureClass), failure.Message)
}
func validateEvidenceBundlePreconditions(state baziCharterState) error {
	return bazidomain.ValidateEvidenceBundlePreconditions(state)
}
func validateStaticSynthesisResult(state baziCharterState, synthesis baziStaticSynthesis) error {
	return bazidomain.ValidateStaticSynthesisResult(state, synthesis)
}
func validateDynamicSynthesisAfterGraphNormalization(state baziCharterState) error {
	return bazidomain.ValidateDynamicSynthesisAfterGraphNormalization(state)
}
func projectCanonicalStaticSynthesis(state baziCharterState, canonical baziCanonicalSynthesis) baziStaticSynthesis {
	return baziapplication.ProjectCanonicalStaticSynthesis(state, canonical)
}
func projectCanonicalDynamicSynthesis(state baziCharterState, canonical baziCanonicalSynthesis, static baziStaticSynthesis) baziDynamicSynthesis {
	return baziapplication.ProjectCanonicalDynamicSynthesis(state, canonical, static)
}
func canonicalFailureFactsOnly(state baziCharterState, err error, auditCode, fallback string) (baziStaticSynthesis, baziDynamicSynthesis) {
	return baziapplication.CanonicalFailureFactsOnly(state, err, auditCode, fallback)
}
func canonicalDynamicFailureFactsOnly(state baziCharterState, static baziStaticSynthesis, err error) baziDynamicSynthesis {
	return baziapplication.CanonicalDynamicFailureFactsOnly(state, static, err)
}

// RuntimePort is the shared runtime capability surface used by the adapter.
// It contains only callbacks needed by the existing Graph and model stages.
type RuntimePort struct {
	Builder  AgentBuilder
	Registry *tools.Registry
	Sink     EventSink
}

// Executor 是八字 adapter 的单轮执行宿主，由 runtime 注入共享能力。
// 它不拥有 Manager 会话，也不负责跨领域调度。
type Executor struct {
	builder AgentBuilder
	reg     *tools.Registry
	port    RuntimePort
}

// graphFailure 是 adapter 内部的可序列化失败投影，不携带 runtime 错误对象。
type graphFailure struct {
	FailureClass string
	FailureStage string
	FailureCode  string
	Domain       string
	Retryable    bool
	Degraded     bool
	Message      string
	MissingRefs  []string
	AllowedRefs  []string
}

func (f graphFailure) hasFailure() bool {
	return f.FailureClass != "" || f.FailureCode != "" || f.Message != ""
}

// Runner 是八字 Graph 所需的运行时适配器。
// 它持有注入的共享能力，不拥有 Manager 会话或跨领域协调。
type Runner struct {
	Port RuntimePort
}

// Run executes one Bazi Graph request using only the shared capabilities injected by runtime.
func (r *Runner) Run(ctx context.Context, req specialists.Request) (specialists.Result, error) {
	if r == nil || r.Port.Builder == nil {
		return specialists.Result{}, fmt.Errorf("bazi adapter runner requires runtime capabilities")
	}
	if req.Session == nil {
		return specialists.Result{}, fmt.Errorf("bazi adapter runner requires session view")
	}
	executor := &Executor{builder: r.Port.Builder, reg: r.Port.Registry, port: r.Port}
	result, err := executor.baziGraphRuntimeResult(ctx, r.Port.Sink, req.Session, req.UserMessage)
	if err != nil {
		return specialists.Result{}, err
	}
	text, err := baziGraphTerminalText(result)
	if err != nil {
		return specialists.Result{}, err
	}
	return specialists.Result{Domain: "bazi", Summary: text}, nil
}

// EventSink is the runtime-owned event callback projected into the adapter.
type EventSink func(context.Context, Event) error

// Event is the adapter's transport-neutral event envelope.
type Event struct {
	Type string
	Data any
}

// RuntimeFailure 保留旧名称，实际使用公共 runner 失败合同。
type RuntimeFailure = specialists.Failure

type RepairTraceEvent struct {
	Failure           repair.Failure
	Attempt           int
	MaxAttempts       int
	Action            repair.Action
	Feedback          map[string]any
	LearningHintCount int
	Exhausted         bool
	FinalAction       repair.Action
}

func RepairTraceAttrs(event RepairTraceEvent) map[string]any {
	return map[string]any{
		"repair.domain": event.Failure.Domain, "repair.stage": event.Failure.Stage,
		"repair.class": string(event.Failure.Class), "repair.field": event.Failure.Field,
		"repair.attempt": event.Attempt, "repair.max_attempts": event.MaxAttempts,
		"repair.action": string(event.Action), "repair.learning_hint_count": event.LearningHintCount,
		"repair.exhausted": event.Exhausted, "repair.final_action": string(event.FinalAction),
	}
}

func graphFailureFromError(domain, stage string, err error) graphFailure {
	if err == nil {
		return graphFailure{}
	}
	return graphFailure{FailureClass: "invariant_failure", FailureStage: stage, FailureCode: "BAZI_GRAPH_NODE_FAILED", Domain: domain, Retryable: true, Message: err.Error()}
}

const (
	failureClassModelContractViolation = "model_contract_violation"
	failureClassArtifactMissing        = "artifact_missing"
	failureStageFinalWriter            = "final_writer"
	baziRecoveryPolicyHardError        = bazidomain.RecoveryPolicyHardError
	baziRecoveryPolicyRetryOnly        = bazidomain.RecoveryPolicyRetryOnly
	baziRecoveryPolicyStaticFactsOnly  = bazidomain.RecoveryPolicyStaticFactsOnly
	baziRecoveryPolicyDynamicFactsOnly = bazidomain.RecoveryPolicyDynamicFactsOnly
	baziRecoveryPolicyFullFactsOnly    = bazidomain.RecoveryPolicyFullFactsOnly
)

func emitEventWithTrace(ctx context.Context, sink EventSink, event Event, _ map[string]any) error {
	if sink == nil {
		return nil
	}
	return sink(ctx, event)
}

func baziSynthesisRuntimeFailure(stage, code string, cause error) error {
	return &RuntimeFailure{
		Class: failureClassModelContractViolation, Stage: stage, Domain: "bazi",
		Code: code, Retryable: true, UserVisible: true,
		Message: baziSynthesisFailureMessage(stage, cause),
		Cause:   cause,
	}
}

// baziSynthesisFailureMessage keeps user-visible contract failures specific without exposing candidate text.
func baziSynthesisFailureMessage(stage string, cause error) string {
	if failure, ok := baziapplication.ContractFailureFromError(stage, cause); ok {
		switch failure.Class {
		case bazidomain.ContractFailureEvidenceOverclaim:
			return "证据主题不足，已停止展示过度裁断。请稍后重试。"
		case bazidomain.ContractFailureDomainUnauthorized:
			return "岁运综合越过授权领域，已停止展示不稳定内容。请稍后重试。"
		case bazidomain.ContractFailureFactConflict, bazidomain.ContractFailureMethodContract:
			return "候选推演与事实或方法合同冲突，已停止展示不稳定内容。请稍后重试。"
		}
	}
	return "本轮八字综合未通过结构化合同校验，已停止展示不稳定内容。请稍后重试。"
}
