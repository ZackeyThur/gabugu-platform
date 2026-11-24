<template>
  <section
    class="overview-chart-panel"
    aria-labelledby="spending-heading"
    :aria-busy="loading"
  >
    <div class="overview-panel-heading overview-trend-head">
      <div class="overview-trend-title-wrap">
        <h3 id="spending-heading">{{ panelTitle }}</h3>
        <div
          class="overview-metric-tabs g-segmented"
          role="tablist"
          :aria-label="t('dashboard.overview.trendMetric')"
          :style="{ '--seg-count': metricTabs.length, '--seg-index': activeMetricIndex }"
        >
          <span class="g-segmented-thumb" aria-hidden="true" />
          <button
            v-for="tab in metricTabs"
            :key="tab.id"
            type="button"
            role="tab"
            class="overview-metric-tab"
            :class="{ 'is-active': metric === tab.id }"
            :aria-selected="metric === tab.id"
            @click="metric = tab.id"
          >
            {{ tab.label }}
          </button>
        </div>
      </div>

      <div
        v-if="metric === 'cost' || metric === 'requests'"
        class="overview-chart-style-toggle g-segmented"
        role="group"
        :aria-label="t('dashboard.overview.chartStyle')"
        :style="{ '--seg-count': 2, '--seg-index': chartStyle === 'bar' ? 0 : 1 }"
      >
        <span class="g-segmented-thumb" aria-hidden="true" />
        <button
          type="button"
          class="overview-style-btn"
          :class="{ 'is-active': chartStyle === 'bar' }"
          :aria-pressed="chartStyle === 'bar'"
          :title="t('dashboard.overview.chartBar')"
          @click="chartStyle = 'bar'"
        >
          <Icon name="chartBar" size="sm" />
          <span>{{ t('dashboard.overview.chartBarShort') }}</span>
        </button>
        <button
          type="button"
          class="overview-style-btn"
          :class="{ 'is-active': chartStyle === 'area' }"
          :aria-pressed="chartStyle === 'area'"
          :title="t('dashboard.overview.chartArea')"
          @click="chartStyle = 'area'"
        >
          <Icon name="trendingUp" size="sm" />
          <span>{{ t('dashboard.overview.chartAreaShort') }}</span>
        </button>
      </div>
    </div>

    <div v-if="loading" class="overview-chart-loading" role="status">
      <div class="overview-skeleton" />
      {{ t('common.loading') }}
    </div>
    <div v-else-if="error" class="overview-empty" role="alert">
      <p>{{ t('dashboard.overview.chartError') }}</p>
      <button type="button" class="overview-retry-btn" @click="$emit('retry')">
        <Icon name="refresh" size="xs" />
        {{ t('common.retry') }}
      </button>
    </div>
    <template v-else>
      <div class="overview-chart-summary overview-trend-summary">
        <div class="overview-trend-kpis">
          <div class="overview-trend-kpi-primary">
            <p class="overview-muted">{{ primaryMetricLabel }}</p>
            <strong :title="primaryMetricTitle">
              <CountUp
                v-if="metric === 'cost'"
                :value="periodTotalCost"
                :formatter="overviewMoney"
                :duration="320"
                :update-duration="260"
                split-decimals
              />
              <template v-else-if="splitMetricUnit(primaryMetricFormatted)">
                <span>{{ splitMetricUnit(primaryMetricFormatted)!.value }}</span><span class="overview-unit-suffix">{{ splitMetricUnit(primaryMetricFormatted)!.unit }}</span>
              </template>
              <template v-else>{{ primaryMetricFormatted }}</template>
            </strong>
          </div>
          <div class="overview-trend-kpi-pills">
            <span v-if="metric !== 'cache'" class="overview-kpi-chip">
              <span>{{ t('dashboard.overview.periodAvg') }}</span>
              <strong>{{ avgFormatted }}</strong>
            </span>
            <span v-if="peakBucket" class="overview-kpi-chip">
              <span>{{ t('dashboard.overview.periodPeak') }}</span>
              <strong>{{ peakBucket.label }} · {{ formatBucketPrimary(peakBucket) }}</strong>
            </span>
            <span v-if="metric === 'cache' && periodCacheEfficiency.cacheReadTokens > 0" class="overview-kpi-chip">
              <span>{{ t('dashboard.overview.tokenCacheRead') }}</span>
              <strong>{{ formatCompactTokens(periodCacheEfficiency.cacheReadTokens) }}</strong>
            </span>
            <span v-else-if="metric !== 'cache' && periodCacheRate !== null" class="overview-kpi-chip">
              <span>{{ t('dashboard.overview.cacheHitShort') }}</span>
              <strong>{{ periodCacheRate.toFixed(1) }}%</strong>
            </span>
            <span v-if="legendItems.length === 0 && hasToday" class="overview-today-note is-inline">
              {{ t('dashboard.overview.todayIncomplete') }}
            </span>
          </div>
        </div>

        <!-- Custom Interactive Legend Bar (replaces ugly native canvas legend) -->
        <div v-if="legendItems.length > 0" class="overview-trend-legend" role="group" :aria-label="t('dashboard.overview.trendMetric')">
          <button
            v-for="item in legendItems"
            :key="item.id"
            type="button"
            class="overview-legend-pill"
            :class="{ 'is-off': !item.active, 'is-interactive': item.toggleable }"
            :aria-pressed="item.toggleable ? item.active : undefined"
            :disabled="!item.toggleable"
            @click="item.toggleable && toggleSeries(item.id)"
          >
            <span
              class="overview-legend-swatch"
              :class="`is-${item.kind}`"
              :style="{ '--swatch-color': item.color }"
              aria-hidden="true"
            />
            <span>{{ item.label }}</span>
          </button>
          <span v-if="hasToday" class="overview-today-note">{{ t('dashboard.overview.todayIncomplete') }}</span>
        </div>
      </div>

      <div
        ref="canvasWrapRef"
        class="overview-trend-canvas-wrap"
        tabindex="0"
        role="region"
        :aria-label="t('dashboard.overview.trendCanvasAria')"
        @keydown="onCanvasKeydown"
        @mouseleave="onCanvasLeave"
      >
        <Bar
          v-if="useBarComponent"
          ref="barChartRef"
          :data="chartData as any"
          :options="chartOptions as any"
          :plugins="chartPlugins"
        />
        <Line
          v-else
          ref="lineChartRef"
          :data="chartData as any"
          :options="chartOptions as any"
          :plugins="chartPlugins"
        />
      </div>

      <div class="overview-selected-bar">
        <p class="overview-selected-day" aria-live="polite">
          <span class="overview-selected-dot" :data-metric="metric" aria-hidden="true" />
          <span>{{ focusedBucket?.key ?? endDate }}</span>
          <strong>{{ focusedBucket ? formatBucketPrecise(focusedBucket) : preciseMoney(0) }}</strong>
          <span v-if="metric === 'cost' && focusedBucket && (focusedBucket.cost ?? 0) > (focusedBucket.actualCost ?? 0)" class="overview-selected-saved">
            {{ t('dashboard.overview.savedShort', { amount: overviewMoney((focusedBucket.cost ?? 0) - (focusedBucket.actualCost ?? 0)) }) }}
          </span>
          <span v-if="periodTotalCost === 0 && metric === 'cost'">{{ t('dashboard.overview.noSpending') }}</span>
        </p>
        <router-link
          v-if="focusedBucket && !focusedBucket.future"
          :to="drilldownLocation(focusedBucket)"
          class="overview-link overview-selected-drill"
        >
          {{ t('dashboard.overview.viewDayUsage') }}
          <Icon name="chevronRight" size="sm" />
        </router-link>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  BarController,
  LineController,
  BarElement,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Tooltip,
  Legend,
  Filler,
  type Chart,
  type ChartEvent,
  type ActiveElement,
} from 'chart.js'
import { Bar, Line, type ChartComponentRef } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import CountUp from '@/components/common/CountUp.vue'
import { CHART_FONT_FAMILY, useChartTheme } from '@/composables/useChartTheme'
import {
  chartAnimationConfig,
  createBarCraftPlugin,
  createCrosshairPlugin,
  createGlassTooltipHandler,
  createMeanLinePlugin,
  createPeakCalloutPlugin,
  formatAdaptiveCurrency,
  hexToRgba,
  hideGlassTooltip,
  type GlassTooltipRow,
} from '@/utils/chartPlugins'
import type { TrendDataPoint } from '@/types'
import {
  buildTrendBuckets,
  calculateCacheEfficiency,
  formatCompactTokens,
  overviewMoney,
  preciseMoney,
  splitMetricUnit,
  type DashboardTrendBucket,
} from '@/utils/dashboardOverview'

