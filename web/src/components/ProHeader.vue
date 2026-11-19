<script setup lang="ts">
import ProBrand from './ProBrand.vue'
import ProIcon from './ProIcon.vue'
import ProThemeButton from './ProThemeButton.vue'

defineProps<{ developer?: boolean }>()
</script>
<template>
  <header class="masthead">
    <div class="wrap head-inner">
      <ProBrand />
      <nav v-if="!developer" aria-label="网站导航">
        <a class="current" href="#parser">链接解析</a>
        <a href="#platforms">支持平台</a>
        <a href="#guide">使用指南</a>
        <a href="#faq">常见问题</a>
      </nav>
      <div class="head-actions">
        <RouterLink v-if="developer" class="btn dev-link" to="/">返回解析工具 <ProIcon name="arrow" :size="16" /></RouterLink>
        <ProThemeButton />
      </div>
    </div>
  </header>
</template>
