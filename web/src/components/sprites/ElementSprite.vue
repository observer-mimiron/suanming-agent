<template>
  <svg :width="size" :height="size" viewBox="0 0 48 48" v-bind="$attrs">
    <!-- Wood -->
    <template v-if="element === 'wood'">
      <circle cx="24" cy="18" r="13" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.2"/>
      <circle cx="24" cy="18" r="8" :fill="c.fillInner" :stroke="c.stroke" stroke-width="0.8"/>
      <rect x="21" y="29" width="6" height="12" rx="3" fill="#d4c8b0" stroke="#b0a080" stroke-width="0.8"/>
      <line x1="20" y1="18" x2="14" y2="12" :stroke="c.stroke" stroke-width="0.8" stroke-linecap="round"/>
      <ellipse cx="12" cy="10" rx="4" ry="2.5" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.7" transform="rotate(-25 12 10)"/>
      <line x1="28" y1="16" x2="34" y2="10" :stroke="c.stroke" stroke-width="0.7" stroke-linecap="round"/>
      <ellipse cx="36" cy="8" rx="3.5" ry="2" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.6" transform="rotate(20 36 8)"/>
      <circle cx="24" cy="18" r="3" fill="#faf8f5" :stroke="c.stroke" stroke-width="0.6"/>
    </template>

    <!-- Fire -->
    <template v-if="element === 'fire'">
      <ellipse cx="24" cy="24" rx="14" ry="18" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.2"/>
      <ellipse cx="24" cy="24" rx="10" ry="14" :fill="c.fillInner" :stroke="c.stroke" stroke-width="0.8"/>
      <path d="M24 3 Q28 14 24 18 Q20 14 24 3" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="1"/>
      <circle cx="18" cy="20" r="2.5" fill="#faf8f5" :stroke="c.stroke" stroke-width="0.6"/>
      <circle cx="30" cy="22" r="2" fill="#faf8f5" :stroke="c.stroke" stroke-width="0.6"/>
      <ellipse cx="16" cy="30" rx="4" ry="7" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.6" transform="rotate(-15 16 30)"/>
      <ellipse cx="32" cy="32" rx="3.5" ry="6" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.6" transform="rotate(10 32 32)"/>
    </template>

    <!-- Earth -->
    <template v-if="element === 'earth'">
      <polygon points="24,2 42,34 6,34" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.2" stroke-linejoin="round"/>
      <polygon points="24,10 36,34 12,34" :fill="c.fillInner" :stroke="c.stroke" stroke-width="0.7" stroke-linejoin="round"/>
      <line x1="24" y1="2" x2="24" y2="34" :stroke="c.stroke" stroke-width="0.4" stroke-dasharray="2 3" opacity="0.4"/>
      <circle cx="24" cy="8" r="2.5" fill="#faf8f5" :stroke="c.stroke" stroke-width="0.8"/>
      <rect x="16" y="20" width="2.5" height="2.5" rx="0.5" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="0.5" transform="rotate(25 17 21)"/>
      <rect x="29" y="16" width="2" height="2" rx="0.5" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="0.5" transform="rotate(-20 30 17)"/>
    </template>

    <!-- Metal -->
    <template v-if="element === 'metal'">
      <circle cx="24" cy="24" r="18" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.6" opacity="0.5"/>
      <polygon points="24,2 38,24 24,46 10,24" :fill="c.fillInner" :stroke="c.stroke" stroke-width="1.2" stroke-linejoin="round"/>
      <polygon points="24,9 34,24 24,39 14,24" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="0.8" stroke-linejoin="round"/>
      <line x1="10" y1="24" x2="38" y2="24" :stroke="c.stroke" stroke-width="0.4" opacity="0.3"/>
      <line x1="24" y1="2" x2="24" y2="46" :stroke="c.stroke" stroke-width="0.4" opacity="0.3"/>
      <circle cx="24" cy="24" r="3.5" fill="#faf8f5" :stroke="c.stroke" stroke-width="1"/>
      <circle cx="16" cy="16" r="1" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="0.4"/>
      <circle cx="34" cy="32" r="1" :fill="c.fillAccent" :stroke="c.stroke" stroke-width="0.4"/>
    </template>

    <!-- Water -->
    <template v-if="element === 'water'">
      <ellipse cx="24" cy="24" rx="14" ry="15" :fill="c.fillBg" :stroke="c.stroke" stroke-width="1.2"/>
      <ellipse cx="24" cy="24" rx="10" ry="12" :fill="c.fillInner" :stroke="c.stroke" stroke-width="0.8"/>
      <path d="M14 18 Q24 12 34 18" :stroke="c.stroke" stroke-width="0.7" fill="none" stroke-linecap="round" opacity="0.5"/>
      <path d="M16 22 Q24 17 32 22" :stroke="c.stroke" stroke-width="0.6" fill="none" stroke-linecap="round" opacity="0.4"/>
      <circle cx="24" cy="28" r="2.5" fill="#faf8f5" :stroke="c.stroke" stroke-width="0.8"/>
      <circle cx="18" cy="10" r="2" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.5" opacity="0.6"/>
      <circle cx="30" cy="8" r="1.5" :fill="c.fillBg" :stroke="c.stroke" stroke-width="0.5" opacity="0.4"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  element: 'wood' | 'fire' | 'earth' | 'metal' | 'water'
  size?: number
}>(), { size: 48 })

const colorMap: Record<string, { stroke: string; fillBg: string; fillInner: string; fillAccent: string }> = {
  wood:  { stroke: '#8aa880', fillBg: '#e8f0e4', fillInner: '#d4e4cc', fillAccent: '#c0d8b4' },
  fire:  { stroke: '#c4806a', fillBg: '#fae8e0', fillInner: '#f5d8cc', fillAccent: '#f0c0a0' },
  earth: { stroke: '#b09070', fillBg: '#f0e8d8', fillInner: '#e8dcc8', fillAccent: '#d4c0a0' },
  metal: { stroke: '#c4a870', fillBg: '#f8f4e8', fillInner: '#f5eed8', fillAccent: '#efe0c0' },
  water: { stroke: '#7a9ab8', fillBg: '#e0e8f2', fillInner: '#d0dce8', fillAccent: '#c0d0e0' },
}

const c = computed(() => colorMap[props.element] || colorMap.earth)
</script>
