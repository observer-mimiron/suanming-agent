// This test file belongs to the manager-owned runtime layer.
// It verifies Executor behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	solartime "github.com/observer-mimiron/suanming-agent/internal/calendar"
	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	qimenapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/qimen/application"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

func TestExecutor_RouterField(t *testing.T) {
	e := &Executor{router: &intent.SemanticRouter{}}
	if e.router == nil {
		t.Fatal("router field not set")
	}
}

func TestHasCurrentBaziLiuNianRequiresSameDayAndSelectionMetadata(t *testing.T) {
	now := time.Date(2026, time.July, 28, 15, 0, 0, 0, time.Local)
	valid := map[string]any{
		"liunian_target_at":       "2026-07-28 12:00:00",
		"liunian_ganzhi":          "丙午",
		"current_dayun_selection": "date_boundary",
		"current_dayun":           map[string]any{},
	}
	if !hasCurrentBaziLiuNian(valid, now) {
		t.Fatal("valid same-day result with an empty pre-start period should be reusable")
	}
	if hasCurrentBaziLiuNian(map[string]any{}, now) {
		t.Fatal("empty liunian cache must be recalculated")
	}
	valid["liunian_target_at"] = "2026-07-27 12:00:00"
	if hasCurrentBaziLiuNian(valid, now) {
		t.Fatal("stale liunian cache must be recalculated")
	}
}

type executorGuardTool struct {
	called bool
}

func (t *executorGuardTool) Name() string        { return "needs_query" }
func (t *executorGuardTool) Description() string { return "test tool" }
func (t *executorGuardTool) Label() string       { return "Test Tool" }
func (t *executorGuardTool) Execute(context.Context, map[string]any) (any, error) {
	t.called = true
	return map[string]any{"ok": true}, nil
}

func TestExecutorCallTool_UsesToolRunnerInvalidParamGuard(t *testing.T) {
	tool := &executorGuardTool{}
	reg := tools.NewRegistry()
	reg.RegisterWithContract(tool, tools.ToolContract{
		Name:       "needs_query",
		Version:    "v1",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: tools.SideEffectRead,
		RiskLevel:  tools.RiskLow,
		Params: []tools.ParamSpec{
			{Name: "query", Type: "string", Required: true},
		},
		Retry: tools.RetryPolicy{MaxAttempts: 1},
	})

	e := &Executor{reg: reg}
	result := e.callTool(context.Background(), "needs_query", map[string]any{})
	if result != nil {
		t.Fatalf("result = %v, want nil when params are invalid", result)
	}
	if tool.called {
		t.Fatal("tool must not execute when ToolRunner rejects invalid params")
	}
}

type captureQimenTool struct {
	params map[string]any
}

func (t *captureQimenTool) Name() string        { return "qimen_dunjia" }
func (t *captureQimenTool) Description() string { return "test qimen tool" }
func (t *captureQimenTool) Label() string       { return "奇门遁甲" }
func (t *captureQimenTool) Execute(_ context.Context, params map[string]any) (any, error) {
	t.params = params
	return map[string]any{"ok": true}, nil
}

func TestPrefillQimen_UsesQuestionTimeInsteadOfBirthProfile(t *testing.T) {
	tool := &captureQimenTool{}
	reg := tools.NewRegistry()
	reg.Register(tool)
	e := &Executor{reg: reg}
	st := state.NewSession("qimen-question-time")
	st.Profile = map[string]any{
		"year": 1991.0, "month": 10.0, "day": 5.0, "hour": 12.0, "minute": 40.0,
		"gender": "男", "birthplace": "北京",
	}
	questionTime := "2026-08-05T14:30:00+08:00"
	parsedQuestionTime, err := time.Parse(time.RFC3339, questionTime)
	if err != nil {
		t.Fatalf("parse question time: %v", err)
	}
	oldCase := st.StartCaseAt("qimen", "旧问题", &parsedQuestionTime, true)
	st.StoreChartForOwner(state.AssetKindQimenCaseChart, state.AssetRef{Kind: "case", ID: oldCase.ID}, map[string]any{"ju": "old"}, "test")
	newCase := st.StartCaseAt("qimen", "当前问题", &parsedQuestionTime, true)
	plan := ExecutionPlan{TurnContext: contracts.TurnContext{
		QuestionTime: questionTime,
		CaseID:       newCase.ID,
	}}
	vals := map[string]any{}
	if !e.prefillQimen(context.Background(), nil, st, plan, vals) {
		t.Fatal("prefillQimen() = false, want true")
	}
	if len(tool.params) != 1 {
		t.Fatalf("qimen params = %+v, want exactly question_time", tool.params)
	}
	if got := tool.params["question_time"]; got != questionTime {
		t.Fatalf("question_time = %v, want %s", got, questionTime)
	}
	for _, forbidden := range []string{"year", "month", "day", "hour", "minute", "gender", "birthplace", "longitude"} {
		if _, ok := tool.params[forbidden]; ok {
			t.Fatalf("qimen params leaked %q: %+v", forbidden, tool.params)
		}
	}
	result, ok := vals["qimen_result"].(map[string]any)
	if !ok {
		t.Fatalf("qimen_result = %T, want map[string]any", vals["qimen_result"])
	}
	if got := result["case_id"]; got != newCase.ID {
		t.Fatalf("case_id = %v, want %s", got, newCase.ID)
	}
	if got := result["purpose"]; got != "event_question" {
		t.Fatalf("purpose = %v, want event_question", got)
	}
	owner, ok := result["owner_ref"].(map[string]any)
	if !ok || owner["kind"] != "case" || owner["id"] != newCase.ID {
		t.Fatalf("owner_ref = %v, want exact case owner", result["owner_ref"])
	}
	if got := result["time_source"]; got != "question_time" {
		t.Fatalf("time_source = %v, want question_time", got)
	}
	if got := st.QimenChartForCase(newCase.ID)["case_id"]; got != newCase.ID {
		t.Fatalf("stored current case_id = %v, want %s", got, newCase.ID)
	}
	if got := st.QimenChartForCase(oldCase.ID)["ju"]; got != "old" {
		t.Fatalf("old case chart = %v, want old", got)
	}
}

