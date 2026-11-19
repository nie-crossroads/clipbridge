<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { mediaURL } from '../api/client'
import type { Media } from '../types/media'
import ProIcon from './ProIcon.vue'

const props = defineProps<{ media: Media; platformName: string }>()
const video = ref<HTMLVideoElement>()
const preview = ref<HTMLDialogElement>()
const photo = ref('')
const part = ref(-1)
const variant = ref(0)
const started = ref(false)
const feedback = ref('')
const copyStatus = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined
const savingImages = ref(new Set<string>())
const unavailableImages = ref(new Set<string>())
const avatarFailed = ref(false)
const typeNames = { video: '视频', image: '图集', live: '实况', audio: '音频' }
const imageExtensions: Record<string, string> = { 'image/jpeg': '.jpg', 'image/png': '.png', 'image/webp': '.webp', 'image/avif': '.avif', 'image/gif': '.gif' }

const live = computed(() => props.media.type === 'live' ? props.media.live_photos[0] : undefined)
const main = computed(() => part.value < 0 ? props.media.video ?? live.value?.video : props.media.videos[part.value]?.video)
const sources = computed(() => main.value ? [main.value, ...(main.value.alternatives ?? [])].filter(s => mediaURL(s.url)) : [])
const source = computed(() => sources.value[variant.value])
const cover = computed(() => mediaURL(live.value?.image.url ?? props.media.cover))
const avatar = computed(() => mediaURL(props.media.author?.avatar))
const pictures = computed(() => props.media.images.filter(image => mediaURL(image.url)))
const audio = computed(() => mediaURL(props.media.music?.url))
const watermark = computed(() => source.value?.watermark_status === 'watermark_free' ? '无水印'
  : source.value?.watermark_status === 'watermarked' ? '有水印' : '')
const dateFormatter = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', timeZone: 'Asia/Shanghai' })
const publishedDate = computed(() => props.media.published_at
  ? dateFormatter.format(new Date(props.media.published_at)).replaceAll('/', '-')
  : '')
const duration = computed(() => {
  if (props.media.duration_seconds == null) return ''
  const total = Math.floor(props.media.duration_seconds)
  return String(Math.floor(total / 60)).padStart(2, '0') + ':' + String(total % 60).padStart(2, '0')
})
const description = computed(() => {
  const value = props.media.description?.replace(/一键三连[，,]\s*推荐给更多朋友[！!]?/g, '').trim()
  return value === '-' ? '' : value
})
const summary = computed(() => {
  const lines = [props.media.author?.name, publishedDate.value]
  if (pictures.value.length) lines.push(pictures.value.length + ' 张图片')
  lines.push(duration.value)
  return lines.filter(Boolean).join(' · ')
})
const firstResource = computed(() => mediaURL(source.value?.url ?? pictures.value[0]?.url) || audio.value || cover.value)

watch(() => props.media, () => { part.value = -1; variant.value = 0; started.value = false; feedback.value = ''; unavailableImages.value.clear(); avatarFailed.value = false })
onUnmounted(() => clearTimeout(copyTimer))
async function play() {
  started.value = true
  feedback.value = ''
  await nextTick()
  const target = video.value
  target?.focus({ preventScroll: true })
  try { await target?.play() }
  catch { if (target?.isConnected && !target.error) feedback.value = '请使用播放器内的播放按钮继续。' }
}
function selectPart() {
  variant.value = 0
  started.value = false
  feedback.value = ''
}
function handlePlaybackError() {
  if (variant.value + 1 < sources.value.length) {
    variant.value++
    void play()
    return
  }
  feedback.value = '无法播放此媒体，链接可能已过期。请重新解析。'
}
async function downloadImage(event: MouseEvent, url: string, label: string) {
  if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || unavailableImages.value.has(url)) return
  event.preventDefault()
  if (savingImages.value.has(url)) return

  savingImages.value.add(url)
  feedback.value = ''
  try {
    const response = await fetch(url, { credentials: 'omit', referrerPolicy: 'no-referrer' })
    if (!response.ok) throw new Error('Image request failed')
    const blob = await response.blob()
    if (!blob.type.startsWith('image/')) throw new Error('Expected an image')
    const download = document.createElement('a')
    const blobURL = URL.createObjectURL(blob)
    download.href = blobURL
    download.download = label + (imageExtensions[blob.type] ?? '')
    download.click()
    window.setTimeout(() => URL.revokeObjectURL(blobURL), 1000)
  } catch {
    if (cover.value === url || avatar.value === url) {
      unavailableImages.value.add(url)
      feedback.value = `平台限制了${label}下载。再次点击可打开原图。`
    }
  } finally {
    savingImages.value.delete(url)
  }
}
async function copy() {
  try { await navigator.clipboard.writeText(firstResource.value); copyStatus.value = '复制成功' }
  catch { copyStatus.value = '剪贴板不可用，请从下载链接的右键菜单复制地址。' }
  clearTimeout(copyTimer)
  copyTimer = setTimeout(() => { copyStatus.value = '' }, 3000)
}
function openPhoto(url: string) { photo.value = url; preview.value?.showModal() }
</script>