ChartJS.register(
  BarController,
  LineController,
  BarElement,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Tooltip,
  Legend,
  Filler
)

export type TrendMetric = 'cost' | 'tokens' | 'requests' | 'cache'

const props = withDefaults(defineProps<{
  startDate: string
  endDate: string
  today: string
  granularity: 'day' | 'hour'
  trend: TrendDataPoint[]
  loading: boolean
  error: boolean
  apiKeyId?: number | null
  groupId?: number | null
}>(), {
  apiKeyId: null,
  groupId: null,
})

defineEmits<{ retry: [] }>()

const { t } = useI18n()
const router = useRouter()
const { tokens } = useChartTheme()

const metric = ref<TrendMetric>('cost')
const chartStyle = ref<'bar' | 'area'>('bar')
const canvasWrapRef = ref<HTMLElement | null>(null)
const barChartRef = ref<ChartComponentRef | null>(null)
const lineChartRef = ref<ChartComponentRef | null>(null)
const focusedIndex = ref<number>(0)
const hoveredIndex = ref<number | null>(null)
const showStandardOverlay = ref(true)
const showMeanOverlay = ref(true)
const skipNextAnim = ref(false)
const isWideCanvas = ref(typeof window !== 'undefined' ? window.innerWidth >= 640 : true)

function syncCanvasWidth() {
  if (typeof window !== 'undefined') {
    isWideCanvas.value = window.innerWidth >= 640
  }
}

onMounted(() => {
  syncCanvasWidth()
  window.addEventListener('resize', syncCanvasWidth, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncCanvasWidth)
})

function clearTooltip() {
  hideGlassTooltip(canvasWrapRef.value)
}

function onCanvasLeave() {
  hoveredIndex.value = null
  clearTooltip()
}

watch(
  () => [metric.value, chartStyle.value, props.startDate, props.endDate, props.granularity],
  () => {
    hoveredIndex.value = null
    clearTooltip()
  }
)

watch(hoveredIndex, () => {
  const chart = (useBarComponent.value ? barChartRef.value?.chart : lineChartRef.value?.chart) as Chart | undefined
  if (chart && typeof chart.update === 'function') {
    chart.update('none')
  }
})

