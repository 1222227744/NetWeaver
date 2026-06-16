<script setup lang="ts">
import { Connection, Monitor, Share } from '@element-plus/icons-vue'

defineProps<{
  controllerUptimeText: string
  onlineNodes: number
  totalEdges: number
  totalNodes: number
}>()
</script>

<template>
  <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
    <article class="panel-surface p-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="panel-heading">总节点数</p>
          <p class="mt-4 text-3xl font-semibold text-slate-900">{{ totalNodes }}</p>
        </div>
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-50 text-brand-700">
          <el-icon class="text-xl"><Share /></el-icon>
        </div>
      </div>
    </article>

    <article class="panel-surface p-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="panel-heading">在线节点</p>
          <p class="mt-4 text-3xl font-semibold text-slate-900">{{ onlineNodes }}</p>
        </div>
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
          <el-icon class="text-xl"><Monitor /></el-icon>
        </div>
      </div>
    </article>

    <article class="panel-surface p-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="panel-heading">链路数</p>
          <p class="mt-4 text-3xl font-semibold text-slate-900">{{ totalEdges }}</p>
        </div>
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-50 text-cyan-700">
          <el-icon class="text-xl"><Connection /></el-icon>
        </div>
      </div>
    </article>

    <article class="panel-surface p-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="panel-heading">运行时长</p>
          <p class="mt-4 text-3xl font-semibold text-slate-900">{{ controllerUptimeText }}</p>
        </div>
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-700">
          <el-icon class="text-xl"><Share /></el-icon>
        </div>
      </div>
    </article>
  </section>
</template>
