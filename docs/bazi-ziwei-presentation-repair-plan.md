# 八字成文与紫微星盘展示修复方案

> 状态：待实施。本文只处理用户可见报告和已缓存紫微命盘的投递，不改排盘算法、路由、八字 Graph、检索、重试预算或 SSE 协议。

## 目标与验收

### 八字报告

1. 不再向用户展示 `格局评价已定`、`格局判断暂定` 等内部验收状态。
2. 格局段直接说明已验收的主轴、优势、限制和成立条件；证据不足时说明缺的是哪项命理依据，不说明系统验收过程。
3. 全程每步大运均带一个明确标签：`偏吉`、`平`、`偏凶` 或 `待观察`，并保留已有的结构原因。
4. 总览放在全文最后，只收束已经展示过的本命、全程与当前信息。
5. 古籍短引文仅在能直接说明同段结论时展示；无对应关系则省略，不以检索命中代替证据。

### 紫微斗数

1. 用户明确请求紫微，且会话已有兼容紫微命盘时，仍向该轮 SSE 发送一次 `component/ziwei-chart`；不重新排盘。
2. 新排紫微盘和复用紫微盘都由同一前端 `ZiweiChartCard` 展示十二宫、主星、辅星和杂曜。
3. 紫微 specialist 已收到 `ZiWeiResult` 时，回答不得声称“没有完整十二宫星曜数据”；宫位型问题至少引用相关宫位的确定性星曜事实。若无法形成解释，只说明解释边界，不否认已存在的命盘。

## 已确认根因

### 八字

- `TierAssessmentJudgment` 把内部 `rated/provisional/withheld` 转为用户文本，renderer 在格局评价与总览直接输出该字段。
- 全程大运已有 `support_use` 等稳定枚举，但 renderer 只映射为“扶助用神”等结构标签，没有转换为用户能直接读懂的极性。
- 当前测试反而断言这些内部词必须出现；在线质量评测也没有“结论可理解、每运极性明确、引文相关”的合同。

### 紫微

- `prefillZiWeiForPlan` 命中活动缓存盘时只写入 `vals["ziwei_result"]`，没有发出 `ziwei-chart` component；只有新调用 `ziwei_calc` 时才发出。
- 最近 trace `trc_43c8023189f4`、`trc_ee2015ba3fc6` 都是 `reuse_cached_result=true`，且没有 `ziwei_calc` 和 component span。因此前端没有可渲染的星盘，尽管会话中已有紫微资产。
- `specialistSessionView` 已将 `ZiWeiResult` 投影给紫微 runner，但 `specialists/ziwei/application/BuildDataBlock` 只投影命宫、身宫主星、年柱、五行局与流年，未投影其余十宫。因此 prompt 声称“十二宫已就绪”而模型实际看不到完整十二宫；需补齐现有数据投影，不能再造一套排盘数据通道。

## 实施批次

### Batch 1：八字展示合同

**修改位置**

- `backend/internal/specialists/bazi/presentation/renderer_templates.go`
- `backend/internal/specialists/bazi/presentation/renderer_sections.go`
- `backend/internal/specialists/bazi/presentation/renderer_facts.go`
- 相应 `renderer_test.go`；必要时同步 `eval/datasets/bazi-answer-quality-v2.json`。

**做法**

1. 新增仅供 presentation 使用的格局结论拼装：消费已验收 `MainAxis`、`PatternOutcome`、`TierStatus` 与具体限制，不重新裁断命理，不输出内部状态。
2. 总览和“格局评价”共用同一用户结论，避免一个说“成立”、另一个只说“已定”。
3. 在 `lifetimePeriodEffectLabel` 的单点映射中前置极性：
   - `complete_pattern`、`support_use` -> `偏吉`
   - `carry_balance`、`transform_pattern` -> `平`
   - `damage_use`、`break_pattern` -> `偏凶`
   - `undetermined` -> `待观察`
4. 保留“仅指对原局结构的影响”的简短边界说明一次，不逐条重复系统术语。
5. 引文展示前比对该引文的来源主题/引用事实与当前段主张；不满足时不展示。

**不做**

