package runtime

import (
	"context"
	"fmt"

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

func (r *ADKSpecialistRunner) Run(ctx context.Context, req specialists.Request) (specialists.Result, error) {
	if r == nil || r.Executor == nil {
		return specialists.Result{}, fmt.Errorf("adk specialist runner requires executor")
	}
	if req.Session == nil {
		return specialists.Result{}, fmt.Errorf("adk specialist runner requires session state")
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
		r.Executor.saveToolResult(req.Session, toolName, resultJSON)
	}, r.Executor.reg.DisplayName, true)
	if err != nil {
		return specialists.Result{}, err
	}

	return specialists.Result{
		Domain:  domain,
		Summary: finalText,
	}, nil
}

func validatePlanArtifacts(st *state.SessionState, plan ExecutionPlan) error {
	for _, requirement := range plan.Requirements {
		if hasRequiredAsset(st, requirement) {
			continue
		}
		return artifactMissingFailure(plan, requirement.Kind)
	}
	return nil
}

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

func assetMatchesRequirement(asset state.DomainAsset, requirement ArtifactRequirement) bool {
	if asset.OwnerKind != requirement.OwnerRef.Kind || asset.OwnerID != requirement.OwnerRef.ID {
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

func artifactMissingFailure(plan ExecutionPlan, artifact string) error {
	domain := firstNonEmpty(plan.Route.PrimaryDomain, firstNonEmpty(plan.Domains...), "bazi")
	return artifactMissingFailureForDomain(domain, artifact)
}

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
