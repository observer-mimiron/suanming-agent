package ziwei

import (
	"github.com/wikiglobal/suanming-agent/internal/prompts"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
)

var cfg = specialists.Config{
	Domain:      "ziwei",
	Name:        "ziwei_specialist",
	Description: "紫微斗数专家。根据出生信息排盘，分析十二宫星曜布局、四化飞星。",
	Instruction: prompts.ZiWeiInstruction,
	ToolNames:   []string{"knowledge_catalog", "knowledge_search", "ziwei_liunian"},
}

func Register(r *specialists.Registry) {
	r.Register(cfg)
}

// GetConfig 返回 ziwei specialist 的当前配置，供测试使用。
func GetConfig() specialists.Config {
	return cfg
}
