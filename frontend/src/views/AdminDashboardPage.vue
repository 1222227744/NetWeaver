<script setup lang="ts">
import {
  Bell,
  Connection,
  Document,
  Files,
  Grid,
  Monitor,
  Setting,
  User
} from '@element-plus/icons-vue'
import { ref } from 'vue'

import OnlineNodeTable from '@/components/dashboard/OnlineNodeTable.vue'
import AdminShell from '@/components/layout/AdminShell.vue'
import type { SidebarMenuGroup } from '@/components/layout/layout.types'

// activeMenu 表示当前高亮的菜单。
// ref('dashboard') 的意思是：默认先高亮 dashboard 这个菜单。
const activeMenu = ref('dashboard')

// 这里先用本地常量模拟后台菜单，后续接路由时可把 index 改成 route name/path。
const menuGroups: SidebarMenuGroup[] = [
  {
    title: '工作台',
    items: [
      { index: 'dashboard', label: '概览面板', icon: Grid },
      { index: 'monitor', label: '在线节点', icon: Monitor, hint: 'LIVE' }
    ]
  },
  {
    title: '协作管理',
    items: [
      { index: 'project', label: '项目空间', icon: Files },
      { index: 'team', label: '成员权限', icon: User },
      { index: 'message', label: '消息中心', icon: Bell, hint: '12' }
    ]
  },
  {
    title: '系统设置',
    items: [
      { index: 'api', label: '接口配置', icon: Connection },
      { index: 'document', label: '文档中心', icon: Document },
      { index: 'setting', label: '基础设置', icon: Setting }
    ]
  }
]
</script>

<template>
  <!--
    v-model:active-menu 可以先粗略理解成：
    把 activeMenu 这个变量交给 AdminShell 使用，
    同时允许 AdminShell 在菜单变化时把新值再同步回来。
  -->
  <AdminShell
    v-model:active-menu="activeMenu"
    :menu-groups="menuGroups"
    page-title="在线节点与关系预览"
    page-description="当前页面会读取在线节点接口，并在同一页中展示节点表格、节点关系图，以及节点悬停时的详细信息预览。"
  >
    <template #header-actions>
      <el-tag size="large" round>GET /api/v1/dashboard/nodes</el-tag>
      <el-tag size="large" round type="success">Real API</el-tag>
    </template>

    <OnlineNodeTable />
  </AdminShell>
</template>
