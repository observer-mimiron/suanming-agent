<template>
  <div v-if="hasDebug" class="debug-trace">
    <div class="debug-summary" @click="showTree = !showTree">
      <span>{{ showTree ? '▾' : '▸' }} 执行链路</span>
      <span class="debug-meta">{{ fmtMs(root?.ms || digest?.total_ms || 0) }}</span>
    </div>
    <div v-if="runtimeTags.length" class="debug-runtime">
      <span v-for="tag in runtimeTags" :key="tag" class="debug-runtime-tag">{{ tag }}</span>
    </div>
    <div v-if="showTree && root" class="debug-tree">
      <ExecutionNodeItem
        v-for="(child, i) in root.children"
        :key="'ph-' + i"
        :node="child"
        :depth="0"
      />
    </div>
    <!-- Fallback: old flat format for backward compat -->
    <div v-else-if="showTree && digest?.steps?.length" class="debug-section">
      <div v-for="(step, i) in digest.steps" :key="'s-old-' + i" class="debug-old-step">
        <div class="debug-row-old">
          <span class="debug-label">{{ step.label }}</span>
          <span class="debug-kind">{{ step.kind }}</span>
          <span class="debug-ms">{{ fmtMs(step.ms) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import ExecutionNodeItem from './ExecutionNodeItem.vue'
import type { DebugTraceDigest, ExecutionNode } from '../types/chat'

const props = defineProps<{
  digest: DebugTraceDigest | null
  events: any[]
}>()

const showTree = ref(false)
const hasDebug = computed(() => {
  return !!(props.digest?.root || props.digest?.steps?.length)
})
const root = computed<ExecutionNode | null>(() => {
  return props.digest?.root ?? null
})

const runtimeTags = computed(() => {
  const runtime = props.digest?.runtime
  if (!runtime) return []

  const tags: string[] = []
  if (runtime.primary_domain) tags.push(`主领域 ${runtime.primary_domain}`)
  if (runtime.domains?.length) tags.push(`执行域 ${runtime.domains.join(' + ')}`)
  if (runtime.task_intent) tags.push(`任务 ${runtime.task_intent}`)
  if (runtime.execution_mode) tags.push(`执行模式 ${runtime.execution_mode}`)
  if (runtime.decision_source === 'cheap_followup_reuse') tags.push('决策来源 cheap gate')
  if (runtime.gate_reason) tags.push(`gate ${runtime.gate_reason}`)
  if (runtime.required_artifacts?.length) tags.push(`产物 ${runtime.required_artifacts.join(', ')}`)
  if (runtime.followup_policy) tags.push(`followup ${runtime.followup_policy}`)
  return tags
})

function fmtMs(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}
</script>

<style scoped>
.debug-trace {
  border: 1px dashed var(--border);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--bg-secondary) 86%, transparent);
  overflow: hidden;
}
.debug-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  padding: 10px 14px;
  font-size: 12px;
  color: var(--text-muted);
  user-select: none;
}
.debug-summary:hover { background: var(--bg-hover); }
.debug-meta { font-size: 11px; }
.debug-runtime {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 14px 10px;
}
.debug-runtime-tag {
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 2px 7px;
}
.debug-tree { padding: 0 8px 12px; }
.debug-section { padding: 0 14px 14px; }
.debug-old-step {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  padding: 8px 10px;
  margin-bottom: 6px;
}
.debug-row-old { display: flex; gap: 8px; align-items: center; }
.debug-label { flex: 1; font-size: 12px; color: var(--text-primary); }
.debug-kind, .debug-ms { font-size: 10px; color: var(--text-muted); }
</style>
