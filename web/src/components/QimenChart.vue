<template>
  <div class="qimen-card">
    <div class="qm-info">
      {{ data.ju_text }} · {{ data.duty_text }} · {{ data.question_time || '—' }}
    </div>

    <div class="qm-grid">
      <div
        v-for="cell in gridCells" :key="cell.palace"
        class="qm-cell"
        :class="{
          'qm-duty': cell.palace === dutyPalace && cell.palace !== '中',
          'qm-center': cell.palace === '中',
          'qm-dummy-cell': cell.isCenterDummy
        }"
        :style="{ animationDelay: cellDelay(cell.palace) + 'ms' }"
        @mousemove="handleTilt"
        @mouseleave="resetTilt"
      >
        <!-- 四角几何点 -->
        <span class="corner-dot tl"></span>
        <span class="corner-dot tr"></span>
        <span class="corner-dot bl"></span>
        <span class="corner-dot br"></span>

        <!-- 中宫/太极 dummy 内容 -->
        <template v-if="cell.isCenterDummy">
          <div class="qm-center-dummy">
            <svg class="taiji-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 2a5 5 0 0 0 0 10 5 5 0 0 1 0 10" />
              <circle cx="12" cy="7" r="1.5" fill="currentColor" />
              <circle cx="12" cy="17" r="1.5" fill="none" stroke="currentColor" />
            </svg>
            <span class="qm-dummy-label">中宫定盘</span>
          </div>
        </template>

        <!-- 正常宫位内容 -->
        <template v-else>
          <!-- 卡片虚线页眉：属性小徽章 -->
          <div class="qm-cell-header">
            <div class="qm-palace-badge" :class="'wx-badge-' + palaceElement(cell.palace)">
              <ElementSprite :element="palaceElement(cell.palace)" :size="11" class="qm-badge-icon" />
              <span class="qm-palace-name">{{ cell.palace }}宫</span>
              <span class="qm-palace-wx">{{ palaceWuxingZh[cell.palace] }}</span>
            </div>
          </div>

          <!-- 八神（神） -->
          <div class="qm-god">{{ cell.god || '—' }}</div>

          <!-- 天人星门对偶 -->
          <div class="qm-core-pair">
            <div class="qm-star-wrap">
              <span class="qm-label">星</span>
              <span class="qm-star">{{ cell.star || '—' }}</span>
            </div>
            <div class="qm-door-wrap">
              <span class="qm-label">门</span>
              <span class="qm-door">{{ cell.door || '—' }}</span>
            </div>
          </div>

          <!-- 客主克应天干胶囊 -->
          <div class="qm-gans-matrix">
            <span class="qm-gan-cell guest">{{ cell.guest_gan || '—' }}</span>
            <span class="qm-gan-divider">/</span>
            <span class="qm-gan-cell host">{{ cell.host_gan || '—' }}</span>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ElementSprite from './sprites/ElementSprite.vue'

const props = defineProps<{ data: any }>()

const gridLayoutPalaces = ['巽', '离', '坤', '震', '中', '兑', '艮', '坎', '乾']

const gridCells = computed(() => {
  const raw = props.data?.cells || []
  const map = new Map<string, any>()
  for (const c of raw) {
    map.set(c.palace, c)
  }
  
  return gridLayoutPalaces.map(name => {
    if (map.has(name)) {
      return {
        ...map.get(name),
        isCenterDummy: false
      }
    }
    return {
      palace: name,
      god: '',
      star: '',
      door: '',
      guest_gan: '',
      host_gan: '',
      isCenterDummy: true
    }
  })
})

const dutyPalace = computed(() => props.data?.duty_palace || '')

// Palace name → five element Chinese mapping
const palaceWuxingZh: Record<string, string> = {
  '坎': '水', '坤': '土', '震': '木', '巽': '木',
  '中': '土', '乾': '金', '兑': '金', '艮': '土', '离': '火',
}

// Palace name → five element key mapping for sprite
const palaceWuxing: Record<string, 'wood'|'fire'|'earth'|'metal'|'water'> = {
  '坎': 'water', '坤': 'earth', '震': 'wood', '巽': 'wood',
  '中': 'earth', '乾': 'metal', '兑': 'metal', '艮': 'earth', '离': 'fire',
}
function palaceElement(palace: string) {
  return palaceWuxing[palace] || 'earth'
}

