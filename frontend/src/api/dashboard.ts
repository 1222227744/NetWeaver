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

// 这里保留一个最小响应结构，读取方式和 axios 常见的 response.data 一致。
// 当前分支先用它承接 mock 请求，等仓库允许安装依赖后可平滑替换成真实 axios。
interface MockHttpResponse<T> {
  data: T
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

const mockGet = async <T>(url: string, mockData: T): Promise<MockHttpResponse<T>> => {
  // 这里保留一个短暂延迟，用来模拟真实请求返回的过程。
  // 页面上的 loading 状态也会因此真实触发。
  // url 参数保留下来，是为了让调用处继续维持真实接口的写法。
  void url
  await new Promise((resolve) => setTimeout(resolve, 300))

  return {
    data: mockData
  }
}

export const getDashboardNodes = async (): Promise<ApiResponse<DashboardNodesData>> => {
  // 这里保留真实接口的 URL，和 API 文档保持一致：
  // GET /api/v1/dashboard/nodes
  // 当前先用本地 mock 请求返回数据，但页面读取方式依旧保持 response.data 结构。
  const response = await mockGet<ApiResponse<DashboardNodesData>>(
    '/api/v1/dashboard/nodes',
    mockDashboardNodesResponse
  )

  return response.data
}
