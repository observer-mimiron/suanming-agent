<!--
  QimenChart renders the Qi Men Dun Jia nine-palace board.
  It owns palace-level visual layering only; the backend remains responsible for
  casting the chart and deciding which symbols belong in each palace.
-->
<template>
  <div class="qimen-card">
    <header class="qm-header">
      <div>
        <div class="qm-kicker">奇门遁甲</div>
        <div class="qm-title">九宫三盘</div>
      </div>
      <div class="qm-info">
        <span class="qm-info-chip">{{ data.ju_text || '局数未定' }}</span>
        <span class="qm-info-chip accent">{{ data.duty_text || '值符值使未定' }}</span>
        <span class="qm-info-chip">{{ data.pan_schema || '口径未标注' }}</span>
        <span class="qm-info-chip">{{ data.question_time || '未提供起局时间' }}</span>
      </div>
    </header>

    <section class="qm-case-meta" aria-label="问事 Case 信息">
      <div><span>问事目的</span><strong>{{ data.purpose || '—' }}</strong></div>
      <div><span>Case ID</span><strong>{{ data.case_id || '—' }}</strong></div>
      <div><span>资产归属</span><strong>{{ ownerRefText(data.owner_ref) }}</strong></div>
      <div><span>提问时间</span><strong>{{ data.question_time || '—' }}</strong></div>
      <div><span>时间来源</span><strong>{{ data.time_source || '—' }}</strong></div>
      <div><span>符号体系</span><strong>{{ data.symbol_system || '—' }}</strong></div>
    </section>

    <p v-if="rotatingEightWarning" class="qm-warning" role="alert">{{ rotatingEightWarning }}</p>

    <section class="qm-board" aria-label="奇门九宫盘">
      <article
        v-for="cell in gridCells"
        :key="cell.palace"
        class="qm-cell"
        :class="[
          'qm-' + palaceElement(cell.palace),
          {
            'is-duty': isDutyCell(cell.palace),
            'is-center': cell.palace === '中',
            'is-empty-center': cell.isCenterDummy,
          },
        ]"
        :style="{ animationDelay: cellDelay(cell.palace) + 'ms' }"
      >
        <template v-if="cell.isCenterDummy">
          <div class="qm-center-mark">
            <svg class="qm-taiji" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 2a5 5 0 0 0 0 10 5 5 0 0 1 0 10" />
              <circle cx="12" cy="7" r="1.5" fill="currentColor" />
              <circle cx="12" cy="17" r="1.5" fill="none" stroke="currentColor" />
            </svg>
            <span>中宫</span>
            <em>寄宫定盘</em>
          </div>
        </template>

        <template v-else>
          <div class="qm-palace-head">
            <div class="qm-palace-name">
              <strong>{{ cell.palace }}</strong>
              <span>{{ palaceNumber[cell.palace] }}</span>
            </div>
            <div class="qm-palace-meta">
              <span>{{ palaceDirection[cell.palace] }}</span>
              <span>{{ palaceWuxingZh[cell.palace] }}</span>
            </div>
          </div>

          <div class="qm-symbol-row top">
            <span class="qm-layer-label">神</span>
            <strong class="qm-god">{{ cell.god || '—' }}</strong>
            <span v-if="dutyBadge(cell.palace)" class="qm-duty-badge">{{ dutyBadge(cell.palace) }}</span>
          </div>

          <div class="qm-main-layer">
            <div :class="['qm-door-token', doorClass(cell.door)]">
              <span>门</span>
              <strong>{{ cell.door || '—' }}</strong>
            </div>
            <div class="qm-star-token">
              <span>星</span>
              <strong>{{ cell.star || '—' }}</strong>
            </div>
          </div>

          <div class="qm-stem-stack">
            <div>
              <span>天盘</span>
              <strong>{{ cell.guest_gan || '—' }}</strong>
            </div>
            <div>
              <span>地盘</span>
              <strong>{{ cell.host_gan || '—' }}</strong>
            </div>
          </div>
        </template>
      </article>
    </section>

    <div class="qm-legend">
      <span><i class="qm-dot duty"></i>值符/值使宫</span>
      <span><i class="qm-dot door"></i>八门为行动入口</span>
      <span><i class="qm-dot star"></i>九星为天时性质</span>
      <span><i class="qm-dot stem"></i>天盘干 / 地盘干看组合</span>
    </div>

    <div class="qm-actions">
      <button type="button" class="qm-copy-btn" @click="copyMarkdown">
        {{ copied ? '已复制' : '复制命盘' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { QimenCell, QimenChartPayload } from '../types/chat'

const props = defineProps<{ data: QimenChartPayload }>()
const copied = ref(false)

const rotatingEightWarnings = ['中门', '太常', '勾陈', '朱雀']

const rotatingEightWarning = computed(() => rotatingEightWarningFor(props.data))

const gridLayoutPalaces = ['巽', '离', '坤', '震', '中', '兑', '艮', '坎', '乾']

const gridCells = computed(() => {
  const raw = props.data?.cells || []
  const map = new Map<string, QimenCell>()
  for (const c of raw) {
    map.set(c.palace, c)
  }

  return gridLayoutPalaces.map((name) => {
    const cell = map.get(name)
    if (cell) {
      return {
        ...cell,
        isCenterDummy: false,
      }
    }
    return {
      palace: name,
      god: '',
      star: '',
      door: '',
      guest_gan: '',
      host_gan: '',
      isCenterDummy: true,
    }
  })
})

const dutyStarPalace = computed(() => props.data?.duty_star_palace || props.data?.duty_palace || '')
const dutyDoorPalace = computed(() => props.data?.duty_door_palace || props.data?.duty_palace || '')

const palaceWuxingZh: Record<string, string> = {
  坎: '水',
  坤: '土',
  震: '木',
  巽: '木',
  中: '土',
  乾: '金',
  兑: '金',
  艮: '土',
  离: '火',
}

const palaceWuxing: Record<string, 'wood' | 'fire' | 'earth' | 'metal' | 'water'> = {
  坎: 'water',
  坤: 'earth',
  震: 'wood',
  巽: 'wood',
  中: 'earth',
  乾: 'metal',
  兑: 'metal',
  艮: 'earth',
  离: 'fire',
}

const palaceNumber: Record<string, string> = {
  坎: '一',
  坤: '二',
  震: '三',
  巽: '四',
  中: '五',
  乾: '六',
  兑: '七',
  艮: '八',
  离: '九',
}

const palaceDirection: Record<string, string> = {
  坎: '北',
  坤: '西南',
  震: '东',
  巽: '东南',
  中: '中',
  乾: '西北',
  兑: '西',
  艮: '东北',
  离: '南',
}

const displayOrder: Record<string, number> = {
  中: 0,
  坎: 1,
  离: 1,
  震: 1,
  兑: 1,
  坤: 2,
  乾: 2,
  艮: 2,
  巽: 2,
}

// palaceElement returns the five-element class for each palace.
function palaceElement(palace: string) {
  return palaceWuxing[palace] || 'earth'
}

// cellDelay keeps the board reveal centered instead of strictly row-based.
function cellDelay(palace: string) {
  return (displayOrder[palace] ?? 2) * 48
}

// doorClass gives the Eight Gates quick visual tone without changing facts.
function doorClass(door?: string) {
  if (!door) return 'door-neutral'
  if (door.includes('开') || door.includes('休') || door.includes('生')) return 'door-open'
  if (door.includes('死') || door.includes('惊') || door.includes('伤')) return 'door-tense'
  return 'door-neutral'
}

// isDutyCell marks value-star and value-door palaces separately when available.
function isDutyCell(palace: string) {
  return palace !== '中' && (palace === dutyStarPalace.value || palace === dutyDoorPalace.value)
}

// dutyBadge labels whether the palace holds the value star, value door, or both.
function dutyBadge(palace: string) {
  const hasStar = palace === dutyStarPalace.value
  const hasDoor = palace === dutyDoorPalace.value
  if (hasStar && hasDoor) return '符使'
  if (hasStar) return '符'
  if (hasDoor) return '使'
  return ''
}

// copyMarkdown copies the complete Qi Men board as readable Markdown.
async function copyMarkdown() {
  await navigator.clipboard.writeText(formatQimenMarkdown(props.data))
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 2000)
}

// formatQimenMarkdown keeps the copied board in the visible nine-palace order.
function formatQimenMarkdown(data: QimenChartPayload): string {
  const cellMap = new Map<string, QimenCell>()
  for (const cell of data.cells || []) cellMap.set(cell.palace, cell)

  const lines = ['# 奇门遁甲命盘', '', '## 基本信息']
  lines.push('- 局数：' + field(data.ju_text))
  lines.push('- 问事目的：' + field(data.purpose))
  lines.push('- Case ID：' + field(data.case_id))
  lines.push('- 资产归属：' + ownerRefText(data.owner_ref))
  lines.push('- 值符值使：' + field(data.duty_text))
  lines.push('- 盘式口径：' + field(data.pan_schema))
  lines.push('- 符号体系：' + field(data.symbol_system))
  lines.push('- 起局来源：' + field(data.time_source))
  lines.push('- 值符宫：' + field(data.duty_star_palace || data.duty_palace))
  lines.push('- 值使宫：' + field(data.duty_door_palace || data.duty_palace))
  lines.push('- 起局时间：' + field(data.question_time))
  const warning = rotatingEightWarningFor(data)
  if (warning) lines.push('- 异常警告：' + warning)
  lines.push('', '## 九宫')

  for (const palace of gridLayoutPalaces) {
    const cell = cellMap.get(palace) || { palace }
    lines.push('### ' + palace + '宫（' + palaceDirection[palace] + ' · ' + palaceWuxingZh[palace] + '）')
    if (palace === (data.duty_star_palace || data.duty_palace)) lines.push('- 标记：值符所在宫')
    if (palace === (data.duty_door_palace || data.duty_palace)) lines.push('- 标记：值使所在宫')
    lines.push('- 神：' + field(cell.god))
    lines.push('- 门：' + field(cell.door))
    lines.push('- 星：' + field(cell.star))
    lines.push('- 天盘干：' + field(cell.guest_gan))
    lines.push('- 地盘干：' + field(cell.host_gan))
  }
  return lines.join('\n')
}

// rotatingEightWarningFor reports only the payload/schema contract violation.
function rotatingEightWarningFor(data: QimenChartPayload): string {
  if (data.pan_schema !== 'rotating_8') return ''
  const serialized = JSON.stringify(data)
  const found = rotatingEightWarnings.filter((term) => serialized.includes(term))
  if (!found.length) return ''
  return `盘式合同异常：rotating_8 payload 出现${found.map((term) => `“${term}”`).join('、')}，请核对后端起局结果。`
}

// ownerRefText keeps the Case ownership contract visible in copied Markdown.
function ownerRefText(owner: unknown): string {
  if (!owner || typeof owner !== 'object') return field(owner)
  const ref = owner as { kind?: unknown; id?: unknown }
  return `${field(ref.kind)}/${field(ref.id)}`
}

// field normalizes absent copied fields to an em dash.
function field(value: unknown): string {
  if (value === undefined || value === null || value === '') return '—'
  return String(value)
}
</script>

<style scoped>
.qimen-card {
  max-width: 100%;
  padding: 18px;
  border: 1px solid rgba(184, 149, 106, 0.18);
  border-radius: 20px;
  background:
    radial-gradient(circle at top, rgba(30, 39, 49, 0.98), rgba(17, 22, 31, 0.99)),
    linear-gradient(180deg, rgba(184, 149, 106, 0.08), transparent 28%);
  color: #f3ecdf;
  text-align: left;
}

.qm-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.qm-kicker {
  color: rgba(243, 236, 223, 0.68);
  font-size: 11px;
  letter-spacing: 0.16em;
}

.qm-title {
  margin-top: 2px;
  font-family: var(--serif);
  color: #f0ddba;
  font-size: 22px;
  font-weight: 800;
}

.qm-info {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.qm-info-chip {
  display: inline-flex;
  align-items: center;
  padding: 5px 10px;
  border: 1px solid rgba(240, 221, 186, 0.12);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  color: rgba(243, 236, 223, 0.82);
  font-size: 11px;
}

.qm-info-chip.accent {
  border-color: rgba(184, 149, 106, 0.28);
  background: rgba(184, 149, 106, 0.14);
  color: #f3e0b7;
}

.qm-case-meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 14px;
  margin-bottom: 14px;
  padding: 10px 12px;
  border-left: 2px solid rgba(184, 149, 106, 0.42);
  background: rgba(255, 255, 255, 0.035);
}

.qm-case-meta div {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  min-width: 0;
  font-size: 11px;
}

.qm-case-meta span {
  color: rgba(243, 236, 223, 0.56);
}

.qm-case-meta strong {
  overflow-wrap: anywhere;
  color: rgba(243, 236, 223, 0.9);
  font-weight: 600;
}

.qm-warning {
  margin: 0 0 14px;
  padding: 9px 12px;
  border: 1px solid rgba(244, 151, 102, 0.52);
  background: rgba(112, 48, 34, 0.28);
  color: #ffd2b8;
  font-size: 12px;
  line-height: 1.5;
}

.qm-board {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 14px;
  border: 1px solid rgba(240, 221, 186, 0.12);
  border-radius: 18px;
  background:
    linear-gradient(90deg, rgba(240, 221, 186, 0.04) 1px, transparent 1px),
    linear-gradient(0deg, rgba(240, 221, 186, 0.04) 1px, transparent 1px),
    radial-gradient(circle at center, rgba(184, 149, 106, 0.08), transparent 68%);
  background-size: 33.33% 33.33%, 33.33% 33.33%, auto;
}

.qm-cell {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 9px;
  min-height: 154px;
  padding: 12px;
  border: 1px solid rgba(240, 221, 186, 0.12);
  border-radius: 16px;
  background: rgba(248, 244, 235, 0.05);
  opacity: 0;
  overflow: hidden;
  animation: qm-cell-in 0.3s cubic-bezier(0.22, 0.61, 0.36, 1) forwards;
}

@keyframes qm-cell-in {
  from { opacity: 0; transform: scale(0.95); }
  to { opacity: 1; transform: scale(1); }
}

.qm-cell::before {
  content: "";
  position: absolute;
  inset: 0;
  border-left: 4px solid transparent;
  pointer-events: none;
}

.qm-water::before { border-left-color: rgba(107, 138, 168, 0.75); }
.qm-wood::before { border-left-color: rgba(122, 158, 126, 0.75); }
.qm-fire::before { border-left-color: rgba(196, 122, 106, 0.8); }
.qm-earth::before { border-left-color: rgba(184, 149, 106, 0.78); }
.qm-metal::before { border-left-color: rgba(196, 169, 106, 0.78); }

.qm-cell.is-duty {
  border-color: #d8b97d;
  background: rgba(184, 149, 106, 0.12);
  box-shadow: 0 0 0 1px rgba(216, 185, 125, 0.52), 0 0 22px rgba(184, 149, 106, 0.2);
}

.qm-cell.is-center {
  justify-content: center;
  border-style: dashed;
  background: rgba(248, 244, 235, 0.035);
}

.qm-center-mark {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  color: rgba(243, 236, 223, 0.7);
  text-align: center;
}

.qm-taiji {
  width: 42px;
  height: 42px;
  color: #e2c48d;
}

.qm-center-mark span {
  color: #f0ddba;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.qm-center-mark em {
  color: rgba(243, 236, 223, 0.5);
  font-size: 10px;
  font-style: normal;
}

.qm-palace-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding-bottom: 7px;
  border-bottom: 1px solid rgba(240, 221, 186, 0.12);
}

