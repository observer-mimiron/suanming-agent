# 命理大师前端重设计 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标:** 将命理大师从暗黑 Naive UI 默认风格升级为亮底扁平插画卡片风格（v4 Modern Tarot 方向）

**架构:** 阶段式重写 — 先建 Foundation（Token + 依赖 + 字体），再改布局容器（ChatPanel/WelcomePanel），然后建 Sprite 系统（5精灵+参数化星/门），最后重写图表组件（八字/奇门）。每阶段可独立验证。

**技术栈:** Vue 3 + Naive UI + TypeScript + lucide-vue-next + Playfair Display（Google Fonts）

---

## 文件结构

```
web/
├── index.html                          # +Google Fonts link
├── package.json                        # +lucide-vue-next
├── src/
│   ├── style.css                       # 全局 Token 重写
│   ├── App.vue                         # 移除 darkTheme
│   ├── types/chat.ts                   # 不变
│   ├── composables/useSSE.ts           # 不变
│   ├── utils/assistantTurn.ts          # 不变
│   └── components/
│       ├── sprites/
│       │   ├── ElementSprite.vue       # 新增: 五行精灵 (5合1)
│       │   ├── SpriteStar.vue          # 新增: 九星参数化
│       │   └── SpriteDoor.vue          # 新增: 八门参数化
│       ├── ChatPanel.vue               # 重写: 空状态居中 + 输入框
│       ├── WelcomePanel.vue            # 重写: lucide图标 + 3列卡片
│       ├── AssistantTurn.vue           # 重写: 竖线链路
│       ├── ChatBubble.vue              # 适配: 新配色
│       ├── ResultBlock.vue             # 适配: 新配色
│       ├── BaziChartCard.vue           # 完全重写
│       ├── QimenChart.vue              # 完全重写
│       ├── TextSegment.vue             # 不变
│       ├── ThinkingSegment.vue         # 移除 (合并到AssistantTurn)
│       ├── ToolCallSegment.vue         # 移除 (合并到AssistantTurn)
│       ├── TracePanel.vue              # 保留，适配新配色
│       ├── KnowledgeSourceCard.vue     # 保留，适配新配色
│       └── icons.ts                    # 移除 (被lucide替代)
```

ThinkingSegment 和 ToolCallSegment 目前只在 ChatBubble.vue 的 segment 循环中使用。重构后 AssistantTurn 直接内联处理这些 segment 类型，不再需要独立组件。

---

## Phase 1: Foundation

### Task 1: 安装依赖 + 加载字体

**Files:**
- Modify: `web/package.json`
- Modify: `web/index.html`

- [ ] **Step 1: 安装 lucide-vue-next**

```bash
cd web && bun add lucide-vue-next
```

- [ ] **Step 2: 在 index.html 添加 Google Fonts**

```html
<!-- 在 <head> 中添加，在现有 <link> 之前 -->
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,500;0,600;0,700;1,400&family=Inter:wght@300;400;500;600&display=swap" rel="stylesheet">
```

- [ ] **Step 3: 验证**

```bash
cd web && grep -q "lucide-vue-next" package.json && grep -q "Playfair" index.html && echo "OK"
```

Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add web/package.json web/index.html web/bun.lockb
git commit -m "chore: add lucide-vue-next and Playfair Display font"
```

---

### Task 2: 全局 CSS Token 重写

**Files:**
- Modify: `web/src/style.css`

- [ ] **Step 1: 替换 style.css 为新的暖色 Token 系统**

```css
:root {
  --bg:            #f2efe9;
  --bg-secondary:  #faf8f5;
  --bg-hover:      #f5f1ea;
  --border:        #e4ded4;
  --border-light:  #ede8df;
  --text-primary:  #3a3632;
  --text-secondary:#6b6050;
  --text-muted:    #8a7a68;
  --accent:        #c4a978;
  --accent-bg:     rgba(184,149,106,0.1);
  --accent-dim:    #8a7a60;

  --wx-wood:   #7a9e7e;
  --wx-fire:   #c47a6a;
  --wx-earth:  #b8956a;
  --wx-metal:  #c4a96a;
  --wx-water:  #6b8aa8;

  --serif: "Playfair Display", "Cormorant Garamond", serif;
  --sans:  "Inter", "SF Pro Text", system-ui, -apple-system, sans-serif;
  --mono:  "SF Mono", "JetBrains Mono", ui-monospace, monospace;

  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --shadow-card: 0 8px 30px rgba(58,54,50,0.08);
  --shadow-glow: 0 0 0 4px rgba(184,149,106,0.1);

  font: 15px/1.65 var(--sans);
  color: var(--text-primary);
  background: var(--bg);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

body {
  margin: 0;
}

#app {
  width: 100%;
  max-width: 100%;
  min-height: 100svh;
  display: flex;
  flex-direction: column;
}
```

删除原有所有 CSS 规则（`h1/h2/code/.hero/#center/#next-steps` 等），只保留上述内容。

- [ ] **Step 2: 验证构建不引入新错误**

```bash
cd web && npx vite build 2>&1 | tail -5
```

Expected: 构建成功（预存在的 TS 类型错误忽略）

- [ ] **Step 3: Commit**

```bash
git add web/src/style.css
git commit -m "refactor: rewrite global CSS tokens to warm cream palette"
```

---

## Phase 2: Layout & Simple Components

### Task 3: 重写 ChatPanel.vue

**Files:**
- Modify: `web/src/components/ChatPanel.vue`

- [ ] **Step 1: 重写 template — 空状态居中 + 对话状态底部输入**

