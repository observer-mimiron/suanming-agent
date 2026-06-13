package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

const summarySystemPrompt = `你是一个结构化摘要助手。将对话历史压缩为简洁的结构化摘要，与新发生的对话合并。

已有摘要时，你必须在保留已有摘要关键信息的基础上补充新内容，而不是从头重写。
如果新对话没有覆盖某个方面（如出生资料无变化），直接保留已有摘要中该方面的内容。

输出格式严格如下（用中文，每个方面最多2句话）：

出生资料：已确认的出生资料及其变化
问题主线：用户当前主要关心的问题方向
关键结论：已给出的重要分析结论和建议
待解决问题：仍然未解答或需要跟进的问题`

// summarizeTurns 将溢出的对话轮次压缩合并到 RunningSummary 中。
// oldSummary 是已有的滚动摘要，turns 是需要新增压缩的轮次。
// 失败时返回旧摘要和 false，不中断主流程。
func (o *Orchestrator) summarizeTurns(ctx context.Context, oldSummary string, turns []state.Turn) (string, bool) {
	if len(turns) == 0 {
		return oldSummary, true
	}

	var dialogBuilder strings.Builder
	for _, t := range turns {
		role := "用户"
		if t.Role == "assistant" {
			role = "助手"
		}
		// 截断过长内容，保持摘要输入可控
		content := t.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		dialogBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, content))
	}

	existingSummary := oldSummary
	if existingSummary == "" {
		existingSummary = "无"
	}

	userMsg := fmt.Sprintf(
		"已有摘要：\n%s\n\n新对话轮次：\n%s\n\n请生成更新后的摘要。",
		existingSummary,
		dialogBuilder.String(),
	)

	messages := []llm.Message{{Role: "user", Content: userMsg}}
	result, _, err := o.flash.Generate(ctx, summarySystemPrompt, messages)
	if err != nil {
		// 摘要失败不中断主流程，保留旧摘要
		return oldSummary, false
	}

	summary := strings.TrimSpace(result)
	if summary == "" {
		return oldSummary, false
	}
	return summary, true
}
