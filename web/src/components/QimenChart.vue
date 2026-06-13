<template>
  <div class="qimen-content">
    <div class="qm-info">
      <span>问卦时间：{{ data.question_time || '—' }}</span>
    </div>
    <div class="qm-info">
      <span>{{ data.ju_text }}</span>
      <span class="qm-sep">|</span>
      <span>{{ data.duty_text }}</span>
    </div>

    <div class="qm-grid">
      <div
        v-for="cell in cells"
        :key="cell.palace"
        class="qm-cell"
        :class="{
          'qm-duty': cell.palace === dutyPalace && cell.palace !== '中',
        }"
      >
        <div class="qm-palace">{{ cell.palace }}</div>
        <div class="qm-star">{{ cell.star || '—' }}</div>
        <div class="qm-door">{{ cell.door || '—' }}</div>
        <div class="qm-god">{{ cell.god || '—' }}</div>
        <div class="qm-gans">
          <span class="qm-gan heaven">{{ cell.guest_gan || '—' }}</span>
          <span class="qm-gan earth">{{ cell.host_gan || '—' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ data: any }>()

const cells = computed(() => props.data?.cells || [])
const dutyPalace = computed(() => props.data?.duty_palace || '')
</script>

<style scoped>
.qimen-content {
  font-size: 13px;
}

.qm-info {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 12px;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.qm-sep { opacity: 0.3; }

.qm-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-template-rows: repeat(3, 1fr);
  gap: 4px;
  aspect-ratio: 1;
}

.qm-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 4px 2px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  text-align: center;
  min-height: 0;
  overflow: hidden;
}

.qm-cell.qm-duty {
  border-color: var(--accent);
  background: rgba(90, 158, 143, 0.1);
}

.qm-palace {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 600;
}

.qm-star { font-size: 12px; color: #c8b896; }
.qm-door { font-size: 12px; color: #7acc7e; font-weight: 600; }
.qm-god  { font-size: 11px; color: #88b8e0; }

.qm-gans {
  display: flex;
  gap: 4px;
  font-size: 11px;
}
.qm-gan.heaven { color: var(--text-primary); font-weight: 600; }
.qm-gan.earth  { color: var(--text-muted); }
</style>
