import { ref } from 'vue'

const key = 'cb-theme'
const system = window.matchMedia('(prefers-color-scheme: dark)')
export const dark = ref(false)
let preference: 'light' | 'dark' | null = null

function apply() {
  dark.value = preference ? preference === 'dark' : system.matches
  document.documentElement.dataset.theme = dark.value ? 'dark' : 'light'
  document.documentElement.classList.toggle('dark', dark.value)
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', dark.value ? '#151d2b' : '#f5f8fd')
}

function loadPreference() {
  try {
    const value = localStorage.getItem(key)
    preference = value === 'dark' || value === 'light' ? value : null
  } catch { preference = null }
  apply()
}

export function restoreTheme() {
  loadPreference()
  system.addEventListener('change', apply)
  window.addEventListener('storage', event => {
    if (event.key === key || event.key === null) loadPreference()
  })
}

export function toggleTheme() {
  preference = dark.value ? 'light' : 'dark'
  apply()
  try { localStorage.setItem(key, preference) }
  catch { /* 当前页面仍可切换主题 */ }
}
