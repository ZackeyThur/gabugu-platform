<template>
  <section
    class="overview-ranking-panel"
    aria-labelledby="ranking-heading"
    :aria-busy="loading"
  >
    <div class="overview-panel-heading overview-breakdown-head">
      <div
        class="overview-breakdown-tabs g-segmented"
        role="tablist"
        :aria-label="t('dashboard.overview.breakdownDimension')"
        :style="{ '--seg-count': dimensionTabs.length, '--seg-index': activeDimensionIndex }"
      >
        <span class="g-segmented-thumb" aria-hidden="true" />
        <h3 id="ranking-heading" class="sr-only">{{ t('dashboard.overview.modelRanking') }}</h3>
        <button
          v-for="dim in dimensionTabs"
          :key="dim.id"
          type="button"
          role="tab"
          class="overview-metric-tab"
          :class="{ 'is-active': dimension === dim.id }"
          :aria-selected="dimension === dim.id"
          @click="dimension = dim.id"
        >
          {{ dim.label }}
        </button>
      </div>

      <div
        class="overview-breakdown-metrics g-segmented"
        role="group"
        :aria-label="t('dashboard.overview.breakdownMetric')"
        :style="{ '--seg-count': metricOptions.length, '--seg-index': activeMetricIndex }"
      >
        <span class="g-segmented-thumb" aria-hidden="true" />
        <button
          v-for="m in metricOptions"
          :key="m.id"
          type="button"
          class="overview-mini-pill"
          :class="{ 'is-active': metric === m.id }"
          :aria-pressed="metric === m.id"
          @click="metric = m.id"
        >
          {{ m.label }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="overview-chart-loading" role="status">
      <div class="overview-skeleton" />
      {{ t('common.loading') }}
    </div>
    <div v-else-if="error" class="overview-empty" role="alert">
      <p>{{ t('dashboard.overview.modelsError') }}</p>
      <button type="button" class="overview-retry-btn" @click="$emit('retry')">
        <Icon name="refresh" size="xs" />
        {{ t('common.retry') }}
      </button>
    </div>
    <div v-else-if="allItems.length === 0" class="overview-empty">
      <p>{{ emptyText }}</p>
    </div>
    <template v-else>
      <div class="overview-ring-stage">
        <div class="overview-ring-canvas-wrap">
          <Doughnut ref="ringChartRef" :data="ringChartData" :options="ringChartOptions as any" />
          <div class="overview-ring-center" aria-live="polite">
            <span class="overview-ring-center-label" :title="centerReadout.label">{{ centerReadout.label }}</span>
            <strong class="overview-ring-center-value" :title="centerReadout.title">
              <template v-if="centerValueParts && !centerValueParts.suffix">
                <span>{{ centerValueParts.major }}</span><span class="overview-cents">{{ centerValueParts.cents }}</span>
              </template>
              <template v-else-if="centerValueParts && centerValueParts.suffix">
                <span>{{ centerValueParts.major }}{{ centerValueParts.cents }}</span><span class="overview-unit-suffix">{{ centerValueParts.suffix }}</span>
              </template>
              <template v-else>{{ centerReadout.value }}</template>
            </strong>
            <span class="overview-ring-center-sub">{{ centerReadout.sub }}</span>
          </div>
        </div>
        <div v-if="ringHighlights" class="overview-ring-highlights">
          <div class="overview-ring-hl-item">
            <span class="overview-ring-hl-label">{{ t('dashboard.overview.topContributor') }}</span>
            <strong class="overview-ring-hl-name" :title="ringHighlights.topLabel">
              <span class="overview-ring-hl-dot" :style="{ backgroundColor: ringHighlights.topColor }" aria-hidden="true" />
              <span>{{ ringHighlights.topLabel }}</span>
            </strong>
            <span class="overview-ring-hl-sub">{{ ringHighlights.topShare }} · {{ ringHighlights.topValue }}</span>
          </div>
          <div class="overview-ring-hl-foot">
            <div>
              <span>{{ t('dashboard.overview.topThreeShare') }}</span>
              <strong class="overview-number">{{ ringHighlights.topThreeShare }}</strong>
            </div>
            <div>
              <span>{{ t('dashboard.overview.periodRequestsShort') }}</span>
              <strong class="overview-number">{{ ringHighlights.totalRequests }}</strong>
            </div>
          </div>
        </div>
      </div>

      <ol class="overview-models overview-breakdown-list" @mouseleave="clearHover">
        <li
          v-for="(row, idx) in visibleRows"
          :key="row.key"
          class="overview-model-row overview-breakdown-row"
          :style="{ '--row-accent': row.color }"
          :class="{
            'is-highlighted': hoveredIndex === idx,
            'is-dimmed': hoveredIndex !== null && hoveredIndex !== idx,
            'is-clickable': row.canDrilldown,
            'is-disabled-drill': !row.canDrilldown,
          }"
          :tabindex="row.canDrilldown ? 0 : undefined"
          :role="row.canDrilldown ? 'button' : undefined"
          @mouseenter="highlightSlice(idx)"
          @focusin="highlightSlice(idx)"
          @focusout="clearHover"
          @click="row.canDrilldown && handleRowClick(row)"
          @keydown.enter.prevent="row.canDrilldown && handleRowClick(row)"
          @keydown.space.prevent="row.canDrilldown && handleRowClick(row)"
        >
          <span
            class="overview-model-logo"
            :class="{ 'is-tinted': dimension !== 'model' && !row.other }"
            aria-hidden="true"
          >
            <Icon v-if="row.other" name="cube" size="sm" />
            <ModelIcon v-else-if="dimension === 'model'" :model="row.rawName" size="20px" />
            <Icon v-else-if="dimension === 'group'" name="server" size="sm" />
            <Icon v-else name="key" size="sm" />
          </span>
          <div class="overview-model-data">
            <div class="overview-model-top">
              <span class="overview-model-name-wrap">
                <span class="overview-model-name" :title="row.label">{{ row.label }}</span>
                <span v-if="row.deleted" class="overview-deleted-tag">{{ t('dashboard.overview.deletedKeyTag') }}</span>
              </span>
              <strong :title="row.preciseValue">
                <template v-if="splitFormattedDecimal(row.formattedValue) && !splitFormattedDecimal(row.formattedValue)!.suffix">
                  <span>{{ splitFormattedDecimal(row.formattedValue)!.major }}</span><span class="overview-row-cents">{{ splitFormattedDecimal(row.formattedValue)!.cents }}</span>
                </template>
                <template v-else-if="splitFormattedDecimal(row.formattedValue) && splitFormattedDecimal(row.formattedValue)!.suffix">
                  <span>{{ splitFormattedDecimal(row.formattedValue)!.major }}{{ splitFormattedDecimal(row.formattedValue)!.cents }}</span><span class="overview-unit-suffix">{{ splitFormattedDecimal(row.formattedValue)!.suffix }}</span>
                </template>
                <template v-else>{{ row.formattedValue }}</template>
              </strong>
            </div>
            <div class="overview-model-bottom">
              <div class="overview-model-track" aria-hidden="true">
                <span :style="{ width: Math.max(0, Math.min(100, row.share)) + '%', background: row.color }" />
              </div>
              <span>{{ shareLabel(row.share) }}</span>
            </div>
            <div class="overview-breakdown-meta">
              <span>{{ row.requests.toLocaleString() }} {{ t('dashboard.overview.requestsUnit') }}</span>
              <span>·</span>
              <span>{{ formatCompactTokens(row.tokens) }} Tokens</span>
              <template v-if="row.extraMeta">
                <span>·</span>
                <span>{{ row.extraMeta }}</span>
              </template>
            </div>
          </div>
          <span v-if="row.canDrilldown" class="overview-row-chevron" aria-hidden="true">
            <Icon name="chevronRight" size="xs" />
          </span>
        </li>
      </ol>

      <button
        v-if="allItems.length > 5"
        type="button"
        class="overview-link overview-expand-models"
        :aria-expanded="expanded"
        @click="expanded = !expanded"
      >
        {{ t(expanded ? 'dashboard.overview.collapseModels' : 'dashboard.overview.allItems', { count: allItems.length }) }}
        <Icon :name="expanded ? 'chevronUp' : 'chevronRight'" size="sm" />
      </button>
      <p v-if="metric === 'actual_cost' && allItems.every(item => item.actualCost === 0)" class="overview-muted">
        {{ t('dashboard.overview.zeroCostModels') }}
      </p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, DoughnutController, ArcElement, Tooltip, Legend, type Chart, type ChartEvent, type ActiveElement } from 'chart.js'
