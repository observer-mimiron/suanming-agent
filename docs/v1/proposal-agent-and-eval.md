# Agent 架构 + 评测体系方案

> 调研日期：2026-06-11

---

## 一、Self-Reflection Agent 方案

### 调研结论

EMNLP 2024 的预算感知评估发现：**当控制总计算预算时，CoT + Self-Consistency（多次独立采样后投票）一致优于 Reflexion 和多 Agent 辩论。** Reflexion 在某些数据集增加预算反而降低性能——LLM 对自己的错误缺乏准确自我认知，可能把正确的改成错误的。

强模型（如 DeepSeek v4）的反思边际收益通常在 1% 以内，但需要额外 1-3 秒延迟。

### 推荐方案：提示词级元认知（零额外成本）

不在 Go 代码中增加反射调用，而是在 prompt 内嵌入自检指令，让 LLM 在**一次生成内部**完成"生成+自检"：

```
在给出最终解读前，先在心中核对：
1. 日主是否与排盘结果一致
2. 五行分析是否有矛盾陈述
3. 每个运势判断是否有原理支撑
4. 流年分析是否优先使用了冲合刑害而非用神标签
如果发现不一致，修正后再输出。
```

**优点**：零额外延迟、零额外成本、强模型有效。已被 Andrew Ng 的 Reflection 设计模式推荐。

### 后续可选：选择性反思

针对健康、婚恋、重大决策等高风险问题，执行一次额外的反思 LLM 调用（非流式反思 → 流式输出修正版）。预计 2-4 秒延迟，仅在 20-30% 的问题触发。

### 不推荐
- Reflexion 完整循环（多轮迭代 + 记忆存储）—— 成本高、复杂度高
- 多 Agent 辩论 —— 过度设计
- 每个问题都反思 —— 边际收益极低

---

## 二、评测体系方案

### 调研结论

市场上有两个叫 "Harness" 的产品，都不适合我们：
- **Harness DevOps 平台**：通用 CI/CD 工具，适合 UI 测试，不适合 Prompt 对比
- **EleutherAI lm-evaluation-harness**：评测模型本身能力，不适合应用层 prompt 变体对比

### 推荐工具：promptfoo

开箱即用的最佳选择：

| 特性 | 说明 |
|------|------|
| **MIT 开源** | 完全免费 |
| **Go Provider** | 原生 Go 集成，直接调用后端 API |
| **YAML 配置** | 160 道题 + 多个 prompt 变体在一个文件 |
| **LLM-as-Judge** | 内置评分，支持 3 级 (正确/部分/错误) |
| **CI 集成** | 嵌入 GitHub Actions，每次变更自动跑 |
| **完全本地** | 数据不出本机 |

### 集成方案

**第一步：加评测端点**（5 分钟）

```go
// cmd/server/main.go
r.POST("/api/evaluate", func(c *gin.Context) {
    var req struct{ Message, SessionID string }
    c.BindJSON(&req)
    var buf bytes.Buffer
    // 非流式：累积所有 SSE 事件到 buf
    orch.Run(sse.NewWriter(c), req.SessionID, req.Message)
})
```

**第二步：写 promptfoo 配置**

```yaml
# promptfooconfig.yaml
providers:
  - id: https
    config:
      url: 'http://localhost:8080/api/chat'
      method: 'POST'
      headers: { 'Content-Type': 'application/json' }
      body: { message: '{{prompt}}', session_id: 'bench-{{testId}}' }
      transformResponse: 'json.output'

prompts:
  - file://prompts/generic.txt
  - file://prompts/career.txt
  - file://prompts/marriage.txt

tests:
  - description: 'Case 1: 男 1974-04-28'
    vars:
      prompt: '男命，1974年4月28日 16:40，usa'
  - description: 'Case 1 Q1: 1996年'
    vars:
      prompt: '此命1996年发生何事？'
```

**第三步：运行评测**

```bash
npx promptfoo eval
npx promptfoo view  # 查看对比报告
```

### 评测流程

```
160 题 YAML → promptfoo → Go 后端 → DeepSeek
                    ↓
              LLM-as-Judge (对比标准答案)
                    ↓
              评分: 正确/部分/错误
                    ↓
              报告: 对比不同 prompt 变体
```

评分使用 3 级准则：
- **正确** (1.0)：核心结论与标准答案一致
- **部分正确** (0.5)：方向对但细节有偏差
- **错误** (0.0)：方向完全错误

---

## 三、优先级建议

| 优先级 | 事项 | 工作量 | 影响 |
|--------|------|--------|------|
| **P0** | 提示词元认知（零成本反思） | 5min 改 prompt | 小幅提升准确性 |
| **P1** | promptfoo 评测体系 | 1h | 可量化衡量改进效果 |
| **P2** | 选择性反思 Agent | 2h | 高风险场景额外保障 |

建议先做 P0（今天就能上），再做 P1（有了评测才知道改 prompt 到底有没有用）。
