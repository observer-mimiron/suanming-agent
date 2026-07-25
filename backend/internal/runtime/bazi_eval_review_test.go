package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewBaziEvalOutput_PassesForStructuredConstrainedOutput(t *testing.T) {
	fixture := mustLoadBaziEvalFixture(t, "case_001_abandon_seal_follow_wealth.json")
	output := strings.Join([]string{
		"## 格局视角",
		"**结论：命局主轴为弃印就财，但调候不足，财星偏湿，药力不足，难以拔高。**",
		"为何不取其他路线：不是杀印相生为主轴，也不按从印格处理。",
		"## 大运验证",
		"当前正行辛卯大运，整体吉中有阻。",
		"## 流年应期",
		"**结论：2026 更像窗口年，机会伴随强变动，不宜激进。**",
		"子午双冲直接引动原局关键位置。",
	}, "\n")

	review := reviewBaziEvalOutput(fixture, output)
	if !review.Passed {
		t.Fatalf("expected review to pass, got violations: %v", review.Violations)
	}
}

func TestReviewBaziEvalOutput_FailsForAxisDriftAndOverstatement(t *testing.T) {
	fixture := mustLoadBaziEvalFixture(t, "case_001_abandon_seal_follow_wealth.json")
	output := strings.Join([]string{
		"## 格局视角",
		"**结论：命局主轴为杀印相生，一飞冲天。**",
		"## 大运验证",
		"当前正行辛卯大运。",
		"## 流年应期",
		"**结论：2026 是关键翻身年，可激进 all-in。**",
		"命局贵人众多，福泽深厚，可享清福。",
	}, "\n")

	review := reviewBaziEvalOutput(fixture, output)
	if review.Passed {
		t.Fatalf("expected review to fail")
	}
	if len(review.Violations) < 4 {
		t.Fatalf("expected multiple violations, got %v", review.Violations)
	}
	assertContainsViolation(t, review.Violations, "expected primary axis")
	assertContainsViolation(t, review.Violations, "required limitation")
	assertContainsViolation(t, review.Violations, "2026 forbidden phrase")
	assertContainsViolation(t, review.Violations, "forbidden flourish")
}

func TestReviewBaziEvalOutput_HistoricalSessionFlagsKnownFlourishRegression_Case001(t *testing.T) {
	fixture := mustLoadBaziEvalFixture(t, "case_001_abandon_seal_follow_wealth.json")
	sessionPath := filepath.Join("..", "..", "..", "data", "sessions", fixture.SourceSessionID+".json")
	result, err := reviewBaziEvalSessionFile(fixture, sessionPath)
	if err != nil {
		t.Fatalf("review session: %v", err)
	}
	if !result.HasOutput {
		t.Fatalf("expected historical session to contain assistant output")
	}
	if result.Review.Passed {
		t.Fatalf("expected historical session output to fail review")
	}
	assertContainsViolation(t, result.Review.Violations, "forbidden flourish")
}

func TestReviewBaziEvalOutput_HistoricalSessionFlagsKnownFlourishRegression_Case002(t *testing.T) {
	fixture := mustLoadBaziEvalFixture(t, "case_002_food_god_controls_killing.json")
	sessionPath := filepath.Join("..", "..", "..", "data", "sessions", fixture.SourceSessionID+".json")
	result, err := reviewBaziEvalSessionFile(fixture, sessionPath)
	if err != nil {
		t.Fatalf("review session: %v", err)
	}
	if !result.HasOutput {
		t.Fatalf("expected historical session to contain assistant output")
	}
	if result.Review.Passed {
		t.Fatalf("expected historical session output to fail review")
	}
	assertContainsViolation(t, result.Review.Violations, "forbidden flourish")
}

func TestReviewBaziEvalOutput_HistoricalSessionFlagsCrossSchoolArbitrationRegression_Case003(t *testing.T) {
	fixture := mustLoadBaziEvalFixture(t, "case_003_kill_print_vs_tiaohou_conflict.json")
	sessionPath := filepath.Join("..", "..", "..", "data", "sessions", fixture.SourceSessionID+".json")
	result, err := reviewBaziEvalSessionFile(fixture, sessionPath)
	if err != nil {
		t.Fatalf("review session: %v", err)
	}
	if !result.HasOutput {
		t.Fatalf("expected historical session to contain assistant output")
	}
	if result.Review.Passed {
		t.Fatalf("expected historical session output to fail review")
	}
	assertContainsViolation(t, result.Review.Violations, "forbidden axis")
	assertContainsViolation(t, result.Review.Violations, "required limitation")
}

func TestReviewBaziEvalSessionFile_FlagsMissingAssistantOutput(t *testing.T) {
	fixture := mustLoadBaziEvalFixture(t, "case_001_abandon_seal_follow_wealth.json")
	sessionPath := filepath.Join(t.TempDir(), "missing-assistant-output.json")
	raw := `{"SessionID":"missing-assistant-output","recent_turns":[{"role":"user","content":"只留用户消息"}]}`
	if err := os.WriteFile(sessionPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write temp session: %v", err)
	}

	result, err := reviewBaziEvalSessionFile(fixture, sessionPath)
	if err != nil {
		t.Fatalf("review session: %v", err)
	}
	if result.HasOutput {
		t.Fatalf("expected session without stored assistant text to report no output")
	}
	assertContainsViolation(t, result.Review.Violations, "no assistant output")
}

func assertContainsViolation(t *testing.T, violations []string, want string) {
	t.Helper()
	for _, violation := range violations {
		if strings.Contains(violation, want) {
			return
		}
	}
	t.Fatalf("expected violation containing %q, got %v", want, violations)
}
