# 命理大师

AI 命理咨询聊天应用。自然语言输入出生信息，系统排盘后结合古典籍知识库给出针对性解答。

支持**八字命理、奇门遁甲、紫微斗数**三大领域。v1 原生 Go 实现，无 Agent 框架依赖。

---

## 架构总览

```
                         ┌─────────────────────────┐
                         │    Vue 3 前端 (:5173)     │
                         │                          │
                         │  WelcomePanel            │
                         │  ChatPanel               │
                         │    ├─ ChatBubble (用户)   │
                         │    └─ AssistantTurn (助手) │
                         │         ├─ ResultBlock    │
                         │         │    ├─ BaziChartCard
                         │         │    └─ QimenChart
                         │         ├─ TracePanel     │
                         │         └─ KnowledgeSourceCard
                         └──────────┬──────────────┘
                                    │ SSE (7 种事件)
                                    ▼
                         ┌─────────────────────────┐
                         │    Gin HTTP (:8080)       │
                         │    POST /api/chat         │
                         │    GET  /api/health       │
                         └──────────┬──────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Orchestrator.Run()                        │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ① Supervisor 路由 (LLM)                                  │   │
│  │                                                         │   │
│  │  用户消息 ──→ LLM (Flash Model) ──→ SupervisorDecision    │   │
│  │                                           │              │   │
│  │  三层防御:  structuredDecide               │              │   │
│  │            → textDecide (失败回退)          │              │   │
│  │            → safeFallback (完全不可用)       │              │   │
│  │                                           ▼              │   │
│  │  {                                                     │   │
│  │    conversation_intent: "chitchat" | "consult"          │   │
│  │    primary_domain:      "bazi" | "qimen" | "ziwei"     │   │
│  │    task_intent:         "collect_profile"               │   │
│  │                       | "amend_profile"                 │   │
│  │                       | "interpret_chart"               │   │
│  │                       | "fortune_followup"              │   │
│  │                       | "timing_followup"               │   │
│  │                       | "cross_domain_consult"          │   │
│  │                       | "direct_bazi"                   │   │
│  │                       | "chitchat"                      │   │
│  │    slots: {                                             │   │
│  │      profile, question_text, time_scope, target_subject │   │
│  │    }                                                    │   │
│  │    confidence: 0.0-1.0                                  │   │
│  │    policy_hints: { needs_knowledge, can_reuse_* }       │   │
│  │  }                                                     │   │
│  └───────────────────────┬─────────────────────────────────┘   │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ② Policy Gate (Go 确定性修正)                             │   │
│  │                                                         │   │
│  │  规则 A: 已有资料 + collect_profile → amend_profile       │   │
│  │  规则 B: 已有命盘 + collect_profile → fortune_followup   │   │
│  │                                                         │   │
│  │  行业实践: LLM 负责内容分类，Go 代码负责状态判定           │   │
│  └───────────────────────┬─────────────────────────────────┘   │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ③ 领域调度 (executeRoute)                                │   │
│  │                                                         │   │
│  │  task_intent ──→ 路由分支:                               │   │
│  │                                                         │   │
│  │  direct_bazi ──────────→ executeDirectBaziRoute          │   │
│  │   四柱直输，跳过资料采集                                   │   │
│  │                                                         │   │
│  │  collect_profile ──────→ executeCollectProfileRoute      │   │
│  │   追问缺失字段 (year/month/day/hour/gender/birthplace)     │   │
│  │                                                         │   │
│  │  amend_profile ────────→ executeAmendProfileRoute        │   │
│  │   合并新资料 → 排盘 → 完整解读                             │   │
│  │                                                         │   │
│  │  interpret_chart ──────→ executeFullReadingRoute         │   │
│  │  fortune_followup ─────→ 首次完整解读 / 运势追问          │   │
│  │                                                         │   │
│  │  timing_followup ──────→ qimen primary lane              │   │
│  │  cross_domain_consult ─→ 奇门排盘 → 结合八字分析          │   │
│  │                                                         │   │
│  │  ziwei primary ────────→ executeZiweiPrimaryRoute        │   │
│  │   紫微排盘 → 紫微解读                                    │   │
│  └───────────────────────┬─────────────────────────────────┘   │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ④ Tools 工具链                                           │   │
│  │                                                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │ bazi_calc    │  │ yongshen     │  │ dayun_       │   │   │
│  │  │ 八字排盘     │  │ 用神喜忌分析 │  │ analyzer     │   │   │
│  │  │ (lunar-go)  │  │              │  │ 大运分析     │   │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │ shensha      │  │ qimen_dunjia │  │ ziwei_calc   │   │   │
│  │  │ 神煞         │  │ 时家奇门     │  │ 紫微斗数     │   │   │
│  │  │              │  │ (拆补法)     │  │ (框架就绪)   │   │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │ knowledge_search (MCP :3100)                     │   │   │
│  │  │ 古籍原文检索 → 八字格局/用神/调候/冲合 定向查询   │   │   │
│  │  └──────────────────────────────────────────────────┘   │   │
│  └───────────────────────┬─────────────────────────────────┘   │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ⑤ Prompt 动态注入                                        │   │
│  │                                                         │   │
│  │  interpret.md (统一基座)                                 │   │
│  │    ├─ 身份定义 + 分析优先级 + 依据边界                     │   │
│  │    ├─ 用神运用 + 回答策略 + 输出结构                       │   │
│  │    ├─ 知识库运用规则 + 引用格式 + 禁止事项                  │   │
│  │    └─ <!-- TASK_BLOCK -->  ← 运行时替换                   │   │
│  │                                                         │   │
│  │  snippets/ (动态注入的任务片段，3-10 行)                    │   │
│  │    ├─ fortune.md     ← fortune_followup / interpret_chart│   │
│  │    ├─ year_event.md  ← timing_followup + 具体年份        │   │
│  │    ├─ marriage.md    ← target_subject = 婚姻/感情         │   │
│  │    ├─ career.md      ← target_subject = 事业/财运         │   │
│  │    ├─ health.md      ← target_subject = 健康             │   │
│  │    ├─ personality.md ← target_subject = 性格             │   │
│  │    └─ default.md     ← 通用回退                          │   │
│  │                                                         │   │
│  │  独立模式 (整份 prompt 切换):                              │   │
│  │    ├─ qimen.md       ← 奇门主力，无八字命盘时             │   │
│  │    └─ direct.md      ← PROMPT_MODE=direct benchmark      │   │
│  └───────────────────────┬─────────────────────────────────┘   │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ⑥ LLM 流式解读 → SSE 推送                                │   │
│  │                                                         │   │
│  │  Base Prompt + 运行时上下文 + 知识库 passages             │   │
│  │    ├─ 出生资料 JSON                                      │   │
│  │    ├─ 命盘结果 JSON (八字/奇门/紫微)                      │   │
│  │    ├─ 历史摘要 (RunningSummary)                          │   │
│  │    ├─ 最近对话 (RecentTurns, 最多 4 轮)                  │   │
│  │    └─ 当前问题                                           │   │
│  │                                                         │   │
│  │  LLM 流式响应 ──→ SSE text 事件 ──→ 前端逐字渲染          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

       ┌──────────────────────────────────────────────┐
       │              持久化 & 可观测性                  │
       │                                              │
       │  data/sessions/{id}.json  会话状态            │
       │    ├─ Profile, BaziResult, QimenResult        │
       │    ├─ Routing (路由快照)                       │
       │    ├─ DomainStates (领域状态)                  │
       │    ├─ RecentTurns (最近对话)                   │
       │    └─ RunningSummary (滚动摘要)                │
       │                                              │
       │  logs/traces/   TurnTrace (DEBUG_TRACE=1)     │
       │  logs/debug/    SSE 事件记录 (DEBUG_HTTP=1)    │
       └──────────────────────────────────────────────┘
```

