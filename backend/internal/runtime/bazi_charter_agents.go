package runtime

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// BuildEphemeralInnerAgent 复用现有 specialist 构建链，为内层 bazi graph 创建一次性 agent。
// 在测试场景中若尚未注入模型或工具注册表，则返回 nil agent 但不报错，用于先验证接口与调用链存在。
func (b *AgentBuilder) BuildEphemeralInnerAgent(ctx context.Context, cfg specialists.Config, st *state.SessionState) (adk.Agent, error) {
	if b == nil || b.model == nil || b.reg == nil {
		return nil, nil
	}
	return b.BuildSpecialist(ctx, cfg, st)
}

func baziEvidencePlannerConfig() specialists.Config {
	cfg := newBaziCharterConfig("bazi_evidence_planner", "八字证据规划器", prompts.BaziEvidencePlannerInstruction, false, true)
	cfg.UseFastModel = true
	return cfg
}

func baziAnalysisPlannerConfig() specialists.Config {
	cfg := newBaziCharterConfig("bazi_analysis_planner", "八字分析模式判定器", prompts.BaziAnalysisPlannerInstruction, false, true)
	cfg.UseFastModel = true
	return cfg
}

func baziStaticSynthesisConfig() specialists.Config {
	return newBaziCharterConfig("bazi_static_synthesis", "八字静态综合器", prompts.BaziStaticSynthesisInstruction, false, true)
}

func baziDynamicSynthesisConfig() specialists.Config {
	return newBaziCharterConfig("bazi_dynamic_synthesis", "八字动态综合器", prompts.BaziDynamicSynthesisInstruction, false, true)
}

func newBaziCharterConfig(name, description, instruction string, withKnowledgeTools, useJSON bool) specialists.Config {
	cfg := specialists.Config{
		UseJSONMode:          useJSON,
		Domain:               "bazi",
		Name:                 name,
		Description:          description,
		Instruction:          strings.TrimSpace(prompts.BaziMethodologyCharterInstruction + "\n\n" + prompts.BaziConstitutionInstruction + "\n\n" + instruction),
		InjectSessionContext: false,
	}
	if withKnowledgeTools {
		cfg.ToolNames = []string{"knowledge_catalog", "knowledge_search"}
	}
	return cfg
}
