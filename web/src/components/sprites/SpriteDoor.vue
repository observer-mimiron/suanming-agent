<template>
  <svg :width="size" :height="size" viewBox="0 0 40 40">
    <!-- 开门 — 金属，双扇大开 -->
    <template v-if="doorType === 'kai'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#f8f4e8" stroke="#c4a96a" stroke-width="1"/>
      <rect x="8" y="7" width="10" height="26" rx="1" fill="#efe4cc" stroke="#c4a96a" stroke-width="0.8"/>
      <rect x="22" y="7" width="10" height="26" rx="1" fill="#efe4cc" stroke="#c4a96a" stroke-width="0.8"/>
      <line x1="8" y1="7" x2="20" y2="4" stroke="#c4a96a" stroke-width="0.9" stroke-linecap="round"/>
      <line x1="32" y1="7" x2="20" y2="4" stroke="#c4a96a" stroke-width="0.9" stroke-linecap="round"/>
      <circle cx="20" cy="22" r="2" fill="#faf8f5" stroke="#c4a96a" stroke-width="0.7"/>
    </template>

    <!-- 休门 — 水，半掩 -->
    <template v-if="doorType === 'xiu'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#e8eef4" stroke="#6b8aa8" stroke-width="1"/>
      <rect x="8" y="7" width="14" height="26" rx="1" fill="#d8e4ee" stroke="#6b8aa8" stroke-width="0.8"/>
      <line x1="22" y1="7" x2="17" y2="7" stroke="#6b8aa8" stroke-width="0.7"/>
      <line x1="22" y1="33" x2="17" y2="33" stroke="#6b8aa8" stroke-width="0.7"/>
      <path d="M8 7 Q14 20 8 33" stroke="#6b8aa8" stroke-width="0.8" fill="none" stroke-linecap="round"/>
      <circle cx="14" cy="22" r="2" fill="#faf8f5" stroke="#6b8aa8" stroke-width="0.7"/>
    </template>

    <!-- 生门 — 土，推开门缝 -->
    <template v-if="doorType === 'sheng'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#f2ece0" stroke="#b8956a" stroke-width="1"/>
      <rect x="8" y="7" width="16" height="26" rx="1" fill="#e8dcc8" stroke="#b8956a" stroke-width="0.8"/>
      <line x1="8" y1="7" x2="16" y2="4" stroke="#b8956a" stroke-width="1" stroke-linecap="round"/>
      <path d="M24 7 L24 33" stroke="#b8956a" stroke-width="0.5" stroke-dasharray="2 3"/>
      <!-- 光透出来 -->
      <line x1="26" y1="12" x2="30" y2="12" stroke="#b8956a" stroke-width="0.6" opacity="0.4"/>
      <line x1="26" y1="18" x2="32" y2="18" stroke="#b8956a" stroke-width="0.6" opacity="0.4"/>
      <line x1="26" y1="24" x2="30" y2="24" stroke="#b8956a" stroke-width="0.6" opacity="0.3"/>
      <circle cx="12" cy="22" r="1.8" fill="#faf8f5" stroke="#b8956a" stroke-width="0.7"/>
    </template>

    <!-- 伤门 — 木，碎裂/不规则封锁 -->
    <template v-if="doorType === 'shang'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#ecf2ec" stroke="#7a9e7e" stroke-width="1"/>
      <rect x="8" y="7" width="14" height="26" rx="1" fill="#e0ecd8" stroke="#7a9e7e" stroke-width="0.8"/>
      <!-- 碎片线 -->
      <line x1="8" y1="14" x2="22" y2="12" stroke="#7a9e7e" stroke-width="0.8" stroke-linecap="round"/>
      <line x1="10" y1="22" x2="22" y2="20" stroke="#7a9e7e" stroke-width="0.7" stroke-linecap="round"/>
      <line x1="8" y1="30" x2="22" y2="28" stroke="#7a9e7e" stroke-width="0.6" stroke-linecap="round"/>
      <path d="M22 8 L28 18 L22 28" stroke="#7a9e7e" stroke-width="0.8" fill="none" stroke-linejoin="round"/>
      <circle cx="11" cy="18" r="1.5" fill="#faf8f5" stroke="#7a9e7e" stroke-width="0.6"/>
    </template>

    <!-- 杜门 — 木，闭合十字 -->
    <template v-if="doorType === 'du'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#ecf2ec" stroke="#7a9e7e" stroke-width="1"/>
      <rect x="8" y="7" width="24" height="26" rx="1" fill="#d8e8d4" stroke="#7a9e7e" stroke-width="0.8"/>
      <!-- 锁/十字 -->
      <line x1="14" y1="12" x2="26" y2="28" stroke="#7a9e7e" stroke-width="1.2" stroke-linecap="round"/>
      <line x1="26" y1="12" x2="14" y2="28" stroke="#7a9e7e" stroke-width="1.2" stroke-linecap="round"/>
      <circle cx="20" cy="20" r="3" fill="#faf8f5" stroke="#7a9e7e" stroke-width="0.8"/>
    </template>

    <!-- 景门 — 火，半开观景 -->
    <template v-if="doorType === 'jing'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#faeae4" stroke="#c47a6a" stroke-width="1"/>
      <rect x="8" y="7" width="12" height="26" rx="1" fill="#f5d8cc" stroke="#c47a6a" stroke-width="0.8"/>
      <!-- 开着的门扇 -->
      <line x1="8" y1="7" x2="16" y2="3" stroke="#c47a6a" stroke-width="1" stroke-linecap="round"/>
      <!-- 透出的景 -->
      <circle cx="28" cy="15" r="5" fill="none" stroke="#c47a6a" stroke-width="0.6" opacity="0.4"/>
      <path d="M24 18 L32 18" stroke="#c47a6a" stroke-width="0.6" opacity="0.3"/>
      <circle cx="28" cy="22" r="1.5" fill="#faf8f5" stroke="#c47a6a" stroke-width="0.7"/>
    </template>

    <!-- 死门 — 土，完全封闭 -->
    <template v-if="doorType === 'si'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#f2ece0" stroke="#b8956a" stroke-width="1"/>
      <rect x="8" y="7" width="24" height="26" rx="1" fill="#e0d4c0" stroke="#b8956a" stroke-width="0.8"/>
      <!-- 密封横杠 -->
      <line x1="12" y1="14" x2="28" y2="14" stroke="#b8956a" stroke-width="1.1" stroke-linecap="round"/>
      <line x1="12" y1="20" x2="28" y2="20" stroke="#b8956a" stroke-width="1.1" stroke-linecap="round"/>
      <line x1="12" y1="26" x2="28" y2="26" stroke="#b8956a" stroke-width="1.1" stroke-linecap="round"/>
    </template>

    <!-- 惊门 — 金，半阻/不规则 -->
    <template v-if="doorType === 'jingmen'">
      <rect x="2" y="2" width="36" height="36" rx="5" fill="#f8f4e8" stroke="#c4a96a" stroke-width="1"/>
      <rect x="8" y="7" width="16" height="26" rx="1" fill="#efe4cc" stroke="#c4a96a" stroke-width="0.8"/>
      <!-- 不规则遮挡线 -->
      <path d="M8 11 L12 14 L10 18 L14 21 L8 24" stroke="#c4a96a" stroke-width="0.9" fill="none" stroke-linejoin="round"/>
      <path d="M8 28 L13 30 L8 33" stroke="#c4a96a" stroke-width="0.8" fill="none" stroke-linejoin="round"/>
      <line x1="24" y1="7" x2="20" y2="18" stroke="#c4a96a" stroke-width="0.7" stroke-linecap="round"/>
      <line x1="26" y1="22" x2="20" y2="33" stroke="#c4a96a" stroke-width="0.6" stroke-linecap="round"/>
      <circle cx="12" cy="20" r="1.5" fill="#faf8f5" stroke="#c4a96a" stroke-width="0.7"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  doorType: 'kai' | 'xiu' | 'sheng' | 'shang' | 'du' | 'jing' | 'si' | 'jingmen'
  size?: number
}>(), { size: 40 })
</script>
