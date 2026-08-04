// This file belongs to the Qi Men specialist layer.
// It owns domain specialist configuration for this package.
// It configures Qi Men worker behavior; chart construction stays deterministic.
package qimen

import (
	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

var cfg = specialists.Config{
	Domain:               "qimen",
	Name:                 "qimen_specialist",
	Description:          "奇门遁甲专家。分析当前时空的吉凶方位、门星神组合。",
	Instruction:          prompts.QimenInstruction,
	ToolNames:            []string{"knowledge_catalog", "knowledge_search"},
	InjectSessionContext: true,
}

// GetConfig 返回 qimen specialist 的当前配置，供测试使用。
func GetConfig() specialists.Config {
	return cfg
}
