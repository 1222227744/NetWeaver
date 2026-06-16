<script setup lang="ts">
import type { DashboardNode } from '@/api/dashboard'

defineProps<{
  empty: boolean
  hasOnlyOfflineNodes: boolean
  loading: boolean
  nodes: DashboardNode[]
  offlineCount: number
  onlineCount: number
}>()

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

const formatLastSeen = (timestamp?: number) => {
  return timestamp ? new Date(timestamp * 1000).toLocaleString('zh-CN', { hour12: false }) : '未上报'
}
</script>

<template>
  <section class="panel-surface h-full p-6">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <p class="panel-heading">节点清单</p>
        <h2 class="mt-2 text-xl font-semibold text-slate-900">节点列表</h2>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <el-tag round type="success">在线 {{ onlineCount }} 台</el-tag>
        <el-tag round type="info">离线 {{ offlineCount }} 台</el-tag>
        <el-tag round>总计 {{ nodes.length }} 台</el-tag>
      </div>
    </div>

    <el-alert
      v-if="hasOnlyOfflineNodes"
      class="mt-6"
      type="warning"
      show-icon
      :closable="false"
      title="当前暂无在线节点。"
    />

    <el-empty
      v-if="empty"
      class="mt-6 rounded-[1.5rem] border border-dashed border-slate-200 bg-slate-50"
      description="暂无节点"
    />

    <div v-else class="mt-6 overflow-x-auto rounded-[1.5rem]">
      <el-table
        v-loading="loading"
        :data="nodes"
        stripe
        border
        class="min-w-[58rem]"
        max-height="620"
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
    </div>
  </section>
</template>
