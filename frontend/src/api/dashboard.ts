import axios from 'axios'

// 全局响应结构按照 API 文档统一定义。
// 这样后面无论接真实后端还是继续扩展其他接口，返回值结构都能保持一致。
export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

// 单个节点的数据结构直接对应接口文档里的 nodes 数组元素。
export interface DashboardNode {
  node_id: string
  hostname: string
  virtual_ip: string
  public_ip: string
  nat_type: string
  status: 'online' | 'offline'
  connected_peers: number
}

export interface DashboardNodesData {
  nodes: DashboardNode[]
}

export interface GraphNodeItem {
  id: string
  name: string
  x: number
  y: number
  symbolSize: number
  value: number
  hostname: string
  virtualIp: string
  publicIp: string
  natType: string
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
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '',
  timeout: 10000
})

export const getDashboardNodes = async (): Promise<ApiResponse<DashboardNodesData>> => {
  // 真实接口路径直接与 API 文档保持一致：
  // GET /api/v1/dashboard/nodes
  // 页面组件只关心 response.data，不需要知道底层是 axios 还是其他实现。
  const response = await dashboardHttp.get<ApiResponse<DashboardNodesData>>(
    '/api/v1/dashboard/nodes'
  )

  return response.data
}

const getRelationText = (sourceNode: DashboardNode, targetNode: DashboardNode): 'P2P' | 'Relay' => {
  // 这里先按 NAT 类型做一条简单规则：
  // 只要两端有任意一端是 Symmetric，就先视为更依赖中转链路，标记为 Relay。
  // 其他情况先标记为 P2P。
  const natTypes = [sourceNode.nat_type, targetNode.nat_type]
  return natTypes.includes('Symmetric') ? 'Relay' : 'P2P'
}

export const buildDashboardGraphData = (onlineNodes: DashboardNode[]): DashboardGraphData => {
  // 当前需求是先渲染一个“两节点单连线”的关系图，
  // 所以这里只取前两个在线节点做最小图结构。
  const graphNodesSource = onlineNodes.slice(0, 2)

  const nodes: GraphNodeItem[] = graphNodesSource.map((node, index) => ({
    id: node.node_id,
    name: node.hostname,
    x: index === 0 ? 180 : 520,
    y: 170,
    symbolSize: 74,
    value: node.connected_peers,
    hostname: node.hostname,
    virtualIp: node.virtual_ip,
    publicIp: node.public_ip,
    natType: node.nat_type,
    itemStyle: {
      color: index === 0 ? '#258d7d' : '#0ea5e9'
    },
    label: {
      show: true,
      color: '#0f172a',
      fontWeight: 600
    }
  }))

  const links: GraphLinkItem[] =
    graphNodesSource.length >= 2
      ? [
          {
            source: graphNodesSource[0].node_id,
            target: graphNodesSource[1].node_id,
            relationText: getRelationText(graphNodesSource[0], graphNodesSource[1]),
            lineStyle: {
              color: '#64748b',
              width: 3,
              curveness: 0.06
            },
            label: {
              show: true,
              formatter: getRelationText(graphNodesSource[0], graphNodesSource[1]),
              color: '#1e293b',
              fontWeight: 700
            }
          }
        ]
      : []

  return {
    nodes,
    links
  }
}
