package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Builder 负责构建知识检索查询和解读 prompt。
type Builder struct {
	
	promptMode string
}

// NewBuilder 创建 prompt builder。
func NewBuilder(promptMode string) *Builder {
	return &Builder{promptMode: promptMode}
}

// CurrentQuestion 返回当前轮应该送给模型的用户问题。
func CurrentQuestion(st *state.SessionState) string {
	if st.LastUserQuestion != "" {
		return st.LastUserQuestion
	}
	return "请先给出一段简明的命盘总评。"
}


// cleanSourceName 将知识库来源标识转换为可读的中文书名。
func cleanSourceName(raw string) string {
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
// BuildInterpretPrompt 组装完整的 LLM 系统提示词用于命盘解读。
func (b *Builder) BuildInterpretPrompt(st *state.SessionState, passages []mcp.Passage, primaryDomain string) string {
	tpl := b.selectPrompt(st, primaryDomain)
	refBlock := b.buildReferenceBlock(passages)

	prompt := string(tpl) + `

## 运行时上下文

当前日期：` + time.Now().Format("2006-01-02") + `

### 出生资料
` + b.buildProfileSection(st) + `
` + b.buildChartSection(st, primaryDomain) + `
` + refBlock + b.buildAnswerGuide(st, primaryDomain)

	prompt += b.buildHistoryContext(st)

	prompt += `
### 当前问题
` + CurrentQuestion(st)
	return prompt
}



// buildReferenceBlock 构建知识库参考资料文本块。
func (b *Builder) buildReferenceBlock(passages []mcp.Passage) string {
	if len(passages) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n### 参考资料（知识库检索结果）\n\n以下是从知识库中检索到的命理典籍和相关资料。**你必须**在关键论断中引用这些资料——直接引用原文并标注出处，不要只用自己的话复述。\n\n")
	for i, p := range passages {
		name := cleanSourceName(p.Source)
		content := strings.TrimSpace(p.Content)
		if len(content) > 300 {
			content = content[:300] + "…"
		}
		sb.WriteString(fmt.Sprintf("%d. **%s**\n   %s\n\n", i+1, name, content))
	}
	sb.WriteString("**引用要求：** 上述资料中如果有与当前命盘特征直接对应的原文，你必须在分析中引用 1-2 条。引用时使用资料中标注的书名，格式如：\"《渊海子平》云：'……'\" 或 \"据《格局论命》记载：'……'\"。引用后用自己的话解释一句。禁止把参考资料当背景默默使用——要么引用，要么不用。\n\n")
	return sb.String()
}

// buildHistoryContext 构建历史摘要和最近对话文本。
func (b *Builder) buildHistoryContext(st *state.SessionState) string {
	var sb strings.Builder
	if st.RunningSummary != "" {
		sb.WriteString("\n### 历史摘要\n\n以下是此前对话的压缩摘要，包含了已确认的资料、问题主线和已给出的关键结论。请在回答时结合这些上下文：\n\n")
		sb.WriteString(st.RunningSummary)
		sb.WriteString("\n")
	}
	if len(st.RecentTurns) > 0 {
		recentCount := len(st.RecentTurns)
		if recentCount > 4 {
			recentCount = 4
		}
		recent := st.RecentTurns[len(st.RecentTurns)-recentCount:]
		sb.WriteString("\n### 最近对话\n")
		for _, t := range recent {
			role := "用户"
			if t.Role == "assistant" {
				role = "助手"
			}
			content := t.Content
			if len(content) > 300 {
				content = content[:300] + "...\n"
			}
			sb.WriteString(role + "：" + content + "\n\n")
		}
	}
	return sb.String()
}

// BuildKnowledgeQuery 根据会话状态构建知识搜索查询。
func (b *Builder) BuildKnowledgeQuery(ctx context.Context, st *state.SessionState, primaryDomain string) string {
	if primaryDomain == "qimen" && !st.HasBaziResult() && st.QimenResult != nil {
		return b.BuildQimenKnowledgeQuery(CurrentQuestion(st), st.QimenResult)
	}

	if primaryDomain == "ziwei" && st.ZiWeiResult != nil {
		return b.buildZiweiKnowledgeQuery(CurrentQuestion(st), st.ZiWeiResult)
	}
	question := CurrentQuestion(st)

	var terms []string
	if st.BaziResult == nil {
		if question != "" {
			return "八字 命理 " + question
		}
		return "八字 命理 基础"
	}

	dayGan, _ := st.BaziResult["dayGan"].(string)
	stemWx := map[string]string{"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土", "己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水"}
	dayWx := stemWx[dayGan]
	terms = append(terms, dayGan+"日主", dayWx+"命")

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

	monthBranch := pd[1].branch
	if dayGan != "" && monthBranch != "" {
		seasonZhi := map[string]string{"巳": "夏", "午": "夏", "未": "夏", "亥": "冬", "子": "冬", "丑": "冬"}
		if s, ok := seasonZhi[monthBranch]; ok {
			terms = append(terms, monthBranch+"月"+dayGan, s+"季调候")
		}
	}

	chongPairs := map[string]string{"子": "午", "午": "子", "丑": "未", "未": "丑", "寅": "申", "申": "寅", "卯": "酉", "酉": "卯", "辰": "戌", "戌": "辰", "巳": "亥", "亥": "巳"}
	branches := []string{pd[0].branch, pd[1].branch, pd[2].branch, pd[3].branch}
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			if chongPairs[branches[i]] == branches[j] {
				terms = append(terms, branches[i]+branches[j]+"冲")
			}
		}
	}

	if wx, ok := st.BaziResult["wuxing"].(map[string]int); ok {
		for k, v := range wx {
			if v >= 3 {
				terms = append(terms, k+"旺")
			} else if v == 0 {
				terms = append(terms, "缺"+k)
			}
		}
	}

	if question != "" {
		keywords := b.extractSearchKeywords(ctx, question, dayGan+"日主"+dayWx+"命")
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

// BuildQimenKnowledgeQuery 根据奇门盘数据构建知识搜索查询。
func (b *Builder) BuildQimenKnowledgeQuery(question string, qimenData map[string]any) string {
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

// buildZiweiKnowledgeQuery 从紫微斗数命盘结果中提取关键词构建知识搜索查询。
func (b *Builder) buildZiweiKnowledgeQuery(question string, result map[string]any) string {
	var terms []string
	terms = append(terms, "紫微斗数")

	// 提取命宫主星名
	if palaces, ok := result["palaces"].([]any); ok {
		for _, p := range palaces {
			if palace, ok := p.(map[string]any); ok {
				name, _ := palace["name"].(string)
				if name != "命宫" {
					continue
				}
				if stars, ok := palace["major_stars"].([]any); ok {
					for _, s := range stars {
						if star, ok := s.(map[string]any); ok {
							if starName, ok := star["name"].(string); ok && starName != "" {
								terms = append(terms, starName)
							}
						}
					}
				}
			}
		}
	}

	// 提取四化信息
	if fourPillars, ok := result["four_pillars"].(map[string]any); ok {
		if yearPillar, ok := fourPillars["年柱"].(string); ok && len(yearPillar) >= 2 {
			yearStem := string([]rune(yearPillar)[0])
			terms = append(terms, yearStem+"年四化")
		}
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

// BuildAgentInstruction 为 ADK ChatModelAgent 构建系统指令。
func (b *Builder) BuildAgentInstruction(st *state.SessionState, primaryDomain string) string {
	tpl := b.selectPrompt(st, primaryDomain)

	prompt := string(tpl) + `

## 运行时上下文

当前日期：` + time.Now().Format("2006-01-02") + `

### 出生资料
` + b.buildProfileSection(st) + `
` + b.buildChartSection(st, primaryDomain) + `

### 可用工具及执行建议

你拥有以下工具，请根据当前需求选择合适的工具并按合适的顺序调用：

1. **bazi_calc** — 根据出生时间排八字四柱命盘。需要完整的年/月/日/时/性别。输出 JSON 含 pillars、dayGan、wuxing、dayun 等。
2. **yongshen** — 分析日主强弱、取用神忌神。需要先调用 bazi_calc 获得日主和月令。
3. **dayun_analyzer** — 分析大运走势、各步大运的起止时间。需要先调用 bazi_calc。
4. **qimen_dunjia** — 排奇门遁甲盘，分析当前时空的吉凶方位、门星神组合。
5. **ziwei_calc** — 排紫微斗数命盘，包括十二宫星曜分布、四化飞星。需要完整的出生年月日时和性别。
6. **knowledge_search** — 从命理古籍知识库（含《渊海子平》《滴天髓》《子平真诠》等）中检索相关原文。建议在给出关键论断前先调用此工具获取经典依据。

### 执行建议

- **八字分析流程：** 先 bazi_calc 排盘，然后可调用 yongshen/dayun_analyzer 补充分析，再用 knowledge_search 查古籍，最后综合解读。
- **奇门分析流程：** 先 qimen_dunjia 排盘，用 knowledge_search 查相关典籍，最后分析时空吉凶。
- **紫微分析流程：** 先 ziwei_calc 排盘，用 knowledge_search 查相关典籍，最后解读命盘。
- 你可以根据实际情况调整顺序或跳过不必要的步骤。
- 如果上方「出生资料」和「命盘结果」已提供，直接使用，严禁再次索要出生信息或重新调用 bazi_calc`

	if st.RunningSummary != "" {
		prompt += "\n### 会话摘要\n\n" + st.RunningSummary + "\n"
	}
	return prompt
}

func (b *Builder) extractSearchKeywords(ctx context.Context, question string, chartContext string) string {
	_ = ctx
	_ = chartContext
	return question
}

func (b *Builder) selectPrompt(st *state.SessionState, primaryDomain string) []byte {
	if primaryDomain == "qimen" && !st.HasBaziResult() {
		return readFile("prompts/qimen.md")
	}
	if b.promptMode == "direct" {
		return readFile("prompts/direct.md")
	}
	base := readFile("prompts/interpret.md")
	snippet := b.pickSnippet(st)
	if len(snippet) == 0 {
		return base
	}
	return []byte(strings.Replace(string(base), "<!-- TASK_BLOCK -->", string(snippet), 1))
}

func (b *Builder) pickSnippet(st *state.SessionState) []byte {
	if st.Routing.TaskIntent == "timing_followup" && isYearScope(st.Routing.TimeScope) {
		return readFile("prompts/snippets/year_event.md")
	}
	if st.Routing.TaskIntent == "fortune_followup" || st.Routing.TaskIntent == "interpret_chart" {
		return readFile("prompts/snippets/fortune.md")
	}
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

func readFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return b
}

var domainLabels = map[string]string{
	"bazi":  "八字命盘",
	"qimen": "奇门遁甲",
	"ziwei": "紫微斗数",
}

// buildProfileSection 构建出生资料文本。同时给出原始出生时间和系统已计算的权威四柱。
func (b *Builder) buildProfileSection(st *state.SessionState) string {
	year, _ := st.Profile["year"]
	month, _ := st.Profile["month"]
	day, _ := st.Profile["day"]
	hour, _ := st.Profile["hour"]
	minute, _ := st.Profile["minute"]
	gender, _ := st.Profile["gender"].(string)
	birthplace, _ := st.Profile["birthplace"].(string)

	var lines []string

	// 原始出生时间
	timeStr := fmt.Sprintf("%v年%v月%v日%v时", year, month, day, hour)
	if minute != nil {
		timeStr = fmt.Sprintf("%v年%v月%v日%v:%v", year, month, day, hour, minute)
	}
	meta := []string{timeStr}
	if gender != "" {
		meta = append(meta, gender)
	}
	if birthplace != "" {
		meta = append(meta, birthplace)
	}
	lines = append(lines, "出生时间："+strings.Join(meta, "，"))

	// 系统计算的权威四柱（已做真太阳时校正和晚子时处理）
	if st.HasBaziResult() {
		if pillars, ok := st.BaziResult["pillars"].([]map[string]any); ok && len(pillars) == 4 {
			lines = append(lines, fmt.Sprintf(
				"**系统排盘结果（权威，你必须使用此结果，不得自行推算）：%s%s年 %s%s月 %s%s日 %s%s时**",
				pillars[0]["stem"], pillars[0]["branch"],
				pillars[1]["stem"], pillars[1]["branch"],
				pillars[2]["stem"], pillars[2]["branch"],
				pillars[3]["stem"], pillars[3]["branch"],
			))
		}
		if dayGan, ok := st.BaziResult["dayGan"].(string); ok && dayGan != "" {
			lines = append(lines, fmt.Sprintf("日主%s", dayGan))
		}
		if birthday, ok := st.BaziResult["birthday"].(string); ok && birthday != "" {
			lines = append(lines, fmt.Sprintf("（系统校正后时间：%s）", birthday))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func (b *Builder) buildChartSection(st *state.SessionState, primaryDomain string) string {
	var sb strings.Builder
	primaryJSON := b.marshalDomainResult(st, primaryDomain)
	if primaryJSON != nil {
		sb.WriteString(fmt.Sprintf("### 命盘结果（%s）\n\n%s\n", domainLabels[primaryDomain], primaryJSON))
	}
	for _, domain := range []string{"bazi", "qimen", "ziwei"} {
		if domain == primaryDomain {
			continue
		}
		secondaryJSON := b.marshalDomainResult(st, domain)
		if secondaryJSON != nil {
			sb.WriteString(fmt.Sprintf("### 辅助参考（%s）\n\n%s\n", domainLabels[domain], secondaryJSON))
		}
	}
	return sb.String()
}

func (b *Builder) marshalDomainResult(st *state.SessionState, domain string) []byte {
	var data map[string]any
	switch domain {
	case "bazi":
		data = st.BaziResult
	case "qimen":
		data = st.QimenResult
	case "ziwei":
		data = st.ZiWeiResult
	}
	if data == nil {
		return nil
	}
	js, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return js
}

func (b *Builder) buildAnswerGuide(st *state.SessionState, primaryDomain string) string {
	switch primaryDomain {
	case "qimen":
		if st.HasBaziResult() {
			return `
**回答要求：**

1. 先引用八字命盘中的日主、五行偏旺、用神忌神做 1-2 句个人背景铺垫
2. 再结合奇门盘中的门、星、神、宫位分析当前时机吉凶
3. 最后给出基于两者综合的具体建议

八字看先天格局与长期趋势，奇门看当前时空窗口是否有利。两者缺一不可。
`
		}
		return `
**回答要求：**

1. 以奇门盘中的门、星、神、宫位为核心，分析当前时机的吉凶趋势
2. 结合问卜时间所属的节气、局数，给出具体的时机判断和建议
3. 说明当前时空窗口对问卜事项的有利或不利因素

本次分析以奇门遁甲为主。
`
	case "ziwei":
		if st.HasBaziResult() {
			return `
**回答要求：**

1. 先以紫微斗数命盘为主，分析命宫、身宫、三方四正的星曜组合
2. 再结合八字命盘中的日主和格局做交叉验证
3. 两者结论一致的论断可做重点强调，不一致的需说明各自由来
`
		}
		return `
**回答要求：**

1. 以紫微斗数命盘为主，分析命宫、身宫、三方四正的星曜组合
2. 结合大限、流年判断运势起伏的时间节点
3. 重点关注命宫主星和四化飞星的吉凶应期
`
	default:
		return ""
	}
}