.qm-palace-name {
  display: inline-flex;
  align-items: baseline;
  gap: 5px;
}

.qm-palace-name strong {
  font-family: var(--serif);
  color: #fff3d8;
  font-size: 20px;
}

.qm-palace-name span,
.qm-palace-meta {
  color: rgba(243, 236, 223, 0.58);
  font-size: 10px;
}

.qm-palace-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.qm-symbol-row {
  display: flex;
  align-items: center;
  gap: 7px;
}

.qm-layer-label {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(243, 236, 223, 0.54);
  font-size: 10px;
}

.qm-god {
  color: #f2d8a1;
  font-size: 13px;
  letter-spacing: 0.12em;
}

.qm-duty-badge {
  margin-left: auto;
  padding: 2px 6px;
  border-radius: 999px;
  background: rgba(216, 185, 125, 0.16);
  color: #f0d49b;
  font-size: 10px;
  font-weight: 800;
}

.qm-main-layer {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.qm-door-token,
.qm-star-token {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  padding: 9px 8px;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.18);
  border: 1px solid rgba(240, 221, 186, 0.1);
}

.qm-door-token span,
.qm-star-token span,
.qm-stem-stack span {
  color: rgba(243, 236, 223, 0.54);
  font-size: 10px;
}

.qm-door-token strong,
.qm-star-token strong {
  color: #fff8eb;
  font-size: 15px;
}

