<template>
  <div class="qimen-card">
    <div class="qm-info">
      {{ data.ju_text }} · {{ data.duty_text }} · {{ data.question_time || '—' }}
    </div>

    <div class="qm-grid">
      <div
        v-for="cell in cells" :key="cell.palace"
        class="qm-cell"
        :class="{
          'qm-duty': cell.palace === dutyPalace && cell.palace !== '中',
          'qm-center': cell.palace === '中',
        }"
        :style="{ animationDelay: cellDelay(cell.palace) + 'ms' }"
      >
        <!-- 四角几何点 -->
        <span class="corner-dot tl"></span>
        <span class="corner-dot tr"></span>
        <span class="corner-dot bl"></span>
        <span class="corner-dot br"></span>

        <div class="qm-palace">{{ cell.palace }}</div>
        <ElementSprite :element="palaceElement(cell.palace)" :size="22" class="qm-sprite" />
        <div class="qm-god">{{ cell.god || '—' }}</div>
        <div class="qm-star-row">
          <SpriteStar v-if="cell.star" :starType="starType(cell.star)" :size="16" />
          <span class="qm-star-text">{{ cell.star || '—' }}</span>
        </div>
        <div class="qm-door-row">
          <SpriteDoor v-if="cell.door" :doorType="doorType(cell.door)" :size="16" />
          <span class="qm-door-text">{{ cell.door || '—' }}</span>
        </div>
        <div class="qm-gans">{{ cell.guest_gan || '—' }} · {{ cell.host_gan || '—' }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ElementSprite from './sprites/ElementSprite.vue'
import SpriteStar from './sprites/SpriteStar.vue'
import SpriteDoor from './sprites/SpriteDoor.vue'

const props = defineProps<{ data: any }>()

const cells = computed(() => props.data?.cells || [])
const dutyPalace = computed(() => props.data?.duty_palace || '')

const palaceWuxing: Record<string, 'wood'|'fire'|'earth'|'metal'|'water'> = {
  '坎': 'water', '坤': 'earth', '震': 'wood', '巽': 'wood',
  '中': 'earth', '乾': 'metal', '兑': 'metal', '艮': 'earth', '离': 'fire',
}
function palaceElement(palace: string) {
  return palaceWuxing[palace] || 'earth'
}

// Backend star names → sprite prop mapping
const starNameMap: Record<string, string> = {
  '天蓬星':'tianpeng','天英星':'tianying','天冲星':'tianchong',
  '天辅星':'tianfu','天禽星':'tianqin','天心星':'tianxin',
  '天柱星':'tianzhu','天任星':'tianren','天芮星':'tianrui',
}
function starType(name: string): any {
  return starNameMap[name] || 'tianqin'
}

// Backend door names → sprite prop mapping
const doorNameMap: Record<string, string> = {
  '休门':'xiu','生门':'sheng','伤门':'shang','杜门':'du',
  '景门':'jing','死门':'si','惊门':'jingmen','开门':'kai',
}
function doorType(name: string): any {
  return doorNameMap[name] || 'du'
}

const displayOrder: Record<string, number> = {
  '中': 0, '坎': 1, '离': 1, '震': 1, '兑': 1,
  '坤': 2, '乾': 2, '艮': 2, '巽': 2,
}
function cellDelay(palace: string) {
  return (displayOrder[palace] ?? 2) * 60
}
</script>

<style scoped>
.qimen-card {
  text-align: left;
  max-width: 560px;
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
  padding: 16px 8px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 128px;
  opacity: 0;
  animation: cell-in 0.3s cubic-bezier(0.22,0.61,0.36,1) forwards;
}
@keyframes cell-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

.qm-cell.qm-duty {
  border-color: var(--accent) !important;
  background: var(--bg-secondary) !important;
  box-shadow: 0 0 16px var(--accent-bg), var(--shadow-glow) !important;
}
.qm-cell.qm-center {
  background: var(--bg-hover);
  border-color: var(--border);
}

.corner-dot {
  position: absolute;
  width: 3px;
  height: 3px;
  background: var(--accent);
  border-radius: 50%;
  opacity: 0.3;
  pointer-events: none;
}
.corner-dot.tl { top: 6px; left: 6px; }
.corner-dot.tr { top: 6px; right: 6px; }
.corner-dot.bl { bottom: 6px; left: 6px; }
.corner-dot.br { bottom: 6px; right: 6px; }

.qm-cell.qm-duty .corner-dot {
  opacity: 0.8;
}

.qm-palace {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  letter-spacing: 1px;
}
.qm-sprite { margin: 2px 0; }
.qm-god {
  font-size: 10px;
  font-weight: 600;
  color: var(--accent-dim);
  letter-spacing: 0.5px;
}
.qm-star-row {
  display: flex;
  align-items: center;
  gap: 3px;
  margin-top: 1px;
}
.qm-star-text {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
}
.qm-door-row {
  display: flex;
  align-items: center;
  gap: 3px;
}
.qm-door-text {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary);
}
.qm-gans {
  font-size: 10px;
  font-family: var(--mono);
  color: var(--text-muted);
  letter-spacing: 0.5px;
  border-top: 1px dashed var(--border-light);
  padding-top: 4px;
  width: 80%;
  margin-top: 4px;
}
</style>
