<script setup lang="ts">
withDefaults(defineProps<{ name: string; size?: number }>(), { size: 20 })

const paths: Record<string, string> = {
  sun: 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5',
  moon: 'M20.9 13.3A9 9 0 0 1 10.7 3.1 9 9 0 1 0 20.9 13.3Z',
  arrow: 'M4 12h15m-6-6 6 6-6 6',
  external: 'M13 5h6v6m0-6-9 9M19 13v5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5',
  link: 'm10 13 4-4m-6 7-1 1a4 4 0 0 1-6-6l4-4a4 4 0 0 1 6 0m5 1 1-1a4 4 0 0 1 6 6l-4 4a4 4 0 0 1-6 0',
  play: 'M8 5l12 7-12 7z',
  pause: 'M9 5v14m6-14v14',
  download: 'M12 3v12m-5-5 5 5 5-5M4 16v4h16v-4',
  copy: 'M10 8h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-8a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2ZM16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2',
  refresh: 'M20 11a8 8 0 1 0-2 6m2-11v5h-5',
  chevron: 'm9 18 6-6-6-6',
  layers: 'm12 3 9 5-9 5-9-5Zm-9 9 9 5 9-5m-18 4 9 5 9-5',
  plus: 'M12 5v14M5 12h14',
  'eye-off': 'm3 3 18 18M9.8 6.3c.7-.2 1.4-.3 2.2-.3 6 0 10 6 10 6s-1 1.5-2.8 3.1M6.2 7.8C3.5 9.6 2 12 2 12s4 6 10 6c1.4 0 2.7-.3 3.9-.8m-5.6-6.9a2.4 2.4 0 0 0 3.4 3.4',
  close: 'm6 6 12 12M18 6 6 18',
  warning: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM12 8v5m0 3h.01',
  info: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM12 11v5m0-8h.01',
  pencil: 'm13.5 6.5 4 4M4 20l4.5-1L20 7.5a1.4 1.4 0 0 0 0-2L18.5 4a1.4 1.4 0 0 0-2 0L5 15.5 4 20Z',
  box: 'm12 3 8 4.5v9L12 21l-8-4.5v-9L12 3Zm-8 4.5 8 4.5 8-4.5M12 12v9',
  image: 'M5 4h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Zm2 5h.01M3 16l4-4 4 4 4-5 6 6',
  user: 'M16 8a4 4 0 1 1-8 0 4 4 0 0 1 8 0Zm4 12v-1a5 5 0 0 0-5-5H9a5 5 0 0 0-5 5v1',
  calendar: 'M6 5h12a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm2-2v4m8-4v4M4 10h16',
}
</script>
<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <template v-if="name === 'clipboard'">
      <rect x="5" y="5" width="14" height="16" rx="2" />
      <rect x="9" y="3" width="6" height="4" rx="1" fill="var(--icon-fill, #fff)" />
      <path d="M9 12h6m-6 4h4" />
    </template>
    <template v-else-if="name === 'terminal'">
      <rect x="3" y="5" width="18" height="14" rx="3" /><path d="m7 9 3 3-3 3m6 0h4" />
    </template>
    <template v-else-if="name === 'eye'">
      <path d="M2 12s4-6 10-6 10 6 10 6-4 6-10 6-10-6-10-6Z" /><circle cx="12" cy="12" r="2.4" />
    </template>
    <path v-else :d="paths[name]" :transform="name === 'link' ? 'translate(0 1) scale(.95)' : undefined" />
  </svg>
</template>
