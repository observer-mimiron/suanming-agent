package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// yearEventRe matches questions that ask about a specific year or event time.
var yearEventRe = regexp.MustCompile(`\d{4}\s*年|哪年|何年|发生何事|哪一年|何年何月|流年`)

// diagnosticRe matches questions that ask for a diagnostic judgment about an
// existing chart ("how is X"), as opposed to consulting questions ("when will I").
// When a chart already exists, these should use forensic prompt.
var diagnosticRe = regexp.MustCompile(`如何|怎么样|怎样|好不好|是否|会不会`)

// existing chart ("how is X", "what is X like"), as opposed to consulting
// questions ("when will I get married"). When a chart already exists, these
// should use forensic prompt instead of warm consulting prompts.

func (o *Orchestrator) buildKnowledgeQuery(ctx context.Context, st *state.SessionState, qimenData map[string]any) string {
	// When qimen data is available and no bazi chart exists, search qimen terms.
	if qimenData != nil && !st.HasBaziResult() {
		return o.buildQimenKnowledgeQuery(currentQuestion(st), qimenData)
	}
	question := currentQuestion(st)

	var terms []string

	if st.BaziResult == nil {
		if question != "" {
			return "八字 命理 " + question
		}
		return "八字 命理 基础"
	}

	dayGan, _ := st.BaziResult["dayGan"].(string)
	stemWx := map[string]string{"甲":"木","乙":"木","丙":"火","丁":"火","戊":"土","己":"土","庚":"金","辛":"金","壬":"水","癸":"水"}
	dayWx := stemWx[dayGan]

	// Core identity
	terms = append(terms, dayGan+"日主", dayWx+"命")

	// ---- extract SPECIFIC pattern signals from the chart ----

	// Collect pillar data
	type pillarData struct{ stem, branch, shiShen string }
	var pd [4]pillarData
	if pillars, ok := st.BaziResult["pillars"].([]map[string]any); ok {
		for i, p := range pillars {
			if i >= 4 {
				break
			}
			pd[i].stem, _ = p["stem"].(string)
			pd[i].branch, _ = p["branch"].(string)
			pd[i].shiShen, _ = p["shiShen"].(string)
		}
	}

	// 格局 signals (most diagnostic for retrieval)
	shiShenSet := map[string]bool{}
	for _, p := range pd {
		if p.shiShen != "" {
			shiShenSet[p.shiShen] = true
		}
	}
	if shiShenSet["正官"] && shiShenSet["七杀"] {
		terms = append(terms, "官杀混杂")
	}
	if shiShenSet["伤官"] && shiShenSet["正官"] {
		terms = append(terms, "伤官见官")
	}
	if shiShenSet["食神"] && shiShenSet["七杀"] {
		terms = append(terms, "食神制杀")
	}
	if shiShenSet["正印"] || shiShenSet["偏印"] {
		terms = append(terms, "印星")
	}
	if shiShenSet["正财"] || shiShenSet["偏财"] {
		terms = append(terms, "财星")
	}

	// 调候: day-gan + month-branch
	monthBranch := pd[1].branch
	if dayGan != "" && monthBranch != "" {
		// seasonal adjustment: 巳午未→夏需水, 亥子丑→冬需火
		seasonZhi := map[string]string{"巳":"夏","午":"夏","未":"夏","亥":"冬","子":"冬","丑":"冬"}
		if s, ok := seasonZhi[monthBranch]; ok {
			terms = append(terms, monthBranch+"月"+dayGan, s+"季调候")
		}
	}

	// 冲 signal
	chongPairs := map[string]string{"子":"午","午":"子","丑":"未","未":"丑","寅":"申","申":"寅","卯":"酉","酉":"卯","辰":"戌","戌":"辰","巳":"亥","亥":"巳"}
	branches := []string{pd[0].branch, pd[1].branch, pd[2].branch, pd[3].branch}
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			if chongPairs[branches[i]] == branches[j] {
				terms = append(terms, branches[i]+branches[j]+"冲")
			}
		}
	}

	// Five element imbalance
	if wx, ok := st.BaziResult["wuxing"].(map[string]int); ok {
		for k, v := range wx {
			if v >= 3 {
				terms = append(terms, k+"旺")
			} else if v == 0 {
				terms = append(terms, "缺"+k)
			}
		}
	}

	// User question keywords
	if question != "" {
		keywords := o.extractSearchKeywords(ctx, question)
		if keywords != "" {
			terms = append(terms, keywords)
		}
	}

	query := "八字 " + strings.Join(terms, " ")
	if len(query) > 200 {
		query = query[:200]
	}
	return query
}

