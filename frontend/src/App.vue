<script setup lang="ts">
// App.vue 是整个前端应用的“总入口组件”。
// 目前项目还没有接入 vue-router，所以这里先直接渲染一个后台首页。
// 以后如果你学到路由，通常会把这里换成 <RouterView />。
import AdminDashboardPage from '@/views/AdminDashboardPage.vue'
</script>

<template>
  <!-- 当前整个应用只显示这一个后台示例页面 -->
  <AdminDashboardPage />
</template>
