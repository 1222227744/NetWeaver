<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import type {
  DashboardMetricPoint,
  DashboardMetricsTimeRange,
  DashboardNode,
  GraphLinkItem
} from '@/api/dashboard'

// 这个组件只负责“悬浮卡片怎么展示”和“折线图怎么画”。
// 它不发请求，也不决定当前应该展示哪条链路。
//
// v1.8 之后要特别注意：
// - metrics 是“两个节点之间的链路延迟”
// - 不是“某个节点自己的延迟”
// 所以 props 里同时接收 sourceNode、targetNode 和 link。

interface HoverPanelPosition {
  left: number
  top: number
}

const props = defineProps<{
  link: GraphLinkItem | null
  loading: boolean
  metrics: DashboardMetricPoint[]
  node: DashboardNode | null
  position: HoverPanelPosition
  selectedTimeRange: DashboardMetricsTimeRange
  sourceNode: DashboardNode | null
  targetNode: DashboardNode | null
  visible: boolean
}>()

const chartContainer = ref<HTMLDivElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)
let themeRenderFrame = 0

const hoverCardStyle = computed(() => ({
  left: `${props.position.left}px`,
  top: `${props.position.top}px`
}))

const latencyHistory = computed(() => {
  // 虽然接口文档规定后端按 timestamp 升序返回，
  // 但前端这里再排序一次，可以提升容错性。
  return [...props.metrics]
    .sort((previous, current) => previous.timestamp - current.timestamp)
    .map((point) => ({
      timeLabel: new Date(point.timestamp * 1000).toLocaleTimeString('zh-CN', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      }),
      latencyMs: point.latency_ms
    }))
})

const cardTitle = computed(() => {
  if (props.link) {
    return '链路详情'
  }

  return '节点详情'
})

const formatOptionalText = (value: string | number | null | undefined) => {
  if (value === null || value === undefined || value === '') {
    return '未上报'
  }

  return String(value)
}

const formatStatusText = (status: DashboardNode['status'] | undefined) => {
  if (status === 'online') {
    return '在线'
  }

  if (status === 'offline') {
    return '离线'
  }

  return '未知'
}

const getThemeValue = (name: string, fallback: string) => {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback
}

const getChartTheme = () => {
  // 悬浮卡片里的折线图也是 ECharts canvas。
  // 主题切换后，需要重新把 CSS 变量转换成 ECharts 能识别的颜色。
  return {
    axis: getThemeValue('--app-panel-border', '#cbd5e1'),
    fill: getThemeValue('--app-primary-soft', 'rgba(15, 118, 110, 0.14)'),
    line: getThemeValue('--app-primary', '#0f766e'),
    panelBackground: getThemeValue('--app-panel-bg-solid', '#ffffff'),
    text: getThemeValue('--app-text-soft', '#475569'),
    textMuted: getThemeValue('--app-text-muted', '#64748b')
  }
}

const renderChart = async () => {
  await nextTick()

  if (!props.visible || !props.link || !chartContainer.value) {
    chartInstance.value?.clear()
    return
  }

  if (!chartInstance.value) {
    chartInstance.value = echarts.init(chartContainer.value)
  }

  const chartTheme = getChartTheme()

  if (props.loading) {
    chartInstance.value.clear()
    chartInstance.value.setOption({
      title: {
        text: '链路数据加载中',
        left: 'center',
        top: 'middle',
        textStyle: {
          color: chartTheme.textMuted,
          fontSize: 15,
          fontWeight: 500
        }
      }
    }, true)
    chartInstance.value.resize()
    return
  }

  if (latencyHistory.value.length === 0) {
    chartInstance.value.clear()
    chartInstance.value.setOption({
      title: {
        text: '暂无链路数据',
        left: 'center',
        top: 'middle',
        textStyle: {
          color: chartTheme.textMuted,
          fontSize: 15,
          fontWeight: 500
        }
      }
    }, true)
    chartInstance.value.resize()
    return
  }

  chartInstance.value.setOption({
    title: {
      show: false
    },
    animationDuration: 250,
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartTheme.panelBackground,
      borderColor: chartTheme.axis,
      textStyle: {
        color: chartTheme.text
      },
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
          color: chartTheme.axis
        }
      },
      axisLabel: {
        color: chartTheme.text,
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
          color: chartTheme.axis
        }
      },
      splitLine: {
        lineStyle: {
          color: chartTheme.axis
        }
      },
      axisLabel: {
        color: chartTheme.text,
        fontSize: 11
      }
    },
    series: [
      {
        name: '链路延迟',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        data: latencyHistory.value.map((point) => point.latencyMs),
        lineStyle: {
          width: 3,
          color: chartTheme.line
        },
        itemStyle: {
          color: chartTheme.line
        },
        areaStyle: {
          color: chartTheme.fill
        }
      }
    ]
  }, true)

  chartInstance.value.resize()
}

const handleResize = () => {
  chartInstance.value?.resize()
}

