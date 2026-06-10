<script setup lang="ts">
import { Connection, Grid, Monitor } from '@element-plus/icons-vue'
import { ref } from 'vue'

import OnlineNodeTable from '@/components/dashboard/OnlineNodeTable.vue'
import AdminShell from '@/components/layout/AdminShell.vue'
import type { SidebarMenuGroup } from '@/components/layout/layout.types'

// 这个页面文件是“当前前端真正展示给用户看的页面入口”。
// 推荐阅读顺序：
// 1. 先看这里，知道页面挂了哪些大组件
// 2. 再看 AdminShell，理解后台骨架
// 3. 再看 OnlineNodeTable，理解真实业务数据怎么加载
//
// 当前这个页面本身不去请求接口。
// 它主要做两件事：
// 1. 组织页面结构
// 2. 提供侧边栏菜单数据

// activeMenu 表示当前高亮的菜单。
// ref('dashboard') 的意思是：默认先高亮 dashboard 这个菜单。
const activeMenu = ref('dashboard')

// 这里先用本地常量模拟后台菜单，后续接路由时可把 index 改成 route name/path。
const menuGroups: SidebarMenuGroup[] = [
  {
    title: '网络',
    items: [
      { index: 'dashboard', label: '概览面板', icon: Grid },
      { index: 'monitor', label: '在线节点', icon: Monitor },
      { index: 'links', label: '链路拓扑', icon: Connection }
    ]
  }
]
</script>

<template>
  <!--
    v-model:active-menu 可以先粗略理解成：
    把 activeMenu 这个变量交给 AdminShell 使用，
    同时允许 AdminShell 在菜单变化时把新值再同步回来。
    这是父子组件双向同步菜单状态的写法。
  -->
  <AdminShell
    v-model:active-menu="activeMenu"
    :menu-groups="menuGroups"
    page-title="NetWeaver 控制台"
    page-description=""
  >
    <!--
      当前真正的业务主体在 OnlineNodeTable 里。
      名字虽然叫 Table，但现在它实际上同时负责：
      1. Dashboard 登录和 JWT 管理
      2. stats / nodes / edges / metrics 请求
      3. 统计卡片、节点关系图、在线节点表格
    -->
    <OnlineNodeTable />
  </AdminShell>
</template>
