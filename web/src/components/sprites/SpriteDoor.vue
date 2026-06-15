<template>
  <svg :width="size" :height="size" viewBox="0 0 52 52">
    <!-- 开门 · 乾 · 金 — 朝阳地平线 -->
    <template v-if="doorType === 'kai'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fdfcf8"/><stop offset="100%" stop-color="#f4eed8"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a96a" stroke-width="1.1"/>
      <line x1="8" y1="32" x2="44" y2="32" stroke="#b8956a" stroke-width="1.3" stroke-linecap="round"/>
      <circle cx="26" cy="24" r="8" fill="none" stroke="#b8956a" stroke-width="1.8"/>
      <line x1="26" y1="10" x2="26" y2="16" stroke="#b8956a" stroke-width="1" stroke-linecap="round" opacity="0.5"/>
      <line x1="26" y1="32" x2="26" y2="40" stroke="#b8956a" stroke-width="1" stroke-linecap="round" opacity="0.5"/>
    </template>

    <!-- 休门 · 坎 · 水 — 弯月静水 -->
    <template v-if="doorType === 'xiu'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f8fafb"/><stop offset="100%" stop-color="#dce8f0"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#8aacc4" stroke-width="1.1"/>
      <path d="M22,12 Q32,22 22,32" fill="none" stroke="#6b8aa8" stroke-width="2.2" stroke-linecap="round"/>
      <line x1="8" y1="36" x2="44" y2="36" stroke="#8aacc4" stroke-width="1.2" stroke-linecap="round" opacity="0.5"/>
      <path d="M12,38 Q18,35 26,38 Q34,41 40,38" fill="none" stroke="#6b8aa8" stroke-width="0.8" opacity="0.35"/>
    </template>

    <!-- 生门 · 艮 · 土 — 种子破土 -->
    <template v-if="doorType === 'sheng'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fafdf8"/><stop offset="100%" stop-color="#e2eedc"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#7a9e7e" stroke-width="1.1"/>
      <line x1="8" y1="34" x2="44" y2="34" stroke="#7a9e7e" stroke-width="1.3" stroke-linecap="round"/>
      <path d="M26,34 Q24,24 26,18" fill="none" stroke="#7a9e7e" stroke-width="2" stroke-linecap="round"/>
      <path d="M26,22 Q18,14 10,18" fill="none" stroke="#7a9e7e" stroke-width="1.4" stroke-linecap="round" opacity="0.7"/>
      <path d="M26,22 Q34,14 42,18" fill="none" stroke="#7a9e7e" stroke-width="1.4" stroke-linecap="round" opacity="0.7"/>
    </template>

    <!-- 伤门 · 震 · 木 — 碎裂圆 -->
    <template v-if="doorType === 'shang'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fafdf8"/><stop offset="100%" stop-color="#e2eedc"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#7a9e7e" stroke-width="1.1"/>
      <circle cx="26" cy="26" r="13" fill="none" stroke="#7a9e7e" stroke-width="1.6"/>
      <line x1="18" y1="14" x2="26" y2="26" stroke="#7a9e7e" stroke-width="2" stroke-linecap="round"/>
      <line x1="26" y1="26" x2="36" y2="18" stroke="#7a9e7e" stroke-width="1.5" stroke-linecap="round" opacity="0.6"/>
      <line x1="22" y1="32" x2="14" y2="40" stroke="#7a9e7e" stroke-width="1" stroke-linecap="round" opacity="0.35"/>
    </template>

    <!-- 杜门 · 巽 · 木 — 封锁X -->
    <template v-if="doorType === 'du'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fafdf8"/><stop offset="100%" stop-color="#e2eedc"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#7a9e7e" stroke-width="1.1"/>
      <rect x="8" y="8" width="36" height="36" rx="6" fill="none" stroke="#7a9e7e" stroke-width="1.5"/>
      <line x1="14" y1="14" x2="38" y2="38" stroke="#7a9e7e" stroke-width="1.8" stroke-linecap="round"/>
      <line x1="38" y1="14" x2="14" y2="38" stroke="#7a9e7e" stroke-width="1.8" stroke-linecap="round"/>
      <circle cx="26" cy="26" r="4" fill="#fafdf8" stroke="#7a9e7e" stroke-width="1.3"/>
    </template>

    <!-- 景门 · 离 · 火 — 开窗见景 -->
    <template v-if="doorType === 'jing'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fefcfa"/><stop offset="100%" stop-color="#fae4d8"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#cc9888" stroke-width="1.1"/>
      <rect x="12" y="10" width="28" height="28" rx="5" fill="none" stroke="#c47a6a" stroke-width="1.6"/>
      <path d="M16,28 Q22,22 28,28 Q34,34 40,28" fill="none" stroke="#c47a6a" stroke-width="1.2" opacity="0.5"/>
      <circle cx="26" cy="18" r="3.5" fill="none" stroke="#c47a6a" stroke-width="1.2" opacity="0.4"/>
    </template>

    <!-- 死门 · 坤 · 土 — 落日 -->
    <template v-if="doorType === 'si'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fdfaf6"/><stop offset="100%" stop-color="#f0e6d2"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a980" stroke-width="1.1"/>
      <line x1="8" y1="32" x2="44" y2="32" stroke="#b8956a" stroke-width="1.3" stroke-linecap="round"/>
      <path d="M17,30 A9,9 0 0,1 35,30" fill="none" stroke="#b8956a" stroke-width="1.8"/>
    </template>

    <!-- 惊门 · 兑 · 金 — 突刺 -->
    <template v-if="doorType === 'jingmen'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fdfcf8"/><stop offset="100%" stop-color="#f4ecd6"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a96a" stroke-width="1.1"/>
      <line x1="26" y1="6" x2="26" y2="44" stroke="#b8956a" stroke-width="1.8" stroke-linecap="round"/>
      <line x1="14" y1="16" x2="26" y2="20" stroke="#b8956a" stroke-width="1.6" stroke-linecap="round"/>
      <line x1="14" y1="30" x2="26" y2="30" stroke="#b8956a" stroke-width="1.6" stroke-linecap="round"/>
      <line x1="14" y1="40" x2="26" y2="40" stroke="#b8956a" stroke-width="1.2" stroke-linecap="round" opacity="0.5"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  doorType: 'kai' | 'xiu' | 'sheng' | 'shang' | 'du' | 'jing' | 'si' | 'jingmen'
  size?: number
}>(), { size: 48 })

const gid = `sd-${Math.random().toString(36).slice(2, 8)}`
</script>
