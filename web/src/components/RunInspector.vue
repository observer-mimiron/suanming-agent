<template>
  <div v-if="hasInspector" class="run-inspector" :class="'status-' + normalizedStatus">
    <button class="inspector-header" type="button" @click="expanded = !expanded">
      <div class="header-main">
        <span class="status-dot" :class="'dot-' + normalizedStatus" />
        <span class="header-title">Run Inspector</span>
        <span v-if="inspection?.trace_id" class="trace-id">{{ shortTraceId }}</span>
      </div>
      <div class="header-meta">
        <span v-if="summaryTags.length" class="header-tags">{{ summaryTags.join(' / ') }}</span>
        <span v-if="inspection?.total_ms !== undefined" class="header-ms">{{ fmtMs(inspection.total_ms) }}</span>
        <span class="toggle">{{ expanded ? '收起' : '展开' }}</span>
      </div>
    </button>

    <div class="triage">
      <div class="triage-title">{{ triageTitle }}</div>
      <div v-if="triageEvidence.length" class="triage-evidence">
        <span v-for="item in triageEvidence" :key="item" class="evidence-chip">{{ item }}</span>
      </div>
      <div v-if="triageNextAction" class="next-action">{{ triageNextAction }}</div>
      <div v-if="transportDiagnostics.length" class="transport-warnings">
        <div v-for="warning in transportDiagnostics" :key="warning" class="transport-warning">{{ warning }}</div>
      </div>
    </div>

    <div v-if="expanded" class="inspector-body">
      <div class="actions">
        <button class="action-btn" type="button" :disabled="!inspection?.trace_id" @click="copyTraceId">
          {{ traceCopied ? 'trace_id 已复制' : '复制 trace_id' }}
        </button>
        <button class="action-btn" type="button" :disabled="!inspection" @click="copyDiagnosisJson">
          {{ jsonCopied ? '诊断 JSON 已复制' : '复制诊断 JSON' }}
        </button>
        <button class="action-btn" type="button" :disabled="!inspection?.trace_id || rawTraceLoading" @click="loadRawTrace">
          {{ rawTraceLoading ? '加载中...' : rawTrace ? '刷新全量 Trace' : '加载全量 Trace' }}
        </button>
      </div>

      <div v-if="rawTrace || rawTraceError || rawTraceLoading" class="raw-trace">
        <div class="panel-title raw-title">
          <span>Raw Trace</span>
          <span v-if="rawTraceSearch" class="raw-hit">{{ rawTraceMatchCount }} 行匹配</span>
        </div>
        <div class="raw-toolbar">
          <input
            v-model="rawTraceSearch"
            class="raw-search"
            type="search"
            placeholder="搜索 key / value"
          />
          <label class="sensitive-toggle">
            <input v-model="showSensitiveRaw" type="checkbox" />
            显示敏感字段
          </label>
          <button class="action-btn" type="button" :disabled="!rawTrace" @click="copyRawTrace">
            {{ rawCopied ? 'Raw 已复制' : '复制全量 Raw' }}
          </button>
        </div>
        <div v-if="rawTraceLoading" class="empty-state">正在加载完整 TurnTrace...</div>
        <div v-else-if="rawTraceError" class="raw-error">{{ rawTraceError }}</div>
        <pre v-else class="raw-json">{{ rawTraceVisibleText }}</pre>
      </div>

      <div v-if="stageSummaries.length" class="stage-flow">
        <div class="panel-title">Agent Chain</div>
        <div class="stage-items">
          <button
            v-for="stage in stageSummaries"
            :key="stage.key"
            type="button"
            class="stage-chip"
            :class="'stage-' + stage.status"
            @click="selectedSpanId = stage.firstSpanId"
          >
            <span class="stage-label">{{ stage.label }}</span>
            <span class="stage-meta">{{ stage.count }} spans · {{ fmtMs(stage.durationMs) }}</span>
          </button>
        </div>
      </div>

      <div class="inspector-grid">
        <div class="span-tree">
          <div class="panel-title">Span Tree</div>
          <button
            v-for="row in flatSpans"
            :key="row.span.span_id"
            type="button"
            class="span-row"
            :class="['row-' + rowStatus(row.span), { selected: row.span.span_id === selectedSpanId }]"
            :style="{ paddingLeft: 10 + row.depth * 16 + 'px' }"
            @click="selectedSpanId = row.span.span_id"
          >
            <span class="span-status" :class="'dot-' + rowStatus(row.span)" />
            <span class="span-label">{{ row.span.label || row.span.name }}</span>
            <span class="span-kind">{{ row.span.category || row.span.kind }}</span>
            <span class="span-ms">{{ fmtMs(row.span.duration_ms) }}</span>
          </button>
          <div v-if="!flatSpans.length" class="empty-state">暂无 span 数据</div>
        </div>

        <div class="span-detail">
          <div class="panel-title">Span Detail</div>
          <template v-if="selectedSpan">
            <div class="detail-head">
              <div class="detail-name">{{ selectedSpan.label || selectedSpan.name }}</div>
              <div class="detail-sub">
                {{ selectedSpan.name }} · {{ selectedSpan.kind }} · {{ selectedSpan.category }} · {{ fmtMs(selectedSpan.duration_ms) }}
              </div>
            </div>
            <div class="detail-facts">
              <div class="fact-item">
                <span class="fact-key">status</span>
                <span class="fact-value" :class="'fact-' + rowStatus(selectedSpan)">{{ selectedSpan.status }}</span>
              </div>
              <div class="fact-item">
                <span class="fact-key">span_id</span>
                <code class="fact-value">{{ selectedSpan.span_id }}</code>
              </div>
              <div v-if="selectedSpan.parent_span_id" class="fact-item">
                <span class="fact-key">parent</span>
                <code class="fact-value">{{ selectedSpan.parent_span_id }}</code>
              </div>
              <div class="fact-item">
                <span class="fact-key">duration</span>
                <span class="fact-value">{{ fmtMs(selectedSpan.duration_ms) }}</span>
              </div>
            </div>
            <div v-if="selectedSpan.error" class="detail-error">{{ selectedSpan.error }}</div>
            <div v-if="selectedAttrGroups.length" class="attr-list">
              <div v-for="group in selectedAttrGroups" :key="group.label" class="attr-group">
                <div class="attr-group-title">{{ group.label }}</div>
                <div v-for="[key, value] in group.entries" :key="key" class="attr-row">
                  <span class="attr-key">{{ key }}</span>
                  <code class="attr-value">{{ formatAttr(value) }}</code>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">该 span 没有可展示的白名单属性</div>
          </template>
          <div v-else class="empty-state">选择一个 span 查看详情</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { RunDiagnostic, RunInspection, RunSpan, TransportInspection } from '../types/chat'

