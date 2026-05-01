import axios from 'axios'
import type { AxiosAdapter, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

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

// 当前分支先没有真实后端，所以把接口文档里的数据格式直接写成 mock JSON。
// 这里故意保留了一条 offline 数据，页面里会再过滤出“在线节点列表”。
const mockDashboardNodesResponse: ApiResponse<DashboardNodesData> = {
  code: 200,
  msg: 'success',
  data: {
    nodes: [
      {
        node_id: 'nw-node-a1b2',
        hostname: 'ubuntu-server-01',
        virtual_ip: '10.0.0.2',
        public_ip: '203.0.113.5',
        nat_type: 'Full Cone',
        status: 'online',
        connected_peers: 3
      },
      {
        node_id: 'nw-node-c3d4',
        hostname: 'macbook-pro',
        virtual_ip: '10.0.0.3',
        public_ip: '198.51.100.12',
        nat_type: 'Symmetric',
        status: 'online',
        connected_peers: 3
      },
      {
        node_id: 'nw-node-e5f6',
        hostname: 'windows-lab',
        virtual_ip: '10.0.0.4',
        public_ip: '192.0.2.88',
        nat_type: 'Port Restricted Cone',
        status: 'offline',
        connected_peers: 0
      },
      {
        node_id: 'nw-node-g7h8',
        hostname: 'edge-gateway-01',
        virtual_ip: '10.0.0.5',
        public_ip: '203.0.113.19',
        nat_type: 'Full Cone',
        status: 'online',
        connected_peers: 5
      }
    ]
  }
}

const dashboardNodesMockAdapter: AxiosAdapter = async (
  config: InternalAxiosRequestConfig
): Promise<AxiosResponse<ApiResponse<DashboardNodesData>>> => {
  // 当前还没有接真实后端，所以依旧通过 mock 数据返回。
  // 但请求层已经切回 axios 标准写法，后续替换真实接口时不需要动页面组件。
  await new Promise((resolve) => setTimeout(resolve, 300))

  return {
    data: mockDashboardNodesResponse,
    status: 200,
    statusText: 'OK',
    headers: {},
    config
  }
}

export const getDashboardNodes = async (): Promise<ApiResponse<DashboardNodesData>> => {
  // 这里保留真实接口的 URL，和 API 文档保持一致：
  // GET /api/v1/dashboard/nodes
  // 当前虽然返回的是 mock 数据，但调用方式已经恢复为 axios.get(...).data。
  const response = await axios.get<ApiResponse<DashboardNodesData>>('/api/v1/dashboard/nodes', {
    adapter: dashboardNodesMockAdapter
  })

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
