// 本文件属于奇门领域层。
// 本文件负责问事盘的纯领域结果与转盘八门八神符号合同；
// 不负责解析工具参数、访问 Session、模型、检索、trace、SSE 或用户输出。
package domain

import "fmt"

// Cell 表示奇门问事盘中的一个九宫格。
type Cell struct {
	Palace   string
	Door     string
	Star     string
	God      string
	HostGan  string
	GuestGan string
}

// Chart 表示已按问事时间生成的转盘八门八神盘面。
// 它是工具适配器与 runtime 之间的领域结果，不拥有会话或传输层状态。
type Chart struct {
	PanType          string
	PanSchema        string
	SymbolSystem     string
	TimeSource       string
	QuestionTime     string
	Method           string
	ValueStar        string
	ValueDoor        string
	DutyPalace       string
	DutyStarPalace   string
	DutyDoorPalace   string
	DutyStarPosition int
	DutyDoorPosition int
	JuText           string
	DutyText         string
	JieQi            string
	Cells            []Cell
}

// Validate 检查转盘八门八神盘面是否混入其他展示体系的符号。
// 返回错误时，调用方必须拒绝该盘面，不能静默替换符号后继续解释。
func (c Chart) Validate() error {
	for _, cell := range c.Cells {
		switch cell.Door {
		case "中门", "中":
			return fmt.Errorf("rotating_8 does not allow door symbol %q", cell.Door)
		}
		switch cell.God {
		case "太常", "勾陈", "朱雀":
			return fmt.Errorf("rotating_8 does not allow god symbol %q", cell.God)
		}
	}
	return nil
}
