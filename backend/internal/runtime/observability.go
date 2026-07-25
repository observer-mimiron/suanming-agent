package runtime

import (
	"context"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
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

	hasArtifact, failureMessage := primaryArtifactGuard(route, st)
	sp.SetAttribute("artifact_present", hasArtifact)
	if !hasArtifact {
		failure := &RuntimeFailure{
			Class:       failureClassSpecialistContractViolation,
			Stage:       failureStageFinalGuard,
			Domain:      route.PrimaryDomain,
			Degraded:    false,
			UserVisible: true,
			Message:     failureMessage,
		}
		sp.SetAttribute("guardrail_result", "blocked")
		sp.SetAttribute("failure.class", failure.Class)
		sp.SetAttribute("failure.stage", failure.Stage)
		sp.SetAttribute("failure.domain", failure.Domain)
		sp.SetStatus("error")
		annotateRuntimeFailureTrace(ctx, failure)
		return "guardrail_blocked", failure.Message
	}

	if ok, reason := outputBoundaryGuard(finalText); !ok {
		failure := &RuntimeFailure{
			Class:       failureClassSpecialistContractViolation,
			Stage:       failureStageFinalGuard,
			Domain:      route.PrimaryDomain,
			Degraded:    false,
			UserVisible: true,
			Message:     "最终回答包含内部执行细节，已拦截本轮输出。请重试；若再次出现，请检查 manager compose 或 specialist 输出是否泄漏系统提示、trace 或工具调用细节。",
		}
		sp.SetAttribute("guardrail_result", "blocked")
		sp.SetAttribute("failure.class", failure.Class)
		sp.SetAttribute("failure.stage", failure.Stage)
		sp.SetAttribute("failure.domain", failure.Domain)
		sp.SetAttribute("failure.reason", reason)
		sp.SetStatus("error")
		annotateRuntimeFailureTrace(ctx, failure)
		return "guardrail_blocked", failure.Message
	}

	sp.SetAttribute("guardrail_result", "passed")
	return "agent_reading", finalText
}

func primaryArtifactGuard(route policy.ApprovedRoute, st *state.SessionState) (bool, string) {
	switch route.PrimaryDomain {
	case "bazi":
		if st != nil && st.HasBaziResult() {
			return true, ""
		}
		return false, "本轮问题已判定为八字主链，但运行时没有拿到八字命盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `bazi_calc` 是否真正被调用。"
	case "qimen":
		if st != nil && st.HasQimenResult() {
			return true, ""
		}
		return false, "本轮问题已判定为奇门主链，但运行时没有拿到奇门盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `qimen_dunjia` 是否真正被调用。"
	case "ziwei":
		if st != nil && st.HasZiWeiResult() {
			return true, ""
		}
		return false, "本轮问题已判定为紫微主链，但运行时没有拿到紫微命盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `ziwei_calc` 是否真正被调用。"
	default:
		return true, ""
	}
}

func outputBoundaryGuard(finalText string) (bool, string) {
	text := strings.ToLower(strings.TrimSpace(finalText))
	if text == "" {
		return true, ""
	}
	for _, marker := range []string{
		"system prompt",
		"系统提示",
		"chain-of-thought",
		"trace_id",
		"tool_call",
		"工具调用参数",
		"调用工具参数",
	} {
		if strings.Contains(text, strings.ToLower(marker)) {
			return false, marker
		}
	}
	return true, ""
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
		"gate.reason":                      route.Gate.Reason,
		"gate.execution_mode":              route.Gate.ExecutionMode,
		"gate.followup_policy":             route.Gate.FollowupPolicy,
		"decision_source":                  decisionSourceForRoute(route),
		"reuse_cached_result":              route.Gate.ReuseCachedResult,
		"reuse_session_profile":            route.Gate.ReuseSessionProfile,
	})
}

func decisionSourceForRoute(route policy.ApprovedRoute) string {
	if route.Gate.Reason == "cheap_followup_reuse" {
		return "cheap_followup_reuse"
	}
	return "supervisor"
}
