import type { Component } from 'vue'

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
