<template>
  <div class="assistant-turn">
    <!-- Header: thin label + timestamp -->
    <div class="turn-header">
      <span class="turn-role">命理大师</span>
      <span class="turn-meta" v-if="vm.process">
        {{ fmtMs(vm.process.trace.total_ms) }}
      </span>
    </div>

    <!-- Optional: thinking & tool calls (compact, collapsed) -->
    <div v-if="vm.thoughts.length || vm.toolCalls.length" class="turn-meta-block">
      <div class="meta-toggle" @click="showMeta = !showMeta">
        <span v-if="vm.thoughts.length" class="meta-badge">思考 {{ vm.thoughts.length }} 段</span>
        <span v-if="vm.toolCalls.length" class="meta-badge">工具 {{ vm.toolCalls.length }} 次</span>
        <span class="meta-arrow">{{ showMeta ? '▴' : '▾' }}</span>
      </div>
      <div v-if="showMeta" class="meta-detail">
        <div v-for="(t, i) in vm.thoughts" :key="'th-' + i" class="meta-thought">{{ t }}</div>
        <div v-for="(tc, i) in vm.toolCalls" :key="'tc-' + i" class="meta-tool">
          <span class="meta-tool-name">{{ tc.name }}</span>
          <code v-if="tc.arguments" class="meta-tool-args">{{ tc.arguments }}</code>
        </div>
      </div>
    </div>

    <!-- Zone 1: Structured Results -->
    <section v-if="vm.resultBlocks.length" class="turn-zone turn-results">
      <ResultBlock v-for="(rb, i) in vm.resultBlocks" :key="'rb-' + i">
        <template #icon>
          <span v-html="rb.type === 'bazi-chart' ? Icons.crystal : Icons.yinyang" />
        </template>
        <template #title>
          {{ rb.type === 'bazi-chart' ? '八字命盘' : '奇门遁甲' }}
        </template>
        <BaziChartCard v-if="rb.type === 'bazi-chart'" :data="rb.payload" />
        <QimenChart v-else-if="rb.type === 'qimen-chart'" :data="rb.payload" />
      </ResultBlock>
    </section>

    <!-- Zone 2: Main Answer -->
    <section v-if="vm.answerBlocks.length" class="turn-zone turn-answer">
      <div class="answer-content markdown-body" v-html="renderedAnswer" />
      <div class="answer-actions">
        <button class="act-btn" @click="copyAnswer" v-html="copied ? Icons.check : Icons.copy" />
      </div>
    </section>

    <!-- Zone 3: Process -->
    <section v-if="vm.process" class="turn-zone turn-process">
      <TracePanel :digest="vm.process.trace" />
    </section>

    <!-- Zone 4: Evidence -->
    <section v-if="vm.evidence?.length" class="turn-zone turn-evidence">
      <KnowledgeSourceCard :groups="vm.evidence" />
    </section>

    <!-- Tail: Error & Loading -->
    <div v-if="vm.errors.length" class="turn-errors">
      <div v-for="(err, i) in vm.errors" :key="'err-' + i" class="turn-error-item">{{ err }}</div>
    </div>

    <div v-if="isLoading" class="turn-loading">
      <span class="dot"></span><span class="dot"></span><span class="dot"></span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MarkdownIt from 'markdown-it'
import type { ChatMessage } from '../types/chat'
import { buildAssistantTurnViewModel } from '../utils/assistantTurn'
import ResultBlock from './ResultBlock.vue'
import BaziChartCard from './BaziChartCard.vue'
import QimenChart from './QimenChart.vue'
import TracePanel from './TracePanel.vue'
import KnowledgeSourceCard from './KnowledgeSourceCard.vue'
import { Icons } from './icons'

const props = defineProps<{ message: ChatMessage; isLoading?: boolean }>()

const md = new MarkdownIt({ html: false, breaks: true, linkify: true })

const vm = computed(() => buildAssistantTurnViewModel(props.message))

const renderedAnswer = computed(() => md.render(vm.value.answerBlocks.join('\n\n')))

const copied = ref(false)
const showMeta = ref(true) // 默认展开，让用户看到工具调用和思考过程

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

<style scoped>
.assistant-turn {
  margin-bottom: 24px;
  padding: 0;
  max-width: 100%;
}

.turn-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  padding: 0 4px;
}

.turn-role {
  font-size: 13px;
  font-weight: 600;
  color: var(--accent);
}

.turn-meta {
  font-size: 11px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

.turn-zone {
  margin-bottom: 12px;
}

.turn-zone:last-child {
  margin-bottom: 0;
}

/* Meta block (thinking, tool_calls) */
.turn-meta-block {
  margin-bottom: 12px;
}
.meta-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-muted);
  user-select: none;
  transition: background 0.12s;
}
.meta-toggle:hover {
  background: var(--bg-hover);
}
.meta-badge {
  padding: 1px 7px;
  border-radius: 4px;
  background: var(--bg-subtle);
  color: var(--text-muted);
}
.meta-arrow {
  font-size: 9px;
}
.meta-detail {
  padding: 8px 8px 4px;
  border-left: 2px solid var(--line-soft);
  margin-left: 8px;
  margin-top: 6px;
}
.meta-thought {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.5;
  margin-bottom: 4px;
  font-style: italic;
}
.meta-tool {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 12px;
  margin-bottom: 4px;
}
.meta-tool-name {
  color: var(--gold);
  font-weight: 500;
}
.meta-tool-args {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-subtle);
  padding: 1px 5px;
  border-radius: 3px;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Answer zone — article-like */
.answer-content {
  position: relative;
  line-height: 1.85;
  color: var(--text-primary);
  font-size: 15px;
  padding: 0 4px;
}

.answer-content :deep(p) { margin: 0 0 10px; }
.answer-content :deep(p:last-child) { margin-bottom: 0; }
.answer-content :deep(ul), .answer-content :deep(ol) { margin: 6px 0; padding-left: 22px; }
.answer-content :deep(li) { margin: 3px 0; }
.answer-content :deep(h1), .answer-content :deep(h2), .answer-content :deep(h3), .answer-content :deep(h4) {
  font-size: 16px; font-weight: 600; margin: 14px 0 6px; color: var(--text-primary);
}
.answer-content :deep(strong) { font-weight: 600; color: var(--text-primary); }
.answer-content :deep(blockquote) {
  margin: 10px 0; padding-left: 14px; border-left: 2px solid var(--accent); color: var(--text-secondary);
}
.answer-content :deep(code) {
  font-family: var(--mono); font-size: 13px; background: var(--bg-secondary); padding: 2px 6px; border-radius: 4px;
}

.answer-actions {
  display: flex;
  justify-content: flex-start;
  gap: 9px;
  padding: 6px 4px 0;
}

.act-btn {
  width: 26px;
  height: 26px;
  border: none;
  background: none;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.act-btn:hover { color: var(--text-primary); background: var(--bg-secondary); }

.turn-errors {
  margin-top: 8px;
  padding: 0 4px;
}
.turn-error-item {
  color: #e05555;
  font-size: 13px;
  padding: 4px 0;
}

.turn-loading {
  display: flex;
  gap: 5px;
  padding: 8px 4px;
}
.turn-loading .dot {
  width: 5px; height: 5px;
  background: var(--text-muted);
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
