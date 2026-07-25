<template>
  <div class="qimen-card">
    <div class="qm-header">
      <div class="qm-kicker">奇门遁甲九宫盘</div>
      <div class="qm-info">
        <span class="qm-info-chip">{{ data.ju_text || '局数未定' }}</span>
        <span class="qm-info-chip qm-info-chip-accent">{{ data.duty_text || '值符值使未定' }}</span>
        <span class="qm-info-chip">{{ data.question_time || '未提供起局时间' }}</span>
      </div>
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
        <span class="corner-dot tl"></span>
        <span class="corner-dot tr"></span>
        <span class="corner-dot bl"></span>
        <span class="corner-dot br"></span>

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

        <template v-else>
          <div class="qm-cell-header">
            <div class="qm-palace-badge" :class="'wx-badge-' + palaceElement(cell.palace)">
              <ElementSprite :element="palaceElement(cell.palace)" :size="11" class="qm-badge-icon" />
              <span class="qm-palace-name">{{ cell.palace }}宫</span>
              <span class="qm-palace-wx">{{ palaceWuxingZh[cell.palace] }}</span>
            </div>
          </div>

          <div class="qm-god">{{ cell.god || '—' }}</div>

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

const palaceWuxingZh: Record<string, string> = {
  坎: '水', 坤: '土', 震: '木', 巽: '木',
  中: '土', 乾: '金', 兑: '金', 艮: '土', 离: '火',
}

const palaceWuxing: Record<string, 'wood'|'fire'|'earth'|'metal'|'water'> = {
  坎: 'water', 坤: 'earth', 震: 'wood', 巽: 'wood',
  中: 'earth', 乾: 'metal', 兑: 'metal', 艮: 'earth', 离: 'fire',
}

function palaceElement(palace: string) {
  return palaceWuxing[palace] || 'earth'
}

const displayOrder: Record<string, number> = {
  中: 0, 坎: 1, 离: 1, 震: 1, 兑: 1,
  坤: 2, 乾: 2, 艮: 2, 巽: 2,
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
  padding: 18px;
  border-radius: 20px;
  background:
    radial-gradient(circle at top, rgba(26, 38, 53, 0.96), rgba(22, 27, 39, 0.98)),
    linear-gradient(180deg, rgba(184, 149, 106, 0.08), transparent 28%);
  color: #f3ecdf;
  border: 1px solid rgba(184, 149, 106, 0.18);
}

.qm-header {
  margin-bottom: 16px;
}

.qm-kicker {
  font-family: var(--serif);
  font-size: 17px;
  font-weight: 700;
  color: #f0ddba;
  margin-bottom: 10px;
}

.qm-info {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.qm-info-chip {
  display: inline-flex;
  align-items: center;
  padding: 5px 10px;
  border-radius: 999px;
  font-size: 11px;
  color: rgba(243, 236, 223, 0.82);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(240, 221, 186, 0.12);
}

.qm-info-chip-accent {
  color: #f3e0b7;
  background: rgba(184, 149, 106, 0.12);
  border-color: rgba(184, 149, 106, 0.24);
}

.qm-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  padding: 16px;
  border: 1px solid rgba(240, 221, 186, 0.12);
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.02), rgba(255, 255, 255, 0.01)),
    radial-gradient(circle at center, rgba(184, 149, 106, 0.06), transparent 68%);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.02);
}

.qm-cell {
  position: relative;
  background: rgba(248, 244, 235, 0.05);
  border: 1px solid rgba(240, 221, 186, 0.12);
  border-radius: 16px;
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
  background-image: radial-gradient(circle at center, transparent 40%, rgba(184, 149, 106, 0.02) 41%, rgba(184, 149, 106, 0.02) 43%, transparent 44%),
                    radial-gradient(circle at center, transparent 65%, rgba(184, 149, 106, 0.015) 66%, rgba(184, 149, 106, 0.015) 67%, transparent 68%);
}

.qm-cell::after {
  content: "";
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
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
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.12), 0 0 12px var(--accent-bg);
}

@keyframes cell-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

.qm-cell.qm-duty {
  border-color: #d8b97d !important;
  background: rgba(184, 149, 106, 0.1) !important;
  animation: duty-glow-cells 3s infinite ease-in-out;
}

