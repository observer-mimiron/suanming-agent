package runtime

import bazitool "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"

func isCurrentBaziCalendarRule(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	version, _ := result["calendar_rule_version"].(string)
	return version == bazitool.CalendarRuleVersion
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
