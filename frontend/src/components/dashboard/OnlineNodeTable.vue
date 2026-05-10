<script setup lang="ts">
import { Connection, Monitor, Share } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import axios from 'axios'

import {
  getDashboardEdges,
  getDashboardNodes,
  getDashboardStats,
  getNodeMetrics,
  type DashboardEdge,
  type DashboardMetricPoint,
  type DashboardNode
} from '@/api/dashboard'
import NodeRelationGraph from '@/components/dashboard/NodeRelationGraph.vue'

// 这个组件虽然名字叫 OnlineNodeTable，
// 但它现在其实是整个控制台页面里最核心的“业务容器”。
//
// 它负责 4 件事：
// 1. 拉取 stats 数据，给顶部统计卡片用
// 2. 拉取 nodes 数据，给在线节点表格和关系图用
// 3. 拉取 edges 数据，给关系图的连线用
// 4. 当用户悬停某个节点时，再去请求 metrics，给折线图卡片用
//
// 所以如果你只想抓住“前端业务主线”，优先看这个文件最合适。

// tableData 用 ref 保存表格数组。
// 这里选择 ref，是因为每次请求回来时会整体替换成一份新的数组。
const tableData = ref<DashboardNode[]>([])

// 真实关系边来自后端的 edges 接口。
// 本周开始不再在前端本地猜测连线。
const graphEdges = ref<DashboardEdge[]>([])

// 当前悬停节点对应的链路监控数据。
// 悬浮卡片只负责展示这里传下去的真实 metrics 结果或空态。
const selectedNodeMetrics = ref<DashboardMetricPoint[]>([])
const hoveredGraphNode = ref<DashboardNode | null>(null)
const pollingTimer = ref<number | null>(null)

// state 用 reactive 保存一组彼此相关的页面状态。
// 这里包含 loading、错误提示、统计数字和最近一次刷新时间。
// 之所以用 reactive，是因为这些状态是一组“彼此相关的页面状态”，
// 放在一个对象里更容易统一管理。
const state = reactive({
  loading: false,
  metricsLoading: false,
  errorMessage: '',
  totalNodes: 0,
  onlineNodes: 0,
  totalTrafficGb: 0,
  controllerUptimeSec: 0,
  lastUpdatedAt: ''
})

const getStatusTagType = (status: DashboardNode['status']) => {
  return status === 'online' ? 'success' : 'info'
}

const resetOverviewState = () => {
  state.totalNodes = 0
  state.onlineNodes = 0
  state.totalTrafficGb = 0
  state.controllerUptimeSec = 0
  state.lastUpdatedAt = ''
}

const updateOverviewState = (
  totalNodes: number,
  onlineNodes: number,
  totalTrafficGb: number,
  controllerUptimeSec: number
) => {
  // 这里只是把后端返回的 stats 字段，一一同步到页面状态里。
  state.totalNodes = totalNodes
  state.onlineNodes = onlineNodes
  state.totalTrafficGb = totalTrafficGb
  state.controllerUptimeSec = controllerUptimeSec
  state.lastUpdatedAt = new Date().toLocaleString('zh-CN', { hour12: false })
}

const formatUptime = (uptimeSeconds: number) => {
  // 后端返回的是“总秒数”，人直接看不友好。
  // 所以这里把它格式化成“X 天 X 小时 X 分钟”。
  if (!uptimeSeconds) {
    return '0 秒'
  }

  const days = Math.floor(uptimeSeconds / 86400)
  const hours = Math.floor((uptimeSeconds % 86400) / 3600)
  const minutes = Math.floor((uptimeSeconds % 3600) / 60)
  const parts: string[] = []

  if (days > 0) {
    parts.push(`${days} 天`)
  }

  if (hours > 0) {
    parts.push(`${hours} 小时`)
  }

  if (minutes > 0) {
    parts.push(`${minutes} 分钟`)
  }

  return parts.length > 0 ? parts.join(' ') : `${uptimeSeconds} 秒`
}

