// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件负责 Manager 对跨轮 SessionState 的最小上下文投影和本轮时间合同；
// 不负责路由建议、Graph 节点、最终文本合成或传输层事件。
package runtime

import (
	"fmt"
	"strings"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// ensureTurnContext 填充本轮时间合同，并在需要时创建唯一的奇门 Case。
func (m *Manager) ensureTurnContext(st *state.SessionState, route policy.ApprovedRoute, message string, turn contracts.TurnContext) contracts.TurnContext {
	if strings.TrimSpace(turn.QuestionTime) == "" {
		now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60))
		turn.QuestionTime = now.Format(time.RFC3339)
		if strings.TrimSpace(turn.TargetAt) == "" {
			turn.TargetAt = turn.QuestionTime
		}
		if strings.TrimSpace(turn.Source) == "" {
			turn.Source = "server_clock"
		}
		if strings.TrimSpace(turn.TurnID) == "" {
			turn.TurnID = fmt.Sprintf("turn-%d", now.UnixNano())
		}
	}
	if strings.TrimSpace(turn.TargetAt) == "" {
		turn.TargetAt = turn.QuestionTime
	}
	if strings.TrimSpace(turn.Source) == "" {
		turn.Source = "server_clock"
	}
	if strings.TrimSpace(turn.TurnID) == "" {
		turn.TurnID = fmt.Sprintf("turn-%s", strings.TrimSpace(turn.QuestionTime))
	}
	if strings.TrimSpace(turn.TemporalGranularity) == "" {
		turn.TemporalGranularity = temporalGranularityForRoute(route)
	}
	if requiresQimenCase(route) && strings.TrimSpace(turn.CaseID) == "" && st != nil && !route.NeedsClarification {
		if questionTime, ok := parseTurnTime(turn.QuestionTime); ok {
			item := st.StartCaseAt("qimen", firstNonEmpty(strings.TrimSpace(route.Slots.QuestionText), strings.TrimSpace(message)), &questionTime, true)
			turn.CaseID = item.ID
		}
	}
	return turn
}

// requiresQimenCase identifies event-question routes whose primary artifact belongs to a Case.
func requiresQimenCase(route policy.ApprovedRoute) bool {
	return route.ConsultationKind == contracts.ConsultationKindEventQuestion
}

// temporalGranularityForRoute keeps the time contract explicit for downstream prefill.
func temporalGranularityForRoute(route policy.ApprovedRoute) string {
	scope := strings.ToLower(strings.TrimSpace(route.Slots.TimeScope))
	switch {
	case strings.Contains(scope, "月"), strings.Contains(scope, "month"):
		return "month"
	case strings.Contains(scope, "年"), strings.Contains(scope, "year"):
		return "year"
	case route.ConsultationKind == contracts.ConsultationKindEventQuestion:
		return "instant"
	default:
		return "range"
	}
}

// BeginTurn 在 runtime 进入本轮执行前更新 Manager 的最小会话上下文。
func (m *Manager) BeginTurn(st *state.SessionState, route policy.ApprovedRoute) {
	m.updateManagerTurnContext(st, route)
}

// beginTurnForPlan 更新 Manager 上下文并保留计划拥有的 CaseID。
func (m *Manager) beginTurnForPlan(st *state.SessionState, route policy.ApprovedRoute, turn contracts.TurnContext) {
	turn = m.ensureTurnContext(st, route, firstNonEmpty(route.Slots.QuestionText, route.TaskIntent), turn)
	_ = turn
	m.updateManagerTurnContext(st, route)
}

// updateManagerTurnContext 只写入会话里的 Manager 对话投影。
func (m *Manager) updateManagerTurnContext(st *state.SessionState, route policy.ApprovedRoute) {
	if st == nil {
		return
	}
	st.ManagerContext.ActiveDomain = route.PrimaryDomain
	st.ManagerContext.CurrentTopic = firstNonEmpty(
		strings.TrimSpace(route.Slots.QuestionText),
		strings.TrimSpace(route.Slots.TargetSubject),
		strings.TrimSpace(route.TaskIntent),
	)
}

// FinishTurn 在本轮成功结束后同步 Manager 状态。
func (m *Manager) FinishTurn(st *state.SessionState, route policy.ApprovedRoute, turnType string) {
	if st == nil {
		return
	}
	m.updateManagerTurnContext(st, route)
	st.ManagerContext.LastReplyOwner = "manager"
	st.ManagerContext.WaitingOn = waitingOnForTurnType(turnType)
}

// waitingOnForTurnType maps terminal turn types to the Manager 的下一步用户动作。
func waitingOnForTurnType(turnType string) string {
	switch turnType {
	case "clarification", "ask_missing_profile":
		return "user_reply"
	default:
		return ""
	}
}
