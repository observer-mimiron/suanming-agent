// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件把已批准计划适配为普通 ADK specialist 执行，并校验调度前资产；
// 不负责八字角色选择、Graph 入口或最终答复合成。
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// ADKSpecialistRunner is the bounded specialist execution contract used by the
// manager-owned runtime path. It runs one specialist agent directly as the
// current default execution path.
type ADKSpecialistRunner struct {
	Domain   string
	Config   specialists.Config
	Executor *Executor
}

// Run builds the configured specialist agent, streams its ADK events to SSE,
// invokes the runtime-provided tool-result writer, and returns a normalized result.
func (r *ADKSpecialistRunner) Run(ctx context.Context, req specialists.Request) (specialists.Result, error) {
	if r == nil || r.Executor == nil {
		return specialists.Result{}, fmt.Errorf("adk specialist runner requires executor")
	}
	if req.Session == nil {
		return specialists.Result{}, fmt.Errorf("adk specialist runner requires session view")
	}

	sink := eventSinkFromContext(ctx)
	agent, err := r.Executor.builder.BuildSpecialist(ctx, r.Config, req.Session)
	if err != nil {
		return specialists.Result{}, fmt.Errorf("build specialist %s: %w", r.Config.Name, err)
	}

	domain := firstNonEmpty(r.Domain, r.Config.Domain, req.Route.PrimaryDomain, "bazi")
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "adk_specialist_agent",
		Kind: tracing.KindChain,
		Attributes: map[string]any{
			"model":  r.Executor.llmModel,
			"domain": domain,
		},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true})
	iter := runner.Run(
		ctx,
		r.Executor.buildConversationMessages(req.Session, req.UserMessage),
		adk.WithSessionValues(r.Executor.buildSessionValues(req.Session, req.Route)),
	)

	finalText, err := specialistEventBridge(ctx, sink, iter, func(toolName, resultJSON string) {
		if req.SaveToolResult != nil {
			req.SaveToolResult(toolName, resultJSON)
		}
	}, r.Executor.reg.DisplayName, true)
	if err != nil {
		return specialists.Result{}, err
	}

	return specialists.Result{
		Domain:  domain,
		Summary: finalText,
	}, nil
}

// validatePlanArtifacts blocks specialist dispatch unless prefill satisfied every
// exact ArtifactRequirement from the Manager's plan.
func validatePlanArtifacts(st *state.SessionState, plan ExecutionPlan) error {
	for _, requirement := range plan.Requirements {
		if hasRequiredAsset(st, requirement) {
			continue
		}
		return artifactMissingFailure(plan, requirement.Kind)
	}
	return nil
}

// hasRequiredAsset checks active focus refs first, then verifies the underlying
// asset owner, subject set, and calendar rule match the requirement.
func hasRequiredAsset(st *state.SessionState, requirement ArtifactRequirement) bool {
	if st == nil {
		return false
	}
	for _, ref := range st.ActiveFocus.PrimaryAssetRefs {
		if ref.Kind != requirement.Kind {
			continue
		}
		for _, asset := range st.Assets {
			if asset.Ref != ref || !assetMatchesRequirement(asset, requirement) {
				continue
			}
			return true
		}
	}
	return false
}

// assetMatchesRequirement verifies one persisted asset against a precise plan requirement.
func assetMatchesRequirement(asset state.DomainAsset, requirement ArtifactRequirement) bool {
	if asset.OwnerKind != requirement.OwnerRef.Kind || asset.OwnerID != requirement.OwnerRef.ID {
		return false
	}
	if requirement.Kind == artifactQimenChart && !qimenPayloadMatchesRequirement(asset.Payload, requirement) {
		return false
	}
	if rule := requirement.CalendarRule; rule != "" && asset.CalendarRule != rule {
		return false
	}
	for _, requiredSubjectID := range requirement.SubjectIDs {
		found := false
		for _, assetSubjectID := range asset.SubjectIDs {
			if assetSubjectID == requiredSubjectID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// qimenPayloadMatchesRequirement verifies the runtime metadata that legacy
// QimenResult projections do not carry and therefore cannot satisfy.
func qimenPayloadMatchesRequirement(payload map[string]any, requirement ArtifactRequirement) bool {
	if len(payload) == 0 || stringValue(payload["case_id"]) != requirement.OwnerRef.ID {
		return false
	}
	owner, ok := payload["owner_ref"].(map[string]any)
	if !ok || stringValue(owner["kind"]) != "case" || stringValue(owner["id"]) != requirement.OwnerRef.ID {
		return false
	}
	if stringValue(payload["purpose"]) != "event_question" ||
		stringValue(payload["time_source"]) != "question_time" ||
		stringValue(payload["pan_schema"]) != "rotating_8" ||
		stringValue(payload["symbol_system"]) != "eight_gate_eight_god" {
		return false
	}
	if targetAt := strings.TrimSpace(requirement.TargetAt); targetAt != "" && stringValue(payload["question_time"]) != targetAt {
		return false
	}
	return true
}

// artifactMissingFailure maps a missing plan artifact into a user-visible runtime failure.
func artifactMissingFailure(plan ExecutionPlan, artifact string) error {
	domain := firstNonEmpty(plan.Route.PrimaryDomain, firstNonEmpty(plan.Domains...), "bazi")
	return artifactMissingFailureForDomain(domain, artifact)
}

// artifactMissingFailureForDomain builds the concrete prefill failure reported to handler/SSE.
func artifactMissingFailureForDomain(domain, artifact string) error {
	return &RuntimeFailure{
		Class:       failureClassArtifactMissing,
		Stage:       failureStagePrefill,
		Domain:      domain,
		Code:        "REQUIRED_ARTIFACT_UNAVAILABLE",
		Retryable:   true,
		Degraded:    false,
		UserVisible: true,
		Message:     fmt.Sprintf("required artifact %s missing before specialist dispatch", artifact),
	}
}
