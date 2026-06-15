<template>
  <div class="bazi-card">
    <!-- Header -->
    <div class="bz-header">
      <span class="bz-title">八字命盘</span>
      <span class="bz-meta">日主 <strong>{{ dayGan }}</strong>（{{ dayGanWuxing }}）· {{ lunarDate }}</span>
    </div>

    <!-- Four Pillars as sprite cards -->
    <div class="bz-pillars">
      <div v-for="(p, i) in pillars" :key="p.name" class="bz-pillar" :style="{ animationDelay: i * 80 + 'ms' }">
        <div class="bz-p-label">{{ p.name }}</div>
        <ElementSprite :element="pillarElement(p)" :size="44" class="bz-p-sprite" />
        <div class="bz-p-ganzhi">{{ p.stem }}{{ p.branch }}</div>
        <div class="bz-p-shishen">{{ p.shiShen }}</div>
        <div class="bz-p-nayin">{{ p.naYin }}</div>
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

    <!-- Dayun chips -->
    <div class="bz-dayun">
      <span v-for="(d, i) in dayun" :key="i" class="bz-dy-tag" :class="{ active: i === currentDayunIdx }">
        {{ d.startAge }}-{{ d.endAge }}岁 {{ d.ganZhi }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ElementSprite from './sprites/ElementSprite.vue'

const props = defineProps<{ data: any }>()
const pillars = computed(() => props.data?.pillars || [])
const dayGan = computed(() => props.data?.dayGan || '')
const dayGanWuxing = computed(() => props.data?.dayGanWuxing || '')
const lunarDate = computed(() => props.data?.lunarDate || '')
const wuxing = computed(() => props.data?.wuxing || {})
const dayun = computed(() => props.data?.dayun || [])
const mingGong = computed(() => props.data?.mingGong || '')
const mingGongNaYin = computed(() => props.data?.mingGongNaYin || '')
const shenGong = computed(() => props.data?.shenGong || '')
const shenGongNaYin = computed(() => props.data?.shenGongNaYin || '')
const taiYuan = computed(() => props.data?.taiYuan || '')
const taiYuanNaYin = computed(() => props.data?.taiYuanNaYin || '')

function pillarElement(p: any): 'wood' | 'fire' | 'earth' | 'metal' | 'water' {
  const wx = p.stemWuxing || p.branchWuxing || ''
  if (wx.includes('木')) return 'wood'
  if (wx.includes('火')) return 'fire'
  if (wx.includes('土')) return 'earth'
  if (wx.includes('金')) return 'metal'
  if (wx.includes('水')) return 'water'
  return 'earth'
}

function elementKey(k: string): string {
  const map: Record<string,string> = { '木':'wood','火':'fire','土':'earth','金':'metal','水':'water' }
  return map[k] || 'earth'
}

const currentDayunIdx = computed(() => {
  if (!props.data?.birthday) return -1
  const birthYear = parseInt(props.data.birthday) || 0
  const age = new Date().getFullYear() - birthYear
  return dayun.value.findIndex((d: any) => age >= d.startAge && age <= d.endAge)
})
</script>

<style scoped>
.bazi-card {
  text-align: left;
  font-size: 13px;
}
.bz-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 16px;
  padding: 0 4px;
}
.bz-title {
  font-family: var(--serif);
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
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
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 22px 10px 16px;
  text-align: center;
  opacity: 0;
  animation: pillar-in 0.4s cubic-bezier(0.22,0.61,0.36,1) forwards;
  transition: transform 0.2s, box-shadow 0.2s;
}
.bz-pillar:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-card);
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

.bz-dayun { display: flex; flex-wrap: wrap; gap: 6px; }
.bz-dy-tag {
  padding: 5px 12px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 500;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
}
.bz-dy-tag.active {
  background: var(--accent-bg);
  border-color: var(--accent);
  color: var(--accent-dim);
}
</style>
