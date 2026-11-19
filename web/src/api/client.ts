export class ApiError extends Error {
  constructor(public code: string, message: string, public status: number, public retryable = false, public requestId = '') {
    super(message)
  }
}
export async function request<T>(path: string, options?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, {
      ...options,
      credentials: path.startsWith('/api/web/') ? 'omit' : 'same-origin',
      headers: { Accept: 'application/json', ...(options?.body ? { 'Content-Type': 'application/json' } : {}), ...options?.headers },
    })
  } catch (error) {
    if (options?.signal?.aborted) throw error
    throw new ApiError('NETWORK_ERROR', '连接服务失败，请稍后重试。', 0, true)
  }
  const body = await response.json().catch(() => null)
  if (!response.ok || !body || !('data' in body)) {
    throw new ApiError(body?.error?.code ?? 'SERVICE_UNAVAILABLE',
      body?.error?.message ?? '服务暂时不可用，请稍后重试。', response.status,
      body?.error?.retryable ?? response.status >= 500, body?.request_id)
  }
  return body.data
}

export function post<T>(path: string, data: object, signal?: AbortSignal) {
  return request<T>(path, { method: 'POST', body: JSON.stringify(data), signal })
}
export function message(error: unknown) {
  return error instanceof ApiError ? error.message : '操作失败，请重试。'
}

const dateFormat = new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', dateStyle: 'medium', timeStyle: 'medium' })
export function displayDate(value: string | null) {
  return value ? dateFormat.format(new Date(value)) : '—'
}

export function mediaURL(value?: string | null) {
  try {
    const url = new URL(value ?? '')
    if (url.username || url.password) return ''
    if (url.protocol === 'https:' || (url.protocol === 'http:' && location.protocol === 'http:')) return url.href
  } catch { /* 尚无可用直链 */ }
  return ''
}
