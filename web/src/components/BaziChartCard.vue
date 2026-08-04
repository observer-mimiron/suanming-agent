<!--
  BaziChartCard renders the user-facing BaZi chart surface.
  It owns visual grouping, color coding, and copy affordances only; deterministic
  chart facts still come from the backend payload and markdown formatter.
-->
<template>
  <div class="bazi-card">
    <div class="bz-hero">
      <div>
        <div class="bz-kicker">八字命盘</div>
        <div class="bz-title-row">
          <span class="bz-title">四柱五行</span>
          <span
            v-if="dayGan"
            :class="['bz-daymaster', elementTokenClass(stemElement(dayGan))]"
          >
            日主 {{ dayGan }} · {{ dayGanWuxing || stemElement(dayGan) }}
          </span>
        </div>
      </div>

      <div class="bz-meta-grid">
        <span v-if="pillarsSummary" class="bz-meta">四柱 {{ pillarsSummary }}</span>
        <span v-if="lunarDate" class="bz-meta">农历 {{ lunarDate }}</span>
      </div>
    </div>

    <div class="bz-legend" aria-label="五行颜色图例">
      <span
        v-for="item in elementLegend"
        :key="item.name"
        :class="['bz-legend-item', elementTokenClass(item.name)]"
      >
        <span class="bz-legend-dot"></span>
        {{ item.name }}
      </span>
      <span class="bz-yinyang-note">阳为实框 · 阴为虚框</span>
    </div>

    <div class="bz-pillars" aria-label="四柱命盘">
      <article
        v-for="(p, i) in pillars"
        :key="p.name"
        :class="['bz-pillar', { 'is-day-pillar': isDayPillar(p) }]"
        :style="{ animationDelay: i * 60 + 'ms' }"
      >
        <header class="bz-pillar-head">
          <span>{{ p.name }}</span>
          <strong v-if="isDayPillar(p)">日柱</strong>
        </header>

        <div class="bz-char-stack">
          <div class="bz-char-row">
            <span class="bz-layer-label">天干</span>
            <span
              :class="[
                'bz-char-token',
                elementTokenClass(stemElement(p.stem)),
                polarityClass(stemPolarity(p.stem)),
                { 'is-daymaster': isDayPillar(p) },
              ]"
            >
              {{ p.stem || '—' }}
            </span>
            <span class="bz-char-meta">
              <span>{{ stemElement(p.stem) || '—' }} · {{ stemPolarity(p.stem) || '—' }}</span>
              <em v-if="p.shiShen" class="bz-god-mark">{{ p.shiShen }}</em>
            </span>
          </div>

          <div class="bz-char-row">
            <span class="bz-layer-label">地支</span>
            <span
              :class="[
                'bz-char-token',
                elementTokenClass(branchElement(p.branch)),
                polarityClass(branchPolarity(p.branch)),
              ]"
            >
              {{ p.branch || '—' }}
            </span>
            <span class="bz-char-meta">
              <span>{{ branchElement(p.branch) || '—' }} · {{ branchPolarity(p.branch) || '—' }}</span>
              <em v-if="branchMainTenGod(p)" class="bz-god-mark">{{ branchMainTenGod(p) }}</em>
            </span>
          </div>
        </div>

        <div class="bz-pillar-main">
          <span class="bz-nayin">{{ p.naYin || '纳音未返回' }}</span>
        </div>

        <div v-if="p.shensha?.length" class="bz-shensha">
          <span v-for="ss in p.shensha" :key="ss.name" :class="['bz-ss-badge', ss.tone]">
            {{ ss.name }}
          </span>
        </div>
      </article>
    </div>

    <section class="bz-readout">
      <div class="bz-panel">
        <div class="bz-panel-title">五行分布</div>
        <div class="bz-wuxing">
          <div v-for="item in wuxingEntries" :key="item.name" class="bz-wx-row">
            <span :class="['bz-wx-name', elementTokenClass(item.name)]">{{ item.name }}</span>
            <div class="bz-wx-bar">
              <div
                class="bz-wx-fill"
                :class="'wx-' + elementKey(item.name)"
                :style="{ width: wuxingWidth(item.value) }"
              ></div>
            </div>
            <span class="bz-wx-count">{{ item.value }}</span>
          </div>
        </div>
      </div>

      <div class="bz-panel">
        <div class="bz-panel-title">宫位补充</div>
        <div class="bz-extra">
          <span>命宫 <strong>{{ mingGong || '—' }}</strong><em>{{ mingGongNaYin }}</em></span>
          <span>身宫 <strong>{{ shenGong || '—' }}</strong><em>{{ shenGongNaYin }}</em></span>
          <span>胎元 <strong>{{ taiYuan || '—' }}</strong><em>{{ taiYuanNaYin }}</em></span>
        </div>
      </div>
    </section>

    <table class="bz-table" aria-label="四柱明细">
      <thead>
        <tr>
          <th>柱</th>
          <th>天干</th>
          <th>地支</th>
          <th>天干十神</th>
          <th>地支十神</th>
          <th>藏干</th>
          <th>纳音</th>
          <th>空亡</th>
          <th>地势</th>
          <th>旬</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in pillars" :key="'dt-' + p.name">
          <td>{{ p.name }}</td>
          <td>
            <span :class="['bz-mini-token', elementTokenClass(stemElement(p.stem))]">{{ p.stem || '—' }}</span>
          </td>
          <td>
            <span :class="['bz-mini-token', elementTokenClass(branchElement(p.branch))]">{{ p.branch || '—' }}</span>
          </td>
          <td class="shishen">{{ p.shiShen || '—' }}</td>
          <td class="branch-main">{{ branchMainTenGod(p) || '—' }}</td>
          <td class="hidden-stems">
            <span
              v-for="item in hiddenStemPairs(p)"
              :key="'hidden-' + p.name + item.stem"
              class="bz-hidden-pair"
            >
              <span :class="['bz-mini-token', elementTokenClass(stemElement(item.stem))]">
                {{ item.stem }}
              </span>
              <em>{{ item.tenGod || '—' }}</em>
            </span>
            <span v-if="!hiddenStemPairs(p).length" class="dim">—</span>
          </td>
          <td class="dim">{{ p.naYin || '—' }}</td>
          <td class="dim">{{ p.xunKong || '—' }}</td>
          <td class="dim">{{ p.diShi || '—' }}</td>
          <td class="dim">{{ p.xun || '—' }}</td>
        </tr>
      </tbody>
    </table>

    <div v-if="dayun.length" class="bz-dayun-timeline" aria-label="大运时间轴">
      <div class="bz-timeline-line"></div>
      <div class="bz-timeline-nodes">
        <div
          v-for="(d, i) in dayun"
          :key="i"
          class="bz-timeline-node"
          :class="{ active: i === currentDayunIdx }"
        >
          <span class="bz-node-age">{{ d.startAge }}-{{ d.endAge }}岁</span>
          <div class="bz-node-dot">
            <span class="bz-node-dot-inner"></span>
          </div>
          <span class="bz-node-ganzhi">
            <span :class="['bz-node-char', elementTextClass(stemElement(dayunStem(d.ganZhi)))]">
              {{ dayunStem(d.ganZhi) }}
            </span>
            <span :class="['bz-node-char', elementTextClass(branchElement(dayunBranch(d.ganZhi)))]">
              {{ dayunBranch(d.ganZhi) }}
            </span>
          </span>
        </div>
      </div>
    </div>

    <div class="bz-actions">
      <button
        type="button"
        class="bz-copy-btn"
        data-testid="copy-bazi-markdown"
        @click="copyMarkdown"
      >
        {{ copied ? '已复制' : '复制命盘' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { formatBaziChartMarkdown } from '../utils/baziMarkdown'

interface Pillar {
  name: string
  stem: string
  branch: string
  shiShen: string
  naYin: string
  xunKong?: string
  diShi?: string
  xun?: string
  hideGan?: string[]
  subShiShen?: string[]
  shensha?: { name: string; tone: string }[]
}

interface HiddenStemPair {
  stem: string
  tenGod: string
}

interface Dayun {
  startAge: number
  endAge: number
  ganZhi: string
}

const props = defineProps<{ data: any }>()
const pillars = computed<Pillar[]>(() => props.data?.pillars || [])
const dayGan = computed(() => props.data?.dayGan || '')
const dayGanWuxing = computed(() => props.data?.dayGanWuxing || '')
const lunarDate = computed(() => props.data?.lunarDate || '')
const pillarsSummary = computed(() =>
  pillars.value
    .map((pillar) => (pillar.stem || '') + (pillar.branch || ''))
    .filter(Boolean)
    .join(' · '),
)
const wuxing = computed<Record<string, number>>(() => props.data?.wuxing || {})
const dayun = computed<Dayun[]>(() => props.data?.dayun || [])
const mingGong = computed(() => props.data?.mingGong || '')
const mingGongNaYin = computed(() => props.data?.mingGongNaYin || '')
const shenGong = computed(() => props.data?.shenGong || '')
const shenGongNaYin = computed(() => props.data?.shenGongNaYin || '')
const taiYuan = computed(() => props.data?.taiYuan || '')
const taiYuanNaYin = computed(() => props.data?.taiYuanNaYin || '')
const copied = ref(false)

const elementLegend = [
  { name: '木' },
  { name: '火' },
  { name: '土' },
  { name: '金' },
  { name: '水' },
]

const stemWuxing: Record<string, string> = {
  甲: '木',
  乙: '木',
  丙: '火',
  丁: '火',
  戊: '土',
  己: '土',
  庚: '金',
  辛: '金',
  壬: '水',
  癸: '水',
}

const branchWuxing: Record<string, string> = {
  寅: '木',
  卯: '木',
  巳: '火',
  午: '火',
  辰: '土',
  戌: '土',
  丑: '土',
  未: '土',
  申: '金',
  酉: '金',
  亥: '水',
  子: '水',
}

const yangStems = new Set(['甲', '丙', '戊', '庚', '壬'])
const yangBranches = new Set(['子', '寅', '辰', '午', '申', '戌'])

const wuxingEntries = computed(() =>
  elementLegend.map((item) => ({
    name: item.name,
    value: Number(wuxing.value[item.name] || 0),
  })),
)

const maxWuxing = computed(() => Math.max(1, ...wuxingEntries.value.map((item) => item.value)))

const currentDayunIdx = computed(() => {
  const liunianCurrent = props.data?.liunian?.current_dayun
  if (liunianCurrent?.ganZhi) {
    return dayun.value.findIndex(
      (d: any) =>
        d.ganZhi === liunianCurrent.ganZhi &&
        d.startAge === liunianCurrent.startAge &&
        d.endAge === liunianCurrent.endAge,
    )
  }

  if (!props.data?.birthday) return -1
  const birthYear = parseInt(props.data.birthday) || 0
  if (!birthYear) return -1
  const referenceYear = Number(props.data?.liunian?.liunian_year) || new Date().getFullYear()
  const age = referenceYear - birthYear + 1
  return dayun.value.findIndex((d: any) => age >= d.startAge && age <= d.endAge)
})

// stemElement returns the five-element category for a heavenly stem.
function stemElement(stem?: string): string {
  return stem ? stemWuxing[stem] || '' : ''
}

// branchElement returns the five-element category for an earthly branch.
function branchElement(branch?: string): string {
  return branch ? branchWuxing[branch] || '' : ''
}

// stemPolarity labels a heavenly stem as yin or yang for quick scanning.
function stemPolarity(stem?: string): string {
  if (!stem) return ''
  return yangStems.has(stem) ? '阳' : '阴'
}

// branchPolarity labels an earthly branch as yin or yang for quick scanning.
function branchPolarity(branch?: string): string {
  if (!branch) return ''
  return yangBranches.has(branch) ? '阳' : '阴'
}

// elementKey maps Chinese five elements to CSS suffixes.
function elementKey(element?: string): string {
  const map: Record<string, string> = { 木: 'wood', 火: 'fire', 土: 'earth', 金: 'metal', 水: 'water' }
  return map[element || ''] || 'earth'
}

// elementTokenClass provides the shared class for colored element tokens.
function elementTokenClass(element?: string): string {
  return 'wx-token-' + elementKey(element)
}

// elementTextClass colors compact text where a filled token would be too heavy.
function elementTextClass(element?: string): string {
  return 'wx-text-' + elementKey(element)
}

// polarityClass makes yin visually lighter without changing the element color.
function polarityClass(polarity?: string): string {
  return polarity === '阴' ? 'is-yin' : 'is-yang'
}

// isDayPillar identifies the day pillar whose stem is the chart owner.
function isDayPillar(pillar: Pillar): boolean {
  return pillar.name.includes('日')
}

// branchMainTenGod treats the first hidden stem as the branch's main-qi ten god.
function branchMainTenGod(pillar: Pillar): string {
  return Array.isArray(pillar.subShiShen) ? pillar.subShiShen[0] || '' : ''
}

// hiddenStemPairs pairs hidden stems with their ten-god labels for the table.
function hiddenStemPairs(pillar: Pillar): HiddenStemPair[] {
  const stems = Array.isArray(pillar.hideGan) ? pillar.hideGan : []
  const gods = Array.isArray(pillar.subShiShen) ? pillar.subShiShen : []
  return stems.map((stem, index) => ({ stem, tenGod: gods[index] || '' }))
}

// wuxingWidth scales each five-element bar against the strongest returned value.
function wuxingWidth(value: number): string {
  if (value <= 0) return '0%'
  return Math.max(10, Math.round((value / maxWuxing.value) * 100)) + '%'
}

// dayunStem extracts the heavenly stem from a GanZhi pair.
function dayunStem(ganZhi?: string): string {
  return ganZhi?.slice(0, 1) || ''
}

// dayunBranch extracts the earthly branch from a GanZhi pair.
function dayunBranch(ganZhi?: string): string {
  return ganZhi?.slice(1, 2) || ''
}

// copyMarkdown copies the full chart facts through the existing readable formatter.
async function copyMarkdown() {
  const markdown = formatBaziChartMarkdown(props.data || {})
  await navigator.clipboard.writeText(markdown)
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 2000)
}
</script>

<style scoped>
.bazi-card {
  text-align: left;
  font-size: 13px;
  max-width: 100%;
}

.bz-hero {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--bg-secondary), rgba(255, 255, 255, 0.62));
}

