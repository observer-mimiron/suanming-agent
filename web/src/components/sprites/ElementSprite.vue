<template>
  <svg :width="size" :height="size" viewBox="0 0 72 72" v-bind="$attrs">
    <!-- 木 · 仁 — 参天大树 -->
    <template v-if="element === 'wood'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fafdf8"/><stop offset="100%" stop-color="#e2eedc"/></linearGradient></defs>
      <rect x="2" y="2" width="68" height="68" rx="17" :fill="`url(#${gid})`" stroke="#7a9e7e" stroke-width="1.2"/>
      <path d="M2,48 Q20,38 38,48 Q54,38 70,48" fill="none" stroke="#7a9e7e" stroke-width="0.9" opacity="0.18"/>
      <line x1="6" y1="50" x2="66" y2="50" stroke="#7a9e7e" stroke-width="1.3" stroke-linecap="round"/>
      <path d="M12,50 Q14,46 16,50" fill="none" stroke="#7a9e7e" stroke-width="0.7" opacity="0.3"/>
      <path d="M18,50 Q19,47 20,50" fill="none" stroke="#7a9e7e" stroke-width="0.7" opacity="0.25"/>
      <path d="M52,50 Q54,46 56,50" fill="none" stroke="#7a9e7e" stroke-width="0.7" opacity="0.3"/>
      <path d="M28,50 Q26,54 22,56" fill="none" stroke="#7a9e7e" stroke-width="1.1" stroke-linecap="round" opacity="0.45"/>
      <path d="M36,50 Q38,54 42,56" fill="none" stroke="#7a9e7e" stroke-width="1.1" stroke-linecap="round" opacity="0.45"/>
      <path d="M32,50 Q32,56 34,56" fill="none" stroke="#7a9e7e" stroke-width="0.8" stroke-linecap="round" opacity="0.3"/>
      <path d="M32,50 Q33,36 32,24" fill="none" stroke="#7a9e7e" stroke-width="2.4" stroke-linecap="round"/>
      <path d="M32,36 Q24,28 18,22" fill="none" stroke="#7a9e7e" stroke-width="1.6" stroke-linecap="round"/>
      <path d="M32,36 Q40,28 46,22" fill="none" stroke="#7a9e7e" stroke-width="1.6" stroke-linecap="round"/>
      <path d="M32,30 Q28,22 32,14" fill="none" stroke="#7a9e7e" stroke-width="1.4" stroke-linecap="round"/>
      <path d="M32,30 Q36,20 32,10" fill="none" stroke="#7a9e7e" stroke-width="1" stroke-linecap="round" opacity="0.5"/>
      <path d="M24,26 Q20,22 16,24" fill="none" stroke="#7a9e7e" stroke-width="0.9" stroke-linecap="round" opacity="0.4"/>
      <path d="M40,26 Q44,22 48,24" fill="none" stroke="#7a9e7e" stroke-width="0.9" stroke-linecap="round" opacity="0.4"/>
      <circle cx="18" cy="20" r="4" fill="none" stroke="#7a9e7e" stroke-width="1.2" opacity="0.55"/>
      <circle cx="46" cy="20" r="4" fill="none" stroke="#7a9e7e" stroke-width="1.2" opacity="0.55"/>
      <circle cx="32" cy="12" r="5" fill="none" stroke="#7a9e7e" stroke-width="1.4" opacity="0.7"/>
      <circle cx="24" cy="16" r="3" fill="none" stroke="#7a9e7e" stroke-width="0.9" opacity="0.35"/>
      <circle cx="40" cy="16" r="3" fill="none" stroke="#7a9e7e" stroke-width="0.9" opacity="0.35"/>
      <circle cx="32" cy="6" r="3" fill="none" stroke="#7a9e7e" stroke-width="0.9" opacity="0.4"/>
      <circle cx="32" cy="10" r="2" fill="#7a9e7e" opacity="0.35"/>
      <circle cx="18" cy="18" r="1.5" fill="#7a9e7e" opacity="0.25"/>
      <circle cx="46" cy="18" r="1.5" fill="#7a9e7e" opacity="0.25"/>
      <ellipse cx="22" cy="42" rx="2.5" ry="1" fill="none" stroke="#7a9e7e" stroke-width="0.8" opacity="0.25" transform="rotate(-25 22 42)"/>
      <ellipse cx="48" cy="40" rx="2" ry="0.8" fill="none" stroke="#7a9e7e" stroke-width="0.7" opacity="0.2" transform="rotate(15 48 40)"/>
      <ellipse cx="56" cy="44" rx="1.8" ry="0.7" fill="none" stroke="#7a9e7e" stroke-width="0.6" opacity="0.15" transform="rotate(35 56 44)"/>
    </template>

    <!-- 火 · 礼 — 篝火太阳 -->
    <template v-if="element === 'fire'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fefcfa"/><stop offset="100%" stop-color="#fae4d8"/></linearGradient></defs>
      <rect x="2" y="2" width="68" height="68" rx="17" :fill="`url(#${gid})`" stroke="#cc9888" stroke-width="1.2"/>
      <circle cx="36" cy="30" r="18" fill="none" stroke="#c47a6a" stroke-width="0.6" opacity="0.1"/>
      <circle cx="36" cy="30" r="12" fill="none" stroke="#c47a6a" stroke-width="0.5" opacity="0.12"/>
      <line x1="36" y1="6" x2="36" y2="14" stroke="#c47a6a" stroke-width="0.6" opacity="0.15"/>
      <line x1="14" y1="20" x2="20" y2="24" stroke="#c47a6a" stroke-width="0.6" opacity="0.12"/>
      <line x1="58" y1="20" x2="52" y2="24" stroke="#c47a6a" stroke-width="0.6" opacity="0.12"/>
      <line x1="10" y1="34" x2="18" y2="34" stroke="#c47a6a" stroke-width="0.5" opacity="0.1"/>
      <line x1="62" y1="34" x2="54" y2="34" stroke="#c47a6a" stroke-width="0.5" opacity="0.1"/>
      <line x1="22" y1="52" x2="34" y2="44" stroke="#c47a6a" stroke-width="2" stroke-linecap="round" opacity="0.35"/>
      <line x1="50" y1="52" x2="38" y2="44" stroke="#c47a6a" stroke-width="2" stroke-linecap="round" opacity="0.35"/>
      <line x1="28" y1="54" x2="36" y2="46" stroke="#c47a6a" stroke-width="1.5" stroke-linecap="round" opacity="0.25"/>
      <line x1="44" y1="54" x2="36" y2="46" stroke="#c47a6a" stroke-width="1.5" stroke-linecap="round" opacity="0.25"/>
      <path d="M36,20 Q44,34 40,46" fill="none" stroke="#c47a6a" stroke-width="2.4" stroke-linecap="round"/>
      <path d="M36,20 Q28,34 32,46" fill="none" stroke="#c47a6a" stroke-width="2.4" stroke-linecap="round"/>
      <path d="M36,26 Q40,36 38,46" fill="none" stroke="#c47a6a" stroke-width="1.8" stroke-linecap="round" opacity="0.7"/>
      <path d="M36,26 Q32,36 34,46" fill="none" stroke="#c47a6a" stroke-width="1.8" stroke-linecap="round" opacity="0.7"/>
      <path d="M36,32 Q38,40 36,46" fill="none" stroke="#c47a6a" stroke-width="1.2" stroke-linecap="round" opacity="0.4"/>
      <path d="M36,32 Q34,40 36,46" fill="none" stroke="#c47a6a" stroke-width="1.2" stroke-linecap="round" opacity="0.4"/>
      <path d="M24,34 Q28,40 26,46" fill="none" stroke="#c47a6a" stroke-width="1.3" stroke-linecap="round" opacity="0.4"/>
      <path d="M28,30 Q30,36 30,44" fill="none" stroke="#c47a6a" stroke-width="1.1" stroke-linecap="round" opacity="0.3"/>
      <path d="M48,34 Q44,40 46,46" fill="none" stroke="#c47a6a" stroke-width="1.3" stroke-linecap="round" opacity="0.4"/>
      <path d="M44,30 Q42,36 42,44" fill="none" stroke="#c47a6a" stroke-width="1.1" stroke-linecap="round" opacity="0.3"/>
      <circle cx="34" cy="14" r="2" fill="#c47a6a" opacity="0.25"/>
      <circle cx="38" cy="10" r="1.5" fill="#c47a6a" opacity="0.2"/>
      <circle cx="30" cy="22" r="1.2" fill="#c47a6a" opacity="0.2"/>
      <circle cx="42" cy="18" r="1.2" fill="#c47a6a" opacity="0.18"/>
      <circle cx="36" cy="12" r="1" fill="#c47a6a" opacity="0.15"/>
      <circle cx="28" cy="26" r="0.8" fill="#c47a6a" opacity="0.12"/>
      <circle cx="44" cy="24" r="0.8" fill="#c47a6a" opacity="0.12"/>
      <circle cx="50" cy="30" r="0.8" fill="#c47a6a" opacity="0.1"/>
    </template>

    <!-- 土 · 信 — 层峦叠嶂 -->
    <template v-if="element === 'earth'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fdfaf6"/><stop offset="100%" stop-color="#f0e6d2"/></linearGradient></defs>
      <rect x="2" y="2" width="68" height="68" rx="17" :fill="`url(#${gid})`" stroke="#c4a980" stroke-width="1.2"/>
      <circle cx="36" cy="14" r="5" fill="none" stroke="#b8956a" stroke-width="1" opacity="0.3"/>
      <circle cx="36" cy="14" r="3" fill="#b8956a" opacity="0.2"/>
      <polygon points="2,44 16,32 30,44" fill="none" stroke="#b8956a" stroke-width="1" stroke-linejoin="round" opacity="0.25"/>
      <polygon points="26,44 40,28 54,44" fill="none" stroke="#b8956a" stroke-width="1" stroke-linejoin="round" opacity="0.25"/>
      <polygon points="48,44 60,34 70,44" fill="none" stroke="#b8956a" stroke-width="1" stroke-linejoin="round" opacity="0.25"/>
      <polygon points="2,52 20,38 36,52" fill="none" stroke="#b8956a" stroke-width="1.4" stroke-linejoin="round" opacity="0.45"/>
      <polygon points="32,52 48,34 66,52" fill="none" stroke="#b8956a" stroke-width="1.4" stroke-linejoin="round" opacity="0.45"/>
      <polygon points="10,56 36,18 62,56" fill="none" stroke="#b8956a" stroke-width="2.2" stroke-linejoin="round"/>
      <path d="M32,24 L36,18 L40,24" fill="none" stroke="#b8956a" stroke-width="1.2" stroke-linejoin="round" opacity="0.45"/>
      <path d="M22,40 Q26,36 24,32" fill="none" stroke="#b8956a" stroke-width="0.9" stroke-linecap="round" opacity="0.35"/>
      <path d="M48,40 Q44,36 46,32" fill="none" stroke="#b8956a" stroke-width="0.9" stroke-linecap="round" opacity="0.35"/>
      <path d="M34,30 Q32,26 30,24" fill="none" stroke="#b8956a" stroke-width="0.7" stroke-linecap="round" opacity="0.25"/>
      <path d="M18,42 Q16,36 18,32" fill="none" stroke="#b8956a" stroke-width="0.8" stroke-linecap="round" opacity="0.3"/>
      <path d="M54,42 Q56,36 54,32" fill="none" stroke="#b8956a" stroke-width="0.8" stroke-linecap="round" opacity="0.3"/>
      <path d="M10,16 Q14,12 18,16" fill="none" stroke="#b8956a" stroke-width="0.6" opacity="0.15"/>
      <path d="M54,18 Q58,14 62,18" fill="none" stroke="#b8956a" stroke-width="0.6" opacity="0.15"/>
    </template>

    <!-- 金 · 义 — 剑锋晶簇 -->
    <template v-if="element === 'metal'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#fefdfa"/><stop offset="100%" stop-color="#f4eed8"/></linearGradient></defs>
      <rect x="2" y="2" width="68" height="68" rx="17" :fill="`url(#${gid})`" stroke="#c4a96a" stroke-width="1.2"/>
      <ellipse cx="36" cy="50" rx="22" ry="6" fill="none" stroke="#b8956a" stroke-width="1.1" opacity="0.35"/>
      <polygon points="36,8 42,34 36,52 30,34" fill="none" stroke="#b8956a" stroke-width="2.2" stroke-linejoin="round"/>
      <line x1="36" y1="12" x2="36" y2="48" stroke="#b8956a" stroke-width="0.9" stroke-linecap="round" opacity="0.35"/>
      <line x1="33" y1="24" x2="39" y2="24" stroke="#b8956a" stroke-width="0.8" stroke-linecap="round" opacity="0.3"/>
      <line x1="32" y1="34" x2="40" y2="34" stroke="#b8956a" stroke-width="0.8" stroke-linecap="round" opacity="0.3"/>
      <polygon points="18,32 24,46 16,54 12,44" fill="none" stroke="#b8956a" stroke-width="1.5" stroke-linejoin="round" opacity="0.65"/>
      <polygon points="52,28 58,42 50,54 46,40" fill="none" stroke="#b8956a" stroke-width="1.5" stroke-linejoin="round" opacity="0.65"/>
      <polygon points="22,46 26,52 20,54" fill="none" stroke="#b8956a" stroke-width="0.9" stroke-linejoin="round" opacity="0.3"/>
      <polygon points="46,44 50,50 44,52" fill="none" stroke="#b8956a" stroke-width="0.9" stroke-linejoin="round" opacity="0.3"/>
      <polygon points="30,50 34,56 28,56" fill="none" stroke="#b8956a" stroke-width="0.7" stroke-linejoin="round" opacity="0.2"/>
      <circle cx="32" cy="16" r="1.5" fill="#b8956a" opacity="0.25"/>
      <circle cx="40" cy="20" r="1" fill="#b8956a" opacity="0.2"/>
      <circle cx="20" cy="36" r="1" fill="#b8956a" opacity="0.18"/>
      <circle cx="54" cy="32" r="0.8" fill="#b8956a" opacity="0.15"/>
      <circle cx="28" cy="44" r="0.8" fill="#b8956a" opacity="0.12"/>
    </template>

    <!-- 水 · 智 — 瀑布深潭 -->
    <template v-if="element === 'water'">
      <defs><linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#f8fafc"/><stop offset="100%" stop-color="#dee8f2"/></linearGradient></defs>
      <rect x="2" y="2" width="68" height="68" rx="17" :fill="`url(#${gid})`" stroke="#7a9ab8" stroke-width="1.2"/>
      <path d="M16,10 L16,40" stroke="#7a9ab8" stroke-width="1.2" stroke-linecap="round" opacity="0.3"/>
      <path d="M20,10 Q24,24 20,40" fill="none" stroke="#6b8aa8" stroke-width="1.8" stroke-linecap="round" opacity="0.5"/>
      <path d="M24,10 Q28,24 24,40" fill="none" stroke="#6b8aa8" stroke-width="1.5" stroke-linecap="round" opacity="0.4"/>
      <path d="M28,10 Q32,24 28,40" fill="none" stroke="#6b8aa8" stroke-width="1.2" stroke-linecap="round" opacity="0.3"/>
      <line x1="8" y1="42" x2="64" y2="42" stroke="#7a9ab8" stroke-width="1.3" stroke-linecap="round" opacity="0.45"/>
      <ellipse cx="36" cy="42" rx="8" ry="2" fill="none" stroke="#6b8aa8" stroke-width="0.9" opacity="0.35"/>
      <ellipse cx="36" cy="42" rx="16" ry="4" fill="none" stroke="#6b8aa8" stroke-width="0.7" opacity="0.2"/>
      <ellipse cx="36" cy="42" rx="24" ry="6" fill="none" stroke="#6b8aa8" stroke-width="0.5" opacity="0.12"/>
      <path d="M44,18 Q60,36 44,50 Q28,36 44,18Z" fill="none" stroke="#6b8aa8" stroke-width="2.2" stroke-linejoin="round"/>
      <path d="M40,28 Q42,34 40,40" fill="none" stroke="#6b8aa8" stroke-width="0.9" stroke-linecap="round" opacity="0.25"/>
      <path d="M54,24 Q62,32 54,40 Q46,32 54,24Z" fill="none" stroke="#6b8aa8" stroke-width="1.2" stroke-linejoin="round" opacity="0.35"/>
      <path d="M12,26 Q18,34 12,40 Q6,34 12,26Z" fill="none" stroke="#6b8aa8" stroke-width="1" stroke-linejoin="round" opacity="0.3"/>
      <path d="M14,44 Q18,42 22,44" fill="none" stroke="#6b8aa8" stroke-width="0.7" stroke-linecap="round" opacity="0.2"/>
      <path d="M50,44 Q54,42 58,44" fill="none" stroke="#6b8aa8" stroke-width="0.7" stroke-linecap="round" opacity="0.2"/>
    </template>
  </svg>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  element: 'wood' | 'fire' | 'earth' | 'metal' | 'water'
  size?: number
}>(), { size: 68 })

const gid = `es-${Math.random().toString(36).slice(2, 8)}`
</script>
