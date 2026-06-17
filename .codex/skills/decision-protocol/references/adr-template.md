# ADR Template (MADR Simplified)

基于 MADR 格式 (adr/madr, ⭐2,270)，适配单人开发场景。仅在需要持久化记录决策时使用。

## Template

```md
# ADR-NNN: Short Imperative Title

**Date:** YYYY-MM-DD
**Status:** proposed | accepted | superseded
**Decision Maker:** <name or "model recommendation, user approved">

## Context

为什么现在需要这个决定。引用具体约束：现有代码路径、依赖、性能数据、架构决策。

## Decision

我们选择做什么。一句话或简短列表。避免 hedging 措辞 ("should consider", "perhaps")。

## Alternatives Considered

至少 2 个替代方案，每个附带拒绝原因。

- **Alternative A.** 拒绝原因。
- **Alternative B.** 拒绝原因。

## Consequences

**正面：**
- 变容易了什么。

**负面：**
- 变困难了什么、新增了什么约束。

## Revisit If

什么条件变化时值得重新审视这个决定。
```

## Example

```md
# ADR-001: Use lunar-go for Bazi Calculation

**Date:** 2026-06-17
**Status:** accepted
**Decision Maker:** model recommendation, user approved

## Context

八字排盘需要计算农历、节气、天干地支。自研需要 2-4 周且容易出错。项目是 Go 后端单进程，已有 HTTP+SSE 通信架构。

## Decision

使用 lunar-go (github.com/nosyliu/lunar-go) 作为八字计算引擎，通过 Go tool 接口封装为 bazi_calc tool。

## Alternatives Considered

- **自研农历计算。** 开发周期过长，节气计算边界条件复杂。
- **调用第三方 API。** 增加网络延迟和外部依赖，离线不可用。
- **不做八字功能。** 与项目目标冲突。

## Consequences

**正面：**
- 开发周期从周级降到天级。
- 节气计算由维护 10+ 年的库处理，准确性有保证。

**负面：**
- 依赖外部库版本，升级需回归测试。
- 定制排盘规则（如神煞、纳音）需在 lunar-go 之上封装。

## Revisit If

lunar-go 不再维护且出现阻断性 bug；或需要特殊命理流派规则（如盲派排盘）且 lunar-go 无法扩展。
```
