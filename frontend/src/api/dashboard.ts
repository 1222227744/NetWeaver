import axios from 'axios'

// 这个文件是“前端请求层 + 图数据转换层”。
// 建议把它理解成两个职责：
//
// 第一部分：定义接口返回的数据长什么样
// 例如 stats、nodes、edges、metrics 这些接口返回什么字段
//
// 第二部分：提供请求函数和转换函数
// - 请求函数：负责向后端发请求
// - 转换函数：负责把接口数据加工成 ECharts 能直接使用的格式
//
// 如果以后页面显示不对，排查顺序通常是：
// 1. 后端接口有没有返回
// 2. 这里的 TypeScript 类型有没有和后端字段对齐
// 3. 这里的转换函数有没有把数据正确转换给图表

// 全局响应结构按照 API 文档统一定义。
// 这样后面无论接真实后端还是继续扩展其他接口，返回值结构都能保持一致。
export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

// DashboardStatsData 对应接口：
// GET /api/v1/dashboard/stats
//
// 它描述的是“整个控制平面的总览统计”，
// 不是某一个节点自己的数据。
export interface DashboardStatsData {
  total_nodes: number
  online_nodes: number
  total_traffic_gb: number
  controller_uptime_sec: number
}

// 单个节点的数据结构直接对应接口文档里的 nodes 数组元素。
export interface DashboardNode {
  node_id: string
  hostname: string
  virtual_ip: string
  public_ip: string | null
  nat_type: string
  status: 'online' | 'offline'
  connected_peers: number
}

// DashboardNodesData 对应接口：
// GET /api/v1/dashboard/nodes
//
// 后端返回的不是“直接一个数组”，
// 而是 { data: { nodes: [...] } } 这种结构，
// 所以这里要再包一层 nodes。
export interface DashboardNodesData {
  nodes: DashboardNode[]
}

// DashboardEdge 表示节点之间的一条连线。
// source 是起点节点 ID，target 是终点节点 ID。
// type 用来区分这条线到底是 P2P 直连，还是 Relay 中转。
export interface DashboardEdge {
  source: string
  target: string
  type: 'p2p' | 'relay'
}

// DashboardEdgesData 对应接口：
// GET /api/v1/dashboard/edges
export interface DashboardEdgesData {
  edges: DashboardEdge[]
}

// DashboardMetricPoint 表示折线图上的一个点。
// timestamp 是时间戳，latency_ms 是这个时间点的延迟值。
export interface DashboardMetricPoint {
  timestamp: number
  latency_ms: number
}

// DashboardNodeMetricsData 对应接口：
// GET /api/v1/dashboard/nodes/{node_id}/metrics
export interface DashboardNodeMetricsData {
  metrics: DashboardMetricPoint[]
}

// 下面这几个 Graph* 类型不是后端直接返回的格式，
// 而是“给 ECharts 图表组件使用的中间格式”。
//
// 为什么要单独定义？
// 因为后端只关心业务数据，
// 但图表还需要 symbolSize、lineStyle、label 这类纯展示字段。
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
  source: string
  target: string
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

// dashboardHttp 专门负责控制台相关接口请求。
// 这里不再写死 mock 数据，而是直接发真实 HTTP 请求。
const dashboardHttp = axios.create({
  // 如果以后单独配置了 VITE_API_BASE_URL，就优先用它。
  // 没配置时保持空字符串，表示继续走当前站点同源地址。
  //
  // 所谓“同源地址”，可以简单理解成：
  // 浏览器当前打开的是哪个地址，就先按这个地址去拼请求。
  // 如果路径是 /api/...，再由 Vite 代理帮我们转出去。
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '',
  timeout: 10000
})

// 请求 stats：
// 页面顶部四张统计卡片会直接读取这个接口。
export const getDashboardStats = async (): Promise<ApiResponse<DashboardStatsData>> => {
  const response = await dashboardHttp.get<ApiResponse<DashboardStatsData>>('/api/v1/dashboard/stats')

  return response.data
}

