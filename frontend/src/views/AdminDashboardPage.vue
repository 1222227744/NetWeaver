<script setup lang="ts">
import {
  Bell,
  Connection,
  DataAnalysis,
  Document,
  Files,
  Grid,
  Monitor,
  Setting,
  User
} from '@element-plus/icons-vue'
import { ref } from 'vue'

import AdminShell from '@/components/layout/AdminShell.vue'
import type { SidebarMenuGroup } from '@/components/layout/layout.types'

// 这是“概览卡片”数据的类型说明。
// 你可以把 interface 理解成：先约定好每张卡片应该长什么样。
interface SummaryCard {
  id: string
  label: string
  value: string
  progress: number
  trend: string
  icon: typeof Grid
}

// activeMenu 表示当前高亮的菜单。
// ref('dashboard') 的意思是：默认先高亮 dashboard 这个菜单。
const activeMenu = ref('dashboard')

// 这里先用本地常量模拟后台菜单，后续接路由时可把 index 改成 route name/path。
const menuGroups: SidebarMenuGroup[] = [
  {
    title: '工作台',
    items: [
      { index: 'dashboard', label: '概览面板', icon: Grid },
      { index: 'monitor', label: '运行监控', icon: Monitor, hint: 'NEW' }
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

// 首页四张统计卡片的数据。
// 以后你如果接了接口，通常就是把这里的假数据替换成接口返回值。
const summaryCards: SummaryCard[] = [
  {
    id: 'project',
    label: '进行中项目',
    value: '12',
    progress: 76,
    trend: '+3 本周新增',
    icon: Files
  },
  {
    id: 'member',
    label: '在线成员',
    value: '28',
    progress: 64,
    trend: '协作状态稳定',
    icon: User
  },
  {
    id: 'api',
    label: '接口健康度',
    value: '98%',
    progress: 98,
    trend: '核心服务正常',
    icon: Connection
  },
  {
    id: 'report',
    label: '数据看板',
    value: '7',
    progress: 52,
    trend: '待补充图表',
    icon: DataAnalysis
  }
]

// 中间任务列表的示例数据。
const taskItems = [
  {
    id: 'task-1',
    title: '补齐成员列表筛选器交互',
    owner: '前端组',
    status: '进行中',
    type: 'warning' as const
  },
  {
    id: 'task-2',
    title: '接入登录态和权限菜单',
    owner: '全栈组',
    status: '待开发',
    type: 'info' as const
  },
  {
    id: 'task-3',
    title: '完善数据卡片接口映射',
    owner: '产品组',
    status: '已排期',
    type: 'success' as const
  }
]

// 右侧时间线/步骤说明的示例数据。
const releaseSteps = [
  {
    title: '布局骨架',
    time: '今天',
    description: '已完成侧边栏、头部栏和内容区拆分。'
  },
  {
    title: '业务组件',
    time: '下一步',
    description: '建议优先补图表卡片、表格分页和筛选表单。'
  },
  {
    title: '权限与路由',
    time: '待接入',
    description: '菜单数据可以直接迁移到路由配置或权限接口返回值。'
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
    page-title="后台管理骨架"
    page-description="这是一套适合作为后台起点继续开发的页面骨架，结构、间距和响应式已经先帮你铺好。"
  >
    <template #header-actions>
      <el-button plain>导出日报</el-button>
      <el-button type="primary">新建任务</el-button>
    </template>

    <div class="space-y-6">
      <!-- 概览卡片：后续最适合接首页统计接口或替换成图表组件 -->
      <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <!-- summaryCards 里有几条数据，就会渲染几张卡片 -->
        <article v-for="card in summaryCards" :key="card.id" class="panel-surface p-5">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="panel-heading">{{ card.label }}</p>
              <p class="mt-4 text-3xl font-semibold text-slate-900">{{ card.value }}</p>
              <p class="mt-2 text-sm text-slate-500">{{ card.trend }}</p>
            </div>
            <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-50 text-brand-700">
              <el-icon class="text-xl">
                <component :is="card.icon" />
              </el-icon>
            </div>
          </div>
          <el-progress
            :percentage="card.progress"
            :show-text="false"
            :stroke-width="6"
            color="#258d7d"
            class="mt-6"
          />
        </article>
      </section>

      <section class="grid gap-6 xl:grid-cols-[1.4fr_0.9fr]">
        <article class="panel-surface p-6">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="panel-heading">当前任务</p>
              <h2 class="mt-3 text-xl font-semibold text-slate-900">开发待办区</h2>
            </div>
            <el-tag round type="primary">可替换为表格 / 列表组件</el-tag>
          </div>

          <div class="mt-6 space-y-4">
            <!-- 这里是一个“循环渲染出来的任务列表” -->
            <div
              v-for="task in taskItems"
              :key="task.id"
              class="rounded-[1.5rem] border border-slate-200/80 bg-slate-50/80 p-4"
            >
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p class="text-base font-medium text-slate-900">{{ task.title }}</p>
                  <p class="mt-1 text-sm text-slate-500">{{ task.owner }}</p>
                </div>
                <el-tag round :type="task.type">{{ task.status }}</el-tag>
              </div>
            </div>
          </div>
        </article>

        <article class="panel-surface p-6">
          <p class="panel-heading">迭代建议</p>
          <h2 class="mt-3 text-xl font-semibold text-slate-900">接入路线</h2>

          <div class="mt-6 space-y-5">
            <!-- 右侧小时间线，同样也是通过数组循环生成 -->
            <div
              v-for="step in releaseSteps"
              :key="step.title"
              class="relative pl-6 before:absolute before:left-[0.4rem] before:top-2 before:h-full before:w-px before:bg-slate-200 last:before:hidden"
            >
              <span class="absolute left-0 top-1.5 h-3 w-3 rounded-full bg-brand-500" />
              <div class="flex items-center justify-between gap-3">
                <p class="font-medium text-slate-900">{{ step.title }}</p>
                <span class="text-xs uppercase tracking-[0.22em] text-slate-400">{{ step.time }}</span>
              </div>
              <p class="mt-2 text-sm leading-6 text-slate-500">{{ step.description }}</p>
            </div>
          </div>
        </article>
      </section>

      <section class="panel-surface p-6">
        <p class="panel-heading">开发提示</p>
        <h2 class="mt-3 text-xl font-semibold text-slate-900">你可以直接从这些位置继续扩展</h2>

        <!-- 这一块专门写给后续开发时参考，告诉你应该去改哪些地方 -->
        <div class="mt-6 grid gap-4 lg:grid-cols-3">
          <div class="rounded-[1.5rem] bg-slate-50 p-4">
            <p class="text-base font-medium text-slate-900">侧边栏菜单</p>
            <p class="mt-2 text-sm leading-6 text-slate-500">
              在当前页面顶部的 <code>menuGroups</code> 常量里补充菜单结构，后续接路由时基本不用改模板。
            </p>
          </div>
          <div class="rounded-[1.5rem] bg-slate-50 p-4">
            <p class="text-base font-medium text-slate-900">顶部操作区</p>
            <p class="mt-2 text-sm leading-6 text-slate-500">
              通过 <code>header-actions</code> 插槽放筛选器、用户下拉菜单、批量操作按钮会比较顺手。
            </p>
          </div>
          <div class="rounded-[1.5rem] bg-slate-50 p-4">
            <p class="text-base font-medium text-slate-900">内容主区域</p>
            <p class="mt-2 text-sm leading-6 text-slate-500">
              这里现在是示例卡片，后续换成表格、图表、表单或二级页面内容即可。
            </p>
          </div>
        </div>
      </section>
    </div>
  </AdminShell>
</template>
