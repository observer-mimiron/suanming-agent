# 命理大师

AI 八字命理咨询聊天应用。自然语言输入出生信息，系统排盘后结合知识库给出针对性解答。

**v1 主线：Go/Eino 单栈。** LangGraph 对照版延后到 v2。

---

## 架构

```
Vue 3 → SSE → Gin (:8080) → Session State → Tools → lunar-go / MCP / Claude
```

Go 单一后端，管理会话状态、工具执行、SSE 流式推送。详见 [docs/architecture.md](docs/architecture.md)。

---

## 技术栈

| 层 | 技术 |
|---|------|
| 前端 | Vue 3 + Naive UI + Vite |
| HTTP | Gin |
| Agent | Eino Tool System |
| 八字 | [lunar-go](https://github.com/6tail/lunar-go) |
| 知识检索 | 项目知识库 MCP |
| LLM | Claude API |
| 流式 | SSE |

---

## 快速开始

```bash
# Go 后端
LLM_API_KEY=sk-xxx go run ./cmd/server/

# 前端
cd web && npm install && npm run dev
```

浏览器打开 `http://localhost:5173`

---

## 文档

| 文档 | 说明 |
|------|------|
| [docs/product.md](docs/product.md) | 产品定义 |
| [docs/architecture.md](docs/architecture.md) | 架构设计 |
| [docs/v1/](docs/v1/) | v1 技术方案、验收标准、实施计划 |
| [docs/v2/](docs/v2/) | v2 LangGraph 对照版 |
| [docs/learning/](docs/learning/) | Agent 学习路线与素材 |

---

## 项目状态

**阶段：** 设计完成，待实施