@keyframes duty-glow-cells {
  0%, 100% {
    box-shadow: 0 0 12px rgba(184, 149, 106, 0.18), 0 0 0 1px #d8b97d;
  }
  50% {
    box-shadow: 0 0 24px rgba(184, 149, 106, 0.25), 0 0 0 1.5px #d8b97d;
  }
}

.qm-cell.qm-center {
  background: rgba(248, 244, 235, 0.03);
  border-color: rgba(240, 221, 186, 0.12);
}

.corner-dot {
  position: absolute;
  width: 6px;
  height: 6px;
  pointer-events: none;
  transition: opacity 0.25s ease;
  opacity: 0.3;
}

.corner-dot.tl {
  top: 6px;
  left: 6px;
  border-top: 1px solid #c7a76f;
  border-left: 1px solid #c7a76f;
}

.corner-dot.tr {
  top: 6px;
  right: 6px;
  border-top: 1px solid #c7a76f;
  border-right: 1px solid #c7a76f;
}

.corner-dot.bl {
  bottom: 6px;
  left: 6px;
  border-bottom: 1px solid #c7a76f;
  border-left: 1px solid #c7a76f;
}

.corner-dot.br {
  bottom: 6px;
  right: 6px;
  border-bottom: 1px solid #c7a76f;
  border-right: 1px solid #c7a76f;
}

.qm-cell.qm-duty .corner-dot,
.qm-cell:hover .corner-dot {
  opacity: 0.8;
}

.qm-cell-header {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  width: 100%;
  border-bottom: 1px dashed rgba(240, 221, 186, 0.14);
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

.wx-badge-water {
  background: rgba(107, 138, 168, 0.08);
  color: #8bb2d5;
  border: 1px solid rgba(107, 138, 168, 0.22);
}

.wx-badge-wood {
  background: rgba(122, 158, 126, 0.08);
  color: #9cc69f;
  border: 1px solid rgba(122, 158, 126, 0.22);
}

.wx-badge-fire {
  background: rgba(196, 122, 106, 0.08);
  color: #efb0a2;
  border: 1px solid rgba(196, 122, 106, 0.22);
}

.wx-badge-earth {
  background: rgba(184, 149, 106, 0.08);
  color: #e0c18e;
  border: 1px solid rgba(184, 149, 106, 0.22);
}

.wx-badge-metal {
  background: rgba(196, 169, 106, 0.08);
  color: #efd597;
  border: 1px solid rgba(196, 169, 106, 0.22);
}

.qm-center-dummy {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin-top: auto;
  margin-bottom: auto;
  color: rgba(243, 236, 223, 0.62);
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
  color: #e2c48d;
  margin-bottom: 10px;
  animation: rotate-taiji 20s linear infinite;
}

.qm-dummy-label {
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 1px;
}

.qm-dummy-cell {
  background: rgba(248, 244, 235, 0.03) !important;
  border: 1px dashed rgba(240, 221, 186, 0.14) !important;
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
  color: #f2d8a1;
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

.qm-star-wrap,
.qm-door-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.qm-label {
  font-size: 8px;
  color: rgba(243, 236, 223, 0.54);
  text-transform: uppercase;
  margin-bottom: 2px;
}

.qm-star {
  font-size: 13px;
  font-weight: 600;
  color: #fff8eb;
}

.qm-door {
  font-size: 13px;
  font-weight: 600;
  color: #f0c878;
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
  color: rgba(243, 236, 223, 0.66);
  background: rgba(0, 0, 0, 0.18);
  padding: 2px 8px;
  border-radius: 6px;
  margin-top: auto;
  border: 1px solid rgba(240, 221, 186, 0.12);
  transition: transform 0.25s;
}

.qm-cell:hover .qm-gans-matrix {
  transform: translateZ(16px);
}

.qm-gan-cell {
  font-weight: 600;
  color: #f4ead4;
}

.qm-gan-divider {
  opacity: 0.4;
  font-size: 9px;
}

@media (max-width: 640px) {
  .qimen-card {
    padding: 14px;
    border-radius: 16px;
  }

  .qm-grid {
    gap: 8px;
    padding: 10px;
  }

  .qm-cell {
    min-height: 124px;
    padding: 12px 8px;
  }
}
</style>
