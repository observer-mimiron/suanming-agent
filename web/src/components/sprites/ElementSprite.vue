<template>
  <svg :width="size" :height="size" viewBox="0 0 48 48" v-bind="$attrs">
    <!-- Wood — a sprouting seed -->
    <template v-if="element === 'wood'">
      <ellipse cx="24" cy="34" rx="12" ry="10" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.5"/>
      <path d="M24 26 Q22 16 24 6" :stroke="c.stroke" stroke-width="2" fill="none" stroke-linecap="round"/>
      <path d="M24 14 Q16 8 10 12" :stroke="c.stroke" stroke-width="1.5" fill="none" stroke-linecap="round"/>
      <path d="M24 14 Q32 8 38 12" :stroke="c.stroke" stroke-width="1.5" fill="none" stroke-linecap="round"/>
    </template>

    <!-- Fire — a flame -->
    <template v-if="element === 'fire'">
      <ellipse cx="24" cy="30" rx="10" ry="14" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.5"/>
      <path d="M24 4 Q30 18 24 24" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="1.5" stroke-linecap="round"/>
      <path d="M24 4 Q18 18 24 24" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="1.5" stroke-linecap="round"/>
      <circle cx="24" cy="20" r="3" fill="#faf8f5" :stroke="c.stroke" stroke-width="1"/>
    </template>

    <!-- Earth — a mountain -->
    <template v-if="element === 'earth'">
      <polygon points="24,4 44,40 4,40" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.5" stroke-linejoin="round"/>
      <polygon points="24,16 34,40 14,40" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="1" stroke-linejoin="round"/>
      <circle cx="24" cy="12" r="3" fill="#faf8f5" :stroke="c.stroke" stroke-width="1.2"/>
    </template>

    <!-- Metal — a crystal/diamond -->
    <template v-if="element === 'metal'">
      <polygon points="24,2 42,24 24,46 6,24" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.5" stroke-linejoin="round"/>
      <polygon points="24,10 34,24 24,38 14,24" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="1" stroke-linejoin="round"/>
      <circle cx="24" cy="24" r="4" fill="#faf8f5" :stroke="c.stroke" stroke-width="1.2"/>
    </template>

    <!-- Water — a droplet -->
    <template v-if="element === 'water'">
      <path d="M24 4 Q38 24 24 42 Q10 24 24 4Z" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.5" stroke-linejoin="round"/>
      <path d="M24 14 Q32 24 24 34 Q16 24 24 14Z" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="1" stroke-linejoin="round"/>
      <circle cx="24" cy="24" r="3" fill="#faf8f5" :stroke="c.stroke" stroke-width="1"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  element: 'wood' | 'fire' | 'earth' | 'metal' | 'water'
  size?: number
}>(), { size: 48 })

const colorMap: Record<string, { stroke: string; fillBg: string; fillAccent: string; fillInner?: string }> = {
  wood:  { stroke: '#8aa880', fillBg: '#e8f0e4', fillAccent: '#d0e0cc' },
  fire:  { stroke: '#c4806a', fillBg: '#fae8e0', fillAccent: '#f0c0a0' },
  earth: { stroke: '#b09070', fillBg: '#f0e8d8', fillAccent: '#e0d0b8' },
  metal: { stroke: '#c4a870', fillBg: '#f8f4e8', fillAccent: '#efe0c0' },
  water: { stroke: '#7a9ab8', fillBg: '#e0e8f2', fillAccent: '#c8d8e8' },
}

const c = computed(() => colorMap[props.element] || colorMap.earth)
</script>
