<script setup lang="ts">
import { ElAlert, ElCard, ElDescriptions, ElDescriptionsItem, ElEmpty, ElTag } from 'element-plus'
import ParseForm from '../../components/ParseForm.vue'
import { platformNames } from './labels'
</script>

<template>
  <div class="admin-stack">
    <ParseForm>
      <template #default="{ result, failure, pending }">
        <ElCard v-if="result" shadow="never">
          <template #header><div class="admin-card-header"><h2>归一化结果</h2><ElTag type="success">成功</ElTag></div></template>
          <ElDescriptions class="admin-descriptions" :column="2" border>
            <ElDescriptionsItem label="标题">{{ result.title || '—' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="平台"><ElTag class="badge--platform" :data-platform="result.platform">{{ platformNames[result.platform] ?? result.platform }}</ElTag></ElDescriptionsItem>
            <ElDescriptionsItem label="作品 ID"><span class="mono">{{ result.item_id }}</span></ElDescriptionsItem>
            <ElDescriptionsItem label="类型">{{ result.type }}</ElDescriptionsItem>
            <ElDescriptionsItem label="直链">{{ result.video || result.music || result.images.length || result.live_photos.length ? '已返回资源' : '—' }}</ElDescriptionsItem>
          </ElDescriptions>
          <ElAlert v-for="warning in result.warnings" :key="warning" :title="warning" type="warning" show-icon :closable="false" class="admin-test-notice" />
        </ElCard>
        <ElAlert v-else-if="failure" :title="failure.message" type="error" show-icon :closable="false">
          <p class="mono">{{ failure.code }} <span v-if="failure.requestId">{{ failure.requestId }}</span></p>
        </ElAlert>
        <ElCard v-else shadow="never"><ElEmpty :description="pending ? '正在验证平台…' : '粘贴一条分享文本开始验证'" /></ElCard>
      </template>
    </ParseForm>
  </div>
</template>
