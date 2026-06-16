<script setup lang="ts">
import { ref } from 'vue'

import AdminSidebar from './AdminSidebar.vue'
import AdminTopbar from './AdminTopbar.vue'

// 这个组件是后台页面最外层的“骨架”：
// 左边是侧边栏，右边是顶部栏和内容区。
// 具体内容不写死在这里，而是通过 slot 由外部传进来。
//
// 推荐把它理解成“后台页面的模板”：
// - 左边永远是品牌侧栏
// - 上面永远是标题栏
// - 中间大块区域才是每个具体业务页面的内容
const props = defineProps<{
  pageDescription?: string
  pageTitle: string
  sidebarTitle: string
}>()

// 控制移动端抽屉菜单是否打开。
// ref(...) 可以理解成“会跟着界面一起更新的变量”。
const mobileSidebarVisible = ref(false)

// 只在移动端时需要主动打开侧边栏。
const openSidebar = () => {
  mobileSidebarVisible.value = true
}
</script>

<template>
  <!--
    这个组件自己不关心“业务数据是什么”，
    它只负责搭出后台的通用框架。
    所以后面你们新增页面时，大概率仍然会复用这个组件。
  -->
  <div class="app-theme-shell min-h-screen text-slate-900">
    <div class="mx-auto flex min-h-screen max-w-[1800px] gap-6 p-4 sm:p-6">
      <!-- 大屏下保持固定侧边导航，后台常用入口不需要折叠到内容流里 -->
      <AdminSidebar
        class="hidden h-[calc(100vh-3rem)] shrink-0 lg:flex"
        :sidebar-title="props.sidebarTitle"
      />

      <!--
        el-drawer 是 Element Plus 的抽屉组件。
        小屏幕时把侧边栏放进抽屉里，避免内容区被挤得太窄。
      -->
      <el-drawer v-model="mobileSidebarVisible" :with-header="false" direction="ltr" size="300px">
        <!-- 移动端沿用同一份侧边栏组件，避免未来维护两套导航结构 -->
        <AdminSidebar
          class="h-full rounded-none"
          :sidebar-title="props.sidebarTitle"
        />
      </el-drawer>

      <main class="flex min-w-0 flex-1 flex-col gap-6">
        <!-- 顶部栏负责标题、说明和右侧操作区 -->
        <AdminTopbar
          :page-description="props.pageDescription"
          :page-title="props.pageTitle"
          @toggle-sidebar="openSidebar"
        >
          <!--
            header-actions 是顶部栏右侧的“自定义插槽”。
            谁使用 AdminShell，谁就可以往这个位置塞按钮、标签或别的操作内容。
          -->
          <template #actions>
            <slot name="header-actions" />
          </template>
        </AdminTopbar>

        <!-- 业务主内容区：后续可替换成真实图表、表格、表单或子页面容器 -->
        <section class="flex-1">
          <!-- 这里的 slot 就是“内容插槽”，具体显示什么由外层页面决定 -->
          <slot />
        </section>
      </main>
    </div>
  </div>
</template>

<style scoped>
/* 去掉抽屉默认背景和默认内边距，让里面的侧边栏能完整铺满。 */
:deep(.el-drawer) {
  background: transparent;
}

:deep(.el-drawer__body) {
  padding: 0;
}
</style>