### 关键设计决策

**LLM 分类 + Go 确定性修正 (Pattern 1: Routing)**

LLM 回答「这条消息包含什么内容」（`collect_profile` / `fortune_followup`），Go 代码回答「当前状态下应该走哪个分支」（`amend_profile` / 复用命盘）。LLM 不擅长跨轮状态比对——确定性代码一行就够了：

```
规则 A: 会话已有资料 + LLM 判 collect_profile → 纠正为 amend_profile
        场景: T1 存了 year=1990，T2 用户说「5月20日早上8点，男，北京」

规则 B: 已有命盘 + 用户纯追问 (无新出生时间) → 纠正为 fortune_followup
        场景: T1 排了盘，T2 用户说「今年运势怎么样」「我适合做什么工作」
```

**Supervisor 三层降级防御**

| 层 | 机制 | 触发条件 | 行为 |
|---|------|---------|------|
| L1 | structuredDecide | 正常 | 强制 tool_choice，数学保证 JSON schema 匹配 |
| L2 | textDecide | L1 校验失败 / API 错误 | 纯文本 + 错误反馈重试 (最多 3 次) |
| L3 | safeFallback | L2 全部失败 | 确定性硬编码，零网络调用 |

**Prompt 动态注入 (基座 + 片段)**

所有常态回答共享 `interpret.md` 基座（身份、约束、风格、输出格式），运行时根据 `task_intent` + `target_subject` + `time_scope` 从 `snippets/` 选取 3-10 行领域规则注入到 `<!-- TASK_BLOCK -->` 占位符。不切换整份 prompt，保证行为一致性。

