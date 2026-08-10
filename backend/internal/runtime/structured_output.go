// Package runtime 包含 Manager 拥有的结构化输出合同。
//
// 本文件负责 Schema registry、prompt 注入和原始输出严格解码；
// 不负责八字语义、恢复策略、渲染或 SSE 表示。
package runtime

import (
	_ "embed"
	"fmt"

	"github.com/observer-mimiron/suanming-agent/internal/structured"
)

var baziStructuredConfidenceValues = []string{"保守判断", "倾向成立", "明确成立"}

// 以下文件是当前活跃节点的 Draft-07 合同唯一来源；prompt 与校验器读取同一份原文。
//
//go:embed schemas/bazi-analysis-plan.schema.json
var baziAnalysisPlanSchema []byte

//go:embed schemas/bazi-evidence-plan.schema.json
var baziEvidencePlanSchema []byte

//go:embed schemas/bazi-static-synthesis.schema.json
var baziStaticSynthesisSchema []byte

//go:embed schemas/bazi-dynamic-synthesis.schema.json
var baziDynamicSynthesisSchema []byte

//go:embed schemas/bazi-lifetime-dayun-synthesis.schema.json
var baziLifetimeDayunSynthesisSchema []byte

const (
	structuredSchemaBaziAnalysisPlan           = "bazi.analysis_plan"
	structuredSchemaBaziEvidencePlan           = "bazi.evidence_plan"
	structuredSchemaBaziStaticSynthesis        = "bazi.static_synthesis"
	structuredSchemaBaziDynamicSynthesis       = "bazi.dynamic_synthesis"
	structuredSchemaBaziLifetimeDayunSynthesis = "bazi.lifetime_dayun_synthesis"
)

// init 启动时逐份注册节点合同，避免 bundle 让独立节点共享不可维护的大 Schema 文件。
func init() {
	for _, schema := range []struct {
		name string
		raw  []byte
	}{
		{structuredSchemaBaziAnalysisPlan, baziAnalysisPlanSchema},
		{structuredSchemaBaziEvidencePlan, baziEvidencePlanSchema},
		{structuredSchemaBaziStaticSynthesis, baziStaticSynthesisSchema},
		{structuredSchemaBaziDynamicSynthesis, baziDynamicSynthesisSchema},
		{structuredSchemaBaziLifetimeDayunSynthesis, baziLifetimeDayunSynthesisSchema},
	} {
		if err := structured.RegisterJSON(schema.name, schema.raw); err != nil {
			panic(err)
		}
	}
}

// structuredOutputPromptContract 将校验器使用的同一份 Schema 注入结构化 prompt。
func structuredOutputPromptContract(name string) (string, error) {
	return structured.PromptContract(name)
}

// structuredSchemaHash 返回稳定指纹，供 prompt 与校验器同源回归测试使用。
func structuredSchemaHash(name string) (string, error) {
	return structured.Hash(name)
}

// decodeStructuredOutput 在 DTO 进入领域校验前拒绝空、fence 或不合规的原始输出。
func decodeStructuredOutput(name, raw string, target any) error {
	if err := structured.Decode(name, raw, target); err != nil {
		return err
	}
	return validateStructuredDTO(name, target)
}

// validateStructuredDTO 拒绝结构正确但违反节点语义的原始模型输出。
func validateStructuredDTO(name string, target any) error {
	switch value := target.(type) {
	case *baziStructuredStaticSynthesis:
		_ = value
	case *baziStructuredDynamicSynthesis:
		for index, claim := range value.PeriodClaims {
			if err := validateStructuredClaimConfidence(name, fmt.Sprintf("period_claims[%d]", index), claim.Confidence); err != nil {
				return err
			}
		}
		if err := validateStructuredClaimConfidence(name, "liunian_claim", value.LiunianClaim.Confidence); err != nil {
			return err
		}
	case *baziLifetimeDayunSynthesis:
		for index, claim := range value.PeriodClaims {
			if err := validateStructuredClaimConfidence(name, fmt.Sprintf("period_claims[%d]", index), claim.Confidence); err != nil {
				return err
			}
		}
	case *baziAnalysisPlan:
		if value.WriterTemplate == "topic" && value.TopicMode == "" {
			return fmt.Errorf("schema_error[%s]: topic writer requires topic_mode", name)
		}
	}
	return nil
}

// validateStructuredClaimConfidence keeps the Go semantic vocabulary identical
// to the confidence enum in every BaZi synthesis Schema without coupling DTO shapes.
func validateStructuredClaimConfidence(name, field, confidence string) error {
	if !containsString(baziStructuredConfidenceValues, confidence) {
		return fmt.Errorf("schema_error[%s]: %s.confidence is outside the closed model enum", name, field)
	}
	return nil
}
