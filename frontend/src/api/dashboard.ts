import axios from 'axios'

// 这个文件是“前端和后端接口文档之间的桥”。
// 推荐把它理解成 3 层：
//
// 第 1 层：类型定义
// 这里把 API 文档 v1.8 中的请求字段、响应字段写成 TypeScript 类型。
// 这样后面写 Vue 组件时，编辑器就能提示字段名，也能尽早发现拼错字段的问题。
//
// 第 2 层：请求函数
// 页面组件不会直接写 axios.get(...)，而是调用这里封装好的函数。
// 好处是：如果后端路径、鉴权方式或响应结构以后变了，只需要优先改这里。
//
// 第 3 层：图表数据转换
// 后端返回的是业务数据，ECharts 需要的是 nodes / links / lineStyle 等展示数据。
// 所以这里负责把“接口数据”加工成“图表能直接画的数据”。

// localStorage 里保存 Dashboard 登录态时使用的 key。
// 单独抽成常量，是为了避免多个地方手写字符串导致拼错。
const DASHBOARD_AUTH_STORAGE_KEY = 'netweaver.dashboard.auth'

// API 文档 v1.8 规定所有接口都返回 code / msg / data。
// 注意：失败时 data 会是 null，所以这里必须写成 T | null。
export interface ApiResponse<T> {
  code: number
  msg: string
  data: T | null
}

// POST /api/v1/auth/login 的请求体。
export interface DashboardLoginRequest {
  username: string
  password: string
}

// POST /api/v1/auth/login 成功后的 data。
export interface DashboardLoginData {
  token: string
  token_type: 'Bearer'
  expires_in: number
  expires_at: number
}

// 前端本地保存登录态时，在接口返回字段外额外加一个 saved_at。
// 这个字段不是后端返回的，只用于前端排查“这个 token 是什么时候保存的”。
export interface DashboardAuthStorage extends DashboardLoginData {
  saved_at: number
}

// GET /api/v1/dashboard/stats 的 data。
// v1.8 中 total_nodes、online_nodes、total_traffic_gb、controller_uptime_sec 是核心字段。
// 下面几个带 ? 的字段是扩展字段：后端有就显示或备用，没有也不影响页面运行。
export interface DashboardStatsData {
  total_nodes: number
  online_nodes: number
  total_traffic_gb: number
  controller_uptime_sec: number
  offline_nodes?: number
  total_edges?: number
  total_rx_bytes?: number
  total_tx_bytes?: number
  total_traffic_bytes?: number
  last_updated_at?: number
}

// 单个节点的数据结构，对应 GET /api/v1/dashboard/nodes 中 nodes 数组的每一项。
// 前端表格和关系图依赖核心字段；扩展字段用于悬浮卡片展示更多细节。
export interface DashboardNode {
  node_id: string
  hostname: string
  virtual_ip: string
  public_ip: string | null
  nat_type: string
  status: 'online' | 'offline'
  connected_peers: number
  machine_id?: string
  os?: string
  local_ip?: string
  public_port?: number
  current_rx_bytes?: number
  current_tx_bytes?: number
  last_seen?: number
}

export interface DashboardNodesData {
  nodes: DashboardNode[]
}

// 真实关系边，对应 GET /api/v1/dashboard/edges。
// 注意：这里的 type 是真实链路类型，不是 peers 接口里的 recommend_mode。
export interface DashboardEdge {
  source: string
  target: string
  type: 'p2p' | 'relay'
}

export interface DashboardEdgesData {
  edges: DashboardEdge[]
}

// metrics 的时间范围来自接口文档，只允许这三个值。
export type DashboardMetricsTimeRange = '1h' | '12h' | '24h'

// 折线图上的一个采样点，对应 GET /api/v1/dashboard/nodes/{node_id}/metrics。
export interface DashboardMetricPoint {
  timestamp: number
  latency_ms: number
}

export interface DashboardNodeMetricsData {
  metrics: DashboardMetricPoint[]
}