const getRequestErrorMessage = (error: unknown) => {
  // axios 请求失败时，优先尝试读取后端返回的 msg 字段。
  if (axios.isAxiosError(error)) {
    const responseMessage = (error.response?.data as { msg?: string } | undefined)?.msg

    if (responseMessage) {
      return responseMessage
    }

    if (error.code === 'ECONNABORTED') {
      return '请求超时，后端接口在 10 秒内没有返回数据。'
    }

    if (error.response) {
      return `接口请求失败，HTTP 状态码：${error.response.status}`
    }

    if (error.request) {
      return '请求已发出，但当前没有收到后端响应。请确认后端服务和接口路由已经启动。'
    }
  }

  if (error instanceof Error) {
    return error.message
  }

  return '控制台接口请求失败，请检查后端服务状态。'
}

const loadNodeMetrics = async (node: DashboardNode) => {
  // API 文档中的 metrics 需要 node_id 与 target_id。
  // 当前后端还没有完全给出节点间 target 选择逻辑，所以这里先用自身 node_id 接通真实请求链路。
  // 也就是说：现在这一步的重点是先把“真实请求流程”跑通，
  // 等后端和前端后续再一起补“选择具体对端节点”的完整交互。
  state.metricsLoading = true

  try {
    const result = await getNodeMetrics(node.node_id, node.node_id)

    if (result.code !== 200) {
      throw new Error(result.msg || '链路监控接口返回失败。')
    }

    selectedNodeMetrics.value = result.data.metrics
  } catch (error) {
    selectedNodeMetrics.value = []

    const message = getRequestErrorMessage(error)
    ElMessage.error(`链路监控加载失败：${message}`)
  } finally {
    state.metricsLoading = false
  }
}

const handleGraphNodeHover = async (node: DashboardNode | null) => {
  // 这个函数是父子组件联动的关键：
  // 1. 子组件 NodeRelationGraph 发现鼠标悬停到了某个节点
  // 2. 它通过自定义事件，把这个节点对象抛给这里
  // 3. 这里再决定要不要去请求 metrics
  hoveredGraphNode.value = node

  if (!node) {
    selectedNodeMetrics.value = []
    return
  }

  await loadNodeMetrics(node)
}

const loadDashboard = async () => {
  state.loading = true
  state.errorMessage = ''

  try {
    // 本周开始把统计卡片、节点列表、关系图分别切到各自的真实接口。
    const [statsResult, nodesResult, edgesResult] = await Promise.all([
      getDashboardStats(),
      getDashboardNodes(),
      getDashboardEdges()
    ])

    if (statsResult.code !== 200) {
      throw new Error(statsResult.msg || '统计接口返回失败。')
    }

    if (nodesResult.code !== 200) {
      throw new Error(nodesResult.msg || '节点列表接口返回失败。')
    }

    if (edgesResult.code !== 200) {
      throw new Error(edgesResult.msg || '节点关系接口返回失败。')
    }

    // API 文档返回的是全部节点，所以这里再过滤出 status === online 的节点。
    const allNodes = nodesResult.data.nodes
    const onlineNodes = allNodes.filter((node) => node.status === 'online')

    tableData.value = onlineNodes
    graphEdges.value = edgesResult.data.edges

    updateOverviewState(
      statsResult.data.total_nodes,
      statsResult.data.online_nodes,
      statsResult.data.total_traffic_gb,
      statsResult.data.controller_uptime_sec
    )
  } catch (error) {
    // 改成真实接口后，请求失败就不能继续拿旧 mock 数据兜底。
    // 这里直接清空页面数据，并把失败原因展示出来。
    tableData.value = []
    graphEdges.value = []
    selectedNodeMetrics.value = []
    hoveredGraphNode.value = null
    resetOverviewState()
    state.errorMessage = getRequestErrorMessage(error)

    ElMessage.error(state.errorMessage)
  } finally {
    state.loading = false
  }
}

const startPolling = () => {
  if (pollingTimer.value !== null) {
    window.clearInterval(pollingTimer.value)
  }

  // 控制台每 5 秒轮询一次，确保节点状态和统计卡片会持续刷新。
  pollingTimer.value = window.setInterval(() => {
    void loadDashboard()
  }, 5000)
}

onMounted(() => {
  // 组件第一次挂载到页面上时，立即拉取一次控制台数据。
  // 然后再开启轮询，让页面后续自动刷新。
  void loadDashboard()
  startPolling()
})

