<script setup lang="ts">
import { Lock } from '@element-plus/icons-vue'

const props = defineProps<{
  errorMessage: string
  loading: boolean
  username: string
  password: string
}>()

const emit = defineEmits<{
  (event: 'update:username', value: string): void
  (event: 'update:password', value: string): void
  (event: 'submit'): void
}>()
</script>

<template>
  <section class="panel-surface mx-auto max-w-xl overflow-hidden p-8">
    <div class="absolute inset-x-10 top-0 h-24 rounded-full bg-brand-50/70 blur-3xl" />

    <div class="relative flex items-start gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-brand-50 text-brand-700">
        <el-icon class="text-xl"><Lock /></el-icon>
      </div>
      <div>
        <p class="panel-heading">安全入口</p>
        <h2 class="mt-2 text-2xl font-semibold text-slate-900">登录控制台</h2>
      </div>
    </div>

    <form class="relative mt-8 space-y-5" @submit.prevent="emit('submit')">
      <label class="block">
        <span class="text-sm font-medium text-slate-700">用户名</span>
        <el-input
          :model-value="props.username"
          class="mt-2"
          placeholder="例如：admin"
          size="large"
          @update:model-value="emit('update:username', $event)"
        />
      </label>

      <label class="block">
        <span class="text-sm font-medium text-slate-700">密码</span>
        <el-input
          :model-value="props.password"
          class="mt-2"
          placeholder="请输入控制台密码"
          show-password
          size="large"
          type="password"
          @update:model-value="emit('update:password', $event)"
        />
      </label>

      <el-alert
        v-if="props.errorMessage"
        :title="props.errorMessage"
        type="warning"
        show-icon
        :closable="false"
      />

      <el-button :loading="props.loading" native-type="submit" size="large" type="primary" class="w-full">
        登录并进入控制台
      </el-button>
    </form>
  </section>
</template>

<style scoped>
.panel-surface {
  position: relative;
}
</style>
