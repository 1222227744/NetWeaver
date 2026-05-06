<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

import { buildDashboardGraphData, type DashboardNode } from '@/api/dashboard'
import LinkLatencyChart from '@/components/dashboard/LinkLatencyChart.vue'

const props = defineProps<{
  onlineNodes: DashboardNode[]
}>()

const chartContainer = ref<HTMLDivElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)
const hoveredNode = ref<DashboardNode | null>(null)

const hoverPanel = reactive({
  visible: false,
  left: 0,
  top: 0
})

// 关系图的数据不是接口直接返回的，而是从节点数组转换而来。
// 这里统一在 computed 里生成 ECharts graph 需要的 nodes 和 links。
const graphData = computed(() => buildDashboardGraphData(props.onlineNodes))

const relationSummary = computed(() => {
  if (graphData.value.links.length === 0) {
    return '当前在线节点少于 2 个，无法生成连线。'
  }

  return `当前关系图使用前两个在线节点生成一条链路，链路类型为 ${graphData.value.links[0].relationText}。`
})

const updateHoverPanelPosition = (clientX: number, clientY: number) => {
  const cardWidth = 360
  const cardHeight = 420
  const viewportPadding = 18
  const cursorOffsetX = 24
  const cursorOffsetY = 20

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

const findDashboardNodeById = (nodeId: string) => {
  return props.onlineNodes.find((node) => node.node_id === nodeId) ?? null
}

const bindChartHoverEvents = () => {
  if (!chartInstance.value) {
    return
  }

  chartInstance.value.off('mouseover')
  chartInstance.value.off('mousemove')
  chartInstance.value.off('mouseout')
  chartInstance.value.off('drag')
  chartInstance.value.getZr().off('globalout')

  chartInstance.value.on('mouseover', (params: Record<string, unknown>) => {
    if (params.dataType !== 'node') {
      return
    }

    const nodeData = params.data as Record<string, string>
    const targetNode = findDashboardNodeById(nodeData.id)

    if (!targetNode) {
      return
    }

    hoveredNode.value = targetNode
    hoverPanel.visible = true

    const eventObject = (params.event as { event?: MouseEvent } | undefined)?.event

    if (eventObject) {
      updateHoverPanelPosition(eventObject.clientX, eventObject.clientY)
    }
  })

  chartInstance.value.on('mousemove', (params: Record<string, unknown>) => {
    if (params.dataType !== 'node' || !hoverPanel.visible) {
      return
    }

    const eventObject = (params.event as { event?: MouseEvent } | undefined)?.event

    if (eventObject) {
      updateHoverPanelPosition(eventObject.clientX, eventObject.clientY)
    }
  })

  chartInstance.value.on('mouseout', (params: Record<string, unknown>) => {
    if (params.dataType !== 'node') {
      return
    }

    hoverPanel.visible = false
  })

  chartInstance.value.getZr().on('globalout', () => {
    hoverPanel.visible = false
  })

  // 节点拖拽过程中也同步更新悬浮卡片的位置，这样卡片会跟着鼠标走。
  chartInstance.value.on('drag', (params: Record<string, unknown>) => {
    if (params.dataType !== 'node' || !hoverPanel.visible) {
      return
    }

    const eventObject = (params.event as { event?: MouseEvent } | undefined)?.event

    if (eventObject) {
      updateHoverPanelPosition(eventObject.clientX, eventObject.clientY)
    }
  })
}

const renderChart = async () => {
  await nextTick()

  if (!chartContainer.value) {
    return
  }

  if (!chartInstance.value) {
    chartInstance.value = echarts.init(chartContainer.value)
  }

  chartInstance.value.setOption({
    animationDuration: 500,
    tooltip: {
      trigger: 'item',
      formatter: (params: Record<string, unknown>) => {
        if (params.dataType === 'edge') {
          const edgeData = params.data as Record<string, string>
          return `链路类型：${edgeData.relationText}`
        }

        return ''
      }
    },
    series: [
      {
        type: 'graph',
        layout: 'none',
        roam: true,
        draggable: true,
        symbol: 'circle',
        edgeSymbol: ['none', 'arrow'],
        edgeSymbolSize: [0, 12],
        label: {
          position: 'bottom',
          distance: 10
        },
        edgeLabel: {
          show: true,
          fontSize: 14,
          backgroundColor: '#ffffff',
          borderColor: '#cbd5e1',
          borderWidth: 1,
          padding: [4, 8],
          borderRadius: 999
        },
        emphasis: {
          focus: 'adjacency'
        },
        data: graphData.value.nodes,
        links: graphData.value.links,
        lineStyle: {
          opacity: 0.95
        }
      }
    ]
  })

  bindChartHoverEvents()
}

const handleResize = () => {
  chartInstance.value?.resize()
}

watch(
  graphData,
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
  chartInstance.value?.getZr().off('globalout')
  chartInstance.value?.dispose()
  chartInstance.value = null
})
</script>

<template>
  <section class="panel-surface p-6">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <p class="panel-heading">Node Graph</p>
        <h2 class="mt-3 text-xl font-semibold text-slate-900">节点关系图</h2>
        <p class="mt-2 text-sm leading-6 text-slate-500">
          先将接口返回的节点数组转换成 ECharts graph 结构中的 <code>nodes</code> 与 <code>links</code>，再渲染为两节点单连线图。
        </p>
      </div>

      <el-tag round :type="graphData.links.length > 0 ? 'success' : 'info'">
        {{ graphData.links.length > 0 ? graphData.links[0].relationText : '无链路' }}
      </el-tag>
    </div>

    <div class="mt-6 rounded-[1.5rem] border border-slate-200 bg-slate-50 p-4">
      <p class="text-sm leading-6 text-slate-600">{{ relationSummary }}</p>
    </div>

    <div ref="chartContainer" class="mt-6 h-[360px] rounded-[1.5rem] border border-slate-200 bg-white" />

    <LinkLatencyChart
      :node="hoveredNode"
      :visible="hoverPanel.visible"
      :position="{ left: hoverPanel.left, top: hoverPanel.top }"
    />
  </section>
</template>