```vue
<template>
  <div class="chat-shell" :class="{ 'is-empty': messages.length === 0 && !isLoading }">
    <!-- 空状态 -->
    <div v-if="messages.length === 0 && !isLoading" class="empty-center">
      <WelcomePanel @ask="handleSend" />
      <div class="input-row empty-input">
        <n-input
          v-model:value="inputText"
          placeholder="请输入出生年月日时..."
          @keydown.enter="handleSend"
          size="large"
        >
          <template #suffix>
            <button
              class="send-btn"
              :class="{ active: inputText.trim() }"
              :disabled="!inputText.trim()"
              @click="handleSend"
            >
              <ArrowUp :size="18" />
            </button>
          </template>
        </n-input>
      </div>
    </div>

    <!-- 对话状态 -->
    <template v-else>
      <div class="chat-body">
        <n-scrollbar ref="scrollRef" class="messages">
          <template v-for="msg in messages" :key="msg.id">
            <AssistantTurn
              v-if="msg.role === 'assistant'"
              :message="msg"
              :isLoading="isLoading && msg === messages[messages.length - 1]"
            />
            <ChatBubble v-else :message="msg" />
          </template>
        </n-scrollbar>
      </div>
      <div class="input-row chat-input">
        <n-input
          v-model:value="inputText"
          placeholder="继续提问..."
          :disabled="isLoading"
          @keydown.enter="handleSend"
          size="large"
        >
          <template #suffix>
            <button
              class="send-btn"
              :class="{ active: inputText.trim() }"
              :disabled="!inputText.trim() || isLoading"
              @click="handleSend"
            >
              <ArrowUp :size="18" />
            </button>
          </template>
        </n-input>
      </div>
    </template>
  </div>
</template>
```

- [ ] **Step 2: 更新 script — 移除 NSpin，添加 lucide 图标**

```typescript
<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { NScrollbar, NInput } from 'naive-ui'
import { ArrowUp } from 'lucide-vue-next'
import ChatBubble from './ChatBubble.vue'
import AssistantTurn from './AssistantTurn.vue'
import WelcomePanel from './WelcomePanel.vue'
import { useSSE } from '../composables/useSSE'

const { messages, isLoading, sendMessage } = useSSE()
const inputText = ref('')
const scrollRef = ref()

async function handleSend(t?: string) {
  const text = (typeof t === 'string' ? t : inputText.value).trim()
  if (!text || isLoading.value) return
  inputText.value = ''
  await sendMessage(text)
  await nextTick()
  scrollRef.value?.scrollTo({ top: 999999, behavior: 'smooth' })
}
</script>
```

- [ ] **Step 3: 更新 CSS**

```css
<style scoped>
.chat-shell {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}
.chat-shell.is-empty {
  justify-content: center;
}
.empty-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 32px;
  padding: 24px;
}
.empty-input {
  width: 100%;
  max-width: 520px;
}
.empty-input :deep(.n-input) {
  --n-height: 52px;
  --n-border-radius: 14px;
}
.chat-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.messages {
  flex: 1;
  padding: 24px;
}
.input-row {
  padding: 16px 24px;
}
.chat-input {
  border-top: 1px solid var(--border);
}
.send-btn {
  width: 36px; height: 36px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
  flex-shrink: 0;
}
.send-btn.active {
  border-color: var(--accent);
  color: var(--accent-dim);
}
.send-btn:hover:not(:disabled) {
  background: var(--bg-hover);
}
.send-btn:disabled {
  opacity: 0.4;
  cursor: default;
}
</style>
```

- [ ] **Step 4: 验证**

```bash
cd web && npx vite build 2>&1 | tail -5
```

Expected: 构建成功

- [ ] **Step 5: Commit**

```bash
git add web/src/components/ChatPanel.vue
git commit -m "refactor: center empty state, restyle input with lucide icon"
```

---

### Task 4: 重写 WelcomePanel.vue

**Files:**
- Modify: `web/src/components/WelcomePanel.vue`

- [ ] **Step 1: 重写整个组件**

```vue
<template>
  <div class="welcome">
    <div class="brand">
      <h1>命理大师</h1>
      <p class="subtitle">AI 八字命理咨询</p>
    </div>
    <div class="quick-asks">
      <button v-for="q in prompts" :key="q.label" class="ask-card" @click="$emit('ask', q.text)">
        <div class="ask-icon" v-html="q.icon"></div>
        <div class="ask-text">
          <span class="ask-label">{{ q.label }}</span>
          <span class="ask-sub">{{ q.sub }}</span>
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineEmits<{ ask: [text: string] }>()

const prompts = [
  {
    label: '算八字',
    sub: '四柱·十神·大运',
    icon: `<svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="14" cy="10" r="8"/><path d="M8 14 L6 22 L14 18 L22 22 L20 14"/></svg>`,
    text: '帮我算一下八字',
  },
  {
    label: '排大运',
    sub: '流年·起运·岁运',
    icon: `<svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="14,2 26,8 26,20 14,26 2,20 2,8"/><line x1="14" y1="26" x2="14" y2="16"/><polyline points="2,8 14,14 26,8"/></svg>`,
    text: '帮我排大运',
  },
  {
    label: '五行分析',
    sub: '旺衰·喜用·调候',
    icon: `<svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="14" cy="14" r="10"/><path d="M14 4a7 7 0 0 1 0 20"/><path d="M14 4a3 3 0 0 0 0 20"/></svg>`,
    text: '分析一下我的五行',
  },
]
</script>

<style scoped>
.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 32px;
}
.brand { text-align: center; }
.brand h1 {
  font-family: var(--serif);
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px;
}
.subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0;
}
.quick-asks {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-width: 340px;
}
.ask-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  color: var(--text-primary);
  cursor: pointer;
  transition: transform 0.18s, border-color 0.18s, box-shadow 0.18s;
}
.ask-card:hover {
  transform: translateY(-2px);
  border-color: var(--accent);
  box-shadow: var(--shadow-card);
}
.ask-icon {
  color: var(--accent-dim);
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.ask-label {
  font-size: 14px;
  font-weight: 600;
  display: block;
}
.ask-sub {
  font-size: 11px;
  color: var(--text-muted);
  display: block;
  margin-top: 2px;
}
</style>
```

- [ ] **Step 2: 验证**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 3: Commit**

```bash
git add web/src/components/WelcomePanel.vue
git commit -m "refactor: redesign welcome with lucide icons and card layout"
```

---

### Task 5: 重写 AssistantTurn.vue

**Files:**
- Modify: `web/src/components/AssistantTurn.vue`

当前 AssistantTurn 使用折叠面板展示思考/工具调用。改为竖线时间线 + chip 标签。