// Instantaneous zero-flicker theme repaint when toggling Dark <-> Light
watch(
  () => tokens.value.isDark,
  () => {
    skipNextAnim.value = true
    void nextTick(() => {
      const chart = (useBarComponent.value ? barChartRef.value?.chart : lineChartRef.value?.chart) as Chart | undefined
      if (chart && typeof chart.update === 'function') {
        chart.update('none')
      }
      if (typeof requestAnimationFrame === 'function') {
        requestAnimationFrame(() => {
          skipNextAnim.value = false
        })
      } else {
        skipNextAnim.value = false
      }
    })
  },
  { flush: 'sync' }
)

const metricTabs = computed(() => [
  { id: 'cost' as const, label: t('dashboard.overview.metricCost') },
  { id: 'tokens' as const, label: t('dashboard.overview.metricTokens') },
  { id: 'requests' as const, label: t('dashboard.overview.metricRequests') },
  { id: 'cache' as const, label: t('dashboard.overview.metricCacheRate') },
])

const activeMetricIndex = computed(() =>
  Math.max(0, metricTabs.value.findIndex((tab) => tab.id === metric.value))
)

const buckets = computed<DashboardTrendBucket[]>(() =>
  buildTrendBuckets(props.trend, props.startDate, props.endDate, props.granularity, props.today)
)

const activeBuckets = computed(() => buckets.value.filter((b) => !b.future))

watch(
  () => [props.startDate, props.endDate, props.granularity, buckets.value.length],
  () => {
    const lastActive = buckets.value.reduce((acc, b, idx) => (!b.future ? idx : acc), 0)
    focusedIndex.value = lastActive
  },
  { immediate: true }
)

const focusedBucket = computed<DashboardTrendBucket | null>(() => {
  const list = buckets.value
  if (!list.length) return null
  return list[Math.max(0, Math.min(list.length - 1, focusedIndex.value))] ?? null
})

const hasToday = computed(() => buckets.value.some((b) => b.date === props.today))

const periodTotalCost = computed(() =>
  activeBuckets.value.reduce((sum, b) => sum + (b.actualCost ?? 0), 0)
)
const periodTotalStandard = computed(() =>
  activeBuckets.value.reduce((sum, b) => sum + (b.cost ?? 0), 0)
)
const periodTotalTokens = computed(() =>
  activeBuckets.value.reduce((sum, b) => sum + (b.totalTokens ?? 0), 0)
)
const periodTotalRequests = computed(() =>
  activeBuckets.value.reduce((sum, b) => sum + (b.requests ?? 0), 0)
)
const periodCacheEfficiency = computed(() => {
  const input = activeBuckets.value.reduce((s, b) => s + (b.inputTokens ?? 0), 0)
  const create = activeBuckets.value.reduce((s, b) => s + (b.cacheCreationTokens ?? 0), 0)
  const read = activeBuckets.value.reduce((s, b) => s + (b.cacheReadTokens ?? 0), 0)
  return calculateCacheEfficiency(input, create, read)
})
const periodCacheRate = computed(() =>
  periodCacheEfficiency.value.promptTokens > 0 ? periodCacheEfficiency.value.rate : null
)

const bucketCount = computed(() => Math.max(1, activeBuckets.value.length))
const meanActualCost = computed(() => periodTotalCost.value / bucketCount.value)
const meanRequests = computed(() => periodTotalRequests.value / bucketCount.value)
const hasStandardDifference = computed(() => periodTotalStandard.value > periodTotalCost.value * 1.001)

interface LegendItem {
  id: string
  label: string
  color: string
  kind: 'solid' | 'dashed' | 'dotted'
  active: boolean
  toggleable: boolean
}

const legendItems = computed<LegendItem[]>(() => {
  const c = tokens.value
  if (metric.value === 'cost') {
    const items: LegendItem[] = [
      { id: 'actual', label: t('dashboard.overview.gabuguBilling'), color: c.primary, kind: 'solid', active: true, toggleable: false },
    ]
    if (hasStandardDifference.value) {
      items.push({
        id: 'standard',
        label: t('dashboard.overview.standardBilling'),
        color: c.secondary,
        kind: 'dashed',
        active: showStandardOverlay.value,
        toggleable: true,
      })
    }
    if (meanActualCost.value > 0 && activeBuckets.value.length > 1) {
      items.push({
        id: 'mean',
        label: t('dashboard.overview.periodAvg'),
        color: c.tick,
        kind: 'dotted',
        active: showMeanOverlay.value,
        toggleable: true,
      })
    }
    return items
  }
  if (metric.value === 'tokens') {
    return [
      { id: 'input', label: t('dashboard.overview.tokenInput'), color: c.primary, kind: 'solid', active: true, toggleable: false },
      { id: 'output', label: t('dashboard.overview.tokenOutput'), color: c.emerald, kind: 'solid', active: true, toggleable: false },
      { id: 'cacheRead', label: t('dashboard.overview.tokenCacheRead'), color: c.violet, kind: 'solid', active: true, toggleable: false },
      { id: 'cacheCreate', label: t('dashboard.overview.tokenCacheCreate'), color: c.amber, kind: 'solid', active: true, toggleable: false },
    ]
  }
  return []
})

function toggleSeries(id: string) {
  if (id === 'standard') showStandardOverlay.value = !showStandardOverlay.value
  else if (id === 'mean') showMeanOverlay.value = !showMeanOverlay.value
}

