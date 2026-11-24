<template>
  <section
    class="overview-savings"
    :class="{ 'is-compact': compact }"
    aria-labelledby="savings-heading"
  >
    <div class="overview-savings-head">
      <div class="overview-savings-main">
        <div class="overview-savings-toprow">
          <p id="savings-heading" class="overview-savings-label">
            {{ t(difference < 0 ? 'dashboard.overview.aboveStandard' : 'dashboard.overview.savingsTitle') }}
          </p>
          <button type="button" class="overview-link overview-savings-explain" @click="open = true">
            {{ t('dashboard.overview.comparisonMethod') }}
            <Icon name="chevronRight" size="sm" />
          </button>
        </div>

        <div v-if="loading" class="overview-skeleton overview-skeleton-amount" />
        <p v-else-if="error" role="alert">
          {{ t('dashboard.overview.statsError') }}
          <button type="button" class="overview-link" @click="$emit('retry')">{{ t('common.retry') }}</button>
        </p>
        <div v-else class="overview-savings-value">
          <strong :title="preciseMoney(Math.abs(difference))">
            <CountUp :value="Math.abs(difference)" :formatter="overviewMoney" split-decimals />
          </strong>
          <div v-if="(stats && stats.total_cost > 0 && difference > 0) || todaySaved > 0.005" class="overview-savings-meta-inline">
            <span v-if="stats && stats.total_cost > 0 && difference > 0" class="overview-savings-pct">
              {{ t('dashboard.overview.lessBy', { percent: (difference / stats.total_cost * 100).toFixed(1) }) }}
            </span>
            <span
              v-if="stats && stats.total_cost > 0 && difference > 0 && todaySaved > 0.005"
              class="overview-inline-sep"
              aria-hidden="true"
            >·</span>
            <span v-if="todaySaved > 0.005" class="overview-savings-today">
              {{ t('dashboard.overview.todayDifference') }} {{ overviewMoney(todaySaved) }}
            </span>
          </div>
        </div>
        <p class="overview-savings-basis">{{ t('dashboard.overview.savingsBasis') }}</p>
      </div>

      <!-- Open 3-Column Architectural KPI Strip -->
      <div v-if="stats && !loading && !error" class="overview-savings-kpis">
        <div class="overview-savings-kpi">
          <span>{{ t('dashboard.overview.effectivePayRatio') }}</span>
          <strong class="overview-number">
            <template v-if="splitMetricUnit(effectiveRatioLabel)">
              <span>{{ splitMetricUnit(effectiveRatioLabel)!.value }}</span><span class="overview-unit-suffix">{{ splitMetricUnit(effectiveRatioLabel)!.unit }}</span>
            </template>
            <template v-else>{{ effectiveRatioLabel }}</template>
          </strong>
        </div>
        <div class="overview-savings-kpi">
          <span>{{ t('dashboard.overview.lifetimeRequests') }}</span>
          <strong class="overview-number">{{ (stats.total_requests ?? 0).toLocaleString() }}</strong>
        </div>
        <div class="overview-savings-kpi">
          <span>{{ t('dashboard.totalTokens') }}</span>
          <strong class="overview-number">
            <template v-if="splitMetricUnit(formatCompactTokens(stats.total_tokens ?? 0))">
              <span>{{ splitMetricUnit(formatCompactTokens(stats.total_tokens ?? 0))!.value }}</span><span class="overview-unit-suffix">{{ splitMetricUnit(formatCompactTokens(stats.total_tokens ?? 0))!.unit }}</span>
            </template>
            <template v-else>{{ formatCompactTokens(stats.total_tokens ?? 0) }}</template>
          </strong>
        </div>
      </div>
    </div>

    <div v-if="stats && !loading && !error" class="overview-savings-body">
      <!-- Unified Architectural Differential Bar: Actual Spend + Shimmering Visualized Savings Gap -->
      <div
        class="overview-savings-comparison"
        :data-hover-segment="hoveredDiff ?? 'none'"
        @mouseleave="hoveredDiff = null"
      >
        <div class="overview-diff-labels">
          <div class="overview-diff-side is-actual" :class="{ 'is-active': hoveredDiff === 'actual' }">
            <span class="overview-diff-dot is-actual" aria-hidden="true" />
            <span>{{ t('dashboard.overview.gabuguBilling') }}</span>
            <strong class="overview-number">{{ overviewMoney(stats.total_actual_cost) }}</strong>
            <span v-if="hoveredDiff === 'actual' && actualBarPercent > 0" class="overview-diff-pct">
              ({{ actualBarPercent.toFixed(1) }}%)
            </span>
          </div>
          <div class="overview-diff-side is-standard" :class="{ 'is-active': hoveredDiff === 'saved' }">
            <template v-if="hoveredDiff === 'saved' && difference > 0">
              <span class="overview-diff-dot is-saved" aria-hidden="true" />
              <span>{{ t('dashboard.overview.savingsTitle') }}</span>
              <strong class="overview-number is-saved-accent">
                {{ overviewMoney(difference) }} ({{ savedGapPercent.toFixed(1) }}%)
              </strong>
            </template>
            <template v-else>
              <span>{{ t('dashboard.overview.standardBilling') }}</span>
              <strong class="overview-number">{{ overviewMoney(stats.total_cost) }}</strong>
            </template>
          </div>
        </div>
        <div class="overview-diff-track" aria-hidden="true">
          <span
            class="overview-diff-actual"
            :class="{ 'is-active': hoveredDiff === 'actual', 'is-dimmed': hoveredDiff === 'saved' }"
            :style="{ width: actualBarPercent + '%' }"
            @mouseenter="hoveredDiff = 'actual'"
          />
          <span
            v-if="savedGapPercent > 0.5"
            class="overview-diff-saved"
            :class="{ 'is-active': hoveredDiff === 'saved', 'is-dimmed': hoveredDiff === 'actual' }"
            :style="{ width: savedGapPercent + '%' }"
            @mouseenter="hoveredDiff = 'saved'"
          />
        </div>
      </div>

      <!-- Swiss Mechanical Chronograph Cache Efficiency Dial (24-Tick Bezel) -->
      <div class="overview-cache-proof" :title="t('dashboard.overview.cacheEfficiencyTooltip')">
        <div class="overview-cache-arc" aria-hidden="true">
          <svg viewBox="0 0 68 68" width="68" height="68">
            <g class="overview-chrono-ticks">
              <line
                v-for="tick in chronoTicks"
                :key="tick.angle"
                :x1="tick.x1"
                :y1="tick.y1"
                :x2="tick.x2"
                :y2="tick.y2"
                :class="{ 'is-cardinal': tick.cardinal }"
              />
            </g>
            <circle class="overview-cache-arc-bg" cx="34" cy="34" r="24" />
            <circle
              class="overview-cache-arc-fg"
              cx="34"
              cy="34"
              r="24"
              :stroke-dasharray="ringCircumference"
              :stroke-dashoffset="ringDashOffset"
            />
          </svg>
          <span class="overview-cache-arc-pct">{{ cacheRateLabel }}</span>
        </div>
        <div class="overview-cache-meta">
          <span class="overview-cache-title">{{ t('dashboard.overview.cacheEfficiencyTitle') }}</span>
          <span class="overview-cache-sub">
            {{
              cacheInfo.promptTokens > 0
                ? t('dashboard.overview.cacheReadSummary', { tokens: formatCompactTokens(cacheInfo.cacheReadTokens) })
                : t('dashboard.overview.cacheNoPromptYet')
            }}
          </span>
        </div>
        <span v-if="todayCacheRate !== null" class="overview-cache-today-pill">
          {{ t('dashboard.overview.todayCacheHit') }} {{ todayCacheRate.toFixed(0) }}%
        </span>
      </div>
    </div>
  </section>

  <DashboardDrawer :open="open" :title="t('dashboard.overview.comparisonMethod')" @close="open = false">
    <p class="overview-detail-intro">{{ t('dashboard.overview.savingsExplanation') }}</p>
    <p class="overview-detail-intro">{{ t('dashboard.overview.officialComparisonBoundary') }}</p>
    <p class="overview-detail-intro">{{ t('dashboard.overview.subscriptionExplanation') }}</p>
    <dl v-if="stats && !loading && !error" class="overview-detail-list">
      <div><dt>{{ t('dashboard.overview.lifetimeStandard') }}</dt><dd>{{ preciseMoney(stats.total_cost) }}</dd></div>
      <div><dt>{{ t('dashboard.overview.gabuguBilling') }}</dt><dd>{{ preciseMoney(stats.total_actual_cost) }}</dd></div>
      <div><dt>{{ t('dashboard.overview.signedDifference') }}</dt><dd>{{ preciseMoney(difference) }}</dd></div>
      <div><dt>{{ t('dashboard.overview.todayDifference') }}</dt><dd>{{ preciseMoney(stats.today_cost - stats.today_actual_cost) }}</dd></div>
      <div><dt>{{ t('dashboard.overview.cacheEfficiencyTitle') }}</dt><dd>{{ cacheRateLabel }}</dd></div>
      <div><dt>{{ t('dashboard.overview.tokenCacheRead') }}</dt><dd>{{ (stats.total_cache_read_tokens ?? 0).toLocaleString() }}</dd></div>
    </dl>
    <p v-else>{{ t(loading ? 'common.loading' : 'dashboard.overview.statsError') }}</p>
  </DashboardDrawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserDashboardStats } from '@/api/usage'
