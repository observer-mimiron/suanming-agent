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
        <div class="qm-palace">{{ cell.palace }}</div>
        <ElementSprite :element="palaceElement(cell.palace)" :size="28" class="qm-sprite" />
        <div class="qm-star">{{ cell.star || '—' }}</div>
        <div class="qm-door">{{ cell.door || '—' }}</div>
        <div class="qm-god">{{ cell.god || '—' }}</div>
        <div class="qm-gans">{{ cell.guest_gan || '—' }} · {{ cell.host_gan || '—' }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ElementSprite from './sprites/ElementSprite.vue'

const props = defineProps<{ data: any }>()

const cells = computed(() => props.data?.cells || [])
const dutyPalace = computed(() => props.data?.duty_palace || '')

// Palace name → five element mapping
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
  gap: 8px;
}

.qm-cell {
  background: #fcfbf9;
  border: 1px solid #e0dbd2;
  border-radius: 14px;
  padding: 16px 6px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  min-height: 120px;
  opacity: 0;
  animation: cell-in 0.3s cubic-bezier(0.22,0.61,0.36,1) forwards;
}
@keyframes cell-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

.qm-cell.qm-duty {
  border-color: #b8956a;
  box-shadow: 0 0 0 3px rgba(184,149,106,0.12);
  background: #fdfaf5;
}
.qm-cell.qm-center {
  background: #f5f2ec;
  border-color: #d8d2c8;
}

.qm-palace {
  font-size: 10px;
  font-weight: 600;
  color: #b0a498;
  letter-spacing: 2px;
  text-transform: uppercase;
  margin-bottom: 1px;
}
.qm-sprite { margin: 3px 0; }
.qm-star {
  font-size: 13px;
  font-weight: 600;
  color: #3a3530;
  margin-top: 1px;
}
.qm-door {
  font-size: 11px;
  font-weight: 500;
  color: #6b6358;
}
.qm-god {
  font-size: 10px;
  color: #9b9288;
  font-style: italic;
}
.qm-gans {
  font-size: 10px;
  font-family: var(--mono);
  color: #8a8078;
  letter-spacing: 0.5px;
  margin-top: 1px;
}
</style>
