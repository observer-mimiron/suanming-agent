package bazi

import (
	"log"
	"os"

	"github.com/wikiglobal/suanming-agent/internal/specialists"
)

// loadInstruction 从 prompts/interpret.md 读取八字 specialist 的系统提示词。
// 文件不存在时使用内置 fallback，保证服务可启动。
func loadInstruction() string {
	data, err := os.ReadFile("prompts/interpret.md")
	if err != nil {
		log.Printf("WARNING: failed to load prompts/interpret.md: %v, using fallback", err)
		return fallbackInstruction
	}
	return string(data)
}

// fallbackInstruction 在 interpret.md 文件不可用时使用的最小指令。
const fallbackInstruction = `你是八字命理专家。命盘数据已由系统预执行并注入在上方上下文中，直接引用解读。你只有 knowledge_catalog 和 knowledge_search 两个工具。不得自行推算四柱，不得伪造典籍出处。`

var cfg = specialists.Config{
	Domain:      "bazi",
	Name:        "bazi_specialist",
	Description: "八字命理专家。根据出生信息排盘，分析八字格局、用神、大运、流年。",
	Instruction: loadInstruction(),
	ToolNames:   []string{"knowledge_catalog", "knowledge_search"},
}

func Register(r *specialists.Registry) {
	r.Register(cfg)
}

// GetConfig 返回 bazi specialist 的当前配置，供测试使用。
func GetConfig() specialists.Config {
	return cfg
}