const props = defineProps<{
  inspection: RunInspection | null
  transport: TransportInspection | null
  isLoading?: boolean
}>()

const expanded = ref(false)
const selectedSpanId = ref<string | null>(null)
const traceCopied = ref(false)
const jsonCopied = ref(false)
const rawCopied = ref(false)
const rawTrace = ref<unknown | null>(null)
const rawTraceError = ref('')
const rawTraceLoading = ref(false)
const rawTraceSearch = ref('')
const showSensitiveRaw = ref(false)

const transportDiagnostics = computed(() => {
  const transport = props.transport
  if (!transport) return []
  const warnings = [...transport.parseWarnings]
  if (transport.requestError) warnings.push(transport.requestError)
  if (!props.isLoading && !transport.doneReceived) {
    warnings.push('前端未收到 done 事件，SSE 可能未完整收口。')
  }
  return warnings
})

const hasInspector = computed(() => {
  return !!props.inspection || transportDiagnostics.value.length > 0
})

const normalizedStatus = computed(() => {
  const status = props.inspection?.status || (transportDiagnostics.value.length ? 'warn' : 'ok')
  if (status === 'error') return 'error'
  if (status === 'degraded' || status === 'fallback' || status === 'warn') return 'warn'
  return 'ok'
})

const actionableDiagnostics = computed(() => {
  return (props.inspection?.diagnostics ?? []).filter((item) => item.severity === 'error' || item.severity === 'warn')
})