function bucketNumericValue(b: DashboardTrendBucket): number {
  switch (metric.value) {
    case 'cost': return b.actualCost ?? 0
    case 'tokens': return b.totalTokens ?? 0
    case 'requests': return b.requests ?? 0
    case 'cache': return b.cacheHitRate ?? 0
  }
}

const peakBucket = computed<DashboardTrendBucket | null>(() => {
  let best: DashboardTrendBucket | null = null
  let bestVal = 0
  for (const b of activeBuckets.value) {
    const val = bucketNumericValue(b)
    if (val > bestVal) {
      bestVal = val
      best = b
    }
  }
  return best
})

const maxCostInSeries = computed(() =>
  Math.max(
    0,
    ...activeBuckets.value.map((b) => Math.max(b.actualCost ?? 0, showStandardOverlay.value ? (b.cost ?? 0) : 0))
  )
)

const panelTitle = computed(() => {
  if (props.granularity === 'hour') return t('dashboard.overview.hourlyTrend')
  switch (metric.value) {
    case 'tokens': return t('dashboard.overview.tokenTrend')
    case 'requests': return t('dashboard.overview.requestsTrend')
    case 'cache': return t('dashboard.overview.cacheTrend')
    default: return t('dashboard.overview.dailySpending')
  }
})

const primaryMetricLabel = computed(() => {
  switch (metric.value) {
    case 'tokens': return t('dashboard.overview.periodTotalTokens')
    case 'requests': return t('dashboard.overview.periodTotalRequests')
    case 'cache': return t('dashboard.overview.periodAvgCacheRate')
    default: return t('dashboard.overview.periodTotal')
  }
})

const primaryMetricFormatted = computed(() => {
  switch (metric.value) {
    case 'tokens': return formatCompactTokens(periodTotalTokens.value)
    case 'requests': return periodTotalRequests.value.toLocaleString()
    case 'cache': return periodCacheRate.value !== null ? `${periodCacheRate.value.toFixed(1)}%` : '—'
    default: return overviewMoney(periodTotalCost.value)
  }
})

const primaryMetricTitle = computed(() => {
  if (metric.value === 'cost') return preciseMoney(periodTotalCost.value)
  if (metric.value === 'tokens') return periodTotalTokens.value.toLocaleString()
  return primaryMetricFormatted.value
})

const avgFormatted = computed(() => {
  switch (metric.value) {
    case 'tokens': return formatCompactTokens(periodTotalTokens.value / bucketCount.value)
    case 'requests': return meanRequests.value.toFixed(1)
    case 'cache': return periodCacheRate.value !== null ? `${periodCacheRate.value.toFixed(1)}%` : '—'
    default: return overviewMoney(meanActualCost.value)
  }
})

function formatBucketPrimary(b: DashboardTrendBucket): string {
  switch (metric.value) {
    case 'tokens': return formatCompactTokens(b.totalTokens ?? 0)
    case 'requests': return (b.requests ?? 0).toLocaleString()
    case 'cache': return b.cacheHitRate !== null ? `${b.cacheHitRate.toFixed(1)}%` : '—'
    default: return overviewMoney(b.actualCost ?? 0)
  }
}

function formatBucketPrecise(b: DashboardTrendBucket): string {
  if (b.future) return '—'
  switch (metric.value) {
    case 'tokens': return `${(b.totalTokens ?? 0).toLocaleString()} Tokens`
    case 'requests': return `${(b.requests ?? 0).toLocaleString()} ${t('dashboard.overview.requestsUnit')}`
    case 'cache': return b.cacheHitRate !== null ? `${b.cacheHitRate.toFixed(1)}%` : '—'
    default: return preciseMoney(b.actualCost ?? 0)
  }
}

const useBarComponent = computed(() => {
  if (metric.value === 'cache') return false
  if (metric.value === 'tokens') return true
  return chartStyle.value === 'bar'
})

const todayBucketIndex = computed<number | null>(() => {
  if (props.granularity !== 'day') return null
  const idx = buckets.value.findIndex((b) => b.date === props.today)
  return idx >= 0 ? idx : null
})

const hasMeanLine = computed(() => {
  if (activeBuckets.value.length <= 1) return false
  if (metric.value === 'cost') return showMeanOverlay.value && meanActualCost.value > 0
  if (metric.value === 'requests') return meanRequests.value > 0
  return false
})

const peakBucketIndex = computed<number | null>(() => {
  const peak = peakBucket.value
  if (!peak) return null
  const idx = buckets.value.findIndex((b) => b.key === peak.key)
  return idx >= 0 ? idx : null
})

