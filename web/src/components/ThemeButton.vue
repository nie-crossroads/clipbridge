<script setup lang="ts">
import { dark, toggleTheme } from '../theme'
import Icon from './Icon.vue'
</script>
<template>
  <button type="button" class="icon-btn" :aria-label="dark ? '切换到浅色主题' : '切换到深色主题'" @click="toggleTheme">
    <Icon :name="dark ? 'moon' : 'sun'" />
  </button>
</template>
