<template>
  <svg :width="size" :height="size" viewBox="0 0 32 32">
    <circle cx="16" cy="16" r="14" :fill="cfg.bg" :stroke="cfg.stroke" stroke-width="1"/>
    <polygon
      :points="starPoints"
      :stroke="cfg.stroke"
      :fill="cfg.fill"
      stroke-width="0.8"
      stroke-linejoin="round"
    />
    <text x="16" y="27" text-anchor="middle" :fill="cfg.stroke" font-size="6" font-family="Inter,sans-serif" font-weight="600">{{ cfg.label }}</text>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  starType: 'tianpeng' | 'tianying' | 'tianchong' | 'tianfu' | 'tianqin' | 'tianxin' | 'tianzhu' | 'tianren' | 'tianrui'
  size?: number
}>(), { size: 32 })

const configs: Record<string, { points: number; stroke: string; bg: string; fill: string; label: string; starR: number }> = {
  tianpeng:  { points: 9, stroke: '#6b8aa8', bg: '#e8eef4', fill: '#d0dce8', label: '蓬', starR: 9 },
  tianying:  { points: 8, stroke: '#c47a6a', bg: '#faeae4', fill: '#f0d0c0', label: '英', starR: 8 },
  tianchong: { points: 7, stroke: '#7a9e7e', bg: '#ecf2ec', fill: '#d0e4d0', label: '冲', starR: 8 },
  tianfu:    { points: 6, stroke: '#7a9e7e', bg: '#ecf2ec', fill: '#d0e4d0', label: '辅', starR: 7 },
  tianqin:   { points: 5, stroke: '#b8956a', bg: '#f2ece0', fill: '#e8dcc8', label: '禽', starR: 7 },
  tianxin:   { points: 6, stroke: '#c4a96a', bg: '#f6f2e4', fill: '#efe4cc', label: '心', starR: 7 },
  tianzhu:   { points: 7, stroke: '#c4a96a', bg: '#f6f2e4', fill: '#efe4cc', label: '柱', starR: 8 },
  tianren:   { points: 8, stroke: '#b8956a', bg: '#f2ece0', fill: '#e8dcc8', label: '任', starR: 8 },
  tianrui:   { points: 9, stroke: '#b8956a', bg: '#f2ece0', fill: '#e8dcc8', label: '芮', starR: 9 },
}

const cfg = computed(() => configs[props.starType] || configs.tianqin)

const starPoints = computed(() => {
  const { points: n, starR: r } = cfg.value
  const cx = 16, cy = 16
  const pts: string[] = []
  for (let i = 0; i < n * 2; i++) {
    const radius = i % 2 === 0 ? r : r * 0.45
    const angle = (Math.PI * i) / n - Math.PI / 2
    pts.push(`${cx + radius * Math.cos(angle)},${cy + radius * Math.sin(angle)}`)
  }
  return pts.join(' ')
})
</script>
