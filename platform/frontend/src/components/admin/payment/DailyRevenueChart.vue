<template>
  <div class="card p-4">
    <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
      {{ t('payment.admin.dailyRevenue') }}
    </h3>
    <div class="h-64">
      <div v-if="loading" class="flex h-full items-center justify-center">
        <LoadingSpinner size="md" />
      </div>
      <Line v-else-if="chartData" :data="chartData" :options="chartOptions" />
      <div
        v-else
        class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('payment.admin.noData') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useChartTheme } from '@/composables/useChartTheme'
import { hexToRgba } from '@/utils/chartPlugins'
import type { DailyPaymentStats } from '@/types/payment'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = defineProps<{
  data: DailyPaymentStats[]
  loading?: boolean
}>()

const { tokens, seriesColor, tooltipTheme, legendTheme } = useChartTheme()

const chartData = computed(() => {
  if (!props.data || props.data.length === 0) return null
  const currencies = [...new Set(props.data.flatMap(day => Object.keys(day.amount)))].sort()
  return {
    labels: props.data.map(d => d.date),
    datasets: [
      ...currencies.map((currency, index) => {
        const borderColor = seriesColor(index)
        const backgroundColor = hexToRgba(borderColor, 0.12)
        return {
          label: `${currency} ${t('payment.admin.revenue')}`,
          data: props.data.map(day => day.amount[currency] || 0),
          borderColor,
          backgroundColor,
          fill: true,
          tension: 0.3,
          pointRadius: 3,
          pointHoverRadius: 5,
        }
      }),
      {
        label: t('payment.admin.orderCount'),
        data: props.data.map(d => d.count),
        borderColor: tokens.value.emerald,
        backgroundColor: hexToRgba(tokens.value.emerald, 0.12),
        fill: false,
        tension: 0.3,
        pointRadius: 3,
        pointHoverRadius: 5,
        yAxisID: 'y1',
      }
    ]
  }
})

const chartOptions = computed(() => {
  const c = tokens.value
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    scales: {
      x: {
        border: { display: false },
        grid: { display: false },
        ticks: { color: c.tick, font: { size: 10 } },
      },
      y: {
        type: 'linear' as const,
        display: true,
        position: 'left' as const,
        border: { display: false, dash: [4, 4] },
        grid: { color: c.grid },
        ticks: { color: c.tick, font: { size: 10 } },
        title: { display: true, text: t('payment.admin.revenue'), color: c.tick, font: { size: 11 } },
      },
      y1: {
        type: 'linear' as const,
        display: true,
        position: 'right' as const,
        border: { display: false },
        ticks: { color: c.emerald, font: { size: 10 } },
        title: { display: true, text: t('payment.admin.orderCount'), color: c.emerald, font: { size: 11 } },
        grid: { drawOnChartArea: false },
      }
    },
    plugins: {
      legend: legendTheme(),
      tooltip: tooltipTheme(),
    }
  }
})
</script>
