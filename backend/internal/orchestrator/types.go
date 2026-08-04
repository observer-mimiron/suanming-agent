// This file belongs to the session orchestration layer.
// It owns shared model-facing types for this package.
// It owns turn lifecycle; domain reasoning stays in runtime and tools.
package orchestrator

import (
	"context"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	appRuntime "github.com/observer-mimiron/suanming-agent/internal/runtime"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// RouteAdvisor 是 LLM 驱动的路由决策接口。
type RouteAdvisor interface {
	Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error)
}

type Event = appRuntime.Event
type EventSink = appRuntime.EventSink
