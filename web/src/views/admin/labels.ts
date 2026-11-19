import type { TagProps } from 'element-plus'

export const channels = [
  { value: 'web', label: '公开网页', type: 'primary' },
  { value: 'admin', label: '后台', type: 'info' },
  { value: 'rest', label: 'REST', type: 'primary' },
  { value: 'mcp', label: 'MCP', type: 'success' },
  { value: 'proxy', label: '代理', type: 'warning' },
] satisfies { value: string; label: string; type: TagProps['type'] }[]

export const platformNames: Record<string, string> = {
  douyin: '抖音', kuaishou: '快手', bilibili: 'B站', xiaohongshu: '小红书', weibo: '微博',
  toutiao: '今日头条', pipixia: '皮皮虾', pipigx: '皮皮搞笑', zuiyou: '最右', jimeng: '即梦', doubao: '豆包', netease: '网易云音乐',
}

export const capabilityNames: Record<string, string> = { video: '视频', image: '图集', live: '实况', audio: '音频' }

export function channelLabel(value: string) {
  return channels.find(item => item.value === value) ?? { label: value, type: 'info' as const }
}
