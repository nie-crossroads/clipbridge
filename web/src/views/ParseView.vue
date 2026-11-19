<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { message, post, request } from '../api/client'
import { useShareParser } from '../api/share-parser'
import type { Media, Platform } from '../types/media'
import ProHeader from '../components/ProHeader.vue'
import ProIcon from '../components/ProIcon.vue'
import MediaResult from '../components/MediaResult.vue'
import './pro-base.css'
import './pro-site.css'

const featured = [
  { code: 'douyin', name: '抖音' },
  { code: 'bilibili', name: '哔哩哔哩' },
  { code: 'xiaohongshu', name: '小红书' },
  { code: 'kuaishou', name: '快手' },
  { code: 'weibo', name: '微博' },
]
const platformIcons = import.meta.glob<string>('../assets/platforms/*.png', { eager: true, import: 'default', query: '?url' })
function platformIcon(code: string) {
  return platformIcons[`../assets/platforms/${code}.png`]
}
const guides = [
  { icon: 'link', title: '复制分享链接', text: '打开平台作品，点击分享并复制链接或分享文案。' },
  { icon: 'clipboard', title: '粘贴并解析', text: '把内容粘贴到输入框。ClipBridge 会自动识别链接所属平台。' },
  { icon: 'download', title: '预览或保存', text: '在结果中播放视频，或按需保存视频、图片和实况资源。' },
]
const faqs = [
  { title: '为什么有些视频无法解析？', text: '部分平台会更新防护策略，可能导致暂时失败，我们会持续维护。受版权限制的内容也可能无法解析。' },
  { title: '下载的视频仍有水印怎么办？', text: '若仍有水印，可能是平台策略变化导致去水印逻辑需更新，欢迎反馈以便我们尽快处理。' },
  { title: '为什么点下载后打开了视频页面？', text: '直链通常由平台 CDN 提供，浏览器可能直接打开媒体。此时可用浏览器的「另存为」功能保存。' },
  { title: '解析会移除画面中的水印吗？', text: '解析会返回平台提供的媒体资源。资源的水印状态以结果标记为准，烧录在画面中的水印会随原有画面保留。' },
]
const dots = [[8,26,4,-2], [14,14,7,-6], [18,67,5,-8], [25,31,3,-1], [31,12,4,-11], [65,15,4,-5], [76,27,5,-8], [84,57,4,-3], [89,20,6,-10], [94,38,3,-5]]
const capabilities: Record<string, string> = { image: '图集', live: '实况', audio: '音频' }
const platforms = ref<Platform[]>([])
const platformError = ref('')
const supportedPlatforms = computed(() => platforms.value.filter(item => item.implemented))
const expandedQuestions = ref<Record<number, boolean>>({})
const text = ref('')
const selected = ref('')
const inlineError = ref('')
const input = ref<HTMLInputElement>()
const resultSection = ref<HTMLElement>()
const resultHeading = ref<HTMLElement>()
const { result, failure, pending, parse, reset } = useShareParser((body, signal) => post<Media>('/api/web/parse', body, signal))
const problem = computed(() => inlineError.value ? undefined : failure.value)
const parsedName = computed(() => {
  const code = result.value?.platform
  return featured.find(item => item.code === code)?.name ?? platforms.value.find(item => item.platform === code)?.name ?? code ?? '未知'
})

onMounted(async () => {
  try { platforms.value = await request('/api/web/platforms') }
  catch (error) { platformError.value = message(error) }
})
function platformText(platform: Platform) {
  if (!platform.enabled) return '暂未开放'
  return platform.capabilities.filter(type => type !== 'video').map(type => capabilities[type] ?? type).join('、')
}
function selectPlatform(code: string) {
  selected.value = code
  clearInlineError()
}
function clearInlineError() {
  if (inlineError.value) reset()
  inlineError.value = ''
}
function pasteShare(event: ClipboardEvent) {
  const pasted = event.clipboardData?.getData('text/plain') ?? ''
  const field = input.value
  if (!field || !/[\r\n]/.test(pasted)) return

  event.preventDefault()
  const start = field.selectionStart ?? field.value.length
  const end = field.selectionEnd ?? start
  const available = field.maxLength - (field.value.length - (end - start))
  field.setRangeText(pasted.replace(/[\r\n]+/g, ' ').slice(0, Math.max(available, 0)), start, end, 'end')
  text.value = field.value
  clearInlineError()
}
async function revealResult() {
  await nextTick()
  resultHeading.value?.focus({ preventScroll: true })
  resultSection.value?.scrollIntoView({ block: 'start' })
}
async function submit() {
  if (pending.value) return
  inlineError.value = ''
  reset()
  const value = text.value
  if (!/https?:\/\//i.test(value)) {
    inlineError.value = '请粘贴包含 http 或 https 链接的分享内容。'
    input.value?.focus()
    return
  }
  await parse(value, selected.value)
  if (failure.value && ['INVALID_INPUT', 'UNSUPPORTED_URL'].includes(failure.value.code)) {
    inlineError.value = failure.value.message
    input.value?.focus()
    return
  }
  if (result.value || problem.value) await revealResult()
}
</script>

