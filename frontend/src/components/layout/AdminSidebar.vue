<script setup lang="ts">
import type { SidebarMenuGroup } from './layout.types'

// defineProps 用来接收“父组件传进来的数据”。
// 这里的父组件是 AdminShell。
defineProps<{
  // 当前应该高亮哪一个菜单。
  activeMenu: string
  // 整个侧边栏菜单的数据。
  menuGroups: SidebarMenuGroup[]
}>()

// defineEmits 用来声明“我要向父组件发什么消息”。
// 这里当用户点击菜单时，会通知父组件更新当前激活菜单。
const emit = defineEmits<{
  (event: 'update:activeMenu', value: string): void
}>()

// 用户点击菜单项后，不直接改 props，
// 而是把新值告诉父组件，这样数据流会更清晰。
const handleSelect = (menuKey: string) => {
  emit('update:activeMenu', menuKey)
}
</script>

<template>
  <!--
    aside 标签可以理解成“侧边栏区域”。
    这个组件当前只负责：
    1. 展示导航分组
    2. 高亮当前菜单
    3. 把点击结果通知给父组件
    它自己不负责切页面，因为项目当前还没有接 vue-router。
  -->
  <aside
    class="relative flex h-full w-[17.5rem] flex-col overflow-hidden rounded-[2rem] bg-[linear-gradient(180deg,#0f172a_0%,#111827_45%,#0f3d39_100%)] p-4 text-slate-100"
  >
    <!-- 两个发光圆只是装饰背景，没有交互作用。 -->
    <div class="pointer-events-none absolute -right-10 top-8 h-28 w-28 rounded-full bg-cyan-300/15 blur-3xl" />
    <div class="pointer-events-none absolute -left-6 bottom-20 h-24 w-24 rounded-full bg-emerald-300/10 blur-3xl" />

    <!-- 品牌区：后续可以替换为公司 Logo、租户切换器或当前站点信息 -->
    <section class="relative z-10 rounded-[1.5rem] border border-white/10 bg-white/5 p-4 backdrop-blur">
      <span class="inline-flex rounded-full bg-white/10 px-3 py-1 text-xs tracking-[0.22em] text-slate-300">
        NETWEAVER
      </span>
      <h2 class="mt-4 text-2xl font-semibold tracking-[0.06em]">管理后台</h2>
      <p class="mt-2 text-sm leading-6 text-slate-300/80">
        这里保留为全局导航入口，你后续接入真实菜单、权限和路由时，可以直接替换下面的菜单数据。
      </p>
    </section>

    <!--
      菜单区放在可滚动容器里。
      这样如果以后菜单变多，侧边栏不会直接撑出屏幕外，而是出现滚动条。
    -->
    <el-scrollbar class="relative z-10 mt-5 flex-1 pr-1">
      <div class="space-y-6">
        <section v-for="group in menuGroups" :key="group.title">
          <p class="mb-3 px-3 text-xs font-semibold uppercase tracking-[0.28em] text-slate-400">
            {{ group.title }}
          </p>

          <!-- 菜单当前只负责展示结构，后续接入 vue-router 时可把 index 改为 path/name -->
          <el-menu
            :default-active="activeMenu"
            class="admin-menu"
            :collapse-transition="false"
            @select="handleSelect"
          >
            <!--
              v-for 的意思是“循环渲染”。
              group.items 里有几个菜单对象，这里就会生成几个菜单项。
              :index 可以理解成“这个菜单项的唯一编号”。
            -->
            <el-menu-item v-for="item in group.items" :key="item.index" :index="item.index">
              <el-icon class="text-base">
                <component :is="item.icon" />
              </el-icon>
              <span>{{ item.label }}</span>
              <el-tag
                v-if="item.hint"
                effect="dark"
                round
                size="small"
                class="ml-auto border-none !bg-white/10 !text-slate-200"
              >
                {{ item.hint }}
              </el-tag>
            </el-menu-item>
          </el-menu>
        </section>
      </div>
    </el-scrollbar>

    <section class="relative z-10 mt-4 rounded-[1.5rem] border border-emerald-200/10 bg-white/5 p-4">
      <p class="text-sm font-medium text-slate-200">当前环境</p>
      <div class="mt-3 flex items-center gap-2">
        <el-tag round type="success">开发中</el-tag>
        <!-- 这是底部的小提示块，你以后也可以换成版本号、构建时间、登录信息 -->
        <span class="text-xs text-slate-400">适合作为后台框架起点继续扩展</span>
      </div>
    </section>
  </aside>
</template>

<style scoped>
/* scoped 表示这些样式默认只作用于当前组件。 */

/* :deep(...) 用来修改 Element Plus 组件内部生成的类名。
   因为 el-menu 里面的真实 DOM 不是我们手写的，所以要用 :deep 才能选中。 */
:deep(.admin-menu) {
  border-right: none;
  background: transparent;
}

/* 普通菜单项的样式 */
:deep(.admin-menu .el-menu-item) {
  margin-bottom: 0.5rem;
  height: 3rem;
  border-radius: 1rem;
  color: rgba(226, 232, 240, 0.82);
  background: transparent;
}

/* 鼠标移上去时的样式 */
:deep(.admin-menu .el-menu-item:hover) {
  color: #ffffff;
  background: rgba(255, 255, 255, 0.08);
}

/* 当前选中菜单的样式 */
:deep(.admin-menu .el-menu-item.is-active) {
  color: #ffffff;
  background: linear-gradient(135deg, rgba(45, 212, 191, 0.22), rgba(14, 165, 233, 0.26));
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}
</style>