- [ ] **Step 1: 重写 template**

```vue
<template>
  <div class="assistant-turn">
    <div class="turn-header">
      <span class="turn-role">命理大师</span>
      <span class="turn-meta" v-if="vm.process">{{ fmtMs(vm.process.trace.total_ms) }}</span>
    </div>

    <!-- 思考+工具链路：左侧竖线 -->
    <div v-if="vm.thoughts.length || vm.toolCalls.length" class="turn-chain">
      <div class="chain-line"></div>
      <div class="chain-items">
        <div v-for="(t, i) in vm.thoughts" :key="'th-' + i" class="chain-thought">{{ t }}</div>
        <div v-for="(tc, i) in vm.toolCalls" :key="'tc-' + i" class="chain-tool">
          <Wrench :size="12" class="tool-icon" />
          <span class="tool-name">{{ tc.name }}</span>
          <code v-if="tc.arguments" class="tool-args">{{ tc.arguments }}</code>
        </div>
      </div>
    </div>

    <!-- 结构化结果 -->
    <section v-if="vm.resultBlocks.length" class="turn-zone">
      <ResultBlock v-for="(rb, i) in vm.resultBlocks" :key="'rb-' + i">
        <template #title>{{ rb.type === 'bazi-chart' ? '八字命盘' : '奇门遁甲' }}</template>
        <BaziChartCard v-if="rb.type === 'bazi-chart'" :data="rb.payload" />
        <QimenChart v-else-if="rb.type === 'qimen-chart'" :data="rb.payload" />
      </ResultBlock>
    </section>

    <!-- 答案正文 -->
    <section v-if="vm.answerBlocks.length" class="turn-answer">
      <div class="answer-content markdown-body" v-html="renderedAnswer" />
      <div class="answer-actions">
        <button class="act-btn" @click="copyAnswer">
          <CheckCheck v-if="copied" :size="14" />
          <Copy v-else :size="14" />
        </button>
      </div>
    </section>

    <!-- Trace -->
    <section v-if="vm.process" class="turn-zone">
      <TracePanel :digest="vm.process.trace" />
    </section>

    <!-- Evidence -->
    <section v-if="vm.evidence?.length" class="turn-zone">
      <KnowledgeSourceCard :groups="vm.evidence" />
    </section>

    <!-- 错误 -->
    <div v-if="vm.errors.length" class="turn-errors">
      <div v-for="(err, i) in vm.errors" :key="'err-' + i" class="turn-error-item">{{ err }}</div>
    </div>

    <!-- 加载 -->
    <div v-if="isLoading" class="turn-loading">
      <span class="dot"></span><span class="dot"></span><span class="dot"></span>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 更新 script — 使用 lucide 图标**

```typescript
<script setup lang="ts">
import { computed, ref } from 'vue'
import MarkdownIt from 'markdown-it'
import { Copy, CheckCheck, Wrench } from 'lucide-vue-next'
import type { ChatMessage } from '../types/chat'
import { buildAssistantTurnViewModel } from '../utils/assistantTurn'
import ResultBlock from './ResultBlock.vue'
import BaziChartCard from './BaziChartCard.vue'
import QimenChart from './QimenChart.vue'
import TracePanel from './TracePanel.vue'
import KnowledgeSourceCard from './KnowledgeSourceCard.vue'

const props = defineProps<{ message: ChatMessage; isLoading?: boolean }>()
const md = new MarkdownIt({ html: false, breaks: true, linkify: true })
const vm = computed(() => buildAssistantTurnViewModel(props.message))
const renderedAnswer = computed(() => md.render(vm.value.answerBlocks.join('\n\n')))
const copied = ref(false)

