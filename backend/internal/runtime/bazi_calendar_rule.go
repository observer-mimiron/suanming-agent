// Package runtime 包含 Manager 拥有的资产兼容门禁。
//
// 本文件负责为不同领域资产选择历法/方法版本并校验缓存是否仍可复用；
// 不负责排盘算法、ExecutionPlan 路由、Graph 编排或用户可见答复。
package runtime

import bazitool "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"

// currentBaziCalendarRule 返回缓存八字盘资产必须使用的历法版本。
func currentBaziCalendarRule() string {
	return bazitool.CalendarRuleVersion
}

// isCurrentBaziCalendarRule 判断缓存八字盘是否使用当前历法口径。
func isCurrentBaziCalendarRule(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	version, _ := result["calendar_rule_version"].(string)
	return version == currentBaziCalendarRule()
}

// isCurrentZiWeiSolarTime 判断缓存紫微盘是否使用当前真太阳时口径。
func isCurrentZiWeiSolarTime(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	return stringValue(result["solar_time_version"]) == bazitool.TrueSolarTimeVersion
}

// ziWeiMethodVersion 返回紫微资产持久化时使用的兼容版本。
func ziWeiMethodVersion() string {
	return "ziwei-" + bazitool.TrueSolarTimeVersion
}
