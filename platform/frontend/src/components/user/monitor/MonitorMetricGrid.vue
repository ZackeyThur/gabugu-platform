<template>
  <div class="mt-4 grid grid-cols-3 gap-2">
    <div
      class="min-w-0 rounded-xl bg-pearl-raised/45 p-2 sm:p-3 dark:bg-white/[0.022]"
    >
      <div
        class="truncate text-xs font-semibold tracking-normal text-gray-400 sm:tracking-wide"
        :title="availabilityLabel"
        data-testid="metric-label"
      >
        {{ availabilityLabel }}
      </div>
      <div class="mt-1.5 flex items-baseline gap-0.5 font-mono text-lg font-bold leading-none tabular-nums sm:text-xl">
        <span class="truncate" :style="availabilityStyle" data-testid="metric-value">{{ availabilityText }}</span>
        <span v-if="hasAvailability" class="text-xs font-semibold" :style="availabilityStyle" data-testid="metric-unit">%</span>
      </div>
    </div>

    <div
      class="min-w-0 rounded-xl bg-pearl-raised/45 p-2 sm:p-3 dark:bg-white/[0.022]"
    >
      <div class="flex items-center gap-1 text-xs font-semibold tracking-normal text-gray-400 sm:tracking-wide">
        <Icon name="bolt" size="xs" class="shrink-0" />
        <span class="truncate" :title="t('monitorCommon.dialogLatency')" data-testid="metric-label">{{ t('monitorCommon.dialogLatency') }}</span>
      </div>
      <div class="mt-1.5 flex items-baseline gap-0.5 font-mono text-lg font-bold leading-none tabular-nums text-gray-900 sm:text-xl dark:text-gray-100">
        <span class="truncate" data-testid="metric-value">{{ latencyText }}</span>
        <span v-if="hasLatency" class="text-xs font-normal text-gray-400" data-testid="metric-unit">ms</span>
      </div>
    </div>

    <div
      class="min-w-0 rounded-xl bg-pearl-raised/45 p-2 sm:p-3 dark:bg-white/[0.022]"
    >
      <div class="flex items-center gap-1 text-xs font-semibold tracking-normal text-gray-400 sm:tracking-wide">
        <Icon name="globe" size="xs" class="shrink-0" />
        <span class="truncate" :title="t('monitorCommon.endpointPing')" data-testid="metric-label">{{ t('monitorCommon.endpointPing') }}</span>
      </div>
      <div class="mt-1.5 flex items-baseline gap-0.5 font-mono text-lg font-bold leading-none tabular-nums text-gray-900 sm:text-xl dark:text-gray-100">
        <span class="truncate" data-testid="metric-value">{{ pingText }}</span>
        <span v-if="hasPing" class="text-xs font-normal text-gray-400" data-testid="metric-unit">ms</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  useChannelMonitorFormat,
  hslForPct,
} from '@/composables/useChannelMonitorFormat'

const props = defineProps<{
  availabilityLabel: string
  availabilityValue: number | null
  latencyMs: number | null
  pingMs: number | null
}>()

const { t } = useI18n()
const { formatLatency } = useChannelMonitorFormat()

/**
 * 用有限数判断代替 === null：接口缺字段时是 undefined，直接调 toFixed 会抛。
 * 数值和单位共用同一个判断——真实的 0 要保留成 0.00% / 0ms，缺失值只留破折号，
 * 不能出现「—ms」或「NaN%」。
 */
function isNum(value: number | null | undefined): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

const hasAvailability = computed(() => isNum(props.availabilityValue))
const hasLatency = computed(() => isNum(props.latencyMs))
const hasPing = computed(() => isNum(props.pingMs))

const availabilityText = computed(() => {
  if (!isNum(props.availabilityValue)) return t('monitorCommon.latencyEmpty')
  return props.availabilityValue.toFixed(2)
})

const latencyText = computed(() =>
  isNum(props.latencyMs) ? formatLatency(props.latencyMs) : t('monitorCommon.latencyEmpty')
)

const pingText = computed(() =>
  isNum(props.pingMs) ? formatLatency(props.pingMs) : t('monitorCommon.latencyEmpty')
)

const availabilityStyle = computed(() => {
  // hslForPct 只挡 NaN，Infinity 会被夹成绿色；这里只对有限数上色。
  if (!isNum(props.availabilityValue)) return { color: 'rgb(156 163 175)' }
  const colour = hslForPct(props.availabilityValue)
  return colour ? { color: colour } : { color: 'rgb(156 163 175)' }
})
</script>
