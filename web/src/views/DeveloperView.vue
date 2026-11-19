<script setup lang="ts">
import { ref } from 'vue'
import ProHeader from '../components/ProHeader.vue'
import ProIcon from '../components/ProIcon.vue'
import './pro-base.css'
import './pro-site.css'

const endpoints = [['POST /api/v1/parse', '单条解析'], ['POST /api/v1/parse/batch', '批量解析'], ['GET /api/v1/platforms', '查询平台能力'], ['GET /api/v1/usage', '查看密钥用量']]
const tools = [['parse_share_url', '解析分享链接'], ['batch_parse_urls', '按顺序返回逐项结果'], ['list_supported_platforms', '查询平台能力'], ['get_usage', '查看密钥用量']]
const sample = 'POST /api/v1/parse\nAuthorization: Bearer <REST Key>\nContent-Type: application/json\n\n{"url":"https://example.com/share/123"}'
const copyStatus = ref('')
async function copy() {
  try { await navigator.clipboard.writeText(sample); copyStatus.value = '已复制请求示例' }
  catch { copyStatus.value = '剪贴板不可用，请手动复制请求示例。' }
}
</script>
<template>
  <div class="pro-page developer-page">
    <a class="skip" href="#main">跳到接口说明</a>
    <ProHeader developer />
    <main class="wrap doc" id="main">
      <span class="badge good">接口设计预览</span>
      <h1>开发者接口</h1>
      <p class="muted">REST 和 MCP 调用入口待接入。以下为计划提供的接入方式。</p>
      <section class="doc-block" aria-labelledby="rest">
        <h2 id="rest">REST API</h2>
        <p class="muted">在请求头中使用 <code>Authorization: Bearer &lt;REST Key&gt;</code>。</p>
        <div v-for="[route, label] in endpoints" :key="route" class="endpoint"><code>{{ route }}</code><span>{{ label }}</span></div>
        <pre>{{ sample }}</pre>
        <button type="button" class="btn" @click="copy"><ProIcon name="copy" :size="16" />复制请求示例</button>
        <p v-if="copyStatus" class="muted" role="status">{{ copyStatus }}</p>
      </section>
      <section class="doc-block" aria-labelledby="mcp">
        <h2 id="mcp">MCP</h2>
        <p class="muted">计划通过 Streamable HTTP 提供 <code>/mcp</code>，使用独立的 MCP 密钥。</p>
        <div v-for="[tool, label] in tools" :key="tool" class="endpoint"><code>{{ tool }}</code><span>{{ label }}</span></div>
      </section>
    </main>
  </div>
</template>
