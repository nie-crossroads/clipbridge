export const groups = [
  { label: '工作台', pages: [
    { id: 'dashboard', title: '概览', icon: 'dashboard', description: '查看解析用量与当前服务状态。' },
    { id: 'parse-test', title: '解析测试', icon: 'link', description: '验证平台适配器并查看归一化结果。' },
  ] },
  { label: '资源', pages: [
    { id: 'platforms', title: '平台配置', icon: 'platforms', description: '管理平台开放状态和请求超时，查看最近一次测试。' },
    { id: 'keys', title: '密钥管理', icon: 'keys', description: '管理 REST 与 MCP 密钥、权限与额度。' },
  ] },
  { label: '运维', pages: [
    { id: 'requests', title: '调用记录', icon: 'requests', description: '查看解析结果、耗时与错误码。' },
    { id: 'settings', title: '系统设置', icon: 'settings', description: '配置访问限速、并发、日志保留和解析缓存。' },
    { id: 'audit', title: '审计日志', icon: 'audit', description: '查询管理员操作、配置变更和密钥事件。' },
  ] },
]
export const pages = groups.flatMap(group => group.pages)
