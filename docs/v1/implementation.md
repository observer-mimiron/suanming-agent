# 命理大师 — 实施总览

**版本：** v1.2  
**原则：** 先完成 v1 Go/Eino 主线，再做 v2 LangGraph 对照版。

---

## 1. 实施策略

本项目采用两阶段实施：

- **v1 主线**
  - 目标：做出可运行、可演示、可讲清楚的产品
  - 技术主线：Go / Eino / lunar-go / 项目知识库 MCP / Vue 3
- **v2 对照版**
  - 目标：验证 LangGraph 在复杂状态编排中的收益
  - 技术主线：在同一产品域里引入 LangGraph planner / routing

---

## 2. v1 模块拆分

```text
M0: 项目脚手架
│
├── M1: 八字引擎
│
├── M3: Go Orchestrator
│   ├── 会话状态
│   ├── 对话状态机
│   └── SSE 输出
│
├── M4: 项目知识库 MCP + LLM
│   ├── knowledge_search
│   └── llm_generate
│
├── M5: Vue 3 前端
│
└── M6: 集成联调
```

### v1 模块说明

- **M0**：项目骨架、目录结构、启动方式
- **M1**：`lunar-go` 集成和八字工具
- **M3**：Go 控制面，包括状态机、工具编排、SSE
- **M4**：接入项目知识库 MCP 和最终解读
- **M5**：前端聊天界面和结构化渲染
- **M6**：端到端联调

---

## 3. v2 对照版模块

```text
docs/v2/implementation/m2-langgraph.md
```

### M2 的定位

- 不属于 v1 主链路
- 作为对照实现，在 v1 完成后再做
- 重点关注：
  - 复杂信息收集
  - 条件边
  - reasoning flow
  - planner / routing 能否明显优于 Go 手写状态机

---

## 4. 依赖关系

```text
M0 → M1 → M3 → M4 → M5 → M6
                 │
                 └── M2 (v2 optional)
```

说明：

- `M2` 不阻塞 `M3/M4/M5/M6`
- 只有在 v1 跑通后，才开始评估和实现 M2

---

## 5. 实施顺序建议

```text
Week 1: M0 → M1 → M3
Week 2: M4 → M5
Week 3: M6
Week 4+: M2 (LangGraph 对照版)
```

---

## 6. 当前开发重点

当前应优先完成：

1. Go 会话状态模型
2. 八字排盘工具
3. 追问 / 排盘 / 追问复用的状态机
4. 项目知识库 MCP 检索
5. SSE 结构化输出

不要在 v1 提前引入：

- LangGraph 主链路
- 多 agent 闭环
- tool result callback
- 分布式状态同步

---

## 7. 下一步

下一步先同步以下实施文档：

- `docs/v1/implementation/m3.2-orchestrator.md`
- `docs/v1/implementation/m4.2-llm-client.md`
- `docs/v1/implementation/m5.1-types-sse.md`

然后再决定是否细化 `docs/v2/implementation/m2-langgraph.md` 作为 v2 方案文档。
