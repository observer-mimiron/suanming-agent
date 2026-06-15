// Package ziwei 提供紫微斗数排盘功能。包含安命宫身宫、定五行局、安紫微天府、
// 安主星辅星、安长生博士十二神、起大限、安杂曜和流年分析等完整算法。
// 紫微斗数以年柱天干地支为核心，通过紫微星系和天府星系的分布构建十二宫星曜布局。
package ziwei

import (
	"context"
	"fmt"

	"github.com/6tail/lunar-go/calendar"
)

// ZiWeiCalcTool 紫微斗数排盘工具。根据出生年月日时和性别，排布紫微斗数十二宫命盘，
// 包含主星（紫微、天机、太阳、武曲、天同等14主星）、辅星（文昌、文曲、左辅、右弼、天魁、天钺等14辅星）、
// 杂曜（红鸾、天喜、三台、八座等数十种）、长生十二神、博士十二神和大限信息。
type ZiWeiCalcTool struct{}

func (t *ZiWeiCalcTool) Name() string        { return "ziwei_calc" }
func (t *ZiWeiCalcTool) Description() string { return "紫微斗数排盘，输入出生年月日时+性别，返回命盘十二宫星曜布局" }

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

	y, m, d, h := int(year), int(month), int(day), int(hour)

	solar := calendar.NewSolar(y, m, d, h, 0, 0)
	timeIndex := TimeToIndex(h)

	chart, err := BuildChart(solar, timeIndex, gender)
	if err != nil {
		return nil, fmt.Errorf("紫微斗数排盘失败: %w", err)
	}

	return chart.ToMap(), nil
}
