# 八字输出合同

## 目标

八字 runtime 只做三件事：提供可复算事实、组织 evidence bundle、校验明显结构错误。命理裁断由静态/动态 synthesis 输出，renderer 只格式化。

## 数据边界

1. `chart_facts`：四柱、藏干层级、十神映射、透干、月令、历法版本、大运日期边界与标准关系表。它们由 Go 工具计算，可复算。
2. `rule_materials`：检索到的古籍证据、方法论提示或未来显式规则表。当前 runtime 不再注入默认 `ziping_classic_v1` profile，也不生成 claim。
3. `assertion`：模型输出的最小结构化裁断。每条声明 `kind`、`verdict`、`fact_refs`、可选 `claim_refs` 与 `boundary`。
4. `verdict`：旧字段兼容层，来自模型输出或 assertion 投影，供 renderer 成文。

## 当前决策

- 删除运行时默认 rule profile：不再有 `defaultBaziRuleProfile`、`applyZipingBasicClaims`、`applyZipingMonthJieClaim` 或调候单行 overlay。
- `selected_rule_profile` 可为空；为空时模型不得虚构 claim_refs，validator 不因“缺 profile claim”降级。
- `claim_refs` 只做来源审计：未知 claim 进入 trace soft audit，不触发整段重试或 facts-only 兜底。
- 大运干支、起运时间、日期边界、十神和已计算关系仍由 Go 工具负责；大运趋势由动态 synthesis 负责。
- recovery 只做 enum 规范化和 facts-only 降级，不做中文短语替换，不把单个命盘样例写入运行时代码。

## Validator 口径

硬失败只保留：

- 结构字段缺失，导致 renderer 无法成文。
- 可证明的事实冲突，例如逐运 assertion 与计算大运干支不一致。
- 大运覆盖缺失：已计算 period 没有对应动态输出。
- 未声明关系被写成事实，例如工具未返回的三合三会、冲刑害。
- 直接医疗、法律、伤灾、具体财务事故等高风险应事。

软审计只记录：

- 未知 `fact_refs` 别名。
- 未知 `claim_refs`。
- 普通命理强词、层次措辞或未实现流派表达。
- broad tendency 是否过强这类需要 eval 判断的问题。

## Renderer 口径

- renderer 不读取问题关键词推导主轴、强弱、层级、大运吉凶。
- renderer 不从 `佩印`、`先丙后癸`、`月劫` 等词补充取用、建议或解释。
- renderer 只展示 synthesis 字段；字段缺失时显示“上游未提供”，不自行补命理判断。

## Eval 口径

- 单个命盘、trace 或用户样例只能进入 `backend/internal/runtime/testdata/bazi_eval_cases/` 等 fixture。
- fixture 可记录期望文本、禁止文本、assertion 覆盖和 violation，但不能推动 runtime 新增专项分支。
- 行业/流派规则要么进入检索知识库和 prompt，要么未来做成数据驱动规则表；不能散落在 Go runtime 分支里。
