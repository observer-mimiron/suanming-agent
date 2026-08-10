// 本文件属于 runtime 执行层，负责按 ExecutionPlan 预填充确定性领域资产。
// 它只负责排盘、缓存复用和 session/vals 写入，不负责路由、Graph 拓扑或最终答复。
package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// prefill 按领域预执行可安全复用的确定性排盘工具链。所有领域的排盘由 Go 确定性执行，结果注入 vals 和 session state，LLM 不再接触排盘工具。
func (e *Executor) prefill(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) {
	span := tracing.SpanFromContext(ctx, "prefill", tracing.KindChain)
	span.SetAttribute("domain", plan.Route.PrimaryDomain)
	executed := false
	defer func() {
		span.SetAttribute("executed", executed)
		span.End()
	}()

	for _, requirement := range plan.Requirements {
		switch requirement.Kind {
		case artifactBaziChart:
			if e.prefillBaziForPlan(ctx, sink, st, plan, vals) {
				executed = true
			}
		case artifactQimenChart:
			if e.prefillQimen(ctx, sink, st, plan, vals) {
				executed = true
			}
		case artifactZiweiChart:
			if e.prefillZiWeiForPlan(ctx, sink, st, plan, vals) {
				executed = true
			}
		}
	}
}

// prefillQimen 确定性执行奇门遁甲排盘，结果注入 vals 和 session state。
func (e *Executor) prefillQimen(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) bool {
	caseID := strings.TrimSpace(plan.TurnContext.CaseID)
	questionTime, ok := parseTurnTime(plan.TurnContext.QuestionTime)
	if st == nil || caseID == "" || !ok {
		return false
	}
	if chart := st.QimenChartForCase(caseID); qimenChartMatchesTurn(chart, plan.TurnContext) {
		vals["qimen_result"] = chart
		return true
	}

	// 奇门主链是问事盘：即使会话已有出生资料，也按本轮问课时刻起局。
	params := qimenQuestionTimeParams(questionTime)
	if result := e.callTool(ctx, "qimen_dunjia", params); result != nil {
		if questionTimeValue := questionTime.Format(time.RFC3339); stringValue(result["question_time"]) != questionTimeValue {
			result["question_time"] = questionTimeValue
		}
		result["case_id"] = caseID
		result["purpose"] = "event_question"
		result["owner_ref"] = map[string]any{"kind": "case", "id": caseID}
		result["time_source"] = "question_time"
		ref := st.StoreChartForOwner(state.AssetKindQimenCaseChart, state.AssetRef{Kind: "case", ID: caseID}, result, "qimen-go-v1")
		if ref.ID == "" {
			return false
		}
		vals["qimen_result"] = result
		if bj, err := json.Marshal(result); err == nil && sink != nil {
			emitChartFromToolResult(ctx, sink, "qimen_dunjia", string(bj))
		}
		return true
	}
	return false
}

// qimenChartMatchesTurn accepts only a chart whose runtime metadata binds it to
// the current Case and question time; legacy payloads without metadata are stale.
func qimenChartMatchesTurn(chart map[string]any, turn contracts.TurnContext) bool {
	if len(chart) == 0 || stringValue(chart["case_id"]) != strings.TrimSpace(turn.CaseID) {
		return false
	}
	owner, ok := chart["owner_ref"].(map[string]any)
	if !ok || stringValue(owner["kind"]) != "case" || stringValue(owner["id"]) != strings.TrimSpace(turn.CaseID) {
		return false
	}
	if stringValue(chart["purpose"]) != "event_question" || stringValue(chart["time_source"]) != "question_time" {
		return false
	}
	return stringValue(chart["question_time"]) == strings.TrimSpace(turn.QuestionTime) &&
		stringValue(chart["pan_schema"]) == "rotating_8" &&
		stringValue(chart["symbol_system"]) == "eight_gate_eight_god"
}

// qimenQuestionTimeParams builds the minimal deterministic params for a Qi Men
// event chart; birth profile fields must not leak into this question-time chart.
func qimenQuestionTimeParams(questionTime time.Time) map[string]any {
	return map[string]any{
		"question_time": questionTime.Format(time.RFC3339),
	}
}

// prefillZiWei 确定性执行紫微斗数排盘，结果注入 vals 和 session state。
func (e *Executor) prefillZiWei(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	return e.prefillZiWeiForPlan(ctx, sink, st, ExecutionPlan{}, vals)
}

// prefillZiWeiForPlan executes Zi Wei prefill with the plan's dynamic target time.
func (e *Executor) prefillZiWeiForPlan(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) bool {
	profile := st.ActiveProfile()
	params := buildToolParams(profile)
	if params == nil {
		return false
	}
	targetAt := prefillTargetTime(plan)

	if chart := st.ActiveChart(state.AssetKindZiweiChart); chart != nil && isCurrentZiWeiSolarTime(chart) {
		vals["ziwei_result"] = chart
	} else if result := e.callTool(ctx, "ziwei_calc", params); result != nil {
		st.StoreChart(state.AssetKindZiweiChart, result, ziWeiMethodVersion())
		vals["ziwei_result"] = result
		if bj, err := json.Marshal(result); err == nil && sink != nil {
			emitChartFromToolResult(ctx, sink, "ziwei_calc", string(bj))
		}
	}

	if st.ZiWeiResult != nil {
		// 预排当前流年 (ziwei_liunian)
		if !targetAt.IsZero() {
			if _, ok := st.ZiWeiResult["liunian"]; !ok {
				// ZiWeiLiuNianTool 需要 year/month/day/hour（出生信息）+ target_year + age。
				// params 已含出生信息，复用并补充流年年份和虚岁年龄。
				liunianParams := map[string]any{
					"year":        params["year"],
					"month":       params["month"],
					"day":         params["day"],
					"hour":        params["hour"],
					"gender":      params["gender"],
					"target_year": float64(targetAt.Year()),
					"age":         float64(targetAt.Year() - int(toFloat(params["year"])) + 1),
				}
				if minute, ok := params["minute"]; ok {
					liunianParams["minute"] = minute
				}
				if longitude, ok := params["longitude"]; ok {
					liunianParams["longitude"] = longitude
				}
				if result := e.callTool(ctx, "ziwei_liunian", liunianParams); result != nil {
					st.ZiWeiResult["liunian"] = result
					vals["ziwei_liunian"] = result
				}
			}
		}
	}

	return st.ZiWeiResult != nil
}