const primaryDiagnostic = computed<RunDiagnostic | null>(() => {
  return actionableDiagnostics.value.find((item) => item.severity === 'error') ?? actionableDiagnostics.value[0] ?? null
})

const hasProblem = computed(() => {
  return normalizedStatus.value !== 'ok' || actionableDiagnostics.value.length > 0 || transportDiagnostics.value.length > 0
})

watch(hasProblem, (value) => {
  if (value) expanded.value = true
}, { immediate: true })

const shortTraceId = computed(() => {
  const id = props.inspection?.trace_id ?? ''
  if (id.length <= 14) return id
  return id.slice(0, 10) + '…'
})

const summaryTags = computed(() => {
  const summary = props.inspection?.summary
  if (!summary) return []
  return [
    summary.primary_domain,
    summary.task_intent,
    summary.decision_source,
  ].filter((value): value is string => !!value)
})

const triageTitle = computed(() => {
  if (primaryDiagnostic.value) return primaryDiagnostic.value.title
  return props.inspection?.summary?.inspection_text ?? '暂无运行诊断'
})

const triageEvidence = computed(() => primaryDiagnostic.value?.evidence ?? [])
const triageNextAction = computed(() => primaryDiagnostic.value?.next_action ?? '')

interface FlatSpanRow {
  span: RunSpan
  depth: number
}

interface StageSummary {
  key: string
  label: string
  status: string
  count: number
  durationMs: number
  firstSpanId: string
}

interface AttrGroup {
  label: string
  entries: [string, unknown][]
}

const flatSpans = computed<FlatSpanRow[]>(() => {
  const spans = props.inspection?.spans ?? []
  const children = new Map<string, RunSpan[]>()
  const roots: RunSpan[] = []
  for (const span of spans) {
    const parent = span.parent_span_id || ''
    if (!parent) {
      roots.push(span)
      continue
    }
    if (!children.has(parent)) children.set(parent, [])
    children.get(parent)!.push(span)
  }
  const knownIds = new Set(spans.map((span) => span.span_id))
  for (const span of spans) {
    if (span.parent_span_id && !knownIds.has(span.parent_span_id)) {
      roots.push(span)
    }
  }
  const seen = new Set<string>()
  const rows: FlatSpanRow[] = []
  const visit = (span: RunSpan, depth: number) => {
    if (seen.has(span.span_id)) return
    seen.add(span.span_id)
    rows.push({ span, depth })
    for (const child of children.get(span.span_id) ?? []) {
      visit(child, depth + 1)
    }
  }
  for (const root of roots) visit(root, 0)
  for (const span of spans) visit(span, 0)
  return rows
})

const selectedSpan = computed(() => {
  return flatSpans.value.find((row) => row.span.span_id === selectedSpanId.value)?.span ?? null
})

const stageSummaries = computed<StageSummary[]>(() => {
  const order: string[] = []
  const byKey = new Map<string, StageSummary>()
  for (const row of flatSpans.value) {
    const stage = stageForSpan(row.span)
    let current = byKey.get(stage.key)
    if (!current) {
      current = {
        key: stage.key,
        label: stage.label,
        status: rowStatus(row.span),
        count: 0,
        durationMs: 0,
        firstSpanId: row.span.span_id,
      }
      byKey.set(stage.key, current)
      order.push(stage.key)
    }
    current.count += 1
    current.durationMs += row.span.duration_ms || 0
    current.status = worseStatus(current.status, rowStatus(row.span))
  }
  return order.map((key) => byKey.get(key)!)
})

const selectedAttrGroups = computed<AttrGroup[]>(() => {
  const attrs = selectedSpan.value?.attributes
  if (!attrs || !Object.keys(attrs).length) return []
  const groups = new Map<string, [string, unknown][]>()
  for (const [key, value] of Object.entries(attrs).sort(([a], [b]) => a.localeCompare(b))) {
    const label = attrGroupLabel(key)
    if (!groups.has(label)) groups.set(label, [])
    groups.get(label)!.push([key, value])
  }
  return Array.from(groups.entries()).map(([label, entries]) => ({ label, entries }))
})

