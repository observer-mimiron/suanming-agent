// package presentation 包含 Manager 拥有的八字最终渲染。
// 本文件只把已验证的静态和动态槽位转成用户可见 Markdown；
// 不重判命理事实、不改写层次资格，也不承担模型修复或图调度。
package presentation

import bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"

// renderBaziFinalReply 将 Graph 后的状态先收敛为 presentation DTO，再渲染最终 markdown。
// 这样最终成文不再依赖自由文本 writer 自行排版，从根上消除标题、加粗结论、
// 编号步骤等展示合同漂移导致的整轮失败。
func renderBaziFinalReply(plan baziAnalysisPlan, state baziCharterState, question string) string {
	return RenderFinalReply(buildBaziFinalPresentationInput(plan, state, question))
}

// RenderFinalReplyForState renders an accepted Bazi state for the runtime adapter.
func RenderFinalReplyForState(plan bazidomain.AnalysisPlan, state bazidomain.CharterState, question string) string {
	return renderBaziFinalReply(plan, state, question)
}
