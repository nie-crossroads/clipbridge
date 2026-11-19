<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  ElAlert, ElButton, ElCheckbox, ElCheckboxGroup, ElDatePicker, ElDescriptions, ElDescriptionsItem,
  ElDialog, ElDrawer, ElDropdown, ElDropdownItem, ElDropdownMenu, ElEmpty, ElForm, ElFormItem, ElInput,
  ElInputNumber, ElMessage, ElMessageBox, ElRadio, ElRadioGroup, ElStep, ElSteps,
  ElTable, ElTableColumn, ElTag, vLoading,
} from 'element-plus'
import { adminRequest } from '../../api/admin'
import { displayDate, message } from '../../api/client'
import { channelLabel, platformNames } from './labels'

interface KeySettings {
  name: string
  channel: 'rest' | 'mcp'
  scopes: string[]
  platform_allowlist: string[]
  rate_limit_per_minute: number | undefined
  daily_parse_limit: number | undefined
  max_concurrency: number | undefined
  expires_at: string | null
}
interface KeyRow extends KeySettings {
  id: string
  key_hint: string
  status: string
  last_used_at: string | null
}
interface Usage { date: string; used_units: number; daily_parse_limit: number }
type Action = 'usage' | 'rotate' | 'disable' | 'enable' | 'revoke'

const steps = ['基本信息', '权限', '白名单', '额度限制', '确认']
const scopeOptions = {
  'media:parse': '单条解析', 'media:batch': '批量解析', 'platforms:read': '平台能力', 'media:proxy': '媒体代理',
}
const states = {
  active: { label: '启用', type: 'success' },
  disabled: { label: '停用', type: 'info' },
  revoked: { label: '已撤销', type: 'danger' },
  expired: { label: '已过期', type: 'warning' },
} as const
const initialSettings = (): KeySettings => ({
  name: '', channel: 'rest', scopes: ['media:parse'], platform_allowlist: [],
  rate_limit_per_minute: 60, daily_parse_limit: 1000, max_concurrency: 4, expires_at: null,
})
const keys = ref<KeyRow[]>([])
const busy = ref(false)
const loadError = ref('')
const wizard = ref(false)
const step = ref(0)
const form = reactive(initialSettings())
const anyPlatform = ref(true)
const expiryMode = ref<'never' | 'date'>('never')
const date = ref('')
const formError = ref('')
const submitting = ref(false)
const secret = ref('')
const reveal = ref(false)
const visible = ref(false)
const saved = ref(false)
let hideTimer: ReturnType<typeof setTimeout> | undefined
const revealTitle = ref('')
const usageOpen = ref(false)
const usage = ref<Usage>()
const usageKey = ref<KeyRow>()
const revokeOpen = ref(false)
const target = ref<KeyRow>()
const confirmation = ref('')
const expiry = computed(() => {
  if (expiryMode.value !== 'date') return null
  const end = new Date(date.value + 'T23:59:59.999+08:00')
  return Number.isFinite(end.getTime()) ? end.toISOString() : null
})
const maskedSecret = computed(() => secret.value ? secret.value.slice(0, 7) + '••••••••' + secret.value.slice(-4) : '')

function state(status: string) {
  return states[status as keyof typeof states] ?? { label: status, type: 'info' as const }
}
function platformList(codes: string[] | null) {
  return codes?.length ? codes.map(code => platformNames[code] ?? code).join('、') : '不限'
}
const limitText = (n?: number) => n === 0 ? '不限' : n == null ? '—' : n.toLocaleString('zh-CN')

