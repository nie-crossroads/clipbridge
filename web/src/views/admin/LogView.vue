<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  ElAlert, ElButton, ElCard, ElDatePicker, ElDescriptions, ElDescriptionsItem, ElDrawer, ElForm, ElFormItem,
  ElInput, ElOption, ElPagination, ElSelect, ElTable, ElTableColumn, ElTag, vLoading,
} from 'element-plus'
import { adminRequest } from '../../api/admin'
import { displayDate, message } from '../../api/client'
import { capabilityNames, channelLabel, channels, platformNames } from './labels'

interface RequestRow {
  request_id: string; channel: string; platform: string; item_id: string; media_type: string
  http_status: number; error_code: string | null; duration_ms: number; created_at: string
  key_hint: string | null; cache_hit: boolean | null; source_ip: string | null; operation: string
}
interface AuditRow {
  id: string; action: string; actor: string; target_id: string; target_type: string; target_label: string
  details: Record<string, unknown> | null; request_id: string | null; created_at: string
}
interface Page<T> { items: T[]; has_more: boolean }
interface KeyOption { id: string; name: string; key_hint: string }
type Option = { value: string; label: string }
type Filter = 'range' | 'channel' | 'key' | 'platform' | 'status' | 'actor' | 'action' | 'target_type'
const isRequests = useRoute().params.page === 'requests'
const requests = ref<RequestRow[]>([])
const audits = ref<AuditRow[]>([])
const keys = ref<KeyOption[]>([])
const actors = ref<string[]>([])
const busy = ref(false)
const failure = ref('')
const optionsError = ref('')
const page = ref(1)
const more = ref(false)
const detail = ref<RequestRow | AuditRow>()
const drawer = ref(false)
const customDates = ref<[string, string] | null>(null)
const initialFilters = () => ({ range: '', channel: '', key: '', platform: '', error_code: '', status: '', request_id: '', actor: '', action: '', target_type: '', object: '' })
const filters = ref(initialFilters())
let applied = new URLSearchParams()
let active: AbortController | undefined
const rangeOptions = [
  { value: '24h', label: '近 24 小时' }, { value: '7d', label: '近 7 天' }, { value: '30d', label: '近 30 天' },
]
const objectNames: Record<string, string> = { key: '密钥', platform: '平台', settings: '系统设置', admin: '管理员' }
const auditNames: Record<string, string> = {
  bootstrap_admin: '创建管理员', reset_admin_password: '重置管理员密码', update_platform: '更新平台配置',
  create_key: '创建密钥', rotate_key: '轮换密钥', disable_key: '停用密钥', enable_key: '启用密钥', revoke_key: '撤销密钥',
  update_settings: '更新系统设置', clear_requests: '清空调用记录', clear_cache: '清空缓存',
}
const settingsLabels: Record<string, string> = {
  web_rate: '来源限速', web_window_seconds: '限速窗口', parse_concurrency: '解析并发', request_days: '调用保留天数',
  audit_days: '审计保留天数', cache_items: '缓存项数', cache_mib: '缓存容量（MiB）', cache_ttl_seconds: '缓存 TTL（秒）',
}
const named = (values: Record<string, string>) => Object.entries(values).map(([value, label]) => ({ value, label }))
const masked = (value?: string | null) => value?.replaceAll('*', '•') || '—'
const controls = computed<{ key: Filter; label: string; options: Option[] }[]>(() => isRequests ? [
  { key: 'range', label: '时间范围', options: [{ value: '1h', label: '近 1 小时' }, ...rangeOptions.slice(0, 2), { value: 'custom', label: '自定义' }] },
  { key: 'key', label: '密钥', options: [{ value: 'web', label: '网页公开入口（无密钥）' }, ...keys.value.map(key => ({ value: key.id, label: key.name + ' · ' + masked(key.key_hint) }))] },
  { key: 'channel', label: '渠道', options: channels },
  { key: 'platform', label: '平台', options: named(platformNames) },
  { key: 'status', label: '状态', options: [{ value: 'success', label: '成功' }, { value: 'failed', label: '失败' }] },
] : [
  { key: 'range', label: '时间范围', options: rangeOptions },
  { key: 'actor', label: '操作者', options: actors.value.map(actor => ({ value: actor, label: actor })) },
  { key: 'action', label: '动作类型', options: named(auditNames) },
  { key: 'target_type', label: '对象类型', options: named(objectNames) },
])
function requestStatus(error: string | null, status: number) {
  if (error || status >= 400) return { type: 'danger' as const, label: '失败' }
  if (status >= 200 && status < 300) return { type: 'success' as const, label: '成功' }
  return { type: 'info' as const, label: '未知' }
}
function auditStatus(action: string) {
  if (['revoke_key', 'disable_key', 'clear_requests', 'clear_cache'].includes(action)) return 'danger'
  if (action.startsWith('create_') || action === 'bootstrap_admin') return 'success'
  return 'info'
}
function target(row: AuditRow) {
  return row.target_type === 'platform' ? platformNames[row.target_id] ?? row.target_label : masked(row.target_label)
}
function changes(row: AuditRow) {
  const data = row.details
  if (!data) return '—'
  if (row.action === 'update_platform') {
    return [
      typeof data.enabled === 'boolean' ? data.enabled ? '开放解析' : '关闭解析' : '',
      typeof data.timeout_seconds === 'number' ? '超时 ' + data.timeout_seconds + ' 秒' : '',
    ].filter(Boolean).join('、') || '—'
  }
  if (row.action === 'update_settings') {
    return Object.entries(settingsLabels).flatMap(([key, label]) => typeof data[key] === 'number' ? [label + ' ' + data[key]] : []).join('、') || '—'
  }
  return '—'
}
const mediaLabels: Record<string, string> = { ...capabilityNames, live: '实况' }
function mediaType(code: string) { return code ? mediaLabels[code] ?? code : '—' }
function show(row: RequestRow | AuditRow) { detail.value = row; drawer.value = true }
const requestDetail = computed(() => isRequests ? detail.value as RequestRow | undefined : undefined)
const auditDetail = computed(() => isRequests ? undefined : detail.value as AuditRow | undefined)

