<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

import {
  buildDashboardGraphData,
  type DashboardEdge,
  type DashboardMetricPoint,
  type DashboardMetricsTimeRange,
  type DashboardNode,
  type GraphLinkItem
} from '@/api/dashboard'
import LinkLatencyChart from '@/components/dashboard/LinkLatencyChart.vue'

// 这个组件只负责“拓扑图怎么画”和“鼠标碰到图上元素时怎么通知父组件”。
// 它不直接请求接口。
//
// 当前 v1.8 的关键规则是：
// - 节点 node 只代表一台机器
// - 边 edge 才代表两台机器之间的一条真实链路
// - 延迟 metrics 必须查询 source -> target，不能查询 node -> node 自己
//
// 所以这个组件会分别处理两种悬停：
// - 鼠标悬停节点：显示节点基本信息，不请求延迟曲线
// - 鼠标悬停连线：把这条边抛给父组件，由父组件请求真实链路 metrics

const props = defineProps<{
  edges: DashboardEdge[]
  metrics: DashboardMetricPoint[]
  metricsLoading: boolean
  onlineNodes: DashboardNode[]
  selectedTimeRange: DashboardMetricsTimeRange
}>()

const emit = defineEmits<{
  (event: 'node-hover', node: DashboardNode | null): void
  (event: 'link-hover', link: GraphLinkItem | null): void
}>()

const chartContainer = ref<HTMLDivElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)
const hoveredNode = ref<DashboardNode | null>(null)
const hoveredLink = ref<GraphLinkItem | null>(null)
let themeRenderFrame = 0

// ECharts graph 默认滚轮缩放比较猛，轻轻一滚就会变化很大。
// 这里把内置滚轮缩放关掉后，自己用更小的比例触发 graphRoam。
const graphWheelZoomSpeed = 0.0006

const hoverPanel = reactive({
  visible: false,
  left: 0,
  top: 0
})

interface ChartPointerPosition {
  clientX: number
  clientY: number
}

type HoverPanelKind = 'node' | 'link'

// computed 表示“根据 props 自动推导出来的数据”。
// graphData 是真正交给 ECharts 使用的数据格式。
// 注意：它可以随着 props 自动重新计算，但下面不会再直接深度监听 graphData。
// 原因是 dashboard 页面会定时轮询后端，节点里的 last_seen、流量等字段经常变化；
// 如果直接监听完整 graphData，就会导致 ECharts 每几秒强制重绘和重新排布一次。
const graphData = computed(() => buildDashboardGraphData(props.onlineNodes, props.edges))

// 这个签名只保留“会影响拓扑图本身”的字段。
// 它的作用可以理解为：给当前关系图拍一张很小的“身份证照片”。
//
// 会触发关系图重绘的变化：
// - 在线节点新增或消失
// - 节点名称、虚拟 IP、NAT 类型、状态、邻居数量变化
// - 连线 source / target / type 变化
//
// 不会触发关系图重绘的变化：
// - last_seen 心跳时间变化
// - current_rx_bytes / current_tx_bytes 流量变化
// - metrics 折线图采样变化
//
// 这样轮询仍然可以刷新表格和悬浮卡片数据，但关系图不会被无意义地重新布局。
const graphTopologySignature = computed(() => {
  const nodeSignature = props.onlineNodes
    .map((node) =>
      [
        node.node_id,
        node.hostname,
        node.virtual_ip,
        node.public_ip ?? '',
        node.nat_type,
        node.status,
        node.connected_peers
      ].join('::')
    )
    .sort()
    .join('|')

  const edgeSignature = props.edges
    .map((edge) => [edge.source, edge.target, edge.type].join('::'))
    .sort()
    .join('|')

  return `${nodeSignature}__${edgeSignature}`
})

const hoveredSourceNode = computed(() => {
  if (!hoveredLink.value) {
    return null
  }

  return findDashboardNodeById(hoveredLink.value.source)
})