<template>
  <div class="pro-page parse-page">
    <a class="skip" href="#main">跳到解析工具</a>
    <ProHeader />
    <main id="main">
      <section class="hero" id="parser" aria-labelledby="hero-title">
        <div class="hero-atmosphere" aria-hidden="true">
          <div class="hero-glow"></div>
          <span v-for="([x, y, size, delay], index) in dots" :key="index" class="hero-dot" :style="{ '--x': x + '%', '--y': y + '%', '--size': size + 'px', '--delay': delay + 's' }"></span>
        </div>
        <div class="wrap hero-inner">
          <div class="hero-copy">
            <h1 id="hero-title"><span>多平台</span>视频解析</h1>
            <p class="hero-desc">粘贴抖音、B 站、小红书等平台的分享链接，一键提取无水印视频，简单免费。</p>
          </div>
          <form class="parser" id="parse-form" novalidate @submit.prevent="submit()">
            <div class="platform-options" id="platform-options" role="group" aria-label="选择解析平台">
              <button type="button" :aria-pressed="!selected" @click="selectPlatform('')"><ProIcon name="layers" :size="16" />自动识别</button>
              <button v-for="item in featured" :key="item.code" type="button" :aria-pressed="selected === item.code" @click="selectPlatform(item.code)">
                <img v-if="platformIcon(item.code)" class="platform-icon" :src="platformIcon(item.code)" alt="" width="22" height="22">{{ item.name }}
              </button>
            </div>
            <div class="parser-body">
              <label for="share">分享链接或分享文案</label>
              <input id="share" ref="input" v-model="text" type="text" class="control share-input" maxlength="500"
                autocomplete="off" autocapitalize="off" :spellcheck="false"
                placeholder="在此粘贴分享链接或包含链接的分享文案…" aria-describedby="share-error"
                :aria-invalid="!!inlineError" @input="clearInlineError" @paste="pasteShare">
              <span class="feedback" id="share-error" aria-live="polite">{{ inlineError }}</span>
              <div class="parse-actions" :aria-busy="pending">
                <button type="submit" class="btn primary" :disabled="pending" :aria-busy="pending">
                  {{ pending ? '正在解析…' : '开始解析' }}
                  <span v-if="pending" class="parse-spinner" aria-hidden="true"></span>
                  <ProIcon v-else name="arrow" :size="19" />
                </button>
              </div>
            </div>
          </form>
        </div>
      </section>
      <section v-if="result || problem" ref="resultSection" class="wrap result-section" id="result" aria-labelledby="result-heading">
        <div class="section-head centered">
          <h2 ref="resultHeading" tabindex="-1" id="result-heading">解析结果</h2>
        </div>
        <MediaResult v-if="result" :media="result" :platform-name="parsedName" />
        <div v-else-if="problem" class="notice warning result-error" role="status">
          <div class="row"><ProIcon name="warning" :size="21" /><strong>{{ problem.code === 'UPSTREAM_TIMEOUT' ? '平台响应超时' : '暂时无法解析' }}</strong><code>{{ problem.code }}</code></div>
          <p>{{ problem.message }}</p>
        </div>
      </section>
      <section class="wrap under" id="platforms" aria-labelledby="platform-heading">
        <div class="section-head centered"><div><h2 id="platform-heading">支持平台</h2><p>查看各平台的媒体能力和接入状态。</p></div></div>
        <p v-if="platformError" class="notice warning" role="status">{{ platformError }}</p>
        <div v-if="supportedPlatforms.length" class="platform-marquee" id="platform-marquee" role="region" aria-label="平台列表" tabindex="0">
          <div class="platform-track">
            <ul v-for="repeat in 2" :key="repeat" class="platform-set" :class="{ 'platform-repeat': repeat === 2 }" :aria-hidden="repeat === 2 || undefined" role="list">
              <li v-for="item in supportedPlatforms" :key="item.platform" class="platform-entry">
                <article class="platform-card">
                  <img v-if="platformIcon(item.platform)" class="platform-icon" :src="platformIcon(item.platform)" alt="" width="48" height="48">
                  <h3>{{ item.name }}</h3><p v-if="platformText(item)">{{ platformText(item) }}</p>
                </article>
              </li>
            </ul>
          </div>
        </div>
      </section>
      <section class="wrap under guide" id="guide" aria-labelledby="guide-heading">
        <div class="section-head centered"><h2 id="guide-heading">使用指南</h2></div>
        <ol class="steps"><li v-for="item in guides" :key="item.title"><span class="step-icon"><ProIcon :name="item.icon" /></span><h3>{{ item.title }}</h3><p>{{ item.text }}</p></li></ol>
      </section>
      <section class="wrap under questions" id="faq" aria-labelledby="faq-heading">
        <div class="section-head centered"><h2 id="faq-heading">常见问题</h2></div>
        <div class="faq">
          <div v-for="(item, index) in faqs" :key="item.title" class="faq-item">
            <h3>
              <button type="button" :aria-expanded="!!expandedQuestions[index]" :aria-controls="'faq-answer-' + index" @click="expandedQuestions[index] = !expandedQuestions[index]">
                {{ item.title }}<ProIcon name="plus" :size="18" />
              </button>
            </h3>
            <div :id="'faq-answer-' + index" class="faq-answer" :aria-hidden="!expandedQuestions[index]">
              <div><p>{{ item.text }}</p></div>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>