async function load() {
  active?.abort()
  const controller = new AbortController()
  active = controller
  busy.value = true
  failure.value = ''
  const query = new URLSearchParams(applied)
  query.set('page', String(page.value))
  try {
    if (isRequests) {
      const result = await adminRequest<Page<RequestRow>>('/requests?' + query, undefined, controller.signal)
      requests.value = result.items
      more.value = result.has_more
    } else {
      const result = await adminRequest<Page<AuditRow>>('/audit?' + query, undefined, controller.signal)
      audits.value = result.items
      more.value = result.has_more
    }
  } catch (error) { if (!controller.signal.aborted) failure.value = message(error) }
  finally { if (active === controller) busy.value = false }
}
async function loadOptions() {
  optionsError.value = ''
  try {
    if (isRequests) keys.value = await adminRequest('/keys')
    else actors.value = await adminRequest('/audit/actors')
  } catch (error) { optionsError.value = message(error) }
}
function search() {
  const query = new URLSearchParams()
  for (const [name, value] of Object.entries(filters.value)) if (value?.trim()) query.set(name, value.trim())
  if (filters.value.range === 'custom') {
    if (!customDates.value) { failure.value = '请选择开始和结束日期。'; return }
    const [from, to] = customDates.value
    query.set('from', new Date(from + 'T00:00:00+08:00').toISOString())
    query.set('to', new Date(new Date(to + 'T00:00:00+08:00').getTime() + 86400000).toISOString())
  }
  applied = query
  page.value = 1
  void load()
}
function reset() {
  filters.value = initialFilters()
  customDates.value = null
  search()
}
function paginate(next: number) { page.value = next; void load() }
onMounted(() => { void load(); void loadOptions() })
onBeforeUnmount(() => active?.abort())
</script>