// extractSearchKeywords uses LLM to extract concise search keywords from user question
func (o *Orchestrator) extractSearchKeywords(ctx context.Context, question string) string {
	prompt := "将用户问题提炼为3-5个搜索关键词，用空格分隔。只返回关键词，不要任何解释。问题：" + question
	messages := []llm.Message{{Role: "user", Content: question}}
	resp, _, err := o.llm.Generate(ctx, prompt, messages)
	if err != nil {
		return question
	}
	keywords := strings.TrimSpace(resp)
	if len(keywords) > 80 {
		keywords = keywords[:80]
	}
	return keywords
}

func currentQuestion(st *state.SessionState) string {
	if st.LastUserQuestion != "" {
		return st.LastUserQuestion
	}
	return "请先给出一段简明的命盘总评。"
}

// selectPrompt chooses a specialized prompt based on the question category.
// When o.promptMode is "direct", it bypasses keyword routing and uses a testing-oriented
// prompt that prioritizes definitive answers over hedged consulting language.
//
// TODO: direct mode is temporarily exposed via PROMPT_MODE=direct env var for benchmark testing.
// Once the test harness is stable, this should be gated behind an internal flag or removed.
func (o *Orchestrator) selectPrompt(st *state.SessionState, qimenPrimary bool) []byte {
	// ---- qimen primary: standalone qimen prompt when no bazi chart ----
	if qimenPrimary && !st.HasBaziResult() {
		tpl, err := os.ReadFile("prompts/qimen.md")
		if err == nil {
			return tpl
		}
	}
	// ---- direct mode: benchmark-oriented, no hedges ----
	if o.promptMode == "direct" {
		tpl, err := os.ReadFile("prompts/direct.md")
		if err == nil {
			return tpl
		}
	}
	// ---- soft mode: user-facing, warm and restrained ----

	question := currentQuestion(st)
	var promptFile string

	// Year-specific forensic routing: questions about a specific year or "what happened"
	// use the forensic prompt which enforces definitive judgment over hedged consulting.
	if yearEventRe.MatchString(question) {
		tpl, err := os.ReadFile("prompts/forensic.md")
		if err == nil {
			return tpl
		}
	}

	// Diagnostic routing: when a chart already exists, "how is X" questions
	// are forensic diagnostics about a specific chart, not personal consulting.
	// Route them to forensic instead of warm consulting prompts.
	if st.HasBaziResult() && diagnosticRe.MatchString(question) {
		tpl, err := os.ReadFile("prompts/forensic.md")
		if err == nil {
			return tpl
		}
	}

	// Keyword-based routing to specialized prompts (user-facing consulting)
	kwMap := map[string]string{
		"prompts/marriage.md":    "结婚|婚姻|感情|恋爱|配偶|夫妻|离婚|男友|女友|丈夫|妻子|单身|桃花|拍拖",
		"prompts/career.md":      "事业|工作|职业|财运|工资|收入|生意|创业|老板|公司|升职|跳槽|投资",
		"prompts/health.md":      "健康|病|疾病|手术|身体|住院|抑郁|癌症|死|伤|痛",
		"prompts/personality.md": "性格|个性|脾气|人品|外貌|身材|长相",
	}

	for file, keywords := range kwMap {
		for _, kw := range strings.Split(keywords, "|") {
			if strings.Contains(question, kw) {
				tpl, err := os.ReadFile(file)
				if err == nil {
					return tpl
				}
				break
			}
		}
		if promptFile != "" {
			break
		}
	}

	// Fallback to generic prompt
	tpl, err := os.ReadFile("prompts/interpret.md")
	if err != nil {
		return []byte("你是精通八字命理的中文咨询师。请基于给定命盘和问题作答，不要暴露推理过程。")
	}
	return tpl
}

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

	// cleanSourceName converts a knowledge source slug to a readable book/chapter name.
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

	// Format passages as a readable reference list, not JSON
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

### 出生资料
` + string(profileJSON) + `

### 命盘结果
` + string(baziJSON) + `
` + refBlock

	// Inject qimen result if available
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

// buildQimenKnowledgeQuery builds a knowledge search query from qimen chart data.
func (o *Orchestrator) buildQimenKnowledgeQuery(question string, qimenData map[string]any) string {
	var terms []string
	terms = append(terms, "奇门遁甲")

	if dutyStar, ok := qimenData["value_star"].(string); ok && dutyStar != "" {
		terms = append(terms, dutyStar)
	}
	if dutyDoor, ok := qimenData["value_door"].(string); ok && dutyDoor != "" {
		terms = append(terms, dutyDoor)
	}
	if juText, ok := qimenData["ju_text"].(string); ok && juText != "" {
		terms = append(terms, juText)
	}
	if question != "" {
		terms = append(terms, question)
	}

	query := strings.Join(terms, " ")
	if len(query) > 200 {
		query = query[:200]
	}
	return query
}