const rawTraceDisplay = computed(() => {
  if (!rawTrace.value) return ''
  const value = showSensitiveRaw.value ? rawTrace.value : redactSensitiveTrace(rawTrace.value)
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
})

const rawTraceVisibleLines = computed(() => {
  const text = rawTraceDisplay.value
  const query = rawTraceSearch.value.trim().toLowerCase()
  if (!query) return text.split('\n')
  return text.split('\n').filter((line) => line.toLowerCase().includes(query))
})

const rawTraceMatchCount = computed(() => {
  const query = rawTraceSearch.value.trim().toLowerCase()
  if (!query) return 0
  return rawTraceDisplay.value.split('\n').filter((line) => line.toLowerCase().includes(query)).length
})

const rawTraceVisibleText = computed(() => rawTraceVisibleLines.value.join('\n'))

watch(flatSpans, (rows) => {
  if (!rows.length) {
    selectedSpanId.value = null
    return
  }
  if (selectedSpanId.value && rows.some((row) => row.span.span_id === selectedSpanId.value)) return
  const diagnosticSpan = primaryDiagnostic.value?.span_id
  selectedSpanId.value = rows.find((row) => row.span.span_id === diagnosticSpan)?.span.span_id ?? rows[0].span.span_id
}, { immediate: true })

function rowStatus(span: RunSpan): string {
  if (span.status === 'error' || span.error) return 'error'
  if (span.status === 'degraded' || span.status === 'fallback') return 'warn'
  return 'ok'
}

function worseStatus(current: string, next: string): string {
  const rank: Record<string, number> = { ok: 1, warn: 2, error: 3 }
  return (rank[next] ?? 1) > (rank[current] ?? 1) ? next : current
}

function stageForSpan(span: RunSpan): { key: string; label: string } {
  const name = span.name || ''
  if (name.includes('supervisor') || name.includes('route') || name === 'chat.turn') {
    return { key: 'route', label: '路由/入口' }
  }
  if (name.includes('policy') || name.includes('preflight') || name.includes('prefill')) {
    return { key: 'policy', label: '策略/准备' }
  }
  if (span.category === 'retriever' || name.includes('knowledge')) {
    return { key: 'retrieval', label: '知识检索' }
  }
  if (span.category === 'bazi_graph' || span.category === 'agent' || span.category === 'llm' || name.includes('specialist')) {
    return { key: 'agent', label: '领域 Agent' }
  }
  if (span.category === 'tool') {
    return { key: 'tool', label: '工具执行' }
  }
  if (span.category === 'guard' || name.includes('guard') || name.includes('contract')) {
    return { key: 'guard', label: '合同/护栏' }
  }
  if (span.category === 'sse' || name.includes('sse')) {
    return { key: 'sse', label: 'SSE 输出' }
  }
  return { key: 'runtime', label: '运行时' }
}

function attrGroupLabel(key: string): string {
  if (key.startsWith('failure.')) return '失败'
  if (key.startsWith('bazi.')) return '八字合同'
  if (key.startsWith('gate.') || ['decision_source', 'primary_domain', 'task_intent', 'followup_mode', 'turn_type', 'domains', 'required_artifacts'].includes(key)) return '路由/运行时'
  if (key.startsWith('gen_ai.') || ['model', 'output_tokens', 'input.message_count', 'input.message_roles'].includes(key)) return '模型'
  if (['query', 'hits', 'top_k', 'filter', 'degrade_reason', 'topic', 'source_tier', 'degraded'].includes(key)) return '检索'
  if (key.startsWith('tool.') || key.includes('dispatch')) return '工具/调度'
  if (key.includes('sse') || key === 'event_type') return '传输'
  return '其他'
}

function fmtMs(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}

