<script setup lang="ts">
import { Connection, Lock, Monitor, Share } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

import {
  buildDashboardGraphData,
  clearDashboardAuth,
  getDashboardEdges,
  getDashboardNodes,
  getDashboardStats,
  getNodeMetrics,
  getStoredDashboardAuth,
  hasValidDashboardAuth,
  loginDashboard,
  setDashboardAuth,
  type DashboardEdge,
  type DashboardMetricPoint,
  type DashboardMetricsTimeRange,
  type DashboardNode,
  type GraphLinkItem
} from '@/api/dashboard'
import NodeRelationGraph from '@/components/dashboard/NodeRelationGraph.vue'

// 这个组件是 Dashboard 页面当前最核心的“业务容器”。
// 它负责把接口、状态、关系图、表格串起来。
//
// 推荐按这个顺序理解：
// 1. 先看 authState 和 loginForm：决定有没有登录、能不能请求 dashboard/* 接口
// 2. 再看 loadDashboard：并发请求 stats / nodes / edges，更新页面主体数据
// 3. 再看 handleGraphLinkHover：用户悬停连线后，才查询 source -> target 的 metrics
// 4. 最后看 template：把上面这些响应式数据渲染成 Element Plus 页面

const initialAuth = getStoredDashboardAuth()

const loginForm = reactive({
  username: 'admin',
  password: ''
})

const authState = reactive({
  isAuthenticated: hasValidDashboardAuth(),
  loading: false,
  errorMessage: '',
  tokenExpiresAt: initialAuth?.expires_at ?? 0
})

// allNodes 保存后端返回的全部节点，包含 online 和 offline。
// API v1.8 的 nodes 接口语义就是“所有节点”，
// 所以关系图和表格都应该基于 allNodes 渲染，不能只保留在线节点。
//
// onlineNodes 仍然保留，是因为统计标签、空态判断和部分文案需要单独知道在线数量。
const allNodes = ref<DashboardNode[]>([])
const graphEdges = ref<DashboardEdge[]>([])
const selectedLinkMetrics = ref<DashboardMetricPoint[]>([])
const selectedGraphLink = ref<GraphLinkItem | null>(null)
const pollingTimer = ref<number | null>(null)
const selectedTimeRange = ref<DashboardMetricsTimeRange>('1h')
const linkMetricsCache = new Map<string, DashboardMetricPoint[]>()
const linkMetricsRequests = new Map<string, Promise<DashboardMetricPoint[]>>()

// metricsRequestSerial 用来避免“旧请求后返回，把新请求结果覆盖掉”。
// 每次发起新的 metrics 请求时加 1，只允许最新编号的请求更新页面。
let metricsRequestSerial = 0

const state = reactive({
  loading: false,
  refreshing: false,
  metricsLoading: false,
  errorMessage: '',
  metricsErrorMessage: '',
  totalNodes: 0,
  onlineNodes: 0,
  totalTrafficGb: 0,
  controllerUptimeSec: 0,
  offlineNodes: 0,
  totalEdges: 0,
  lastUpdatedAt: ''
})

const timeRangeOptions: Array<{ label: string; value: DashboardMetricsTimeRange }> = [
  { label: '近 1 小时', value: '1h' },
  { label: '近 12 小时', value: '12h' },
  { label: '近 24 小时', value: '24h' }
]

const onlineNodes = computed(() => allNodes.value.filter((node) => node.status === 'online'))
const offlineNodes = computed(() => allNodes.value.filter((node) => node.status === 'offline'))

const isNodeListEmptyState = computed(() => {
  return !state.loading && !state.errorMessage && allNodes.value.length === 0
})

const hasOnlyOfflineNodes = computed(() => {
  return !state.loading && !state.errorMessage && allNodes.value.length > 0 && onlineNodes.value.length === 0
})

const getStatusTagType = (status: DashboardNode['status']) => {
  return status === 'online' ? 'success' : 'info'
}

const getStatusText = (status: DashboardNode['status']) => {
  return status === 'online' ? '在线' : '离线'
}

const formatOptionalText = (value: string | number | null | undefined) => {
  if (value === null || value === undefined || value === '') {
    return '未上报'
  }

  return String(value)
}