.bz-kicker {
  font-size: 11px;
  color: var(--text-muted);
  letter-spacing: 0.12em;
}

.bz-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
}

.bz-title {
  font-family: var(--serif);
  font-size: 23px;
  font-weight: 700;
  color: var(--text-primary);
}

.bz-daymaster {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 700;
}

.bz-meta-grid {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  min-width: 190px;
}

.bz-meta {
  font-size: 12px;
  color: var(--text-muted);
}

.bz-legend {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.bz-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border-radius: 999px;
  padding: 4px 9px;
  font-size: 11px;
  font-weight: 700;
}

.bz-legend-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}

.bz-yinyang-note {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-muted);
}

.bz-pillars {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.bz-pillar {
  position: relative;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-secondary);
  padding: 12px;
  box-shadow: var(--shadow-card);
  opacity: 0;
  animation: pillar-in 0.34s cubic-bezier(0.22, 0.61, 0.36, 1) forwards;
}

.bz-pillar.is-day-pillar {
  border-color: rgba(184, 149, 106, 0.7);
  box-shadow: 0 0 0 3px var(--accent-bg), var(--shadow-card);
}

.bz-pillar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text-muted);
  font-size: 11px;
  margin-bottom: 10px;
}

.bz-pillar-head strong {
  color: var(--accent-dim);
  font-size: 10px;
  letter-spacing: 0.08em;
}