async function copyAnswer() {
  await navigator.clipboard.writeText(vm.value.answerBlocks.join('\n\n'))
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

function fmtMs(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}
</script>
```

- [ ] **Step 3: 更新 CSS — 竖线时间线链路**

```css
<style scoped>
.assistant-turn {
  margin-bottom: 24px;
  max-width: 100%;
}

.turn-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  padding: 0 8px;
}
.turn-role {
  font-family: var(--serif);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.turn-meta {
  font-size: 11px;
  color: var(--text-muted);
}

/* 竖线时间线 */
.turn-chain {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
  padding: 0 8px;
}
.chain-line {
  width: 2px;
  background: var(--border);
  border-radius: 1px;
  flex-shrink: 0;
}
.chain-items {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.chain-thought {
  font-size: 12px;
  color: var(--text-muted);
  font-style: italic;
  line-height: 1.5;
}
.chain-tool {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 4px 8px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.tool-icon { color: var(--accent-dim); flex-shrink: 0; }
.tool-name { color: var(--text-secondary); font-weight: 500; }
.tool-args {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text-muted);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.turn-zone { margin-bottom: 12px; }
.turn-zone:last-child { margin-bottom: 0; }

.answer-content {
  line-height: 1.85;
  color: var(--text-primary);
  font-size: 15px;
  padding: 0 8px;
}
.answer-content :deep(p) { margin: 0 0 10px; }
.answer-content :deep(p:last-child) { margin-bottom: 0; }
.answer-content :deep(ul), .answer-content :deep(ol) { margin: 6px 0; padding-left: 22px; }
.answer-content :deep(li) { margin: 3px 0; }
.answer-content :deep(h2), .answer-content :deep(h3) {
  font-family: var(--serif);
  font-size: 17px;
  font-weight: 600;
  margin: 16px 0 8px;
  color: var(--text-primary);
}
.answer-content :deep(blockquote) {
  margin: 10px 0;
  padding-left: 14px;
  border-left: 2px solid var(--accent);
  color: var(--text-secondary);
}

.answer-actions {
  display: flex;
  gap: 9px;
  padding: 6px 8px 0;
}
.act-btn {
  width: 28px; height: 28px;
  border: 1px solid var(--border);
  background: var(--bg-secondary);
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  transition: all 0.15s;
}
.act-btn:hover { color: var(--accent-dim); border-color: var(--accent); }

.turn-errors { padding: 0 8px; margin-top: 8px; }
.turn-error-item { color: #c47a6a; font-size: 13px; padding: 4px 0; }

.turn-loading {
  display: flex;
  gap: 5px;
  padding: 8px 8px;
}
.turn-loading .dot {
  width: 5px; height: 5px;
  background: var(--accent);
  border-radius: 50%;
  animation: blink 1.4s infinite both;
}
.turn-loading .dot:nth-child(2) { animation-delay: 0.2s; }
.turn-loading .dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes blink {
  0%, 80%, 100% { opacity: 0.2; }
  40% { opacity: 1; }
}
</style>
```

- [ ] **Step 4: 验证**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AssistantTurn.vue
git commit -m "refactor: timeline chain layout with lucide icons"
```

---

### Task 6: 适配 ChatBubble.vue + ResultBlock.vue 配色

**Files:**
- Modify: `web/src/components/ChatBubble.vue`
- Modify: `web/src/components/ResultBlock.vue`

- [ ] **Step 1: ChatBubble.vue — 更新用户气泡配色**

```css
/* 替换现有 style scoped */
<style scoped>
.bubble.assistant { max-width: 85%; text-align: left; }
.bubble.user {
  text-align: left;
  max-width: 70%;
  margin-left: auto;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  padding: 12px 16px;
  border-radius: var(--radius-md);
  color: var(--text-primary);
}
.error-msg { color: #c47a6a; padding: 8px 0; }
</style>
```

移除 ThinkingSegment/ToolCallSegment 引用（这些 segment 类型在 AssistantTurn 中处理）。

- [ ] **Step 2: ResultBlock.vue — 更新卡片配色**

```css
/* 替换现有 style scoped */
<style scoped>
.result-block {
  margin: 0 0 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  overflow: hidden;
  transition: box-shadow 0.2s;
}
.result-block:hover {
  box-shadow: var(--shadow-card);
}
.result-block__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px 0;
}
.result-block__title {
  font-family: var(--serif);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}
.result-block__body {
  padding: 14px 16px 16px;
}
</style>
```

- [ ] **Step 3: 验证**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add web/src/components/ChatBubble.vue web/src/components/ResultBlock.vue
git commit -m "refactor: adapt bubble and result block to warm palette"
```

---

## Phase 3: Sprite System

### Task 7: 创建 ElementSprite.vue（五行精灵）

**Files:**
- Create: `web/src/components/sprites/ElementSprite.vue`

5 个精灵合并在一个组件中，通过 `element` prop 切换。

- [ ] **Step 1: 创建组件**

```vue
<template>
  <svg :width="size" :height="size" viewBox="0 0 48 48" v-bind="$attrs">
    <!-- 木 -->
    <template v-if="element === 'wood'">
      <circle cx="24" cy="18" r="13" :fill="fillBg" :stroke="stroke" stroke-width="1.2"/>
      <circle cx="24" cy="18" r="8" :fill="fillInner" :stroke="stroke" stroke-width="0.8"/>
      <rect x="21" y="29" width="6" height="12" rx="3" fill="#d4c8b0" stroke="#b0a080" stroke-width="0.8"/>
      <line x1="20" y1="18" x2="14" y2="12" :stroke="stroke" stroke-width="0.8" stroke-linecap="round"/>
      <ellipse cx="12" cy="10" rx="4" ry="2.5" :fill="fillBg" :stroke="stroke" stroke-width="0.7" transform="rotate(-25 12 10)"/>
      <line x1="28" y1="16" x2="34" y2="10" :stroke="stroke" stroke-width="0.7" stroke-linecap="round"/>
      <ellipse cx="36" cy="8" rx="3.5" ry="2" :fill="fillBg" :stroke="stroke" stroke-width="0.6" transform="rotate(20 36 8)"/>
      <circle cx="24" cy="18" r="3" fill="#faf8f5" :stroke="stroke" stroke-width="0.6"/>
    </template>

    <!-- 火 -->
    <template v-if="element === 'fire'">
      <ellipse cx="24" cy="24" rx="14" ry="18" :fill="fillBg" :stroke="stroke" stroke-width="1.2"/>
      <ellipse cx="24" cy="24" rx="10" ry="14" :fill="fillInner" :stroke="stroke" stroke-width="0.8"/>
      <path d="M24 3 Q28 14 24 18 Q20 14 24 3" :fill="fillAccent" :stroke="stroke" stroke-width="1"/>
      <circle cx="18" cy="20" r="2.5" fill="#faf8f5" :stroke="stroke" stroke-width="0.6"/>
      <circle cx="30" cy="22" r="2" fill="#faf8f5" :stroke="stroke" stroke-width="0.6"/>
      <ellipse cx="16" cy="30" rx="4" ry="7" :fill="fillBg" :stroke="stroke" stroke-width="0.6" transform="rotate(-15 16 30)"/>
      <ellipse cx="32" cy="32" rx="3.5" ry="6" :fill="fillBg" :stroke="stroke" stroke-width="0.6" transform="rotate(10 32 32)"/>
    </template>

    <!-- 土 -->
    <template v-if="element === 'earth'">
      <polygon points="24,2 42,34 6,34" :fill="fillBg" :stroke="stroke" stroke-width="1.2" stroke-linejoin="round"/>
      <polygon points="24,10 36,34 12,34" :fill="fillInner" :stroke="stroke" stroke-width="0.7" stroke-linejoin="round"/>
      <line x1="24" y1="2" x2="24" y2="34" :stroke="stroke" stroke-width="0.4" stroke-dasharray="2 3" opacity="0.4"/>
      <circle cx="24" cy="8" r="2.5" fill="#faf8f5" :stroke="stroke" stroke-width="0.8"/>
      <rect x="16" y="20" width="2.5" height="2.5" rx="0.5" :fill="fillAccent" :stroke="stroke" stroke-width="0.5" transform="rotate(25 17 21)"/>
      <rect x="29" y="16" width="2" height="2" rx="0.5" :fill="fillAccent" :stroke="stroke" stroke-width="0.5" transform="rotate(-20 30 17)"/>
    </template>

    <!-- 金 -->
    <template v-if="element === 'metal'">
      <circle cx="24" cy="24" r="18" :fill="fillBg" :stroke="stroke" stroke-width="0.6" opacity="0.5"/>
      <polygon points="24,2 38,24 24,46 10,24" :fill="fillInner" :stroke="stroke" stroke-width="1.2" stroke-linejoin="round"/>
      <polygon points="24,9 34,24 24,39 14,24" :fill="fillAccent" :stroke="stroke" stroke-width="0.8" stroke-linejoin="round"/>
      <line x1="10" y1="24" x2="38" y2="24" :stroke="stroke" stroke-width="0.4" opacity="0.3"/>
      <line x1="24" y1="2" x2="24" y2="46" :stroke="stroke" stroke-width="0.4" opacity="0.3"/>
      <circle cx="24" cy="24" r="3.5" fill="#faf8f5" :stroke="stroke" stroke-width="1"/>
      <circle cx="16" cy="16" r="1" :fill="fillAccent" :stroke="stroke" stroke-width="0.4"/>
      <circle cx="34" cy="32" r="1" :fill="fillAccent" :stroke="stroke" stroke-width="0.4"/>
    </template>

    <!-- 水 -->
    <template v-if="element === 'water'">
      <ellipse cx="24" cy="24" rx="14" ry="15" :fill="fillBg" :stroke="stroke" stroke-width="1.2"/>
      <ellipse cx="24" cy="24" rx="10" ry="12" :fill="fillInner" :stroke="stroke" stroke-width="0.8"/>
      <path d="M14 18 Q24 12 34 18" :stroke="stroke" stroke-width="0.7" fill="none" stroke-linecap="round" opacity="0.5"/>
      <path d="M16 22 Q24 17 32 22" :stroke="stroke" stroke-width="0.6" fill="none" stroke-linecap="round" opacity="0.4"/>
      <circle cx="24" cy="28" r="2.5" fill="#faf8f5" :stroke="stroke" stroke-width="0.8"/>
      <circle cx="18" cy="10" r="2" :fill="fillBg" :stroke="stroke" stroke-width="0.5" opacity="0.6"/>
      <circle cx="30" cy="8" r="1.5" :fill="fillBg" :stroke="stroke" stroke-width="0.5" opacity="0.4"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  element: 'wood' | 'fire' | 'earth' | 'metal' | 'water'
  size?: number
}>(), { size: 48 })

const colors: Record<string, { stroke: string; fillBg: string; fillInner: string; fillAccent: string }> = {
  wood:   { stroke: '#8aa880', fillBg: '#e8f0e4', fillInner: '#d4e4cc', fillAccent: '#c0d8b4' },
  fire:   { stroke: '#c4806a', fillBg: '#fae8e0', fillInner: '#f5d8cc', fillAccent: '#f0c0a0' },
  earth:  { stroke: '#b09070', fillBg: '#f0e8d8', fillInner: '#e8dcc8', fillAccent: '#d4c0a0' },
  metal:  { stroke: '#c4a870', fillBg: '#f8f4e8', fillInner: '#f5eed8', fillAccent: '#efe0c0' },
  water:  { stroke: '#7a9ab8', fillBg: '#e0e8f2', fillInner: '#d0dce8', fillAccent: '#c0d0e0' },
}

const c = computed(() => colors[props.element])
const stroke = computed(() => c.value.stroke)
const fillBg = computed(() => c.value.fillBg)
const fillInner = computed(() => c.value.fillInner)
const fillAccent = computed(() => c.value.fillAccent)
</script>
```

- [ ] **Step 2: 验证 linter**

```bash
cd web && npx vue-tsc --noEmit src/components/sprites/ElementSprite.vue 2>&1 | head -5
```

- [ ] **Step 3: Commit**

```bash
git add web/src/components/sprites/ElementSprite.vue
git commit -m "feat: add 5-element sprite component"
```

---

### Task 8: 创建 SpriteStar.vue + SpriteDoor.vue

**Files:**
- Create: `web/src/components/sprites/SpriteStar.vue`
- Create: `web/src/components/sprites/SpriteDoor.vue`

- [ ] **Step 1: SpriteStar.vue — 九星参数化圆形图标**

```vue
<template>
  <svg :width="size" :height="size" viewBox="0 0 32 32">
    <circle cx="16" cy="16" r="14" :fill="cfg.bg" :stroke="cfg.stroke" stroke-width="1"/>
    <polygon
      :points="starPoints"
      :stroke="cfg.stroke"
      :fill="cfg.fill"
      stroke-width="0.8"
      stroke-linejoin="round"
    />
    <text x="16" y="27" text-anchor="middle" :fill="cfg.stroke" font-size="6" font-family="Inter,sans-serif" font-weight="600">{{ cfg.label }}</text>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  starType: 'tianpeng' | 'tianying' | 'tianchong' | 'tianfu' | 'tianqin' | 'tianxin' | 'tianzhu' | 'tianren' | 'tianrui'
  size?: number
}>(), { size: 32 })

