package orchestrator

import (
	"context"

	appRuntime "github.com/wikiglobal/suanming-agent/internal/runtime"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// RouteAdvisor 是 LLM 驱动的路由决策接口。
type RouteAdvisor interface {
	Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error)
}

type Event = appRuntime.Event
type EventSink = appRuntime.EventSink
