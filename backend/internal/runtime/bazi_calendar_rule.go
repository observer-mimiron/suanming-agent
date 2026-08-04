// This file belongs to the manager-owned runtime layer.
// It owns BaZi calendar-rule selection for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import bazitool "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"

// currentBaziCalendarRule returns the single runtime source of truth for bazi
// chart calendar ownership checks.
func currentBaziCalendarRule() string {
	return bazitool.CalendarRuleVersion
}

func isCurrentBaziCalendarRule(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	version, _ := result["calendar_rule_version"].(string)
	return version == currentBaziCalendarRule()
}

func isCurrentZiWeiSolarTime(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	return stringValue(result["solar_time_version"]) == bazitool.TrueSolarTimeVersion
}

func ziWeiMethodVersion() string {
	return "ziwei-" + bazitool.TrueSolarTimeVersion
}
