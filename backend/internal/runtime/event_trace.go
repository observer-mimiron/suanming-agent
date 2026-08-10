// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件负责事件发送 trace 和 route/gate 观测；
// 不负责事件分类、最终合同或领域解释。
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

// annotateApprovedRouteTrace records the approved route and gate projection for debugging.
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

// decisionSourceForRoute reports whether the current route came from supervisor or cheap reuse.
func decisionSourceForRoute(route policy.ApprovedRoute) string {
	if route.Gate.Reason == "cheap_followup_reuse" {
		return "cheap_followup_reuse"
	}
	return "supervisor"
}
