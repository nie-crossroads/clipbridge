<script setup lang="ts">
defineProps<{ name: string }>()
const paths: Record<string, string> = {
  bridge: 'M2 18h20M3 18c2.2-7.5 5.2-11 9-11s6.8 3.5 9 11M7.4 18c.8-4 2.2-6.5 4.6-8.6M16.6 18c-.8-4-2.2-6.5-4.6-8.6M12 7v11',
  sun: 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19m0-14-1.5 1.5m-11 11L5 19',
  moon: 'M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z',
  search: 'M11 17a6 6 0 1 0 0-12 6 6 0 0 0 0 12Zm4.4-1.6L21 21',
  link: 'M9.5 14.5l5-5M8 11l-2 2a3.5 3.5 0 0 0 5 5l2-2m3-3 2-2a3.5 3.5 0 0 0-5-5l-2 2',
  play: 'M9 5l11 7-11 7Z',
  check: 'm5 12 4 4L19 6',
  copy: 'M8 8h12v12H8ZM4 15V4h11',
  download: 'M12 3v13m-5-5 5 5 5-5M4 17v4h16v-4',
  dashboard: 'M3 3h7v7H3zm11 0h7v7h-7zM3 14h7v7H3zm11 0h7v7h-7z',
  platforms: 'm12 3 9 5-9 5-9-5Zm-9 9 9 5 9-5M3 16l9 5 9-5',
  keys: 'M15 4a5 5 0 1 1-4.7 6.7L3 18v3h3l2-2v-2h2l2-2m4-6h.01',
  requests: 'M5 3h14v18H5Zm3 5h8m-8 4h8m-8 4h5',
  settings: 'M4 6h16M4 12h16M4 18h16M8 4v4m8 2v4m-6 2v4',
  audit: 'm12 3 8 3v5c0 5-4 8-8 10-4-2-8-5-8-10V6Zm-3 9 2 2 4-5',
  menu: 'M4 6h16M4 12h16M4 18h16',
}
</script>
<template>
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="paths[name]" /></svg>
</template>
