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
    这是父子组件双向同步菜单状态的写法。
  -->
  <AdminShell
    v-model:active-menu="activeMenu"
    :menu-groups="menuGroups"
    page-title="控制台真实接口总览"
    page-description="当前页面会同时读取 stats、nodes、edges 和 metrics 相关接口，在同一页中展示真实统计卡片、真实节点关系图、在线节点表格，以及节点悬停时的链路监控空态或真实曲线。"
  >
    <!--
      这里往 AdminShell 顶部栏右侧的 actions 插槽里塞了两个标签。
      所以页面顶栏右边显示什么，不是写死在 AdminShell 里的，
      而是由这个页面自己决定。
    -->
    <template #header-actions>
      <el-tag size="large" round>GET /api/v1/dashboard/*</el-tag>
      <el-tag size="large" round type="success">Real API</el-tag>
    </template>

    <!--
      当前真正的业务主体在 OnlineNodeTable 里。
      名字虽然叫 Table，但现在它实际上同时负责：
      1. 统计卡片
      2. 节点关系图
      3. 在线节点表格
    -->
    <OnlineNodeTable />
  </AdminShell>
</template>
