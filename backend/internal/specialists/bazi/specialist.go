// This file belongs to the BaZi specialist layer.
// It owns domain specialist configuration for this package.
// It configures BaZi worker behavior; deterministic chart facts stay in tools/runtime.
package bazi

import (
	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

var cfg = specialists.Config{
	Domain:               "bazi",
	Name:                 "bazi_specialist",
	Description:          "八字命理专家。根据出生信息排盘，分析八字格局、用神、大运、流年。",
	Instruction:          prompts.BaziInstruction,
	ToolNames:            []string{"knowledge_catalog", "knowledge_search"},
	InjectSessionContext: true,
}

// GetConfig 返回 bazi specialist 的当前配置，供测试使用。
func GetConfig() specialists.Config {
	return cfg
}
