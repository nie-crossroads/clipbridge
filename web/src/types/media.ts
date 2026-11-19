export interface Platform {
  platform: string
  name: string
  implemented: boolean
  enabled: boolean
  timeout_seconds?: number
  capabilities: string[]
  last_test_status: string | null
  last_tested_at: string | null
}
export interface Source {
  url: string
  quality: string
  watermark_status: string
  mime_type: string | null
  codec: string | null
  width: number | null
  height: number | null
  bitrate: number | null
  size_bytes: number | null
  expires_at: string | null
  alternatives?: Source[]
}
export interface Media {
  platform: string
  item_id: string
  type: 'video' | 'image' | 'live' | 'audio'
  title: string
  description: string
  author: { id: string; name: string; avatar: string } | null
  cover: string
  duration_seconds: number | null
  published_at: string | null
  video: Source | null
  videos: { item_id: string; title: string; video: Source | null }[]
  images: { url: string }[]
  live_photos: { image: { url: string }; video: Source }[]
  music: Source | null
  warnings: string[]
}