async function load() {
  busy.value = true
  loadError.value = ''
  try { keys.value = await adminRequest('/keys') }
  catch (error) { loadError.value = message(error) }
  finally { busy.value = false }
}
function openWizard() {
  Object.assign(form, initialSettings())
  anyPlatform.value = true
  expiryMode.value = 'never'
  date.value = ''
  formError.value = ''
  step.value = 0
  wizard.value = true
}
function next() {
  formError.value = ''
  if (step.value === 0 && !form.name.trim()) formError.value = '请填写密钥名称。'
  if (step.value === 1 && !form.scopes.length) formError.value = '请至少选择一项权限。'
  if (step.value === 2 && !anyPlatform.value && !form.platform_allowlist.length) formError.value = '请至少选择一个平台。'
  if (step.value === 3 && (form.rate_limit_per_minute == null || form.daily_parse_limit == null || form.max_concurrency == null)) {
    formError.value = '请填写全部额度和并发限制。'
  }
  if (step.value === 3 && expiryMode.value === 'date' && (!expiry.value || Date.parse(expiry.value) <= Date.now())) {
    formError.value = '请选择尚未到期的日期。'
  }
  if (!formError.value) step.value++
}
function updateScopes() {
  if (form.scopes.includes('media:batch') && !form.scopes.includes('media:parse')) form.scopes.push('media:parse')
}
function openReveal(value: string, title: string) {
  secret.value = value
  visible.value = false
  saved.value = false
  revealTitle.value = title
  reveal.value = true
}
async function copySecret() {
  try {
    await navigator.clipboard.writeText(secret.value)
    clearTimeout(hideTimer)
    hideTimer = setTimeout(() => { visible.value = false }, 1500)
    ElMessage.success('已复制密钥')
  } catch { ElMessage.error('无法访问剪贴板，请显示密钥后手动复制。') }
}
function clearSecret() {
  clearTimeout(hideTimer)
  secret.value = ''
  visible.value = false
}
async function create() {
  if (submitting.value) return
  submitting.value = true
  formError.value = ''
  try {
    const result = await adminRequest<{ key: string }>('/keys', {
      ...form, name: form.name.trim(), platform_allowlist: anyPlatform.value ? [] : form.platform_allowlist, expires_at: expiry.value,
    })
    wizard.value = false
    openReveal(result.key, '密钥已创建')
    void load()
  } catch (error) { formError.value = message(error) }
  finally { submitting.value = false }
}
async function runAction(row: KeyRow, action: Exclude<Action, 'usage'>, confirmName = '') {
  submitting.value = true
  try {
    const result = await adminRequest<{ key?: string }>('/keys/' + row.id + '/action', { action, confirm_name: confirmName })
    if (result.key) openReveal(result.key, '密钥已轮换')
    else ElMessage.success({ enable: '密钥已启用', disable: '密钥已停用', rotate: '密钥已轮换', revoke: '密钥已撤销' }[action])
    revokeOpen.value = false
    void load()
  } catch (error) { ElMessage.error(message(error)) }
  finally { submitting.value = false }
}
async function action(row: KeyRow, command: string | number | object) {
  if (submitting.value) return
  if (command === 'usage') {
    try {
      usage.value = await adminRequest('/keys/' + row.id + '/usage')
      usageKey.value = row
      usageOpen.value = true
    } catch (error) { ElMessage.error(message(error)) }
    return
  }
  if (command === 'revoke') {
    target.value = row
    confirmation.value = ''
    revokeOpen.value = true
    return
  }
  if (command === 'rotate' || command === 'disable') {
    const title = command === 'rotate' ? '轮换密钥' : '停用密钥'
    try {
      await ElMessageBox.confirm(command === 'rotate' ? '轮换后旧密钥将立即失效。' : '停用后可从操作菜单重新启用。', title,
        { type: command === 'rotate' ? 'warning' : 'error', confirmButtonText: command === 'rotate' ? '确认轮换' : '确认停用', cancelButtonText: '取消', customClass: 'admin-confirm' })
    } catch { return }
    await runAction(row, command)
  }
  if (command === 'enable') await runAction(row, command)
}
onMounted(load)
onBeforeUnmount(clearSecret)
</script>