- 不改变 `TierAssessment`、九级量表、Graph 资格、repair 或静态/动态事实。
- 不让 renderer 推断新的吉凶或改写原始 `PeriodEffect`。

**验证**

- renderer 单测：用户文本不含三种内部状态词；每一条 accepted 大运均有且仅有一个极性；总览在最后；不相关引文被省略。
- `go test ./backend/internal/specialists/bazi/... -count=1`
- 真实 SSE 回放：检查最终 text，而不是只检查 trace audit 为 clean。

### Batch 2：紫微缓存盘投递与事实使用

**修改位置**

- `backend/internal/runtime/executor_prefill.go`
- `backend/internal/specialists/ziwei/application/prompt_projection.go`
- `backend/internal/prompts/ziwei.md`
- `backend/internal/specialists/ziwei/application/prompt_projection_test.go`
- `backend/internal/runtime/*_test.go` 与必要的 `web/src` 组件/SSE 测试。

**做法**

1. 在 `prefillZiWeiForPlan` 的“命中兼容缓存盘”分支中，将该盘以现有 `emitChartFromToolResult(..., "ziwei_calc", ...)` 发出；这只投递既有资产，不调用工具，不改变缓存。
2. 以“一轮请求最多一次 `ziwei-chart`”为合同。该 prefill 对同一资产只运行一次；不增加前端缓存、全局去重器或第二个事件协议。
3. 扩展现有 `BuildDataBlock`：以紧凑的一行一宫格式投影十二宫名称、宫位干支、主星与大限；不把完整 JSON 或所有杂曜重复塞进模型上下文。前端 component 继续使用完整原始 payload。
4. 在 `ziwei.md` 明确：十二宫核心资料已完整列出；宫位型问题先引用对应宫位，再说明星曜组合；只有该宫核心字段为空时才能说明边界，禁止把已注入的盘说成缺失。
5. 先用 prompt 投影合同和真实回放验证模型是否遵守；不预先增加基于措辞黑名单的 final guard。若仍稳定发生事实否认，再单独设计结构化紫微回答合同。

**不做**

- 不重新计算或复制 `ZiWeiChart`、不修改十二宫/星曜算法。
- 不把缓存盘重复保存到 session，不改变 follow-up 路由或 `reuse_cached_result` 策略。
- 不因补星盘而把紫微 support 的自由文本升级为主结论。

**验证**

- runtime 单测：已有兼容紫微盘时不调用 `ziwei_calc`，但发出一个 `ziwei-chart` component，payload 含 12 宫。
- runtime 单测：新盘同样只发出一个 component。
- application 单测：紫微 prompt 数据块覆盖十二宫名称、宫位干支、主星和大限，且不退回整盘 JSON。
- 前端测试：接收 `component/ziwei-chart` 后 `AssistantTurn` 生成 `ZiweiChartCard` result block。
- 真实 SSE 两轮回放：首轮紫微排盘、同 session 紫微追问；两轮都收到一个 `ziwei-chart`，第二轮没有 `ziwei_calc`，且事业/婚姻等宫位问题实际引用对应宫位。

### Batch 3：评测收口

1. 给 `bazi-answer-quality-v2` 增加用户可读输出断言，而非继续扩充禁词：格局结论有具体主轴与条件、全程每运有极性、系统词不出现。
2. 新增最小紫微缓存复用 eval：`primary_domain=ziwei`、`reuse_cached_result=true`、component 类型为 `ziwei-chart`、最终文本不否认已有十二宫。
3. 更新 `docs/acceptance-criteria.md`：补充紫微命盘卡在新盘与复用盘都要展示的验收标准。

## 验证顺序与回退

1. 先跑改动包的 Go 测试和前端单测，再跑 `go test ./backend/... -count=1`、`go build ./backend/cmd/server/`、`cd web && npm run build`、`git diff --check`。
2. 重启后端，按两个真实 SSE 用例检查事件顺序、唯一 text/done 与 component payload。
3. 任一批失败时，只回退该批的 presentation/prefill/prompt/test 改动；不回退当前工作区已有的错误处理、调候或 Graph 修改。
