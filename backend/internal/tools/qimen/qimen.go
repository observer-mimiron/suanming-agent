// Package qimen 提供奇门遁甲时家奇门排盘功能。基于 qimen-go 库实现九宫、八门、八神、九星的推演，
// 支持拆补法和置闰法两种起局方式，适用于择吉、决策和命理咨询场景。
package qimen

import (
	"context"
	"fmt"
	"time"

	"github.com/6tail/lunar-go/calendar"
	"github.com/deminzhang/qimen-go/qimen"
)

// Tool 奇门遁甲排盘工具（时家奇门）。依据问事时间起局，生成转盘八门八神九宫格局，
// 不接收出生资料，也不把年月日时分拆成对外参数。
type Tool struct{}

func (t *Tool) Name() string        { return "qimen_dunjia" }
func (t *Tool) Description() string { return "奇门遁甲排盘，返回时家奇门九宫信息" }

func (t *Tool) Label() string { return "奇门遁甲" }

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

	cells := make([]map[string]any, 9)
	for i := 1; i <= 9; i++ {
		g := pp.Gongs[i]
		cells[i-1] = map[string]any{
			"palace":    palaceNames[i],
			"door":      g.Door,
			"star":      g.Star,
			"god":       g.God,
			"host_gan":  g.HostGan,
			"guest_gan": g.GuestGan,
		}
	}
	if err := validateRotatingSymbols(cells); err != nil {
		return nil, err
	}

	method := "拆补"
	if qimen.QMJuType[pp.StartType] == "置闰" {
		method = "置闰"
	} else if qimen.QMJuType[pp.StartType] == "茅山" {
		method = "茅山"
	}

	return map[string]any{
		"pan_type":           "时家奇门",
		"pan_schema":         "rotating_8",
		"symbol_system":      "eight_gate_eight_god",
		"time_source":        "question_time",
		"question_time":      questionTime.Format(time.RFC3339),
		"method":             method,
		"value_star":         pp.DutyStar,
		"value_door":         pp.DutyDoor,
		"duty_palace":        palaceNames[pp.DutyStarPos], // legacy: 值符宫，前端新字段用 duty_star_palace。
		"duty_star_palace":   palaceNames[pp.DutyStarPos],
		"duty_door_palace":   palaceNames[pp.DutyDoorPos],
		"duty_star_position": pp.DutyStarPos,
		"duty_door_position": pp.DutyDoorPos,
		"ju_text":            pp.JuText,
		"duty_text":          pp.DutyText,
		"jie_qi":             pp.JieQi,
		"cells":              cells,
	}, nil
}

// validateRotatingSymbols rejects symbols from other Qi Men display systems;
// silently replacing them would make the returned rotating_8 chart inaccurate.
func validateRotatingSymbols(cells []map[string]any) error {
	for _, cell := range cells {
		switch cell["door"] {
		case "中门", "中":
			return fmt.Errorf("rotating_8 does not allow door symbol %q", cell["door"])
		}
		switch cell["god"] {
		case "太常", "勾陈", "朱雀":
			return fmt.Errorf("rotating_8 does not allow god symbol %q", cell["god"])
		}
	}
	return nil
}