<template>
  <div class="admin-stack">
    <ElAlert type="info" show-icon :closable="false" title="可创建和管理 REST 与 MCP 密钥。对应调用入口待接入。" />
    <ElAlert v-if="loadError" type="error" :title="loadError" show-icon :closable="false" />
    <div class="admin-toolbar">
      <span class="caption">共 {{ keys.length }} 把密钥</span>
      <ElButton v-if="loadError" @click="load">重新加载</ElButton>
      <ElButton type="primary" @click="openWizard">创建密钥</ElButton>
    </div>
    <ElTable v-loading="busy" :data="keys" row-key="id" border class="admin-table">
      <ElTableColumn prop="name" label="名称" :min-width="120" show-overflow-tooltip />
      <ElTableColumn label="密钥片段" :min-width="155" show-overflow-tooltip>
        <template #default="{ row }"><span class="mono">{{ row.key_hint.replaceAll('*', '•') }}</span></template>
      </ElTableColumn>
      <ElTableColumn label="渠道" :min-width="120">
        <template #default="{ row }"><ElTag :type="channelLabel(row.channel).type">{{ channelLabel(row.channel).label }}</ElTag></template>
      </ElTableColumn>
      <ElTableColumn label="权限" :min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ row.scopes?.join('、') || '—' }}</template>
      </ElTableColumn>
      <ElTableColumn label="平台白名单" :min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ platformList(row.platform_allowlist) }}</template>
      </ElTableColumn>
      <ElTableColumn label="每分钟限额" :min-width="120" align="right"><template #default="{ row }">{{ limitText(row.rate_limit_per_minute) }}</template></ElTableColumn>
      <ElTableColumn label="每日额度" :min-width="120" align="right"><template #default="{ row }">{{ limitText(row.daily_parse_limit) }}</template></ElTableColumn>
      <ElTableColumn prop="max_concurrency" label="并发" :min-width="120" align="right" />
      <ElTableColumn label="状态" :min-width="120">
        <template #default="{ row }"><ElTag :type="state(row.status).type">{{ state(row.status).label }}</ElTag></template>
      </ElTableColumn>
      <ElTableColumn label="到期时间" :min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ row.expires_at ? displayDate(row.expires_at) : '永不过期' }}</template>
      </ElTableColumn>
      <ElTableColumn label="最近使用" :min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ displayDate(row.last_used_at) }}</template>
      </ElTableColumn>
      <ElTableColumn label="操作" :min-width="120" fixed="right">
        <template #default="{ row }">
          <ElDropdown trigger="click" @command="action(row as KeyRow, $event)">
            <ElButton type="primary" link :aria-label="row.name + ' 操作'">操作</ElButton>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem command="usage">查看用量</ElDropdownItem>
                <ElDropdownItem command="rotate" :disabled="!['active', 'disabled'].includes(row.status)">轮换</ElDropdownItem>
                <ElDropdownItem v-if="row.status === 'active'" command="disable">停用</ElDropdownItem>
                <ElDropdownItem v-if="row.status === 'disabled'" command="enable">启用</ElDropdownItem>
                <ElDropdownItem command="revoke" divided :disabled="row.status === 'revoked'">撤销</ElDropdownItem>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </template>
      </ElTableColumn>
      <template #empty>
        <ElEmpty v-if="!busy && !loadError" description="尚未创建密钥">
          <ElButton type="primary" @click="openWizard">创建第一把密钥</ElButton>
        </ElEmpty>
      </template>
    </ElTable>

    <ElDialog v-model="wizard" title="创建密钥" class="admin-dialog" width="min(720px, 94vw)" align-center append-to-body destroy-on-close
      :close-on-click-modal="false" :before-close="done => { if (!submitting) done() }">
      <ElSteps :active="step" finish-status="success" align-center class="key-steps">
        <ElStep v-for="label in steps" :key="label" :title="label" />
      </ElSteps>
      <ElForm label-position="top" class="admin-form admin-form-stack key-step-body" @submit.prevent="step < 4 ? next() : create()">
        <template v-if="step === 0">
          <ElFormItem label="名称" required>
            <ElInput v-model="form.name" maxlength="40" show-word-limit placeholder="例如：生产 REST 主密钥" />
          </ElFormItem>
          <ElFormItem label="渠道">
            <ElRadioGroup v-model="form.channel" class="admin-choices"><ElRadio value="rest" border>REST 接口</ElRadio><ElRadio value="mcp" border>MCP 工具</ElRadio></ElRadioGroup>
            <p class="caption admin-field-hint">需要两个入口时，请为每个渠道各创建一把密钥。</p>
          </ElFormItem>
        </template>
        <template v-else-if="step === 1">
          <ElAlert type="info" title="选择批量解析时会自动勾选单条解析权限。" show-icon :closable="false" />
          <ElFormItem label="权限" required>
            <ElCheckboxGroup v-model="form.scopes" class="key-scope-choices" @change="updateScopes">
              <ElCheckbox v-for="(label, scope) in scopeOptions" :key="scope" :value="scope" border
                :disabled="scope === 'media:parse' && form.scopes.includes('media:batch')">{{ label }} <span class="mono key-scope-code">{{ scope }}</span></ElCheckbox>
            </ElCheckboxGroup>
          </ElFormItem>
        </template>
        <ElFormItem v-else-if="step === 2" label="平台白名单">
          <div class="admin-stack full-width">
            <ElCheckbox v-model="anyPlatform" border class="key-unlimited">不限平台</ElCheckbox>
            <ElCheckboxGroup v-model="form.platform_allowlist" :disabled="anyPlatform" aria-label="选择允许的平台" class="platform-choices">
              <ElCheckbox v-for="(name, code) in platformNames" :key="code" :value="code" border>{{ name }}</ElCheckbox>
            </ElCheckboxGroup>
          </div>
        </ElFormItem>
        <div v-else-if="step === 3" class="admin-field-grid">
          <ElFormItem label="每分钟调用数" required>
            <ElInputNumber v-model="form.rate_limit_per_minute" :min="0" :max="100000" :precision="0" />
            <span class="caption admin-field-hint">0 表示不限</span>
          </ElFormItem>
          <ElFormItem label="每日解析作品数" required>
            <ElInputNumber v-model="form.daily_parse_limit" :min="0" :max="1000000000" :precision="0" />
            <span class="caption admin-field-hint">0 表示不限</span>
          </ElFormItem>
          <ElFormItem label="密钥并发数" required><ElInputNumber v-model="form.max_concurrency" :min="1" :max="100" :precision="0" /></ElFormItem>
          <ElFormItem label="有效期">
            <ElRadioGroup v-model="expiryMode" class="admin-choices"><ElRadio value="never" border>永不过期</ElRadio><ElRadio value="date" border>指定日期</ElRadio></ElRadioGroup>
            <div v-if="expiryMode === 'date'" class="full-width">
              <ElDatePicker v-model="date" value-format="YYYY-MM-DD" placeholder="选择到期日期" />
            </div>
          </ElFormItem>
        </div>
        <template v-else>
          <ElDescriptions class="admin-descriptions key-summary" :column="2" :label-width="110" border>
            <ElDescriptionsItem label="名称">{{ form.name }}</ElDescriptionsItem>
            <ElDescriptionsItem label="渠道">{{ channelLabel(form.channel).label }}</ElDescriptionsItem>
            <ElDescriptionsItem label="权限" :span="2">{{ form.scopes.join('、') }}</ElDescriptionsItem>
            <ElDescriptionsItem label="平台白名单" :span="2">{{ anyPlatform ? '不限' : platformList(form.platform_allowlist) }}</ElDescriptionsItem>
            <ElDescriptionsItem label="每分钟调用数">{{ limitText(form.rate_limit_per_minute) }}</ElDescriptionsItem>
            <ElDescriptionsItem label="每日解析作品数">{{ limitText(form.daily_parse_limit) }}</ElDescriptionsItem>
            <ElDescriptionsItem label="密钥并发数">{{ form.max_concurrency }}</ElDescriptionsItem>
            <ElDescriptionsItem label="有效期">{{ expiry ? displayDate(expiry) : '永不过期' }}</ElDescriptionsItem>
          </ElDescriptions>
          <ElAlert type="warning" title="完整密钥只展示一次，请妥善保存。" show-icon :closable="false" />
        </template>
        <ElAlert v-if="formError" type="error" :title="formError" show-icon :closable="false" />
      </ElForm>
      <template #footer>
        <div class="key-wizard-footer">
          <ElButton v-if="step > 0" :disabled="submitting" @click="step--; formError = ''">上一步</ElButton>
          <div class="admin-actions key-wizard-actions">
            <ElButton :disabled="submitting" @click="wizard = false">取消</ElButton>
            <ElButton v-if="step < 4" type="primary" @click="next">下一步</ElButton>
            <ElButton v-else type="primary" :loading="submitting" @click="create">创建并生成密钥</ElButton>
          </div>
        </div>
      </template>
    </ElDialog>

    <ElDialog v-model="reveal" :title="revealTitle" class="admin-dialog" width="min(580px, 94vw)" align-center append-to-body
      destroy-on-close :close-on-click-modal="false" @close="clearSecret">
      <div class="admin-stack">
        <ElAlert type="warning" title="关闭后将无法再次查看完整密钥。" show-icon :closable="false" />
        <ElInput :model-value="visible ? secret : maskedSecret" readonly aria-label="一次性密钥" class="mono" />
        <ElCheckbox v-model="saved" border class="key-unlimited">我已妥善保存</ElCheckbox>
      </div>
      <template #footer>
        <div class="admin-actions admin-actions-end">
          <ElButton @click="visible = !visible">{{ visible ? '隐藏' : '显示' }}</ElButton>
          <ElButton type="primary" @click="copySecret">复制密钥</ElButton>
          <ElButton :disabled="!saved" @click="reveal = false">完成</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog v-model="revokeOpen" title="撤销密钥" class="admin-dialog" width="min(520px, 94vw)" align-center append-to-body
      :close-on-click-modal="false" :before-close="done => { if (!submitting) done() }">
      <div class="admin-form-stack">
        <ElAlert type="error" title="撤销后无法恢复。" show-icon :closable="false" />
        <ElForm label-position="top" class="admin-form" @submit.prevent>
          <ElFormItem :label="'请输入密钥名称以确认：' + (target?.name ?? '')">
            <ElInput v-model="confirmation" autocomplete="off" />
          </ElFormItem>
        </ElForm>
      </div>
      <template #footer>
        <div class="admin-actions admin-actions-end">
          <ElButton :disabled="submitting" @click="revokeOpen = false">取消</ElButton>
          <ElButton type="danger" :disabled="!target || confirmation !== target.name" :loading="submitting"
            @click="target && runAction(target, 'revoke', confirmation)">确认撤销</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDrawer v-model="usageOpen" title="用量与配额" class="admin-drawer" append-to-body size="min(420px, 94vw)">
      <ElDescriptions v-if="usage && usageKey" class="admin-descriptions" :column="1" border>
        <ElDescriptionsItem label="密钥名称">{{ usageKey.name }}</ElDescriptionsItem>
        <ElDescriptionsItem label="统计日期">{{ usage.date }}</ElDescriptionsItem>
        <ElDescriptionsItem label="已用作品数">{{ usage.used_units }}</ElDescriptionsItem>
        <ElDescriptionsItem label="每日额度">{{ limitText(usage.daily_parse_limit) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="剩余额度">{{ usage.daily_parse_limit ? Math.max(0, usage.daily_parse_limit - usage.used_units) : '不限' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="每分钟限额">{{ limitText(usageKey.rate_limit_per_minute) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="并发上限">{{ usageKey.max_concurrency }}</ElDescriptionsItem>
        <ElDescriptionsItem label="最近使用">{{ displayDate(usageKey.last_used_at) }}</ElDescriptionsItem>
      </ElDescriptions>
    </ElDrawer>
  </div>
</template>
