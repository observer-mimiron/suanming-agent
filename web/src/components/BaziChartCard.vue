<template>
  <div class="bazi-card">
    <!-- Header -->
    <div class="bz-header">
      <span class="bz-meta">
        日主 <strong>{{ dayGan }}</strong>（{{ dayGanWuxing }}）
      </span>
      <span v-if="pillarsSummary" class="bz-meta">四柱 {{ pillarsSummary }}</span>
      <span v-if="lunarDate" class="bz-meta">农历 {{ lunarDate }}</span>
    </div>

    <!-- Four Pillars as sprite cards -->
    <div class="bz-pillars">
      <div
        v-for="(p, i) in pillars"
        :key="p.name"
        class="bz-pillar"
        :style="{ animationDelay: i * 80 + 'ms' }"
        @mousemove="handleTilt"
        @mouseleave="resetTilt"
      >
        <!-- 四角几何包角 -->
        <span class="corner-dot tl"></span>
        <span class="corner-dot tr"></span>
        <span class="corner-dot bl"></span>
        <span class="corner-dot br"></span>

        <!-- 纸牌内边双线框 -->
        <div class="bz-pillar-inner">
          <div class="bz-p-label">{{ p.name }}</div>
          <ElementSprite :element="pillarElement(p)" :size="44" class="bz-p-sprite" />
          <div class="bz-p-ganzhi">{{ p.stem }}{{ p.branch }}</div>
          <div class="bz-p-shishen">{{ p.shiShen }}</div>
          <div class="bz-p-nayin">{{ p.naYin }}</div>
          <!-- 神煞小徽章 -->
          <div v-if="p.shensha && p.shensha.length" class="bz-p-shensha">
            <span v-for="ss in p.shensha" :key="ss.name" :class="['bz-ss-badge', ss.tone]">
              {{ ss.name }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Detail table -->
    <table class="bz-table">
      <thead>
        <tr><th>柱</th><th>天干</th><th>地支</th><th>十神</th><th>纳音</th><th>空亡</th><th>地势</th><th>旬</th></tr>
      </thead>
      <tbody>
        <tr v-for="p in pillars" :key="'dt-'+p.name">
          <td>{{ p.name }}</td>
          <td class="strong">{{ p.stem }}</td>
          <td class="strong">{{ p.branch }}</td>
          <td class="shishen">{{ p.shiShen }}</td>
          <td class="dim">{{ p.naYin }}</td>
          <td class="dim">{{ p.xunKong }}</td>
          <td class="dim">{{ p.diShi }}</td>
          <td class="dim">{{ p.xun }}</td>
        </tr>
      </tbody>
    </table>

    <!-- Hidden stems -->
    <div class="bz-hidegan">
      <span class="bz-hg-label">藏干</span>
      <span v-for="p in pillars" :key="'hg-'+p.name" class="bz-hg-item">
        {{ p.name }} <strong>{{ (p.hideGan || []).join(' ') }}</strong>
      </span>
    </div>

    <div class="bz-hidegan">
      <span class="bz-hg-label">副星</span>
      <span v-for="p in pillars" :key="'ss-'+p.name" class="bz-hg-item">
        {{ p.name }} <strong>{{ (p.subShiShen || []).join(' ') || '—' }}</strong>
      </span>
    </div>

    <!-- Ming/Shen/Tai -->
    <div class="bz-extra">
      <span>命宫 <strong>{{ mingGong }}</strong><span class="dim"> {{ mingGongNaYin }}</span></span>
      <span>身宫 <strong>{{ shenGong }}</strong><span class="dim"> {{ shenGongNaYin }}</span></span>
      <span>胎元 <strong>{{ taiYuan }}</strong><span class="dim"> {{ taiYuanNaYin }}</span></span>
    </div>

    <!-- Wuxing bars -->
    <div class="bz-wuxing">
      <div v-for="(v, k) in wuxing" :key="k" class="bz-wx-row">
        <span class="bz-wx-label">{{ k }}</span>
        <div class="bz-wx-bar"><div class="bz-wx-fill" :class="'wx-' + elementKey(k)" :style="{ width: (v / 8 * 100) + '%' }"></div></div>
        <span class="bz-wx-count">{{ v }}</span>
      </div>
    </div>

    <!-- Fate Timeline for Dayun -->
    <div class="bz-dayun-timeline">
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
          <span class="bz-node-ganzhi">{{ d.ganZhi }}</span>
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
import ElementSprite from './sprites/ElementSprite.vue'
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
    .map((pillar) => `${pillar.stem || ''}${pillar.branch || ''}`.trim())
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

function handleTilt(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const xc = rect.width / 2
  const yc = rect.height / 2
  const rotateY = ((x - xc) / xc) * 8
  const rotateX = -((y - yc) / yc) * 8
  el.style.transform = `perspective(600px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) translateY(-4px)`
  el.style.boxShadow = `0 10px 20px rgba(0, 0, 0, 0.08), 0 0 12px var(--accent-bg)`
}

function resetTilt(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  el.style.transform = `perspective(600px) rotateX(0deg) rotateY(0deg) translateY(0)`
  el.style.boxShadow = ``
}

// 天干→五行
const stemWuxing: Record<string, string> = {
  '甲':'木','乙':'木','丙':'火','丁':'火','戊':'土',
  '己':'土','庚':'金','辛':'金','壬':'水','癸':'水',
}
// 地支→五行
const branchWuxing: Record<string, string> = {
  '寅':'木','卯':'木','巳':'火','午':'火',
  '辰':'土','戌':'土','丑':'土','未':'土',
  '申':'金','酉':'金','亥':'水','子':'水',
}
function pillarElement(p: any): 'wood' | 'fire' | 'earth' | 'metal' | 'water' {
  const wx = stemWuxing[p.stem] || branchWuxing[p.branch] || '土'
  if (wx === '木') return 'wood'
  if (wx === '火') return 'fire'
  if (wx === '土') return 'earth'
  if (wx === '金') return 'metal'
  if (wx === '水') return 'water'
  return 'earth'
}

function elementKey(k: string): string {
  const map: Record<string,string> = { '木':'wood','火':'fire','土':'earth','金':'metal','水':'water' }
  return map[k] || 'earth'
}

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
.bz-header {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-bottom: 16px;
  padding: 0 4px;
}
.bz-meta { font-size: 12px; color: var(--text-muted); }
.bz-meta strong { color: var(--text-secondary); }

.bz-pillars {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}
.bz-pillar {
  position: relative;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 4px;
  text-align: center;
  opacity: 0;
  overflow: hidden;
  transform-style: preserve-3d;
  animation: pillar-in 0.4s cubic-bezier(0.22,0.61,0.36,1) forwards;
  transition: transform 0.2s cubic-bezier(0.25, 0.8, 0.25, 1), box-shadow 0.2s ease;
}
.bz-pillar::after {
  content: "";
  position: absolute;
  top: -50%; left: -50%;
  width: 200%; height: 200%;
  background: linear-gradient(
    115deg,
    transparent 40%,
    rgba(212, 175, 55, 0.08) 48%,
    rgba(255, 255, 255, 0.15) 50%,
    rgba(212, 175, 55, 0.08) 52%,
    transparent 60%
  );
  transform: translate(-30%, -30%);
  pointer-events: none;
  opacity: 0;
}
.bz-pillar:hover::after {
  transform: translate(15%, 15%);
  transition: transform 0.8s cubic-bezier(0.19, 1, 0.22, 1);
  opacity: 1;
}
.bz-pillar-inner {
  border: 1px dashed var(--border-light);
  border-radius: calc(var(--radius-md) - 3px);
  padding: 18px 6px 12px;
  height: 100%;
  box-sizing: border-box;
}
/* 几何直角 L 包角 (现代塔罗风格) */
.corner-dot {
  position: absolute;
  width: 6px;
  height: 6px;
  pointer-events: none;
  transition: opacity 0.25s ease;
  opacity: 0.3;
}
.corner-dot.tl {
  top: 6px; left: 6px;
  border-top: 1px solid var(--accent);
  border-left: 1px solid var(--accent);
}
.corner-dot.tr {
  top: 6px; right: 6px;
  border-top: 1px solid var(--accent);
  border-right: 1px solid var(--accent);
}
.corner-dot.bl {
  bottom: 6px; left: 6px;
  border-bottom: 1px solid var(--accent);
  border-left: 1px solid var(--accent);
}
.corner-dot.br {
  bottom: 6px; right: 6px;
  border-bottom: 1px solid var(--accent);
  border-right: 1px solid var(--accent);
}
.bz-pillar:hover .corner-dot {
  opacity: 0.8;
}
.bz-pillar:hover {
  /* hover shadow will be set dynamically via JS client, providing fallback here */
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.08), 0 0 12px var(--accent-bg);
}
@keyframes pillar-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.bz-p-label {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 2px;
  margin-bottom: 10px;
}
.bz-p-sprite { margin-bottom: 8px; display: block; margin-inline: auto; }
.bz-p-ganzhi {
  font-family: var(--serif);
  font-size: 26px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
  margin-bottom: 4px;
}
.bz-p-shishen { font-size: 12px; color: var(--accent-dim); font-weight: 500; }
.bz-p-nayin { font-size: 11px; color: var(--text-muted); margin-top: 2px; }

