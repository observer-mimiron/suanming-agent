// Package prompts 通过 go:embed 将领域专家的系统提示词嵌入二进制，
// 避免运行时 os.ReadFile 的 cwd 依赖，runtime 和 test 行为一致。
//
// 新增 specialist 提示词时：
//  1. 将 .md 文件放入本包目录
//  2. 在下方添加 //go:embed 指令和导出变量
//  3. 在对应 specialist.go 中引用 prompts.XxxInstruction
package prompts

import _ "embed"

//go:embed interpret.md
var BaziInstruction string

//go:embed bazi_constitution.md
var BaziConstitutionInstruction string

//go:embed bazi_methodology_charter.md
var BaziMethodologyCharterInstruction string

//go:embed bazi_analysis_planner.md
var BaziAnalysisPlannerInstruction string

//go:embed bazi_static_synthesis.md
var BaziStaticSynthesisInstruction string

//go:embed bazi_dynamic_synthesis.md
var BaziDynamicSynthesisInstruction string

//go:embed bazi_canonical_synthesis.md
var BaziCanonicalSynthesisInstruction string

//go:embed bazi_contract_audit.md
var BaziContractAuditInstruction string

//go:embed bazi_evidence_planner.md
var BaziEvidencePlannerInstruction string

//go:embed qimen.md
var QimenInstruction string

//go:embed ziwei.md
var ZiWeiInstruction string

//go:embed supervisor/unified_router.md
var SupervisorUnifiedRouter string