import {
  calculateCacheEfficiency,
  formatCompactTokens,
  overviewMoney,
  preciseMoney,
  splitMetricUnit,
} from '@/utils/dashboardOverview'
import DashboardDrawer from './DashboardDrawer.vue'
import CountUp from '@/components/common/CountUp.vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  stats: UserDashboardStats | null
  loading: boolean
  error: boolean
  subscriptionEnabled: boolean
  compact?: boolean
}>(), {
  compact: false,
})

defineEmits<{ retry: [] }>()

const { t } = useI18n()
const open = ref(false)
const hoveredDiff = ref<'actual' | 'saved' | null>(null)
const ringCircumference = 2 * Math.PI * 24

// 24-tick Swiss chronograph bezel around r=29..32 (cardinals at 12/3/6/9 o'clock)
const chronoTicks = Array.from({ length: 24 }, (_, idx) => {
  const angle = idx * 15
  const rad = (angle * Math.PI) / 180
  const cardinal = idx % 6 === 0
  const rInner = cardinal ? 28.6 : 29.8
  const rOuter = 32.2
  return {
    angle,
    cardinal,
    x1: Number((34 + rInner * Math.cos(rad)).toFixed(2)),
    y1: Number((34 + rInner * Math.sin(rad)).toFixed(2)),
    x2: Number((34 + rOuter * Math.cos(rad)).toFixed(2)),
    y2: Number((34 + rOuter * Math.sin(rad)).toFixed(2)),
  }
})