// 下面的 Graph* 类型不是后端直接返回的格式。
// 它们是为了 ECharts 关系图准备的中间格式。
export interface GraphNodeItem {
  id: string
  name: string
  symbolSize: number
  value: number
  hostname: string
  virtualIp: string
  publicIp: string
  natType: string
  status: DashboardNode['status']
  connectedPeers: number
  itemStyle: {
    color: string
  }
  label: {
    show: boolean
    color: string
    fontWeight: number
  }
}

export interface GraphLinkItem {
  id: string
  source: string
  target: string
  edgeType: DashboardEdge['type']
  relationText: 'P2P' | 'Relay'
  lineStyle: {
    color: string
    width: number
    curveness: number
  }
  label: {
    show: boolean
    formatter: string
    color: string
    fontWeight: number
  }
}

export interface DashboardGraphData {
  nodes: GraphNodeItem[]
  links: GraphLinkItem[]
}

// dashboardHttp 专门负责 NetWeaver 控制台相关接口请求。
// baseURL 的规则：
// - VITE_API_BASE_URL 有值：浏览器直接请求这个完整后端地址
// - VITE_API_BASE_URL 为空：浏览器请求 /api/...，再交给 Vite 代理转发
const dashboardHttp = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '',
  timeout: 10000
})

// 判断一个请求是不是 Dashboard 受保护接口。
// 登录接口 /api/v1/auth/login 不需要 JWT，dashboard/* 才需要。
const isDashboardProtectedUrl = (url?: string) => {
  return typeof url === 'string' && url.includes('/api/v1/dashboard/')
}

// 从 localStorage 读取登录态。
// 这里使用 try/catch，是因为 localStorage 中的内容可能被手动改坏。
export const getStoredDashboardAuth = (): DashboardAuthStorage | null => {
  const rawValue = window.localStorage.getItem(DASHBOARD_AUTH_STORAGE_KEY)

  if (!rawValue) {
    return null
  }

  try {
    const parsedValue = JSON.parse(rawValue) as DashboardAuthStorage

    if (!parsedValue.token || !parsedValue.expires_at) {
      return null
    }

    return parsedValue
  } catch {
    return null
  }
}

// 保存登录接口返回的 JWT。
// 后续 dashboard/* 请求会通过 axios 拦截器自动带上这个 token。
export const setDashboardAuth = (authData: DashboardLoginData) => {
  const storageValue: DashboardAuthStorage = {
    ...authData,
    saved_at: Math.floor(Date.now() / 1000)
  }

  window.localStorage.setItem(DASHBOARD_AUTH_STORAGE_KEY, JSON.stringify(storageValue))
}

// 清空登录态。遇到 401、用户主动退出或本地 token 过期时都会调用它。
export const clearDashboardAuth = () => {
  window.localStorage.removeItem(DASHBOARD_AUTH_STORAGE_KEY)
}

// 判断当前 token 是否还在有效期内。
// 这里预留 30 秒余量：如果 token 马上过期，就当作已经不可用了，避免刚发请求就过期。
export const hasValidDashboardAuth = () => {
  const authData = getStoredDashboardAuth()

  if (!authData) {
    return false
  }

  const nowSeconds = Math.floor(Date.now() / 1000)

  if (authData.expires_at <= nowSeconds + 30) {
    clearDashboardAuth()
    return false
  }

  return true
}

export const getDashboardToken = () => {
  return hasValidDashboardAuth() ? getStoredDashboardAuth()?.token ?? '' : ''
}

// 请求拦截器：每次请求 dashboard/* 前，自动加 Authorization: Bearer <jwt>。
// 组件层不用每个请求都手动写 header，这就是“请求层统一处理鉴权”的意义。
dashboardHttp.interceptors.request.use((config) => {
  if (isDashboardProtectedUrl(config.url)) {
    const token = getDashboardToken()

    if (token) {
      config.headers.set('Authorization', `Bearer ${token}`)
    }
  }

  return config
})

// 响应拦截器：如果 dashboard/* 返回 401，说明 token 缺失、错误或过期。
// 这里先清掉本地 token，页面组件捕获错误后会切回登录状态。
dashboardHttp.interceptors.response.use(
  (response) => response,
  (error) => {
    if (
      axios.isAxiosError(error) &&
      error.response?.status === 401 &&
      isDashboardProtectedUrl(error.config?.url)
    ) {
      clearDashboardAuth()
    }

    return Promise.reject(error)
  }
)

