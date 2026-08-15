// 本文件属于 runtime 执行层，负责确定性工具调用和出生资料参数转换。
// 它只负责把已确定的输入交给 ToolRunner，不负责路由、排盘编排或领域解释。
package runtime

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

func (e *Executor) callTool(ctx context.Context, name string, params map[string]any) map[string]any {
	if e == nil || e.reg == nil {
		return nil
	}
	if e.toolRunner == nil {
		e.toolRunner = tools.NewToolRunner(e.reg)
	}
	result := e.toolRunner.Run(ctx, tools.ToolRunRequest{
		ToolName:       name,
		Params:         params,
		DecisionSource: "prefill",
	})
	if result.Status != tools.ToolRunStatusOK && result.Status != tools.ToolRunStatusFallback {
		if result.Error != nil {
			log.Printf("prefill: tool %s failed: %v", name, result.Error)
		}
		return nil
	}
	m, _ := result.Data.(map[string]any)
	return m
}

func buildToolParams(profile map[string]any) map[string]any {
	year := toFloat(profile["year"])
	month := toFloat(profile["month"])
	day := toFloat(profile["day"])
	hour := toFloat(profile["hour"])
	gender := state.NormalizeGender(profile["gender"])
	if year == 0 || month == 0 || day == 0 || gender == "" {
		return nil
	}
	params := map[string]any{"year": year, "month": month, "day": day, "hour": hour, "gender": gender}
	if minute, ok := profile["minute"]; ok {
		params["minute"] = toFloat(minute)
	}
	if longitude, ok := profile["longitude"]; ok {
		params["longitude"] = toFloat(longitude)
	} else if longitude, ok := longitudeForBirthplace(stringValue(profile["birthplace"])); ok {
		params["longitude"] = longitude
	}
	return params
}

func longitudeForBirthplace(birthplace string) (float64, bool) {
	// 城市级出生地只能给出近似经度；显式 longitude 总是优先，避免用城市中心点
	// 覆盖用户的精确地点。表只覆盖当前产品收集的常见城市，其他地点保持不修正。
	longitudes := map[string]float64{
		"北京": 116.4074,
		"上海": 121.4737,
		"广州": 113.2644,
		"深圳": 114.0579,
		"成都": 104.0665,
		"重庆": 106.5516,
		"武汉": 114.3054,
		"西安": 108.9398,
		"杭州": 120.1551,
		"南京": 118.7969,
		"天津": 117.2000,
		"香港": 114.1694,
	}
	longitude, ok := longitudes[strings.TrimSpace(birthplace)]
	return longitude, ok
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}
