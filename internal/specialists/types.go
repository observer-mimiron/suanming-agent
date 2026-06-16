// Package specialists 定义了领域专家的通用接口和类型。
// 专家模式中每个 Specialist 负责一个特定命理领域，
// 通过 DomainHandler 接口统一调度执行。
package specialists

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Event 是专家执行过程中发出的事件。
type Event struct {
	Type string
	Data any
}

// EventSink 是专家事件接收回调函数，用于消费执行过程中的事件流。
type EventSink func(ctx context.Context, evt Event) error

// DomainHandler 是领域专家必须实现的接口契约。所有命理领域专家（八字、奇门、紫微斗数等）都需实现该接口。
type DomainHandler interface {
	Name() string
	Run(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, sink EventSink) (schemas.DomainResult, error)
}