---

## SSE 事件协议

前端通过 `POST /api/chat` 建立长连接，服务端以 SSE 流式推送 7 种事件：

| 事件 | 方向 | 格式 | 说明 |
|------|------|------|------|
| `specialist_bazi` | S→C | `"collect_profile"` 等 | 当前激活的领域专家 + 调度动作 |
| `specialist_qimen` | S→C | `"timing_followup"` 等 | 奇门专家并行调度 |
| `thinking` | S→C | `{"agent":"orchestrator","text":"..."}` | 编排器状态提示 |
| `tool_call` | S→C | `{"tool":"bazi_calc","params":{...}}` | 工具调用及参数 |
| `component` | S→C | `{"type":"bazi-chart","payload":{...}}` | 结构化组件数据 |
| `text` | S→C | `{"content":"..."}` | LLM 流式文本片段 |
| `error` | S→C | `{"message":"..."}` | 错误信息 |
| `done` | S→C | `{}` | 本轮结束 |

`component` 子类型：`bazi-chart` | `qimen-chart` | `trace-panel` | `knowledge-sources`

---

## 前端组件树

```
App.vue
└─ ChatPanel.vue                    ← SSE 接收 + 消息管理
     ├─ WelcomePanel.vue            ← 空状态，快捷提问入口
     ├─ ChatBubble.vue              ← 用户消息 (气泡样式)
     │    └─ TextSegment.vue
     └─ AssistantTurn.vue           ← 助手回复 (结构化分区)
          ├─ ResultBlock.vue        ← 命盘 / 奇门结果卡片
          │    ├─ BaziChartCard.vue ← 八字四柱 + 大运 + 用神
          │    └─ QimenChart.vue    ← 奇门九宫格
          ├─ KnowledgeSourceCard.vue← 知识引证来源
          ├─ TracePanel.vue         ← 链路耗时面板
          ├─ ThinkingSegment.vue    ← 思考过程 (可折叠)
          └─ ToolCallSegment.vue    ← 工具调用记录
```

---

## 技术栈

