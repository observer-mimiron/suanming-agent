package runtime

import (
	"regexp"
	"strings"
)

// finalOutputSanitizerRe 匹配 LLM 服务端/平台追加到输出末尾的免责声明。
//
// 仅匹配「末尾整行」的声明，避免误伤正文中合法出现的「仅供参考」。
// 当前覆盖：
//   - DeepSeek：「以上内容由 DeepSeek 生成，仅供参考。」（含 markdown bold 变体）
//
// 扩展点：新增模型/平台尾部声明时，扩展此正则或在 sanitizeFinalOutput 内追加规则。
var finalOutputSanitizerRe = regexp.MustCompile(`\s*\**\s*以上内容由[^\n]*?生成[^\n]*?仅供参考[^\n]*?[。.]?\**\s*$`)

// sanitizeFinalOutput 清洗 LLM 输出末尾的模型/平台追加声明（免责声明、AI 生成提示等）。
//
// 兜底逻辑：即使 specialist prompt 没有显式禁止，也能在运行时统一剥除，
// 避免每个领域 prompt 都重复写「不得追加免责声明」规则。
//
// 空串输入原样返回。多行重复声明会被循环剥净。
func sanitizeFinalOutput(text string) string {
	if text == "" {
		return text
	}
	for {
		trimmed := finalOutputSanitizerRe.ReplaceAllString(text, "")
		if len(trimmed) == len(text) {
			break
		}
		text = trimmed
	}
	return strings.TrimSpace(text)
}
