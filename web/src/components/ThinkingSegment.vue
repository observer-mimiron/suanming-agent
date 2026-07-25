<template>
  <div class="thinking-segment">
    <n-tag size="small" :bordered="false" type="info">{{ agent }}</n-tag>
    <p class="thinking-text">{{ text }}</p>
  </div>
</template>

<script setup lang="ts">
import { NTag } from 'naive-ui'

defineProps<{
  text: string
  agent: string
}>()
</script>

<style scoped>
.thinking-segment {
  padding: 8px 12px;
}
.thinking-text {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
}
</style>