.bz-char-stack {
  display: grid;
  gap: 8px;
}

.bz-char-row {
  display: grid;
  grid-template-columns: 34px 54px 1fr;
  align-items: center;
  gap: 8px;
}

.bz-layer-label {
  font-size: 11px;
  color: var(--text-muted);
}

.bz-char-token {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border: 2px solid currentColor;
  border-radius: 14px;
  font-family: var(--serif);
  font-size: 30px;
  font-weight: 800;
  line-height: 1;
  box-sizing: border-box;
}

.bz-char-token.is-yin {
  border-style: dashed;
  background-image: repeating-linear-gradient(
    -45deg,
    rgba(255, 255, 255, 0.22) 0,
    rgba(255, 255, 255, 0.22) 3px,
    transparent 3px,
    transparent 7px
  );
}

.bz-char-token.is-daymaster {
  transform: translateY(-1px);
  box-shadow: 0 6px 14px rgba(45, 42, 40, 0.12);
}

.bz-char-meta {
  display: grid;
  gap: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.bz-god-mark {
  display: inline-flex;
  width: max-content;
  border-radius: 999px;
  padding: 2px 7px;
  background: rgba(184, 149, 106, 0.12);
  color: var(--accent-dim);
  font-size: 11px;
  font-style: normal;
  font-weight: 700;
}

.bz-pillar-main {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--border-light);
}

