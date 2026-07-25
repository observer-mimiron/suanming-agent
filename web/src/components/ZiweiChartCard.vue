<template>
  <div class="zw-card">
    <div class="zw-header">
      <div class="zw-header-row">
        <div>
          <div class="zw-kicker">紫微斗数命盘</div>
          <div class="zw-meta">{{ data.gender }} · {{ data.lunar_date }} · {{ data.five_elements_class }}</div>
        </div>
        <div class="zw-pillars">
          <span v-for="(value, key) in data.four_pillars || {}" :key="key" class="zw-pillar-chip">
            {{ key }} {{ value }}
          </span>
        </div>
      </div>

      <div class="zw-overview">
        <div class="zw-overview-card">
          <span class="zw-overview-label">命宫</span>
          <strong class="zw-overview-value">{{ data.soul_palace_branch }} · {{ data.soul_palace_ganzhi }}</strong>
        </div>
        <div class="zw-overview-card">
          <span class="zw-overview-label">身宫</span>
          <strong class="zw-overview-value">{{ data.body_palace_branch }}</strong>
        </div>
        <div class="zw-overview-card">
          <span class="zw-overview-label">命主</span>
          <strong class="zw-overview-value">{{ data.soul_master }}</strong>
        </div>
        <div class="zw-overview-card">
          <span class="zw-overview-label">身主</span>
          <strong class="zw-overview-value">{{ data.body_master }}</strong>
        </div>
      </div>
    </div>

    <div class="zw-grid">
      <div
        v-for="(p, i) in palaces"
        :key="p.index"
        class="zw-cell"
        :class="{
          'zw-cell-soul': p.name === '命宫',
          'zw-cell-body': p.is_body_palace,
          'zw-cell-original': p.is_original_palace,
        }"
        :style="{ animationDelay: i * 50 + 'ms' }"
        @mousemove="handleTilt"
        @mouseleave="resetTilt"
      >
        <div class="zw-cell-top">
          <div class="zw-title-stack">
            <span class="zw-palace-name">{{ p.name }}</span>
            <div class="zw-palace-tags">
              <span v-if="p.is_body_palace" class="zw-tag zw-tag-body">身宫</span>
              <span v-if="p.is_original_palace" class="zw-tag zw-tag-origin">来因宫</span>
            </div>
          </div>
          <span class="zw-palace-branch">{{ p.heavenly_stem }}{{ p.earthly_branch }}</span>
        </div>

        <div class="zw-section">
          <div class="zw-section-label">主星</div>
          <div
            v-for="star in p.major_stars"
            :key="star.name"
            class="zw-star-item zw-star-major"
            :class="brightnessClass(star.brightness)"
          >
            <span class="zw-star-name">{{ star.name }}</span>
            <span v-if="star.brightness" class="zw-star-brightness">{{ star.brightness }}</span>
            <span v-if="star.mutagen" class="zw-star-mutagen" :class="mutagenClass(star.mutagen)">{{ mutagenShort(star.mutagen) }}</span>
          </div>
        </div>

        <div class="zw-section" v-if="p.minor_stars?.length">
          <div class="zw-section-label">辅曜</div>
          <div class="zw-minor-stars">
            <span
              v-for="star in p.minor_stars"
              :key="star.name"
              class="zw-star-item zw-star-minor"
            >
              <span class="zw-star-name">{{ star.name }}</span>
              <span v-if="star.mutagen" class="zw-star-mutagen" :class="mutagenClass(star.mutagen)">{{ mutagenShort(star.mutagen) }}</span>
            </span>
          </div>
        </div>

        <div class="zw-foot">
          <div class="zw-growth" v-if="p.changsheng_12 || p.boshi_12">
            <span v-if="p.changsheng_12">{{ p.changsheng_12 }}</span>
            <span v-if="p.boshi_12">{{ p.boshi_12 }}</span>
          </div>
          <div v-if="p.decadal" class="zw-decadal">
            大限 {{ p.decadal.start_age }}-{{ p.decadal.end_age }} 岁
          </div>
        </div>
      </div>
    </div>

    <div v-if="palaces.length" class="zw-legend">
      <span class="zw-legend-item"><i class="zw-legend-dot soul"></i> 命宫</span>
      <span class="zw-legend-item"><i class="zw-legend-dot body"></i> 身宫</span>
      <span class="zw-legend-item"><i class="zw-legend-dot origin"></i> 来因宫</span>
      <span class="zw-legend-item">主星含亮度与四化标记</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ data: any }>()

const palaces = computed<any[]>(() => props.data?.palaces || [])

function brightnessClass(brightness?: string): string {
  if (!brightness) return 'zm-br-default'
  const map: Record<string, string> = {
    庙: 'zm-br-miao',
    旺: 'zm-br-wang',
    得: 'zm-br-de',
    利: 'zm-br-li',
    平: 'zm-br-ping',
    不: 'zm-br-bu',
    陷: 'zm-br-xian',
  }
  return map[brightness] || 'zm-br-default'
}

function mutagenShort(mutagen: string): string {
  const map: Record<string, string> = {
    化禄: '禄',
    化权: '权',
    化科: '科',
    化忌: '忌',
  }
  return map[mutagen] || mutagen
}

function mutagenClass(mutagen: string): string {
  const map: Record<string, string> = {
    化禄: 'zm-mu-lu',
    化权: 'zm-mu-quan',
    化科: 'zm-mu-ke',
    化忌: 'zm-mu-ji',
  }
  return map[mutagen] || ''
}

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
</script>