function formatAttr(value: unknown): string {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

async function copyTraceId() {
  const traceId = props.inspection?.trace_id
  if (!traceId) return
  await navigator.clipboard.writeText(traceId)
  traceCopied.value = true
  setTimeout(() => { traceCopied.value = false }, 1800)
}

async function copyDiagnosisJson() {
  if (!props.inspection) return
  await navigator.clipboard.writeText(JSON.stringify(props.inspection, null, 2))
  jsonCopied.value = true
  setTimeout(() => { jsonCopied.value = false }, 1800)
}

async function loadRawTrace() {
  const traceId = props.inspection?.trace_id
  if (!traceId || rawTraceLoading.value) return
  rawTraceLoading.value = true
  rawTraceError.value = ''
  try {
    const resp = await fetch('/api/debug/traces/' + encodeURIComponent(traceId))
    if (!resp.ok) {
      rawTrace.value = null
      rawTraceError.value = resp.status === 404
        ? '未找到全量 trace。请确认本地已开启 DEBUG_HTTP=1 且 DEBUG_TRACE=1。'
        : '加载全量 trace 失败：HTTP ' + resp.status
      return
    }
    rawTrace.value = await resp.json()
  } catch (err) {
    rawTrace.value = null
    rawTraceError.value = err instanceof Error ? err.message : '加载全量 trace 失败。'
  } finally {
    rawTraceLoading.value = false
  }
}

async function copyRawTrace() {
  if (!rawTrace.value) return
  await navigator.clipboard.writeText(JSON.stringify(rawTrace.value, null, 2))
  rawCopied.value = true
  setTimeout(() => { rawCopied.value = false }, 1800)
}

function redactSensitiveTrace(value: unknown): unknown {
  if (Array.isArray(value)) return value.map((item) => redactSensitiveTrace(item))
  if (!value || typeof value !== 'object') return value
  const out: Record<string, unknown> = {}
  for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
    if (isSensitiveTraceKey(key)) {
      out[key] = '[已折叠：勾选“显示敏感字段”查看]'
    } else {
      out[key] = redactSensitiveTrace(child)
    }
  }
  return out
}

function isSensitiveTraceKey(key: string): boolean {
  const normalized = key.toLowerCase()
  return normalized === 'user_message' ||
    normalized === 'input.value' ||
    normalized === 'output.value' ||
    normalized === 'input.messages.preview' ||
    normalized === 'langfuse.trace.input' ||
    normalized === 'langfuse.trace.output' ||
    normalized.includes('prompt') ||
    normalized.includes('raw_output')
}
</script>

