package orchestrator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/sse"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
)

type Orchestrator struct {
	tools    *tools.Registry
	sessions sync.Map
}

func New(reg *tools.Registry) *Orchestrator {
	return &Orchestrator{tools: reg}
}

func (o *Orchestrator) Run(sw sse.Sender, sessionID, message string) error {
	st := o.loadOrCreate(sessionID)

	// 1. Use LLM to classify intent and extract profile
	action, profilePatch, userQuestion := classifyAndExtract(message, st)

	switch action {
	case "new_profile":
		// User wants a new reading — clear old data and start fresh
		st.Profile = make(map[string]any)
		st.BaziResult = nil
		for k, v := range profilePatch {
			st.Profile[k] = v
		}
		if userQuestion != "" {
			st.LastUserQuestion = userQuestion
		}
		return o.handleFullReading(sw, st)

	case "update_profile":
		// User is correcting info — merge changes
		changed := st.MergeProfile(profilePatch)
		if userQuestion != "" {
			st.LastUserQuestion = userQuestion
		}
		if changed {
			st.BaziResult = nil
		}
		if !st.IsProfileComplete() {
			return o.handleAsk(sw, st)
		}
		if st.BaziResult == nil {
			return o.handleFullReading(sw, st)
		}
		return o.handleFollowupReading(sw, st)

	case "incomplete":
		// Profile still missing fields — merge and ask
		st.MergeProfile(profilePatch)
		if userQuestion != "" {
			st.LastUserQuestion = userQuestion
		}
		return o.handleAsk(sw, st)

	default: // "followup"
		// User is asking about existing profile
		if userQuestion != "" {
			st.LastUserQuestion = userQuestion
		}
		if !st.IsProfileComplete() {
			return o.handleAsk(sw, st)
		}
		if st.BaziResult == nil {
			return o.handleFullReading(sw, st)
		}
		return o.handleFollowupReading(sw, st)
	}
}

func (o *Orchestrator) loadOrCreate(id string) *state.SessionState {
	if v, ok := o.sessions.Load(id); ok {
		return v.(*state.SessionState)
	}
	s := state.NewSession(id)
	o.sessions.Store(id, s)
	return s
}

var (
	lunarHintRe = regexp.MustCompile(`农历|阴历|正月|腊月`)
	yearRe      = regexp.MustCompile(`(\d{4})\s*年`)
	monthRe     = regexp.MustCompile(`(\d{1,2})\s*月`)
	dayRe       = regexp.MustCompile(`(\d{1,2})\s*[日号]`)
	morningRe   = regexp.MustCompile(`(?:早上|上午|早晨)\s*(\d{1,2})\s*点`)
	noonRe      = regexp.MustCompile(`中午\s*(\d{1,2})\s*点`)
	pmRe        = regexp.MustCompile(`(?:下午|晚上)\s*(\d{1,2})\s*点`)
	clockRe     = regexp.MustCompile(`(\d{1,2})(?::00|\s*[点时])`)
	timeRe      = regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	genderRe    = regexp.MustCompile(`(?:性别[:：]?\s*)?(男|女)`)
)

// extractProfileAndQuestion 提取出生资料，并把剩余文本视为用户真正的问题
func extractProfileAndQuestion(msg string) (profilePatch map[string]any, question string) {
	normalized := strings.TrimSpace(msg)
	residual := normalized
	patch := map[string]any{"calendar_type": "solar"}
	if lunarHintRe.MatchString(normalized) {
		patch["calendar_type"] = "lunar"
	}

	extractInt := func(re *regexp.Regexp, key string, min, max int) {
		matches := re.FindStringSubmatch(normalized)
		if len(matches) != 2 {
			return
		}
		v, err := strconv.Atoi(matches[1])
		if err != nil || v < min || v > max {
			return
		}
		patch[key] = float64(v)
		residual = strings.Replace(residual, matches[0], "", 1)
	}

	extractInt(yearRe, "year", 1900, 2100)
	extractInt(monthRe, "month", 1, 12)
	extractInt(dayRe, "day", 1, 31)

	hourPatterns := []struct {
		re   *regexp.Regexp
		base int
	}{
		{morningRe, 0},
		{noonRe, 0},
		{pmRe, 12},
		{clockRe, 0},
	}
	for _, hp := range hourPatterns {
		matches := hp.re.FindStringSubmatch(normalized)
		if len(matches) != 2 {
			continue
		}
		h, err := strconv.Atoi(matches[1])
		if err != nil {
			break
		}
		val := h + hp.base
		if hp.base == 12 && h == 12 {
			val = 12
		}
		if val >= 0 && val <= 23 {
			patch["hour"] = float64(val)
			residual = strings.Replace(residual, matches[0], "", 1)
		}
		break
	}

	if matches := genderRe.FindStringSubmatch(normalized); len(matches) == 2 {
		patch["gender"] = matches[1]
		residual = strings.Replace(residual, matches[0], "", 1)
	}

	residual = strings.NewReplacer(
		"，", " ", ",", " ", "。", " ", "；", " ", "！", " ", "？", " ",
	).Replace(residual)
	residual = strings.Join(strings.Fields(residual), " ")
	return patch, residual
}

