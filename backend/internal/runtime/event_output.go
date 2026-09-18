// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件负责从模型输出中投影 analysis/response 段并清理最终文本；
// 不负责 ADK 事件消费、SSE 发送或领域结论。
package runtime

import "strings"

// parseXMLSections 解析 LLM 输出中的 <analysis> 和 <response> XML 标记段。
//
// 返回 (analysisText, responseText, hasTags)。hasTags 为 false 时表示输入不含标记，
// 此时整个 input 视为 responseText（降级行为）。
func parseXMLSections(input string) (analysis, response string, hasTags bool) {
	analysisStart := strings.Index(input, "<analysis>")
	analysisEnd := strings.Index(input, "</analysis>")
	responseStart := strings.Index(input, "<response>")
	responseEnd := strings.Index(input, "</response>")

	if analysisStart == -1 || analysisEnd == -1 || responseStart == -1 || responseEnd == -1 {
		return "", strings.TrimSpace(input), false
	}

	analysis = strings.TrimSpace(input[analysisStart+len("<analysis>") : analysisEnd])
	response = strings.TrimSpace(input[responseStart+len("<response>") : responseEnd])
	return analysis, response, true
}
