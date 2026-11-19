<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElAlert, ElButton, ElCard, ElProgress, ElSegmented, ElStatistic, ElTag, vLoading } from 'element-plus'
import { adminRequest } from '../../api/admin'
import { message } from '../../api/client'

interface Stats {
  accepted: number; success: number; proxy_bytes: number; rss_bytes: number | null; heap_bytes: number
  db_open: number; db_in_use: number; db_idle: number; db_max: number; db_wait_count: number
}
const stats = ref<Stats>()
const busy = ref(false)
const failure = ref('')
const range = ref('24h')
const ranges = [{ label: '近 24 小时', value: '24h' }, { label: '近 7 天', value: '7d' }, { label: '近 14 天', value: '14d' }]
const mib = 1048576
const memoryScale = computed(() => Math.max(64, Math.ceil(Math.max(stats.value?.rss_bytes ?? 0, stats.value?.heap_bytes ?? 0) / mib / 64) * 64))
const percent = (value: number, scale: number) => scale ? Math.min(100, Math.round(100 * value / scale)) : 0
const memory = (bytes: number | null) => bytes == null ? '—' : (bytes / mib).toFixed(1) + ' MiB'
const metrics = computed(() => {
  const data = stats.value
  if (!data) return []
  return [
    { label: '解析次数', value: data.accepted.toLocaleString('zh-CN') },
    { label: '成功率', value: data.accepted ? (100 * data.success / data.accepted).toFixed(1) + '%' : '—' },
    { label: '代理流量', value: data.proxy_bytes.toLocaleString('zh-CN') + ' B' },
    { label: '应用 RSS / 堆内存', value: memory(data.rss_bytes) + ' / ' + memory(data.heap_bytes) },
  ]
})
async function load() {
  busy.value = true
  failure.value = ''
  try { stats.value = await adminRequest('/dashboard?range=' + range.value) }
  catch (error) { failure.value = message(error) }
  finally { busy.value = false }
}
onMounted(load)
</script>

<template>
  <div v-loading="busy" class="admin-stack" :aria-busy="busy">
    <ElAlert v-if="failure" type="error" :title="failure" show-icon :closable="false"><ElButton link @click="load">重新加载</ElButton></ElAlert>
    <div><ElSegmented v-model="range" :options="ranges" aria-label="时间范围" :disabled="busy" @change="load" /></div>
    <template v-if="stats">
      <h2 class="admin-section-heading">关键指标</h2>
      <div class="grid-4">
        <ElCard v-for="metric in metrics" :key="metric.label" shadow="never">
          <ElStatistic :title="metric.label" :formatter="() => metric.value" />
        </ElCard>
      </div>
      <div class="admin-card-header"><h2 class="admin-section-heading">运行状态</h2><span class="caption">本次页面刷新时采样</span></div>
      <div class="admin-two-columns">
        <ElCard shadow="never">
          <template #header><div class="admin-card-header"><h3>应用内存</h3><ElTag :type="stats.rss_bytes == null ? 'info' : 'success'">{{ stats.rss_bytes == null ? 'RSS 不可用' : '采样正常' }}</ElTag></div></template>
          <div class="admin-stack">
            <div>
              <div class="admin-meter-label"><span>进程 RSS <span class="caption">（ClipBridge）</span></span><span class="mono">{{ memory(stats.rss_bytes) }}</span></div>
              <ElProgress v-if="stats.rss_bytes != null" :percentage="percent(stats.rss_bytes, memoryScale * mib)" :show-text="false" aria-label="进程 RSS" />
            </div>
            <div>
              <div class="admin-meter-label"><span>Go 堆内存</span><span class="mono">{{ memory(stats.heap_bytes) }}</span></div>
              <ElProgress :percentage="percent(stats.heap_bytes, memoryScale * mib)" :show-text="false" color="var(--cb-accent)" aria-label="Go 堆内存" />
            </div>
            <p class="caption">进度条参考刻度 {{ memoryScale }} MiB。RSS 统计 ClipBridge 进程。</p>
          </div>
        </ElCard>
        <ElCard shadow="never">
          <template #header><div class="admin-card-header"><h3>数据库连接池</h3><ElTag type="success">数据库已连接</ElTag></div></template>
          <div class="admin-stack">
            <div>
              <div class="admin-meter-label"><span>已用 / 最大</span><span class="mono">{{ stats.db_in_use }} / {{ stats.db_max }}</span></div>
              <ElProgress :percentage="percent(stats.db_in_use, stats.db_max)" :show-text="false" aria-label="已用连接" />
            </div>
            <div>
              <div class="admin-meter-label"><span>空闲</span><span class="mono">{{ stats.db_idle }}</span></div>
              <ElProgress :percentage="percent(stats.db_idle, stats.db_max)" :show-text="false" color="var(--cb-accent)" aria-label="空闲连接" />
            </div>
            <p class="caption">当前打开 {{ stats.db_open }} 个连接，累计等待 {{ stats.db_wait_count }} 次。</p>
          </div>
        </ElCard>
      </div>
    </template>
  </div>
</template>
