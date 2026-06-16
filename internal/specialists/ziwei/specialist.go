// Package ziwei 实现了紫微斗数领域专家 AgentTool 配置。
// Config 由 Register() 注册到 Registry，由 runtime.AgentBuilder 构建 ChatModelAgent。
package ziwei

import "github.com/wikiglobal/suanming-agent/internal/specialists"

// Register 向 Registry 注册紫微斗数领域专家 AgentTool 配置。
func Register(r *specialists.Registry) {
	r.Register(specialists.Config{
		Domain:      "ziwei",
		Name:        "ziwei_specialist",
		Description: "紫微斗数专家。根据出生信息排盘，分析十二宫星曜布局、四化飞星。",
		Instruction: `你是紫微斗数专家。

## 可调用工具
- ziwei_calc：排紫微斗数命盘（需要出生年月日时和性别）
- knowledge_search：检索古籍原文

## 执行规则
1. 用户提供了出生信息 → 调 ziwei_calc 排盘
2. 排盘后 → 调 knowledge_search 获取古籍原文
3. 分析命宫、身宫、三方四正的星曜组合，结合大限流年判断运势

## 输出要求
- 中文表达，专业但不晦涩
- 引用古籍时标注出处`,
		ToolNames: []string{"ziwei_calc", "knowledge_search"},
	})
}
