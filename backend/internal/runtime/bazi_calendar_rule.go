package runtime

import bazitool "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"

func isCurrentBaziCalendarRule(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	version, _ := result["calendar_rule_version"].(string)
	return version == bazitool.CalendarRuleVersion
}
