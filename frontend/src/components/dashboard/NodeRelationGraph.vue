<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { buildDashboardGraphData, type DashboardNode } from '@/api/dashboard'

const props = defineProps<{
  onlineNodes: DashboardNode[]
}>()

const chartContainer = ref<HTMLDivElement | null>(null)
const chartInstance = ref<echarts.ECharts | null>(null)

// 关系图的数据不是接口直接返回的，而是从节点数组转换而来。
// 这里统一在 computed 里生成 ECharts graph 需要的 nodes 和 links。
const graphData = computed(() => buildDashboardGraphData(props.onlineNodes))

const relationSummary = computed(() => {
  if (graphData.value.links.length === 0) {
    return '当前在线节点少于 2 个，无法生成连线。'
  }

  return `当前关系图使用前两个在线节点生成一条链路，链路类型为 ${graphData.value.links[0].relationText}。`
})

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

        const nodeData = params.data as Record<string, string | number>
        return [
          `主机名：${nodeData.hostname}`,
          `节点 ID：${nodeData.id}`,
          `虚拟 IP：${nodeData.virtualIp}`,
          `公网 IP：${nodeData.publicIp}`,
          `NAT 类型：${nodeData.natType}`
        ].join('<br/>')
      }
    },
    series: [
      {
        type: 'graph',
        layout: 'none',
        roam: true,
        draggable: false,
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
  </section>
</template>