import { Doughnut, type ChartComponentRef } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { useChartTheme } from '@/composables/useChartTheme'
import { chartAnimationConfig, hexToRgba } from '@/utils/chartPlugins'
import type { APIKeyStat, GroupStat, ModelStat } from '@/types'
import { calculateCacheEfficiency, formatCompactTokens, overviewMoney, preciseMoney, splitFormattedDecimal } from '@/utils/dashboardOverview'

ChartJS.register(DoughnutController, ArcElement, Tooltip, Legend)

export type BreakdownDimension = 'model' | 'group' | 'api_key'
export type BreakdownMetric = 'actual_cost' | 'total_tokens' | 'requests'

interface NormalizedItem {
  key: string
  id: number | string
  rawName: string
  label: string
  actualCost: number
  standardCost: number
  tokens: number
  requests: number
  cacheRate: number | null
  deleted: boolean
  other: boolean
  canDrilldown: boolean
}

interface DisplayRow extends NormalizedItem {
  value: number
  share: number
  color: string
  formattedValue: string
  preciseValue: string
  extraMeta: string
}

const props = withDefaults(defineProps<{
  startDate: string
  endDate: string
  models: ModelStat[]
  groups: GroupStat[]
  apiKeys: APIKeyStat[]
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
const { tokens, seriesColor, tooltipTheme } = useChartTheme()

const dimension = ref<BreakdownDimension>('model')
const metric = ref<BreakdownMetric>('actual_cost')
const expanded = ref(false)
const hoveredIndex = ref<number | null>(null)
const skipNextAnim = ref(false)
const ringChartRef = ref<ChartComponentRef | null>(null)

const dimensionTabs = computed(() => [
  { id: 'model' as const, label: t('dashboard.overview.byModel') },
  { id: 'group' as const, label: t('dashboard.overview.byGroup') },
  { id: 'api_key' as const, label: t('dashboard.overview.byApiKey') },
])

const metricOptions = computed(() => [
  { id: 'actual_cost' as const, label: t('dashboard.overview.metricCostShort') },
  { id: 'total_tokens' as const, label: 'Token' },
  { id: 'requests' as const, label: t('dashboard.overview.metricRequestsShort') },
])

const activeDimensionIndex = computed(() =>
  Math.max(0, dimensionTabs.value.findIndex((d) => d.id === dimension.value))
)

const activeMetricIndex = computed(() =>
  Math.max(0, metricOptions.value.findIndex((m) => m.id === metric.value))
)

watch(() => [props.startDate, props.endDate, dimension.value], () => {
  expanded.value = false
  hoveredIndex.value = null
})

const emptyText = computed(() => {
  if (dimension.value === 'group') return t('dashboard.overview.noGroups')
  if (dimension.value === 'api_key') return t('dashboard.overview.noApiKeys')
  return t('dashboard.overview.noModels')
})

const allItems = computed<NormalizedItem[]>(() => {
  if (dimension.value === 'group') {
    return (props.groups ?? []).map((g) => ({
      key: `group-${g.group_id}`,
      id: g.group_id,
      rawName: g.group_name,
      label: g.group_id === 0 ? t('dashboard.overview.unassignedGroup') : (g.group_name || `#${g.group_id}`),
      actualCost: Number(g.actual_cost) || 0,
      standardCost: Number(g.cost) || 0,
      tokens: Number(g.total_tokens) || 0,
      requests: Number(g.requests) || 0,
      cacheRate: null,
      deleted: false,
      other: false,
      canDrilldown: g.group_id > 0,
    }))
  }
  if (dimension.value === 'api_key') {
    return (props.apiKeys ?? []).map((k) => ({
      key: `key-${k.api_key_id}`,
      id: k.api_key_id,
      rawName: k.key_name,
      label: k.key_name || `Key #${k.api_key_id}`,
      actualCost: Number(k.actual_cost) || 0,
      standardCost: Number(k.cost) || 0,
      tokens: Number(k.total_tokens) || 0,
      requests: Number(k.requests) || 0,
      cacheRate: null,
      deleted: Boolean(k.deleted),
      other: false,
      canDrilldown: !k.deleted && k.api_key_id > 0,
    }))
  }
  return (props.models ?? []).map((m) => {
    const cache = calculateCacheEfficiency(m.input_tokens, m.cache_creation_tokens, m.cache_read_tokens)
    return {
      key: `model-${m.model}`,
      id: m.model,
      rawName: m.model,
      label: m.model,
      actualCost: Number(m.actual_cost) || 0,
      standardCost: Number(m.cost) || 0,
      tokens: Number(m.total_tokens) || 0,
      requests: Number(m.requests) || 0,
      cacheRate: cache.promptTokens > 0 ? cache.rate : null,
      deleted: false,
      other: false,
      canDrilldown: Boolean(m.model),
    }
  })
})

function metricValue(item: NormalizedItem): number {
  if (metric.value === 'total_tokens') return item.tokens
  if (metric.value === 'requests') return item.requests
  return item.actualCost
}

function formatMetric(value: number): string {
  if (metric.value === 'total_tokens') return formatCompactTokens(value)
  if (metric.value === 'requests') return value.toLocaleString()
  return overviewMoney(value)
}

function formatMetricPrecise(value: number): string {
  if (metric.value === 'total_tokens') return `${value.toLocaleString()} Tokens`
  if (metric.value === 'requests') return value.toLocaleString()
  return preciseMoney(value)
}

const sortedItems = computed(() =>
  [...allItems.value].sort((a, b) => metricValue(b) - metricValue(a) || a.label.localeCompare(b.label))
)

const totalMetricValue = computed(() =>
  sortedItems.value.reduce((sum, item) => sum + metricValue(item), 0)
)

const visibleRows = computed<DisplayRow[]>(() => {
  const total = totalMetricValue.value
  const list = sortedItems.value
  const base = expanded.value ? list : list.slice(0, 5)
  const rows: DisplayRow[] = base.map((item, idx) => {
    const val = metricValue(item)
    const saved = item.standardCost - item.actualCost
    let extraMeta = ''
    if (item.cacheRate !== null && item.cacheRate > 0) {
      extraMeta = `${t('dashboard.overview.cacheHitShort')} ${item.cacheRate.toFixed(0)}%`
    } else if (saved > 0.005) {
      extraMeta = t('dashboard.overview.savedShort', { amount: overviewMoney(saved) })
    }
    return {
      ...item,
      value: val,
      share: total > 0 ? (val / total) * 100 : 0,
      color: seriesColor(idx),
      formattedValue: formatMetric(val),
      preciseValue: formatMetricPrecise(val),
      extraMeta,
    }
  })

  if (!expanded.value && list.length > 5) {
    const rest = list.slice(5)
    const restVal = rest.reduce((s, item) => s + metricValue(item), 0)
    const restCost = rest.reduce((s, item) => s + item.actualCost, 0)
    const restStd = rest.reduce((s, item) => s + item.standardCost, 0)
    const restTokens = rest.reduce((s, item) => s + item.tokens, 0)
    const restReqs = rest.reduce((s, item) => s + item.requests, 0)
    rows.push({
      key: '__other__',
      id: '__other__',
      rawName: '__other__',
      label: dimension.value === 'model' ? t('dashboard.overview.otherModels') : t('dashboard.overview.otherItems'),
      actualCost: restCost,
      standardCost: restStd,
      tokens: restTokens,
      requests: restReqs,
      cacheRate: null,
      deleted: false,
      other: true,
      canDrilldown: false,
      value: restVal,
      share: total > 0 ? (restVal / total) * 100 : 0,
      color: tokens.value.other,
      formattedValue: formatMetric(restVal),
      preciseValue: formatMetricPrecise(restVal),
      extraMeta: '',
    })
  }
  return rows
})

const centerReadout = computed(() => {
  if (hoveredIndex.value !== null && visibleRows.value[hoveredIndex.value]) {
    const row = visibleRows.value[hoveredIndex.value]
    return {
      label: row.label,
      value: row.formattedValue,
      title: row.preciseValue,
      sub: shareLabel(row.share),
    }
  }
  return {
    label: t('dashboard.overview.totalLabel'),
    value: formatMetric(totalMetricValue.value),
    title: formatMetricPrecise(totalMetricValue.value),
    sub: t('dashboard.overview.itemsCount', { count: allItems.value.length }),
  }
})

const centerValueParts = computed(() => splitFormattedDecimal(centerReadout.value.value))

const hasPositiveRingData = computed(() => totalMetricValue.value > 0)

const ringHighlights = computed(() => {
  const rows = visibleRows.value
  if (!rows.length || !hasPositiveRingData.value) return null
  const top = rows[0]
  const topThreePct = rows
    .filter((r) => !r.other)
    .slice(0, 3)
    .reduce((sum, r) => sum + r.share, 0)
  const totalReqs = sortedItems.value.reduce((sum, item) => sum + item.requests, 0)
  return {
    topLabel: top.label,
    topColor: top.color,
    topValue: top.formattedValue,
    topShare: shareLabel(top.share),
    topThreeShare: `${Math.min(100, topThreePct).toFixed(1)}%`,
    totalRequests: totalReqs.toLocaleString(),
  }
})

const ringChartData = computed(() => {
  const rows = visibleRows.value
  if (!hasPositiveRingData.value) {
    return {
      labels: [t('dashboard.overview.totalLabel')],
      datasets: [
        {
          data: [1],
          backgroundColor: [tokens.value.grid],
          borderWidth: 0,
          hoverOffset: 0,
        },
      ],
    }
  }
  return {
    labels: rows.map((r) => r.label),
    datasets: [
      {
        data: rows.map((r) => r.value),
        backgroundColor: (ctx: { dataIndex?: number }) => {
          const idx = ctx.dataIndex ?? 0
          const row = visibleRows.value[idx]
          const baseColor = row?.color ?? tokens.value.other
          if (hoveredIndex.value !== null && hoveredIndex.value !== idx) {
            return hexToRgba(baseColor, 0.28)
          }
          return baseColor
        },
        borderColor: tokens.value.surface,
        borderWidth: 2,
        borderRadius: 4,
        hoverOffset: 6,
      },
    ],
  }
})

watch(hoveredIndex, (idx) => {
  const chart = ringChartRef.value?.chart as Chart | undefined
  if (!chart || !hasPositiveRingData.value) return
  if (typeof chart.setActiveElements === 'function') {
    chart.setActiveElements(idx !== null ? [{ datasetIndex: 0, index: idx }] : [])
  }
  chart.update('none')
})

watch(
  () => tokens.value.isDark,
  () => {
    skipNextAnim.value = true
    void nextTick(() => {
      const chart = ringChartRef.value?.chart as Chart | undefined
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

function handleRingHover(event: ChartEvent, elements: ActiveElement[], chart: Chart) {
  if (!hasPositiveRingData.value) {
    hoveredIndex.value = null
    return
  }
  const idx = elements.length > 0 ? elements[0].index : null
  if (hoveredIndex.value !== idx) {
    hoveredIndex.value = idx
  }
  const target = (event.native?.target ?? chart?.canvas) as HTMLElement | null | undefined
  if (target) {
    const row = idx !== null ? visibleRows.value[idx] : null
    target.style.cursor = row?.canDrilldown ? 'pointer' : 'default'
  }
}

function handleRingClick(_event: ChartEvent, elements: ActiveElement[]) {
  if (!hasPositiveRingData.value) return
  const idx = elements[0]?.index
  if (typeof idx !== 'number') return
  const row = visibleRows.value[idx]
  if (row?.canDrilldown) {
    handleRowClick(row)
  }
}

const ringChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '79%',
  animation: skipNextAnim.value ? false : chartAnimationConfig(),
  onHover: handleRingHover,
  onClick: handleRingClick,
  plugins: {
    legend: { display: false },
    tooltip: tooltipTheme({
      enabled: hasPositiveRingData.value,
      callbacks: {
        label: (ctx: any) => {
          const row = visibleRows.value[ctx.dataIndex]
          if (!row) return ''
          return `${row.label}: ${row.formattedValue} (${shareLabel(row.share)})`
        },
      },
    }),
  },
}))

function highlightSlice(idx: number) {
  hoveredIndex.value = idx
}

function clearHover() {
  hoveredIndex.value = null
}

function handleRowClick(row: DisplayRow) {
  if (!row.canDrilldown) return
  const query: Record<string, string> = {
    start_date: props.startDate,
    end_date: props.endDate,
  }
  if (props.apiKeyId) query.api_key_id = String(props.apiKeyId)
  if (props.groupId) query.group_id = String(props.groupId)

  if (dimension.value === 'model') {
    query.model = String(row.id)
  } else if (dimension.value === 'group') {
    query.group_id = String(row.id)
  } else if (dimension.value === 'api_key') {
    query.api_key_id = String(row.id)
  }
  void router?.push({ path: '/usage', query })
}

const shareLabel = (share: number) => (share > 0 && share < 0.1 ? '<0.1%' : `${share.toFixed(1)}%`)
</script>
