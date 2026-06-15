<template>
  <svg :width="size" :height="size" viewBox="0 0 40 40">
    <!-- 天蓬 — 水，波纹圆环 -->
    <template v-if="starType === 'tianpeng'">
      <circle cx="20" cy="20" r="18" fill="#e8eef4" stroke="#6b8aa8" stroke-width="1"/>
      <circle cx="20" cy="20" r="12" fill="none" stroke="#7a9ab8" stroke-width="0.7"/>
      <circle cx="20" cy="20" r="7" fill="none" stroke="#7a9ab8" stroke-width="0.6"/>
      <path d="M10 16 Q20 10 30 16" stroke="#5a7a98" stroke-width="1" fill="none" stroke-linecap="round"/>
      <path d="M8 22 Q20 15 32 22" stroke="#5a7a98" stroke-width="0.7" fill="none" stroke-linecap="round"/>
      <circle cx="20" cy="20" r="2.5" fill="#faf8f5" stroke="#6b8aa8" stroke-width="0.8"/>
    </template>

    <!-- 天英 — 火，放射菱形 -->
    <template v-if="starType === 'tianying'">
      <circle cx="20" cy="20" r="18" fill="#faeae4" stroke="#c47a6a" stroke-width="1"/>
      <polygon points="20,1 29,13 38,20 29,27 20,39 11,27 2,20 11,13" fill="#f5d8cc" stroke="#c47a6a" stroke-width="0.9" stroke-linejoin="round"/>
      <polygon points="20,7 26,15 32,20 26,25 20,33 14,25 8,20 14,15" fill="#f0c8b4" stroke="#b86858" stroke-width="0.6" stroke-linejoin="round"/>
      <circle cx="20" cy="20" r="2.5" fill="#faf8f5" stroke="#c47a6a" stroke-width="0.8"/>
      <circle cx="8" cy="12" r="1.2" fill="#f0d0c0" stroke="#c47a6a" stroke-width="0.4"/>
      <circle cx="32" cy="28" r="1" fill="#f0d0c0" stroke="#c47a6a" stroke-width="0.4"/>
    </template>

    <!-- 天冲 — 木，上冲枝芽 -->
    <template v-if="starType === 'tianchong'">
      <circle cx="20" cy="20" r="18" fill="#ecf2ec" stroke="#7a9e7e" stroke-width="1"/>
      <path d="M20 36 Q18 24 20 12" stroke="#6a8e6e" stroke-width="1.2" fill="none" stroke-linecap="round"/>
      <path d="M20 18 Q13 10 7 6" stroke="#7a9e7e" stroke-width="1" fill="none" stroke-linecap="round"/>
      <path d="M20 18 Q27 12 33 6" stroke="#7a9e7e" stroke-width="0.9" fill="none" stroke-linecap="round"/>
      <ellipse cx="7" cy="5" rx="4" ry="2.2" fill="#e0ecd8" stroke="#7a9e7e" stroke-width="0.6" transform="rotate(-25 7 5)"/>
      <ellipse cx="33" cy="5" rx="3.5" ry="2" fill="#e0ecd8" stroke="#7a9e7e" stroke-width="0.6" transform="rotate(20 33 5)"/>
      <path d="M20 36 Q23 30 27 28" stroke="#6a8e6e" stroke-width="0.7" fill="none" stroke-linecap="round"/>
      <circle cx="20" cy="12" r="2.2" fill="#faf8f5" stroke="#7a9e7e" stroke-width="0.7"/>
    </template>

    <!-- 天辅 — 木，环抱双弧 -->
    <template v-if="starType === 'tianfu'">
      <circle cx="20" cy="20" r="18" fill="#ecf2ec" stroke="#7a9e7e" stroke-width="1"/>
      <path d="M6 26 Q14 4 20 12 Q26 4 34 26" stroke="#7a9e7e" stroke-width="1.1" fill="#e0ecd8" stroke-linecap="round"/>
      <path d="M10 28 Q15 12 20 18 Q25 12 30 28" stroke="#7a9e7e" stroke-width="0.7" fill="none" stroke-linecap="round"/>
      <line x1="16" y1="30" x2="16" y2="37" stroke="#7a9e7e" stroke-width="0.9" stroke-linecap="round"/>
      <line x1="24" y1="30" x2="24" y2="37" stroke="#7a9e7e" stroke-width="0.9" stroke-linecap="round"/>
      <circle cx="20" cy="22" r="2.5" fill="#faf8f5" stroke="#7a9e7e" stroke-width="0.7"/>
    </template>

    <!-- 天禽 — 土，十字平衡 -->
    <template v-if="starType === 'tianqin'">
      <circle cx="20" cy="20" r="18" fill="#f2ece0" stroke="#b8956a" stroke-width="1"/>
      <circle cx="20" cy="20" r="12" stroke="#b8956a" stroke-width="0.6" fill="none"/>
      <line x1="8" y1="20" x2="32" y2="20" stroke="#b8956a" stroke-width="1.2" stroke-linecap="round"/>
      <line x1="20" y1="8" x2="20" y2="32" stroke="#b8956a" stroke-width="1.2" stroke-linecap="round"/>
      <circle cx="20" cy="20" r="3.5" fill="#faf8f5" stroke="#b8956a" stroke-width="0.9"/>
      <circle cx="20" cy="20" r="1.5" fill="#c4a96a"/>
      <!-- 四角小点 -->
      <circle cx="12" cy="12" r="1.2" fill="#d4c0a0" stroke="#b8956a" stroke-width="0.4"/>
      <circle cx="28" cy="12" r="1.2" fill="#d4c0a0" stroke="#b8956a" stroke-width="0.4"/>
      <circle cx="12" cy="28" r="1.2" fill="#d4c0a0" stroke="#b8956a" stroke-width="0.4"/>
      <circle cx="28" cy="28" r="1.2" fill="#d4c0a0" stroke="#b8956a" stroke-width="0.4"/>
    </template>

    <!-- 天心 — 金，六角同心 -->
    <template v-if="starType === 'tianxin'">
      <circle cx="20" cy="20" r="18" fill="#f6f2e4" stroke="#c4a96a" stroke-width="1"/>
      <polygon points="20,1 37,12 37,28 20,39 3,28 3,12" fill="#efe4cc" stroke="#c4a96a" stroke-width="0.9" stroke-linejoin="round"/>
      <polygon points="20,7 33,14 33,26 20,33 7,26 7,14" fill="#e8d8b8" stroke="#b89860" stroke-width="0.6" stroke-linejoin="round"/>
      <circle cx="20" cy="20" r="3" fill="#faf8f5" stroke="#c4a96a" stroke-width="0.8"/>
      <line x1="20" y1="7" x2="20" y2="33" stroke="#c4a96a" stroke-width="0.3" opacity="0.3"/>
      <line x1="7" y1="14" x2="33" y2="26" stroke="#c4a96a" stroke-width="0.3" opacity="0.3"/>
      <line x1="7" y1="26" x2="33" y2="14" stroke="#c4a96a" stroke-width="0.3" opacity="0.3"/>
    </template>

    <!-- 天柱 — 金，竖柱 -->
    <template v-if="starType === 'tianzhu'">
      <circle cx="20" cy="20" r="18" fill="#f6f2e4" stroke="#c4a96a" stroke-width="1"/>
      <rect x="14" y="3" width="12" height="34" rx="3" fill="#efe4cc" stroke="#c4a96a" stroke-width="0.9"/>
      <line x1="17" y1="7" x2="17" y2="33" stroke="#c4a96a" stroke-width="0.5" opacity="0.4"/>
      <line x1="23" y1="7" x2="23" y2="33" stroke="#c4a96a" stroke-width="0.5" opacity="0.4"/>
      <rect x="16" y="16" width="8" height="3" rx="1" fill="#faf8f5" stroke="#c4a96a" stroke-width="0.6"/>
      <circle cx="20" cy="5" r="2" fill="#faf8f5" stroke="#c4a96a" stroke-width="0.7"/>
    </template>

    <!-- 天任 — 土，负重山形 -->
    <template v-if="starType === 'tianren'">
      <circle cx="20" cy="20" r="18" fill="#f2ece0" stroke="#b8956a" stroke-width="1"/>
      <path d="M3 33 L14 13 L26 13 L37 33" fill="#e8dcc8" stroke="#b8956a" stroke-width="1" stroke-linejoin="round"/>
      <path d="M9 33 L16 19 L24 19 L31 33" fill="#ddd0b8" stroke="#a08060" stroke-width="0.6" stroke-linejoin="round"/>
      <line x1="20" y1="13" x2="20" y2="33" stroke="#b8956a" stroke-width="0.4" opacity="0.3"/>
      <rect x="12" y="30" width="4" height="2" rx="0.5" fill="#d4c0a0" stroke="#b8956a" stroke-width="0.4"/>
      <rect x="24" y="28" width="3" height="2" rx="0.5" fill="#d4c0a0" stroke="#b8956a" stroke-width="0.4"/>
      <circle cx="20" cy="7" r="2.5" fill="#faf8f5" stroke="#b8956a" stroke-width="0.8"/>
    </template>

    <!-- 天芮 — 土，萌芽种 -->
    <template v-if="starType === 'tianrui'">
      <circle cx="20" cy="20" r="18" fill="#f2ece0" stroke="#b8956a" stroke-width="1"/>
      <circle cx="20" cy="23" r="11" fill="#e8dcc8" stroke="#b8956a" stroke-width="0.8"/>
      <path d="M20 12 Q15 5 20 2 Q25 5 20 12" fill="#d4c0a0" stroke="#a08060" stroke-width="0.8"/>
      <path d="M15 20 Q13 13 17 9" stroke="#b8956a" stroke-width="0.7" fill="none" stroke-linecap="round"/>
      <path d="M25 20 Q27 13 23 9" stroke="#b8956a" stroke-width="0.7" fill="none" stroke-linecap="round"/>
      <line x1="20" y1="16" x2="20" y2="34" stroke="#b8956a" stroke-width="0.5" opacity="0.3"/>
      <circle cx="20" cy="23" r="2" fill="#faf8f5" stroke="#b8956a" stroke-width="0.7"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  starType: 'tianpeng' | 'tianying' | 'tianchong' | 'tianfu' | 'tianqin' | 'tianxin' | 'tianzhu' | 'tianren' | 'tianrui'
  size?: number
}>(), { size: 40 })
</script>
