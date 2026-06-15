<template>
  <div :class="['bubble', message.role]">
    <template v-for="(seg,i) in message.segments" :key="i">
      <TextSegment v-if="seg.type==='text'" :content="seg.content"/>
      <ThinkingSegment v-else-if="seg.type==='thinking'" :text="seg.text" :agent="seg.agent"/>
      <ToolCallSegment v-else-if="seg.type==='tool_call'" :tool="seg.tool"/>
      <BaziChartCard v-else-if="seg.type==='component'&&seg.componentType==='bazi-chart'" :data="seg.payload"/>
      <KnowledgeSourceCard v-else-if="seg.type==='component'&&seg.componentType==='knowledge-sources'" :data="seg.payload"/>
      <div v-else-if="seg.type==='error'" class="error-msg">{{ seg.message }}</div>
    </template>
  </div>
</template>
<script setup lang="ts">
import type { ChatMessage } from '../types/chat'
import TextSegment from './TextSegment.vue'
import BaziChartCard from './BaziChartCard.vue'
import KnowledgeSourceCard from './KnowledgeSourceCard.vue'
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
  color: var(--text-primary);
  font-size: 14px;
  line-height: 1.55;
}
.error-msg { color: #c47a6a; padding: 8px 0; }
</style>
