// package adapter 包含 Manager 拥有的执行主链。
//
// 本文件负责八字内层 agent 的构建接线和配置；
// 不负责 Graph 拓扑、会话所有权或最终答复。
package adapter

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

// baziEvidencePlannerConfig 构造受限证据规划模型，将当前问题和命盘事实改写为最多两条按典籍限定的查询。
func baziEvidencePlannerConfig() specialists.Config {
	cfg := newBaziCharterConfig("bazi_evidence_planner", "八字证据规划器", prompts.BaziEvidencePlannerInstruction, false, true, structuredSchemaBaziEvidencePlan)
	cfg.UseFastModel = true
	return cfg
}

func baziAnalysisPlannerConfig() specialists.Config {
	cfg := newBaziCharterConfig("bazi_analysis_planner", "八字分析模式判定器", prompts.BaziAnalysisPlannerInstruction, false, true, structuredSchemaBaziAnalysisPlan)
	cfg.UseFastModel = true
	return cfg
}

func baziStaticSynthesisConfig() specialists.Config {
	return newBaziCharterConfig("bazi_static_synthesis", "八字静态综合器", prompts.BaziStaticSynthesisInstruction, false, true, structuredSchemaBaziStaticSynthesis)
}

func baziDynamicSynthesisConfig() specialists.Config {
	return newBaziCharterConfig("bazi_dynamic_synthesis", "八字动态综合器", prompts.BaziDynamicSynthesisInstruction, false, true, structuredSchemaBaziDynamicSynthesis)
}

// baziLifetimeDayunSynthesisConfig owns the all-period reading and must remain
// separate from the current-period dynamic contract.
func baziLifetimeDayunSynthesisConfig() specialists.Config {
	return newBaziCharterConfig("bazi_lifetime_dayun_synthesis", "八字全程大运综合器", prompts.BaziLifetimeDayunSynthesisInstruction, false, true, structuredSchemaBaziLifetimeDayunSynthesis)
}

func newBaziCharterConfig(name, description, instruction string, withKnowledgeTools, useJSON bool, schemaName string) specialists.Config {
	cfg := specialists.Config{
		UseJSONMode:          useJSON,
		Domain:               "bazi",
		Name:                 name,
		Description:          description,
		Instruction:          strings.TrimSpace(prompts.BaziMethodologyCharterInstruction + "\n\n" + prompts.BaziConstitutionInstruction + "\n\n" + instruction),
		InjectSessionContext: false,
		StructuredSchema:     schemaName,
	}
	if withKnowledgeTools {
		cfg.ToolNames = []string{"knowledge_catalog", "knowledge_search"}
	}
	return cfg
}
