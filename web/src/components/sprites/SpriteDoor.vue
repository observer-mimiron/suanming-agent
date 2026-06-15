<template>
  <svg :width="size" :height="size" viewBox="0 0 40 40">
    <!-- 开门 — 大开 -->
    <template v-if="doorType === 'kai'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#f8f4e8" stroke="#c4a96a" stroke-width="1.5"/>
      <path d="M8 8 L20 4 L32 8" fill="none" stroke="#c4a96a" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
      <line x1="8" y1="8" x2="8" y2="32" stroke="#c4a96a" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="32" y1="8" x2="32" y2="32" stroke="#c4a96a" stroke-width="1.5" stroke-linecap="round"/>
    </template>

    <!-- 休门 — 虚掩 -->
    <template v-if="doorType === 'xiu'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#e8eef4" stroke="#6b8aa8" stroke-width="1.5"/>
      <rect x="8" y="8" width="18" height="24" rx="3" fill="#d8e4ee" stroke="#6b8aa8" stroke-width="1.5"/>
      <line x1="8" y1="8" x2="20" y2="6" stroke="#6b8aa8" stroke-width="1.5" stroke-linecap="round"/>
    </template>

    <!-- 生门 — 推开 -->
    <template v-if="doorType === 'sheng'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#f0e8d8" stroke="#b8956a" stroke-width="1.5"/>
      <rect x="8" y="8" width="18" height="24" rx="3" fill="#e0d0b8" stroke="#b8956a" stroke-width="1.5"/>
      <line x1="8" y1="8" x2="18" y2="4" stroke="#b8956a" stroke-width="2" stroke-linecap="round"/>
      <path d="M28 12 L32 12 M28 18 L34 18 M28 24 L32 24" stroke="#b8956a" stroke-width="1" stroke-linecap="round" opacity="0.4"/>
    </template>

    <!-- 伤门 — 碎裂 -->
    <template v-if="doorType === 'shang'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#e8f0e4" stroke="#7a9e7e" stroke-width="1.5"/>
      <line x1="10" y1="20" x2="30" y2="18" stroke="#7a9e7e" stroke-width="2.5" stroke-linecap="round"/>
      <line x1="12" y1="12" x2="28" y2="10" stroke="#7a9e7e" stroke-width="2" stroke-linecap="round"/>
      <line x1="14" y1="28" x2="26" y2="30" stroke="#7a9e7e" stroke-width="1.5" stroke-linecap="round"/>
      <circle cx="10" cy="14" r="2" fill="#7a9e7e" opacity="0.3"/>
      <circle cx="28" cy="26" r="1.5" fill="#7a9e7e" opacity="0.2"/>
    </template>

    <!-- 杜门 — 封锁 -->
    <template v-if="doorType === 'du'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#e8f0e4" stroke="#7a9e7e" stroke-width="1.5"/>
      <line x1="10" y1="10" x2="30" y2="30" stroke="#7a9e7e" stroke-width="2.5" stroke-linecap="round"/>
      <line x1="30" y1="10" x2="10" y2="30" stroke="#7a9e7e" stroke-width="2.5" stroke-linecap="round"/>
      <circle cx="20" cy="20" r="3" fill="#faf8f5" stroke="#7a9e7e" stroke-width="1.2"/>
    </template>

    <!-- 景门 — 开窗 -->
    <template v-if="doorType === 'jing'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#fae8e0" stroke="#c47a6a" stroke-width="1.5"/>
      <rect x="8" y="10" width="18" height="20" rx="3" fill="#f5d8cc" stroke="#c47a6a" stroke-width="1.5"/>
      <line x1="8" y1="10" x2="18" y2="5" stroke="#c47a6a" stroke-width="2" stroke-linecap="round"/>
      <circle cx="30" cy="20" r="5" fill="none" stroke="#c47a6a" stroke-width="1.2" opacity="0.4"/>
    </template>

    <!-- 死门 — 密封 -->
    <template v-if="doorType === 'si'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#f0e8d8" stroke="#b8956a" stroke-width="1.5"/>
      <rect x="9" y="9" width="22" height="22" rx="2" fill="#e0d0b8" stroke="#b8956a" stroke-width="1.5"/>
      <circle cx="20" cy="20" r="3" fill="#b8956a"/>
    </template>

    <!-- 惊门 — 半阻 -->
    <template v-if="doorType === 'jingmen'">
      <rect x="2" y="2" width="36" height="36" rx="6" fill="#f8f4e8" stroke="#c4a96a" stroke-width="1.5"/>
      <line x1="20" y1="6" x2="20" y2="34" stroke="#c4a96a" stroke-width="2.5" stroke-linecap="round"/>
      <line x1="12" y1="14" x2="20" y2="14" stroke="#c4a96a" stroke-width="2" stroke-linecap="round"/>
      <line x1="12" y1="22" x2="20" y2="22" stroke="#c4a96a" stroke-width="2" stroke-linecap="round"/>
      <line x1="28" y1="12" x2="28" y2="28" stroke="#c4a96a" stroke-width="1.5" stroke-linecap="round" opacity="0.4"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  doorType: 'kai' | 'xiu' | 'sheng' | 'shang' | 'du' | 'jing' | 'si' | 'jingmen'
  size?: number
}>(), { size: 40 })
</script>
