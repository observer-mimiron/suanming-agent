<template>
  <details v-if="hasDebug" class="debug-trace">
    <summary class="debug-summary">
      <span>调试详情</span>
      <span class="debug-meta">{{ debugCount }} 项</span>
    </summary>

    <div v-if="events.length" class="debug-section">
      <div class="debug-section-title">原始事件</div>
      <div v-for="(event, index) in events" :key="'evt-' + index" class="debug-card">
        <div class="debug-head">
          <span class="debug-label">{{ event.label }}</span>
          <span class="debug-kind">{{ event.type }}</span>
        </div>
        <pre class="debug-pre">{{ event.preview }}</pre>
        <pre v-if="event.result" class="debug-pre debug-result">{{ event.result }}</pre>
      </div>
    </div>

    <div v-if="digest?.steps?.length" class="debug-section">
      <div class="debug-section-title">Raw Trace</div>
      <div v-for="(step, index) in digest.steps" :key="'step-' + index" class="debug-card">
        <div class="debug-head">
          <span class="debug-label">{{ step.label }}</span>
          <span class="debug-kind">{{ step.kind }}</span>
          <span class="debug-ms">{{ fmtMs(step.ms) }}</span>
        </div>
        <div class="debug-raw">{{ step.name }}</div>
        <pre v-if="step.meta" class="debug-pre">{{ JSON.stringify(step.meta, null, 2) }}</pre>
      </div>
    </div>
  </details>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DebugEvent, DebugTraceDigest } from '../types/chat'

const props = defineProps<{
  digest: DebugTraceDigest | null
  events: DebugEvent[]
}>()

const debugCount = computed(() => (props.digest?.steps?.length ?? 0) + props.events.length)
const hasDebug = computed(() => debugCount.value > 0)

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
}

.debug-meta,
.debug-kind,
.debug-ms,
.debug-raw {
  color: var(--text-muted);
  font-size: 11px;
}

.debug-section {
  padding: 0 14px 14px;
}

.debug-section-title {
  margin: 6px 0 8px;
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-muted);
}

.debug-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  padding: 10px;
  margin-bottom: 8px;
}

.debug-head {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
}

.debug-label {
  flex: 1;
  font-size: 12px;
  color: var(--text-primary);
}

.debug-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-secondary);
}

.debug-result {
  margin-top: 8px;
}
</style>
