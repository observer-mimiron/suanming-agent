<template>
  <div class="ks-card" v-if="groups && groups.length">
    <div class="ks-header">知识来源</div>
    <div v-for="(g, gi) in groups" :key="gi" class="ks-group">
      <div class="ks-source">{{ g.source }}</div>
      <div v-for="(p, pi) in g.passages" :key="pi" class="ks-passage">
        <blockquote>{{ p }}</blockquote>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { EvidenceGroup } from '../utils/assistantTurn'
defineProps<{ groups: EvidenceGroup[] }>()
</script>

<style scoped>
.ks-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  padding: 14px 16px;
  margin: 8px 0;
}
.ks-header {
  font-family: var(--serif);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 12px;
}
.ks-group {
  margin-bottom: 10px;
}
.ks-group:last-child { margin-bottom: 0; }
.ks-source {
  font-size: 12px;
  font-weight: 600;
  color: var(--accent-dim);
  margin-bottom: 4px;
}
.ks-passage blockquote {
  margin: 0 0 4px;
  padding-left: 12px;
  border-left: 2px solid var(--border);
  color: var(--text-secondary);
  font-size: 13px;
  font-style: italic;
  line-height: 1.5;
}
</style>
