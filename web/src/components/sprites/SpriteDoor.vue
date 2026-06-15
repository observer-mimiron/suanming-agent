<template>
  <svg :width="size" :height="size" viewBox="0 0 32 32">
    <rect x="2" y="2" width="28" height="28" rx="4" :fill="cfg.bg" :stroke="cfg.stroke" stroke-width="1"/>
    <!-- Door frame -->
    <rect x="9" y="6" width="14" height="20" rx="2" :stroke="cfg.stroke" stroke-width="1" fill="none"/>
    <!-- Center divider -->
    <line x1="16" y1="6" x2="16" y2="26" :stroke="cfg.stroke" stroke-width="0.6" opacity="0.4"/>
    <!-- Door swing indicator -->
    <line x1="16" y1="8" :x2="16 + cfg.offset" y2="16" :stroke="cfg.stroke" stroke-width="1.2" stroke-linecap="round"/>
    <line x1="16" y1="8" :x2="16 - cfg.offset" y2="16" :stroke="cfg.stroke" stroke-width="0.7" stroke-linecap="round" opacity="0.4"/>
    <text x="16" y="29" text-anchor="middle" :fill="cfg.stroke" font-size="6" font-family="Inter,sans-serif" font-weight="600">{{ cfg.label }}</text>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  doorType: 'kai' | 'xiu' | 'sheng' | 'shang' | 'du' | 'jing' | 'si' | 'jingmen'
  size?: number
}>(), { size: 32 })

const configs: Record<string, { stroke: string; bg: string; label: string; offset: number }> = {
  kai:     { stroke: '#c4a96a', bg: '#f8f4e8', label: '开', offset: 5 },
  xiu:     { stroke: '#6b8aa8', bg: '#e8eef4', label: '休', offset: 3 },
  sheng:   { stroke: '#b8956a', bg: '#f2ece0', label: '生', offset: -3 },
  shang:   { stroke: '#7a9e7e', bg: '#ecf2ec', label: '伤', offset: -4 },
  du:      { stroke: '#7a9e7e', bg: '#ecf2ec', label: '杜', offset: 1 },
  jing:    { stroke: '#c47a6a', bg: '#faeae4', label: '景', offset: 4 },
  si:      { stroke: '#b8956a', bg: '#f2ece0', label: '死', offset: -5 },
  jingmen: { stroke: '#c4a96a', bg: '#f8f4e8', label: '惊', offset: 2 },
}

const cfg = computed(() => configs[props.doorType] || configs.du)
</script>