const configs: Record<string, { points: number; stroke: string; bg: string; fill: string; label: string; starR: number }> = {
  tianpeng:  { points: 9, stroke: '#6b8aa8', bg: '#e8eef4', fill: '#d0dce8', label: '蓬', starR: 9 },
  tianying:  { points: 8, stroke: '#c47a6a', bg: '#faeae4', fill: '#f0d0c0', label: '英', starR: 8 },
  tianchong: { points: 7, stroke: '#7a9e7e', bg: '#ecf2ec', fill: '#d0e4d0', label: '冲', starR: 8 },
  tianfu:    { points: 6, stroke: '#7a9e7e', bg: '#ecf2ec', fill: '#d0e4d0', label: '辅', starR: 7 },
  tianqin:   { points: 5, stroke: '#b8956a', bg: '#f2ece0', fill: '#e8dcc8', label: '禽', starR: 7 },
  tianxin:   { points: 6, stroke: '#c4a96a', bg: '#f6f2e4', fill: '#efe4cc', label: '心', starR: 7 },
  tianzhu:   { points: 7, stroke: '#c4a96a', bg: '#f6f2e4', fill: '#efe4cc', label: '柱', starR: 8 },
  tianren:   { points: 8, stroke: '#b8956a', bg: '#f2ece0', fill: '#e8dcc8', label: '任', starR: 8 },
  tianrui:   { points: 9, stroke: '#b8956a', bg: '#f2ece0', fill: '#e8dcc8', label: '芮', starR: 9 },
}