const chartPlugins = [
  createCrosshairPlugin(
    () => tokens.value.crosshair,
    () => (tokens.value.isDark ? 'rgba(124, 162, 255, 0.065)' : 'rgba(45, 88, 144, 0.055)')
  ),
  createBarCraftPlugin(() => ({
    enabled: useBarComponent.value && metric.value !== 'tokens',
    todayIndex: metric.value === 'cost' ? todayBucketIndex.value : null,
    hoveredIndex: hoveredIndex.value,
    specularColor: tokens.value.isDark ? 'rgba(216, 232, 255, 0.58)' : 'rgba(255, 255, 255, 0.65)',
    hatchColor: tokens.value.isDark ? 'rgba(255, 255, 255, 0.16)' : 'rgba(255, 255, 255, 0.28)',
  })),
  createPeakCalloutPlugin(() => {
    const c = tokens.value
    const peak = peakBucket.value
    const enabled =
      isWideCanvas.value &&
      activeBuckets.value.length >= 2 &&
      buckets.value.length <= 14 &&
      metric.value !== 'tokens' &&
      !(metric.value === 'cost' && hasStandardDifference.value && showStandardOverlay.value) &&
      peak !== null
    const dotColor =
      metric.value === 'requests' ? c.emerald : metric.value === 'cache' ? c.violet : c.primary
    return {
      enabled,
      peakIndex: peakBucketIndex.value,
      label: peak ? formatBucketPrimary(peak) : '',
      hoveredIndex: hoveredIndex.value,
      textColor: c.text,
      dotColor,
      fontFamily: CHART_FONT_FAMILY,
    }
  }),
  createMeanLinePlugin(() => {
    const c = tokens.value
    if (!hasMeanLine.value) return null
    if (metric.value === 'cost' && showMeanOverlay.value && meanActualCost.value > 0) {
      const prefix = t('dashboard.overview.periodAvg')
      const valueText = overviewMoney(meanActualCost.value)
      return {
        visible: true,
        value: meanActualCost.value,
        label: `${prefix} ${valueText}`,
        prefix,
        valueText,
        lineColor: hexToRgba(c.tick, c.isDark ? 0.36 : 0.42),
        pillBg: 'transparent',
        pillBorder: 'transparent',
        pillText: c.text,
        mutedText: c.tick,
        fontFamily: CHART_FONT_FAMILY,
      }
    }
    if (metric.value === 'requests' && meanRequests.value > 0) {
      const prefix = t('dashboard.overview.periodAvg')
      const valueText = meanRequests.value.toFixed(0)
      return {
        visible: true,
        value: meanRequests.value,
        label: `${prefix} ${valueText}`,
        prefix,
        valueText,
        lineColor: hexToRgba(c.emerald, 0.38),
        pillBg: 'transparent',
        pillBorder: 'transparent',
        pillText: c.text,
        mutedText: c.tick,
        fontFamily: CHART_FONT_FAMILY,
      }
    }
    return null
  }),
]

function makeVerticalGradient(color: string, topAlpha: number, bottomAlpha: number) {
  return (context: { chart: Chart }) => {
    const { ctx, chartArea } = context.chart
    if (!chartArea) return hexToRgba(color, topAlpha)
    const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
    gradient.addColorStop(0, hexToRgba(color, topAlpha))
    gradient.addColorStop(1, hexToRgba(color, bottomAlpha))
    return gradient
  }
}

function makeBarGradientScriptable(color: string, highlightToday: boolean) {
  return (context: { chart: Chart; dataIndex?: number }) => {
    const { ctx, chartArea } = context.chart
    const idx = context.dataIndex ?? 0
    const bucket = buckets.value[idx]
    const isToday = highlightToday && props.granularity === 'day' && bucket?.date === props.today
    const activeHover = hoveredIndex.value
    const isHovered = activeHover === idx
    const isDimmed = activeHover !== null && !isHovered
    const isDark = tokens.value.isDark

    let topAlpha = isToday ? 0.96 : isDark ? 0.86 : 0.84
    let bottomAlpha = isToday ? 0.46 : isDark ? 0.22 : 0.32
    if (isHovered) {
      topAlpha = 1
      bottomAlpha = isDark ? 0.54 : 0.62
    } else if (isDimmed) {
      topAlpha *= 0.36
      bottomAlpha *= 0.36
    }

    if (!chartArea) return hexToRgba(color, topAlpha)
    const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
    gradient.addColorStop(0, hexToRgba(color, topAlpha))
    gradient.addColorStop(1, hexToRgba(color, bottomAlpha))
    return gradient
  }
}

