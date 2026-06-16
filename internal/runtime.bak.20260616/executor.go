package runtime

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// Executor 负责执行已批准的路由。
type Executor struct {
	tools         *tools.Registry
	llm           llm.Chat
	tracer        tracing.Tracer
	llmModel      string
	promptBuilder *Builder
	baziSp        specialists.DomainHandler
	qimenSp       specialists.DomainHandler
	ziweiSp       specialists.DomainHandler
}

// NewExecutor 创建运行时执行器。
func NewExecutor(reg *tools.Registry, llmClient llm.Chat, tracer tracing.Tracer, promptMode string) *Executor {
	return &Executor{
		tools:         reg,
		llm:           llmClient,
		tracer:        tracer,
		promptBuilder: NewBuilder(llmClient, promptMode),
	}
}

// SetLLMModel 设置用于 LLM span 元数据的模型名称。
func (e *Executor) SetLLMModel(model string) { e.llmModel = model }

// SetSpecialists 注入八字、奇门和紫微斗数领域专家。
func (e *Executor) SetSpecialists(baziSp, qimenSp, ziweiSp specialists.DomainHandler) {
	e.baziSp = baziSp
	e.qimenSp = qimenSp
	e.ziweiSp = ziweiSp
}

// PromptBuilder 返回当前运行时使用的 prompt builder。
func (e *Executor) PromptBuilder() *Builder { return e.promptBuilder }

// Execute 执行已批准的路由。
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	updateRoutingSnapshot(st, route)

	if final, summary := e.dispatchSpecialists(ctx, sink, st, route); final {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": summary}})
		return "ask_missing_profile", summary, nil
	}

	return e.ExecuteRoute(ctx, sink, st, route, message)
}

func specialistEventSink(sink EventSink) specialists.EventSink {
	return func(ctx context.Context, evt specialists.Event) error {
		return sink.Emit(ctx, Event{Type: evt.Type, Data: evt.Data})
	}
}

// updateRoutingSnapshot 将已批准的路由写回会话状态，供后续执行链复用。
func updateRoutingSnapshot(st *state.SessionState, route policy.ApprovedRoute) {
	st.Routing = state.RoutingSnapshot{
		ConversationIntent:    route.ConversationIntent,
		PrimaryDomain:         route.PrimaryDomain,
		SecondaryDomains:      route.SecondaryDomains,
		TaskIntent:            route.TaskIntent,
		AwaitingClarification: route.NeedsClarification,
		Confidence:            route.Confidence,
		TimeScope:             route.Slots.TimeScope,
		TargetSubject:         route.Slots.TargetSubject,
	}
}

// dispatchSpecialists 执行领域专家分发；若专家已经给出最终答复则直接返回。
func (e *Executor) dispatchSpecialists(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute) (bool, string) {
	if e.baziSp == nil && e.qimenSp == nil {
		return false, ""
	}

	dispSpan := tracing.SpanFromContext(ctx, "domain_dispatch", tracing.KindChain)
	var spResult schemas.DomainResult
	var spErr error
	if primarySp := e.selectPrimarySpecialist(route.PrimaryDomain); primarySp != nil {
		spResult, spErr = primarySp.Run(ctx, st, route, specialistEventSink(sink))
		dispSpan.SetAttribute("primary_domain", spResult.Domain)
		dispSpan.SetAttribute("final", spResult.Final)
		if spErr != nil {
			dispSpan.SetAttribute("error", spErr.Error())
		}
	}
	dispSpan.End()

	if spResult.Final {
		return true, spResult.Summary
	}

	e.dispatchSecondarySpecialists(ctx, sink, st, route)
	return false, ""
}

// selectPrimarySpecialist 根据主领域选择首个执行的专家。
func (e *Executor) selectPrimarySpecialist(primaryDomain string) specialists.DomainHandler {
	primarySp := e.baziSp
	switch primaryDomain {
	case "qimen":
		if e.qimenSp != nil {
			primarySp = e.qimenSp
		}
	case "ziwei":
		if e.ziweiSp != nil {
			primarySp = e.ziweiSp
		}
	case "bazi":
		if e.baziSp == nil && e.qimenSp != nil {
			primarySp = e.qimenSp
		}
	default:
		if primarySp == nil {
			primarySp = e.qimenSp
		}
	}
	return primarySp
}

// dispatchSecondarySpecialists 执行辅助领域专家带来的补充副作用。
func (e *Executor) dispatchSecondarySpecialists(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute) {
	for _, domain := range route.SecondaryDomains {
		switch domain {
		case "qimen":
			if e.qimenSp != nil {
				qimenSpResult, _ := e.qimenSp.Run(ctx, st, route, specialistEventSink(sink))
				if qimenSpResult.Domain == "qimen" && !qimenSpResult.Final {
					st.NeedsQimen = true
				}
			}
		case "ziwei":
			if e.ziweiSp != nil {
				ziweiSpResult, _ := e.ziweiSp.Run(ctx, st, route, specialistEventSink(sink))
				if ziweiSpResult.Domain == "ziwei" && !ziweiSpResult.Final {
					st.DomainStates.ZiWei.ChartReady = true
				}
			}
		}
	}
}