const cfg = computed(() => configs[props.starType] || configs.tianqin)

const starPoints = computed(() => {
  const { points: n, starR: r } = cfg.value
  const cx = 16, cy = 16
  const pts: string[] = []
  for (let i = 0; i < n * 2; i++) {
    const radius = i % 2 === 0 ? r : r * 0.45
    const angle = (Math.PI * i) / n - Math.PI / 2
    pts.push(`${cx + radius * Math.cos(angle)},${cy + radius * Math.sin(angle)}`)
  }
  return pts.join(' ')
})
</script>
```

- [ ] **Step 2: SpriteDoor.vue — 八门参数化方形图标**

```vue
<template>
  <svg :width="size" :height="size" viewBox="0 0 32 32">
    <rect x="2" y="2" width="28" height="28" rx="4" :fill="bg" :stroke="stroke" stroke-width="1"/>
    <g :stroke="stroke" stroke-width="1.3" fill="none" stroke-linecap="round">
      <!-- 门框 -->
      <rect x="9" y="6" width="14" height="20" rx="2" :stroke="stroke" stroke-width="1"/>
      <!-- 门扇（开合度由 doorType 决定） -->
      <line x1="16" y1="6" x2="16" y2="26" :stroke="stroke" stroke-width="0.6" opacity="0.4"/>
      <path :d="doorPath" />
    </g>
    <text x="16" y="29" text-anchor="middle" :fill="stroke" font-size="6" font-family="Inter,sans-serif" font-weight="600">{{ label }}</text>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  doorType: 'kai' | 'xiu' | 'sheng' | 'shang' | 'du' | 'jing' | 'si' | 'jingmen'
  size?: number
}>(), { size: 32 })

const configs: Record<string, { stroke: string; bg: string; label: string; angle: number }> = {
  kai:     { stroke: '#c4a96a', bg: '#f8f4e8', label: '开', angle: -30 },
  xiu:     { stroke: '#6b8aa8', bg: '#e8eef4', label: '休', angle: -15 },
  sheng:   { stroke: '#b8956a', bg: '#f2ece0', label: '生', angle: -20 },
  shang:   { stroke: '#7a9e7e', bg: '#ecf2ec', label: '伤', angle: -10 },
  du:      { stroke: '#7a9e7e', bg: '#ecf2ec', label: '杜', angle: 0 },
  jing:    { stroke: '#c47a6a', bg: '#faeae4', label: '景', angle: 10 },
  si:      { stroke: '#b8956a', bg: '#f2ece0', label: '死', angle: 20 },
  jingmen: { stroke: '#c4a96a', bg: '#f8f4e8', label: '惊', angle: 30 },
}

const cfg = computed(() => configs[props.doorType] || configs.du)
const stroke = computed(() => cfg.value.stroke)
const bg = computed(() => cfg.value.bg)
const label = computed(() => cfg.value.label)

const doorPath = computed(() => {
  const a = (cfg.value.angle * Math.PI) / 180
  const cx = 16, top = 8, bot = 24, w = 6
  const x1 = cx + Math.sin(a) * w
  const x2 = cx - Math.sin(a) * w
  return `M${cx},${top} L${x1},${top + 8} L${x1},${bot - 4} L${cx},${bot}`
})
</script>
```

- [ ] **Step 3: 验证构建**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add web/src/components/sprites/SpriteStar.vue web/src/components/sprites/SpriteDoor.vue
git commit -m "feat: add parameterized star and door sprite components"
```

---

## Phase 4: Chart Components

### Task 9: 完全重写 BaziChartCard.vue

**Files:**
- Modify: `web/src/components/BaziChartCard.vue`

- [ ] **Step 1: 重写 template**

```vue
<template>
  <div class="bazi-card">
    <!-- 标题 -->
    <div class="bz-header">
      <span class="bz-title">八字命盘</span>
      <span class="bz-meta">日主 <strong>{{ dayGan }}</strong>（{{ dayGanWuxing }}）· {{ lunarDate }}</span>
    </div>

    <!-- 四柱卡片 -->
    <div class="bz-pillars">
      <div v-for="(p, i) in pillars" :key="p.name" class="bz-pillar" :style="{ animationDelay: i * 80 + 'ms' }">
        <div class="bz-p-label">{{ p.name }}</div>
        <ElementSprite :element="pillarElement(p)" :size="44" class="bz-p-sprite" />
        <div class="bz-p-ganzhi">{{ p.stem }}{{ p.branch }}</div>
        <div class="bz-p-shishen">{{ p.shiShen }}</div>
        <div class="bz-p-nayin">{{ p.naYin }}</div>
      </div>
    </div>

    <!-- 详情表 -->
    <table class="bz-table">
      <thead>
        <tr><th>柱</th><th>天干</th><th>地支</th><th>十神</th><th>纳音</th><th>空亡</th><th>地势</th><th>旬</th></tr>
      </thead>
      <tbody>
        <tr v-for="p in pillars" :key="'dt-'+p.name">
          <td>{{ p.name }}</td>
          <td class="strong">{{ p.stem }}</td>
          <td class="strong">{{ p.branch }}</td>
          <td class="shishen">{{ p.shiShen }}</td>
          <td class="dim">{{ p.naYin }}</td>
          <td class="dim">{{ p.xunKong }}</td>
          <td class="dim">{{ p.diShi }}</td>
          <td class="dim">{{ p.xun }}</td>
        </tr>
      </tbody>
    </table>

    <!-- 藏干 -->
    <div class="bz-hidegan">
      <span class="bz-hg-label">藏干</span>
      <span v-for="p in pillars" :key="'hg-'+p.name" class="bz-hg-item">
        {{ p.name }} <strong>{{ (p.hideGan || []).join(' ') }}</strong>
      </span>
    </div>

    <!-- 命宫/身宫/胎元 -->
    <div class="bz-extra">
      <span>命宫 <strong>{{ mingGong }}</strong><span class="dim"> {{ mingGongNaYin }}</span></span>
      <span>身宫 <strong>{{ shenGong }}</strong><span class="dim"> {{ shenGongNaYin }}</span></span>
      <span>胎元 <strong>{{ taiYuan }}</strong><span class="dim"> {{ taiYuanNaYin }}</span></span>
    </div>

    <!-- 五行条 -->
    <div class="bz-wuxing">
      <div v-for="(v, k) in wuxing" :key="k" class="bz-wx-row">
        <span class="bz-wx-label">{{ k }}</span>
        <div class="bz-wx-bar"><div class="bz-wx-fill" :class="'wx-' + elementKey(k)" :style="{ width: (v / 8 * 100) + '%' }"></div></div>
        <span class="bz-wx-count">{{ v }}</span>
      </div>
    </div>

    <!-- 大运 -->
    <div class="bz-dayun">
      <span v-for="(d, i) in dayun" :key="i" class="bz-dy-tag" :class="{ active: i === currentDayunIdx }">
        {{ d.startAge }}-{{ d.endAge }}岁 {{ d.ganZhi }}
      </span>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 更新 script**

```typescript
<script setup lang="ts">
import { computed } from 'vue'
import ElementSprite from './sprites/ElementSprite.vue'

