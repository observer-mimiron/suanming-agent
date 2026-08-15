// Package calendar 提供跨命理领域共用的历法计算。
//
// 本文件负责真太阳时版本和日期经度偏移；
// 不负责具体排盘、会话、模型、传输或用户答复。
package calendar

import (
	"math"
	"time"
)

// TrueSolarTimeVersion 标记已应用经度和均时差校正的出生时刻。
const TrueSolarTimeVersion = "true_solar_v2"

// TrueSolarOffsetMinutes 返回指定日期和经度相对中国标准时间的真太阳时分钟偏移。
func TrueSolarOffsetMinutes(year, month, day int, longitude float64) int {
	// NOAA 近似式在日期粒度的误差远小于当前排盘输入的分钟精度，避免引入额外天文依赖。
	dayOfYear := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC).YearDay()
	gamma := 2 * math.Pi / 365 * (float64(dayOfYear) - 1)
	equationOfTime := 229.18 * (0.000075 + 0.001868*math.Cos(gamma) - 0.032077*math.Sin(gamma) - 0.014615*math.Cos(2*gamma) - 0.040849*math.Sin(2*gamma))
	longitudeCorrection := (longitude - 120.0) * 4
	return int(math.Round(longitudeCorrection + equationOfTime))
}