const chartData = computed(() => {
  const c = tokens.value
  const list = buckets.value
  const labels = list.map((b) => b.label)

  if (metric.value === 'cost') {
    const actualData = list.map((b) => (b.future ? null : b.actualCost))
    const standardData = list.map((b) => (b.future ? null : b.cost))
    const showStandard = hasStandardDifference.value && showStandardOverlay.value

    if (chartStyle.value === 'bar') {
      const datasets: any[] = [
        {
          type: 'bar',
          label: t('dashboard.overview.gabuguBilling'),
          data: actualData,
          backgroundColor: makeBarGradientScriptable(c.primary, true),
          borderColor: (ctx: { dataIndex?: number }) => {
            const idx = ctx.dataIndex ?? 0
            const isDim = hoveredIndex.value !== null && hoveredIndex.value !== idx
            return hexToRgba(c.primary, isDim ? 0.28 : 0.95)
          },
          borderWidth: { top: 1.5, left: 0, right: 0, bottom: 0 },
          borderRadius: { topLeft: 8, topRight: 8, bottomLeft: 2, bottomRight: 2 },
          borderSkipped: 'bottom',
          categoryPercentage: 0.74,
          barPercentage: 0.76,
          minBarLength: 5,
          maxBarThickness: 48,
          order: 2,
        },
      ]
      if (showStandard) {
        datasets.push({
          type: 'line',
          label: t('dashboard.overview.standardBilling'),
          data: standardData,
          borderColor: hexToRgba(c.secondary, c.isDark ? 0.64 : 0.68),
          backgroundColor: makeVerticalGradient(c.secondary, c.isDark ? 0.065 : 0.05, 0.0),
          fill: true,
          borderDash: [4, 4],
          borderWidth: 1.5,
          cubicInterpolationMode: 'monotone',
          tension: 0.35,
          pointRadius: list.length <= 10 ? 2.5 : 0,
          pointBackgroundColor: c.surface,
          pointBorderColor: hexToRgba(c.secondary, 0.78),
          pointBorderWidth: 1.5,
          pointHoverRadius: 4.5,
          pointHoverBackgroundColor: c.secondary,
          order: 1,
        })
      }
      return { labels, datasets }
    }

    const areaDatasets: any[] = [
      {
        label: t('dashboard.overview.gabuguBilling'),
        data: actualData,
        borderColor: c.primary,
        backgroundColor: makeVerticalGradient(c.primary, c.isDark ? 0.36 : 0.26, 0.02),
        borderWidth: 2.5,
        fill: true,
        cubicInterpolationMode: 'monotone',
        tension: 0.35,
        pointRadius: list.length <= 12 ? 3.5 : 0,
        pointBackgroundColor: c.surface,
        pointBorderColor: c.primary,
        pointBorderWidth: 2,
        pointHoverRadius: 5.5,
        pointHoverBackgroundColor: c.primary,
        order: 2,
      },
    ]
    if (showStandard) {
      areaDatasets.push({
        label: t('dashboard.overview.standardBilling'),
        data: standardData,
        borderColor: hexToRgba(c.secondary, 0.88),
        backgroundColor: hexToRgba(c.secondary, c.isDark ? 0.14 : 0.1),
        borderDash: [5, 4],
        borderWidth: 2,
        fill: '-1',
        cubicInterpolationMode: 'monotone',
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 4.5,
        pointHoverBackgroundColor: c.secondary,
        order: 1,
      })
    }
    return { labels, datasets: areaDatasets }
  }

  if (metric.value === 'tokens') {
    const makeStackColor = (hex: string, restAlphaDark = 0.78, restAlphaLight = 0.82) => (ctx: { dataIndex?: number }) => {
      const idx = ctx.dataIndex ?? 0
      const activeHover = hoveredIndex.value
      const baseAlpha = c.isDark ? restAlphaDark : restAlphaLight
      if (activeHover === null) return hexToRgba(hex, baseAlpha)
      return hexToRgba(hex, activeHover === idx ? 0.96 : 0.26)
    }
    return {
      labels,
      datasets: [
        {
          label: t('dashboard.overview.tokenInput'),
          data: list.map((b) => (b.future ? null : b.inputTokens)),
          backgroundColor: makeStackColor(c.primary, 0.82, 0.84),
          borderColor: c.surface,
          borderWidth: 1.5,
          borderRadius: 4,
          stack: 'tokens',
          categoryPercentage: 0.72,
          barPercentage: 0.76,
          maxBarThickness: 44,
        },
        {
          label: t('dashboard.overview.tokenOutput'),
          data: list.map((b) => (b.future ? null : b.outputTokens)),
          backgroundColor: makeStackColor(c.emerald, 0.78, 0.8),
          borderColor: c.surface,
          borderWidth: 1.5,
          borderRadius: 4,
          stack: 'tokens',
          categoryPercentage: 0.72,
          barPercentage: 0.76,
          maxBarThickness: 44,
        },
        {
          label: t('dashboard.overview.tokenCacheRead'),
          data: list.map((b) => (b.future ? null : b.cacheReadTokens)),
          backgroundColor: makeStackColor(c.violet, 0.72, 0.76),
          borderColor: c.surface,
          borderWidth: 1.5,
          borderRadius: 4,
          stack: 'tokens',
          categoryPercentage: 0.72,
          barPercentage: 0.76,
          maxBarThickness: 44,
        },
        {
          label: t('dashboard.overview.tokenCacheCreate'),
          data: list.map((b) => (b.future ? null : b.cacheCreationTokens)),
          backgroundColor: makeStackColor(c.amber, 0.8, 0.82),
          borderColor: c.surface,
          borderWidth: 1.5,
          borderRadius: { topLeft: 7, topRight: 7, bottomLeft: 3, bottomRight: 3 },
          stack: 'tokens',
          categoryPercentage: 0.72,
          barPercentage: 0.76,
          maxBarThickness: 44,
        },
      ],
    }
  }

  if (metric.value === 'requests') {
    const reqData = list.map((b) => (b.future ? null : b.requests))
    if (chartStyle.value === 'bar') {
      return {
        labels,
        datasets: [
          {
            label: t('dashboard.overview.metricRequests'),
            data: reqData,
            backgroundColor: makeBarGradientScriptable(c.emerald, false),
            borderColor: hexToRgba(c.emerald, 0.92),
            borderWidth: { top: 1.5, left: 0, right: 0, bottom: 0 },
            borderRadius: { topLeft: 8, topRight: 8, bottomLeft: 2, bottomRight: 2 },
            borderSkipped: 'bottom',
            categoryPercentage: 0.74,
            barPercentage: 0.76,
            minBarLength: 5,
            maxBarThickness: 48,
          },
        ],
      }
    }
    return {
      labels,
      datasets: [
        {
          label: t('dashboard.overview.metricRequests'),
          data: reqData,
          borderColor: c.emerald,
          backgroundColor: makeVerticalGradient(c.emerald, c.isDark ? 0.36 : 0.24, 0.02),
          borderWidth: 2.5,
          fill: true,
          cubicInterpolationMode: 'monotone',
          tension: 0.35,
          pointRadius: list.length <= 12 ? 3.5 : 0,
          pointBackgroundColor: c.surface,
          pointBorderColor: c.emerald,
          pointBorderWidth: 2,
          pointHoverRadius: 5.5,
        },
      ],
    }
  }

  // Cache hit rate (0-100%)
  return {
    labels,
    datasets: [
      {
        label: t('dashboard.overview.metricCacheRate'),
        data: list.map((b) => (b.future ? null : b.cacheHitRate)),
        borderColor: c.violet,
        backgroundColor: makeVerticalGradient(c.violet, c.isDark ? 0.34 : 0.22, 0.02),
        borderWidth: 2.5,
        fill: true,
        cubicInterpolationMode: 'monotone',
        tension: 0.35,
        spanGaps: true,
        pointRadius: list.length <= 12 ? 3.5 : 0,
        pointBackgroundColor: c.surface,
        pointBorderColor: c.violet,
        pointBorderWidth: 2,
        pointHoverRadius: 5.5,
        pointHoverBackgroundColor: c.violet,
      },
    ],
  }
})

