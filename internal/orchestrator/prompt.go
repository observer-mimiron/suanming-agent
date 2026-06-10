package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

func buildKnowledgeQuery(st *state.SessionState) string {
	question := currentQuestion(st)
	baziJSON, err := json.Marshal(st.BaziResult)
	if err != nil {
		baziJSON = []byte("{}")
	}
	return fmt.Sprintf(
		"命盘：%s；问题：%s；请检索与此问题最相关的命理规则、典籍原文和解释。",
		baziJSON,
		question,
	)
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
