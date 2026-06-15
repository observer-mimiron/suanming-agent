package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

func currentQuestion(st *state.SessionState) string {
	if st.LastUserQuestion != "" {
		return st.LastUserQuestion
	}
	return "请先给出一段简明的命盘总评。"
}

// selectPrompt 根据会话路由状态选择系统提示词。
// 两种独立模式（奇门/直接）使用完整提示词文件。其他所有情况使用 interpret.md 作为稳定基础，
// 并注入任务特定的片段。
func (o *Orchestrator) selectPrompt(st *state.SessionState, qimenPrimary bool) []byte {
	// ---- 奇门主领域：无八字命盘时使用独立的奇门提示词 ----
	if qimenPrimary && !st.HasBaziResult() {
		return readFile("prompts/qimen.md")
	}
	// ---- 直接模式：面向基准测试，不设缓冲 ----
	if o.promptMode == "direct" {
		return readFile("prompts/direct.md")
	}

	// ---- 正常模式：interpret.md 基础 + 任务片段注入 ----
	base := readFile("prompts/interpret.md")
	snippet := o.pickSnippet(st)
	if len(snippet) == 0 {
		return base
	}
	return []byte(strings.Replace(string(base), "<!-- TASK_BLOCK -->", string(snippet), 1))
}

// pickSnippet 根据路由状态选择任务特定的指令片段。
func (o *Orchestrator) pickSnippet(st *state.SessionState) []byte {
	// 特定年份范围 -> 流年事件片段
	if st.Routing.TaskIntent == "timing_followup" && isYearScope(st.Routing.TimeScope) {
		return readFile("prompts/snippets/year_event.md")
	}

	// 运势 / 解读追问 -> 运势片段
	if st.Routing.TaskIntent == "fortune_followup" || st.Routing.TaskIntent == "interpret_chart" {
		return readFile("prompts/snippets/fortune.md")
	}

	// 领域特定 -> 定向片段
	switch st.Routing.TargetSubject {
	case "婚姻", "感情", "恋爱", "配偶":
		return readFile("prompts/snippets/marriage.md")
	case "事业", "财运", "工作", "职业":
		return readFile("prompts/snippets/career.md")
	case "健康", "疾病", "身体":
		return readFile("prompts/snippets/health.md")
	case "性格", "个性":
		return readFile("prompts/snippets/personality.md")
	}

	return readFile("prompts/snippets/default.md")
}

var yearScopeRe = regexp.MustCompile(`\d{4}\s*年`)

func isYearScope(scope string) bool {
	return yearScopeRe.MatchString(scope)
}

// readFile 读取文件并返回其内容，出错时返回空切片。
func readFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return b
}