export const loginDashboard = async (
  payload: DashboardLoginRequest
): Promise<ApiResponse<DashboardLoginData>> => {
  const response = await dashboardHttp.post<ApiResponse<DashboardLoginData>>('/api/v1/auth/login', payload)

  return response.data
}

export const getDashboardStats = async (): Promise<ApiResponse<DashboardStatsData>> => {
  const response = await dashboardHttp.get<ApiResponse<DashboardStatsData>>('/api/v1/dashboard/stats')

  return response.data
}

export const getDashboardNodes = async (): Promise<ApiResponse<DashboardNodesData>> => {
  const response = await dashboardHttp.get<ApiResponse<DashboardNodesData>>('/api/v1/dashboard/nodes')

  return response.data
}

export const getDashboardEdges = async (): Promise<ApiResponse<DashboardEdgesData>> => {
  const response = await dashboardHttp.get<ApiResponse<DashboardEdgesData>>('/api/v1/dashboard/edges')

  return response.data
}

export const getNodeMetrics = async (
  nodeId: string,
  targetId: string,
  timeRange: DashboardMetricsTimeRange = '1h'
): Promise<ApiResponse<DashboardNodeMetricsData>> => {
  // v1.8 明确禁止 node_id == target_id。
  // 所以前端如果传了同一个节点，直接在这里拦住，不再把错误请求发给后端。
  if (nodeId === targetId) {
    throw new Error('target_id must be different from node_id')
  }

  const response = await dashboardHttp.get<ApiResponse<DashboardNodeMetricsData>>(
    `/api/v1/dashboard/nodes/${nodeId}/metrics`,
    {
      params: {
        target_id: targetId,
        time_range: timeRange
      }
    }
  )

  return response.data
}

const getNodeColor = (node: DashboardNode) => {
  if (node.status === 'offline') {
    return '#94a3b8'
  }

  if (node.nat_type === 'Full Cone') {
    return '#0f766e'
  }

  if (node.nat_type === 'Symmetric') {
    return '#0ea5e9'
  }

  return '#2563eb'
}

const getRelationText = (edgeType: DashboardEdge['type']): 'P2P' | 'Relay' => {
  return edgeType === 'relay' ? 'Relay' : 'P2P'
}

const getLinkColor = (edgeType: DashboardEdge['type']) => {
  return edgeType === 'relay' ? '#f59e0b' : '#64748b'
}

const buildGraphLinkId = (edge: DashboardEdge) => {
  return `${edge.source}__${edge.target}__${edge.type}`
}

export const buildDashboardGraphData = (
  nodesSource: DashboardNode[],
  edgesSource: DashboardEdge[]
): DashboardGraphData => {
  const nodes: GraphNodeItem[] = nodesSource.map((node) => ({
    id: node.node_id,
    name: node.hostname,
    symbolSize: Math.max(58, Math.min(84, 58 + node.connected_peers * 6)),
    value: node.connected_peers,
    hostname: node.hostname,
    virtualIp: node.virtual_ip,
    publicIp: node.public_ip || '未上报',
    natType: node.nat_type,
    status: node.status,
    connectedPeers: node.connected_peers,
    itemStyle: {
      color: getNodeColor(node)
    },
    label: {
      show: true,
      color: '#0f172a',
      fontWeight: 600
    }
  }))

  const nodeIds = new Set(nodesSource.map((node) => node.node_id))

  const links: GraphLinkItem[] = edgesSource
    .filter((edge) => nodeIds.has(edge.source) && nodeIds.has(edge.target) && edge.source !== edge.target)
    .map((edge) => {
      const relationText = getRelationText(edge.type)

      return {
        id: buildGraphLinkId(edge),
        source: edge.source,
        target: edge.target,
        edgeType: edge.type,
        relationText,
        lineStyle: {
          color: getLinkColor(edge.type),
          width: edge.type === 'relay' ? 4 : 3,
          curveness: edge.type === 'relay' ? 0.12 : 0.06
        },
        label: {
          show: true,
          formatter: relationText,
          color: '#1e293b',
          fontWeight: 700
        }
      }
    })

  return {
    nodes,
    links
  }
}