const handleThemeChange = () => {
  if (themeRenderFrame) {
    window.cancelAnimationFrame(themeRenderFrame)
  }

  themeRenderFrame = window.requestAnimationFrame(() => {
    themeRenderFrame = 0
    void renderChart()
  })
}

watch(
  () => [
    props.visible,
    props.link?.id,
    props.loading,
    props.metrics.length,
    props.selectedTimeRange
  ],
  () => {
    void renderChart()
  }
)

watch(
  () => props.metrics,
  () => {
    void renderChart()
  },
  { deep: true }
)

onMounted(() => {
  void renderChart()
  window.addEventListener('resize', handleResize)
  window.addEventListener('netweaver-theme-change', handleThemeChange)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('netweaver-theme-change', handleThemeChange)
  if (themeRenderFrame) {
    window.cancelAnimationFrame(themeRenderFrame)
  }
  chartInstance.value?.dispose()
  chartInstance.value = null
})
</script>

<template>
  <Teleport to="body">
    <!--
      Teleport 会把悬浮卡片渲染到 body 下面。
      这样它不会被关系图、表格或其他父容器的层级挡住。
    -->
    <div
      v-show="props.visible && (props.node || props.link)"
      class="latency-hover-card pointer-events-none fixed z-[200] w-[380px] rounded-[1.5rem] border p-5 backdrop-blur"
      :style="hoverCardStyle"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <p class="panel-heading">{{ cardTitle }}</p>
          <h3 v-if="props.link && props.sourceNode && props.targetNode" class="mt-2 truncate text-lg font-semibold text-slate-900">
            {{ props.sourceNode.hostname }} -> {{ props.targetNode.hostname }}
          </h3>
          <h3 v-else class="mt-2 truncate text-lg font-semibold text-slate-900">
            {{ props.node?.hostname }}
          </h3>
          <p v-if="props.link" class="mt-1 text-sm text-slate-500">
            {{ props.link.source }} -> {{ props.link.target }}
          </p>
          <p v-else class="mt-1 text-sm text-slate-500">{{ props.node?.node_id }}</p>
        </div>

        <el-tag v-if="props.link" round :type="props.link.edgeType === 'p2p' ? 'success' : 'warning'">
          {{ props.link.relationText }}
        </el-tag>
        <el-tag v-else round :type="props.node?.status === 'online' ? 'success' : 'info'">
          {{ formatStatusText(props.node?.status) }}
        </el-tag>
      </div>

      <div v-if="props.link && props.sourceNode && props.targetNode" class="mt-4 grid grid-cols-2 gap-3 text-sm text-slate-600">
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">源节点虚拟 IP</p>
          <p class="mt-1 text-slate-700">{{ props.sourceNode.virtual_ip }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">目标节点虚拟 IP</p>
          <p class="mt-1 text-slate-700">{{ props.targetNode.virtual_ip }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">源节点 NAT</p>
          <p class="mt-1 text-slate-700">{{ props.sourceNode.nat_type }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">目标节点 NAT</p>
          <p class="mt-1 text-slate-700">{{ props.targetNode.nat_type }}</p>
        </div>
      </div>

      <div v-else-if="props.node" class="mt-4 grid grid-cols-2 gap-3 text-sm text-slate-600">
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">虚拟 IP</p>
          <p class="mt-1 text-slate-700">{{ props.node.virtual_ip }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">公网 IP</p>
          <p class="mt-1 text-slate-700">{{ formatOptionalText(props.node.public_ip) }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">NAT 类型</p>
          <p class="mt-1 text-slate-700">{{ props.node.nat_type }}</p>
        </div>
        <div class="rounded-2xl bg-slate-50 px-3 py-2">
          <p class="text-xs text-slate-400">已连接邻居</p>
          <p class="mt-1 text-slate-700">{{ props.node.connected_peers }}</p>
        </div>
      </div>

      <div
        v-show="props.link"
        ref="chartContainer"
        class="mt-4 h-[180px] rounded-[1.25rem] border border-slate-200 bg-white"
      />
    </div>
  </Teleport>
</template>

<style scoped>
.latency-hover-card {
  background: var(--app-panel-bg);
  border-color: var(--app-panel-border);
  box-shadow: 0 24px 80px hsl(var(--theme-hue) 45% 10% / 0.22);
  color: var(--app-text);
  transition:
    background-color 0.45s ease,
    border-color 0.45s ease,
    box-shadow 0.45s ease,
    color 0.45s ease;
}

.latency-hover-card .text-slate-900 {
  color: var(--app-text);
}

.latency-hover-card .text-slate-700 {
  color: var(--app-text-soft);
}

.latency-hover-card .text-slate-600,
.latency-hover-card .text-slate-500,
.latency-hover-card .text-slate-400 {
  color: var(--app-text-muted);
}

.latency-hover-card .bg-white,
.latency-hover-card .bg-slate-50 {
  background-color: var(--app-inner-bg);
}

.latency-hover-card .border-slate-200 {
  border-color: var(--app-panel-border);
}
</style>
