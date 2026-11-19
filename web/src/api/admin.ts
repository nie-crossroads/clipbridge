import { shallowRef } from 'vue'
import { ApiError, request } from './client'

export const session = shallowRef<{ name: string; csrf_token: string } | null>(null)

export async function refreshSession() {
  session.value = await request('/api/admin/session')
}

export async function adminRequest<T>(path: string, body?: object, signal?: AbortSignal) {
  try {
    return await request<T>('/api/admin' + path, {
      signal,
      ...(body && { method: 'POST', body: JSON.stringify(body), headers: { 'X-CSRF-Token': session.value?.csrf_token ?? '' } }),
    })
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      session.value = null
      if (location.pathname !== '/admin/login') location.replace('/admin/login')
    }
    throw error
  }
}
