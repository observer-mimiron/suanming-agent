// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件负责一轮执行所需的会话上下文、路由快照和指导状态同步；
// 不负责 Graph 拓扑、领域模型调用或最终文本合同。
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// updateGuidanceState 是 guidance session mutation 的唯一入口。
// preflight 只返回下一状态，避免 graph 节点在澄清、fallback、正常执行之间分散写 session。
func (e *Executor) updateGuidanceState(st *state.SessionState, route policy.ApprovedRoute, message string, result preflightResult) {
	if st == nil {
		return
	}
	if result.ShortCircuit {
		// preflight 返回 GuidanceNext → executor 是唯一 mutation owner
		if result.GuidanceNext != nil {
			st.Guidance = result.GuidanceNext
		}
		return
	}

	// guided_fallback accepted → 清除 guidance
	if result.ForcedRoute != nil {
		st.Guidance = nil
		return
	}

	if shouldPreserveGuidanceOnExecution(route, st) {
		st.Guidance = policy.ReduceGuidance(policy.GuidanceReducerInput{
			Current: st.Guidance,
			Message: message,
			Profile: st.Profile,
		})
		return
	}

	st.Guidance = nil
}

// shouldPreserveGuidanceOnExecution keeps profile-collection guidance alive
// until the user supplies a complete profile or the route leaves profile collection.
func shouldPreserveGuidanceOnExecution(route policy.ApprovedRoute, st *state.SessionState) bool {
	if st == nil || st.Guidance == nil {
		return false
	}
	if st.IsProfileComplete() {
		return false
	}
	switch route.TaskIntent {
	case "collect_profile", "amend_profile":
		return true
	default:
		return false
	}
}

// saveToolResult 将 specialist 允许调用的确定性结果写回会话状态。
// Qimen 排盘只允许由 prefill 按 Manager 的 Case 合同写入；这里即使收到旧式
// qimen_dunjia 回调也必须丢弃，不能在缺少 Case owner 时生成错误资产。
func (e *Executor) saveToolResult(st *state.SessionState, toolName, resultJSON string) {
	// specialist 默认只挂知识工具；这里只保留历史兼容的出生盘写入。
	switch toolName {
	case "bazi_calc", "ziwei_calc":
	default:
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &payload); err != nil || payload == nil {
		if err != nil {
			log.Printf("saveToolResult: json unmarshal %s failed: %v", toolName, err)
		}
		return
	}
	switch toolName {
	case "bazi_calc":
		st.StoreChart(state.AssetKindBaziChart, payload, "lunar-go")
	case "ziwei_calc":
		st.StoreChart(state.AssetKindZiweiChart, payload, ziWeiMethodVersion())
	}
}

// buildSessionValues 构造 specialist prompt 所需的当前会话数据快照。
func (e *Executor) buildSessionValues(view *specialists.SessionView, route policy.ApprovedRoute) map[string]any {
	profile := map[string]any{}
	if view != nil && view.Profile != nil {
		profile = view.Profile
	}
	vals := map[string]any{
		"profile": profile,
		"domain":  route.PrimaryDomain,
	}
	if view == nil {
		return vals
	}
	if view.BaziResult != nil {
		vals["bazi_result"] = view.BaziResult
		if bj, err := json.Marshal(view.BaziResult); err == nil {
			vals["bazi_json"] = string(bj)
		}
	}
	if view.QimenResult != nil {
		vals["qimen_result"] = view.QimenResult
	}
	if view.ZiWeiResult != nil {
		vals["ziwei_result"] = view.ZiWeiResult
	}
	return vals
}

