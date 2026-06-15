<template>
  <svg :width="size" :height="size" viewBox="0 0 40 40">
    <!-- 天蓬 — 水 · 同心圆 -->
    <template v-if="starType === 'tianpeng'">
      <circle cx="20" cy="20" r="18" fill="#e4ecf4" stroke="#6b8aa8" stroke-width="1.5"/>
      <circle cx="20" cy="20" r="9" fill="none" stroke="#6b8aa8" stroke-width="1.5"/>
      <circle cx="20" cy="20" r="3" fill="#6b8aa8"/>
    </template>

    <!-- 天英 — 火 · 八角星 -->
    <template v-if="starType === 'tianying'">
      <circle cx="20" cy="20" r="18" fill="#fae8e0" stroke="#c47a6a" stroke-width="1.5"/>
      <polygon points="20,2 26,10 35,12 28,19 30,28 20,24 10,28 12,19 5,12 14,10" fill="#f0c0a0" stroke="#c47a6a" stroke-width="1.2" stroke-linejoin="round"/>
      <circle cx="20" cy="20" r="3" fill="#c47a6a"/>
    </template>

    <!-- 天冲 — 木 · 上冲箭头 -->
    <template v-if="starType === 'tianchong'">
      <circle cx="20" cy="20" r="18" fill="#e8f0e4" stroke="#7a9e7e" stroke-width="1.5"/>
      <path d="M20 34 L20 10" stroke="#7a9e7e" stroke-width="2.5" stroke-linecap="round"/>
      <path d="M12 18 L20 6 L28 18" fill="none" stroke="#7a9e7e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    </template>

    <!-- 天辅 — 木 · 对称双弧 -->
    <template v-if="starType === 'tianfu'">
      <circle cx="20" cy="20" r="18" fill="#e8f0e4" stroke="#7a9e7e" stroke-width="1.5"/>
      <path d="M6 30 Q18 6 20 18 Q22 6 34 30" fill="none" stroke="#7a9e7e" stroke-width="2.5" stroke-linecap="round"/>
      <circle cx="20" cy="24" r="3" fill="#7a9e7e"/>
    </template>

    <!-- 天禽 — 土 · 十字准星 -->
    <template v-if="starType === 'tianqin'">
      <circle cx="20" cy="20" r="18" fill="#f0e8d8" stroke="#b8956a" stroke-width="1.5"/>
      <line x1="6" y1="20" x2="34" y2="20" stroke="#b8956a" stroke-width="2.5" stroke-linecap="round"/>
      <line x1="20" y1="6" x2="20" y2="34" stroke="#b8956a" stroke-width="2.5" stroke-linecap="round"/>
      <circle cx="20" cy="20" r="4" fill="#faf8f5" stroke="#b8956a" stroke-width="1.5"/>
    </template>

    <!-- 天心 — 金 · 六边形 -->
    <template v-if="starType === 'tianxin'">
      <circle cx="20" cy="20" r="18" fill="#f6f0e0" stroke="#c4a96a" stroke-width="1.5"/>
      <polygon points="20,2 36,12 36,28 20,38 4,28 4,12" fill="#efe0c0" stroke="#c4a96a" stroke-width="1.5" stroke-linejoin="round"/>
      <circle cx="20" cy="20" r="3" fill="#c4a96a"/>
    </template>

    <!-- 天柱 — 金 · 竖柱 -->
    <template v-if="starType === 'tianzhu'">
      <circle cx="20" cy="20" r="18" fill="#f6f0e0" stroke="#c4a96a" stroke-width="1.5"/>
      <rect x="14" y="6" width="12" height="28" rx="4" fill="#efe0c0" stroke="#c4a96a" stroke-width="1.5"/>
      <line x1="17" y1="12" x2="17" y2="28" stroke="#c4a96a" stroke-width="1" stroke-linecap="round"/>
      <line x1="23" y1="12" x2="23" y2="28" stroke="#c4a96a" stroke-width="1" stroke-linecap="round"/>
    </template>

    <!-- 天任 — 土 · 梯形山 -->
    <template v-if="starType === 'tianren'">
      <circle cx="20" cy="20" r="18" fill="#f0e8d8" stroke="#b8956a" stroke-width="1.5"/>
      <polygon points="6,34 14,14 26,14 34,34" fill="#e0d0b8" stroke="#b8956a" stroke-width="1.5" stroke-linejoin="round"/>
      <circle cx="20" cy="24" r="3" fill="#b8956a"/>
    </template>

    <!-- 天芮 — 土 · 种子 -->
    <template v-if="starType === 'tianrui'">
      <circle cx="20" cy="20" r="18" fill="#f0e8d8" stroke="#b8956a" stroke-width="1.5"/>
      <ellipse cx="20" cy="26" rx="10" ry="10" fill="#e0d0b8" stroke="#b8956a" stroke-width="1.5"/>
      <path d="M20 16 Q14 8 20 4 Q26 8 20 16" fill="#d4c0a0" stroke="#b8956a" stroke-width="1.2" stroke-linecap="round"/>
      <circle cx="20" cy="22" r="2.5" fill="#b8956a"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  starType: 'tianpeng' | 'tianying' | 'tianchong' | 'tianfu' | 'tianqin' | 'tianxin' | 'tianzhu' | 'tianren' | 'tianrui'
  size?: number
}>(), { size: 40 })
</script>