const hoveredTargetNode = computed(() => {
  if (!hoveredLink.value) {
    return null
  }

  return findDashboardNodeById(hoveredLink.value.target)
})

const updateHoverPanelPosition = (clientX: number, clientY: number, panelKind: HoverPanelKind) => {
  // fixed 定位的悬浮卡片需要自己计算 left/top。
  // 这里同时做了边界保护，避免卡片跑出浏览器窗口。
  // 节点卡片没有折线图，链路卡片有折线图，所以两者高度不能共用同一个估算值。
  const cardWidth = 380
  const cardHeight = panelKind === 'link' ? 430 : 260
  const viewportPadding = 18
  const cursorOffsetX = 24
  const cursorOffsetY = 16

  let nextLeft = clientX + cursorOffsetX
  let nextTop = clientY - cardHeight - cursorOffsetY

  if (nextLeft + cardWidth > window.innerWidth - viewportPadding) {
    nextLeft = window.innerWidth - cardWidth - viewportPadding
  }

  if (nextLeft < viewportPadding) {
    nextLeft = viewportPadding
  }

  if (nextTop < viewportPadding) {
    nextTop = clientY + cursorOffsetY
  }

  if (nextTop + cardHeight > window.innerHeight - viewportPadding) {
    nextTop = window.innerHeight - cardHeight - viewportPadding
  }

  hoverPanel.left = nextLeft
  hoverPanel.top = nextTop
}

function findDashboardNodeById(nodeId: string) {
  return props.onlineNodes.find((node) => node.node_id === nodeId) ?? null
}

const getPointerPositionFromChartParams = (params: Record<string, unknown>): ChartPointerPosition | null => {
  const eventParams = params.event as
    | {
        event?: MouseEvent
        offsetX?: number
        offsetY?: number
      }
    | undefined

  // ECharts 正常情况下会把浏览器原生 MouseEvent 放在 params.event.event 里。
  // 这是最准确的坐标，适合 fixed 定位的悬浮卡片。
  if (
    typeof eventParams?.event?.clientX === 'number' &&
    typeof eventParams.event.clientY === 'number'
  ) {
    return {
      clientX: eventParams.event.clientX,
      clientY: eventParams.event.clientY
    }
  }

  // 少数情况下拿不到原生 MouseEvent，只能拿到图表内部的 offsetX/offsetY。
  // 这时用图表容器的浏览器位置做一次换算，避免悬浮卡片停在旧位置。
  if (
    chartContainer.value &&
    typeof eventParams?.offsetX === 'number' &&
    typeof eventParams.offsetY === 'number'
  ) {
    const chartRect = chartContainer.value.getBoundingClientRect()

    return {
      clientX: chartRect.left + eventParams.offsetX,
      clientY: chartRect.top + eventParams.offsetY
    }
  }

  return null
}

const showNodeHoverPanel = (node: DashboardNode, pointerPosition: ChartPointerPosition | null) => {
  hoveredNode.value = node
  hoveredLink.value = null
  hoverPanel.visible = true
  emit('node-hover', node)
  emit('link-hover', null)

  if (pointerPosition) {
    updateHoverPanelPosition(pointerPosition.clientX, pointerPosition.clientY, 'node')
  }
}

const showLinkHoverPanel = (link: GraphLinkItem, pointerPosition: ChartPointerPosition | null) => {
  hoveredNode.value = null
  hoveredLink.value = link
  hoverPanel.visible = true
  emit('node-hover', null)
  emit('link-hover', link)

  if (pointerPosition) {
    updateHoverPanelPosition(pointerPosition.clientX, pointerPosition.clientY, 'link')
  }
}

const hideHoverPanel = () => {
  hoverPanel.visible = false
  hoveredNode.value = null
  hoveredLink.value = null
  emit('node-hover', null)
  emit('link-hover', null)
}

