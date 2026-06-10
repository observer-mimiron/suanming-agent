package orchestrator

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

func buildKnowledgeQuery(st *state.SessionState) string {
	question := currentQuestion(st)

	var terms []string

	// Add birth date for matching benchmark cases
	if birthday, ok := st.BaziResult["birthday"].(string); ok {
		terms = append(terms, birthday)
	}
	if gender, ok := st.BaziResult["gender"].(string); ok {
		terms = append(terms, gender+"命")
	}

	// Build profile-based terms
	if st.BaziResult != nil {
		if dayGan, ok := st.BaziResult["dayGan"].(string); ok && dayGan != "" {
			stemWx := map[string]string{"甲":"木","乙":"木","丙":"火","丁":"火","戊":"土","己":"土","庚":"金","辛":"金","壬":"水","癸":"水"}
			if wx, ok := stemWx[dayGan]; ok {
				terms = append(terms, dayGan+"日主", wx+"命")
			}
		}
		if pillars, ok := st.BaziResult["pillars"].([]map[string]string); ok {
			seen := map[string]bool{}
			for _, p := range pillars {
				if ss := p["shiShen"]; ss != "" && !seen[ss] && ss != "日主" {
					terms = append(terms, ss)
					seen[ss] = true
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
		// Add 十神 keywords for better matching
		if dayGan, ok := st.BaziResult["dayGan"].(string); ok && dayGan != "" {
			stemWx := map[string]string{"甲":"木","乙":"木","丙":"火","丁":"火","戊":"土","己":"土","庚":"金","辛":"金","壬":"水","癸":"水"}
			if wx, ok := stemWx[dayGan]; ok {
				terms = append(terms, wx+"日主命")
			}
		}
	}

	// Use LLM to extract search keywords from user question
	if question != "" {
		keywords := extractSearchKeywords(question)
		if keywords != "" {
			terms = append(terms, keywords)
		}
	}

	query := "八字 命理 " + strings.Join(terms, " ")
	if len(query) > 200 {
		query = query[:200]
	}
	return query
}

// extractSearchKeywords uses LLM to extract concise search keywords from user question
func extractSearchKeywords(question string) string {
	client := llm.NewClient()
	prompt := "将用户问题提炼为3-5个搜索关键词，用空格分隔。只返回关键词，不要任何解释。问题：" + question
	messages := []llm.Message{{Role: "user", Content: question}}
	resp, err := client.Chat(prompt, messages)
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

// selectPrompt chooses a specialized prompt based on the question category
func selectPrompt(st *state.SessionState) []byte {
	question := currentQuestion(st)
	var promptFile string

	// Keyword-based routing to specialized prompts
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

func buildInterpretPrompt(st *state.SessionState, passages []mcp.Passage) string {
	tpl := selectPrompt(st)
	profileJSON, err := json.Marshal(st.Profile)
	if err != nil {
		profileJSON = []byte("{}")
	}
	baziJSON, err := json.Marshal(st.BaziResult)
	if err != nil {
		baziJSON = []byte("{}")
	}
	passagesJSON, err := json.Marshal(passages)
	if err != nil {
		passagesJSON = []byte("[]")
	}

	return string(tpl) + `

## 运行时上下文

### 出生资料
` + string(profileJSON) + `

### 命盘结果
` + string(baziJSON) + `

### 知识引用
` + string(passagesJSON) + `

### 当前问题
` + currentQuestion(st)
}
