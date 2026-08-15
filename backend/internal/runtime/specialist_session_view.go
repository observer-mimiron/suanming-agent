// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件负责把完整会话投影为 runtime 内部的 specialist 只读视图；
// 不负责领域执行、资产写回或最终答复合成。
package runtime

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// sessionViewFromState 为 runtime 的前置准备生成完整只读快照。
// 它只在 runtime 内部用于 prefill 和 SessionValues，不得直接传入领域 Runner。
func sessionViewFromState(st *state.SessionState) *specialists.SessionView {
	if st == nil {
		return nil
	}
	st.MigrateLegacyAssets()
	view := &specialists.SessionView{
		Subject:        st.Subject,
		Profile:        st.Profile,
		BaziResult:     st.BaziResult,
		QimenResult:    st.QimenResult,
		ZiWeiResult:    st.ZiWeiResult,
		RunningSummary: st.RunningSummary,
		RecentTurns:    make([]specialists.SessionTurn, 0, len(st.RecentTurns)),
	}
	for _, turn := range st.RecentTurns {
		view.RecentTurns = append(view.RecentTurns, specialists.SessionTurn{
			Role:    turn.Role,
			Content: turn.Content,
		})
	}
	return view
}

// specialistSessionView 返回某个领域实际允许读取的最小会话投影。
// 结构化结果按领域白名单隔离，避免一个 specialist 通过会话输入读取其他领域的结论。
func specialistSessionView(st *state.SessionState, plan ExecutionPlan, domain string) *specialists.SessionView {
	if st == nil {
		return nil
	}
	st.MigrateLegacyAssets()
	conversation := func() ([]specialists.SessionTurn, string) {
		turns := make([]specialists.SessionTurn, 0, len(st.RecentTurns))
		for _, turn := range st.RecentTurns {
			turns = append(turns, specialists.SessionTurn{Role: turn.Role, Content: turn.Content})
		}
		return turns, st.RunningSummary
	}

	switch domain {
	case "qimen":
		caseID := strings.TrimSpace(plan.TurnContext.CaseID)
		// 奇门只消费当前 Case 盘面；出生资料和历史对话不属于问事盘事实。
		return &specialists.SessionView{QimenResult: st.QimenChartForCase(caseID)}
	case "ziwei":
		turns, summary := conversation()
		return &specialists.SessionView{
			Subject:        st.Subject,
			Profile:        st.Profile,
			ZiWeiResult:    st.ZiWeiResult,
			RecentTurns:    turns,
			RunningSummary: summary,
		}
	case "bazi":
		return baziSessionView(st)
	default:
		turns, summary := conversation()
		return &specialists.SessionView{
			Subject:        st.Subject,
			Profile:        st.Profile,
			RecentTurns:    turns,
			RunningSummary: summary,
		}
	}
}

// baziSessionView 返回八字 Graph 与 runner 可读取的最小会话投影。
// 它供旧入口过渡使用，不能用完整 SessionState 代替。
func baziSessionView(st *state.SessionState) *specialists.SessionView {
	if st == nil {
		return nil
	}
	st.MigrateLegacyAssets()
	turns := make([]specialists.SessionTurn, 0, len(st.RecentTurns))
	for _, turn := range st.RecentTurns {
		turns = append(turns, specialists.SessionTurn{Role: turn.Role, Content: turn.Content})
	}
	return &specialists.SessionView{
		Subject:        st.Subject,
		Profile:        st.Profile,
		BaziResult:     st.BaziResult,
		RecentTurns:    turns,
		RunningSummary: st.RunningSummary,
	}
}