const bindChartHoverEvents = () => {
  if (!chartInstance.value) {
    return
  }

  chartInstance.value.off('mouseover')
  chartInstance.value.off('mousemove')
  chartInstance.value.off('mouseout')
  chartInstance.value.off('click')
  chartInstance.value.off('drag')
  chartInstance.value.getZr().off('globalout')

  chartInstance.value.on('mouseover', (params: Record<string, unknown>) => {
    const pointerPosition = getPointerPositionFromChartParams(params)

    if (params.dataType === 'node') {
      const nodeData = params.data as { id?: string }
      const targetNode = nodeData.id ? findDashboardNodeById(nodeData.id) : null

      if (targetNode) {
        showNodeHoverPanel(targetNode, pointerPosition)
      }

      return
    }

    if (params.dataType === 'edge') {
      showLinkHoverPanel(params.data as GraphLinkItem, pointerPosition)
    }
  })

  chartInstance.value.on('click', (params: Record<string, unknown>) => {
    // 细线有时不太好悬停，所以额外支持点击连线。
    // 点击不会改变后端数据，只是让前端明确选中这条链路并刷新一次 metrics。
    if (params.dataType !== 'edge') {
      return
    }

    showLinkHoverPanel(params.data as GraphLinkItem, getPointerPositionFromChartParams(params))
  })

  chartInstance.value.on('mousemove', (params: Record<string, unknown>) => {
    if (!hoverPanel.visible || (params.dataType !== 'node' && params.dataType !== 'edge')) {
      return
    }

    const pointerPosition = getPointerPositionFromChartParams(params)

    if (pointerPosition) {
      updateHoverPanelPosition(
        pointerPosition.clientX,
        pointerPosition.clientY,
        hoveredLink.value ? 'link' : 'node'
      )
    }
  })

  chartInstance.value.on('mouseout', (params: Record<string, unknown>) => {
    if (params.dataType === 'node' || params.dataType === 'edge') {
      hideHoverPanel()
    }
  })

  chartInstance.value.getZr().on('globalout', () => {
    hideHoverPanel()
  })

  chartInstance.value.on('drag', (params: Record<string, unknown>) => {
    if (params.dataType !== 'node' || !hoverPanel.visible) {
      return
    }

    const pointerPosition = getPointerPositionFromChartParams(params)

    if (pointerPosition) {
      updateHoverPanelPosition(pointerPosition.clientX, pointerPosition.clientY, 'node')
    }
  })
}

const getThemeValue = (name: string, fallback: string) => {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback
}

const getChartTheme = () => {
  // ECharts 画在 canvas 上，不能像普通 HTML 一样自动继承 CSS 变量。
  // 所以每次渲染图表时，主动读取当前主题变量并写进 ECharts option。
  return {
    panelBorder: getThemeValue('--app-panel-border', '#cbd5e1'),
    panelBackground: getThemeValue('--app-panel-bg-solid', '#ffffff'),
    text: getThemeValue('--app-text', '#0f172a')
  }
}

const renderChart = async (options: { replace?: boolean } = {}) => {
  await nextTick()

  if (!chartContainer.value) {
    return
  }

  if (!chartInstance.value) {
    chartInstance.value = echarts.init(chartContainer.value)
  }

  const chartTheme = getChartTheme()
  const themedNodes = graphData.value.nodes.map((node) => ({
    ...node,
    label: {
      ...node.label,
      color: chartTheme.text
    }
  }))
  const themedLinks = graphData.value.links.map((link) => ({
    ...link,
    label: {
      ...link.label,
      color: chartTheme.text
    }
  }))

  chartInstance.value.setOption(
    {
      animationDuration: 500,
      tooltip: {
        // 关系图已经有自定义悬浮卡片。
        // 这里必须关掉 ECharts 默认 tooltip，否则会出现两个悬浮提示互相遮挡。
        show: false,
        trigger: 'none'
      },
      series: [
        {
          type: 'graph',
          layout: 'force',
          // roam: 'move' 只保留拖动画布平移。
          // 滚轮缩放由 handleGraphWheel 接管，避免默认缩放过于灵敏。
          roam: 'move',
          scaleLimit: {
            min: 0.45,
            max: 2.2
          },
          nodeScaleRatio: 0.35,
          draggable: true,
          symbol: 'circle',
          edgeSymbol: ['none', 'arrow'],
          edgeSymbolSize: [0, 12],
          force: {
            repulsion: 340,
            edgeLength: 190,
            gravity: 0.08
          },
          label: {
            position: 'bottom',
            distance: 10,
            color: chartTheme.text
          },
          edgeLabel: {
            show: true,
            fontSize: 14,
            backgroundColor: chartTheme.panelBackground,
            borderColor: chartTheme.panelBorder,
            borderWidth: 1,
            color: chartTheme.text,
            padding: [4, 8],
            borderRadius: 999
          },
          emphasis: {
            focus: 'adjacency'
          },
          data: themedNodes,
          links: themedLinks,
          lineStyle: {
            opacity: 0.95
          }
        }
      ]
    },
    options.replace ?? true
  )

  chartInstance.value.dispatchAction({ type: 'hideTip' })
  bindChartHoverEvents()
}

