<template>
  <section class="space-y-3 pb-1">
    <!-- 标题 / 总体结论 / 操作 -->
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="g-blur-in flex items-center gap-2 text-xl font-black text-gray-900 dark:text-white">
          <span
            class="grid h-8 w-8 place-items-center rounded-xl bg-primary-50 text-primary-500 dark:bg-primary-900/30 dark:text-primary-300"
          >
            <Icon name="chart" size="sm" />
          </span>
          {{ t('channelStatus.title') }}
        </h1>

        <div class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          <span class="inline-flex items-center gap-1.5 font-semibold" :class="overallTextClass">
            <span
              class="h-2 w-2 rounded-full"
              :class="[overallDotClass, overallStatus === 'operational' ? 'g-live-dot text-emerald-500' : '']"
            />
            {{ overallLabel }}
          </span>
          <span v-if="counts.total > 0" class="tabular-nums text-gray-500 dark:text-gray-400">
            {{ t('channelStatus.summary.channels', { n: counts.total }) }}
          </span>
          <span
            v-if="counts.operational > 0"
            class="tabular-nums text-emerald-600 dark:text-emerald-400"
          >
            {{ t('channelStatus.summary.operational', { n: counts.operational }) }}
          </span>
          <span v-if="counts.degraded > 0" class="tabular-nums text-amber-600 dark:text-amber-400">
            {{ t('channelStatus.summary.degraded', { n: counts.degraded }) }}
          </span>
          <span v-if="lastCheckLabel" class="tabular-nums text-gray-400 dark:text-gray-500">
            {{ lastCheckLabel }}
          </span>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <div
          role="tablist"
          class="g-segmented text-xs"
          :style="{ '--seg-count': windowOptions.length, '--seg-index': activeWindowIndex }"
        >
          <span class="g-segmented-thumb" aria-hidden="true" />
          <button
            v-for="opt in windowOptions"
            :key="opt.value"
            type="button"
            role="tab"
            :aria-selected="window === opt.value"
            class="rounded-lg px-3 py-1 transition-colors duration-200"
            :class="window === opt.value
              ? 'font-semibold text-gray-900 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:window', opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>

        <button
          type="button"
          data-magnet
          data-spark
          class="flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:opacity-50 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-gray-200"
          :disabled="loading"
          :title="t('common.refresh')"
          :aria-label="t('common.refresh')"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>

        <AutoRefreshButton
          v-if="autoRefresh"
          :enabled="autoRefresh.enabled.value"
          :interval-seconds="autoRefresh.intervalSeconds.value"
          :countdown="autoRefresh.countdown.value"
          :intervals="autoRefresh.intervals"
          @update:enabled="autoRefresh.setEnabled"
          @update:interval="autoRefresh.setInterval"
        />
      </div>
    </div>

    <!-- 平台筛选 + 搜索 -->
    <div v-if="total > 0" class="flex flex-wrap items-center gap-2">
      <div class="flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 text-xs font-medium transition-colors"
          :class="filterChipClass(activeProvider === '')"
          :aria-pressed="activeProvider === ''"
          @click="emit('update:provider', '')"
        >
          {{ t('channelStatus.filter.all') }}
          <span class="font-mono text-xs opacity-60">{{ total }}</span>
        </button>
        <button
          v-for="p in providers"
          :key="p"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 text-xs font-medium transition-colors"
          :class="filterChipClass(activeProvider === p)"
          :aria-pressed="activeProvider === p"
          @click="emit('update:provider', activeProvider === p ? '' : p)"
        >
          <ProviderIcon :provider="p" :size="12" />
          {{ providerLabel(p) }}
        </button>
      </div>

      <div class="relative ml-auto w-full sm:w-56">
        <span
          class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-gray-400"
        >
          <Icon name="search" size="sm" />
        </span>
        <input
          :value="searchQuery"
          type="search"
          class="input !py-1.5 !pl-8 !text-xs"
          :placeholder="t('channelStatus.searchPlaceholder')"
          :aria-label="t('channelStatus.searchPlaceholder')"
          @input="emit('update:search', ($event.target as HTMLInputElement).value)"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import ProviderIcon from './ProviderIcon.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

