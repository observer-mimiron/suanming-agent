<template>
  <div :class="['bubble', message.role]">
    <template v-for="(seg,i) in message.segments" :key="i">
      <TextSegment v-if="seg.type==='text'" :content="seg.content"/>
      <div v-else-if="seg.type==='error'" class="error-msg">{{ seg.message }}</div>
    </template>
  </div>
</template>
<script setup lang="ts">
import type { ChatMessage } from '../types/chat'
import TextSegment from './TextSegment.vue'
defineProps<{message:ChatMessage}>()
</script>
<style scoped>
.bubble.assistant { max-width: 85%; text-align: left; }
.bubble.user {
  text-align: left;
  max-width: 70%;
  margin-left: auto;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  padding: 10px 16px;
  border-radius: 14px;
  color: var(--text-primary) !important;
  font-size: 14px;
  line-height: 1.55;
  font-weight: 450;
  box-shadow: 0 2px 8px rgba(0,0,0,0.02);
}
.bubble.user :deep(.text) {
  color: var(--text-primary) !important;
}
.error-msg { color: #c47a6a; padding: 8px 0; }
</style>
