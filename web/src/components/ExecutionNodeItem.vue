<template>
  <div class="exec-node" :style="{ paddingLeft: depth * 16 + 'px' }">
    <div class="exec-row" @click="toggle">
      <span class="exec-icon">{{ kindIcon }}</span>
      <span class="exec-label" :class="'status-' + node.status">{{ node.label }}</span>
      <span class="exec-kind">{{ node.kind }}</span>
      <span class="exec-ms">{{ fmtMs(node.ms) }}</span>
      <span v-if="hasChildren || hasDetail" class="exec-arrow">{{ expanded ? '▾' : '▸' }}</span>
    </div>

    <!-- Meta line -->
    <div v-if="node.meta && Object.keys(node.meta).length" class="exec-meta" :style="{ paddingLeft: (depth + 1) * 16 + 'px' }">
      <span v-if="node.meta.model" class="meta-tag">🤖 {{ node.meta.model }}</span>
      <span v-if="node.meta.output_tokens" class="meta-tag">📊 {{ node.meta.output_tokens }} tokens</span>
      <span v-if="node.meta.hits" class="meta-tag">📚 {{ node.meta.hits }} 命中</span>
      <span v-if="node.meta.batch_count" class="meta-tag">📤 {{ node.meta.batch_count }} 条</span>
      <span v-if="node.meta.degrade_reason" class="meta-tag meta-warn">⚠ {{ node.meta.degrade_reason }}</span>
    </div>

    <!-- Expanded detail (leaf nodes only) -->
    <div v-if="expanded && hasDetail && !hasChildren" class="exec-detail" :style="{ paddingLeft: (depth + 1) * 16 + 'px' }">
      <div v-if="node.meta?.query" class="detail-block">
        <div class="detail-label">搜索关键词</div>
        <div class="detail-value">{{ node.meta.query }}</div>
      </div>
      <div v-if="node.meta?.args" class="detail-block">
        <div class="detail-label">工具参数</div>
        <div class="detail-value detail-code">{{ fmtJSON(node.meta.args) }}</div>
      </div>
      <div v-if="node.meta?.response" class="detail-block">
        <div class="detail-label">工具返回</div>
        <div class="detail-value detail-code">{{ truncateStr(node.meta.response, 500) }}</div>
      </div>
      <div v-if="node.meta?.thinking" class="detail-block">
        <div class="detail-label">思考过程</div>
        <div class="detail-value">{{ node.meta.thinking }}</div>
      </div>
    </div>

    <!-- Thinking bubble (for phase nodes) -->
    <div v-if="!expanded && node.meta?.thinking" class="exec-thinking" :style="{ paddingLeft: (depth + 1) * 16 + 'px' }">
      <div class="thinking-bubble">{{ node.meta.thinking }}</div>
    </div>

    <!-- Children -->
    <div v-if="expanded && hasChildren" class="exec-children">
      <ExecutionNodeItem
        v-for="(child, i) in (node.children || [])"
        :key="'c-' + i"
        :node="child"
        :depth="depth + 1"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ExecutionNode } from '../types/chat'

const props = defineProps<{ node: ExecutionNode; depth: number }>()

const expanded = ref(false)

const hasChildren = computed(() => props.node.children && props.node.children.length > 0)

// Leaf nodes with meaningful detail (not just the 4 basic meta tags)
const hasDetail = computed(() => {
  const m = props.node.meta
  if (!m) return false
  return !!(m.query || m.args || m.response || m.thinking || m.degrade_reason)
})

const kindIcon = computed(() => {
  switch (props.node.kind) {
    case 'LLM': return '🧠'
    case 'TOOL': return '🔧'
    case 'RETRIEVER': return '📚'
    case 'AGENT': return '🔮'
    default: return '📋'
  }
})

function toggle() { expanded.value = !expanded.value }

function fmtMs(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}

function fmtJSON(v: any): string {
  if (typeof v === 'string') return v
  try { return JSON.stringify(v, null, 2) } catch { return String(v) }
}

function truncateStr(s: string, max: number): string {
  if (s.length <= max) return s
  return s.slice(0, max) + '…(' + s.length + '字符已截断)'
}
</script>

<style scoped>
.exec-node { font-size: 12px; }
.exec-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px 6px;
  cursor: pointer;
  border-radius: 4px;
  user-select: none;
}
.exec-row:hover { background: var(--bg-hover); }
.exec-icon { flex-shrink: 0; width: 18px; text-align: center; }
.exec-label { flex: 1; color: var(--text-primary); }
.exec-label.status-error { color: #c47a6a; font-weight: 500; }
.exec-label.status-degraded { color: #b8956a; }
.exec-kind { font-size: 10px; color: var(--text-muted); }
.exec-ms { font-size: 10px; color: var(--text-muted); min-width: 40px; text-align: right; }
.exec-arrow { font-size: 10px; color: var(--text-muted); width: 12px; }
.exec-meta { display: flex; gap: 8px; padding: 2px 0 4px; flex-wrap: wrap; }
.meta-tag { font-size: 10px; color: var(--text-muted); background: var(--bg-secondary); padding: 1px 6px; border-radius: 3px; }
.meta-warn { color: #b8956a; background: rgba(200,166,78,0.10); }
.exec-children { border-left: 1px solid var(--border); margin-left: 8px; }
.exec-thinking { padding: 2px 0; }
.thinking-bubble { font-size: 11px; color: var(--text-secondary); line-height: 1.5; padding: 6px 10px; border-radius: 6px; background: color-mix(in srgb, var(--bg-secondary) 60%, transparent); white-space: pre-wrap; word-break: break-word; }
.exec-detail { padding: 4px 0 8px; }
.detail-block { margin-bottom: 6px; }
.detail-label { font-size: 10px; color: var(--text-muted); margin-bottom: 2px; }
.detail-value { font-size: 11px; color: var(--text-secondary); line-height: 1.5; white-space: pre-wrap; word-break: break-word; }
.detail-code { font-family: var(--mono, monospace); font-size: 10px; background: var(--bg); padding: 4px 8px; border-radius: 4px; max-height: 200px; overflow-y: auto; }
</style>