onBeforeUnmount(() => {
  if (pollingTimer.value !== null) {
    window.clearInterval(pollingTimer.value)
  }
})
</script>

<template>
  <div class="space-y-6">
    <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <article class="panel-surface p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="panel-heading">总节点数</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.totalNodes }}</p>
            <p class="mt-2 text-sm text-slate-500">当前值来自 <code>GET /api/v1/dashboard/stats</code></p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-50 text-brand-700">
            <el-icon class="text-xl"><Share /></el-icon>
          </div>
        </div>
      </article>

      <article class="panel-surface p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="panel-heading">在线节点</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.onlineNodes }}</p>
            <p class="mt-2 text-sm text-slate-500">后端统计接口直接返回的在线节点数量</p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
            <el-icon class="text-xl"><Monitor /></el-icon>
          </div>
        </div>
      </article>

      <article class="panel-surface p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="panel-heading">总流量</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.totalTrafficGb.toFixed(1) }}</p>
            <p class="mt-2 text-sm text-slate-500">单位：GB，来源于控制平面统计接口</p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-50 text-cyan-700">
            <el-icon class="text-xl"><Connection /></el-icon>
          </div>
        </div>
      </article>

      <article class="panel-surface p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="panel-heading">控制器运行时长</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900">{{ formatUptime(state.controllerUptimeSec) }}</p>
            <p class="mt-2 text-sm text-slate-500">由后端上报，不再由前端自行推算</p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-700">
            <el-icon class="text-xl"><Share /></el-icon>
          </div>
        </div>
      </article>
    </section>

    <!--
      关系图和表格共用同一份在线节点数据。
      本周开始关系图的连线改为读取后端真实 edges 接口，不再在前端本地推导。
      @node-hover 是子组件抛出来的自定义事件。
      子组件只负责“告诉父组件当前悬停了哪个节点”，
      真正要不要请求 metrics，由父组件自己决定。
    -->
    <NodeRelationGraph
      :online-nodes="tableData"
      :edges="graphEdges"
      :metrics="selectedNodeMetrics"
      @node-hover="handleGraphNodeHover"
    />

    <section class="panel-surface p-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p class="panel-heading">Node Table</p>
          <h2 class="mt-3 text-xl font-semibold text-slate-900">在线节点列表</h2>
          <p class="mt-2 text-sm leading-6 text-slate-500">
            当前页面分别通过 <code>stats</code>、<code>nodes</code>、<code>edges</code> 三个真实接口驱动统计卡片、节点表格和拓扑图。
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <el-tag round type="success">最近刷新：{{ state.lastUpdatedAt || '尚未加载' }}</el-tag>
          <el-tag round :type="state.metricsLoading ? 'warning' : 'info'">
            {{ state.metricsLoading ? '链路图加载中' : '链路图待命中' }}
          </el-tag>
          <el-button :loading="state.loading" type="primary" @click="loadDashboard">重新加载</el-button>
        </div>
      </div>

      <el-alert
        v-if="state.errorMessage"
        :title="state.errorMessage"
        type="error"
        show-icon
        :closable="false"
        class="mt-6"
      />

      <!--
        el-table 直接绑定 ref 保存的 tableData。
        当 loadDashboard 更新 tableData.value 后，Vue 会自动把新数据同步到表格。
        所以 Vue 的一个核心体验就是：
        “你主要改数据，界面会自己跟着刷新”。
      -->
      <el-table
        v-loading="state.loading"
        :data="tableData"
        stripe
        border
        class="mt-6"
        empty-text="当前没有在线节点"
      >
        <el-table-column prop="node_id" label="节点 ID" min-width="180" />
        <el-table-column prop="hostname" label="主机名" min-width="160" />
        <el-table-column prop="virtual_ip" label="虚拟 IP" min-width="140" />
        <el-table-column prop="public_ip" label="公网 IP" min-width="150" />
        <el-table-column prop="nat_type" label="NAT 类型" min-width="170" />
        <el-table-column prop="connected_peers" label="已连接节点数" min-width="120" align="center" />

        <el-table-column label="状态" min-width="100" align="center">
          <template #default="{ row }">
            <!-- 这里的 row 就是当前这一行节点数据 -->
            <el-tag round :type="getStatusTagType(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>
