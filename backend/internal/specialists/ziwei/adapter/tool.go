// Package adapter 提供紫微 specialist 的确定性排盘和流年工具适配。
// 本包保留既有工具名称、参数校验和 map payload，不负责模型、Session、trace、SSE 或最终文本。
package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/6tail/lunar-go/calendar"
	solartime "github.com/observer-mimiron/suanming-agent/internal/calendar"
)

// ZiWeiCalcTool 紫微斗数排盘工具。根据出生年月日时和性别，排布紫微斗数十二宫命盘，
// 包含主星（紫微、天机、太阳、武曲、天同等14主星）、辅星（文昌、文曲、左辅、右弼、天魁、天钺等14辅星）、
// 杂曜（红鸾、天喜、三台、八座等数十种）、长生十二神、博士十二神和大限信息。
type ZiWeiCalcTool struct{}

func (t *ZiWeiCalcTool) Name() string { return "ziwei_calc" }
func (t *ZiWeiCalcTool) Description() string {
	return "紫微斗数排盘，输入出生年月日时+性别，返回命盘十二宫星曜布局"
}

func (t *ZiWeiCalcTool) Label() string { return "紫微排盘" }

func (t *ZiWeiCalcTool) Execute(_ context.Context, params map[string]any) (any, error) {
	year, ok := params["year"].(float64)
	if !ok || year < 1900 || year > 2100 {
		return nil, fmt.Errorf("year out of range")
	}
	month, ok := params["month"].(float64)
	if !ok || month < 1 || month > 12 {
		return nil, fmt.Errorf("month out of range")
	}
	day, ok := params["day"].(float64)
	if !ok || day < 1 || day > 31 {
		return nil, fmt.Errorf("day out of range")
	}
	hour, ok := params["hour"].(float64)
	if !ok || hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour out of range")
	}
	gender, ok := params["gender"].(string)
	if !ok || (gender != "男" && gender != "女") {
		return nil, fmt.Errorf("gender must be 男/女")
	}

	solar, timeIndex := correctedBirthSolar(int(year), int(month), int(day), int(hour), params)

	chart, err := BuildChart(solar, timeIndex, gender)
	if err != nil {
		return nil, fmt.Errorf("紫微斗数排盘失败: %w", err)
	}

	result := chart.ToMap()
	result["solar_time_version"] = solartime.TrueSolarTimeVersion
	result["birthday"] = solar.ToYmdHms()
	return result, nil
}

func correctedBirthSolar(year, month, day, hour int, params map[string]any) (*calendar.Solar, int) {
	minute := 0
	if value, ok := params["minute"].(float64); ok {
		minute = int(value)
	}
	instant := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
	if longitude, ok := params["longitude"].(float64); ok && longitude >= -180 && longitude <= 180 {
		instant = instant.Add(time.Duration(solartime.TrueSolarOffsetMinutes(year, month, day, longitude)) * time.Minute)
	}
	solar := calendar.NewSolar(instant.Year(), int(instant.Month()), instant.Day(), instant.Hour(), instant.Minute(), 0)
	return solar, TimeToIndex(instant.Hour())
}
