<script setup lang="ts">
import { ref } from 'vue'
import { ElButton, ElCard, ElForm, ElFormItem, ElInput } from 'element-plus'
import { adminRequest } from '../api/admin'
import { useShareParser } from '../api/share-parser'
import type { Media } from '../types/media'
import Icon from './Icon.vue'

const text = ref('')
const { result, failure, pending, parse, reset } = useShareParser((body, signal) => adminRequest<Media>('/parse', body, signal))
</script>
<template>
  <ElCard class="parse-admin" shadow="never">
    <ElForm class="admin-control-row admin-actions" @submit.prevent="parse(text)">
      <ElFormItem>
        <ElInput v-model="text" size="large" maxlength="500" show-word-limit aria-label="分享链接或分享文本" placeholder="粘贴分享链接或包含链接的文本…" />
      </ElFormItem>
      <ElButton type="primary" size="large" native-type="submit" :loading="pending">
        <template #icon><Icon name="search" /></template>开始解析
      </ElButton>
      <ElButton v-if="pending" size="large" @click="reset">取消</ElButton>
    </ElForm>
  </ElCard>
  <div class="admin-result">
    <slot :result="result" :failure="failure" :pending="pending" />
  </div>
</template>
