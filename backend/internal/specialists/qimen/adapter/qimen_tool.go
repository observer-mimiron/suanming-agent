// 本文件属于奇门 adapter 层。
// 本文件负责调用外部 qimen-go 生成奇门盘，并将已验收盘面恢复为公共 tools.Tool 的 map payload；
// 不负责 Manager Prefill、Case/Session 写入、领域规则、trace、SSE 或最终文本。
package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/6tail/lunar-go/calendar"
	"github.com/deminzhang/qimen-go/qimen"
	qimendomain "github.com/observer-mimiron/suanming-agent/internal/specialists/qimen/domain"
)

// Tool 奇门遁甲排盘工具（时家奇门）。依据问事时间起局，生成转盘八门八神九宫格局，
// 不接收出生资料，也不把年月日时分拆成对外参数。
type Tool struct{}

// Name 返回注册表使用的奇门排盘工具名。
func (t *Tool) Name() string { return "qimen_dunjia" }

// Description 返回供模型和工具注册表使用的工具说明。
func (t *Tool) Description() string { return "奇门遁甲排盘，返回时家奇门九宫信息" }

// Label 返回面向事件和 trace 的中文工具标签。
func (t *Tool) Label() string { return "奇门遁甲" }

// Execute 校验问事时间并生成保持旧 map 合同的转盘奇门盘面。
// 参数缺失、未知或非 RFC3339 时间返回错误；排盘符号违反 rotating_8 合同时拒绝结果。
func (t *Tool) Execute(_ context.Context, params map[string]any) (any, error) {
	for key := range params {
		if key != "question_time" {
			return nil, fmt.Errorf("unknown qimen parameter %q", key)
		}
	}
	rawQuestionTime, ok := params["question_time"].(string)
	if !ok || rawQuestionTime == "" {
		return nil, fmt.Errorf("question_time is required")
	}
	questionTime, err := time.Parse(time.RFC3339, rawQuestionTime)
	if err != nil {
		return nil, fmt.Errorf("question_time must be RFC3339: %w", err)
	}
	if questionTime.Year() < 1900 || questionTime.Year() > 2100 {
		return nil, fmt.Errorf("question_time year out of range")
	}

	solar := calendar.NewSolar(questionTime.Year(), int(questionTime.Month()), questionTime.Day(), questionTime.Hour(), questionTime.Minute(), questionTime.Second())
	pan := qimen.NewQMGame(solar, qimen.QMParams{
		Type:        qimen.QMTypeRotating,  // 默认用传统转盘，避免九门/九神混入八门八神展示。
		HostingType: qimen.QMHostingType28, // 阳艮阴坤寄宫法
		FlyType:     qimen.QMFlyTypeAllOrder,
		JuType:      qimen.QMJuTypeSplit, // 拆补法
		HideGanType: 0,
		YMDH:        qimen.QMGameHour, // 时家奇门
	})

	pan.ShowTimeGame() // 设置 ShowPan 并填充 JieQi、JuText、DutyText
	pp := pan.ShowPan

	palaceNames := map[int]string{1: "坎", 2: "坤", 3: "震", 4: "巽", 5: "中", 6: "乾", 7: "兑", 8: "艮", 9: "离"}

	cells := make([]qimendomain.Cell, 9)
	for i := 1; i <= 9; i++ {
		g := pp.Gongs[i]
		cells[i-1] = qimendomain.Cell{
			Palace:   palaceNames[i],
			Door:     g.Door,
			Star:     g.Star,
			God:      g.God,
			HostGan:  g.HostGan,
			GuestGan: g.GuestGan,
		}
	}

	method := "拆补"
	if qimen.QMJuType[pp.StartType] == "置闰" {
		method = "置闰"
	} else if qimen.QMJuType[pp.StartType] == "茅山" {
		method = "茅山"
	}
	chart := qimendomain.Chart{
		PanType:          "时家奇门",
		PanSchema:        "rotating_8",
		SymbolSystem:     "eight_gate_eight_god",
		TimeSource:       "question_time",
		QuestionTime:     questionTime.Format(time.RFC3339),
		Method:           method,
		ValueStar:        pp.DutyStar,
		ValueDoor:        pp.DutyDoor,
		DutyPalace:       palaceNames[pp.DutyStarPos],
		DutyStarPalace:   palaceNames[pp.DutyStarPos],
		DutyDoorPalace:   palaceNames[pp.DutyDoorPos],
		DutyStarPosition: pp.DutyStarPos,
		DutyDoorPosition: pp.DutyDoorPos,
		JuText:           pp.JuText,
		DutyText:         pp.DutyText,
		JieQi:            pp.JieQi,
		Cells:            cells,
	}
	if err := chart.Validate(); err != nil {
		return nil, err
	}
	return chartPayload(chart), nil
}

// chartPayload preserves the existing map-shaped tool result at the adapter boundary.
func chartPayload(chart qimendomain.Chart) map[string]any {
	cells := make([]map[string]any, len(chart.Cells))
	for i, cell := range chart.Cells {
		cells[i] = map[string]any{
			"palace":    cell.Palace,
			"door":      cell.Door,
			"star":      cell.Star,
			"god":       cell.God,
			"host_gan":  cell.HostGan,
			"guest_gan": cell.GuestGan,
		}
	}
	return map[string]any{
		"pan_type":           chart.PanType,
		"pan_schema":         chart.PanSchema,
		"symbol_system":      chart.SymbolSystem,
		"time_source":        chart.TimeSource,
		"question_time":      chart.QuestionTime,
		"method":             chart.Method,
		"value_star":         chart.ValueStar,
		"value_door":         chart.ValueDoor,
		"duty_palace":        chart.DutyPalace, // legacy: 值符宫，前端新字段用 duty_star_palace。
		"duty_star_palace":   chart.DutyStarPalace,
		"duty_door_palace":   chart.DutyDoorPalace,
		"duty_star_position": chart.DutyStarPosition,
		"duty_door_position": chart.DutyDoorPosition,
		"ju_text":            chart.JuText,
		"duty_text":          chart.DutyText,
		"jie_qi":             chart.JieQi,
		"cells":              cells,
	}
}
