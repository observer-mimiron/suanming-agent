package orchestrator

import (
	"context"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// buildKnowledgeQuery 根据会话状态构建知识搜索查询。
// 当存在八字命盘数据时，查询由日主、十神关系、调候信号、五行偏旺和用户问题关键词构成。
// 当有奇门数据但没有八字命盘时，查询改为针对奇门术语。
func (o *Orchestrator) buildKnowledgeQuery(ctx context.Context, st *state.SessionState, qimenData map[string]any) string {
	// 当存在奇门数据且无八字命盘时，搜索奇门术语。
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
	stemWx := map[string]string{"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土", "己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水"}
	dayWx := stemWx[dayGan]

	// 核心身份标识
	terms = append(terms, dayGan+"日主", dayWx+"命")

	// ---- 从命盘中提取具体的格局信号 ----

	// 收集四柱数据
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

	// 格局信号（对检索最具诊断价值）
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

	// 调候：日干 + 月支
	monthBranch := pd[1].branch
	if dayGan != "" && monthBranch != "" {
		// 季节调候：巳午未→夏需水，亥子丑→冬需火
		seasonZhi := map[string]string{"巳": "夏", "午": "夏", "未": "夏", "亥": "冬", "子": "冬", "丑": "冬"}
		if s, ok := seasonZhi[monthBranch]; ok {
			terms = append(terms, monthBranch+"月"+dayGan, s+"季调候")
		}
	}

	// 冲刑信号
	chongPairs := map[string]string{"子": "午", "午": "子", "丑": "未", "未": "丑", "寅": "申", "申": "寅", "卯": "酉", "酉": "卯", "辰": "戌", "戌": "辰", "巳": "亥", "亥": "巳"}
	branches := []string{pd[0].branch, pd[1].branch, pd[2].branch, pd[3].branch}
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			if chongPairs[branches[i]] == branches[j] {
				terms = append(terms, branches[i]+branches[j]+"冲")
			}
		}
	}

	// 五行偏旺
	if wx, ok := st.BaziResult["wuxing"].(map[string]int); ok {
		for k, v := range wx {
			if v >= 3 {
				terms = append(terms, k+"旺")
			} else if v == 0 {
				terms = append(terms, "缺"+k)
			}
		}
	}

	// 用户问题关键词
	if question != "" {
		keywords := o.extractSearchKeywords(ctx, question, dayGan+"日主"+dayWx+"命")
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

// extractSearchKeywords 使用 LLM 从用户问题中提取简洁的搜索关键词。
// chartContext 提供日主身份，用于领域感知的关键词提取。
func (o *Orchestrator) extractSearchKeywords(ctx context.Context, question string, chartContext string) string {
	prompt := "用户正在咨询八字命理，命主为" + chartContext + "。根据用户当前问题，提炼3-5个需要从命理古籍中检索的关键词（如格局名、十神关系、调候要点等），用空格分隔。只返回关键词，不要任何解释。\n问题：" + question
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

// buildQimenKnowledgeQuery 根据奇门盘数据构建知识搜索查询。
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
