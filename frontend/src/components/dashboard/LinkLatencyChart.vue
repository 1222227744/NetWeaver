<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import type { DashboardNode } from '@/api/dashboard'

interface HoverPanelPosition {
  left: number
  top: number
}

interface LatencyHistoryPoint {
  timeLabel: string
  latencyMs: number
}

const props = defineProps<{
  node: DashboardNode | null
  position: HoverPanelPosition
  visible: boolean
}>()

const chartContainer = ref<HTMLDivElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)

const hoverCardStyle = computed(() => ({
  left: `${props.position.left}px`,
  top: `${props.position.top}px`
}))

const latencyHistory = computed<LatencyHistoryPoint[]>(() => {
  if (!props.node) {
    return []
  }

  // 这里只保留“画图逻辑”，不请求任何后端接口。
  // 为了让每个节点的折线图长得不一样，这里根据节点自身字段生成一组稳定的本地演示数据。
  const baseLatency = props.node.nat_type === 'Symmetric' ? 42 : props.node.nat_type === 'Full Cone' ? 18 : 28
  const peerFactor = props.node.connected_peers * 2
  const nodeSeed = props.node.node_id
    .split('')
    .reduce((total, currentChar) => total + currentChar.charCodeAt(0), 0)

  return Array.from({ length: 8 }, (_, index) => {
    const waveOffset = ((nodeSeed + index * 17) % 9) - 4
    const latencyMs = Math.max(6, baseLatency + peerFactor + waveOffset)

    return {
      timeLabel: `${index + 1}m`,
      latencyMs
    }
  })
})

const chartSummary = computed(() => {
  if (!props.node) {
    return '当前没有可展示的节点信息。'
  }

  return `这里先展示节点 ${props.node.hostname} 的本地折线图预览。后续如果后端补充真实链路延迟接口，再把这组演示数据替换成真实采样结果。`
})

const renderChart = async () => {
  await nextTick()

  if (!props.visible || !chartContainer.value) {
    return
  }

  if (!chartInstance.value) {
    chartInstance.value = echarts.init(chartContainer.value)
  }

  chartInstance.value.setOption({
    animationDuration: 250,
    tooltip: {
      trigger: 'axis',
      formatter: (params: Array<Record<string, unknown>>) => {
        const point = params[0]

        if (!point) {
          return '暂无数据'
        }

        return [`时间片：${point.axisValueLabel ?? ''}`, `延迟：${point.data ?? ''} ms`].join('<br/>')
      }
    },
    grid: {
      top: 28,
      right: 16,
      bottom: 26,
      left: 42
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: latencyHistory.value.map((point) => point.timeLabel),
      axisLine: {
        lineStyle: {
          color: '#cbd5e1'
        }
      },
      axisLabel: {
        color: '#475569',
        fontSize: 11
      }
    },
    yAxis: {
      type: 'value',
      name: 'ms',
      min: 0,
      axisLine: {
        show: true,
        lineStyle: {
          color: '#cbd5e1'
        }
      },
      splitLine: {
        lineStyle: {
          color: '#e2e8f0'
        }
      },
      axisLabel: {
        color: '#475569',
        fontSize: 11
      }
    },
    series: [
      {
        name: '延迟预览',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        data: latencyHistory.value.map((point) => point.latencyMs),
        lineStyle: {
          width: 3,
          color: '#0f766e'
        },
        itemStyle: {
          color: '#0f766e'
        },
        areaStyle: {
          color: 'rgba(15, 118, 110, 0.14)'
        }
      }
    ]
  })

  chartInstance.value.resize()
}

const handleResize = () => {
  chartInstance.value?.resize()
}

watch(
  () => [props.visible, props.node?.node_id, latencyHistory.value.length],
  () => {
    void renderChart()
  },
  { deep: true }
)

onMounted(() => {
  void renderChart()
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance.value?.dispose()
  chartInstance.value = null
})
</script>

<template>
  <Teleport to="body">
    <div
      v-show="props.visible && props.node"
      class="pointer-events-none fixed z-[200] w-[360px] rounded-[1.5rem] border border-slate-200 bg-white/95 p-5 shadow-[0_24px_80px_rgba(15,23,42,0.18)] backdrop-blur"
      :style="hoverCardStyle"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <p class="panel-heading">Node Detail</p>
          <h3 class="mt-2 truncate text-lg font-semibold text-slate-900">{{ props.node?.hostname }}</h3>
          <p class="mt-1 text-sm text-slate-500">{{ props.node?.node_id }}</p>
        </div>

        <el-tag round :type="props.node?.status === 'online' ? 'success' : 'info'">
          {{ props.node?.status }}
        </el-tag>
      </div>

      <div class="mt-4 grid grid-cols-2 gap-3 text-sm text-slate-600">
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">虚拟 IP</p>
          <p class="mt-1 text-slate-700">{{ props.node?.virtual_ip }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">公网 IP</p>
          <p class="mt-1 text-slate-700">{{ props.node?.public_ip }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">NAT 类型</p>
          <p class="mt-1 text-slate-700">{{ props.node?.nat_type }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">已连接节点数</p>
          <p class="mt-1 text-slate-700">{{ props.node?.connected_peers }}</p>
        </div>
      </div>

      <div class="mt-4 rounded-[1.25rem] border border-slate-200 bg-slate-50 px-4 py-3">
        <p class="text-sm leading-6 text-slate-600">{{ chartSummary }}</p>
      </div>

      <div ref="chartContainer" class="mt-4 h-[180px] rounded-[1.25rem] border border-slate-200 bg-white" />
    </div>
  </Teleport>
</template>