const props = defineProps<{ data: any }>()
const pillars = computed(() => props.data?.pillars || [])
const dayGan = computed(() => props.data?.dayGan || '')
const dayGanWuxing = computed(() => props.data?.dayGanWuxing || '')
const lunarDate = computed(() => props.data?.lunarDate || '')
const wuxing = computed(() => props.data?.wuxing || {})
const dayun = computed(() => props.data?.dayun || [])
const mingGong = computed(() => props.data?.mingGong || '')
const mingGongNaYin = computed(() => props.data?.mingGongNaYin || '')
const shenGong = computed(() => props.data?.shenGong || '')
const shenGongNaYin = computed(() => props.data?.shenGongNaYin || '')
const taiYuan = computed(() => props.data?.taiYuan || '')
const taiYuanNaYin = computed(() => props.data?.taiYuanNaYin || '')

function pillarElement(p: any): 'wood' | 'fire' | 'earth' | 'metal' | 'water' {
  const wx = p.stemWuxing || p.branchWuxing || ''
  if (wx.includes('木')) return 'wood'
  if (wx.includes('火')) return 'fire'
  if (wx.includes('土')) return 'earth'
  if (wx.includes('金')) return 'metal'
  if (wx.includes('水')) return 'water'
  return 'earth'
}

function elementKey(k: string): string {
  const map: Record<string,string> = { '木':'wood','火':'fire','土':'earth','金':'metal','水':'water' }
  return map[k] || 'earth'
}

const currentDayunIdx = computed(() => {
  if (!props.data?.birthday) return -1
  const birthYear = parseInt(props.data.birthday) || 0
  const age = new Date().getFullYear() - birthYear
  return dayun.value.findIndex((d: any) => age >= d.startAge && age <= d.endAge)
})
</script>
```

- [ ] **Step 3: 添加 CSS**

```css
<style scoped>
.bazi-card {
  text-align: left;
  font-size: 13px;
}
.bz-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 16px;
  padding: 0 4px;
}
.bz-title {
  font-family: var(--serif);
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}
.bz-meta { font-size: 12px; color: var(--text-muted); }
.bz-meta strong { color: var(--text-secondary); }

.bz-pillars {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}
.bz-pillar {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 22px 10px 16px;
  text-align: center;
  opacity: 0;
  animation: pillar-in 0.4s cubic-bezier(0.22,0.61,0.36,1) forwards;
}
@keyframes pillar-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.bz-pillar:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-card);
}
.bz-p-label {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 2px;
  margin-bottom: 10px;
}
.bz-p-sprite { margin-bottom: 8px; display: block; margin-inline: auto; }
.bz-p-ganzhi {
  font-family: var(--serif);
  font-size: 26px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
  margin-bottom: 4px;
}
.bz-p-shishen { font-size: 12px; color: var(--accent-dim); font-weight: 500; }
.bz-p-nayin { font-size: 11px; color: var(--text-muted); margin-top: 2px; }

.bz-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  margin-bottom: 12px;
}
.bz-table th {
  text-align: center;
  padding: 8px 4px;
  font-weight: 500;
  color: var(--text-muted);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
  font-size: 11px;
}
.bz-table td {
  text-align: center;
  padding: 7px 4px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-light);
}
.bz-table .strong { font-weight: 600; color: var(--text-primary); }
.bz-table .shishen { color: var(--accent-dim); font-weight: 500; }
.bz-table .dim { color: var(--text-muted); }

.bz-hidegan {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.bz-hg-label { font-weight: 500; }
.bz-hg-item strong { color: var(--text-secondary); }

.bz-extra {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-muted);
  flex-wrap: wrap;
  margin-bottom: 14px;
}
.bz-extra strong { color: var(--text-secondary); }
.bz-extra .dim { color: var(--text-muted); opacity: 0.7; }

.bz-wuxing { margin-bottom: 14px; }
.bz-wx-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 5px 0;
  font-size: 12px;
}
.bz-wx-label { width: 18px; font-weight: 500; color: var(--text-muted); }
.bz-wx-bar { flex: 1; height: 6px; background: var(--border-light); border-radius: 3px; overflow: hidden; }
.bz-wx-fill { height: 100%; border-radius: 3px; transition: width 0.6s cubic-bezier(0.22,0.61,0.36,1); }
.bz-wx-fill.wx-wood  { background: var(--wx-wood); }
.bz-wx-fill.wx-fire  { background: var(--wx-fire); }
.bz-wx-fill.wx-earth { background: var(--wx-earth); }
.bz-wx-fill.wx-metal { background: var(--wx-metal); }
.bz-wx-fill.wx-water { background: var(--wx-water); }
.bz-wx-count { width: 18px; text-align: right; font-weight: 500; color: var(--text-secondary); }