// Center first, then cardinal directions, then corners — for stagger animation
const displayOrder: Record<string, number> = {
  '中': 0, '坎': 1, '离': 1, '震': 1, '兑': 1,
  '坤': 2, '乾': 2, '艮': 2, '巽': 2,
}
function cellDelay(palace: string) {
  return (displayOrder[palace] ?? 2) * 60
}

function handleTilt(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const xc = rect.width / 2
  const yc = rect.height / 2
  const rotateY = ((x - xc) / xc) * 8
  const rotateX = -((y - yc) / yc) * 8
  el.style.transform = `perspective(600px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) translateY(-4px)`
  el.style.boxShadow = `0 10px 20px rgba(0, 0, 0, 0.12), 0 0 12px var(--accent-bg)`
}

function resetTilt(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  el.style.transform = `perspective(600px) rotateX(0deg) rotateY(0deg) translateY(0)`
  el.style.boxShadow = ``
}
</script>

<style scoped>
.qimen-card {
  text-align: left;
  max-width: 100%;
}

.qm-info {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 16px;
  line-height: 1.6;
  padding: 0 2px;
}

.qm-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-hover);
}

.qm-cell {
  position: relative;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 14px 10px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  min-height: 140px;
  opacity: 0;
  overflow: hidden;
  transform-style: preserve-3d;
  animation: cell-in 0.3s cubic-bezier(0.22,0.61,0.36,1) forwards;
  transition: transform 0.25s cubic-bezier(0.25, 0.8, 0.25, 1), box-shadow 0.25s ease;
  /* 现代神秘学同心圆几何星轨水印 */
  background-image: radial-gradient(circle at center, transparent 40%, rgba(184, 149, 106, 0.02) 41%, rgba(184, 149, 106, 0.02) 43%, transparent 44%),
                    radial-gradient(circle at center, transparent 65%, rgba(184, 149, 106, 0.015) 66%, rgba(184, 149, 106, 0.015) 67%, transparent 68%);
}
.qm-cell::after {
  content: "";
  position: absolute;
  top: -50%; left: -50%;
  width: 200%; height: 200%;
  background: linear-gradient(
    115deg,
    transparent 40%,
    rgba(212, 175, 55, 0.08) 48%,
    rgba(255, 255, 255, 0.15) 50%,
    rgba(212, 175, 55, 0.08) 52%,
    transparent 60%
  );
  transform: translate(-30%, -30%);
  pointer-events: none;
  opacity: 0;
}
.qm-cell:hover::after {
  transform: translate(15%, 15%);
  transition: transform 0.8s cubic-bezier(0.19, 1, 0.22, 1);
  opacity: 1;
}
.qm-cell:hover {
  z-index: 2;
  /* hover shadow will be set dynamically via JS client, providing fallback here */
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.12), 0 0 12px var(--accent-bg);
}

@keyframes cell-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

/* 值符宫高亮与呼吸灯 */
.qm-cell.qm-duty {
  border-color: var(--accent) !important;
  background: var(--bg-secondary) !important;
  animation: duty-glow-cells 3s infinite ease-in-out;
}
@keyframes duty-glow-cells {
  0%, 100% {
    box-shadow: 0 0 12px var(--accent-bg), 0 0 0 1px var(--accent);
  }
  50% {
    box-shadow: 0 0 24px rgba(184, 149, 106, 0.25), 0 0 0 1.5px var(--accent);
  }
}

.qm-cell.qm-center {
  background: var(--bg-hover);
  border-color: var(--border);
}

/* 几何直角 L 包角 (现代塔罗风格) */
.corner-dot {
  position: absolute;
  width: 6px;
  height: 6px;
  pointer-events: none;
  transition: opacity 0.25s ease;
  opacity: 0.3;
}
.corner-dot.tl {
  top: 6px; left: 6px;
  border-top: 1px solid var(--accent);
  border-left: 1px solid var(--accent);
}
.corner-dot.tr {
  top: 6px; right: 6px;
  border-top: 1px solid var(--accent);
  border-right: 1px solid var(--accent);
}
.corner-dot.bl {
  bottom: 6px; left: 6px;
  border-bottom: 1px solid var(--accent);
  border-left: 1px solid var(--accent);
}
.corner-dot.br {
  bottom: 6px; right: 6px;
  border-bottom: 1px solid var(--accent);
  border-right: 1px solid var(--accent);
}