const glassTooltip = createGlassTooltipHandler(({ tooltip }) => {
  const index = tooltip.dataPoints?.[0]?.dataIndex
  if (typeof index !== 'number') return null
  const b = buckets.value[index]
  if (!b || b.future) return null
  const c = tokens.value

  if (metric.value === 'cost') {
    const actual = b.actualCost ?? 0
    const std = b.cost ?? 0
    const saved = std - actual
    const rows: GlassTooltipRow[] = [
      { label: t('dashboard.overview.gabuguBilling'), value: preciseMoney(actual), color: c.primary },
    ]
    if (std > 0) {
      rows.push({ label: t('dashboard.overview.standardBilling'), value: preciseMoney(std), color: c.secondary, muted: true })
    }
    if (saved > 1e-6) {
      rows.push({ label: t('dashboard.overview.savingsShort'), value: `-${overviewMoney(saved)}`, color: c.emerald })
    }
    rows.push({
      label: t('dashboard.overview.requestsAndTokens'),
      value: `${(b.requests ?? 0).toLocaleString()} · ${formatCompactTokens(b.totalTokens ?? 0)}`,
      muted: true,
    })
    let meanFooter: string | undefined
    let footerTone: 'up' | 'down' | 'neutral' | undefined
    if (activeBuckets.value.length > 1 && meanActualCost.value > 0) {
      const diffPct = ((actual - meanActualCost.value) / meanActualCost.value) * 100
      const sign = diffPct > 0.05 ? '+' : ''
      meanFooter = `${t('dashboard.overview.periodAvg')} ${overviewMoney(meanActualCost.value)} (${sign}${diffPct.toFixed(1)}%)`
      footerTone = diffPct > 5 ? 'up' : diffPct < -5 ? 'down' : 'neutral'
    }
    return {
      title: b.key,
      subtitle: b.cacheHitRate !== null ? `${t('dashboard.overview.cacheHitShort')} ${b.cacheHitRate.toFixed(1)}%` : undefined,
      rows,
      footer: meanFooter,
      footerTone,
      hint: t('dashboard.overview.clickToDrillUsage'),
    }
  }

  if (metric.value === 'tokens') {
    return {
      title: b.key,
      subtitle: `${formatCompactTokens(b.totalTokens ?? 0)} Tokens`,
      rows: [
        { label: t('dashboard.overview.tokenInput'), value: formatCompactTokens(b.inputTokens ?? 0), color: c.primary },
        { label: t('dashboard.overview.tokenOutput'), value: formatCompactTokens(b.outputTokens ?? 0), color: c.emerald },
        { label: t('dashboard.overview.tokenCacheRead'), value: formatCompactTokens(b.cacheReadTokens ?? 0), color: c.violet },
        { label: t('dashboard.overview.tokenCacheCreate'), value: formatCompactTokens(b.cacheCreationTokens ?? 0), color: c.amber },
      ],
      hint: t('dashboard.overview.clickToDrillUsage'),
    }
  }

  if (metric.value === 'requests') {
    const reqs = b.requests ?? 0
    const avgCost = reqs > 0 ? (b.actualCost ?? 0) / reqs : 0
    const avgTok = reqs > 0 ? (b.totalTokens ?? 0) / reqs : 0
    let reqFooter: string | undefined
    let reqTone: 'up' | 'down' | 'neutral' | undefined
    if (activeBuckets.value.length > 1 && meanRequests.value > 0) {
      const diffPct = ((reqs - meanRequests.value) / meanRequests.value) * 100
      const sign = diffPct > 0.05 ? '+' : ''
      reqFooter = `${t('dashboard.overview.periodAvg')} ${meanRequests.value.toFixed(0)} (${sign}${diffPct.toFixed(1)}%)`
      reqTone = diffPct > 5 ? 'up' : diffPct < -5 ? 'down' : 'neutral'
    }
    return {
      title: b.key,
      rows: [
        { label: t('dashboard.overview.metricRequests'), value: reqs.toLocaleString(), color: c.emerald },
        { label: t('dashboard.overview.avgCostPerReq'), value: preciseMoney(avgCost), muted: true },
        { label: t('dashboard.overview.avgTokensPerReq'), value: formatCompactTokens(avgTok), muted: true },
      ],
      footer: reqFooter,
      footerTone: reqTone,
      hint: t('dashboard.overview.clickToDrillUsage'),
    }
  }

  return {
    title: b.key,
    rows: [
      {
        label: t('dashboard.overview.metricCacheRate'),
        value: b.cacheHitRate !== null ? `${b.cacheHitRate.toFixed(1)}%` : '—',
        color: c.violet,
      },
      { label: t('dashboard.overview.tokenCacheRead'), value: formatCompactTokens(b.cacheReadTokens ?? 0), muted: true },
      { label: t('dashboard.overview.tokenInput'), value: formatCompactTokens(b.inputTokens ?? 0), muted: true },
    ],
    hint: t('dashboard.overview.clickToDrillUsage'),
  }
})