// updateRoutingSnapshot 将已批准的路由写回会话状态，供后续执行链复用。
func updateRoutingSnapshot(st *state.SessionState, route policy.ApprovedRoute) {
	// 同步 subject：TargetSubject 非空时更新，为空但有盘时默认"自己"
	if route.Slots.TargetSubject != "" {
		st.Subject = route.Slots.TargetSubject
	} else if st.Subject == "" && (st.HasBaziResult() || st.IsProfileComplete()) {
		st.Subject = "自己"
	}
	st.Routing = state.RoutingSnapshot{
		ConversationIntent:    route.ConversationIntent,
		PrimaryDomain:         route.PrimaryDomain,
		SecondaryDomains:      route.SecondaryDomains,
		TaskIntent:            route.TaskIntent,
		QimenMode:             route.PolicyHints.QimenMode,
		AwaitingClarification: route.NeedsClarification,
		Confidence:            route.Confidence,
		TimeScope:             route.Slots.TimeScope,
		TargetSubject:         route.Slots.TargetSubject,
	}
	st.LastInput = contracts.LastInputState{
		PreferredDomain:       route.PrimaryDomain,
		SecondaryDomains:      append([]string(nil), route.SecondaryDomains...),
		ExplicitMethod:        route.PrimaryDomain,
		ConsultMode:           route.TaskIntent,
		TimeScope:             route.Slots.TimeScope,
		TargetSubject:         route.Slots.TargetSubject,
		QuestionText:          route.Slots.QuestionText,
		GuidanceActive:        st.Guidance != nil,
		GuidanceDirectiveKind: guidanceDirectiveKind(st),
	}
}

// syncExecutionRoute synchronizes the approved route, execution snapshot and Manager turn owner.
func (e *Executor) syncExecutionRoute(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, plan ExecutionPlan) {
	updateRoutingSnapshot(st, route)
	st.Execution = plan.Snapshot
	if tr := tracing.TraceFromContext(ctx); tr != nil {
		tracing.SetLocalExecutionSnapshot(tr, plan.Snapshot)
	}
	if e != nil && e.manager != nil {
		e.manager.beginTurnForPlan(st, route, plan.TurnContext)
	}
	annotateApprovedRouteTrace(ctx, st, route)
}

// captureTurnContext captures the one runtime clock value shared by all deterministic preparation stages in a user turn.
func captureTurnContext(route policy.ApprovedRoute) contracts.TurnContext {
	now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60))
	questionTime := now.Format(time.RFC3339)
	return contracts.TurnContext{
		TurnID:              fmt.Sprintf("turn-%d", now.UnixNano()),
		QuestionTime:        questionTime,
		TargetAt:            questionTime,
		TemporalGranularity: temporalGranularityForRoute(route),
		Source:              "server_clock",
	}
}

// guidanceDirectiveKind 返回当前 guidance 状态的稳定标识。
func guidanceDirectiveKind(st *state.SessionState) string {
	if st == nil || st.Guidance == nil {
		return ""
	}
	return st.Guidance.DirectiveKind
}

// buildConversationMessages 从 specialist 会话读投影构建完整的输入消息列表。
// 若 RunningSummary 非空（对话超 30 轮后产生），作为 SystemMessage 注入消息列表开头，
// 让 supervisor 能看到早期对话的摘要，避免上下文断裂。
func (e *Executor) buildConversationMessages(view *specialists.SessionView, currentMessage string) []*schema.Message {
	if view == nil {
		return []*schema.Message{schema.UserMessage(currentMessage)}
	}
	limit := e.historyLimit
	if limit <= 0 {
		limit = len(view.RecentTurns)
	}

	msgs := make([]*schema.Message, 0, limit+2)

	if view.RunningSummary != "" {
		msgs = append(msgs, schema.SystemMessage("## 之前对话摘要（早期对话的压缩，供参考）\n\n"+view.RunningSummary))
	}

	start := 0
	if len(view.RecentTurns) > limit {
		start = len(view.RecentTurns) - limit
	}
	for i := start; i < len(view.RecentTurns); i++ {
		t := view.RecentTurns[i]
		if t.Role == "user" {
			msgs = append(msgs, schema.UserMessage(t.Content))
		} else {
			msgs = append(msgs, schema.AssistantMessage(t.Content, nil))
		}
	}
	msgs = append(msgs, schema.UserMessage(currentMessage))
	return msgs
}