/* 神煞小徽章 */
.bz-p-shensha {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 4px;
  margin-top: 10px;
  border-top: 1px dashed var(--border-light);
  padding-top: 8px;
}
.bz-ss-badge {
  font-size: 9px;
  padding: 1px 5px;
  border-radius: 4px;
  font-weight: 500;
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
  background: rgba(196, 122, 106, 0.08);
  color: var(--wx-fire);
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
  font-weight: 500;
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
.bz-table .strong { font-weight: 600; color: var(--text-primary); }
.bz-table .shishen { color: var(--accent-dim); font-weight: 500; }
.bz-table .dim { color: var(--text-muted); }

.bz-hidegan {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.bz-hg-label { font-weight: 500; }
.bz-hg-item strong { color: var(--text-secondary); }

.bz-extra {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-muted);
  flex-wrap: wrap;
  margin-bottom: 14px;
}
.bz-extra strong { color: var(--text-secondary); }
.bz-extra .dim { color: var(--text-muted); opacity: 0.7; }

.bz-wuxing { margin-bottom: 14px; }
.bz-wx-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 5px 0;
  font-size: 12px;
}
.bz-wx-label { width: 18px; font-weight: 500; color: var(--text-muted); }
.bz-wx-bar { flex: 1; height: 6px; background: var(--border-light); border-radius: 3px; overflow: hidden; }
.bz-wx-fill { height: 100%; border-radius: 3px; transition: width 0.6s cubic-bezier(0.22,0.61,0.36,1); }
.bz-wx-fill.wx-wood  { background: var(--wx-wood); }
.bz-wx-fill.wx-fire  { background: var(--wx-fire); }
.bz-wx-fill.wx-earth { background: var(--wx-earth); }
.bz-wx-fill.wx-metal { background: var(--wx-metal); }
.bz-wx-fill.wx-water { background: var(--wx-water); }
.bz-wx-count { width: 18px; text-align: right; font-weight: 500; color: var(--text-secondary); }

