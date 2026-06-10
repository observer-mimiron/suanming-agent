# 命理大师 — 前端技术方案

**版本：** v1.0 MVP  
**依赖：** 产品文档 `docs/product.md`、后端方案 `docs/tech-backend.md`

---

## 1. 技术选型

| 层 | 选型 | 原因 |
|---|------|------|
| 框架 | Vue 3 + TypeScript + Vite | 用户会 Vue |
| UI 组件库 | Naive UI | 暗色模式开箱即用，组件全，中文友好 |
| 状态管理 | 组件内 `ref/reactive` | 单页面应用不需要 Pinia |
| 通信 | SSE (EventSource API) | 后端推送 4 种事件，前端按类型消费 |

---

## 2. 页面结构

单一页面，两个区域：

```
┌───────────────────────────────────────────┐
│  🔮 命理大师                               │
├───────────────────────────────────────────┤
│                                           │
│  🤖 客官你好，贫道精通八字命理，            │
│     请告知你的出生年月日时和性别。          │
│                                           │
│                       👤 1990年5月20日     │
│                          早上8点，男        │
│                                           │
│  ┌─ 🔮 八字命盘 ───────────────────────┐ │
│  │  庚午   辛巳   甲子   戊辰           │ │
│  │  年柱   月柱   日柱   时柱           │ │
│  │                                     │ │
│  │  🌳 日主：甲木                      │ │
│  │  五行：木2 火2 土2 金1 水1          │ │
│  │  十神：正官 正印 日主 偏财          │ │
│  └─────────────────────────────────────┘ │
│                                           │
│  🤖 你的日主为甲木，属参天之木，           │
│     性格刚直果敢...                        │
│                                           │
├───────────────────────────────────────────┤
│ 💬 输入消息...                        🔧  │
└───────────────────────────────────────────┘
```

---

## 3. 组件树

```
App.vue
└── ChatPanel.vue               # 聊天主容器
    ├── NScrollbar              # Naive UI 滚动区 (消息列表)
    │   └── ChatBubble.vue      # 单条消息 (v-for messages)
    │       └── [Segment].vue   # 按 segment.type 分发
    │           ├── TextSegment.vue       (type: text)
    │           ├── ThinkingSegment.vue   (type: thinking)
    │           ├── ToolCallSegment.vue   (type: tool_call)
    │           └── BaziChartCard.vue     (type: component + bazi-chart)
    └── NInputGroup             # 底部输入区
        ├── NInput              # 文本输入框
        └── NButton             # 发送按钮
```

---

## 4. 数据模型

```typescript
// types/chat.ts

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  segments: Segment[]
}

type Segment =
  | { type: 'text'; content: string }
  | { type: 'thinking'; text: string; agent: string }
  | { type: 'tool_call'; tool: string; params: Record<string, any> }
  | { type: 'component'; componentType: string; payload: any }
```

---

## 5. SSE 消费逻辑

```typescript
// composables/useSSE.ts

async function sendMessage(content: string) {
  // 1. 添加用户消息到列表
  messages.value.push({ role: 'user', segments: [{ type: 'text', content }] })

  // 2. 创建助手消息占位
  const assistantMsg: ChatMessage = { role: 'assistant', segments: [] }
  messages.value.push(assistantMsg)

  // 3. POST /api/chat → SSE
  const resp = await fetch('/api/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message: content, sessionId })
  })

  // 4. 消费 SSE 流
  const reader = resp.body!.getReader()
  // 解析 event: xxx / data: xxx → push segment 到 assistantMsg.segments
  // text 事件连续 chunk 合并到同一个 TextSegment
}
```

**事件 → Segment 映射：**

| SSE 事件 | 创建/追加 Segment |
|----------|------------------|
| `thinking` | 追加新的 `ThinkingSegment` |
| `tool_call` | 追加新的 `ToolCallSegment` |
| `component` | 追加新的 `{type:'component', ...}` → ComponentRenderer 分发 |
| `text` (chunk) | 追加到最后一个 TextSegment 的 content，或创建新的 |
| `done` | 结束本轮，loading = false |

---

## 6. 组件渲染器 (ComponentRenderer)

```typescript
// 组件注册表 — 按 componentType 映射到 Vue 组件
const componentRegistry: Record<string, Component> = {
  'bazi-chart': BaziChartCard,
  // 后续扩展:
  // 'wuxing-radar': WuxingRadarChart,
  // 'dayun-timeline': DayunTimeline,
}
```

`ComponentRenderer.vue`：
```vue
<component :is="componentRegistry[seg.componentType]" :data="seg.payload" />
```

---

## 7. 组件清单

| 组件 | 职责 | MVP 复杂度 |
|------|------|-----------|
| `ChatPanel.vue` | 消息列表 + 输入框 + SSE 调用入口 | 中 |
| `ChatBubble.vue` | 按 role 渲染气泡，遍历 segments 分发子组件 | 低 |
| `TextSegment.vue` | 纯文本段落，后续可加打字机效果 | 低 |
| `ThinkingSegment.vue` | Naive UI NCollapse 折叠面板，显示 Agent 思考步骤 | 低 |
| `ToolCallSegment.vue` | Naive UI NTag 标签，显示工具名+参数摘要 | 低 |
| `BaziChartCard.vue` | Naive UI NCard，四柱表格 + 五行色块条 + 十神标签 | 中 |
| `ComponentRenderer.vue` | 查注册表动态渲染业务组件 | 低 |

---

## 8. 关键决策

| 决策 | 选择 | 原因 |
|------|------|------|
| 无路由 | 单页应用，不引入 vue-router | 只有一个聊天界面 |
| 无状态管理库 | 组件内状态，用 provide/inject 传 SSE 实例 | 组件树浅，不需要 Pinia |
| 暗色默认 | Naive UI darkTheme 直接套 | 与 Terminal Mystic 方向一致 |
| 无图表库 | 命盘用纯 HTML/CSS + 色块 | MVP 不引入 ECharts |
