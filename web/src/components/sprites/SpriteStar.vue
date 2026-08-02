<template>
  <svg :width="size" :height="size" viewBox="0 0 52 52">
    <!-- 天蓬 · 坎 · 水 — 北斗式散点星图 -->
    <template v-if="starType === 'tianpeng'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#eef4f8"/><stop offset="100%" stop-color="#d4e4ee"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#8aacc4" stroke-width="1.1"/>
      <circle cx="26" cy="12" r="3.5" fill="#6b8aa8"/>
      <circle cx="34" cy="22" r="2" fill="#6b8aa8" opacity="0.6"/>
      <circle cx="18" cy="22" r="2" fill="#6b8aa8" opacity="0.6"/>
      <circle cx="26" cy="30" r="2" fill="#6b8aa8" opacity="0.5"/>
      <circle cx="34" cy="34" r="1.5" fill="#6b8aa8" opacity="0.3"/>
      <circle cx="18" cy="34" r="1.5" fill="#6b8aa8" opacity="0.3"/>
      <line x1="26" y1="12" x2="34" y2="22" stroke="#6b8aa8" stroke-width="0.8" opacity="0.4"/>
      <line x1="26" y1="12" x2="18" y2="22" stroke="#6b8aa8" stroke-width="0.8" opacity="0.4"/>
      <line x1="34" y1="22" x2="26" y2="30" stroke="#6b8aa8" stroke-width="0.6" opacity="0.3"/>
      <line x1="18" y1="22" x2="26" y2="30" stroke="#6b8aa8" stroke-width="0.6" opacity="0.3"/>
    </template>

    <!-- 天芮 · 坤 · 土 — 双星团横列 -->
    <template v-if="starType === 'tianrui'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f6f0e4"/><stop offset="100%" stop-color="#e8dcc8"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a980" stroke-width="1.1"/>
      <circle cx="18" cy="14" r="2.5" fill="#b8956a"/>
      <circle cx="34" cy="14" r="2.5" fill="#b8956a"/>
      <circle cx="14" cy="26" r="2" fill="#b8956a" opacity="0.7"/>
      <circle cx="26" cy="24" r="3" fill="#b8956a"/>
      <circle cx="38" cy="26" r="2" fill="#b8956a" opacity="0.7"/>
      <circle cx="18" cy="36" r="2" fill="#b8956a" opacity="0.6"/>
      <circle cx="34" cy="36" r="2" fill="#b8956a" opacity="0.6"/>
      <line x1="18" y1="14" x2="26" y2="24" stroke="#b8956a" stroke-width="0.8" opacity="0.35"/>
      <line x1="34" y1="14" x2="26" y2="24" stroke="#b8956a" stroke-width="0.8" opacity="0.35"/>
    </template>

    <!-- 天冲 · 震 · 木 — 垂直流星光柱 -->
    <template v-if="starType === 'tianchong'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f2f8f0"/><stop offset="100%" stop-color="#ddecd6"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#7a9e7e" stroke-width="1.1"/>
      <circle cx="26" cy="38" r="2" fill="#7a9e7e" opacity="0.4"/>
      <circle cx="26" cy="28" r="2.2" fill="#7a9e7e" opacity="0.6"/>
      <circle cx="26" cy="18" r="2.5" fill="#7a9e7e"/>
      <circle cx="26" cy="8" r="3.5" fill="#7a9e7e"/>
      <line x1="26" y1="8" x2="26" y2="38" stroke="#7a9e7e" stroke-width="1" opacity="0.45"/>
      <circle cx="16" cy="20" r="1.5" fill="#7a9e7e" opacity="0.3"/>
      <circle cx="36" cy="20" r="1.5" fill="#7a9e7e" opacity="0.3"/>
    </template>

    <!-- 天辅 · 巽 · 木 — 双弧星轨 -->
    <template v-if="starType === 'tianfu'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f2f8f0"/><stop offset="100%" stop-color="#ddecd6"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#7a9e7e" stroke-width="1.1"/>
      <circle cx="10" cy="36" r="2" fill="#7a9e7e" opacity="0.5"/>
      <circle cx="18" cy="22" r="2.2" fill="#7a9e7e"/>
      <circle cx="26" cy="10" r="3" fill="#7a9e7e"/>
      <circle cx="34" cy="22" r="2.2" fill="#7a9e7e"/>
      <circle cx="42" cy="36" r="2" fill="#7a9e7e" opacity="0.5"/>
      <path d="M10,36 Q18,18 26,10 Q34,18 42,36" fill="none" stroke="#7a9e7e" stroke-width="1" stroke-linecap="round" opacity="0.55"/>
    </template>

    <!-- 天禽 · 中 · 土 — 十字准星 -->
    <template v-if="starType === 'tianqin'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f6f0e4"/><stop offset="100%" stop-color="#e8dcc8"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a980" stroke-width="1.1"/>
      <circle cx="26" cy="8" r="2" fill="#b8956a" opacity="0.4"/>
      <circle cx="26" cy="44" r="2" fill="#b8956a" opacity="0.4"/>
      <circle cx="8" cy="26" r="2" fill="#b8956a" opacity="0.4"/>
      <circle cx="44" cy="26" r="2" fill="#b8956a" opacity="0.4"/>
      <circle cx="26" cy="26" r="4" fill="#f6f0e4" stroke="#b8956a" stroke-width="2"/>
      <line x1="26" y1="8" x2="26" y2="44" stroke="#b8956a" stroke-width="0.8" opacity="0.3"/>
      <line x1="8" y1="26" x2="44" y2="26" stroke="#b8956a" stroke-width="0.8" opacity="0.3"/>
    </template>

    <!-- 天心 · 乾 · 金 — 六芒星阵 -->
    <template v-if="starType === 'tianxin'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#faf8f0"/><stop offset="100%" stop-color="#efe4cc"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a96a" stroke-width="1.1"/>
      <circle cx="26" cy="6" r="2.5" fill="#b8956a"/>
      <circle cx="42" cy="18" r="2.2" fill="#b8956a"/>
      <circle cx="42" cy="34" r="2.2" fill="#b8956a"/>
      <circle cx="26" cy="46" r="2.5" fill="#b8956a"/>
      <circle cx="10" cy="34" r="2.2" fill="#b8956a"/>
      <circle cx="10" cy="18" r="2.2" fill="#b8956a"/>
      <polygon points="26,6 42,18 42,34 26,46 10,34 10,18" fill="none" stroke="#b8956a" stroke-width="1" stroke-linejoin="round" opacity="0.5"/>
      <circle cx="26" cy="26" r="3.5" fill="#b8956a"/>
    </template>

    <!-- 天柱 · 兑 · 金 — 双柱星列 -->
    <template v-if="starType === 'tianzhu'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#faf8f0"/><stop offset="100%" stop-color="#efe4cc"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a96a" stroke-width="1.1"/>
      <circle cx="18" cy="8" r="2.5" fill="#b8956a"/>
      <circle cx="34" cy="8" r="2.5" fill="#b8956a"/>
      <circle cx="18" cy="20" r="2" fill="#b8956a" opacity="0.6"/>
      <circle cx="34" cy="20" r="2" fill="#b8956a" opacity="0.6"/>
      <circle cx="18" cy="32" r="2" fill="#b8956a" opacity="0.5"/>
      <circle cx="34" cy="32" r="2" fill="#b8956a" opacity="0.5"/>
      <circle cx="18" cy="44" r="2.5" fill="#b8956a"/>
      <circle cx="34" cy="44" r="2.5" fill="#b8956a"/>
      <line x1="18" y1="8" x2="18" y2="44" stroke="#b8956a" stroke-width="0.9" opacity="0.45"/>
      <line x1="34" y1="8" x2="34" y2="44" stroke="#b8956a" stroke-width="0.9" opacity="0.45"/>
    </template>

    <!-- 天任 · 艮 · 土 — 山形星图 -->
    <template v-if="starType === 'tianren'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f6f0e4"/><stop offset="100%" stop-color="#e8dcc8"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#c4a980" stroke-width="1.1"/>
      <circle cx="8" cy="38" r="2.2" fill="#b8956a"/>
      <circle cx="16" cy="26" r="2.5" fill="#b8956a"/>
      <circle cx="26" cy="14" r="3" fill="#b8956a"/>
      <circle cx="36" cy="26" r="2.5" fill="#b8956a"/>
      <circle cx="44" cy="38" r="2.2" fill="#b8956a"/>
      <polyline points="8,38 16,26 26,14 36,26 44,38" fill="none" stroke="#b8956a" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" opacity="0.55"/>
    </template>

    <!-- 天英 · 离 · 火 — 放射光芒星 -->
    <template v-if="starType === 'tianying'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fef8f6"/><stop offset="100%" stop-color="#f8e0d4"/></linearGradient></defs>
      <rect x="2" y="2" width="48" height="48" rx="12" :fill="`url(#${gid})`" stroke="#cc9888" stroke-width="1.1"/>
      <circle cx="26" cy="26" r="4" fill="#c47a6a"/>
      <circle cx="26" cy="8" r="2.5" fill="#c47a6a" opacity="0.7"/>
      <circle cx="26" cy="44" r="2.5" fill="#c47a6a" opacity="0.7"/>
      <circle cx="8" cy="26" r="2.5" fill="#c47a6a" opacity="0.7"/>
      <circle cx="44" cy="26" r="2.5" fill="#c47a6a" opacity="0.7"/>
      <line x1="26" y1="8" x2="26" y2="44" stroke="#c47a6a" stroke-width="0.8" opacity="0.35"/>
      <line x1="8" y1="26" x2="44" y2="26" stroke="#c47a6a" stroke-width="0.8" opacity="0.35"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  starType: 'tianpeng' | 'tianying' | 'tianchong' | 'tianfu' | 'tianqin' | 'tianxin' | 'tianzhu' | 'tianren' | 'tianrui'
  size?: number
}>(), { size: 48 })

const gid = `ss-${Math.random().toString(36).slice(2, 8)}`
</script>
