<template>
  <div class="zw-card">
    <!-- Header -->
    <div class="zw-header">
      <div class="zw-header-row">
        <span class="zw-label">紫微斗数命盘</span>
        <span class="zw-meta">{{ data.gender }} · {{ data.lunar_date }} · {{ data.five_elements_class }}</span>
      </div>
      <div class="zw-palace-basis">
        <span class="zw-basis-item">命宫{{ data.soul_palace_branch }}·{{ data.soul_palace_ganzhi }}</span>
        <span class="zw-basis-item">身宫{{ data.body_palace_branch }}</span>
        <span class="zw-basis-item">命主{{ data.soul_master }}</span>
        <span class="zw-basis-item">身主{{ data.body_master }}</span>
      </div>
    </div>

    <!-- 12-Palace Grid -->
    <div class="zw-grid">
      <div
        v-for="(p, i) in palaces"
        :key="p.index"
        class="zw-cell"
        :class="{
          'zw-cell-soul': p.name === '命宫',
          'zw-cell-body': p.is_body_palace,
        }"
        :style="{ animationDelay: i * 50 + 'ms' }"
        @mousemove="handleTilt"
        @mouseleave="resetTilt"
      >
        <!-- Corner dots -->
        <span class="corner-dot tl"></span>
        <span class="corner-dot tr"></span>
        <span class="corner-dot bl"></span>
        <span class="corner-dot br"></span>

        <div class="zw-cell-top">
          <span class="zw-palace-name">
            {{ p.name }}
            <span v-if="p.is_body_palace" class="zw-body-badge">身</span>
          </span>
          <span class="zw-palace-branch">{{ p.heavenly_stem }}{{ p.earthly_branch }}</span>
        </div>

        <!-- Major stars -->
        <div class="zw-stars">
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

        <!-- Minor stars -->
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

        <!-- Decadal age bar -->
        <div v-if="p.decadal" class="zw-decadal">
          {{ p.decadal.start_age }}–{{ p.decadal.end_age }}岁
        </div>
      </div>
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
    '庙': 'zm-br-miao',
    '旺': 'zm-br-wang',
    '得': 'zm-br-de',
    '利': 'zm-br-li',
    '平': 'zm-br-ping',
    '不': 'zm-br-bu',
    '陷': 'zm-br-xian',
  }
  return map[brightness] || 'zm-br-default'
}

function mutagenShort(mutagen: string): string {
  const map: Record<string, string> = {
    '化禄': '禄',
    '化权': '权',
    '化科': '科',
    '化忌': '忌',
  }
  return map[mutagen] || mutagen
}

function mutagenClass(mutagen: string): string {
  const map: Record<string, string> = {
    '化禄': 'zm-mu-lu',
    '化权': 'zm-mu-quan',
    '化科': 'zm-mu-ke',
    '化忌': 'zm-mu-ji',
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
}

.zw-header {
  margin-bottom: 16px;
  padding: 0 2px;
}

.zw-header-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 8px;
}

.zw-label {
  font-family: var(--serif);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}

.zw-meta {
  font-size: 11px;
  color: var(--text-muted);
}

.zw-palace-basis {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 11px;
  color: var(--text-muted);
}

.zw-basis-item {
  font-variant-numeric: tabular-nums;
}

/* 12-palace grid: 4 columns */
.zw-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.zw-cell {
  position: relative;
  background: var(--bg-hover);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 10px 10px 12px;
  min-height: 120px;
  opacity: 0;
  overflow: hidden;
  transform-style: preserve-3d;
  animation: zw-cell-in 0.35s cubic-bezier(0.22, 0.61, 0.36, 1) forwards;
  transition: transform 0.25s cubic-bezier(0.25, 0.8, 0.25, 1), box-shadow 0.25s ease, border-color 0.2s;
}

@keyframes zw-cell-in {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.zw-cell::after {
  content: "";
  position: absolute;
  top: -50%; left: -50%;
  width: 200%; height: 200%;
  background: linear-gradient(
    115deg,
    transparent 40%,
    rgba(184, 149, 106, 0.06) 48%,
    rgba(255, 255, 255, 0.12) 50%,
    rgba(184, 149, 106, 0.06) 52%,
    transparent 60%
  );
  transform: translate(-30%, -30%);
  pointer-events: none;
  opacity: 0;
}

.zw-cell:hover::after {
  transform: translate(15%, 15%);
  transition: transform 0.8s cubic-bezier(0.19, 1, 0.22, 1);
  opacity: 1;
}

.zw-cell:hover {
  z-index: 2;
}

/* 几何直角 L 包角 (现代塔罗风格) */
.corner-dot {
  position: absolute;
  width: 6px;
  height: 6px;
  pointer-events: none;
  transition: opacity 0.25s ease;
  opacity: 0.2;
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
.zw-cell:hover .corner-dot {
  opacity: 0.8;
}

/* Soul palace (命宫) highlight */
.zw-cell-soul {
  border-color: var(--accent) !important;
  background: rgba(184, 149, 106, 0.04);
}
.zw-cell-soul .corner-dot {
  opacity: 0.6;
}

/* Body palace (身宫) highlight */
.zw-cell-body {
  border-color: var(--wx-wood) !important;
}
.zw-cell-body .corner-dot {
  border-top-color: var(--wx-wood);
  border-left-color: var(--wx-wood);
  border-bottom-color: var(--wx-wood);
  border-right-color: var(--wx-wood);
}

.zw-cell-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  border-bottom: 1px dashed var(--border-light);
  padding-bottom: 6px;
}

.zw-palace-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.zw-body-badge {
  background: rgba(122, 158, 126, 0.12);
  color: var(--wx-wood);
  font-size: 9px;
  font-weight: 500;
  padding: 1px 5px;
  border-radius: 4px;
  line-height: 1.3;
}

.zw-palace-branch {
  font-size: 11px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

/* Stars */
.zw-stars {
  margin-bottom: 4px;
}

.zw-minor-stars {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 7px;
  margin-bottom: 4px;
}

.zw-star-item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  font-size: 11px;
  line-height: 1.5;
}

.zw-star-major .zw-star-name {
  font-weight: 600;
  color: var(--accent);
}

.zw-star-minor {
  font-size: 10px;
}

.zw-star-minor .zw-star-name {
  color: var(--text-muted);
}

/* Brightness indicators */
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

/* Mutagen badges */
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

/* Decadal bar */
.zw-decadal {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: auto;
  border-top: 1px solid var(--border-light);
  padding-top: 6px;
  text-align: center;
}
</style>
