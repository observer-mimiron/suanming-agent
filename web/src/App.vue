<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider><ChatPanel/></n-message-provider>
  </n-config-provider>
</template>
<script setup lang="ts">
import {darkTheme,NConfigProvider,NMessageProvider} from 'naive-ui'
import ChatPanel from './components/ChatPanel.vue'
</script>
