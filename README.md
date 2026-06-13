# 命理大师

AI 八字命理咨询聊天应用。自然语言输入出生信息，系统排盘后结合知识库给出针对性解答。支持八字命理与奇门遁甲双领域咨询。

**v1 主线：原生 Go 实现。** 不依赖 LangGraph、Eino 等 Agent 框架。路由决策采用行业标准 Pattern 1 (Routing)：LLM 分类 + Go 确定性状态机（Code for control, Model for understanding）。

---

## 架构

```
Vue 3 → SSE → Gin (:8080) → Supervisor → Policy Gate → State Correction → Specialist → Tools
                                      ↑
                              确定性 Go 代码修正 LLM 路由
                              (amend_profile / fortune_followup)
```

Go 单一后端。Supervisor 路由、Policy Gate 校验、确定性状态修正、Specialist 分发、SSE 推送全部手写 Go 代码，无 Agent 框架依赖。详见 [docs/architecture.md](docs/architecture.md)。

---

## 技术栈

| 层 | 技术 |
|---|------|
| 前端 | Vue 3 + Naive UI + Vite |
| HTTP | Gin |
| Agent 路由 | 原生 Go（LLM 分类 + 确定性状态机） |
| 八字 | [lunar-go](https://github.com/6tail/lunar-go) |
| 奇门 | [qimen-go](https://github.com/deminzhang/qimen-go) |
| 知识检索 | 项目知识库 |
| LLM | Claude API / DeepSeek |
| 流式 | SSE |

---

## 快速开始

```bash
# 一键启动
make dev

# 或分步
make knowledge-start   # 知识库 (:3100)
make dev               # 后端 (:8080) + 前端 (:5173)
```

浏览器打开 `http://localhost:5173`

---

## 文档

| 文档 | 说明 |
|------|------|
| [docs/product.md](docs/product.md) | 产品定义 |
| [docs/architecture.md](docs/architecture.md) | 架构设计 |
| [docs/architecture/supervisor/](docs/architecture/supervisor/) | Supervisor 架构分页设计 |
| [docs/learning/](docs/learning/) | Agent 学习路线 |
| [PROGRESS.md](PROGRESS.md) | 项目进度（上下文恢复文件） |

---

## 项目状态

**阶段：** v1.5 Supervisor Phase 1.5 收口 + qimen 独立回答能力
