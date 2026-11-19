<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { session } from '../../api/admin'
import { message, post } from '../../api/client'
import ProBrand from '../../components/ProBrand.vue'
import ProIcon from '../../components/ProIcon.vue'
import ProThemeButton from '../../components/ProThemeButton.vue'
import '../pro-base.css'
import './login.css'

const router = useRouter()
const route = useRoute()
const username = ref('')
const password = ref('')
const userField = ref<HTMLInputElement>()
const passwordField = ref<HTMLInputElement>()
const visible = ref(false)
const busy = ref(false)
const invalid = ref<'username' | 'password'>()
const failure = ref(route.query.unavailable ? '暂时无法连接管理服务，请稍后重试。' : '')

function clearError() {
  failure.value = ''
  invalid.value = undefined
}
async function login() {
  if (busy.value) return
  clearError()
  if (!username.value.trim() || !password.value) {
    invalid.value = !username.value.trim() ? 'username' : 'password'
    failure.value = '请输入用户名和密码。'
    const field = invalid.value === 'username' ? userField.value : passwordField.value
    field?.focus()
    return
  }
  busy.value = true
  try {
    session.value = await post('/api/admin/login', { username: username.value.trim(), password: password.value })
    await router.replace('/admin/dashboard')
  } catch (error) {
    failure.value = message(error)
  } finally {
    password.value = ''
    visible.value = false
    busy.value = false
  }
}
</script>

<template>
  <div class="pro-page login-page">
    <main class="login-shell">
      <div class="login-art">
        <ProBrand />
        <div class="login-visual" aria-hidden="true"><img src="/logo-sky.svg" alt=""></div>
        <div class="login-bottom">平台配置、开放接口与服务状态</div>
      </div>
      <div class="login-content">
        <div class="login-header">
          <RouterLink class="backlink" to="/"><ProIcon name="arrow" :size="16" />返回解析前台</RouterLink>
          <ProThemeButton />
        </div>
        <h1 class="accent-text">管理员登录</h1>
        <p class="muted">使用管理员账户访问 ClipBridge 控制台。</p>
        <form class="form-grid" novalidate @submit.prevent="login" :aria-busy="busy">
          <label class="field">用户名
            <input ref="userField" v-model="username" class="control" name="username" autocomplete="username" required maxlength="64"
              placeholder="请输入用户名" aria-describedby="login-error" :aria-invalid="invalid === 'username'" @input="clearError">
          </label>
          <div class="field"><label for="password">密码</label>
            <span class="password-field">
              <input id="password" ref="passwordField" v-model="password" class="control" name="password"
                :type="visible ? 'text' : 'password'" autocomplete="current-password" required placeholder="请输入密码"
                aria-describedby="login-error" :aria-invalid="invalid === 'password'" @input="clearError">
              <button type="button" class="btn square quiet" :aria-label="visible ? '隐藏密码' : '显示密码'" :aria-pressed="visible" @click="visible = !visible">
                <ProIcon :name="visible ? 'eye-off' : 'eye'" :size="18" />
              </button>
            </span>
          </div>
          <span class="feedback" id="login-error" role="status">{{ failure }}</span>
          <button class="btn primary" type="submit" :disabled="busy">{{ busy ? '正在登录…' : '登录' }} <ProIcon name="arrow" :size="18" /></button>
        </form>
        <div class="login-preview">
          <p class="muted">公开解析工具可直接使用。</p>
          <RouterLink class="btn" :to="{ path: '/', hash: '#guide' }">查看使用指南 <ProIcon name="arrow" :size="17" /></RouterLink>
        </div>
      </div>
    </main>
  </div>
</template>
