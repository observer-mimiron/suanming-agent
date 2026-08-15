// Package presentation 包含八字已验收结果的用户可见投影。
//
// 本文件负责根据展示计划把 FinalReplyInput 渲染成 Markdown；
// 不读取 runtime、SessionState、模型、检索、trace 或 SSE，也不重新校验领域合同。
package presentation

import "strings"

// RenderFinalReply 将已验收的展示输入转换为最终 Markdown。
// 它只选择展示模板；输入的事实、边界和结论必须在进入本包前已完成校验。
func RenderFinalReply(input FinalReplyInput) string {
	if input.StaticSynthesis.FactsOnly {
		return renderPresentationFactsOnlyDegradedTemplate(input)
	}
	switch strings.TrimSpace(input.AnalysisPlan.WriterTemplate) {
	case "topic":
		return renderPresentationTopicTemplate(input)
	case "year":
		return renderPresentationYearTemplate(input)
	default:
		return renderPresentationFullTemplate(input)
	}
}
