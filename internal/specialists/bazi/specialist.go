package bazi

import (
	"github.com/wikiglobal/suanming-agent/internal/prompts"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
)

var cfg = specialists.Config{
	Domain:      "bazi",
	Name:        "bazi_specialist",
	Description: "八字命理专家。根据出生信息排盘，分析八字格局、用神、大运、流年。",
	Instruction: prompts.BaziInstruction,
	ToolNames:   []string{"knowledge_catalog", "knowledge_search"},
}

func Register(r *specialists.Registry) {
	r.Register(cfg)
}

// GetConfig 返回 bazi specialist 的当前配置，供测试使用。
func GetConfig() specialists.Config {
	return cfg
}