<style scoped>
.zw-card {
  text-align: left;
  font-size: 13px;
  max-width: 100%;
  border: 1px solid rgba(115, 92, 68, 0.14);
  border-radius: 20px;
  padding: 18px;
  background:
    linear-gradient(180deg, rgba(255, 251, 244, 0.92), rgba(248, 243, 233, 0.88)),
    radial-gradient(circle at top right, rgba(184, 149, 106, 0.12), transparent 38%);
}

.zw-header {
  margin-bottom: 16px;
}

.zw-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
}

.zw-kicker {
  font-family: var(--serif);
  font-size: 17px;
  font-weight: 700;
  color: #62452f;
}

.zw-meta {
  margin-top: 4px;
  font-size: 12px;
  color: #8b745f;
}

.zw-pillars {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.zw-pillar-chip {
  padding: 5px 9px;
  border-radius: 999px;
  background: rgba(125, 92, 63, 0.08);
  border: 1px solid rgba(125, 92, 63, 0.12);
  font-size: 11px;
  color: #6d533e;
}

.zw-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.zw-overview-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.56);
  border: 1px solid rgba(125, 92, 63, 0.1);
}

.zw-overview-label {
  font-size: 11px;
  color: #8b745f;
}

.zw-overview-value {
  font-family: var(--serif);
  font-size: 14px;
  color: #4d3928;
}

.zw-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.zw-cell {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(125, 92, 63, 0.12);
  border-radius: 16px;
  padding: 12px;
  min-height: 168px;
  opacity: 0;
  animation: zw-cell-in 0.35s cubic-bezier(0.22, 0.61, 0.36, 1) forwards;
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease, background 0.2s ease;
}

@keyframes zw-cell-in {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.zw-cell:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px rgba(108, 84, 62, 0.1);
}

.zw-cell-soul {
  border-color: #b8956a;
  background: rgba(250, 244, 234, 0.9);
}

.zw-cell-body {
  border-color: rgba(122, 158, 126, 0.46);
}

.zw-cell-original {
  box-shadow: inset 0 0 0 1px rgba(107, 138, 168, 0.28);
}

.zw-cell-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
  border-bottom: 1px solid rgba(125, 92, 63, 0.08);
  padding-bottom: 8px;
}

.zw-title-stack {
  min-width: 0;
}

.zw-palace-name {
  font-family: var(--serif);
  font-size: 15px;
  font-weight: 700;
  color: #543b28;
}

.zw-palace-branch {
  font-size: 11px;
  color: #8b745f;
  font-variant-numeric: tabular-nums;
}

.zw-palace-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 6px;
}

.zw-tag {
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 10px;
  line-height: 1.4;
}

.zw-tag-body {
  background: rgba(122, 158, 126, 0.12);
  color: #5b7e5f;
}

.zw-tag-origin {
  background: rgba(107, 138, 168, 0.12);
  color: #5c7693;
}

.zw-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.zw-section-label {
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #977a5d;
}

.zw-minor-stars {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 7px;
}

.zw-star-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  line-height: 1.5;
}

.zw-star-major .zw-star-name {
  font-weight: 600;
  color: #6a4c2f;
}

.zw-star-minor {
  font-size: 10px;
  padding: 2px 0;
}

.zw-star-minor .zw-star-name {
  color: #6f6257;
}

.zw-star-brightness {
  font-size: 9px;
  margin-left: 1px;
}

.zw-star-major.zm-br-miao .zw-star-brightness { color: #b8956a; }
.zw-star-major.zm-br-wang .zw-star-brightness { color: #d4a017; }
.zw-star-major.zm-br-de .zw-star-brightness { color: #c8a040; }
.zw-star-major.zm-br-li .zw-star-brightness { color: var(--text-secondary); }
.zw-star-major.zm-br-ping .zw-star-brightness { color: var(--text-muted); }
.zw-star-major.zm-br-bu .zw-star-brightness { color: #a0a0a0; }
.zw-star-major.zm-br-xian .zw-star-brightness { color: var(--wx-fire); }

.zw-star-mutagen {
  font-size: 9px;
  margin-left: 3px;
  padding: 0 3px;
  border-radius: 3px;
  font-weight: 500;
  line-height: 1.4;
}

.zm-mu-lu {
  background: rgba(122, 158, 126, 0.12);
  color: var(--wx-wood);
}

.zm-mu-quan {
  background: rgba(184, 149, 106, 0.12);
  color: var(--accent);
}

.zm-mu-ke {
  background: rgba(107, 138, 168, 0.12);
  color: var(--wx-water);
}

.zm-mu-ji {
  background: rgba(196, 122, 106, 0.12);
  color: var(--wx-fire);
}

.zw-foot {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px dashed rgba(125, 92, 63, 0.12);
}

.zw-growth {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 10px;
  color: #8b745f;
}

.zw-decadal {
  font-size: 10px;
  color: #6f6257;
  font-variant-numeric: tabular-nums;
}

.zw-legend {
  margin-top: 14px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px 14px;
  font-size: 11px;
  color: #8b745f;
}

.zw-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.zw-legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
}

.zw-legend-dot.soul {
  background: #b8956a;
}

.zw-legend-dot.body {
  background: #7a9e7e;
}

.zw-legend-dot.origin {
  background: #6b8aa8;
}

@media (max-width: 960px) {
  .zw-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .zw-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .zw-card {
    padding: 14px;
    border-radius: 16px;
  }

  .zw-overview,
  .zw-grid {
    grid-template-columns: 1fr;
  }

  .zw-header-row {
    flex-direction: column;
  }

  .zw-pillars {
    justify-content: flex-start;
  }
}
</style>
