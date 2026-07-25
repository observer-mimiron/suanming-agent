<template>
  <n-config-provider>
    <n-message-provider><ChatPanel /></n-message-provider>
  </n-config-provider>
</template>
<script setup lang="ts">
import { NConfigProvider, NMessageProvider } from 'naive-ui'
import ChatPanel from './components/ChatPanel.vue'
</script>
