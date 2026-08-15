// Package adapter 包含八字运行时载荷的局部辅助。
//
// 本文件负责动态值归一化、简单标量边界和跨节点文本判断；
// 不负责执行合同、Manager 流程或最终答复。
package adapter

import (
	"fmt"
	"strings"
)

// stringValue 仅在原始值已经是字符串时返回文本。
func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

// intValue 将常见 JSON 数字形态转换为整数，供合同门禁使用。
func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

// minInt 返回两个整数中的较小值，供 runtime 合同边界计算使用。
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// containsAnyText 判断文本集合中是否包含任一目标片段。
func containsAnyText(texts []string, needles []string) bool {
	for _, text := range texts {
		for _, needle := range needles {
			if needle != "" && strings.Contains(text, needle) {
				return true
			}
		}
	}
	return false
}

// anyToString 将动态载荷的值转成文本，隔离调用方的类型断言。
func anyToString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}