.bz-nayin {
  color: var(--text-muted);
  font-size: 12px;
}

.bz-mini-token {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  border-radius: 8px;
  padding: 0 6px;
  font-size: 12px;
  font-weight: 700;
  box-sizing: border-box;
}

.bz-shensha {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 10px;
}

.bz-ss-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 999px;
  font-weight: 600;
  line-height: 1.2;
}

.bz-ss-badge.good {
  background: var(--accent-bg);
  color: var(--accent-dim);
}

.bz-ss-badge.neutral {
  background: var(--bg-hover);
  color: var(--text-secondary);
}

.bz-ss-badge.bad {
  background: rgba(196, 122, 106, 0.1);
  color: var(--wx-fire);
}

.bz-readout {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 12px;
  margin-bottom: 14px;
}

.bz-panel {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: rgba(250, 248, 245, 0.72);
  padding: 12px;
}

.bz-panel-title {
  margin-bottom: 10px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.bz-wuxing {
  display: grid;
  gap: 7px;
}

.bz-wx-row {
  display: grid;
  grid-template-columns: 42px 1fr 24px;
  align-items: center;
  gap: 9px;
}

.bz-wx-name {
  border-radius: 999px;
  padding: 3px 8px;
  text-align: center;
  font-size: 11px;
  font-weight: 700;
}

.bz-wx-bar {
  height: 8px;
  background: var(--border-light);
  border-radius: 999px;
  overflow: hidden;
}

.bz-wx-fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.55s cubic-bezier(0.22, 0.61, 0.36, 1);
}

