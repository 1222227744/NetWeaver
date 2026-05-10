<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import type { DashboardMetricPoint, DashboardNode } from '@/api/dashboard'

// 这个组件只负责“悬浮详情卡片 + 折线图显示”。
// 它不自己向后端发请求。
//
// 也就是说：
// - 谁请求 metrics？父组件请求
// - 谁决定当前展示哪个节点？父组件和关系图组件一起决定
// - 这个组件做什么？只负责把传进来的数据画出来

interface HoverPanelPosition {
  left: number
  top: number
}

const props = defineProps<{
  metrics: DashboardMetricPoint[]
  node: DashboardNode | null
  position: HoverPanelPosition
  visible: boolean
}>()

const chartContainer = ref<HTMLDivElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)

const hoverCardStyle = computed(() => ({
  // 因为这个卡片是 fixed 定位，所以 left/top 要自己计算像素位置。
  left: `${props.position.left}px`,
  top: `${props.position.top}px`
}))

const latencyHistory = computed(() => {
  // 后端给的是时间戳，这里把它转换成适合图表 x 轴显示的时间字符串。
  return props.metrics.map((point) => ({
    timeLabel: new Date(point.timestamp * 1000).toLocaleTimeString('zh-CN', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    }),
    latencyMs: point.latency_ms
  }))
})

const chartSummary = computed(() => {
  if (!props.node) {
    return '当前没有可展示的节点信息。'
  }

  if (latencyHistory.value.length === 0) {
    return `当前节点 ${props.node.hostname} 还没有返回链路监控数据。等后端补充 metrics 数据后，这里会展示真实延迟曲线。`
  }

  return `这里展示的是节点 ${props.node.hostname} 当前拿到的真实链路延迟采样结果。`
})

const renderChart = async () => {
  await nextTick()

  if (!props.visible || !chartContainer.value) {
    // 卡片不可见时，不需要继续画图。
    return
  }

  if (!chartInstance.value) {
    chartInstance.value = echarts.init(chartContainer.value)
  }

  if (latencyHistory.value.length === 0) {
    // 如果后端还没有返回 metrics，不画假数据，直接显示空态。
    chartInstance.value.clear()
    chartInstance.value.setOption({
      title: {
        text: '暂无链路数据',
        left: 'center',
        top: 'middle',
        textStyle: {
          color: '#94a3b8',
          fontSize: 15,
          fontWeight: 500
        }
      }
    })
    chartInstance.value.resize()
    return
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
        // 现在这里展示的已经不是“本地预览”，
        // 而是父组件传进来的真实 metrics 数据。
        name: '链路延迟',
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
  () => [props.visible, props.node?.node_id, props.metrics.length],
  () => {
    // 只要可见状态、节点 id、metrics 数量任意一个变化，就重画图。
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
    <!--
      Teleport 的意思可以简单理解成：
      “这个组件虽然写在当前层级里，但真正渲染时直接放到 body 下面去”。
      这样能避免被页面里别的盒子遮住。
    -->
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

      <!--
        真正的折线图就画在这个容器里。
        renderChart() 里会拿到这个 DOM，然后交给 ECharts 初始化。
      -->
      <div ref="chartContainer" class="mt-4 h-[180px] rounded-[1.25rem] border border-slate-200 bg-white" />
    </div>
  </Teleport>
</template>