<template>
  <article class="result-card" :class="{ 'video-result': media.type === 'video' }">
    <div v-if="media.type === 'image'" class="result-preview media-grid" aria-label="图片预览">
      <div v-for="(image, index) in pictures" :key="image.url" class="media-tile">
        <button type="button" class="photo-trigger" :aria-label="'放大图片 ' + (index + 1)" @click="openPhoto(mediaURL(image.url))">
          <img :src="mediaURL(image.url)" :alt="'作品图片 ' + (index + 1)" width="400" height="400">
        </button>
        <a :href="mediaURL(image.url)" class="btn" target="_blank" rel="noopener noreferrer" download>
          <ProIcon name="download" :size="17" />保存图片 {{ index + 1 }}
        </a>
      </div>
    </div>
    <div v-else-if="media.type === 'audio'" class="result-preview player">
      <audio v-if="audio" :src="audio" controls preload="none" aria-label="作品音频"></audio>
    </div>
    <div v-else class="result-preview player">
      <video v-if="started && source" :key="source.url" ref="video" :src="mediaURL(source.url)" :poster="cover || undefined"
        controls playsinline preload="none" tabindex="0" :volume.prop="0.5" :aria-label="media.title || '作品视频'"
        @error="handlePlaybackError"></video>
      <template v-else>
        <img v-if="cover" :src="cover" alt="" width="1280" height="720" referrerpolicy="no-referrer">
        <button v-if="source" type="button" class="btn play-btn" @click="play"><ProIcon name="play" :size="18" />播放视频</button>
        <p v-else class="player-empty">当前没有可播放的直链。</p>
      </template>
    </div>
    <div class="result-info">
      <div v-if="media.type === 'video' && (media.author?.name || avatar)" class="video-author">
        <span class="video-author-avatar">
          <img v-if="avatar && !avatarFailed" :src="avatar" alt="" width="60" height="60" referrerpolicy="no-referrer" @error="avatarFailed = true">
          <ProIcon v-else name="user" :size="27" />
        </span>
        <p>{{ media.author?.name || '作者' }}</p>
      </div>
      <div :class="{ 'video-details': media.type === 'video' }">
        <template v-if="media.type === 'video'">
          <h3 class="details-heading"><ProIcon name="info" :size="17" />详情</h3>
          <dl class="details-list">
            <div class="details-item">
              <dt><span class="details-icon"><ProIcon name="pencil" :size="17" /></span>作品标题</dt>
              <dd>{{ media.title || '未提供标题' }}</dd>
            </div>
            <div class="details-item">
              <dt><span class="details-icon"><ProIcon name="box" :size="17" /></span>作品类型</dt>
              <dd>{{ typeNames[media.type] }} <span class="badge good">{{ platformName }}</span><span v-if="watermark" class="badge warn">{{ watermark }}</span></dd>
            </div>
            <div v-if="publishedDate" class="details-item">
              <dt><span class="details-icon"><ProIcon name="calendar" :size="17" /></span>发布时间</dt>
              <dd><time :datetime="media.published_at ?? undefined">{{ publishedDate }}</time></dd>
            </div>
          </dl>
        </template>
        <template v-else>
          <div class="row"><span class="badge good">{{ platformName }}</span><span class="badge">{{ typeNames[media.type] }}</span><span v-if="watermark" class="badge warn">{{ watermark }}</span></div>
          <h3 class="result-title">{{ media.title || '未提供标题' }}</h3>
          <div v-if="description || summary" class="details-extra">
            <p v-if="summary" class="media-info media-author">
              <img v-if="avatar" :src="avatar" width="22" height="22" alt="" referrerpolicy="no-referrer">
              {{ summary }}
            </p>
            <p v-if="description" class="muted result-description">{{ description }}</p>
          </div>
        </template>
      </div>
      <label v-if="media.videos.length > 1" class="field">分 P
        <select v-model.number="part" class="control" @change="selectPart">
          <option :value="-1">当前分 P</option>
          <option v-for="(item, index) in media.videos" :key="item.item_id" :value="index">{{ item.title || '分 P ' + (index + 1) }}</option>
        </select>
      </label>
      <p v-if="feedback" class="feedback" role="status">{{ feedback }}</p>
      <p v-for="warning in media.warnings" :key="warning" class="notice warning">{{ warning }}</p>
    </div>
    <div class="row result-actions">
      <a v-if="source" class="btn primary" :href="mediaURL(source.url)" target="_blank" rel="noopener noreferrer">
        <ProIcon name="download" :size="17" />{{ live ? '保存视频' : '下载视频' }}
      </a>
      <a v-else-if="audio && !source" class="btn primary" :href="audio" download target="_blank" rel="noopener noreferrer">
        <ProIcon name="download" :size="17" />下载音频
      </a>
      <a v-if="live && cover" class="btn" :href="cover" download target="_blank" rel="noopener noreferrer"><ProIcon name="download" :size="17" />保存实况图片</a>
      <a v-if="media.type === 'video' && cover" class="btn" :href="cover" download target="_blank" rel="noopener noreferrer"
        :aria-busy="savingImages.has(cover)" @click="downloadImage($event, cover, '封面')"><ProIcon name="image" :size="17" />下载封面</a>
      <a v-if="media.type === 'video' && avatar" class="btn" :href="avatar" download target="_blank" rel="noopener noreferrer"
        :aria-busy="savingImages.has(avatar)" @click="downloadImage($event, avatar, '作者头像')"><ProIcon name="user" :size="17" />下载作者头像</a>
      <button v-if="firstResource" type="button" class="btn" @click="copy"><ProIcon name="copy" :size="16" />{{ pictures.length ? '复制首图链接' : '复制直链' }}</button>
    </div>
    <div class="toast" :class="{ visible: copyStatus }" role="status">{{ copyStatus }}</div>
    <dialog ref="preview" class="photo-dialog" aria-label="图片预览">
      <div class="dialog-head"><h2>图片预览</h2><button type="button" class="btn square quiet" aria-label="关闭图片预览" @click="preview?.close()"><ProIcon name="close" /></button></div>
      <img v-if="photo" :src="photo" alt="放大的作品图片">
    </dialog>
  </article>
</template>
