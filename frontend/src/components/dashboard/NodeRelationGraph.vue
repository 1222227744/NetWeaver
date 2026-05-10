<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

import { buildDashboardGraphData, type DashboardEdge, type DashboardMetricPoint, type DashboardNode } from '@/api/dashboard'
import LinkLatencyChart from '@/components/dashboard/LinkLatencyChart.vue'

// 这个组件专门负责“把节点和边画成关系图”。
// 它不自己请求接口，而是接收父组件已经准备好的数据：
// - onlineNodes：节点数组
// - edges：连线数组
// - metrics：当前悬停节点对应的折线图数据
//
// 这样分工的好处是：
// - 请求逻辑都集中在父组件 OnlineNodeTable.vue
// - 图组件只管“怎么画”和“怎么响应鼠标”

const props = defineProps<{
  edges: DashboardEdge[]
  metrics: DashboardMetricPoint[]
  onlineNodes: DashboardNode[]
}>()

// defineEmits 用来定义“我要把什么消息告诉父组件”。
// 这里的 node-hover 可以理解成：
// “父组件，我现在悬停到这个节点了，你要不要去做点别的？”
const emit = defineEmits<{
  (event: 'node-hover', node: DashboardNode | null): void
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
// computed 可以简单理解成“根据已有数据自动推导出来的新数据”。
const graphData = computed(() => buildDashboardGraphData(props.onlineNodes, props.edges))

const relationSummary = computed(() => {
  if (graphData.value.links.length === 0) {
    return '当前还没有可渲染的真实链路数据，请确认后端 edges 接口已经返回节点关系。'
  }

  return `当前关系图共渲染 ${graphData.value.nodes.length} 个节点、${graphData.value.links.length} 条真实链路。拖拽节点只会改变画面位置，不会改动后端数据。`
})

const updateHoverPanelPosition = (clientX: number, clientY: number) => {
  // 这个函数只负责一件事：
  // 让悬浮卡片尽量跟着鼠标，又不要跑出屏幕边界。
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
  // 图表事件里通常只会给我们一个节点 id，
  // 所以这里再回到在线节点数组里，把完整节点对象找出来。
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
    // dataType === 'node' 说明当前鼠标压到的是一个节点，不是一条边。
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

    // 通知父组件：当前悬停到哪个节点了。
    // 父组件收到后，会去请求这个节点的 metrics 数据。
    emit('node-hover', targetNode)

    const eventObject = (params.event as { event?: MouseEvent } | undefined)?.event

    if (eventObject) {
      updateHoverPanelPosition(eventObject.clientX, eventObject.clientY)
    }
  })

  chartInstance.value.on('mousemove', (params: Record<string, unknown>) => {
    // 鼠标在节点上移动时，只更新卡片位置，不重新请求数据。
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

    // 鼠标移走后，卡片隐藏，同时告诉父组件“当前没有悬停节点了”。
    hoverPanel.visible = false
    hoveredNode.value = null
    emit('node-hover', null)
  })

  chartInstance.value.getZr().on('globalout', () => {
    hoverPanel.visible = false
    hoveredNode.value = null
    emit('node-hover', null)
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

  // 当前关系图用的是 graph 系列。
  // 这里的 setOption 可以理解成：把图的完整配置一次性告诉 ECharts。
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
        // force 布局会让节点自动分散开，比较适合关系图。
        layout: 'force',
        roam: true,
        draggable: true,
        symbol: 'circle',
        edgeSymbol: ['none', 'arrow'],
        edgeSymbolSize: [0, 12],
        force: {
          repulsion: 320,
          edgeLength: 180,
          gravity: 0.08
        },
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
          当前图直接消费后端返回的真实节点数组与真实边数组，不再使用前端本地推导的两节点演示连线。
        </p>
      </div>

      <el-tag round :type="graphData.links.length > 0 ? 'success' : 'info'">
        {{ graphData.links.length > 0 ? `${graphData.links.length} 条链路` : '无链路' }}
      </el-tag>
    </div>

    <div class="mt-6 rounded-[1.5rem] border border-slate-200 bg-slate-50 p-4">
      <p class="text-sm leading-6 text-slate-600">{{ relationSummary }}</p>
    </div>

    <div ref="chartContainer" class="mt-6 h-[360px] rounded-[1.5rem] border border-slate-200 bg-white" />

    <!--
      折线图卡片本身不再请求数据。
      它只是把这里传进去的 node、metrics、visible、position 显示出来。
    -->
    <LinkLatencyChart
      :metrics="props.metrics"
      :node="hoveredNode"
      :visible="hoverPanel.visible"
      :position="{ left: hoverPanel.left, top: hoverPanel.top }"
    />
  </section>
</template>