<style scoped>
.run-inspector {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--bg-secondary) 88%, transparent);
  overflow: hidden;
}
.inspector-header {
  width: 100%;
  border: 0;
  background: transparent;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  cursor: pointer;
}
.inspector-header:hover { background: var(--bg-hover); }
.header-main, .header-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.header-title { font-size: 13px; font-weight: 600; }
.trace-id, .header-tags, .header-ms, .toggle {
  font-size: 11px;
  color: var(--text-muted);
}
.header-tags {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.status-dot, .span-status {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.dot-ok { background: #5a9e8f; }
.dot-warn { background: #c8a64e; }
.dot-error { background: #e05555; }
.triage {
  border-top: 1px solid var(--border);
  padding: 10px 14px;
}
.triage-title {
  font-size: 12px;
  color: var(--text-primary);
  line-height: 1.5;
}
.triage-evidence {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 7px;
}
.evidence-chip {
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 2px 6px;
  max-width: 100%;
  overflow-wrap: anywhere;
}
.next-action {
  margin-top: 7px;
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.5;
}
.transport-warnings { margin-top: 7px; }
.transport-warning {
  font-size: 11px;
  color: #c47a6a;
  line-height: 1.5;
}
.inspector-body {
  border-top: 1px solid var(--border);
  padding: 10px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}
.action-btn {
  border: 1px solid var(--border);
  background: var(--bg);
  color: var(--text-secondary);
  border-radius: 6px;
  padding: 5px 9px;
  font-size: 11px;
  cursor: pointer;
}
.action-btn:disabled {
  opacity: 0.45;
  cursor: default;
}
.raw-trace {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  margin-bottom: 10px;
  overflow: hidden;
}
.raw-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.raw-hit {
  color: var(--text-muted);
  font-size: 10px;
}
.raw-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 8px;
  border-bottom: 1px solid var(--border);
}
.raw-search {
  min-width: 180px;
  flex: 1;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 11px;
  padding: 6px 8px;
}
.sensitive-toggle {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--text-secondary);
  font-size: 11px;
}
.raw-error {
  color: #c47a6a;
  font-size: 11px;
  line-height: 1.5;
  padding: 10px;
}
.raw-json {
  color: var(--text-secondary);
  font-family: var(--mono, monospace);
  font-size: 10px;
  line-height: 1.55;
  margin: 0;
  max-height: 360px;
  overflow: auto;
  padding: 10px;
  white-space: pre-wrap;
  word-break: break-word;
}
.stage-flow {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  margin-bottom: 10px;
  overflow: hidden;
}
.stage-items {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 6px;
  padding: 8px;
}
.stage-chip {
  border: 1px solid var(--border);
  border-radius: 6px;
  background: color-mix(in srgb, var(--bg-secondary) 70%, transparent);
  color: var(--text-primary);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  padding: 7px 8px;
  text-align: left;
}
.stage-chip:hover { background: var(--bg-hover); }
.stage-error {
  border-color: rgba(224, 85, 85, 0.55);
  background: rgba(224, 85, 85, 0.08);
}
.stage-warn {
  border-color: rgba(200, 166, 78, 0.55);
  background: rgba(200, 166, 78, 0.08);
}
.stage-label {
  font-size: 12px;
  font-weight: 600;
}
.stage-meta {
  color: var(--text-muted);
  font-size: 10px;
}
.inspector-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.9fr);
  gap: 10px;
}
.span-tree, .span-detail {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  min-width: 0;
  overflow: hidden;
}
.panel-title {
  font-size: 11px;
  color: var(--text-muted);
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
}
.span-row {
  width: 100%;
  border: 0;
  border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
  background: transparent;
  color: var(--text-primary);
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  text-align: left;
}
.span-row:hover, .span-row.selected { background: var(--bg-hover); }
.span-label {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.span-kind, .span-ms {
  font-size: 10px;
  color: var(--text-muted);
}
.row-error .span-label { color: #c47a6a; font-weight: 600; }
.row-warn .span-label { color: #b8956a; }
.detail-head { padding: 10px; border-bottom: 1px solid var(--border); }
.detail-name { font-size: 13px; color: var(--text-primary); font-weight: 600; }
.detail-sub { margin-top: 3px; font-size: 10px; color: var(--text-muted); overflow-wrap: anywhere; }
.detail-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
}
.fact-item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fact-key {
  color: var(--text-muted);
  font-size: 10px;
}
.fact-value {
  color: var(--text-secondary);
  font-size: 10px;
  overflow-wrap: anywhere;
}
.fact-error { color: #c47a6a; font-weight: 600; }
.fact-warn { color: #b8956a; font-weight: 600; }
.detail-error {
  margin: 10px;
  padding: 8px;
  border-radius: 6px;
  background: rgba(196, 122, 106, 0.10);
  color: #c47a6a;
  font-size: 11px;
  overflow-wrap: anywhere;
}
.attr-list { padding: 8px 10px 10px; }
.attr-group {
  border-bottom: 1px solid color-mix(in srgb, var(--border) 60%, transparent);
  padding: 6px 0;
}
.attr-group:last-child { border-bottom: 0; }
.attr-group-title {
  color: var(--text-primary);
  font-size: 11px;
  font-weight: 600;
  margin-bottom: 4px;
}
.attr-row {
  display: grid;
  grid-template-columns: minmax(110px, 0.45fr) minmax(0, 1fr);
  gap: 8px;
  padding: 4px 0;
}
.attr-key {
  font-size: 10px;
  color: var(--text-muted);
  overflow-wrap: anywhere;
}
.attr-value {
  font-family: var(--mono, monospace);
  font-size: 10px;
  color: var(--text-secondary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.empty-state {
  padding: 10px;
  font-size: 11px;
  color: var(--text-muted);
}
@media (max-width: 760px) {
  .inspector-header {
    align-items: flex-start;
    flex-direction: column;
  }
  .inspector-grid {
    grid-template-columns: 1fr;
  }
}
</style>