const difference = computed(() =>
  props.stats ? props.stats.total_cost - props.stats.total_actual_cost : 0
)

const todaySaved = computed(() =>
  props.stats ? Math.max(0, props.stats.today_cost - props.stats.today_actual_cost) : 0
)

const effectiveRatioLabel = computed(() => {
  const std = props.stats?.total_cost ?? 0
  const act = props.stats?.total_actual_cost ?? 0
  if (std <= 0) return '—'
  return `${((act / std) * 100).toFixed(1)}%`
})

const cacheInfo = computed(() =>
  calculateCacheEfficiency(
    props.stats?.total_input_tokens ?? 0,
    props.stats?.total_cache_creation_tokens ?? 0,
    props.stats?.total_cache_read_tokens ?? 0
  )
)

const todayCacheRate = computed(() => {
  if (!props.stats) return null
  const c = calculateCacheEfficiency(
    props.stats.today_input_tokens,
    props.stats.today_cache_creation_tokens,
    props.stats.today_cache_read_tokens
  )
  return c.promptTokens > 0 ? c.rate : null
})

const cacheRateLabel = computed(() =>
  cacheInfo.value.promptTokens > 0 ? `${cacheInfo.value.rate.toFixed(0)}%` : '0%'
)

const ringDashOffset = computed(() => {
  const clamped = Math.max(0, Math.min(100, cacheInfo.value.rate))
  return ringCircumference * (1 - clamped / 100)
})

const actualBarPercent = computed(() => {
  const std = props.stats?.total_cost ?? 0
  const act = props.stats?.total_actual_cost ?? 0
  const maximum = Math.max(std, act)
  if (maximum <= 0) return 0
  return Math.max(0, Math.min(100, (act / maximum) * 100))
})

const savedGapPercent = computed(() => {
  const std = props.stats?.total_cost ?? 0
  const act = props.stats?.total_actual_cost ?? 0
  if (std <= 0 || std <= act) return 0
  return Math.max(0, Math.min(100 - actualBarPercent.value, ((std - act) / std) * 100))
})
</script>
