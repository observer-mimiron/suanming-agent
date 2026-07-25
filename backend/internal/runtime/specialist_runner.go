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
	}, r.Executor.reg.DisplayName, shouldBufferFinalAnswer())
	if err != nil {
		return specialists.Result{}, err
	}

	return specialists.Result{
		Domain:  domain,
		Summary: finalText,
	}, nil
}

func validatePlanArtifacts(st *state.SessionState, plan ExecutionPlan) error {
	artifacts := plan.RequiredArtifacts
	if len(artifacts) == 0 {
		artifacts = selectRequiredArtifacts(plan.Domains)
	}
	domain := firstNonEmpty(plan.Route.PrimaryDomain, firstNonEmpty(plan.Domains...), "bazi")
	for _, artifact := range artifacts {
		if hasArtifact(st, artifact) {
			continue
		}
		return &RuntimeFailure{
			Class:       failureClassArtifactMissing,
			Stage:       failureStagePrefill,
			Domain:      domain,
			Degraded:    false,
			UserVisible: true,
			Message:     fmt.Sprintf("required artifact %s missing before specialist dispatch", artifact),
		}
	}
	return nil
}

func hasArtifact(st *state.SessionState, artifact string) bool {
	if st == nil {
		return false
	}
	switch artifact {
	case artifactBaziChart:
		return st.HasBaziResult()
	case artifactQimenChart:
		return st.HasQimenResult()
	case artifactZiweiChart:
		return st.HasZiWeiResult()
	default:
		return false
	}
}
