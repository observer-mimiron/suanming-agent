// Package runtime 包含 Manager 拥有的八字最终渲染。
//
// 本文件只把已验证的静态和动态槽位转成用户可见 Markdown；
// 不重判命理事实、不改写层次资格，也不承担模型修复或图调度。
package runtime

import "strings"

// renderBaziFinalReply 改为由程序直接消费上游结构化结论并渲染最终 markdown。
// 这样最终成文不再依赖自由文本 writer 自行排版，从根上消除标题、加粗结论、
// 编号步骤等展示合同漂移导致的整轮失败。
func renderBaziFinalReply(plan baziAnalysisPlan, state baziCharterState, question string) string {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		return renderFactsOnlyDegradedTemplate(state)
	}
	switch strings.TrimSpace(plan.WriterTemplate) {
	case "topic":
		return renderTopicTemplate(plan, state, question)
	case "year":
		return renderYearTemplate(state, question)
	default:
		return renderFullTemplate(state)
	}
}
