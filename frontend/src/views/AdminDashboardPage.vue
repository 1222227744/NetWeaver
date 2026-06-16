<script setup lang="ts">
import DashboardConsoleView from '@/components/dashboard/DashboardConsoleView.vue'
import AdminShell from '@/components/layout/AdminShell.vue'

// 这个页面文件是“当前前端真正展示给用户看的页面入口”。
// 推荐阅读顺序：
// 1. 先看这里，知道页面挂了哪些大组件
// 2. 再看 AdminShell，理解后台骨架
// 3. 再看 DashboardConsoleView，理解真实业务数据怎么加载
//
// 当前这个页面本身不去请求接口。
// 它主要做两件事：
// 1. 组织页面结构
// 2. 提供侧边栏说明数据

</script>

<template>
  <AdminShell
    page-title="NetWeaver 控制台"
    sidebar-title="网络控制台"
  >
    <!--
      当前真正的业务主体在 DashboardConsoleView 里。
      这个名字更贴近它现在的职责：
      1. Dashboard 登录和 JWT 管理
      2. stats / nodes / edges / metrics 请求
      3. 统计卡片、节点关系图、在线节点表格
    -->
    <DashboardConsoleView />
  </AdminShell>
</template>
