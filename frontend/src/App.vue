<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

// App.vue 是整个前端应用的“总入口组件”。
// 目前项目还没有接入 vue-router，所以这里先直接渲染一个后台首页。
// 以后如果你学到路由，通常会把这里换成 <RouterView />。
//
// 推荐理解方式：
// - main.ts 负责“把 Vue 应用启动起来”
// - App.vue 负责“决定应用最外层显示什么”
// - 具体的业务页面，再由 App.vue 往下一级一级引出去
import AdminDashboardPage from '@/views/AdminDashboardPage.vue'

// cursorVisible 控制“鼠标光晕”是否显示。
// 这里没有隐藏浏览器原生鼠标，只是在原生鼠标下面额外加一层光效。
// 这样做的好处是：
// - 在白色或深色卡片里都更容易找到鼠标位置
// - 输入框、按钮、表格等控件仍然保持正常鼠标形态
const cursorVisible = ref(false)

const hideCursorEffect = () => {
  cursorVisible.value = false
}

const updateCursorEffect = (event: PointerEvent) => {
  // 把鼠标坐标写到 CSS 变量里，真正的动画交给 CSS transform 处理。
  // 这样 Vue 不需要在每次移动鼠标时重新渲染业务组件。
  document.documentElement.style.setProperty('--cursor-x', `${event.clientX}px`)
  document.documentElement.style.setProperty('--cursor-y', `${event.clientY}px`)
  cursorVisible.value = true
}

onMounted(() => {
  window.addEventListener('pointermove', updateCursorEffect)
  window.addEventListener('pointerleave', hideCursorEffect)
  window.addEventListener('blur', hideCursorEffect)
})

onBeforeUnmount(() => {
  window.removeEventListener('pointermove', updateCursorEffect)
  window.removeEventListener('pointerleave', hideCursorEffect)
  window.removeEventListener('blur', hideCursorEffect)
})
</script>

<template>
  <!--
    鼠标光晕是全局视觉效果，所以放在 App.vue 这一层。
    aria-hidden 表示它只是装饰，不需要被读屏软件读取。
  -->
  <div class="cursor-effects" :class="{ 'is-visible': cursorVisible }" aria-hidden="true">
    <div class="cursor-effects__glow" />
    <div class="cursor-effects__ring" />
  </div>

  <!--
    当前整个应用只显示这一个后台示例页面。
    所以如果你现在运行项目后看到的是后台管理页，
    根本原因就是这里直接写了 <AdminDashboardPage />。
  -->
  <AdminDashboardPage />
</template>
