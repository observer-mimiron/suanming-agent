// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件把共享模型、工具和事件能力投影给领域 adapter；
// 不选择领域、执行领域 Graph 或拥有领域会话。
package runtime

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

// SpecialistAgentBuilder 是领域 adapter 构建短生命周期模型 Agent 所需的最小能力。
type SpecialistAgentBuilder interface {
	BuildEphemeralInnerAgent(context.Context, specialists.Config, *specialists.SessionView) (adk.Agent, error)
}

// SpecialistServices 是 container 注入领域 adapter 的共享运行时能力。
// 它不暴露 Executor、SessionState 或 Manager 的跨领域状态。
type SpecialistServices struct {
	Builder  SpecialistAgentBuilder
	Registry *tools.Registry
	Emit     func(context.Context, string, any) error
}

// SpecialistServices 返回领域 adapter 可消费的运行时能力投影。
func (e *Executor) SpecialistServices() SpecialistServices {
	if e == nil {
		return SpecialistServices{}
	}
	return SpecialistServices{
		Builder:  e.builder,
		Registry: e.reg,
		Emit: func(ctx context.Context, eventType string, data any) error {
			return emitEventWithTrace(ctx, eventSinkFromContext(ctx), Event{Type: eventType, Data: data}, nil)
		},
	}
}