<template>
  <div v-loading="busy" class="admin-stack" :aria-busy="busy">
    <ElAlert v-if="optionsError" type="warning" :title="optionsError" show-icon :closable="false"><ElButton link @click="loadOptions">重新加载选项</ElButton></ElAlert>
    <ElAlert v-if="failure" type="error" :title="failure" show-icon :closable="false" />
    <ElCard shadow="never">
      <ElForm label-position="top" class="admin-form log-filters" @submit.prevent="search">
        <ElFormItem v-for="control in controls" :key="control.key" :label="control.label">
          <ElSelect v-model="filters[control.key]" :placeholder="'全部' + (control.key === 'range' ? '时间' : control.label)" filterable clearable>
            <ElOption v-for="option in control.options" :key="option.value" :value="option.value" :label="option.label" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem v-if="isRequests" label="错误码"><ElInput v-model="filters.error_code" placeholder="例如 UPSTREAM_TIMEOUT" maxlength="64" clearable /></ElFormItem>
        <ElFormItem v-else label="对象标识"><ElInput v-model="filters.object" placeholder="名称或脱敏片段" maxlength="128" clearable /></ElFormItem>
        <ElFormItem label="请求 ID"><ElInput v-model="filters.request_id" placeholder="输入 request_id 片段" maxlength="36" clearable /></ElFormItem>
        <ElFormItem v-if="isRequests && filters.range === 'custom'" label="自定义日期（北京时间）" class="log-dates">
          <ElDatePicker v-model="customDates" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" />
        </ElFormItem>
        <div class="admin-actions log-filter-actions">
          <ElButton type="primary" native-type="submit" :disabled="busy">应用筛选</ElButton>
          <ElButton :disabled="busy" @click="reset">重置</ElButton>
        </div>
      </ElForm>
    </ElCard>

    <ElTable v-if="isRequests" :data="requests" row-key="request_id" class="admin-table" border empty-text="没有符合条件的调用记录">
      <ElTableColumn label="时间" :min-width="125" show-overflow-tooltip><template #default="{ row }">{{ displayDate(row.created_at) }}</template></ElTableColumn>
      <ElTableColumn prop="request_id" label="请求 ID" :min-width="125" show-overflow-tooltip />
      <ElTableColumn label="渠道" :min-width="125"><template #default="{ row }"><ElTag :type="channelLabel(row.channel).type">{{ channelLabel(row.channel).label }}</ElTag></template></ElTableColumn>
      <ElTableColumn label="密钥片段" :min-width="125" show-overflow-tooltip><template #default="{ row }">{{ masked(row.key_hint) }}</template></ElTableColumn>
      <ElTableColumn label="平台" :min-width="125"><template #default="{ row }">
        <ElTag v-if="row.platform" class="platform-tag badge--platform" :data-platform="row.platform">{{ platformNames[row.platform] ?? row.platform }}</ElTag><span v-else>—</span>
      </template></ElTableColumn>
      <ElTableColumn label="类型" :min-width="125"><template #default="{ row }">{{ mediaType(row.media_type) }}</template></ElTableColumn>
      <ElTableColumn label="状态 / 错误码" :min-width="145" show-overflow-tooltip>
        <template #default="{ row }"><ElTag :type="requestStatus(row.error_code, row.http_status).type">{{ requestStatus(row.error_code, row.http_status).label }}</ElTag><div v-if="row.error_code" class="caption mono">{{ row.error_code }}</div></template>
      </ElTableColumn>
      <ElTableColumn label="耗时" :min-width="125" align="right"><template #default="{ row }">{{ row.duration_ms }} ms</template></ElTableColumn>
      <ElTableColumn label="缓存命中" :min-width="125"><template #default="{ row }">{{ row.cache_hit == null ? '—' : row.cache_hit ? '是' : '否' }}</template></ElTableColumn>
      <ElTableColumn label="来源 IP（脱敏）" :min-width="145" show-overflow-tooltip><template #default="{ row }">{{ row.source_ip || '—' }}</template></ElTableColumn>
      <ElTableColumn label="操作" :width="88" fixed="right"><template #default="{ row }"><ElButton type="primary" link @click="show(row as RequestRow)">详情</ElButton></template></ElTableColumn>
    </ElTable>

    <ElTable v-else :data="audits" row-key="id" class="admin-table" border empty-text="没有符合条件的审计记录">
      <ElTableColumn label="时间" :min-width="125" show-overflow-tooltip><template #default="{ row }">{{ displayDate(row.created_at) }}</template></ElTableColumn>
      <ElTableColumn prop="actor" label="操作者" :min-width="125" show-overflow-tooltip />
      <ElTableColumn label="动作" :min-width="125"><template #default="{ row }"><ElTag :type="auditStatus(row.action)">{{ auditNames[row.action] ?? row.action }}</ElTag></template></ElTableColumn>
      <ElTableColumn label="对象类型" :min-width="125"><template #default="{ row }">{{ objectNames[row.target_type] ?? row.target_type }}</template></ElTableColumn>
      <ElTableColumn label="对象标识（脱敏）" :min-width="145" show-overflow-tooltip><template #default="{ row }">{{ target(row as AuditRow) }}</template></ElTableColumn>
      <ElTableColumn label="变更摘要" :min-width="125" show-overflow-tooltip><template #default="{ row }">{{ changes(row as AuditRow) }}</template></ElTableColumn>
      <ElTableColumn label="请求 ID" :min-width="125" show-overflow-tooltip><template #default="{ row }">{{ row.request_id || '—' }}</template></ElTableColumn>
      <ElTableColumn label="操作" :width="88" fixed="right"><template #default="{ row }"><ElButton type="primary" link @click="show(row as AuditRow)">详情</ElButton></template></ElTableColumn>
    </ElTable>
    <ElPagination v-if="page > 1 || more" class="admin-pagination" layout="prev, slot, next" :current-page="page"
      :page-count="page + (more ? 1 : 0)" :disabled="busy" @update:current-page="paginate"><span>第 {{ page }} 页</span></ElPagination>

    <ElDrawer v-model="drawer" :title="isRequests ? '调用详情 · 脱敏' : '审计详情 · 脱敏'" class="admin-drawer" append-to-body size="min(560px, 94vw)">
      <ElDescriptions v-if="requestDetail" class="admin-descriptions" :column="1" border :label-width="125">
        <ElDescriptionsItem label="请求 ID">{{ requestDetail.request_id }}</ElDescriptionsItem>
        <ElDescriptionsItem label="时间">{{ displayDate(requestDetail.created_at) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="渠道">{{ channelLabel(requestDetail.channel).label }}</ElDescriptionsItem>
        <ElDescriptionsItem label="密钥片段">{{ masked(requestDetail.key_hint) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="平台">{{ platformNames[requestDetail.platform] ?? (requestDetail.platform || '—') }}</ElDescriptionsItem>
        <ElDescriptionsItem label="作品 ID">{{ requestDetail.item_id || '—' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="类型">{{ mediaType(requestDetail.media_type) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="操作">{{ requestDetail.operation === 'parse' ? '解析' : requestDetail.operation }}</ElDescriptionsItem>
        <ElDescriptionsItem label="HTTP 状态">{{ requestDetail.http_status || '—' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="错误码">{{ requestDetail.error_code || '—' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="耗时">{{ requestDetail.duration_ms }} ms</ElDescriptionsItem>
        <ElDescriptionsItem label="缓存命中">{{ requestDetail.cache_hit == null ? '—' : requestDetail.cache_hit ? '是' : '否' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="来源 IP（脱敏）">{{ requestDetail.source_ip || '—' }}</ElDescriptionsItem>
      </ElDescriptions>
      <ElDescriptions v-if="auditDetail" class="admin-descriptions" :column="1" border :label-width="125">
        <ElDescriptionsItem label="时间">{{ displayDate(auditDetail.created_at) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="操作者">{{ auditDetail.actor }}</ElDescriptionsItem>
        <ElDescriptionsItem label="动作">{{ auditNames[auditDetail.action] ?? auditDetail.action }}</ElDescriptionsItem>
        <ElDescriptionsItem label="对象类型">{{ objectNames[auditDetail.target_type] ?? auditDetail.target_type }}</ElDescriptionsItem>
        <ElDescriptionsItem label="对象标识（脱敏）">{{ target(auditDetail) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="变更摘要">{{ changes(auditDetail) }}</ElDescriptionsItem>
        <ElDescriptionsItem label="请求 ID">{{ auditDetail.request_id || '—' }}</ElDescriptionsItem>
      </ElDescriptions>
    </ElDrawer>
  </div>
</template>
