<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ElAlert, ElAvatar, ElBreadcrumb, ElBreadcrumbItem, ElButton, ElCard, ElConfigProvider, ElDropdown, ElDropdownItem,
  ElDropdownMenu, ElEmpty, ElMenu, ElMenuItem, ElMenuItemGroup,
} from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { adminRequest, session } from '../../api/admin'
import { message } from '../../api/client'
import Icon from '../../components/Icon.vue'
import ThemeButton from '../../components/ThemeButton.vue'
import { groups, pages } from './navigation'
import './element'
import './form-spacing.css'
import './admin.css'

const route = useRoute()
const router = useRouter()
const current = computed(() => pages.find(page => page.id === (route.params.page ?? route.name)) ?? pages[0])
const collapsed = ref(false)
const narrow = ref(false)
const desktopQuery = window.matchMedia('(max-width: 1199px)')
const collapsedMenu = computed(() => narrow.value || collapsed.value)
function updateViewport() { narrow.value = desktopQuery.matches }
onMounted(() => { updateViewport(); desktopQuery.addEventListener('change', updateViewport) })
onBeforeUnmount(() => desktopQuery.removeEventListener('change', updateViewport))
const logoutError = ref('')
const busy = ref(false)
async function logout() {
  if (busy.value) return
  busy.value = true
  logoutError.value = ''
  try {
    await adminRequest('/logout', {})
    session.value = null
    await router.replace('/admin/login')
  } catch (error) { logoutError.value = message(error) }
  finally { busy.value = false }
}
</script>

<template>
  <ElConfigProvider :locale="zhCn">
    <ElCard class="admin-narrow" shadow="never">
      <ElEmpty description="管理后台建议在桌面端使用"><RouterLink to="/">返回拾影首页</RouterLink></ElEmpty>
    </ElCard>
    <div class="app-shell admin-shell" :class="{ 'is-collapsed': collapsedMenu }">
      <a class="skip-link" href="#admin-main">跳到主要内容</a>
      <aside class="admin-sidebar">
        <RouterLink class="sidebar__brand" to="/admin/dashboard" aria-label="ClipBridge 管理后台"><img class="sidebar-logo" src="/logo-sky.svg" alt="" width="28" height="28"><span>拾影管理后台</span></RouterLink>
        <nav aria-label="管理后台导航">
          <ElMenu :default-active="route.path" :collapse="collapsedMenu" :collapse-transition="false" router>
            <ElMenuItemGroup v-for="group in groups" :key="group.label" :title="collapsedMenu ? '' : group.label">
              <ElMenuItem v-for="item in group.pages" :key="item.id" :index="'/admin/' + item.id" :aria-label="item.title">
                <Icon :name="item.icon" /><template #title>{{ item.title }}</template>
              </ElMenuItem>
            </ElMenuItemGroup>
          </ElMenu>
        </nav>
      </aside>
      <div class="main">
        <header class="topbar">
          <div class="topbar__crumb">
            <ElButton class="sidebar-toggle" text :aria-label="collapsed ? '展开侧栏' : '折叠侧栏'" @click="collapsed = !collapsed"><Icon name="menu" /></ElButton>
            <ElBreadcrumb separator="/"><ElBreadcrumbItem>管理后台</ElBreadcrumbItem><ElBreadcrumbItem>{{ current?.title }}</ElBreadcrumbItem></ElBreadcrumb>
          </div>
          <div class="topbar__actions">
            <ThemeButton />
            <ElDropdown trigger="click" @command="logout">
              <ElButton text><span class="admin-control-row"><ElAvatar :size="26">{{ session?.name.slice(0, 1) || '管' }}</ElAvatar>{{ session?.name || '管理员' }}</span></ElButton>
              <template #dropdown><ElDropdownMenu><ElDropdownItem command="logout" :disabled="busy">退出登录</ElDropdownItem></ElDropdownMenu></template>
            </ElDropdown>
          </div>
        </header>
        <main class="content" id="admin-main">
          <header class="page-head">
            <h1 class="page-head__title">{{ current?.title }}</h1>
            <p class="page-head__desc">{{ current?.description }}</p>
          </header>
          <ElAlert v-if="logoutError" type="error" :title="logoutError" show-icon :closable="false" />
          <RouterView :key="route.path" />
        </main>
      </div>
    </div>
  </ElConfigProvider>
</template>
