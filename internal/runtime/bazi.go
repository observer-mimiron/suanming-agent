package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/state"
	qimenTools "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// handleBaziInput 处理用户直接输入四柱八字的场景。
func (e *Executor) handleBaziInput(ctx context.Context, sink EventSink, st *state.SessionState, rawBazi []string) (string, error) {
	parseSpan := tracing.SpanFromContext(ctx, "parse_direct_bazi", tracing.KindChain)
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "识别到直接输入的四柱八字，开始分析...",
	}})

	pillarNames := []string{"年柱", "月柱", "日柱", "时柱"}
	pillars := make([]map[string]any, 4)
	for i := 0; i < 4; i++ {
		s := rawBazi[i]
		pillars[i] = map[string]any{
			"name":   pillarNames[i],
			"stem":   string([]rune(s)[0:1]),
			"branch": string([]rune(s)[1:2]),
		}
	}
	data := map[string]any{
		"pillars": pillars,
		"dayGan":  string([]rune(rawBazi[2])[0:1]),
	}
	st.BaziResult = data
	parseSpan.End()
	emitChartComponent(ctx, sink, "bazi-chart", data)

	if _, hasGender := st.Profile["gender"]; !hasGender {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{
			"content": "⚠️ 八字本身不含性别信息。请问这个八字是男命还是女命？（男女命的大运顺逆、婚姻用神均不同）",
		}})
	}

	return e.answerWithKnowledge(ctx, sink, st, "bazi", "解读失败: ")
}

// handleAsk 处理信息不完整时需要询问用户补充资料的场景。
func (e *Executor) handleAsk(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "正在核实出生信息...",
	}})
	missing := st.MissingFields()

	names := make([]string, len(missing))
	for i, f := range missing {
		names[i] = displayName(f)
	}
	var prompt string
	if len(names) == 0 {
		prompt = "请告诉我你的出生信息（至少需要年份、月份、日期、时辰和性别），例如：1990年5月20日早上8点，男"
	} else {
		prompt = fmt.Sprintf("请告诉我你的%s", strings.Join(names, "、"))
	}
	sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": prompt}})
	return prompt, nil
}

// handleFullReading 执行完整的八字排盘解读流程：排盘 → 用神 → 大运 → 知识检索 → LLM 解读。
func (e *Executor) handleFullReading(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "信息齐全，开始排盘...",
	}})

	data, err := e.runBaziCalc(ctx, sink, st.Profile)
	if err != nil {
		switch err.Error() {
		case "bazi_calc not registered":
			sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "tool bazi_calc not registered"}})
		case "bazi_calc result type invalid":
			sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "排盘结果格式错误"}})
		default:
			sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "排盘失败: " + err.Error()}})
		}
		return "", err
	}

	st.BaziResult = data
	e.runOptionalYongshen(ctx, sink, st)
	e.runOptionalDayun(ctx, st, data)
	emitChartComponent(ctx, sink, "bazi-chart", data)
	return e.answerWithKnowledge(ctx, sink, st, "bazi", "LLM 解读失败: ")
}

// handleFollowupReading 复用已有命盘进行追问解读，可选奇门排盘作为补充。
func (e *Executor) handleFollowupReading(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "复用已有命盘...",
	}})

	reuseSpan := tracing.SpanFromContext(ctx, "reuse_bazi_result", tracing.KindChain)
	reuseSpan.SetAttribute("bazi_reused", true)
	reuseSpan.End()

	if st.NeedsQimen {
		qimenData, _, qimenErr := e.runCurrentQimen(ctx, sink)
		if qimenErr == nil && qimenData != nil {
			if !st.HasQimenResult() {
				emitChartComponent(ctx, sink, "qimen-chart", qimenData)
			}
			st.QimenResult = qimenData
		} else if qimenErr != nil {
			sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
				"agent": "orchestrator", "text": "奇门排盘失败，改按八字继续分析。",
			}})
		}
	}

	return e.answerWithKnowledge(ctx, sink, st, "bazi", "LLM 解读失败: ")
}

// emitChartComponent 发送统一的命盘组件事件。
func emitChartComponent(ctx context.Context, sink EventSink, chartType string, payload map[string]any) {
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type": chartType, "payload": payload,
	}})
}

var fieldNames = map[string]string{
	"year": "出生年份", "month": "出生月份", "day": "出生日期",
	"hour": "出生时辰", "gender": "性别", "birthplace": "出生地（城市）",
}