/* 命运时间轴 (Fate Timeline) */
.bz-dayun-timeline {
  position: relative;
  margin-top: 24px;
  padding: 12px 0 16px;
  overflow-x: auto;
  scrollbar-width: none; /* 隐藏 Firefox 滚动条 */
}
.bz-dayun-timeline::-webkit-scrollbar {
  display: none; /* 隐藏 Chrome/Safari 滚动条 */
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
  flex-direction: column;
  align-items: center;
  cursor: pointer;
  flex: 1;
  transition: transform 0.2s cubic-bezier(0.25, 0.8, 0.25, 1);
}
.bz-node-age {
  font-size: 9px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
  margin-bottom: 6px;
  font-weight: 500;
  transition: color 0.2s;
}
.bz-node-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--bg-secondary);
  border: 1.5px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 10px;
  transition: border-color 0.2s, background-color 0.2s, box-shadow 0.2s;
}
.bz-node-dot-inner {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--border);
  transition: background-color 0.2s;
}
.bz-node-ganzhi {
  font-family: var(--serif);
  font-size: 15px;
  font-weight: 700;
  color: var(--text-secondary);
  line-height: 1.2;
  transition: color 0.2s, font-size 0.2s;
}

/* 命运轴高亮与 hover 交互 */
.bz-timeline-node:hover {
  transform: translateY(-2px);
}
.bz-timeline-node:hover .bz-node-dot {
  border-color: var(--accent-dim);
}
.bz-timeline-node:hover .bz-node-ganzhi {
  color: var(--text-primary);
}

.bz-timeline-node.active .bz-node-age {
  color: var(--accent-dim);
  font-weight: 600;
}
.bz-timeline-node.active .bz-node-dot {
  border-color: var(--accent);
  background: var(--accent-bg);
  box-shadow: 0 0 8px rgba(184, 149, 106, 0.5);
}
.bz-timeline-node.active .bz-node-dot-inner {
  background: var(--accent);
}
.bz-timeline-node.active .bz-node-ganzhi {
  color: var(--accent);
  font-size: 17px;
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
</style>