const handleResize = () => {
  chartInstance.value?.resize()
}

const handleGraphWheel = (event: WheelEvent) => {
  if (!chartInstance.value || !chartContainer.value) {
    return
  }

  event.preventDefault()

  const chartRect = chartContainer.value.getBoundingClientRect()
  const originX = event.clientX - chartRect.left
  const originY = event.clientY - chartRect.top
  const normalizedDelta =
    event.deltaMode === WheelEvent.DOM_DELTA_LINE
      ? event.deltaY * 16
      : event.deltaMode === WheelEvent.DOM_DELTA_PAGE
        ? event.deltaY * window.innerHeight
        : event.deltaY
  const rawZoom = Math.exp(-normalizedDelta * graphWheelZoomSpeed)
  const zoom = Math.min(1.08, Math.max(0.92, rawZoom))

  chartInstance.value.dispatchAction({
    type: 'graphRoam',
    seriesIndex: 0,
    zoom,
    originX,
    originY
  })
}

const handleThemeChange = () => {
  if (themeRenderFrame) {
    window.cancelAnimationFrame(themeRenderFrame)
  }

  themeRenderFrame = window.requestAnimationFrame(() => {
    themeRenderFrame = 0
    void renderChart({ replace: false })
  })
}

watch(graphTopologySignature, () => {
  void renderChart()
})

onMounted(() => {
  void renderChart()
  window.addEventListener('resize', handleResize)
  window.addEventListener('netweaver-theme-change', handleThemeChange)
  chartContainer.value?.addEventListener('wheel', handleGraphWheel, { passive: false })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('netweaver-theme-change', handleThemeChange)
  chartContainer.value?.removeEventListener('wheel', handleGraphWheel)
  if (themeRenderFrame) {
    window.cancelAnimationFrame(themeRenderFrame)
  }
  chartInstance.value?.getZr().off('globalout')
  chartInstance.value?.dispose()
  chartInstance.value = null
})
</script>

<template>
  <section class="panel-surface p-6">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h2 class="text-xl font-semibold text-slate-900">节点关系图</h2>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <el-tag round :type="graphData.links.length > 0 ? 'success' : 'info'">
          {{ graphData.links.length > 0 ? `${graphData.links.length} 条链路` : '无链路' }}
        </el-tag>
      </div>
    </div>

    <div ref="chartContainer" class="mt-6 h-[380px] rounded-[1.5rem] border border-slate-200 bg-white" />

    <LinkLatencyChart
      :link="hoveredLink"
      :loading="props.metricsLoading"
      :metrics="props.metrics"
      :node="hoveredNode"
      :position="{ left: hoverPanel.left, top: hoverPanel.top }"
      :selected-time-range="props.selectedTimeRange"
      :source-node="hoveredSourceNode"
      :target-node="hoveredTargetNode"
      :visible="hoverPanel.visible"
    />
  </section>
</template>