.bz-dayun { display: flex; flex-wrap: wrap; gap: 6px; }
.bz-dy-tag {
  padding: 5px 12px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 500;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
}
.bz-dy-tag.active {
  background: var(--accent-bg);
  border-color: var(--accent);
  color: var(--accent-dim);
}
</style>
```

- [ ] **Step 4: 验证**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add web/src/components/BaziChartCard.vue
git commit -m "refactor: full rewrite of bazi chart with sprite cards"
```

---

### Task 10: 完全重写 QimenChart.vue

**Files:**
- Modify: `web/src/components/QimenChart.vue`

- [ ] **Step 1: 重写整个组件**

```vue
<template>
  <div class="qimen-card">
    <div class="qm-info">
      {{ data.ju_text }} · {{ data.duty_text }} · {{ data.question_time || '—' }}
    </div>

    <div class="qm-grid">
      <div
        v-for="cell in cells" :key="cell.palace"
        class="qm-cell"
        :class="{
          'qm-duty': cell.palace === dutyPalace && cell.palace !== '中',
          'qm-center': cell.palace === '中',
        }"
        :style="{ animationDelay: cellDelay(cell.palace) + 'ms' }"
      >
        <div class="qm-palace">{{ cell.palace }}</div>
        <ElementSprite :element="palaceElement(cell.palace)" :size="28" class="qm-sprite" />
        <div class="qm-star">{{ cell.star || '—' }}</div>
        <div class="qm-door">{{ cell.door || '—' }}</div>
        <div class="qm-god">{{ cell.god || '—' }}</div>
        <div class="qm-gans">{{ cell.guest_gan || '—' }} · {{ cell.host_gan || '—' }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ElementSprite from './sprites/ElementSprite.vue'

const props = defineProps<{ data: any }>()

const cells = computed(() => props.data?.cells || [])
const dutyPalace = computed(() => props.data?.duty_palace || '')

// 宫位→五行映射
const palaceWuxing: Record<string, 'wood'|'fire'|'earth'|'metal'|'water'> = {
  '坎': 'water', '坤': 'earth', '震': 'wood', '巽': 'wood',
  '中': 'earth', '乾': 'metal', '兑': 'metal', '艮': 'earth', '离': 'fire',
}
function palaceElement(palace: string) { return palaceWuxing[palace] || 'earth' }

// 中宫先出，四正宫后出，四隅最后
const displayOrder: Record<string, number> = { '中':0, '坎':1, '离':1, '震':1, '兑':1, '坤':2, '乾':2, '艮':2, '巽':2 }
function cellDelay(palace: string) { return (displayOrder[palace] ?? 2) * 60 }
</script>

<style scoped>
.qimen-card { text-align: left; }

.qm-info {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 16px;
  line-height: 1.6;
}

.qm-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.qm-cell {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 14px 6px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  min-height: 120px;
  opacity: 0;
  animation: cell-in 0.3s cubic-bezier(0.22,0.61,0.36,1) forwards;
}
@keyframes cell-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

.qm-cell.qm-duty {
  border-color: var(--accent);
  box-shadow: var(--shadow-glow);
}
.qm-cell.qm-center {
  background: var(--bg-hover);
}

.qm-palace {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  letter-spacing: 2px;
  text-transform: uppercase;
}
.qm-sprite { margin: 2px 0; }
.qm-star { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.qm-door { font-size: 11px; font-weight: 500; color: var(--text-secondary); }
.qm-god  { font-size: 10px; color: var(--text-muted); font-style: italic; }
.qm-gans {
  font-size: 10px;
  font-family: var(--mono);
  color: var(--text-muted);
  letter-spacing: 0.5px;
}
</style>
```

- [ ] **Step 2: 验证**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 3: Commit**

```bash
git add web/src/components/QimenChart.vue
git commit -m "refactor: full rewrite of qimen chart with sprite grid"
```

---

## Phase 5: Integration

### Task 11: 更新 App.vue

**Files:**
- Modify: `web/src/App.vue`

- [ ] **Step 1: 移除 darkTheme，使用默认亮色主题**

```vue
<template>
  <n-config-provider>
    <n-message-provider><ChatPanel /></n-message-provider>
  </n-config-provider>
</template>
<script setup lang="ts">
import { NConfigProvider, NMessageProvider } from 'naive-ui'
import ChatPanel from './components/ChatPanel.vue'
</script>
```

去掉 `darkTheme` import 和注入。

- [ ] **Step 2: 验证构建**

```bash
cd web && npx vite build 2>&1 | tail -5
```

- [ ] **Step 3: Commit**

```bash
git add web/src/App.vue
git commit -m "refactor: switch to light theme default"
```

---

### Task 12: 清理 + 最终验证

**Files:**
- Remove: `web/src/components/icons.ts`（被 lucide 替代）
- 验证 dev server 运行正常

- [ ] **Step 1: 移除旧 icons.ts**

```bash
rm web/src/components/icons.ts
```

- [ ] **Step 2: 启动 dev server 验证**

```bash
cd web && npx vite --port 5173 &
sleep 3
curl -s http://localhost:5173 | head -20
```

验证返回 HTML 页面，无白屏。

- [ ] **Step 3: 停止 dev server，构建验证**

```bash
kill %1 2>/dev/null
cd web && npx vite build 2>&1 | tail -10
```

Expected: 构建成功。

- [ ] **Step 4: Commit**

```bash
git add web/src/components/icons.ts
git commit -m "chore: remove legacy icons.ts, migrated to lucide"
```

---

## 验证清单

在完成所有任务后：

- [ ] `cd web && npx vite build` 构建成功
- [ ] 空状态：输入框居中、WelcomePanel 卡片正确渲染
- [ ] 发送消息：输入框移到底部、消息正常显示
- [ ] 八字卡片：四柱精灵正确、表格/五行条/大运正常
- [ ] 奇门卡片：九宫牌阵正确、值符高亮、精灵对应五行
- [ ] 加载状态：只有 AssistantTurn 内的 3 点跳动
- [ ] Stream 流式：思考链路竖线、工具 chip、正文渲染
