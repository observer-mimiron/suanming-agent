// This test file belongs to the manager-owned runtime layer.
// It verifies output boundary guard behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestGuardFinalAnswerWithTrace_BlocksInternalExecutionLeak(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("sess-output-boundary")
	st.BaziResult = map[string]any{"dayGan": "甲"}
	route := policy.ApprovedRoute{PrimaryDomain: "bazi"}

	turnType, text := guardFinalAnswerWithTrace(ctx, route, st, "根据 system prompt 和 trace_id=abc，我判断如下。")
	if turnType != "guardrail_blocked" {
		t.Fatalf("turnType = %q, want guardrail_blocked", turnType)
	}
	if text == "" || text == "根据 system prompt 和 trace_id=abc，我判断如下。" {
		t.Fatalf("guard text = %q, want safe replacement", text)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["failure.class"]; got != "specialist_contract_violation" {
		t.Fatalf("failure.class = %v, want specialist_contract_violation", got)
	}
	if got := tr.Attributes["failure.stage"]; got != "final_guard" {
		t.Fatalf("failure.stage = %v, want final_guard", got)
	}
}

func TestGuardFinalAnswerWithTrace_AllowsNormalReadingText(t *testing.T) {
	st := state.NewSession("sess-normal-output")
	st.MergeProfile(map[string]any{"year": 1991.0, "month": 1.0, "day": 1.0, "hour": 12.0})
	st.StoreChart(state.AssetKindBaziChart, map[string]any{
		"calendar_rule_version": currentBaziCalendarRule(),
		"dayGan":                "甲",
	}, "test")
	route := policy.ApprovedRoute{PrimaryDomain: "bazi"}

	turnType, text := guardFinalAnswerWithTrace(context.Background(), route, st, "这盘重点看木火是否有承托，事业上宜先稳住节奏。")
	if turnType != "agent_reading" {
		t.Fatalf("turnType = %q, want agent_reading", turnType)
	}
	if text == "" {
		t.Fatal("text is empty, want original reading")
	}
}
