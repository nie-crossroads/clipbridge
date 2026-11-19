import { onBeforeUnmount, ref, shallowRef } from 'vue'
import { ApiError } from './client'
import type { Media } from '../types/media'

type ParseBody = { url: string; platform?: string }

export function useShareParser(request: (body: ParseBody, signal: AbortSignal) => Promise<Media>) {
  const result = shallowRef<Media>()
  const failure = shallowRef<ApiError>()
  const pending = ref(false)
  let active: AbortController | undefined

  function reset() {
    active?.abort()
    active = undefined
    pending.value = false
    result.value = undefined
    failure.value = undefined
  }

  async function parse(text: string, platform?: string) {
    if (pending.value) return
    reset()
    const url = text.trim()
    if (!url) {
      failure.value = new ApiError('INVALID_INPUT', '请先粘贴分享链接或文案。', 400)
      return
    }
    const controller = new AbortController()
    active = controller
    pending.value = true
    try {
      const data = await request({ url, ...(platform && { platform }) }, controller.signal)
      if (active === controller) result.value = data
    } catch (error) {
      if (active === controller) {
        failure.value = error instanceof ApiError ? error : new ApiError('NETWORK_ERROR', '连接服务失败。', 0, true)
      }
    } finally {
      if (active === controller) {
        active = undefined
        pending.value = false
      }
    }
  }

  onBeforeUnmount(reset)
  return { result, failure, pending, parse, reset }
}
