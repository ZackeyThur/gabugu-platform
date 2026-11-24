<template>
  <div class="mt-3 border-t border-gray-100 pt-3 dark:border-dark-700/60">
    <div
      class="mb-2 flex items-center justify-between gap-2 text-xs font-semibold uppercase tracking-widest text-gray-400"
    >
      <span
        class="truncate transition-colors duration-150"
        :class="hoveredTitle ? 'normal-case tracking-normal font-mono text-gray-700 dark:text-gray-200' : ''"
      >
        {{ hoveredTitle || (hasSamples ? samplesLabel : t('monitorCommon.noSamples')) }}
      </span>
      <span class="shrink-0 tabular-nums">
        {{ autoRefreshOn ? t('monitorCommon.nextUpdateIn', { n: countdownSeconds }) : t('monitorCommon.autoRefreshOff') }}
      </span>
    </div>

    <div
      v-if="maintenance"
      class="flex h-5 w-full items-center justify-center rounded border border-dashed border-gray-300 dark:border-dark-600 text-xs uppercase tracking-widest text-gray-400"
    >
      {{ t('monitorCommon.maintenancePaused') }}
    </div>
    <div
      v-else-if="!hasSamples"
      class="flex h-5 w-full items-center justify-center rounded border border-dashed border-gray-300 text-xs uppercase tracking-widest text-gray-400 dark:border-dark-600"
    >
      {{ t('monitorCommon.noSamples') }}
    </div>
    <!--
      宽格状态条：槽位均分整行宽度、柱体贴底生长。
      旧实现 60 格 + 2px 间隙，在四列布局下每格只有 3～4px，看起来像一排线；
      现在按参考站的量级：7 天 14 格、24 小时 24 格，格间距 4px，轨道高 20px。
      格数只表示「最近 N 次检测」，不冒充任何时间窗口的分桶统计。
    -->
    <div
      v-else
      class="flex h-5 w-full items-end gap-1"
      data-testid="timeline-track"
      @mouseleave="hoveredTitle = ''"
    >
      <div
        v-for="(bar, idx) in displayBars"
        :key="idx"
        class="flex h-full min-w-0 flex-1 items-end"
        data-testid="timeline-slot"
        @mouseenter="hoveredTitle = bar.title"
      >
        <div
          class="monitor-bar w-full origin-bottom rounded-[3px] transition-all duration-200 ease-out hover:scale-y-[1.18] hover:brightness-110"
          :class="bar.colorClass"
          :style="{ height: bar.heightPct + '%', minHeight: '3px', animationDelay: `${idx * 24}ms` }"
          :title="bar.title"
          :data-status="bar.status"
          data-testid="timeline-bar"
        ></div>
      </div>
    </div>

    <div
      class="mt-1 flex justify-between text-xs uppercase tracking-widest text-gray-400"
    >
      <span>{{ t('monitorCommon.past') }}</span>
      <span>{{ t('monitorCommon.now') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const props = withDefaults(defineProps<{
  buckets?: MonitorTimelinePoint[]
  countdownSeconds: number
  /** 自动刷新关闭时，右侧不再显示 "0s 后刷新"。 */
  autoRefreshOn?: boolean
  /**
   * 状态条格数，默认 14 格。
   * 注意语义：这是「最近 N 次逐次检测」，不是按天/小时聚合的分桶，
   * 所以不随 7/15/30 天周期切换而改变含义。缺格用灰色占位，不伪造健康记录。
   */
  length?: number
  maintenance?: boolean
}>(), {
  buckets: () => [],
  autoRefreshOn: true,
  length: 14,
  maintenance: false,
})

const { t } = useI18n()
const { statusLabel, formatLatency, formatRelativeTime } = useChannelMonitorFormat()
const hoveredTitle = ref('')

// 实际样本数才是标题里的数字：只有 3 条记录时不能把 57 个占位格也算成检测。
const hasSamples = computed(() => (props.buckets ?? []).length > 0)

const samplesLabel = computed(() => {
  const n = Math.min(props.buckets?.length ?? 0, props.length)
  return t('monitorCommon.history60pts', { n })
})

interface Bar {
  status: string
  colorClass: string
  heightPct: number
  title: string
}

// 4 级高度 + 颜色双重编码：高=好+绿，短=坏+红，灰=未测试。
// 长绿(正常) > 中黄(降级) > 短红(失败/系统错误) > 很短灰(未测试)。
const STATUS_HEIGHT: Record<string, number> = {
  operational: 100,
  degraded: 65,
  failed: 35,
  error: 35,
  empty: 15,
}

const STATUS_COLOR: Record<string, string> = {
  operational: 'bg-emerald-500',
  degraded: 'bg-amber-500',
  failed: 'bg-red-500',
  error: 'bg-red-500',
  empty: 'bg-gray-300 dark:bg-dark-600',
}

const displayBars = computed<Bar[]>(() => {
  // Real points come newest-first; convert to oldest-first so the rightmost
  // bar represents "now". Pad the left with empty placeholders to keep the
  // bar count stable at `length`.
  const real = [...(props.buckets ?? [])]
    .slice(0, props.length)
    .reverse()

  const padCount = Math.max(0, props.length - real.length)
  const bars: Bar[] = []

  for (let i = 0; i < padCount; i += 1) {
    bars.push({
      status: 'empty',
      colorClass: STATUS_COLOR.empty,
      heightPct: STATUS_HEIGHT.empty,
      title: '',
    })
  }

  for (const point of real) {
    // 用 hasOwnProperty.call 而非 Object.hasOwn：项目 TS lib 目标不含 ES2022，
    // 也不能用 in（会把原型链上的键当成合法状态）
    const status = Object.prototype.hasOwnProperty.call(STATUS_HEIGHT, point.status)
      ? point.status
      : 'empty'
    const hasLatency = typeof point.latency_ms === 'number' && Number.isFinite(point.latency_ms)
    const latency = formatLatency(point.latency_ms)
    const relative = formatRelativeTime(point.checked_at)
    const label = statusLabel(point.status)
    bars.push({
      status,
      colorClass: STATUS_COLOR[status],
      heightPct: STATUS_HEIGHT[status],
      // 缺失延迟时 formatLatency 返回 '-'，不能再拼 ms，否则出现「-ms」
      title: `${relative} · ${label} · ${hasLatency ? `${latency}ms` : '-'}`,
    })
  }

  return bars
})
</script>

<style scoped>
.monitor-bar {
  animation: g-bar-sweep-y 420ms cubic-bezier(0.22, 1, 0.36, 1) both;
}
</style>
