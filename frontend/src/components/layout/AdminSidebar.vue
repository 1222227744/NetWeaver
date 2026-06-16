<script setup lang="ts">
// defineProps 用来接收“父组件传进来的数据”。
// 这里的父组件是 AdminShell。
defineProps<{
  // 侧边栏的大标题。
  sidebarTitle: string
}>()
</script>

<template>
  <!--
    aside 标签可以理解成“侧边栏区域”。
    这个组件现在只负责展示品牌识别和一点视觉层次。
    它不承担导航职责，也不再显示说明标签。
  -->
  <aside
    class="relative flex h-full w-[14rem] flex-col overflow-hidden rounded-[2rem] bg-[linear-gradient(180deg,#0f172a_0%,#111827_45%,#0f3d39_100%)] p-4 text-slate-100"
  >
    <!-- 两个发光圆只是装饰背景，没有交互作用。 -->
    <div class="pointer-events-none absolute -right-10 top-8 h-28 w-28 rounded-full bg-cyan-300/15 blur-3xl" />
    <div class="pointer-events-none absolute -left-6 bottom-20 h-24 w-24 rounded-full bg-emerald-300/10 blur-3xl" />

    <section class="relative z-10 rounded-[1.5rem] border border-white/10 bg-white/5 p-4 backdrop-blur">
      <span class="inline-flex rounded-full bg-white/10 px-3 py-1 text-xs tracking-[0.22em] text-slate-300">
        NETWEAVER
      </span>
      <h2 class="mt-4 text-2xl font-semibold tracking-[0.06em]">{{ sidebarTitle }}</h2>
    </section>

    <div class="relative z-10 mt-6 flex-1 overflow-hidden rounded-[1.75rem] border border-white/10 bg-white/5 p-4 backdrop-blur">
      <div class="relative h-full overflow-hidden rounded-[1.35rem] border border-white/6 bg-[linear-gradient(180deg,rgba(255,255,255,0.05),rgba(15,23,42,0.08))]">
        <div class="absolute -right-6 top-8 h-24 w-24 rounded-full bg-cyan-300/15 blur-3xl" />
        <div class="absolute -left-8 bottom-4 h-28 w-28 rounded-full bg-emerald-300/10 blur-3xl" />
        <div class="absolute inset-x-6 top-1/2 h-px bg-white/10" />
      </div>
    </div>
  </aside>
</template>

<style scoped>
/* scoped 表示这些样式默认只作用于当前组件。 */
</style>