.bz-wx-fill.wx-wood { background: var(--wx-wood); }
.bz-wx-fill.wx-fire { background: var(--wx-fire); }
.bz-wx-fill.wx-earth { background: var(--wx-earth); }
.bz-wx-fill.wx-metal { background: var(--wx-metal); }
.bz-wx-fill.wx-water { background: var(--wx-water); }

.bz-wx-count {
  color: var(--text-secondary);
  font-weight: 700;
  text-align: right;
}

.bz-extra {
  display: grid;
  gap: 8px;
  color: var(--text-muted);
  font-size: 12px;
}

.bz-extra span {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.bz-extra strong {
  color: var(--text-secondary);
  font-family: var(--serif);
  font-size: 16px;
}

.bz-extra em {
  color: var(--text-muted);
  font-style: normal;
}

.bz-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  margin-bottom: 12px;
}

.bz-table th {
  text-align: center;
  padding: 8px 4px;
  font-weight: 600;
  color: var(--text-muted);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
  font-size: 11px;
}

.bz-table td {
  text-align: center;
  padding: 7px 4px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-light);
}

.bz-table .shishen {
  color: var(--accent-dim);
  font-weight: 700;
}

.bz-table .branch-main {
  color: var(--accent-dim);
  font-weight: 700;
}

.bz-table .dim {
  color: var(--text-muted);
}

