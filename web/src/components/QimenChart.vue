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
.qimen-card { text-align: left; }

.qm-info {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 16px;
  line-height: 1.6;
}

.qm-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.qm-cell {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 14px 6px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  min-height: 120px;
  opacity: 0;
  animation: cell-in 0.3s cubic-bezier(0.22,0.61,0.36,1) forwards;
}
@keyframes cell-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}

.qm-cell.qm-duty {
  border-color: var(--accent);
  box-shadow: var(--shadow-glow);
}
.qm-cell.qm-center {
  background: var(--bg-hover);
}

.qm-palace {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  letter-spacing: 2px;
  text-transform: uppercase;
}
.qm-sprite { margin: 2px 0; }
.qm-star { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.qm-door { font-size: 11px; font-weight: 500; color: var(--text-secondary); }
.qm-god  { font-size: 10px; color: var(--text-muted); font-style: italic; }
.qm-gans {
  font-size: 10px;
  font-family: var(--mono);
  color: var(--text-muted);
  letter-spacing: 0.5px;
}
</style>