| 层 | 技术 |
|---|------|
| 前端 | Vue 3 + Naive UI + TypeScript + Vite + markdown-it |
| HTTP | Gin |
| Agent 路由 | 原生 Go（Supervisor LLM 分类 + Policy Gate 确定性修正） |
| 八字 | [lunar-go](https://github.com/6tail/lunar-go) |
| 奇门 | 时家奇门（拆补法），原生 Go 实现 |
| 紫微 | 原生 Go 实现（框架就绪） |
| 知识检索 | 自建知识库 MCP (:3100)，古籍原文，权威分级 ⭐1-5 |
| LLM | DeepSeek / Claude API（兼容 Anthropic 协议） |
| 流式 | SSE（7 种事件类型） |
| 持久化 | JSON 文件存储 + 内存锁并发控制 |
| 追踪 | TurnTrace（每轮耗时、工具调用、状态标记） |

---

## 快速开始

```bash
# 前置条件: Node.js ≥ 18, Go ≥ 1.21, pnpm

# 1. 配置
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY

# 2. 启动知识库 (:3100)
make knowledge-start

# 3. 启动后端 (:8080) + 前端 (:5173)
make dev

# 浏览器打开 http://localhost:5173
```

环境变量：

| 变量 | 必填 | 默认值 | 说明 |
|------|:--:|------|------|
| `LLM_API_KEY` | ✓ | — | LLM API 密钥 |
| `LLM_BASE_URL` | | `api.deepseek.com/anthropic` | API 地址 |
| `LLM_MODEL` | | `deepseek-v4-pro` | 主模型 |
| `LLM_FLASH_MODEL` | | 同主模型 | 快速模型（Supervisor 路由用） |
| `KNOWLEDGE_MCP_URL` | | `http://localhost:3100` | 知识库地址 |
| `DEBUG_TRACE` | | `0` | `1` 启用 TurnTrace 文件记录 |
| `DEBUG_HTTP` | | `0` | `1` 启用 SSE 事件调试记录 |
| `PROMPT_MODE` | | `soft` | `direct` 启用 benchmark 直答模式 |

---

## 项目结构

```
suanming-agent/
├── cmd/server/                # 程序入口
├── internal/
│   ├── config/                # 环境变量读取
│   ├── container/             # DI 容器，组装所有组件
│   ├── handler/               # HTTP handler + SSE 适配
│   ├── llm/                   # LLM 客户端 (Anthropic 协议)
│   ├── mcp/                   # 知识库 MCP 客户端
│   ├── orchestrator/          # 核心编排：Run() + 路由 + prompt 构建
│   ├── policy/                # 策略网关 + 确定性状态修正
│   ├── schemas/               # SupervisorDecision 等共享类型
│   ├── specialists/           # bazi / qimen / ziwei 领域专家
│   ├── sse/                   # SSE 流式推送封装
│   ├── state/                 # 会话状态 + 持久化 + 并发锁
│   ├── supervisor/            # Supervisor LLM 路由 (三层防御)
│   ├── tools/                 # 工具注册表 + 各工具实现
│   └── tracing/               # TurnTrace 链路追踪
├── knowledge/                 # 知识库子项目 (端口 :3100)
│   ├── wiki/                  # 命理古籍原文 (gitignored)
│   └── src/                   # 知识库服务源码 (Next.js + MCP)
├── prompts/
│   ├── interpret.md           # ★ 统一基座 prompt (所有常态回答)
│   ├── qimen.md               # 奇门独立模式 prompt
│   ├── direct.md              # Benchmark 直答模式 prompt
│   └── snippets/              # 动态注入的任务片段 (7 个)
├── web/                       # Vue 3 前端
│   └── src/
│       ├── components/        # UI 组件
│       ├── composables/       # useSSE (SSE 流式接收)
│       ├── utils/             # buildAssistantTurnViewModel
│       └── types/             # ChatMessage / Segment 类型
└── data/sessions/             # 会话持久化文件 (gitignored)
```

---

## 常用命令

```bash
go build ./cmd/server/                  # 编译
go test ./... -v                        # 全部测试
go test ./internal/orchestrator/ -v     # 编排器测试
cd web && npx vue-tsc --noEmit          # 前端类型检查
cd web && npx vitest run                # 前端测试
```

---

## 项目状态

**v1.0.0 首个正式版**

- 八字 / 奇门 / 紫微 三大领域
- Supervisor 三层路由 + Policy Gate 确定性修正
- interpret.md 基座 + 7 个 snippet 动态注入
- AssistantTurn 结构化前端渲染 (命盘卡片 / 追踪面板 / 知识引证)
- TurnTrace 链路追踪 + SSE 调试记录
- 会话状态持久化 + 滚动摘要上下文管理
