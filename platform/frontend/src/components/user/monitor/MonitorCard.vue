<template>
  <button
    type="button"
    data-tilt
    data-tilt-max="2.4"
    data-spark
    class="group ui-soft-card card-hover flex w-full flex-col p-5 text-left active:translate-y-0"
    @click="emit('click')"
  >
    <!-- Header: icon + name/model + status chip -->
    <div class="flex items-start gap-3">
      <span
        class="grid h-10 w-10 flex-shrink-0 place-items-center rounded-xl ring-1 ring-black/5 transition-transform duration-200 group-hover:scale-105 dark:ring-white/10"
        :class="[providerGradient(item.provider), providerTintClass]"
      >
        <ProviderIcon :provider="item.provider" :size="20" />
      </span>
      <div class="min-w-0 flex-1">
        <div class="truncate text-base font-semibold tracking-[-0.01em] text-gray-900 dark:text-gray-100">
          {{ item.name }}
        </div>
        <div class="mt-0.5 flex min-w-0 items-center gap-1.5">
          <span
            class="inline-flex flex-shrink-0 items-center rounded-md px-1.5 py-0.5 text-xs font-medium"
            :class="providerBadgeClass(item.provider)"
          >
            {{ providerLabel(item.provider) }}
          </span>
          <!-- 纯配额模式主模型是占位符 "quota"，展示层替换为本地化「配额」标签 -->
          <span class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">
            {{ formatMonitorModel(item.primary_model) }}
          </span>
          <span
            v-if="item.group_name"
            class="inline-flex flex-shrink-0 items-center rounded-md bg-pearl-raised/80 px-1.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
          >
            {{ item.group_name }}
          </span>
        </div>
      </div>
      <span
        class="inline-flex flex-shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold"
        :class="statusBadgeClass(item.primary_status)"
      >
        <span
          v-if="item.primary_status === 'operational'"
          class="g-live-dot h-1.5 w-1.5 rounded-full bg-emerald-500 text-emerald-500"
          aria-hidden="true"
        />
        {{ statusLabel(item.primary_status) }}
      </span>
    </div>

    <!-- 可用率 / 对话延迟 / 端点 PING -->
    <MonitorMetricGrid
      :availability-label="availabilityLabel"
      :availability-value="availabilityValue"
      :latency-ms="item.primary_latency_ms"
      :ping-ms="item.primary_ping_latency_ms"
    />

    <p
      v-if="extraModelsLabel"
      class="mt-2 text-xs text-gray-500 dark:text-gray-400"
    >
      {{ extraModelsLabel }}
    </p>

    <!-- 配额模式：最新用量/余额快照（服务端已按系统开关剥离，此处 flag 为纵深防御） -->
    <MonitorQuotaView v-if="quotaVisible" :snapshot="item.latest_quota" class="mt-2" />

    <!-- 最近检测信号（最后一次检测时间也在这里显示） -->
    <MonitorTimeline
      :buckets="item.timeline"
      :countdown-seconds="countdownSeconds"
      :auto-refresh-on="autoRefreshOn"
    />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView } from '@/api/channelMonitor'
import {
  useChannelMonitorFormat,
  providerGradient,
} from '@/composables/useChannelMonitorFormat'
import { isChannelMonitorQuotaVisible } from '@/utils/featureFlags'
import ProviderIcon from './ProviderIcon.vue'
import MonitorMetricGrid from './MonitorMetricGrid.vue'
import MonitorTimeline from './MonitorTimeline.vue'
import MonitorQuotaView from '@/components/common/MonitorQuotaView.vue'

// 图标配色与 utils/platformColors.ts 的平台色对齐（新 4 家）。
const PROVIDER_TINT: Record<string, string> = {
  openai: 'text-emerald-600 dark:text-emerald-300',
  anthropic: 'text-orange-600 dark:text-orange-300',
  gemini: 'text-sky-600 dark:text-sky-300',
  grok: 'text-zinc-700 dark:text-zinc-200',
  antigravity: 'text-purple-600 dark:text-purple-300',
  kimi: 'text-pink-600 dark:text-pink-300',
  zhipu: 'text-indigo-600 dark:text-indigo-300',
  deepseek: 'text-teal-600 dark:text-teal-300',
  opencode_go: 'text-amber-700 dark:text-amber-300',
}

const props = withDefaults(defineProps<{
  item: UserMonitorView
  window: '7d' | '15d' | '30d'
  availabilityValue: number | null
  countdownSeconds: number
  /**
   * 自动刷新是否开着。默认 true 只为兼容既有调用；页面必须显式传真实开关，
   * 否则会出现"已关闭刷新却显示倒计时"或反之的错位。
   */
  autoRefreshOn?: boolean
}>(), {
  autoRefreshOn: true,
})

const emit = defineEmits<{
  (e: 'click'): void
}>()

const { t } = useI18n()
const {
  statusLabel,
  statusBadgeClass,
  providerLabel,
  providerBadgeClass,
  formatMonitorModel,
} = useChannelMonitorFormat()

const providerTintClass = computed(() =>
  PROVIDER_TINT[props.item.provider] ?? 'text-gray-500 dark:text-gray-300'
)

const quotaVisible = computed(
  () => isChannelMonitorQuotaVisible() && !!props.item.latest_quota
)

const availabilityLabel = computed(() => {
  const win = t(`channelStatus.windowTab.${props.window}`)
  return `${t('monitorCommon.availabilityPrefix')} · ${win}`
})

const extraModelsLabel = computed(() => {
  const count = props.item.extra_models?.length ?? 0
  if (count === 0) return ''
  return t('monitorCommon.extraModelsCount', { n: count })
})
</script>