const formatTimestamp = (timestamp?: number) => {
  if (!timestamp) {
    return ''
  }

  return new Date(timestamp * 1000).toLocaleString('zh-CN', { hour12: false })
}

const formatUptime = (uptimeSeconds: number) => {
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

const formatLastSeen = (timestamp?: number) => {
  return timestamp ? new Date(timestamp * 1000).toLocaleString('zh-CN', { hour12: false }) : '未上报'
}

const getRequestErrorMessage = (error: unknown) => {
  if (axios.isAxiosError(error)) {
    const responseMessage = (error.response?.data as { msg?: string } | undefined)?.msg

    if (responseMessage) {
      return responseMessage
    }

    if (error.code === 'ECONNABORTED') {
      return '请求超时，服务在 10 秒内没有返回数据。'
    }

    if (error.response) {
      return `数据加载失败，状态码：${error.response.status}`
    }

    if (error.request) {
      return '暂时无法连接到服务，请确认服务已启动。'
    }
  }

  if (error instanceof Error) {
    return error.message
  }

  return '控制台数据加载失败，请检查服务状态。'
}

const isUnauthorizedError = (error: unknown) => {
  return axios.isAxiosError(error) && error.response?.status === 401
}

const resetOverviewState = () => {
  state.totalNodes = 0
  state.onlineNodes = 0
  state.totalTrafficGb = 0
  state.controllerUptimeSec = 0
  state.offlineNodes = 0
  state.totalEdges = 0
  state.lastUpdatedAt = ''
}

const clearDashboardData = () => {
  allNodes.value = []
  graphEdges.value = []
  selectedLinkMetrics.value = []
  selectedGraphLink.value = null
  linkMetricsCache.clear()
  linkMetricsRequests.clear()
  state.metricsErrorMessage = ''
  resetOverviewState()
}

const stopPolling = () => {
  if (pollingTimer.value !== null) {
    window.clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
}

const handleUnauthorized = (message = '登录已过期，请重新登录。') => {
  stopPolling()
  clearDashboardAuth()
  clearDashboardData()
  authState.isAuthenticated = false
  authState.tokenExpiresAt = 0
  authState.errorMessage = message
  state.errorMessage = ''
}

const updateOverviewState = (statsData: NonNullable<Awaited<ReturnType<typeof getDashboardStats>>['data']>) => {
  state.totalNodes = statsData.total_nodes
  state.onlineNodes = statsData.online_nodes
  state.totalTrafficGb = statsData.total_traffic_gb
  state.controllerUptimeSec = statsData.controller_uptime_sec
  state.offlineNodes = statsData.offline_nodes ?? Math.max(statsData.total_nodes - statsData.online_nodes, 0)
  state.totalEdges = statsData.total_edges ?? graphEdges.value.length
  state.lastUpdatedAt = formatTimestamp(statsData.last_updated_at) || new Date().toLocaleString('zh-CN', { hour12: false })
}

const findCurrentGraphLink = (link: GraphLinkItem, edges: DashboardEdge[]) => {
  return (
    buildDashboardGraphData(allNodes.value, edges).links.find(
      (edge) => edge.source === link.source && edge.target === link.target
    ) ?? null
  )
}

const getLinkMetricsCacheKey = (link: GraphLinkItem) => {
  return `${link.id}::${selectedTimeRange.value}`
}

const loadLinkMetrics = async (
  link: GraphLinkItem,
  options: { force?: boolean; silent?: boolean; backgroundRefresh?: boolean } = {}
) => {
  const currentSerial = ++metricsRequestSerial
  const cacheKey = getLinkMetricsCacheKey(link)
  const cachedMetrics = linkMetricsCache.get(cacheKey)
  const shouldKeepVisibleMetrics = options.backgroundRefresh === true && Array.isArray(cachedMetrics)

  // 鼠标移出再移回同一条链路时，优先显示缓存数据。
  // 这样不会每次悬停都重新出现“链路数据加载中”，也不会重复请求同一时间范围的数据。
  if (!options.force && cachedMetrics) {
    selectedLinkMetrics.value = cachedMetrics
    state.metricsLoading = false
    state.metricsErrorMessage = ''
    return
  }

  state.metricsErrorMessage = ''

  // 静默轮询刷新当前链路时，如果页面上已经有旧曲线，就继续显示它，
  // 同时在后台发起一次真实请求把缓存替换成最新数据。
  // 这样可以兼顾“数据实时更新”和“不要每 5 秒闪一次加载中”。
  if (shouldKeepVisibleMetrics && cachedMetrics) {
    selectedLinkMetrics.value = cachedMetrics
    state.metricsLoading = false
  } else {
    state.metricsLoading = true
  }

  try {
    const existingRequest = linkMetricsRequests.get(cacheKey)
    const metricsRequest =
      existingRequest ??
      getNodeMetrics(link.source, link.target, selectedTimeRange.value).then((result) => {
        if (result.code !== 200 || !result.data) {
          throw new Error(result.msg || '链路监控数据加载失败。')
        }

        return result.data.metrics
      })

    if (!existingRequest) {
      linkMetricsRequests.set(cacheKey, metricsRequest)
    }

    const metrics = await metricsRequest
    linkMetricsCache.set(cacheKey, metrics)
    linkMetricsRequests.delete(cacheKey)

    if (currentSerial !== metricsRequestSerial) {
      return
    }

    selectedLinkMetrics.value = metrics
  } catch (error) {
    linkMetricsRequests.delete(cacheKey)

    if (currentSerial !== metricsRequestSerial) {
      return
    }

    if (!shouldKeepVisibleMetrics) {
      selectedLinkMetrics.value = []
    }

    if (isUnauthorizedError(error)) {
      handleUnauthorized('登录状态无效或已过期，请重新登录后再查看链路数据。')
      return
    }

    const message = getRequestErrorMessage(error)
    state.metricsErrorMessage = message

    if (!options.silent) {
      ElMessage.error(`链路监控加载失败：${message}`)
    }
  } finally {
    if (currentSerial === metricsRequestSerial && !shouldKeepVisibleMetrics) {
      state.metricsLoading = false
    }
  }
}

const syncSelectedLinkAfterEdgesRefresh = (latestEdges: DashboardEdge[]) => {
  if (!selectedGraphLink.value) {
    return
  }

  const currentEdge = findCurrentGraphLink(selectedGraphLink.value, latestEdges)

  if (!currentEdge) {
    selectedGraphLink.value = null
    selectedLinkMetrics.value = []
    state.metricsErrorMessage = '当前选中的链路已经不存在，可能是节点离线或建链状态变化。'
    return
  }

  selectedGraphLink.value = currentEdge

  void loadLinkMetrics(currentEdge, { force: true, silent: true, backgroundRefresh: true })
}

const loadDashboard = async (options: { silent?: boolean } = {}) => {
  if (!authState.isAuthenticated) {
    return
  }

  if (options.silent) {
    state.refreshing = true
  } else {
    state.loading = true
  }

  state.errorMessage = ''

  try {
    const [statsResult, nodesResult, edgesResult] = await Promise.all([
      getDashboardStats(),
      getDashboardNodes(),
      getDashboardEdges()
    ])

    if (statsResult.code !== 200 || !statsResult.data) {
      throw new Error(statsResult.msg || '统计数据加载失败。')
    }

    if (nodesResult.code !== 200 || !nodesResult.data) {
      throw new Error(nodesResult.msg || '节点列表加载失败。')
    }

    if (edgesResult.code !== 200 || !edgesResult.data) {
      throw new Error(edgesResult.msg || '节点关系加载失败。')
    }

    allNodes.value = nodesResult.data.nodes
    graphEdges.value = edgesResult.data.edges
    updateOverviewState(statsResult.data)
    syncSelectedLinkAfterEdgesRefresh(edgesResult.data.edges)
  } catch (error) {
    if (isUnauthorizedError(error)) {
      handleUnauthorized('登录状态无效或已过期，请重新登录后继续查看控制台。')
      return
    }

    clearDashboardData()
    state.errorMessage = getRequestErrorMessage(error)

    if (!options.silent) {
      ElMessage.error(state.errorMessage)
    }
  } finally {
    state.loading = false
    state.refreshing = false
  }
}

const startPolling = () => {
  stopPolling()

  // 每 5 秒刷新 stats / nodes / edges。
  // 这样节点上下线、边新增/消失和统计卡片变化能自动反映到页面上。
  pollingTimer.value = window.setInterval(() => {
    void loadDashboard({ silent: true })
  }, 5000)
}

const handleLogin = async () => {
  if (!loginForm.username.trim() || !loginForm.password) {
    authState.errorMessage = '请输入控制台管理员用户名和密码。'
    return
  }

  authState.loading = true
  authState.errorMessage = ''

  try {
    const result = await loginDashboard({
      username: loginForm.username.trim(),
      password: loginForm.password
    })

    if (result.code !== 200 || !result.data) {
      throw new Error(result.msg || '登录失败。')
    }

    setDashboardAuth(result.data)
    authState.isAuthenticated = true
    authState.tokenExpiresAt = result.data.expires_at
    loginForm.password = ''
    ElMessage.success('登录成功，正在加载控制台数据。')

    await loadDashboard()
    startPolling()
  } catch (error) {
    const message = getRequestErrorMessage(error)
    authState.errorMessage = message
    ElMessage.error(`登录失败：${message}`)
  } finally {
    authState.loading = false
  }
}

const handleLogout = () => {
  stopPolling()
  clearDashboardAuth()
  clearDashboardData()
  authState.isAuthenticated = false
  authState.tokenExpiresAt = 0
  authState.errorMessage = '已退出登录。'
  ElMessage.success('已退出控制台。')
}

const handleGraphLinkHover = (link: GraphLinkItem | null) => {
  const previousLinkId = selectedGraphLink.value?.id ?? ''
  selectedGraphLink.value = link
  state.metricsErrorMessage = ''

  if (!link) {
    metricsRequestSerial += 1
    selectedLinkMetrics.value = []
    state.metricsLoading = false
    return
  }

  if (previousLinkId === link.id && state.metricsLoading) {
    return
  }

  void loadLinkMetrics(link)
}

watch(selectedTimeRange, () => {
  if (selectedGraphLink.value) {
    void loadLinkMetrics(selectedGraphLink.value)
  }
})

onMounted(() => {
  if (!authState.isAuthenticated) {
    authState.errorMessage = '请先登录控制台。'
    return
  }

  void loadDashboard()
  startPolling()
})

onBeforeUnmount(() => {
  stopPolling()
})
</script>

<template>
  <div class="space-y-6">
    <section v-if="!authState.isAuthenticated" class="panel-surface mx-auto max-w-xl p-8">
      <div class="flex items-start gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-brand-50 text-brand-700">
          <el-icon class="text-xl"><Lock /></el-icon>
        </div>
        <div>
          <h2 class="text-2xl font-semibold text-slate-900">登录控制台</h2>
        </div>
      </div>

      <form class="mt-8 space-y-5" @submit.prevent="handleLogin">
        <label class="block">
          <span class="text-sm font-medium text-slate-700">用户名</span>
          <el-input v-model="loginForm.username" class="mt-2" placeholder="例如：admin" size="large" />
        </label>

        <label class="block">
          <span class="text-sm font-medium text-slate-700">密码</span>
          <el-input
            v-model="loginForm.password"
            class="mt-2"
            placeholder="请输入控制台密码"
            show-password
            size="large"
            type="password"
          />
        </label>

        <el-alert
          v-if="authState.errorMessage"
          :title="authState.errorMessage"
          type="warning"
          show-icon
          :closable="false"
        />

        <el-button :loading="authState.loading" native-type="submit" size="large" type="primary" class="w-full">
          登录并进入控制台
        </el-button>
      </form>
    </section>

    <template v-else>
      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <article class="panel-surface p-5">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="panel-heading">总节点数</p>
              <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.totalNodes }}</p>
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
              <p class="mt-4 text-3xl font-semibold text-slate-900">
                {{ state.onlineNodes }}
              </p>
            </div>
            <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
              <el-icon class="text-xl"><Monitor /></el-icon>
            </div>
          </div>
        </article>

        <article class="panel-surface p-5">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="panel-heading">链路数</p>
              <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.totalEdges }}</p>
            </div>
            <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-50 text-cyan-700">
              <el-icon class="text-xl"><Connection /></el-icon>
            </div>
          </div>
        </article>

        <article class="panel-surface p-5">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="panel-heading">运行时长</p>
              <p class="mt-4 text-3xl font-semibold text-slate-900">{{ formatUptime(state.controllerUptimeSec) }}</p>
            </div>
            <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-700">
              <el-icon class="text-xl"><Share /></el-icon>
            </div>
          </div>
        </article>
      </section>

      <section class="panel-surface p-4">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <el-radio-group v-model="selectedTimeRange">
            <el-radio-button
              v-for="option in timeRangeOptions"
              :key="option.value"
              :label="option.value"
            >
              {{ option.label }}
            </el-radio-button>
          </el-radio-group>

          <div class="flex flex-wrap items-center gap-3">
            <el-tag round :type="state.refreshing ? 'warning' : 'info'">
              {{ state.refreshing ? '刷新中' : state.lastUpdatedAt || '尚未加载' }}
            </el-tag>
            <el-button :loading="state.loading || state.refreshing" type="primary" @click="loadDashboard()">
              刷新
            </el-button>
            <el-button @click="handleLogout">退出</el-button>
          </div>
        </div>

        <el-alert
          v-if="state.errorMessage"
          :title="state.errorMessage"
          type="error"
          show-icon
          :closable="false"
          class="mt-5"
        />

        <el-alert
          v-if="state.metricsErrorMessage"
          :title="state.metricsErrorMessage"
          type="warning"
          show-icon
          :closable="false"
          class="mt-5"
        />
      </section>

      <NodeRelationGraph
        :edges="graphEdges"
        :metrics="selectedLinkMetrics"
        :metrics-loading="state.metricsLoading"
        :nodes="allNodes"
        :selected-time-range="selectedTimeRange"
        @link-hover="handleGraphLinkHover"
      />

      <section class="panel-surface p-6">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 class="text-xl font-semibold text-slate-900">节点列表</h2>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <el-tag round type="success">在线 {{ onlineNodes.length }} 台</el-tag>
            <el-tag round type="info">离线 {{ offlineNodes.length }} 台</el-tag>
            <el-tag round>总计 {{ allNodes.length }} 台</el-tag>
          </div>
        </div>

        <el-alert
          v-if="hasOnlyOfflineNodes"
          class="mt-6"
          type="warning"
          show-icon
          :closable="false"
          title="当前没有在线节点。列表保留离线节点，便于核对 status、last_seen 和虚拟 IP。"
        />

        <el-empty
          v-if="isNodeListEmptyState"
          class="mt-6 rounded-[1.5rem] border border-dashed border-slate-200 bg-slate-50"
          description="当前还没有任何节点"
        />

        <el-table
          v-loading="state.loading"
          :data="allNodes"
          stripe
          border
          class="mt-6"
          empty-text="当前没有节点"
        >
          <el-table-column prop="node_id" label="节点 ID" min-width="180" />
          <el-table-column prop="hostname" label="主机名" min-width="150" />
          <el-table-column prop="virtual_ip" label="虚拟 IP" min-width="130" />
          <el-table-column label="公网地址" min-width="170">
            <template #default="{ row }">
              {{ formatOptionalText(row.public_ip) }}:{{ formatOptionalText(row.public_port) }}
            </template>
          </el-table-column>
          <el-table-column prop="nat_type" label="NAT 类型" min-width="170" />
          <el-table-column prop="connected_peers" label="真实邻居数" min-width="110" align="center" />
          <el-table-column label="最后心跳" min-width="180">
            <template #default="{ row }">
              {{ formatLastSeen(row.last_seen) }}
            </template>
          </el-table-column>

          <el-table-column label="状态" min-width="100" align="center">
            <template #default="{ row }">
              <el-tag round :type="getStatusTagType(row.status)">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </section>
    </template>
  </div>
</template>