func displayName(field string) string {
	if name, ok := fieldNames[field]; ok {
		return name
	}
	return field
}

// runBaziCalc 执行八字排盘工具并返回结构化命盘结果。
func (e *Executor) runBaziCalc(ctx context.Context, sink EventSink, profile map[string]any) (map[string]any, error) {
	baziSpan := tracing.SpanFromContext(ctx, "bazi_calc", tracing.KindTool)
	tool, ok := e.tools.Get("bazi_calc")
	if !ok {
		baziSpan.RecordError(fmt.Errorf("not registered"))
		baziSpan.End()
		return nil, fmt.Errorf("bazi_calc not registered")
	}

	sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{"tool": "bazi_calc", "params": profile}})
	result, err := tool.Execute(ctx, profile)
	if err != nil {
		baziSpan.RecordError(err)
		baziSpan.End()
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		baziSpan.RecordError(fmt.Errorf("result type invalid"))
		baziSpan.End()
		return nil, fmt.Errorf("bazi_calc result type invalid")
	}

	baziSpan.End()
	return data, nil
}

// runOptionalYongshen 在 yongshen 工具可用时补充用神分析。
func (e *Executor) runOptionalYongshen(ctx context.Context, sink EventSink, st *state.SessionState) {
	ysTool, ok := e.tools.Get("yongshen")
	if !ok {
		return
	}

	ysSpan := tracing.SpanFromContext(ctx, "yongshen", tracing.KindTool)
	defer ysSpan.End()

	ysResult, ysErr := ysTool.Execute(ctx, st.Profile)
	if ysErr != nil {
		ysSpan.RecordError(ysErr)
		return
	}

	if ysMap, ok := ysResult.(map[string]any); ok {
		st.BaziResult["yongshen"] = ysMap
		ysSpan.SetAttribute("day_master", ysMap["day_master"])
		ysSpan.SetAttribute("strength", ysMap["strength"])
		sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{"tool": "yongshen", "params": st.Profile}})
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator",
			"text":  fmt.Sprintf("日主%s 强弱:%s 用神:%v 忌神:%v", ysMap["day_master"], ysMap["strength"], ysMap["yong_shen"], ysMap["ji_shen"]),
		}})
	}
}

// runOptionalDayun 在大运工具可用时补充大运分析。
func (e *Executor) runOptionalDayun(ctx context.Context, st *state.SessionState, baziData map[string]any) {
	daTool, ok := e.tools.Get("dayun_analyzer")
	if !ok {
		return
	}

	daSpan := tracing.SpanFromContext(ctx, "dayun_analyzer", tracing.KindTool)
	defer daSpan.End()

	daParams := map[string]any{
		"dayun":       baziData["dayun"],
		"bazi_result": st.BaziResult,
	}
	if daResult, daErr := daTool.Execute(ctx, daParams); daErr == nil {
		if daMap, ok := daResult.(map[string]any); ok {
			st.BaziResult["dayun_analyzed"] = daMap["dayun_analyzed"]
		}
	} else {
		daSpan.RecordError(daErr)
	}
}

// runCurrentQimen 执行当前时刻的奇门排盘并返回命盘结果。
func (e *Executor) runCurrentQimen(ctx context.Context, sink EventSink) (map[string]any, bool, error) {
	qimenTool, ok := e.tools.Get("qimen_dunjia")
	if !ok {
		qmSpan := tracing.SpanFromContext(ctx, "qimen_dunjia", tracing.KindChain)
		qmSpan.SetStatus("degraded")
		qmSpan.SetAttribute("degrade_reason", "tool_not_registered")
		qmSpan.End()
		return nil, false, nil
	}

	qmSpan := tracing.SpanFromContext(ctx, "qimen_dunjia", tracing.KindTool)
	defer qmSpan.End()

	now := resolveQimenTime(time.Now())
	qimenParams := qimenTools.ResolveTime(now)
	sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
		"tool": "qimen_dunjia", "params": qimenParams,
	}})
	qimenResult, qimenErr := qimenTool.Execute(ctx, qimenParams)
	if qimenErr != nil {
		qmSpan.SetStatus("fallback")
		qmSpan.RecordError(qimenErr)
		return nil, true, qimenErr
	}

	qimenData, ok := qimenResult.(map[string]any)
	if !ok {
		qmSpan.RecordError(fmt.Errorf("result type invalid"))
		return nil, true, fmt.Errorf("qimen_dunjia result type invalid")
	}
	return qimenData, true, nil
}
