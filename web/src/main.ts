import { createApp } from 'vue'
import { createRouter, createWebHistory, RouterView } from 'vue-router'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './base.css'
import { restoreTheme } from './theme'
import { refreshSession, session } from './api/admin'
import { ApiError } from './api/client'

const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined
const refreshHome = location.pathname === '/' && navigation?.type === 'reload'
if (refreshHome && location.hash) {
  history.replaceState(history.state, '', location.pathname + location.search)
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/ParseView.vue') },
    { path: '/developers', component: () => import('./views/DeveloperView.vue') },
    { path: '/admin/login', component: () => import('./views/admin/LoginView.vue') },
    {
      path: '/admin', component: () => import('./views/admin/AdminLayout.vue'), meta: { admin: true },
      children: [
        { path: '', redirect: '/admin/dashboard' },
        { path: 'parse-test', name: 'parse-test', component: () => import('./views/admin/ParseTestView.vue') },
        { path: 'platforms', name: 'platforms', component: () => import('./views/admin/PlatformView.vue') },
        { path: 'keys', name: 'keys', component: () => import('./views/admin/KeysView.vue') },
        { path: 'dashboard', name: 'dashboard', component: () => import('./views/admin/DashboardView.vue') },
        { path: 'settings', name: 'settings', component: () => import('./views/admin/SettingsView.vue') },
        { path: ':page(requests|audit)', component: () => import('./views/admin/LogView.vue') },
      ],
    },
    { path: '/:missing(.*)*', redirect: '/' },
  ],
  scrollBehavior(to, from, saved) {
    if (refreshHome && !from.matched.length) return { top: 0, left: 0, behavior: 'instant' }
    return saved ?? (to.hash === '#guide' ? { el: '#guide' } : { top: 0 })
  },
})
router.afterEach(to => {
  document.title = to.path === '/' ? 'ClipBridge · 多平台视频解析'
    : to.path === '/developers' ? '开发者接口 · ClipBridge'
      : to.path === '/admin/login' ? '管理员登录 · ClipBridge' : '管理后台 · ClipBridge'
})
router.beforeEach(async (to) => {
  if (!to.meta.admin || session.value) return
  try {
    await refreshSession()
  } catch (error) {
    return { path: '/admin/login', query: error instanceof ApiError && error.status === 401 ? {} : { unavailable: '1' } }
  }
})
restoreTheme()
createApp(RouterView).use(router).mount('#app')
