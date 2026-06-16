// Package bazi 实现了八字（四柱命理）领域专家 AgentTool 配置。
// Config 由 Register() 注册到 Registry，由 runtime.AgentBuilder 构建 ChatModelAgent。
package bazi

import "github.com/wikiglobal/suanming-agent/internal/specialists"

// Register 向 Registry 注册八字领域专家 AgentTool 配置。
func Register(r *specialists.Registry) {
	r.Register(specialists.Config{
		Domain:      "bazi",
		Name:        "bazi_specialist",
		Description: "八字命理专家。根据出生时间排盘、分析用神忌神、解读大运走势。",
		Instruction: `你是八字命理专家。

## 可调用工具
- bazi_calc：排八字四柱命盘（需要年/月/日/时/性别）
- yongshen：分析日主强弱、取用神忌神（需要先有排盘结果）
- dayun_analyzer：分析大运走势、各步大运起止时间（需要先有排盘结果）
- knowledge_search：检索古籍原文（《渊海子平》《滴天髓》等）

## 执行规则
1. 用户提供了出生信息 → 先调 bazi_calc 排盘
2. 排盘后 → 根据需要调 yongshen 或 dayun_analyzer
3. 关键论断前 → 调 knowledge_search 获取古籍原文
4. 综合输出中文解读，引用古籍时标注出处

## 禁止
- 不得自行推算四柱（以系统排盘结果为准）
- 不得跳过排盘直接分析（除非 session 中已有命盘）`,
		ToolNames: []string{"bazi_calc", "yongshen", "dayun_analyzer", "knowledge_search"},
	})
}
