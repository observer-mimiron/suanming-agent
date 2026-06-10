<template>
  <div :class="['bubble', message.role]">
    <template v-for="(seg,i) in message.segments" :key="i">
      <TextSegment v-if="seg.type==='text'" :content="seg.content"/>
      <ThinkingSegment v-else-if="seg.type==='thinking'" :text="seg.text" :agent="seg.agent"/>
      <ToolCallSegment v-else-if="seg.type==='tool_call'" :tool="seg.tool"/>
      <BaziChartCard v-else-if="seg.type==='component'&&seg.componentType==='bazi-chart'" :data="seg.payload"/>
      <KnowledgeSourceCard v-else-if="seg.type==='component'&&seg.componentType==='knowledge-sources'" :data="seg.payload"/>
    </template>
  </div>
</template>
<script setup lang="ts">
import type { ChatMessage } from '../types/chat'
import TextSegment from './TextSegment.vue'
import ThinkingSegment from './ThinkingSegment.vue'
import ToolCallSegment from './ToolCallSegment.vue'
import BaziChartCard from './BaziChartCard.vue'
import KnowledgeSourceCard from './KnowledgeSourceCard.vue'
defineProps<{message:ChatMessage}>()
</script>
<style scoped>.bubble.assistant{max-width:85%}.bubble.user{text-align:right;max-width:70%;margin-left:auto;background:var(--n-color-target);padding:12px 16px;border-radius:12px}</style>