.bz-table .hidden-stems {
  min-width: 132px;
}

.bz-hidden-pair {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 1px 3px;
  white-space: nowrap;
}

.bz-hidden-pair em {
  color: var(--text-secondary);
  font-style: normal;
  font-weight: 600;
}

.bz-dayun-timeline {
  position: relative;
  margin-top: 24px;
  padding: 12px 0 16px;
  overflow-x: auto;
  scrollbar-width: none;
}

.bz-dayun-timeline::-webkit-scrollbar {
  display: none;
}

.bz-timeline-line {
  position: absolute;
  top: 35px;
  left: 4%;
  width: 92%;
  height: 1px;
  background: var(--border);
  z-index: 1;
}

.bz-timeline-nodes {
  display: flex;
  justify-content: space-between;
  position: relative;
  z-index: 2;
  min-width: 540px;
  padding: 0 8px;
}

.bz-timeline-node {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  cursor: default;
  transition: transform 0.2s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.bz-timeline-node:hover {
  transform: translateY(-2px);
}

.bz-node-age {
  margin-bottom: 6px;
  color: var(--text-muted);
  font-size: 9px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.bz-node-dot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
  margin-bottom: 9px;
  border: 1.5px solid var(--border);
  border-radius: 50%;
  background: var(--bg-secondary);
}

.bz-node-dot-inner {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--border);
}

.bz-node-ganzhi {
  display: inline-flex;
  gap: 1px;
  font-family: var(--serif);
  font-size: 15px;
  font-weight: 800;
  line-height: 1.2;
}

.bz-timeline-node.active .bz-node-age {
  color: var(--accent-dim);
}

.bz-timeline-node.active .bz-node-dot {
  border-color: var(--accent);
  background: var(--accent-bg);
  box-shadow: 0 0 8px rgba(184, 149, 106, 0.42);
}

.bz-timeline-node.active .bz-node-dot-inner {
  background: var(--accent);
}

.bz-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 18px;
}

.bz-copy-btn {
  border: 1px solid var(--border);
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border-radius: 999px;
  padding: 8px 14px;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}

.bz-copy-btn:hover {
  background: var(--bg-hover);
  border-color: var(--accent);
  color: var(--text-primary);
}

.wx-token-wood {
  color: #315f3d;
  background: rgba(122, 158, 126, 0.18);
}

.wx-token-fire {
  color: #8f382e;
  background: rgba(196, 122, 106, 0.18);
}

.wx-token-earth {
  color: #7a5d2d;
  background: rgba(184, 149, 106, 0.18);
}

.wx-token-metal {
  color: #7b6726;
  background: rgba(196, 169, 106, 0.2);
}

.wx-token-water {
  color: #315a7a;
  background: rgba(107, 138, 168, 0.18);
}

.wx-text-wood { color: var(--wx-wood); }
.wx-text-fire { color: var(--wx-fire); }
.wx-text-earth { color: var(--wx-earth); }
.wx-text-metal { color: var(--wx-metal); }
.wx-text-water { color: var(--wx-water); }

@keyframes pillar-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 860px) {
  .bz-hero,
  .bz-readout {
    grid-template-columns: 1fr;
  }

  .bz-hero {
    flex-direction: column;
  }

  .bz-meta-grid {
    align-items: flex-start;
    min-width: 0;
  }

  .bz-yinyang-note {
    margin-left: 0;
    width: 100%;
  }

  .bz-pillars {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .bz-pillars {
    grid-template-columns: 1fr;
  }

  .bz-char-row {
    grid-template-columns: 34px 50px 1fr;
  }

  .bz-char-token {
    width: 48px;
    height: 48px;
    font-size: 27px;
  }

  .bz-table {
    display: block;
    overflow-x: auto;
    white-space: nowrap;
  }
}
</style>
