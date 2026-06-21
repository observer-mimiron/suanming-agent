<template>
  <div class="trace-panel" v-if="digest && digest.phases && digest.phases.length">
    <!-- collapsed summary -->
    <div class="trace-summary" @click="expanded = !expanded">
      <div class="trace-summary-left">
        <span class="trace-status-dot" :class="'dot-' + summaryStatus()" />
        <div class="trace-summary-copy">
          <div class="trace-summary-head">
            <span class="trace-label">处理过程</span>
            <span class="trace-total-ms">{{ fmtMs(digest.total_ms) }}</span>
            <span v-if="summaryStatus() !== 'ok'" class="trace-summary-badge" :class="'badge-' + summaryStatus()">{{ summaryLabel() }}</span>
          </div>
          <div v-if="summaryPhase()" class="trace-brief">
            <span class="trace-brief-label">{{ summaryPhase()?.label }}</span>
            <span v-if="summaryPhase()?.summary" class="trace-brief-text">{{ summaryPhase()?.summary }}</span>
          </div>
        </div>
      </div>
      <div class="trace-summary-right">
        <span class="trace-step-count">{{ digest.phases.length }} 个阶段</span>
        <span class="trace-toggle-icon">{{ expanded ? '▴' : '▾' }}</span>
      </div>
    </div>

    <!-- expanded detail -->
    <div v-if="expanded" class="trace-detail">
      <div class="trace-timeline">
        <div
          v-for="(phase, i) in digest.phases"
          :key="i"
          class="trace-step"
          :class="'step-' + phase.status"
        >
          <div class="step-line">
            <span class="step-dot" :class="'step-dot-' + phase.status" />
            <span v-if="i < digest.phases.length - 1" class="step-connector" />
          </div>
          <div class="step-content">
            <div class="step-head">
              <span class="step-label">{{ phase.label }}</span>
              <span class="step-ms">{{ fmtMs(phase.ms) }}</span>
            </div>
            <div v-if="phase.summary" class="step-summary">{{ phase.summary }}</div>
            <div class="step-meta" v-if="phase.meta">
              <span v-if="phase.meta.model" class="step-meta-tag">模型 {{ phase.meta.model }}</span>
              <span v-if="phase.meta.hits !== undefined" class="step-meta-tag">命中 {{ phase.meta.hits }}</span>
            </div>
            <div v-if="phase.status === 'error'" class="step-badge step-badge-err">失败</div>
            <div v-else-if="phase.status === 'degraded'" class="step-badge step-badge-warn">降级</div>
            <div v-else-if="phase.status === 'fallback'" class="step-badge step-badge-warn">回退</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ProcessDigest } from '../types/chat'

const props = defineProps<{ digest: ProcessDigest }>()

const expanded = ref(false)

function fmtMs(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}

function summaryStatus(): string {
  const s = props.digest?.status
  if (!s || s === 'ok') return 'ok'
  if (s === 'error') return 'err'
  return 'warn'
}

function summaryLabel(): string {
  const map: Record<string, string> = { degraded: '降级', fallback: '回退', error: '失败' }
  return map[props.digest?.status] || ''
}

function summaryPhase() {
  const phases = props.digest?.phases ?? []
  return phases[phases.length - 1]
}

</script>

<style scoped>
.trace-panel {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  overflow: hidden;
}

.trace-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}
.trace-summary:hover {
  background: var(--bg-hover);
}

.trace-summary-left {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.trace-summary-copy {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.trace-summary-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.trace-status-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.dot-ok  { background: #5a9e8f; }
.dot-err { background: #e05555; }
.dot-warn { background: #c8a64e; }

.trace-summary-badge {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  font-weight: 500;
  white-space: nowrap;
}
.badge-err  { background: rgba(224,85,85,0.12); color: #e08888; }
.badge-warn { background: rgba(200,166,78,0.12); color: #d4b96a; }

.trace-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.trace-total-ms {
  font-size: 12px;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.trace-brief {
  display: flex;
  gap: 6px;
  align-items: baseline;
  font-size: 11px;
  line-height: 1.5;
}

.trace-brief-label {
  color: var(--text-secondary);
  white-space: nowrap;
}

.trace-brief-text {
  color: var(--text-muted);
}

.trace-summary-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.trace-step-count {
  font-size: 12px;
  color: var(--text-muted);
}

.trace-toggle-icon {
  font-size: 10px;
  color: var(--text-muted);
}

.trace-detail {
  border-top: 1px solid var(--border);
  padding: 12px 14px 14px;
}

.trace-timeline {
  display: flex;
  flex-direction: column;
}

.trace-step {
  display: flex;
  gap: 10px;
}

.step-line {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 14px;
  flex-shrink: 0;
  padding-top: 5px;
}
.step-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.step-dot-ok { background: #5a9e8f; }
.step-dot-degraded, .step-dot-fallback { background: #c8a64e; }
.step-dot-error { background: #e05555; }

.step-connector {
  width: 1px;
  flex: 1;
  min-height: 12px;
  background: var(--border);
  margin: 2px 0;
}

.step-content {
  flex: 1;
  min-width: 0;
  padding: 0 0 12px;
}
.step-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.step-label {
  font-size: 13px;
  color: var(--text-primary);
  flex: 1;
}
.step-ms {
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

.step-summary {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-secondary);
}

.step-meta {
  display: flex;
  gap: 6px;
  margin-top: 3px;
  flex-wrap: wrap;
}
.step-meta-tag {
  font-size: 11px;
  color: var(--text-muted);
}

.step-badge {
  display: inline-block;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
  margin-top: 2px;
  font-weight: 500;
}
.step-badge-err  { background: rgba(224,85,85,0.12); color: #e08888; }
.step-badge-warn { background: rgba(200,166,78,0.12); color: #d4b96a; }
</style>
