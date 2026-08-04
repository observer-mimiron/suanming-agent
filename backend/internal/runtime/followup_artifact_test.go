// This test file belongs to the manager-owned runtime layer.
// It verifies follow-up artifact reuse and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type rejectingFollowupChat struct {
	generateCalls int
}

func (c *rejectingFollowupChat) Stream(context.Context, string, []llm.Message, func(string)) error {
	return nil
}

func (c *rejectingFollowupChat) Generate(context.Context, string, []llm.Message) (string, llm.TokenUsage, error) {
	c.generateCalls++
	return "must not be called", llm.TokenUsage{}, nil
}

func TestMaybeReuseFollowupArtifact_ProfileBoundaryNeverCallsLLM(t *testing.T) {
	boundary := "## 本轮结果\n\n**结论：本次仅输出已确认的命盘事实。**\n\n### 本轮不作裁断\n- 旺衰与格局层次。"
	st := state.NewSession("profile-boundary-followup")
	st.MergeProfile(map[string]any{
		"year": 1991.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男",
	})
	st.StoreChart(state.AssetKindBaziChart, map[string]any{"calendar_rule_version": currentBaziCalendarRule()}, "test")
	st.StoreInterpretation("bazi", map[string]any{
		"domain":           "bazi",
		"summary":          boundary,
		"profile_boundary": true,
	})
	chat := &rejectingFollowupChat{}
	text, ok := maybeReuseFollowupArtifact(NewManager(chat), st, policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
	}, []string{"bazi"}, "继续解读刚才的命盘")
	if !ok || text != boundary {
		t.Fatalf("profile boundary must return the stored deterministic result, got ok=%v text=%q", ok, text)
	}
	if chat.generateCalls != 0 {
		t.Fatalf("profile boundary follow-up must not call the generic LLM, calls=%d", chat.generateCalls)
	}
}
