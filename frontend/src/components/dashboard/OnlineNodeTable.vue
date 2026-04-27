<script setup lang="ts">
import { Connection, Monitor, Share } from '@element-plus/icons-vue'
import { onMounted, reactive, ref } from 'vue'

import { getDashboardNodes, type DashboardNode } from '@/api/dashboard'

// tableData 用 ref 保存表格数组。
// 这里选择 ref，是因为每次请求回来时会整体替换成一份新的数组。
const tableData = ref<DashboardNode[]>([])

// state 用 reactive 保存一组彼此相关的页面状态。
// 这里包含 loading、统计数字和最近一次刷新时间。
const state = reactive({
  loading: false,
  totalNodes: 0,
  onlineNodes: 0,
  totalConnectedPeers: 0,
  fullConeNodes: 0,
  lastUpdatedAt: ''
})

const getStatusTagType = (status: DashboardNode['status']) => {
  return status === 'online' ? 'success' : 'info'
}

const updateOverviewState = (allNodes: DashboardNode[], onlineNodes: DashboardNode[]) => {
  state.totalNodes = allNodes.length
  state.onlineNodes = onlineNodes.length
  state.totalConnectedPeers = onlineNodes.reduce((total, node) => total + node.connected_peers, 0)
  state.fullConeNodes = onlineNodes.filter((node) => node.nat_type === 'Full Cone').length
  state.lastUpdatedAt = new Date().toLocaleString('zh-CN', { hour12: false })
}

const loadOnlineNodes = async () => {
  state.loading = true

  try {
    // 这里调用的是 API 请求层方法。
    // 当前返回的是 mock JSON，但页面层依旧通过 response.data 结构读取结果。
    const result = await getDashboardNodes()

    // API 文档返回的是全部节点，所以这里再过滤出 status === online 的节点。
    const allNodes = result.data.nodes
    const onlineNodes = allNodes.filter((node) => node.status === 'online')

    tableData.value = onlineNodes
    updateOverviewState(allNodes, onlineNodes)
  } finally {
    state.loading = false
  }
}

onMounted(() => {
  // 组件第一次挂载到页面上时，立即拉取一次在线节点数据。
  void loadOnlineNodes()
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
            <p class="mt-2 text-sm text-slate-500">来自节点拓扑接口的完整返回结果</p>
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
            <p class="mt-2 text-sm text-slate-500">页面只渲染 status 为 online 的节点</p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
            <el-icon class="text-xl"><Monitor /></el-icon>
          </div>
        </div>
      </article>

      <article class="panel-surface p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="panel-heading">已连接对等节点</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.totalConnectedPeers }}</p>
            <p class="mt-2 text-sm text-slate-500">在线节点 connected_peers 的求和结果</p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-50 text-cyan-700">
            <el-icon class="text-xl"><Connection /></el-icon>
          </div>
        </div>
      </article>

      <article class="panel-surface p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="panel-heading">Full Cone 节点</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900">{{ state.fullConeNodes }}</p>
            <p class="mt-2 text-sm text-slate-500">在线节点里 NAT 类型为 Full Cone 的数量</p>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-700">
            <el-icon class="text-xl"><Share /></el-icon>
          </div>
        </div>
      </article>
    </section>

    <section class="panel-surface p-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p class="panel-heading">Node Table</p>
          <h2 class="mt-3 text-xl font-semibold text-slate-900">在线节点列表</h2>
          <p class="mt-2 text-sm leading-6 text-slate-500">
            当前数据来自 <code>GET /api/v1/dashboard/nodes</code> 的本地 mock JSON，并通过独立请求层读入页面。
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <el-tag round type="success">最近刷新：{{ state.lastUpdatedAt || '尚未加载' }}</el-tag>
          <el-button :loading="state.loading" type="primary" @click="loadOnlineNodes">重新加载</el-button>
        </div>
      </div>

      <!--
        el-table 直接绑定 ref 保存的 tableData。
        当 loadOnlineNodes 更新 tableData.value 后，Vue 会自动把新数据同步到表格。
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
            <el-tag round :type="getStatusTagType(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>
