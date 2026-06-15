<template>
  <div class="assistant-turn">
    <div class="turn-header">
      <span class="turn-role">命理大师</span>
      <span class="turn-meta" v-if="vm.process">{{ fmtMs(vm.process.trace.total_ms) }}</span>
    </div>

    <!-- Thinking + tool chain: vertical timeline -->
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

    <!-- Structured results -->
    <section v-if="vm.resultBlocks.length" class="turn-zone">
      <ResultBlock v-for="(rb, i) in vm.resultBlocks" :key="'rb-' + i">
        <template #title>{{ rb.type === 'bazi-chart' ? '八字命盘' : '奇门遁甲' }}</template>
        <BaziChartCard v-if="rb.type === 'bazi-chart'" :data="rb.payload" />
        <QimenChart v-else-if="rb.type === 'qimen-chart'" :data="rb.payload" />
      </ResultBlock>
    </section>

    <!-- Main answer -->
    <section v-if="vm.answerBlocks.length" class="turn-zone turn-answer">
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

    <!-- Errors -->
    <div v-if="vm.errors.length" class="turn-errors">
      <div v-for="(err, i) in vm.errors" :key="'err-' + i" class="turn-error-item">{{ err }}</div>
    </div>

    <!-- Loading (only one in the app) -->
    <div v-if="isLoading" class="turn-loading">
      <span class="dot"></span><span class="dot"></span><span class="dot"></span>
    </div>
  </div>
</template>

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

/* Vertical timeline chain */
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
