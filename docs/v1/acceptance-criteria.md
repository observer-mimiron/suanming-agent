# 验收标准

**原则：** 先验收 v1 主线，再考虑 v2 LangGraph 对照版。

---

## 1. v1 主线验收

### M0: 项目骨架

| # | 验收用例 | 预期结果 | 验证方式 |
|---|---------|---------|---------|
| M0-1 | `go build ./cmd/server/` | 编译成功 | 命令行 |
| M0-2 | `go run ./cmd/server/` 后 `curl localhost:8080/api/health` | `{"status":"ok"}` | curl |
| M0-3 | `cd web && npm run build` | 构建成功，无 TS 错误 | 命令行 |
| M0-4 | `cd web && npm run dev` 后浏览器打开 | 看到“命理大师”页面 | 浏览器 |

### M1: 八字引擎

| # | 验收用例 | 预期结果 | 验证方式 |
|---|---------|---------|---------|
| M1-1 | `bazi_calc(year=1990, month=5, day=20, hour=8, gender="男")` | 返回 4 柱八字，日主非空 | 单元测试 |
| M1-2 | `bazi_calc(year=2000, month=1, day=1, hour=0, gender="女")` | 返回 4 柱八字 | 单元测试 |
| M1-3 | `bazi_calc(year=1899, ...)` | 返回错误 | 单元测试 |
| M1-4 | 闰年日期 | 处理正确 | 单元测试 |
| M1-5 | 五行统计 | 天干地支总数正确 | 单元测试 |
| M1-6 | 大运排盘 | 返回结构完整 | 单元测试 |

### M3: Go Orchestrator

| # | 验收用例 | 预期结果 | 验证方式 |
|---|---------|---------|---------|
| M3-1 | `POST /api/chat {message:"帮我算八字"}` | SSE 流返回 `thinking + text + done` | `curl -N` |
| M3-2 | `POST /api/chat {message:"1990年5月20日早上8点，男"}` | 返回 `tool_call(bazi_calc)` + `component(bazi-chart)` | `curl -N` |
| M3-3 | 同一 `session_id` 多轮请求 | 出生信息跨轮保持 | `curl -N` |
| M3-4 | 修改出生信息后再次请求 | 旧命盘失效并重新排盘 | `curl -N` |
| M3-5 | `GET /api/tools` | 返回已注册工具列表 | curl |
| M3-6 | 缺少 `message` 字段 | HTTP 400 | curl |

### M4: 项目知识库 MCP + LLM

| # | 验收用例 | 预期结果 | 验证方式 |
|---|---------|---------|---------|
| M4-1 | `knowledge_search(query=...)` 正常 | 返回 passages 数组，每项含 `content/source` | 单元测试 |
| M4-2 | 知识库 MCP 不可用 | 返回空 passages，不阻塞回答，`fallback=true` | 单元测试 |
| M4-3 | 完整排盘流程含知识检索 | SSE 流含 `component(knowledge-sources)` | `curl -N` |
| M4-4 | LLM 问答文本逐字推送 | 多个 `text` chunk 连续返回 | `curl -N` |
| M4-5 | `LLM_API_KEY` 错误 | 启动检查给出明确错误 | 单元测试 |

### M5: Vue 3 前端

| # | 验收用例 | 预期结果 | 验证方式 |
|---|---------|---------|---------|
| M5-1 | 页面加载 | 聊天界面正常显示 | 浏览器 |
| M5-2 | 发送普通文本 | 用户消息在右，助手消息在左 | 浏览器 |
| M5-3 | 输入出生信息 | 出现八字命盘卡片 | 浏览器 |
| M5-4 | thinking 可见 | thinking 事件可折叠展示 | 浏览器 |
| M5-5 | tool_call 可见 | 显示工具名 | 浏览器 |
| M5-6 | 知识引用可见 | 显示知识引用卡片 | 浏览器 |
| M5-7 | 流式文本可见 | 文本逐步出现 | 浏览器 |
| M5-8 | 空输入不发送 | 按钮 disabled | 浏览器 |
| M5-9 | 刷新后 `sessionId` 保持 | `localStorage` 中值不变 | DevTools |

### M6: 集成联调

| # | 验收用例 | 预期结果 | 验证方式 |
|---|---------|---------|---------|
| M6-1 | 全流程：`帮我算八字 → 1990年5月20日早上8点 → 男` | 显示命盘卡片 + 知识引用 + 回答 | 浏览器 |
| M6-2 | 排盘后追问：`今年运势怎么样` | 复用命盘，不重新排盘，并给出针对性回答 | 浏览器 |
| M6-3 | 知识库 MCP 不可用 | 跳过知识检索，仍可完成回答 | 浏览器 |
| M6-4 | `start.sh` 一键启动 | Go + Web 启动成功 | 命令行 |
| M6-5 | 解读风格 | 口吻符合产品定位 | 人工判断 |
| M6-6 | 响应速度 | 首个 `thinking` 在 3 秒内到达 | 计时 |

---

## 2. 总体原则

- v1 不以 LangGraph 是否存在为完成条件
- v2 不阻塞 v1 上线和演示
- 面试时优先讲清楚 v1 如何完成“排盘 + 咨询问答”，再讲 v2 的对照价值
- v2 详细验收见 [docs/v2/acceptance-criteria.md](/Users/wikiglobal/workSapce/suanming-agent/docs/v2/acceptance-criteria.md)
