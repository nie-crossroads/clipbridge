<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInputNumber, ElMessage, ElMessageBox, ElSwitch, ElTag, vLoading } from 'element-plus'
import { adminRequest } from '../../api/admin'
import { message } from '../../api/client'

type Field = 'web_rate' | 'web_window_seconds' | 'parse_concurrency' | 'request_days' | 'audit_days' | 'cache_items' | 'cache_mib' | 'cache_ttl_seconds'
type Settings = Record<Field, number>
interface SettingsResponse extends Settings { retention_scheduled: boolean; proxy_enabled: boolean }
type Action = 'clear_requests' | 'clear_cache'
const groups: { id: string; title: string; description: string; fields: { key: Field; label: string; unit: string; max: number }[] }[] = [
  { id: 'web', title: '前台访问限速', description: '公开网页按访问来源限制解析次数。限流计数在服务重启后重新开始。', fields: [
    { key: 'web_rate', label: '来源限速阈值', unit: '次 / 窗口', max: 10000 },
    { key: 'web_window_seconds', label: '窗口时长', unit: '秒', max: 3600 },
  ] },
  { id: 'concurrency', title: '全局并发', description: '限制同一时间可以执行的解析请求数量。', fields: [
    { key: 'parse_concurrency', label: '解析并发', unit: '个', max: 32 },
  ] },
  { id: 'retention', title: '数据保留', description: '保存配置后，服务每小时分批清理超期记录。', fields: [
    { key: 'request_days', label: '调用记录保留', unit: '天', max: 3650 },
    { key: 'audit_days', label: '审计日志保留', unit: '天', max: 3650 },
  ] },
  { id: 'cache', title: '缓存容量', description: '公开解析使用短期内存缓存。到达容量上限时优先移除最早到期的条目。', fields: [
    { key: 'cache_items', label: '项数', unit: '项', max: 2000 },
    { key: 'cache_mib', label: '总容量', unit: 'MiB', max: 128 },
    { key: 'cache_ttl_seconds', label: '最长有效期（TTL）', unit: '秒', max: 120 },
  ] },
]
const fields = groups.flatMap(group => group.fields)
const form = reactive<Partial<Settings>>({})
const baseline = ref<Partial<Settings>>({})
const loaded = ref(false)
const busy = ref(false)
const saving = ref(false)
const clearing = ref<Action>()
const failure = ref('')
const scheduled = ref(false)
const dirty = computed(() => fields.some(({ key }) => form[key] !== baseline.value[key]))
const complete = computed(() => fields.every(({ key, max }) => Number.isInteger(form[key]) && form[key]! >= 1 && form[key]! <= max))
const canSave = computed(() => complete.value && (dirty.value || !scheduled.value))

function apply(data: SettingsResponse) {
  for (const field of fields) form[field.key] = data[field.key]
  baseline.value = { ...form }
  scheduled.value = data.retention_scheduled
}
async function load() {
  busy.value = true
  failure.value = ''
  try {
    apply(await adminRequest('/settings'))
    loaded.value = true
  } catch (error) { failure.value = message(error) }
  finally { busy.value = false }
}
async function save() {
  if (saving.value || !canSave.value) return
  saving.value = true
  failure.value = ''
  try {
    apply(await adminRequest('/settings', { ...form }))
    ElMessage.success('设置已保存')
  } catch (error) { failure.value = message(error) }
  finally { saving.value = false }
}
async function clear(action: Action) {
  if (clearing.value) return
  const logs = action === 'clear_requests'
  const title = logs ? '清空调用记录' : '清空缓存'
  try {
    await ElMessageBox.confirm(
      logs ? '将永久删除全部调用记录和解析历史，保留审计日志及配额数据。' : '将清空全部解析缓存。下一次解析需要重新请求平台。',
      title, { type: 'warning', confirmButtonText: '确认' + title, cancelButtonText: '取消', confirmButtonClass: 'el-button--danger', customClass: 'admin-confirm' },
    )
  } catch { return }
  clearing.value = action
  try {
    await adminRequest('/settings/clear', { action })
    ElMessage.success(logs ? '调用记录已清空' : '缓存已清空')
  } catch (error) { ElMessage.error(message(error)) }
  finally { clearing.value = undefined }
}
onMounted(load)
</script>

<template>
  <div v-loading="busy" class="admin-stack">
    <ElAlert v-if="failure" type="error" :title="failure" :closable="false" show-icon>
      <ElButton v-if="!loaded" link @click="load">重新加载</ElButton>
    </ElAlert>
    <template v-if="loaded">
      <ElAlert v-if="dirty" type="warning" title="有未保存的更改，保存后生效。" :closable="false" show-icon />
      <ElForm label-position="top" class="admin-form admin-stack settings-form" :disabled="saving" @submit.prevent="save">
        <div class="admin-two-columns">
          <template v-for="group in groups" :key="group.id">
            <ElCard v-if="group.id === 'retention'" shadow="never" class="settings-proxy">
              <template #header><div class="admin-card-header"><h3>网页代理开关</h3><ElTag type="info">直连模式</ElTag></div></template>
              <div class="admin-stack">
                <div class="admin-control-row"><ElSwitch :model-value="false" disabled aria-label="启用网页代理" /><span>网页代理已关闭</span></div>
                <ElAlert type="info" title="媒体代理服务待接入，当前通过平台 CDN 直连。" show-icon :closable="false" />
              </div>
            </ElCard>
            <ElCard shadow="never">
              <template #header><h3>{{ group.title }}</h3></template>
              <p class="caption">{{ group.description }}</p>
              <div class="settings-fields" :class="{ 'settings-fields-three': group.fields.length === 3 }">
                <ElFormItem v-for="field in group.fields" :key="field.key" :label="field.label">
                  <ElInputNumber v-model="form[field.key]" :min="1" :max="field.max" :precision="0" />
                  <span class="caption">{{ field.unit }}</span>
                </ElFormItem>
                <ElFormItem v-if="group.id === 'concurrency'" label="代理并发">
                  <ElInputNumber :model-value="undefined" disabled placeholder="待接入" />
                </ElFormItem>
              </div>
              <p v-if="group.id === 'retention'" class="caption settings-footnote">{{ scheduled ? '保留策略已启用。' : '保存设置后启用自动清理。' }}</p>
            </ElCard>
          </template>
        </div>
        <ElCard shadow="never">
          <div class="admin-card-header">
            <span class="caption">{{ dirty ? '有未保存的更改' : scheduled ? '设置已同步' : '正在使用默认值' }}</span>
            <div class="admin-actions">
              <ElButton v-if="dirty" @click="Object.assign(form, baseline)">放弃更改</ElButton>
              <ElButton type="primary" native-type="submit" :disabled="!canSave" :loading="saving">保存配置</ElButton>
            </div>
          </div>
        </ElCard>
      </ElForm>
      <ElCard shadow="never" class="settings-danger">
        <template #header><div class="admin-card-header"><h3>危险操作</h3><ElTag type="danger">需确认</ElTag></div></template>
        <div class="admin-card-header settings-danger-body">
          <p class="caption">清空调用记录会删除解析历史。清空缓存后，后续请求会重新解析。</p>
          <div class="admin-actions">
            <ElButton type="danger" plain :loading="clearing === 'clear_requests'" :disabled="!!clearing" @click="clear('clear_requests')">清空调用记录</ElButton>
            <ElButton type="danger" plain :loading="clearing === 'clear_cache'" :disabled="!!clearing" @click="clear('clear_cache')">清空缓存</ElButton>
          </div>
        </div>
      </ElCard>
    </template>
  </div>
</template>