func TestSpecialistSessionView_QimenContainsOnlyCaseChartContext(t *testing.T) {
	st := state.NewSession("qimen-specialist-view")
	st.MergeProfile(map[string]any{
		"year": 1991.0, "month": 10.0, "day": 5.0, "hour": 12.0,
		"gender": "男", "birthplace": "北京",
	})
	st.RecordTurn("user", "我的出生资料")
	st.RunningSummary = "早期对话摘要"
	st.BaziResult = map[string]any{"dayGan": "甲"}
	st.ZiWeiResult = map[string]any{"mingGong": "紫微"}
	questionTime := "2026-08-05T14:30:00+08:00"
	parsedQuestionTime, err := time.Parse(time.RFC3339, questionTime)
	if err != nil {
		t.Fatalf("parse question time: %v", err)
	}
	item := st.StartCaseAt("qimen", "这个面试能不能成", &parsedQuestionTime, true)
	payload := map[string]any{
		"case_id":       item.ID,
		"purpose":       "event_question",
		"owner_ref":     map[string]any{"kind": "case", "id": item.ID},
		"question_time": questionTime,
		"time_source":   "question_time",
		"pan_schema":    "rotating_8",
		"symbol_system": "eight_gate_eight_god",
		"cells":         []map[string]any{{"palace": "坎", "door": "休门", "god": "值符"}},
	}
	st.StoreChartForOwner(state.AssetKindQimenCaseChart, state.AssetRef{Kind: "case", ID: item.ID}, payload, "test")

	view := specialistSessionView(st, ExecutionPlan{TurnContext: contracts.TurnContext{CaseID: item.ID}}, "qimen")
	if view == nil {
		t.Fatal("specialistSessionView returned nil")
	}
	if len(view.Profile) != 0 || len(view.RecentTurns) != 0 || view.RunningSummary != "" {
		t.Fatalf("qimen view leaked profile/history: profile=%v turns=%d summary=%q", view.Profile, len(view.RecentTurns), view.RunningSummary)
	}
	if view.BaziResult != nil || view.ZiWeiResult != nil {
		t.Fatalf("qimen view leaked natal charts: bazi=%v ziwei=%v", view.BaziResult, view.ZiWeiResult)
	}
	if got := view.QimenResult["case_id"]; got != item.ID {
		t.Fatalf("qimen case_id = %v, want %s", got, item.ID)
	}

	block := qimenapplication.BuildDataBlock(view.QimenResult)
	for _, want := range []string{"Case", "event_question", item.ID, questionTime, "question_time", "rotating_8", "eight_gate_eight_god"} {
		if !strings.Contains(block, want) {
			t.Fatalf("qimen data block missing %q: %s", want, block)
		}
	}
}

func TestPrefillTargetTime_PrioritizesTargetAtThenQuestionTime(t *testing.T) {
	targetAt := "2026-09-01T09:00:00+08:00"
	questionTime := "2026-08-05T14:30:00+08:00"
	plan := ExecutionPlan{TurnContext: contracts.TurnContext{TargetAt: targetAt, QuestionTime: questionTime}}
	got := prefillTargetTime(plan)
	want, _ := time.Parse(time.RFC3339, targetAt)
	if !got.Equal(want) {
		t.Fatalf("target time = %s, want TargetAt %s", got.Format(time.RFC3339), targetAt)
	}

	plan.TurnContext.TargetAt = ""
	got = prefillTargetTime(plan)
	want, _ = time.Parse(time.RFC3339, questionTime)
	if !got.Equal(want) {
		t.Fatalf("fallback target time = %s, want QuestionTime %s", got.Format(time.RFC3339), questionTime)
	}
}

func TestBuildToolParams_UsesBirthplaceLongitudeForTrueSolarTime(t *testing.T) {
	params := buildToolParams(map[string]any{
		"year": 2025.0, "month": 11.0, "day": 10.0,
		"hour": 23.0, "minute": 53.0, "gender": "男", "birthplace": "上海",
	})
	if got := params["longitude"]; got != 121.4737 {
		t.Fatalf("longitude = %v, want Shanghai longitude 121.4737", got)
	}
}

func TestBuildToolParamsNormalizesGenderExpression(t *testing.T) {
	params := buildToolParams(map[string]any{
		"year": 2025.0, "month": 11.0, "day": 10.0, "hour": 23.0, "gender": "男性",
	})
	if got := params["gender"]; got != "男" {
		t.Fatalf("gender = %q, want 男", got)
	}
}

func TestIsCurrentZiWeiSolarTimeRequiresVersion(t *testing.T) {
	if isCurrentZiWeiSolarTime(map[string]any{"solar_time_version": "legacy"}) {
		t.Fatal("legacy ziwei chart must be recalculated")
	}
	if !isCurrentZiWeiSolarTime(map[string]any{"solar_time_version": solartime.TrueSolarTimeVersion}) {
		t.Fatal("current true-solar ziwei chart should be reusable")
	}
}
