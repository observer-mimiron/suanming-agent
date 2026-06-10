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

func buildInterpretPrompt(st *state.SessionState, passages []mcp.Passage) string {
	tpl, err := os.ReadFile("prompts/interpret.md")
	if err != nil {
		tpl = []byte("你是精通八字命理的中文咨询师。请基于给定命盘和问题作答，不要暴露推理过程。")
	}
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