.qm-door-token.door-open {
  border-color: rgba(122, 158, 126, 0.28);
  background: rgba(122, 158, 126, 0.12);
}

.qm-door-token.door-tense {
  border-color: rgba(196, 122, 106, 0.28);
  background: rgba(196, 122, 106, 0.12);
}

.qm-door-token.door-neutral {
  border-color: rgba(196, 169, 106, 0.22);
  background: rgba(196, 169, 106, 0.08);
}

.qm-star-token {
  background: rgba(107, 138, 168, 0.1);
}

.qm-stem-stack {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 7px;
  margin-top: auto;
}

.qm-stem-stack div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 6px;
  padding: 5px 7px;
  border: 1px solid rgba(240, 221, 186, 0.1);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.04);
}

.qm-stem-stack strong {
  color: #f4ead4;
  font-family: var(--mono);
  font-size: 13px;
}

.qm-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 14px;
  margin-top: 14px;
  color: rgba(243, 236, 223, 0.66);
  font-size: 11px;
}

.qm-legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.qm-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
}

.qm-dot.duty { background: #d8b97d; }
.qm-dot.door { background: #7a9e7e; }
.qm-dot.star { background: #6b8aa8; }
.qm-dot.stem { background: #c4a96a; }

.qm-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 18px;
}

.qm-copy-btn {
  border: 1px solid rgba(240, 221, 186, 0.16);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(243, 236, 223, 0.82);
  border-radius: 999px;
  padding: 8px 14px;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}

.qm-copy-btn:hover {
  background: rgba(184, 149, 106, 0.14);
  border-color: rgba(184, 149, 106, 0.42);
  color: #f3e0b7;
}

@media (max-width: 760px) {
  .qm-header {
    flex-direction: column;
  }

  .qm-info {
    justify-content: flex-start;
  }

  .qm-case-meta {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .qm-board {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 520px) {
  .qimen-card {
    padding: 14px;
    border-radius: 16px;
  }

  .qm-main-layer,
  .qm-stem-stack {
    grid-template-columns: 1fr;
  }

  .qm-case-meta {
    grid-template-columns: 1fr;
  }
}
</style>