function drilldownLocation(b: DashboardTrendBucket) {
  const query: Record<string, string> = {
    start_date: b.date,
    end_date: b.date,
  }
  if (props.apiKeyId) query.api_key_id = String(props.apiKeyId)
  if (props.groupId) query.group_id = String(props.groupId)
  return { path: '/usage', query }
}

function handleChartClick(_event: ChartEvent, elements: ActiveElement[]) {
  const idx = elements[0]?.index
  if (typeof idx !== 'number') return
  const bucket = buckets.value[idx]
  if (!bucket || bucket.future) return
  focusedIndex.value = idx
  void router?.push(drilldownLocation(bucket))
}

function handleChartHover(event: ChartEvent, elements: ActiveElement[], chart: Chart) {
  const idx = elements[0]?.index
  const valid = typeof idx === 'number' && Boolean(buckets.value[idx]) && !buckets.value[idx].future
  if (valid) {
    focusedIndex.value = idx
    if (hoveredIndex.value !== idx) hoveredIndex.value = idx
  } else if (hoveredIndex.value !== null) {
    hoveredIndex.value = null
  }
  const target = (event.native?.target ?? chart?.canvas) as HTMLElement | null | undefined
  if (target) {
    target.style.cursor = valid ? 'pointer' : 'default'
  }
}

const chartOptions = computed(() => {
  const c = tokens.value
  const stacked = metric.value === 'tokens'
  return {
    responsive: true,
    maintainAspectRatio: false,
    layout: {
      padding: { top: 12, right: hasMeanLine.value && isWideCanvas.value ? 104 : 8, left: 0, bottom: 0 },
    },
    animation: skipNextAnim.value ? false : chartAnimationConfig(),
    interaction: {
      mode: 'index' as const,
      intersect: false,
    },
    onClick: handleChartClick,
    onHover: handleChartHover,
    plugins: {
      legend: { display: false },
      tooltip: {
        enabled: false,
        external: glassTooltip,
      },
    },
    scales: {
      x: {
        stacked,
        border: { display: false },
        grid: { display: false },
        ticks: {
          color: c.tick,
          padding: 8,
          maxRotation: 0,
          autoSkip: true,
          maxTicksLimit: props.granularity === 'hour' ? 8 : 10,
          font: { family: CHART_FONT_FAMILY, size: 13, weight: 500 },
        },
      },
      y: {
        stacked,
        beginAtZero: true,
        grace: metric.value === 'cache' ? 0 : '10%',
        ...(metric.value === 'cache' ? { max: 100 } : {}),
        border: { display: false, dash: [2, 4] },
        grid: {
          color: c.isDark ? 'rgba(176, 204, 255, 0.055)' : 'rgba(168, 146, 118, 0.18)',
          drawTicks: false,
        },
        ticks: {
          color: c.tick,
          padding: 10,
          maxTicksLimit: 5,
          font: { family: CHART_FONT_FAMILY, size: 13, weight: 500 },
          callback: (value: string | number) => {
            const num = Number(value)
            if (metric.value === 'cost') return formatAdaptiveCurrency(num, maxCostInSeries.value)
            if (metric.value === 'tokens') return formatCompactTokens(num)
            if (metric.value === 'cache') return `${num}%`
            return num.toLocaleString()
          },
        },
      },
    },
  }
})

function syncActiveChartElement(index: number) {
  const chart = (useBarComponent.value ? barChartRef.value?.chart : lineChartRef.value?.chart) as Chart | undefined
  if (!chart || typeof chart.setActiveElements !== 'function') return
  chart.setActiveElements([{ datasetIndex: 0, index }])
  chart.update('none')
}

function onCanvasKeydown(event: KeyboardEvent) {
  const maxActive = buckets.value.reduce((acc, b, idx) => (!b.future ? idx : acc), 0)
  if (event.key === 'ArrowRight') {
    event.preventDefault()
    focusedIndex.value = Math.min(maxActive, focusedIndex.value + 1)
    syncActiveChartElement(focusedIndex.value)
  } else if (event.key === 'ArrowLeft') {
    event.preventDefault()
    focusedIndex.value = Math.max(0, focusedIndex.value - 1)
    syncActiveChartElement(focusedIndex.value)
  } else if (event.key === 'Home') {
    event.preventDefault()
    focusedIndex.value = 0
    syncActiveChartElement(0)
  } else if (event.key === 'End') {
    event.preventDefault()
    focusedIndex.value = maxActive
    syncActiveChartElement(maxActive)
  } else if (event.key === 'Enter' && focusedBucket.value && !focusedBucket.value.future) {
    event.preventDefault()
    void router?.push(drilldownLocation(focusedBucket.value))
  }
}
</script>
