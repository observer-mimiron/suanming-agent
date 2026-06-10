<template><n-tag type="warning" size="small">🔧 {{ tool }}</n-tag></template>
<script setup lang="ts">import {NTag} from 'naive-ui'; defineProps<{tool:string}>()</script>
