package specialists_test

import (
	"testing"

	baziSpecialist "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi"
	qimenSpecialist "github.com/observer-mimiron/suanming-agent/internal/specialists/qimen/adapter"
	ziweiSpecialist "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/adapter"
)

func TestSpecialistToolNamesExcludeDeterministicChartTools(t *testing.T) {
	configs := []struct {
		name  string
		tools []string
	}{
		{name: "bazi", tools: baziSpecialist.GetConfig().ToolNames},
		{name: "qimen", tools: qimenSpecialist.GetConfig().ToolNames},
		{name: "ziwei", tools: ziweiSpecialist.GetConfig().ToolNames},
	}
	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			for _, toolName := range tc.tools {
				switch toolName {
				case "bazi_calc", "qimen_dunjia", "ziwei_calc":
					t.Fatalf("specialist ToolNames must not include deterministic chart tool %q: %v", toolName, tc.tools)
				}
			}
		})
	}
}
