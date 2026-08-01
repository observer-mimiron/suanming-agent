package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type baziEvalFixture struct {
	CaseID          string `json:"case_id"`
	Label           string `json:"label"`
	SourceSessionID string `json:"source_session_id"`
	InputProfile    struct {
		Year       int    `json:"year"`
		Month      int    `json:"month"`
		Day        int    `json:"day"`
		Hour       int    `json:"hour"`
		Minute     int    `json:"minute"`
		Gender     string `json:"gender"`
		Birthplace string `json:"birthplace"`
	} `json:"input_profile"`
	Expected struct {
		PrimaryAxisCandidates []string `json:"primary_axis_candidates"`
		ExpectedPrimaryAxis   string   `json:"expected_primary_axis"`
		ForbiddenAxes         []string `json:"forbidden_axes"`
		TiaohouPriority       string   `json:"tiaohou_priority"`
		StrengthAssessment    struct {
			Allowed   []string `json:"allowed"`
			Forbidden []string `json:"forbidden"`
		} `json:"strength_assessment_bounds"`
		RequiredLimitations []string `json:"required_limitations"`
		DynamicExpectations struct {
			CurrentDayun string `json:"current_dayun"`
			Liunian2026  struct {
				WindowLevel      string   `json:"window_level"`
				RequiredSignals  []string `json:"required_signals"`
				ForbiddenPhrases []string `json:"forbidden_phrases"`
			} `json:"liunian_2026"`
		} `json:"dynamic_expectations"`
		ForbiddenFlourishes []string `json:"forbidden_flourishes"`
		ReviewFocus         []string `json:"review_focus"`
	} `json:"expected"`
}

type baziEvalReview struct {
	Passed     bool     `json:"passed"`
	Violations []string `json:"violations"`
}

type baziEvalSessionReview struct {
	SessionID  string         `json:"session_id"`
	CaseID     string         `json:"case_id"`
	Output     string         `json:"output"`
	Review     baziEvalReview `json:"review"`
	HasOutput  bool           `json:"has_output"`
	OutputRole string         `json:"output_role,omitempty"`
}

func loadBaziEvalFixture(path string) (baziEvalFixture, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return baziEvalFixture{}, err
	}

	var fixture baziEvalFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return baziEvalFixture{}, err
	}
	return fixture, nil
}

func loadSessionState(path string) (*state.SessionState, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session state.SessionState
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func latestAssistantOutputFromSession(session *state.SessionState) (string, bool) {
	if session == nil {
		return "", false
	}
	for i := len(session.RecentTurns) - 1; i >= 0; i-- {
		turn := session.RecentTurns[i]
		if turn.Role == "assistant" && strings.TrimSpace(turn.Content) != "" {
			return turn.Content, true
		}
	}
	return "", false
}

func reviewBaziEvalSessionFile(fixture baziEvalFixture, sessionPath string) (baziEvalSessionReview, error) {
	session, err := loadSessionState(sessionPath)
	if err != nil {
		return baziEvalSessionReview{}, err
	}

	output, ok := latestAssistantOutputFromSession(session)
	if !ok {
		return baziEvalSessionReview{
			SessionID: session.SessionID,
			CaseID:    fixture.CaseID,
			HasOutput: false,
			Review: baziEvalReview{
				Passed:     false,
				Violations: []string{"session has no assistant output"},
			},
		}, nil
	}

	return baziEvalSessionReview{
		SessionID:  session.SessionID,
		CaseID:     fixture.CaseID,
		Output:     output,
		HasOutput:  true,
		OutputRole: "assistant",
		Review:     reviewBaziEvalOutput(fixture, output),
	}, nil
}

func mustLoadBaziEvalFixture(t *testing.T, filename string) baziEvalFixture {
	t.Helper()

	path := filepath.Join("testdata", "bazi_eval_cases", filename)
	fixture, err := loadBaziEvalFixture(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", filename, err)
	}
	return fixture
}

// reviewBaziEvalOutput 用最小高收益规则审查一份八字成文输出，
// 先覆盖主轴漂移、限制语丢失、窗口年拔高和无依据漂亮话这四类高频误判。
func reviewBaziEvalOutput(fixture baziEvalFixture, output string) baziEvalReview {
	output = strings.TrimSpace(output)
	violations := make([]string, 0, 8)
	if output == "" {
		return baziEvalReview{
			Passed:     false,
			Violations: []string{"empty output"},
		}
	}

	if expected := strings.TrimSpace(fixture.Expected.ExpectedPrimaryAxis); expected != "" && !strings.Contains(output, expected) {
		violations = append(violations, fmt.Sprintf("missing expected primary axis: %s", expected))
	}
	for _, forbidden := range fixture.Expected.ForbiddenAxes {
		forbidden = strings.TrimSpace(forbidden)
		if forbidden != "" && containsAffirmativePhrase(output, forbidden) {
			violations = append(violations, fmt.Sprintf("contains forbidden axis: %s", forbidden))
		}
	}
	for _, limitation := range fixture.Expected.RequiredLimitations {
		limitation = strings.TrimSpace(limitation)
		if limitation != "" && !strings.Contains(output, limitation) {
			violations = append(violations, fmt.Sprintf("missing required limitation: %s", limitation))
		}
	}
	for _, signal := range fixture.Expected.DynamicExpectations.Liunian2026.RequiredSignals {
		signal = strings.TrimSpace(signal)
		if signal != "" && !strings.Contains(output, signal) {
			violations = append(violations, fmt.Sprintf("missing 2026 required signal: %s", signal))
		}
	}
	for _, phrase := range fixture.Expected.DynamicExpectations.Liunian2026.ForbiddenPhrases {
		phrase = strings.TrimSpace(phrase)
		if phrase != "" && containsAffirmativePhrase(output, phrase) {
			violations = append(violations, fmt.Sprintf("contains 2026 forbidden phrase: %s", phrase))
		}
	}
	for _, flourish := range fixture.Expected.ForbiddenFlourishes {
		flourish = strings.TrimSpace(flourish)
		if flourish != "" && containsAffirmativePhrase(output, flourish) {
			violations = append(violations, fmt.Sprintf("contains forbidden flourish: %s", flourish))
		}
	}

	return baziEvalReview{
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}

func containsAffirmativePhrase(text, phrase string) bool {
	if strings.TrimSpace(text) == "" || strings.TrimSpace(phrase) == "" {
		return false
	}

	textRunes := []rune(text)
	phraseRunes := []rune(phrase)
	if len(phraseRunes) == 0 || len(textRunes) < len(phraseRunes) {
		return false
	}

	negationHints := []string{
		"不是",
		"并非",
		"不按",
		"不取",
		"不能按",
		"不可按",
		"并不",
		"非",
	}

	for i := 0; i <= len(textRunes)-len(phraseRunes); i++ {
		if string(textRunes[i:i+len(phraseRunes)]) != phrase {
			continue
		}

		start := i - 6
		if start < 0 {
			start = 0
		}
		prefix := string(textRunes[start:i])
		negated := false
		for _, hint := range negationHints {
			if strings.Contains(prefix, hint) {
				negated = true
				break
			}
		}
		if !negated {
			return true
		}
	}

	return false
}
