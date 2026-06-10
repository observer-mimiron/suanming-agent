<template>
  <n-card title="📚 命理典籍引用" size="small" class="card" v-if="passages.length">
    <div v-for="(p,i) in passages" :key="i" class="passage">
      <blockquote>{{ p.content }}</blockquote>
      <cite v-if="p.source">— {{ p.source }}</cite>
    </div>
  </n-card>
</template>
<script setup lang="ts">
import {computed} from 'vue'; import {NCard} from 'naive-ui'
const props=defineProps<{data:any}>()
const passages=computed(()=>props.data?.passages||[])
</script>
<style scoped>
.card{margin:8px 0}
blockquote{margin:0 0 4px;padding-left:12px;border-left:2px solid #D4A853;color:#C8C0B5;font-style:italic}
cite{font-size:12px;color:var(--n-text-color-3)}
</style>