var fieldNames = map[string]string{
	"year": "出生年份", "month": "出生月份", "day": "出生日期",
	"hour": "出生时辰", "gender": "性别",
}

func (o *Orchestrator) handleAsk(sw sse.Sender, st *state.SessionState) error {
	sw.Send("thinking", map[string]any{
		"agent": "orchestrator", "text": "正在核实出生信息...",
	})
	missing := st.MissingFields()
	names := make([]string, len(missing))
	for i, f := range missing {
		names[i] = fieldNames[f]
	}
	sw.Send("text", map[string]any{
		"content": fmt.Sprintf("请告诉我你的%s", strings.Join(names, "、")),
	})
	sw.Send("done", map[string]any{})
	return nil
}

func (o *Orchestrator) runKnowledgeSearch(sw sse.Sender, st *state.SessionState) []mcp.Passage {
	tool, ok := o.tools.Get("knowledge_search")
	if !ok {
		sw.Send("thinking", map[string]any{
			"agent": "orchestrator", "text": "知识检索未注册，跳过引用检索。",
		})
		return []mcp.Passage{}
	}

	query := buildKnowledgeQuery(st)
	sw.Send("tool_call", map[string]any{
		"tool":   "knowledge_search",
		"params": map[string]any{"query": query, "topK": 3},
	})

	result, err := tool.Execute(map[string]any{"query": query, "topK": 3})
	if err != nil {
		sw.Send("thinking", map[string]any{
			"agent": "orchestrator", "text": "知识检索失败，继续直接解读命盘。",
		})
		return []mcp.Passage{}
	}

	payload, ok := result.(map[string]any)
	if !ok {
		return []mcp.Passage{}
	}
	passages, _ := payload["passages"].([]mcp.Passage)
	if len(passages) > 0 {
		sw.Send("component", map[string]any{
			"type":    "knowledge-sources",
			"payload": passages,
		})
	}
	return passages
}

func (o *Orchestrator) streamInterpretation(sw sse.Sender, st *state.SessionState, passages []mcp.Passage) error {
	client := llm.NewClient()
	systemPrompt := buildInterpretPrompt(st, passages)
	messages := []llm.Message{
		{Role: "user", Content: currentQuestion(st)},
	}

	return client.ChatStream(systemPrompt, messages, func(chunk string) {
		sw.Send("text", map[string]any{"content": chunk})
	})
}

func (o *Orchestrator) handleFullReading(sw sse.Sender, st *state.SessionState) error {
	sw.Send("thinking", map[string]any{
		"agent": "orchestrator", "text": "信息齐全，开始排盘...",
	})

	// 1. bazi_calc
	tool, ok := o.tools.Get("bazi_calc")
	if !ok {
		sw.Send("error", map[string]any{"message": "tool bazi_calc not registered"})
		sw.Send("done", map[string]any{})
		return fmt.Errorf("bazi_calc not registered")
	}
	sw.Send("tool_call", map[string]any{"tool": "bazi_calc", "params": st.Profile})
	result, err := tool.Execute(st.Profile)
	if err != nil {
		sw.Send("error", map[string]any{"message": "排盘失败: " + err.Error()})
		sw.Send("done", map[string]any{})
		return err
	}
	data, ok := result.(map[string]any)
	if !ok {
		sw.Send("error", map[string]any{"message": "排盘结果格式错误"})
		sw.Send("done", map[string]any{})
		return fmt.Errorf("bazi_calc result type invalid")
	}
	st.BaziResult = data

	// Run yongshen analysis
	if ysTool, ok := o.tools.Get("yongshen"); ok {
		ysResult, ysErr := ysTool.Execute(st.Profile)
		if ysErr == nil {
			if ysMap, ok2 := ysResult.(map[string]any); ok2 {
				st.BaziResult["yongshen"] = ysMap
				sw.Send("tool_call", map[string]any{"tool": "yongshen", "params": st.Profile})
				sw.Send("thinking", map[string]any{
					"agent": "orchestrator",
					"text":  fmt.Sprintf("日主%s 强弱:%s 用神:%v 忌神:%v", ysMap["day_master"], ysMap["strength"], ysMap["yong_shen"], ysMap["ji_shen"]),
				})
			}
		}
	}

	sw.Send("component", map[string]any{"type": "bazi-chart", "payload": data})

	// 2. 知识检索 + LLM 解读
	passages := o.runKnowledgeSearch(sw, st)
	if err := o.streamInterpretation(sw, st, passages); err != nil {
		sw.Send("error", map[string]any{"message": "LLM 解读失败: " + err.Error()})
		sw.Send("done", map[string]any{})
		return err
	}
	sw.Send("done", map[string]any{})
	return nil
}

func (o *Orchestrator) handleFollowupReading(sw sse.Sender, st *state.SessionState) error {
	sw.Send("thinking", map[string]any{
		"agent": "orchestrator", "text": "复用已有命盘，正在检索知识库...",
	})
	passages := o.runKnowledgeSearch(sw, st)
	if err := o.streamInterpretation(sw, st, passages); err != nil {
		sw.Send("error", map[string]any{"message": "LLM 解读失败: " + err.Error()})
		sw.Send("done", map[string]any{})
		return err
	}
	sw.Send("done", map[string]any{})
	return nil
}