func (e *Executor) prefillBazi(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	return e.prefillBaziForPlan(ctx, sink, st, ExecutionPlan{}, vals)
}

// prefillBaziForPlan executes Ba Zi prefill with the plan's dynamic target time.
func (e *Executor) prefillBaziForPlan(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) bool {
	profile := st.ActiveProfile()
	baziParams := buildToolParams(profile)
	if baziParams == nil {
		return false
	}
	shouldEmitChart := false

	// 历史会话里可能缓存着旧的“晚子时整段进次日”命盘。
	// 一旦发现口径版本不是当前标准，就先丢弃旧盘，再按当前规则重排。
	if st.HasBaziResult() && !isCurrentBaziCalendarRule(st.BaziResult) {
		st.InvalidateActiveChart(state.AssetKindBaziChart)
		shouldEmitChart = true
	}

	if !st.HasBaziResult() {
		if result := e.callTool(ctx, "bazi_calc", baziParams); result != nil {
			st.StoreChart(state.AssetKindBaziChart, result, "lunar-go")
			vals["bazi_result"] = result
			shouldEmitChart = true
		}
	}

	if st.BaziResult != nil {
		if existing, ok := st.BaziResult["yongshen"].(map[string]any); ok && len(existing) > 0 {
			vals["yongshen"] = existing
		} else if result := e.callTool(ctx, "yongshen", baziParams); result != nil {
			vals["yongshen"] = result
			st.BaziResult["yongshen"] = result
		}
	}

	if st.BaziResult != nil {
		if existing, ok := st.BaziResult["dayun_analyzed"].(map[string]any); ok && len(existing) > 0 {
			vals["dayun_analyzed"] = existing
		} else {
			dayunParams := map[string]any{"dayun": st.BaziResult["dayun"], "bazi_result": st.BaziResult}
			if result := e.callTool(ctx, "dayun_analyzer", dayunParams); result != nil {
				vals["dayun_analyzed"] = result
				st.BaziResult["dayun_analyzed"] = result
			}
		}
	}

	// 预排当前流年（bazi_liunian）：缓存只在同一自然日且结构完整时复用。
	// 空 map、旧日期或缺当前运选择信息都必须重算，否则动态报告会拿到
	// 已有九步大运但没有“当前大运”的自相矛盾输入。
	if st.BaziResult != nil {
		targetAt := prefillTargetTime(plan)
		if !targetAt.IsZero() && !hasCurrentBaziLiuNian(st.BaziResult["liunian"], targetAt) {
			liunianParams := map[string]any{
				"target_year":   float64(targetAt.Year()),
				"target_month":  float64(int(targetAt.Month())),
				"target_day":    float64(targetAt.Day()),
				"target_hour":   float64(targetAt.Hour()),
				"target_minute": float64(targetAt.Minute()),
				"bazi_result":   st.BaziResult,
			}
			if result := e.callTool(ctx, "bazi_liunian", liunianParams); result != nil {
				st.BaziResult["liunian"] = result
				vals["liunian"] = result
			}
		}
	}

	if st.BaziResult != nil {
		vals["bazi_result"] = st.BaziResult
		if bj, err := json.Marshal(st.BaziResult); err == nil {
			vals["bazi_json"] = string(bj)
			if sink != nil && shouldEmitChart {
				// The frontend copies the chart payload directly, so emit only after
				// deterministic enrichment finishes to avoid losing geju/yongshen data.
				// follow-up 只复用已有命盘时不再重复发盘，避免前端重复插入同一 bazi-chart。
				emitChartFromToolResult(ctx, sink, "bazi_calc", string(bj))
			}
		}
	}

	return true
}

// parseTurnTime parses the fixed per-turn time contract without consulting the system clock.
func parseTurnTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	value, err := time.Parse(time.RFC3339, raw)
	return value, err == nil
}

// prefillTargetTime applies the dynamic-target precedence. A zero time signals
// a missing contract; callers must degrade instead of consulting the system clock.
func prefillTargetTime(plan ExecutionPlan) time.Time {
	if targetAt, ok := parseTurnTime(plan.TurnContext.TargetAt); ok {
		return targetAt
	}
	if questionTime, ok := parseTurnTime(plan.TurnContext.QuestionTime); ok {
		return questionTime
	}
	return time.Time{}
}

// hasCurrentBaziLiuNian accepts only a same-day, structurally complete cache.
// The selected current luck period may legitimately be empty before the first
// luck boundary, so selection metadata rather than a non-empty ganZhi marks it
// as a valid calculation.
func hasCurrentBaziLiuNian(raw any, now time.Time) bool {
	liunian, ok := raw.(map[string]any)
	if !ok || len(liunian) == 0 {
		return false
	}
	targetAt := strings.TrimSpace(stringValue(liunian["liunian_target_at"]))
	if !strings.HasPrefix(targetAt, now.Format("2006-01-02")) {
		return false
	}
	if strings.TrimSpace(stringValue(liunian["liunian_ganzhi"])) == "" {
		return false
	}
	_, hasSelection := liunian["current_dayun_selection"]
	return hasSelection
}
