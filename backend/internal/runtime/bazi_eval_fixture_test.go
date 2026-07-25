package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBaziEvalFixtures_AreWellFormed 锁定最小八字评测样本的结构完整性，
// 避免后续增加样本时退化成“只有自然语言，没有可回归约束”的文档。
func TestBaziEvalFixtures_AreWellFormed(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "bazi_eval_cases")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}

	var jsonFiles []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		jsonFiles = append(jsonFiles, filepath.Join(fixtureDir, entry.Name()))
	}
	if len(jsonFiles) < 2 {
		t.Fatalf("expected at least 2 bazi eval fixtures, got %d", len(jsonFiles))
	}

	seenCaseIDs := map[string]string{}
	for _, path := range jsonFiles {
		fixture, err := loadBaziEvalFixture(path)
		if err != nil {
			t.Fatalf("unmarshal fixture %s: %v", path, err)
		}

		if fixture.CaseID == "" {
			t.Fatalf("%s: missing case_id", path)
		}
		if prev, ok := seenCaseIDs[fixture.CaseID]; ok {
			t.Fatalf("%s: duplicate case_id %q already used by %s", path, fixture.CaseID, prev)
		}
		seenCaseIDs[fixture.CaseID] = path

		if fixture.Label == "" {
			t.Fatalf("%s: missing label", path)
		}
		if fixture.SourceSessionID == "" {
			t.Fatalf("%s: missing source_session_id", path)
		}
		if fixture.InputProfile.Year == 0 || fixture.InputProfile.Month == 0 || fixture.InputProfile.Day == 0 {
			t.Fatalf("%s: incomplete input_profile date", path)
		}
		if fixture.InputProfile.Gender == "" || fixture.InputProfile.Birthplace == "" {
			t.Fatalf("%s: incomplete input_profile identity", path)
		}

		if len(fixture.Expected.PrimaryAxisCandidates) < 2 {
			t.Fatalf("%s: expected at least 2 primary_axis_candidates", path)
		}
		if fixture.Expected.ExpectedPrimaryAxis == "" {
			t.Fatalf("%s: missing expected_primary_axis", path)
		}
		if len(fixture.Expected.ForbiddenAxes) == 0 {
			t.Fatalf("%s: missing forbidden_axes", path)
		}
		if !containsString([]string{"high", "medium", "low"}, fixture.Expected.TiaohouPriority) {
			t.Fatalf("%s: invalid tiaohou_priority %q", path, fixture.Expected.TiaohouPriority)
		}
		if len(fixture.Expected.StrengthAssessment.Allowed) == 0 {
			t.Fatalf("%s: missing allowed strength bounds", path)
		}
		if len(fixture.Expected.RequiredLimitations) == 0 {
			t.Fatalf("%s: missing required_limitations", path)
		}
		if fixture.Expected.DynamicExpectations.CurrentDayun == "" {
			t.Fatalf("%s: missing current_dayun", path)
		}
		if !containsString([]string{"承压年", "扰动年", "窗口年", "转折年"}, fixture.Expected.DynamicExpectations.Liunian2026.WindowLevel) {
			t.Fatalf("%s: invalid 2026 window_level %q", path, fixture.Expected.DynamicExpectations.Liunian2026.WindowLevel)
		}
		if len(fixture.Expected.DynamicExpectations.Liunian2026.RequiredSignals) == 0 {
			t.Fatalf("%s: missing 2026 required_signals", path)
		}
		if len(fixture.Expected.ForbiddenFlourishes) == 0 {
			t.Fatalf("%s: missing forbidden_flourishes", path)
		}
		if len(fixture.Expected.ReviewFocus) == 0 {
			t.Fatalf("%s: missing review_focus", path)
		}
	}
}
