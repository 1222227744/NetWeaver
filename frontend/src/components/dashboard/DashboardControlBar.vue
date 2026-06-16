<script setup lang="ts">
import type { DashboardMetricsTimeRange } from '@/api/dashboard'

defineProps<{
  errorMessage: string
  lastUpdatedText: string
  loading: boolean
  metricsErrorMessage: string
  selectedTimeRange: DashboardMetricsTimeRange
  timeRangeOptions: Array<{ label: string; value: DashboardMetricsTimeRange }>
}>()

const emit = defineEmits<{
  (event: 'refresh'): void
  (event: 'update:selectedTimeRange', value: DashboardMetricsTimeRange): void
}>()
</script>

<template>
  <section class="panel-surface p-4">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
        <el-radio-group
          :model-value="selectedTimeRange"
          @update:model-value="emit('update:selectedTimeRange', $event)"
        >
          <el-radio-button
            v-for="option in timeRangeOptions"
            :key="option.value"
            :label="option.value"
          >
            {{ option.label }}
          </el-radio-button>
        </el-radio-group>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <el-tag
          round
          type="info"
          class="min-w-[12rem] justify-center [font-variant-numeric:tabular-nums]"
        >
          {{ lastUpdatedText || '尚未加载' }}
        </el-tag>
        <el-button :loading="loading" type="primary" @click="emit('refresh')">
          刷新
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="errorMessage"
      :title="errorMessage"
      type="error"
      show-icon
      :closable="false"
      class="mt-5"
    />

    <el-alert
      v-if="metricsErrorMessage"
      :title="metricsErrorMessage"
      type="warning"
      show-icon
      :closable="false"
      class="mt-5"
    />
  </section>
</template>
