<template>
  <div class="exec-node" :style="{ paddingLeft: depth * 16 + 'px' }">
    <div class="exec-row" @click="toggle">
      <span class="exec-icon">{{ kindIcon }}</span>
      <span class="exec-label" :class="'status-' + node.status">{{ node.label }}</span>
      <span class="exec-kind">{{ node.kind }}</span>
      <span class="exec-ms">{{ fmtMs(node.ms) }}</span>
      <span v-if="hasChildren" class="exec-arrow">{{ expanded ? '▾' : '▸' }}</span>
    </div>
    <!-- Meta line -->
    <div v-if="node.meta && Object.keys(node.meta).length" class="exec-meta" :style="{ paddingLeft: (depth + 1) * 16 + 'px' }">
      <span v-if="node.meta.model" class="meta-tag">🤖 {{ node.meta.model }}</span>
      <span v-if="node.meta.output_tokens" class="meta-tag">📊 {{ node.meta.output_tokens }} tokens</span>
      <span v-if="node.meta.hits" class="meta-tag">📚 {{ node.meta.hits }} 命中</span>
      <span v-if="node.meta.batch_count" class="meta-tag">📤 {{ node.meta.batch_count }} 条</span>
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
.exec-children { border-left: 1px solid var(--border); margin-left: 8px; }
</style>
