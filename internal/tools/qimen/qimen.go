// Package qimen 提供奇门遁甲时家奇门排盘功能。基于 qimen-go 库实现九宫、八门、八神、九星的推演，
// 支持拆补法和置闰法两种起局方式，适用于择吉、决策和命理咨询场景。
package qimen

import (
	"context"
	"fmt"

	"github.com/6tail/lunar-go/calendar"
	"github.com/deminzhang/qimen-go/qimen"
)

// Tool 奇门遁甲排盘工具（时家奇门）。依据给定的年月日时分起局，生成九宫格局信息，
// 包括八门（休生伤杜景死惊开）、九星（天蓬天芮等）、八神（值符腾蛇等）和引干等。
type Tool struct{}

func (t *Tool) Name() string        { return "qimen_dunjia" }
func (t *Tool) Description() string { return "奇门遁甲排盘，返回时家奇门九宫信息" }

func (t *Tool) Label() string { return "奇门遁甲" }

func (t *Tool) Execute(_ context.Context, params map[string]any) (any, error) {
	year, _ := params["year"].(float64)
	month, _ := params["month"].(float64)
	day, _ := params["day"].(float64)
	hour, _ := params["hour"].(float64)
	minute, _ := params["minute"].(float64)

	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("year out of range")
	}
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("month out of range")
	}
	if day < 1 || day > 31 {
		return nil, fmt.Errorf("day out of range")
	}
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour out of range")
	}

	solar := calendar.NewSolar(int(year), int(month), int(day), int(hour), int(minute), 0)
	pan := qimen.NewQMGame(solar, qimen.QMParams{
		Type:        qimen.QMTypeAmaze,     // 鸣法（含转盘+飞盘信息）
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
			"palace":      palaceNames[i],
			"door":        g.Door,
			"star":        g.Star,
			"god":         g.God,
			"host_gan":    g.HostGan,
			"guest_gan":   g.GuestGan,
		}
	}

	method := "拆补"
	if qimen.QMJuType[pp.StartType] == "置闰" {
		method = "置闰"
	} else if qimen.QMJuType[pp.StartType] == "茅山" {
		method = "茅山"
	}

	return map[string]any{
		"pan_type":       "时家奇门",
		"question_time":  fmt.Sprintf("%d-%02d-%02dT%02d:%02d:00+08:00", int(year), int(month), int(day), int(hour), int(minute)),
		"method":         method,
		"value_star":     pp.DutyStar,
		"value_door":     pp.DutyDoor,
		"duty_palace":    palaceNames[pp.DutyStarPos],
		"ju_text":        pp.JuText,
		"duty_text":      pp.DutyText,
		"jie_qi":         pp.JieQi,
		"cells":          cells,
	}, nil
}

