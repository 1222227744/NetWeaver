<script setup lang="ts">
// App.vue 是整个前端应用的“总入口组件”。
// 目前项目还没有接入 vue-router，所以这里先直接渲染一个后台首页。
// 以后如果你学到路由，通常会把这里换成 <RouterView />。
//
// 推荐理解方式：
// - main.ts 负责“把 Vue 应用启动起来”
// - App.vue 负责“决定应用最外层显示什么”
// - 具体的业务页面，再由 App.vue 往下一级一级引出去
import AdminDashboardPage from '@/views/AdminDashboardPage.vue'
</script>

<template>
  <!--
    当前整个应用只显示这一个后台示例页面。
    所以如果你现在运行项目后看到的是后台管理页，
    根本原因就是这里直接写了 <AdminDashboardPage />。
  -->
  <AdminDashboardPage />
</template>
