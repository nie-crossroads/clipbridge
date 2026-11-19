<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElAlert, ElButton, ElInputNumber, ElMessage, ElTable, ElTableColumn, ElTag, vLoading } from 'element-plus'
import { adminRequest } from '../../api/admin'
import { displayDate, message } from '../../api/client'
import type { Platform } from '../../types/media'
import { capabilityNames } from './labels'

const platforms = ref<Platform[]>([])
const edits = ref<Record<string, number | undefined>>({})
const loading = ref(false)
const saving = ref('')
const failure = ref('')

async function load() {
  loading.value = true
  failure.value = ''
  try {
    const items = await adminRequest<Platform[]>('/platforms')
    platforms.value = items
    edits.value = Object.fromEntries(items.map(item => [item.platform, item.timeout_seconds]))
  } catch (error) { failure.value = message(error) }
  finally { loading.value = false }
}

async function save(item: Platform, change: { enabled?: boolean; timeout_seconds?: number }) {
  saving.value = item.platform
  try {
    const saved = await adminRequest<typeof change>('/platforms/' + item.platform, change)
    Object.assign(item, saved)
    ElMessage.success(saved.timeout_seconds !== undefined ? '平台超时已保存' : item.name + (item.enabled ? '已启动' : '已停用'))
  } catch (error) { ElMessage.error(message(error)) }
  finally { saving.value = '' }
}
onMounted(load)
</script>

<template>
  <div class="admin-stack">
    <ElAlert v-if="failure" type="error" :title="failure" show-icon :closable="false" />
    <ElButton v-if="failure" @click="load">重新加载</ElButton>
    <ElTable v-loading="loading" :data="platforms" row-key="platform" border class="admin-table" empty-text="暂无平台">
      <ElTableColumn label="平台" :min-width="135">
        <template #default="{ row }"><ElTag class="platform-tag badge--platform" :data-platform="row.platform">{{ row.name }}</ElTag></template>
      </ElTableColumn>
      <ElTableColumn label="适配器" :min-width="135">
        <template #default="{ row }"><ElTag :type="row.implemented ? 'success' : 'info'">{{ row.implemented ? '已实现' : '待接入' }}</ElTag></template>
      </ElTableColumn>
      <ElTableColumn label="前台开放" :min-width="135">
        <template #default="{ row }"><ElTag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '已开放' : '未开放' }}</ElTag></template>
      </ElTableColumn>
      <ElTableColumn label="能力" :min-width="135">
        <template #default="{ row }">
          <div v-if="row.capabilities.length" class="admin-tags">
            <ElTag v-for="item in row.capabilities" :key="item" type="primary">{{ capabilityNames[item] ?? item }}</ElTag>
          </div>
          <ElTag v-else type="info">暂无</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="超时（秒）" :min-width="135">
        <template #default="{ row }">
          <div v-if="row.implemented" class="timeout-control">
            <ElInputNumber v-model="edits[row.platform]" :min="1" :max="25" :precision="0" :controls="false" size="small" :aria-label="row.name + '超时秒数'" />
            <ElButton size="small" type="primary" link :disabled="!!saving || !edits[row.platform] || edits[row.platform] === row.timeout_seconds"
              @click="save(row as Platform, { timeout_seconds: edits[row.platform] })">保存</ElButton>
          </div>
          <span v-else>—</span>
        </template>
      </ElTableColumn>
      <ElTableColumn label="最近测试" :min-width="230" show-overflow-tooltip>
        <template #default="{ row }">
          <div class="platform-last-test">
            <ElTag :type="row.last_test_status === 'success' ? 'success' : row.last_test_status ? 'danger' : 'info'">
              {{ row.last_test_status === 'success' ? '通过' : row.last_test_status ? '失败' : '尚未测试' }}
            </ElTag>
            <span v-if="row.last_tested_at" class="caption">{{ displayDate(row.last_tested_at) }}</span>
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="操作" :width="92">
        <template #default="{ row }">
          <ElButton v-if="row.implemented" :type="row.enabled ? 'danger' : 'success'" plain size="small"
            :disabled="!!saving" :aria-label="(row.enabled ? '停用 ' : '启动 ') + row.name"
            @click="save(row as Platform, { enabled: !row.enabled })">{{ row.enabled ? '停用' : '启动' }}</ElButton>
          <span v-else>—</span>
        </template>
      </ElTableColumn>
    </ElTable>
    <ElButton type="primary" link @click="$router.push('/admin/parse-test')">前往解析测试</ElButton>
  </div>
</template>
