// This file belongs to the BaZi deterministic calculation layer.
// It owns BaZi calendar rule constants for this package.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

const (
	// CalendarRuleVersion 标记当前八字排盘使用的历法口径版本。
	// 运行时用它识别旧缓存命盘，确保历史会话里的旧口径结果会被自动重排。
	CalendarRuleVersion = "zi_zheng_true_solar_v2"
)
