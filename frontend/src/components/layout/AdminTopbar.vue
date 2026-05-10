<script setup lang="ts">
import { Bell, Fold, Plus, SwitchButton } from '@element-plus/icons-vue'

// 顶部栏需要知道当前页面标题和页面说明，
// 这些内容由父组件 AdminShell 传进来。
defineProps<{
  pageDescription: string
  pageTitle: string
}>()

// 当屏幕较小时，顶部左上角会出现一个按钮。
// 点击它时，顶部栏会发出 toggle-sidebar 事件，让父组件打开抽屉侧边栏。
const emit = defineEmits<{
  (event: 'toggle-sidebar'): void
}>()
</script>

<template>
  <!--
    sticky top-4 的意思可以简单理解成：
    页面往下滚动时，顶部栏会尽量保持在视口顶部附近，而不是立刻滚走。
  -->
  <header class="panel-surface sticky top-4 z-20 px-4 py-4 sm:px-6">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex items-start gap-3">
        <!-- 这个按钮只在小屏幕出现，大屏幕时侧边栏本来就是常驻显示 -->
        <el-button class="lg:hidden" circle plain @click="emit('toggle-sidebar')">
          <el-icon><Fold /></el-icon>
        </el-button>

        <div>
          <!-- 面包屑和标题区域：告诉用户现在正在后台的哪个位置 -->
          <p class="panel-heading">Admin Workspace</p>
          <div class="mt-2 flex flex-wrap items-center gap-3">
            <h1 class="text-2xl font-semibold tracking-[0.04em] text-slate-900">
              {{ pageTitle }}
            </h1>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item>控制台</el-breadcrumb-item>
              <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          <p class="mt-2 text-sm leading-6 text-slate-500">
            {{ pageDescription }}
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3 lg:justify-end">
        <!--
          头部操作区开放为 slot。
          slot 可以理解成“预留插槽”：
          父组件想往这里塞什么内容都行。
          现在如果父组件没传内容，就显示下面这两个默认按钮。
          这就是为什么 AdminDashboardPage.vue 能往这里插 GET /api/v1/dashboard/* 标签。
        -->
        <slot name="actions">
          <el-button plain>
            <el-icon><Bell /></el-icon>
            消息中心
          </el-button>
          <el-button type="primary">
            <el-icon><Plus /></el-icon>
            新建内容
          </el-button>
        </slot>

        <!-- 右侧用户信息块：后续可替换成头像下拉菜单、退出登录等真实功能 -->
        <div class="flex items-center gap-3 rounded-2xl bg-slate-50 px-3 py-2">
          <div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-brand-600 text-sm font-semibold text-white">
            NW
          </div>
          <div>
            <p class="text-sm font-medium text-slate-900">产品协作组</p>
            <p class="text-xs text-slate-500">frontend@netweaver</p>
          </div>
          <el-button circle plain>
            <el-icon><SwitchButton /></el-icon>
          </el-button>
        </div>
      </div>
    </div>
  </header>
</template>
