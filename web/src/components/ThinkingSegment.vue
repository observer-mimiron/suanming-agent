<template>
  <n-collapse><n-collapse-item title="🧠 Agent 思考中...">
    <n-tag size="small" :bordered="false">{{ agent }}</n-tag>
    <p>{{ text }}</p>
  </n-collapse-item></n-collapse>
</template>
<script setup lang="ts">import {NCollapse,NCollapseItem,NTag} from 'naive-ui'; defineProps<{text:string;agent:string}>()</script>