.qm-cell.qm-duty .corner-dot {
  opacity: 0.8;
}
.qm-cell:hover .corner-dot {
  opacity: 0.8;
}

/* 卡片虚线页眉：塔罗风属性徽章 */
.qm-cell-header {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  width: 100%;
  border-bottom: 1px dashed var(--border-light);
  padding-bottom: 8px;
  margin-bottom: 12px;
}

.qm-palace-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 7px 2px 6px;
  border-radius: 20px;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.5px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.01);
  transition: transform 0.2s;
}
.qm-cell:hover .qm-palace-badge {
  transform: scale(1.02);
}

.qm-badge-icon {
  opacity: 0.95;
  display: block;
}
/* 覆盖徽章内 SVG 外框 */
.qm-badge-icon :deep(rect) {
  fill: none !important;
  stroke: none !important;
}

.qm-palace-name {
  font-family: var(--sans);
}
.qm-palace-wx {
  opacity: 0.75;
  font-weight: 500;
}

/* 五行徽章主题配色 */
.wx-badge-water {
  background: rgba(107, 138, 168, 0.08);
  color: #6b8aa8;
  border: 1px solid rgba(107, 138, 168, 0.16);
}
.wx-badge-wood {
  background: rgba(122, 158, 126, 0.08);
  color: #7a9e7e;
  border: 1px solid rgba(122, 158, 126, 0.16);
}
.wx-badge-fire {
  background: rgba(196, 122, 106, 0.08);
  color: #c47a6a;
  border: 1px solid rgba(196, 122, 106, 0.16);
}
.wx-badge-earth {
  background: rgba(184, 149, 106, 0.08);
  color: #b8956a;
  border: 1px solid rgba(184, 149, 106, 0.16);
}
.wx-badge-metal {
  background: rgba(196, 169, 106, 0.08);
  color: #c4a96a;
  border: 1px solid rgba(196, 169, 106, 0.16);
}

/* 中宫太极定盘 Dummy 卡牌 */
.qm-center-dummy {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin-top: auto;
  margin-bottom: auto;
  color: var(--text-muted);
  opacity: 0.45;
  transition: opacity 0.25s, transform 0.25s;
}
.qm-cell:hover .qm-center-dummy {
  opacity: 0.85;
  transform: scale(1.05);
}
.taiji-icon {
  width: 38px;
  height: 38px;
  color: var(--accent);
  margin-bottom: 10px;
  animation: rotate-taiji 20s linear infinite;
}
.qm-dummy-label {
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 1px;
}
.qm-dummy-cell {
  background: var(--bg-hover) !important;
  border: 1px dashed var(--border) !important;
}

@keyframes rotate-taiji {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.qm-god {
  position: relative;
  z-index: 2;
  transform: translateZ(10px);
  font-size: 11px;
  font-weight: 600;
  color: var(--accent);
  letter-spacing: 2px;
  margin-bottom: 8px;
  margin-top: 2px;
  transition: transform 0.25s;
}
.qm-cell:hover .qm-god {
  transform: translateZ(18px);
}

.qm-core-pair {
  position: relative;
  z-index: 2;
  transform: translateZ(10px);
  display: flex;
  width: 100%;
  justify-content: space-around;
  margin-bottom: 8px;
  gap: 12px;
  transition: transform 0.25s;
}
.qm-cell:hover .qm-core-pair {
  transform: translateZ(18px);
}
.qm-star-wrap, .qm-door-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.qm-label {
  font-size: 8px;
  color: var(--text-muted);
  text-transform: uppercase;
  margin-bottom: 2px;
}
.qm-star {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}
.qm-door {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.qm-gans-matrix {
  position: relative;
  z-index: 2;
  transform: translateZ(10px);
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-family: var(--mono);
  color: var(--text-muted);
  background: var(--bg);
  padding: 2px 8px;
  border-radius: 6px;
  margin-top: auto; /* 天干始终推至底部对齐 */
  border: 1px solid var(--border-light);
  transition: transform 0.25s;
}
.qm-cell:hover .qm-gans-matrix {
  transform: translateZ(16px);
}
.qm-gan-cell {
  font-weight: 600;
  color: var(--text-secondary);
}
.qm-gan-divider {
  opacity: 0.4;
  font-size: 9px;
}
</style>
