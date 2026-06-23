package runtime

import (
	"context"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// emitEventWithTrace wraps SSE emission with a local span so traces can explain
// what the runtime actually sent to the frontend.
func emitEventWithTrace(ctx context.Context, sink EventSink, evt Event, attrs map[string]any) error {
	if sink == nil {
		return nil
	}

	sp := tracing.SpanFromContext(ctx, "sse_emit", tracing.KindChain)
	sp.SetAttribute("event_type", evt.Type)
	sp.SetAttribute("sse.event_type", evt.Type)
	for k, v := range attrs {
		sp.SetAttribute(k, v)
	}
	defer sp.End()

	if err := sink.Emit(ctx, evt); err != nil {
		sp.RecordError(err)
		sp.SetStatus("error")
		return err
	}
	return nil
}

func guardFinalAnswerWithTrace(ctx context.Context, route policy.ApprovedRoute, st *state.SessionState, finalText string) (turnType string, text string) {
	sp := tracing.SpanFromContext(ctx, "contract_gate", tracing.KindChain)
	sp.SetAttribute("primary_domain", route.PrimaryDomain)
	sp.SetAttribute("buffer_final", shouldBufferFinalAnswer())
	defer sp.End()

	if route.PrimaryDomain == "qimen" {
		hasArtifact := st.HasQimenResult()
		sp.SetAttribute("artifact_present", hasArtifact)
		if !hasArtifact {
			sp.SetAttribute("guardrail_result", "blocked")
			sp.SetStatus("error")
			return "guardrail_blocked", "本轮问题已判定为奇门主链，但运行时没有拿到奇门盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `qimen_dunjia` 是否真正被调用。"
		}
	}
	if route.PrimaryDomain == "ziwei" {
		hasArtifact := st.HasZiWeiResult()
		sp.SetAttribute("artifact_present", hasArtifact)
		if !hasArtifact {
			sp.SetAttribute("guardrail_result", "blocked")
			sp.SetStatus("error")
			return "guardrail_blocked", "本轮问题已判定为紫微主链，但运行时没有拿到紫微命盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `ziwei_calc` 是否真正被调用。"
		}
	}

	sp.SetAttribute("artifact_present", true)
	sp.SetAttribute("guardrail_result", "passed")
	return "agent_reading", finalText
}

func annotateApprovedRouteTrace(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute) {
	tracing.SetTraceAttributes(ctx, map[string]any{
		"approved_route.primary_domain":    route.PrimaryDomain,
		"approved_route.secondary_domains": strings.Join(route.SecondaryDomains, ","),
		"task_intent":                      route.TaskIntent,
		"qimen_mode":                       route.PolicyHints.QimenMode,
		"profile_requirement":              route.PolicyHints.ProfileRequirement,
		"needs_clarification":              route.NeedsClarification,
		"profile_complete":                 st != nil && st.IsProfileComplete(),
	})
}