export type MonitorWindow = '7d' | '15d' | '30d'
/**
 * operational = 全部已知正常；degraded = 至少一条失败/降级；
 * partial = 有渠道但状态未知；empty = 没有任何可展示渠道。
 */
export type OverallStatus = 'operational' | 'degraded' | 'partial' | 'empty'

const props = withDefaults(defineProps<{
  overallStatus: OverallStatus
  intervalSeconds: number
  window: MonitorWindow
  loading: boolean
  /** 基于全部渠道统计，不随搜索/筛选变化。 */
  counts?: { total: number; operational: number; degraded: number }
  /** 全部渠道里最近一次真实检测时间（ISO），不是页面刷新时间。 */
  lastCheckAt?: string | null
  providers?: string[]
  activeProvider?: string
  searchQuery?: string
  autoRefresh?: {
    enabled: { value: boolean }
    intervalSeconds: { value: number }
    countdown: { value: number }
    intervals: readonly number[]
    setEnabled: (v: boolean) => void
    setInterval: (v: number) => void
  }
}>(), {
  counts: () => ({ total: 0, operational: 0, degraded: 0 }),
  lastCheckAt: null,
  providers: () => [],
  activeProvider: '',
  searchQuery: '',
})

const emit = defineEmits<{
  (e: 'update:window', value: MonitorWindow): void
  (e: 'update:provider', value: string): void
  (e: 'update:search', value: string): void
  (e: 'refresh'): void
}>()

const { t } = useI18n()
const { providerLabel } = useChannelMonitorFormat()

const windowOptions = computed<{ value: MonitorWindow; label: string }[]>(() => [
  { value: '7d', label: t('channelStatus.windowTab.7d') },
  { value: '15d', label: t('channelStatus.windowTab.15d') },
  { value: '30d', label: t('channelStatus.windowTab.30d') },
])

const activeWindowIndex = computed(() => {
  const idx = windowOptions.value.findIndex((o) => o.value === props.window)
  return idx >= 0 ? idx : 0
})

const total = computed(() => props.counts?.total ?? 0)
const overallLabel = computed(() => t(`channelStatus.overall.${props.overallStatus}`))

const lastCheckLabel = computed(() => {
  if (!props.lastCheckAt) return ''
  const ts = Date.parse(props.lastCheckAt)
  if (Number.isNaN(ts)) return ''
  // 显示月日 + 时分。只给时分的话，一天前或一周前的 10:20 看起来仍像今天。
  // 用绝对时刻而非"几秒前"也能避免数据停更时一直显示同样的"最新"字样。
  const time = new Intl.DateTimeFormat(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(ts))
  return t('channelStatus.lastCheck', { time })
})

const overallTextClass = computed(() => {
  switch (props.overallStatus) {
    case 'operational':
      return 'text-emerald-600 dark:text-emerald-400'
    case 'degraded':
      return 'text-amber-600 dark:text-amber-400'
    case 'partial':
      return 'text-sky-600 dark:text-sky-400'
    case 'empty':
    default:
      return 'text-gray-500 dark:text-gray-400'
  }
})

const overallDotClass = computed(() => {
  switch (props.overallStatus) {
    case 'operational':
      return 'bg-emerald-500'
    case 'degraded':
      return 'bg-amber-500'
    case 'partial':
      return 'bg-sky-500'
    case 'empty':
    default:
      return 'bg-gray-400 dark:bg-dark-500'
  }
})

function filterChipClass(active: boolean): string {
  return active
    ? 'border-transparent bg-primary-600/[0.11] text-primary-700 ring-1 ring-inset ring-primary-500/35 dark:bg-primary-500/15 dark:text-primary-200 dark:ring-primary-400/30'
    : 'ui-pill border-transparent text-gray-600 hover:text-gray-900 dark:bg-white/[0.03] dark:text-gray-300 dark:shadow-none dark:ring-1 dark:ring-inset dark:ring-white/[0.07] dark:hover:bg-white/[0.065] dark:hover:text-white'
}
</script>
