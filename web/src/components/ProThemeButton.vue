<script setup lang="ts">
import { computed } from 'vue'
import { dark, toggleTheme } from '../theme'
import ProIcon from './ProIcon.vue'

const label = computed(() => dark.value ? '切换到浅色模式' : '切换到深色模式')
</script>
<template>
  <button type="button" class="btn square" data-theme-toggle :aria-label="label" :title="label" @click="toggleTheme">
    <ProIcon :name="dark ? 'sun' : 'moon'" :size="19" />
  </button>
</template>