// 请求 nodes：
// 页面表格和关系图的节点本体都来自这里。
export const getDashboardNodes = async (): Promise<ApiResponse<DashboardNodesData>> => {
  // 真实接口路径直接与 API 文档保持一致：
  // GET /api/v1/dashboard/nodes
  // 页面组件只关心 response.data，不需要知道底层是 axios 还是其他实现。
  const response = await dashboardHttp.get<ApiResponse<DashboardNodesData>>(
    '/api/v1/dashboard/nodes'
  )

  return response.data
}

// 请求 edges：
// 关系图上的“线”不再由前端猜，而是直接使用后端返回的真实边关系。
export const getDashboardEdges = async (): Promise<ApiResponse<DashboardEdgesData>> => {
  const response = await dashboardHttp.get<ApiResponse<DashboardEdgesData>>('/api/v1/dashboard/edges')

  return response.data
}

// 请求 metrics：
// 用于节点悬浮卡片里的折线图。
// nodeId 表示当前节点，targetId 表示要观察它和哪个对端之间的链路。
export const getNodeMetrics = async (
  nodeId: string,
  targetId: string,
  timeRange = '1h'
): Promise<ApiResponse<DashboardNodeMetricsData>> => {
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

// 根据节点状态和 NAT 类型决定节点圆点颜色。
// 这不是业务字段，而是纯展示规则。
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

// 后端边类型是小写 p2p / relay，
// 图上显示时转成更适合人直接阅读的 P2P / Relay。
const getRelationText = (edgeType: DashboardEdge['type']): 'P2P' | 'Relay' => {
  return edgeType === 'relay' ? 'Relay' : 'P2P'
}

// 连线颜色规则也统一放在这里，避免组件里散落判断逻辑。
const getLinkColor = (edgeType: DashboardEdge['type']) => {
  return edgeType === 'relay' ? '#f59e0b' : '#64748b'
}

export const buildDashboardGraphData = (
  nodesSource: DashboardNode[],
  edgesSource: DashboardEdge[]
): DashboardGraphData => {
  // 第一步：先把后端节点数组，转换成 ECharts 的节点数组。
  const nodes: GraphNodeItem[] = nodesSource.map((node) => ({
    id: node.node_id,
    name: node.hostname,

    // connected_peers 越多，节点圆点越大。
    // 这里做了最小值和最大值限制，避免节点太小或太夸张。
    symbolSize: Math.max(58, Math.min(84, 58 + node.connected_peers * 6)),
    value: node.connected_peers,
    hostname: node.hostname,
    virtualIp: node.virtual_ip,
    publicIp: node.public_ip || '未上报',
    natType: node.nat_type,
    status: node.status,
    itemStyle: {
      color: getNodeColor(node)
    },
    label: {
      show: true,
      color: '#0f172a',
      fontWeight: 600
    }
  }))

  // 第二步：把所有节点 id 先收集起来。
  // 这样后面处理边时，可以过滤掉那些“边里引用了不存在节点”的异常情况。
  const nodeIds = new Set(nodesSource.map((node) => node.node_id))

  // 第三步：把后端边数组转换成 ECharts 的 links 数组。
  const links: GraphLinkItem[] = edgesSource
    .filter((edge) => nodeIds.has(edge.source) && nodeIds.has(edge.target))
    .map((edge) => ({
      source: edge.source,
      target: edge.target,
      relationText: getRelationText(edge.type),
      lineStyle: {
        color: getLinkColor(edge.type),
        width: edge.type === 'relay' ? 4 : 3,
        curveness: edge.type === 'relay' ? 0.12 : 0.06
      },
      label: {
        show: true,
        formatter: getRelationText(edge.type),
        color: '#1e293b',
        fontWeight: 700
      }
    }))

  return {
    nodes,
    links
  }
}