// buildInterpretPrompt 组装完整的 LLM 系统提示词用于命盘解读。
// 通过 selectPrompt 选择基础提示词，然后注入：资料和命盘 JSON、知识库段落（格式化为可读引用）、
// 可选的奇门/额外数据、滚动摘要上下文、最近对话轮次和当前用户问题。返回纯文本字符串。
func (o *Orchestrator) buildInterpretPrompt(st *state.SessionState, passages []mcp.Passage, extra map[string]any, qimenPrimary bool) string {
	tpl := o.selectPrompt(st, qimenPrimary)
	profileJSON, err := json.Marshal(st.Profile)
	if err != nil {
		profileJSON = []byte("{}")
	}
	baziJSON, err := json.Marshal(st.BaziResult)
	if err != nil {
		baziJSON = []byte("{}")
	}

	// cleanSourceName 将知识来源标识转换为可读的书名/章节名。
	cleanSourceName := func(raw string) string {
		s := strings.TrimPrefix(raw, "knowledge://")
		bookMap := map[string]string{
			"ref-bazi-yuanhai":     "《渊海子平》",
			"ref-bazi-ditiansui":   "《滴天髓》",
			"ref-bazi-ziping":      "《子平真诠》",
			"ref-bazi-qiongtong":   "《穷通宝鉴》",
			"ref-bazi-sanming":     "《三命通会》",
			"ref-bazi-gelulunming": "《格局论命》",
			"ref-bazi-marriage":    "合婚参考资料",
			"ref-bazi-career":      "事业财运参考资料",
			"ref-bazi-dayun":       "大运流年参考资料",
			"ref-bazi-geju":        "格局与用神参考资料",
			"ref-bazi-knowledge":   "命理知识库",
		}
		for prefix, book := range bookMap {
			if strings.HasPrefix(s, prefix) {
				rest := strings.TrimPrefix(s, prefix)
				rest = strings.TrimLeft(rest, "-s0123456789")
				rest = strings.TrimLeft(rest, " -")
				if rest != "" {
					return book + " · " + rest
				}
				return book
			}
		}
		return s
	}

	// 将段落格式化为可读的参考列表，而非 JSON
	var refBlock string
	if len(passages) > 0 {
		refBlock = "\n### 参考资料（知识库检索结果）\n\n以下是从知识库中检索到的命理典籍和相关资料。**你必须**在关键论断中引用这些资料——直接引用原文并标注出处，不要只用自己的话复述。\n\n"
		for i, p := range passages {
			name := cleanSourceName(p.Source)
			content := strings.TrimSpace(p.Content)
			if len(content) > 300 {
				content = content[:300] + "…"
			}
			refBlock += fmt.Sprintf("%d. **%s**\n   %s\n\n", i+1, name, content)
		}
		refBlock += "**引用要求：** 上述资料中如果有与当前命盘特征直接对应的原文，你必须在分析中引用 1-2 条。引用时使用资料中标注的书名，格式如：\"《渊海子平》云：'……'\" 或 \"据《格局论命》记载：'……'\"。引用后用自己的话解释一句。禁止把参考资料当背景默默使用——要么引用，要么不用。\n\n"
	}

	prompt := string(tpl) + `

## 运行时上下文

当前日期：` + time.Now().Format("2006-01-02") + `（奇门排盘和流年判断以此日期为准）

### 出生资料
` + string(profileJSON) + `

### 命盘结果
` + string(baziJSON) + `
` + refBlock

	// 注入奇门结果（如果可用）
	if extra != nil {
		extraJSON, err := json.Marshal(extra)
		if err == nil {
			prompt += `
### 补充术数结果（奇门遁甲）

以下是基于当前问卜时间排出的奇门遁甲盘：

` + string(extraJSON) + `
`
			if st.HasBaziResult() {
				prompt += `
**奇门回答要求：**

1. 先引用命盘中的日主、五行偏旺、用神忌神做 1-2 句个人背景铺垫
2. 再结合奇门盘中的门、星、神、宫位分析当前时机吉凶
3. 最后给出基于两者综合的具体建议

八字看个人先天格局与长期趋势，奇门看当前时空窗口是否有利。两者缺一不可。禁止只输出奇门分析而不提八字背景。
`
			} else {
				prompt += `
**奇门回答要求：**

1. 以奇门盘中的门、星、神、宫位为核心，分析当前时机的吉凶趋势
2. 结合问卜时间所属的节气、局数，给出具体的时机判断和建议
3. 说明当前时空窗口对问卜事项的有利或不利因素

本次分析以奇门遁甲为主。奇门盘反映的是当前时空的能量分布，适用于判断时机、方位和吉凶。
`
			}
		}
	}

	// 历史摘要（滚动压缩后的上下文）
	if st.RunningSummary != "" {
		prompt += `
### 历史摘要

以下是此前对话的压缩摘要，包含了已确认的资料、问题主线和已给出的关键结论。请在回答时结合这些上下文：

` + st.RunningSummary + `
`
	}

	// 最近对话（最近几轮）
	if len(st.RecentTurns) > 0 {
		recentCount := len(st.RecentTurns)
		if recentCount > 4 {
			recentCount = 4
		}
		recent := st.RecentTurns[len(st.RecentTurns)-recentCount:]
		prompt += `
### 最近对话
`
		for _, t := range recent {
			role := "用户"
			if t.Role == "assistant" {
				role = "助手"
			}
			content := t.Content
			if len(content) > 300 {
				content = content[:300] + "...\n"
			}
			prompt += role + "：" + content + "\n\n"
		}
	}

	prompt += `
### 当前问题
` + currentQuestion(st)
	return prompt
}
