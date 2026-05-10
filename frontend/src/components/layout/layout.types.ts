import type { Component } from 'vue'

// 这个文件只放“类型定义”，不放页面逻辑。
// 你可以把它理解成：专门声明“菜单数据长什么样”的说明书。
//
// 为什么要单独拆文件？
// 因为侧边栏、页面、甚至以后路由配置都可能会用到同一套菜单数据结构，
// 单独放出来会更清楚，也避免每个文件重复写一遍。

// 单个菜单项的数据结构。
// 例如：概览面板、运行监控、消息中心，都属于一个菜单项。
export interface SidebarMenuItem {
  // 菜单的唯一标识。
  // 现在用于高亮当前菜单；以后接入路由时，也可以改成路由 name 或 path。
  index: string
  // 菜单显示给用户看的文字。
  label: string
  // 菜单图标组件，例如 Grid、User、Setting。
  icon: Component
  // 可选提示，例如 NEW、12 这样的角标文字。
  hint?: string
}

// 菜单分组的数据结构。
// 例如“工作台”“协作管理”“系统设置”就是菜单分组。
export interface SidebarMenuGroup {
  // 分组标题。
  title: string
  // 当前分组下面有哪些菜单项。
  items: SidebarMenuItem[]
}
