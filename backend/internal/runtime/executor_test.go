package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
	bazitool "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"
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

func TestBuildToolParams_UsesBirthplaceLongitudeForTrueSolarTime(t *testing.T) {
	params := buildToolParams(map[string]any{
		"year": 2025.0, "month": 11.0, "day": 10.0,
		"hour": 23.0, "minute": 53.0, "gender": "男", "birthplace": "上海",
	})
	if got := params["longitude"]; got != 121.4737 {
		t.Fatalf("longitude = %v, want Shanghai longitude 121.4737", got)
	}
}

func TestIsCurrentZiWeiSolarTimeRequiresVersion(t *testing.T) {
	if isCurrentZiWeiSolarTime(map[string]any{"solar_time_version": "legacy"}) {
		t.Fatal("legacy ziwei chart must be recalculated")
	}
	if !isCurrentZiWeiSolarTime(map[string]any{"solar_time_version": bazitool.TrueSolarTimeVersion}) {
		t.Fatal("current true-solar ziwei chart should be reusable")
	}
}
