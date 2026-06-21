<template>
  <div class="assistant-turn">
    <div class="turn-header">
      <span class="turn-role">命理大师</span>
      <span class="turn-meta" v-if="vm.process">{{ fmtMs(vm.process.digest.total_ms) }}</span>
    </div>

    <!-- Structured results -->
    <section v-if="vm.resultBlocks.length" class="turn-zone">
      <ResultBlock v-for="(rb, i) in vm.resultBlocks" :key="'rb-' + i">
        <template #title>{{ rb.type === 'bazi-chart' ? '八字命盘' : rb.type === 'ziwei-chart' ? '紫微斗数' : '奇门遁甲' }}</template>
        <BaziChartCard v-if="rb.type === 'bazi-chart'" :data="rb.payload" />
        <QimenChart v-else-if="rb.type === 'qimen-chart'" :data="rb.payload" />
        <ZiweiChartCard v-else-if="rb.type === 'ziwei-chart'" :data="rb.payload" />
      </ResultBlock>
    </section>

    <!-- Main answer -->
    <section v-if="vm.answerBlocks.length" class="turn-zone turn-answer">
      <div class="answer-content markdown-body" v-html="renderedAnswer" />
      <div class="answer-actions">
        <button class="act-btn" @click="copyAnswer" :title="copied ? '已复制' : '复制'">
          <CheckCheck v-if="copied" :size="13" />
          <Copy v-else :size="13" />
        </button>
      </div>
    </section>

    <!-- Trace -->
    <section v-if="vm.process" class="turn-zone">
      <TracePanel :digest="vm.process.digest" />
    </section>

    <!-- Evidence -->
    <section v-if="vm.evidence?.length" class="turn-zone">
      <KnowledgeSourceCard :groups="vm.evidence" />
    </section>

    <!-- Debug -->
    <section v-if="vm.debugTrace || vm.debugEvents.length" class="turn-zone">
      <DebugTracePanel :digest="vm.debugTrace" :events="vm.debugEvents" />
    </section>

    <!-- Errors -->
    <div v-if="vm.errors.length" class="turn-errors">
      <div v-for="(err, i) in vm.errors" :key="'err-' + i" class="turn-error-item">{{ err }}</div>
    </div>

    <!-- Loading (only one in the app) -->
    <div v-if="isLoading" class="turn-loading">
      <div class="celestial-loader">
        <div class="star-center"></div>
        <div class="orbit inner-orbit">
          <div class="planet p1"></div>
        </div>
        <div class="orbit outer-orbit">
          <div class="planet p2"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MarkdownIt from 'markdown-it'
import { Copy, CheckCheck } from 'lucide-vue-next'
import type { ChatMessage } from '../types/chat'
import { buildAssistantTurnViewModel } from '../utils/assistantTurn'
import ResultBlock from './ResultBlock.vue'
import BaziChartCard from './BaziChartCard.vue'
import QimenChart from './QimenChart.vue'
import ZiweiChartCard from './ZiweiChartCard.vue'
import TracePanel from './TracePanel.vue'
import DebugTracePanel from './DebugTracePanel.vue'
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

.turn-zone { margin-bottom: 12px; }

.answer-content {
  line-height: 1.85;
  color: var(--text-primary);
  font-size: 15px;
  padding: 0 4px;
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
  gap: 6px;
  padding: 8px 8px 0;
}
.act-btn {
  width: 26px; height: 26px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.15s;
  opacity: 0.5;
}
.act-btn:hover {
  opacity: 1;
  color: var(--accent-dim);
  background: var(--bg-hover);
}

.turn-errors { padding: 0 8px; margin-top: 8px; }
.turn-error-item { color: #c47a6a; font-size: 13px; padding: 4px 0; }

.turn-loading {
  display: flex;
  padding: 12px 8px;
  align-items: center;
}
.celestial-loader {
  position: relative;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.star-center {
  width: 4px;
  height: 4px;
  background-color: var(--accent);
  border-radius: 50%;
  box-shadow: 0 0 6px var(--accent);
}
.orbit {
  position: absolute;
  border: 1px dashed var(--border);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}
.inner-orbit {
  width: 18px;
  height: 18px;
  animation: rotate-orbit 4s linear infinite;
}
.outer-orbit {
  width: 30px;
  height: 30px;
  animation: rotate-orbit 7s linear infinite reverse;
}
.planet {
  position: absolute;
  width: 3.5px;
  height: 3.5px;
  background-color: var(--accent);
  border-radius: 50%;
}
.p1 {
  top: -2px;
  box-shadow: 0 0 4px var(--accent);
}
.p2 {
  bottom: -2.5px;
  background-color: var(--accent-dim);
}

@keyframes rotate-orbit {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
