<template>
  <section
    class="group/section ui-soft-card card-hover overflow-hidden"
    :style="{ '--section-accent': platformAccentColor(group.platform) }"
  >
    <!-- Delicate feathered provider hairline crest -->
    <div class="section-top-rail relative z-[2] h-[2px] w-full" aria-hidden="true" />

    <!-- 分组头部 -->
    <header class="border-b border-gray-200/55 px-5 py-[18px] sm:px-6 dark:border-white/[0.08]">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <GroupBadge
            :name="group.name"
            :platform="group.platform as GroupPlatform"
            :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
            :rate-multiplier="group.rate_multiplier"
            :user-rate-multiplier="group.user_rate_multiplier ?? null"
            :peak-rate-enabled="group.peak_rate_enabled"
            :peak-start="group.peak_start"
            :peak-end="group.peak_end"
            :peak-rate-multiplier="group.peak_rate_multiplier"
            always-show-rate
          />
          <span
            v-if="savingsPercent > 0"
            class="inline-flex items-center rounded-full bg-clay-50 px-2.5 py-0.5 font-mono text-xs font-semibold tabular-nums text-clay-700 ring-1 ring-inset ring-clay-200/80 dark:bg-emerald-500/[0.13] dark:text-[#38D3B4] dark:ring-emerald-400/25"
          >
            -{{ savingsPercent }}%
          </span>
          <span
            v-if="group.is_exclusive"
            class="inline-flex items-center gap-1 rounded-full bg-purple-50 px-2.5 py-0.5 text-xs font-medium text-purple-600 dark:bg-purple-900/20 dark:text-purple-400"
          >
            <Icon name="shield" size="xs" class="h-3 w-3" />
            {{ t('modelPlaza.badges.exclusive') }}
          </span>
          <span
            v-if="group.subscription_type === 'subscription'"
            class="inline-flex items-center rounded-full bg-violet-50 px-2.5 py-0.5 text-xs font-medium text-violet-600 dark:bg-violet-900/20 dark:text-violet-400"
          >
            {{ t('modelPlaza.badges.subscription') }}
          </span>
        </div>

        <span
          v-if="group.models.length > 0"
          class="font-mono text-xs tabular-nums text-gray-400 dark:text-dark-400"
        >
          {{ group.models.length }} {{ t('modelPlaza.filters.modelLabel') }}
        </span>
      </div>

      <p v-if="group.description" class="mt-2.5 text-sm leading-relaxed text-gray-600 dark:text-dark-300">
        {{ group.description }}
      </p>

      <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
        <span>{{ t('modelPlaza.detail.standardPriceNote') }}</span>
        <span v-if="tokenCampaignDiscount < 1" class="font-medium text-primary-600 dark:text-primary-400" data-campaign-price>
          {{ t('usageRecordDetails.campaignPrice', { discount: tokenCampaignDiscount }) }}
        </span>
        <span v-else-if="phase === 'unavailable'" class="text-gray-400 dark:text-dark-500">
          {{ t('usageRecordDetails.campaignPriceUnavailable') }}
        </span>
      </div>

      <details v-if="group.additional_charges" class="mt-2 text-xs text-gray-500 dark:text-dark-400" data-additional-charges>
        <summary class="cursor-pointer font-medium transition-colors hover:text-gray-700 dark:hover:text-gray-200">{{ t('modelPlaza.detail.additionalCharges') }}</summary>
        <p class="mt-1">{{ t('modelPlaza.detail.additionalChargesHint') }}</p>
        <ul class="mt-1 space-y-1 font-mono">
          <li>{{ t('modelPlaza.detail.webSearch') }}: ${{ group.additional_charges.web_search_per_call }} / {{ t('modelPlaza.detail.call') }}</li>
          <li>{{ t('modelPlaza.detail.searchTools') }}: ${{ group.additional_charges.search_per_1k }} / 1000 {{ t('modelPlaza.detail.call') }}</li>
          <li>{{ t('modelPlaza.detail.realtimeAudio') }}: ${{ group.additional_charges.audio_realtime_per_min }} / min</li>
          <li>TTS: ${{ group.additional_charges.audio_tts_per_million_chars }} / {{ t('modelPlaza.detail.millionChars') }}</li>
          <li>STT: ${{ group.additional_charges.audio_stt_per_hour }} / h</li>
        </ul>
      </details>
      <p
        v-if="peakNote"
        class="mt-1.5 inline-flex items-center gap-1 text-xs font-medium text-amber-700 dark:text-amber-400"
      >
        <Icon name="clock" size="xs" class="h-3 w-3" />
        {{ peakNote }}
      </p>
      <p
        v-if="longContextNote"
        class="mt-1.5 flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400"
      >
        <Icon name="infoCircle" size="xs" class="h-3 w-3" />
        {{ longContextNote }}
      </p>
    </header>

    <!-- 模型价格表 -->
    <div>
      <PlazaModelPricingTable
        v-if="group.models.length > 0"
        :models="group.models"
        :token-campaign-discount="tokenCampaignDiscount"
        :platform="group.platform"
        :rate-multiplier="group.rate_multiplier"
        :user-rate-multiplier="group.user_rate_multiplier ?? null"
        :image-rate-independent="group.image_rate_independent"
        :image-rate-multiplier="group.image_rate_multiplier"
        :video-rate-independent="group.video_rate_independent"
        :video-rate-multiplier="group.video_rate_multiplier"
        :peak-window="peakWindow"
        :peak-rate-multiplier="group.peak_rate_multiplier"
      />
      <p v-else class="px-5 py-6 text-center text-sm text-gray-400 dark:text-dark-500">
        {{ t('modelPlaza.detail.noModels') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlazaModelPricingTable from './PlazaModelPricingTable.vue'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformAccentColor } from '@/utils/platformColors'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { useBillingCampaign } from '@/composables/useBillingCampaign'
import { useAppStore } from '@/stores/app'

const props = defineProps<{
  group: ModelPlazaGroup
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { phase, discountForGroup } = useBillingCampaign()
const tokenCampaignDiscount = computed(() => discountForGroup(props.group.id))

const savingsPercent = computed(() => {
  const rate = props.group.user_rate_multiplier ?? props.group.rate_multiplier
  if (!Number.isFinite(rate) || rate >= 1 || rate <= 0) return 0
  return Math.round((1 - rate) * 100)
})

/** 高峰窗口描述(含倍率与服务器时区标注);分组未启用高峰为空串。 */
const peakWindow = computed(() => {
  if (!hasPeakRate(props.group)) return ''
  return formatPeakRateWindow(
    props.group,
    serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)
  )
})

const peakNote = computed(() => {
  if (!peakWindow.value) return ''
  return t('modelPlaza.detail.peakNote', {
    window: peakWindow.value,
    multiplier: props.group.peak_rate_multiplier
  })
})

/**
 * 分组关闭了长上下文阶梯、但组内有模型官方带阶梯时提示:实付列只展示基础档,
 * 官方阶梯仅供参考。字段缺失(旧后端)不提示。
 */
const longContextNote = computed(() => {
  if (props.group.long_context_pricing_enabled !== false) return ''
  const hasOfficialLadder = props.group.models.some(
    (m) => (m.official_pricing?.intervals?.length ?? 0) > 1
  )
  return hasOfficialLadder ? t('modelPlaza.detail.longContextDisabledNote') : ''
})
</script>

<style scoped>
.section-top-rail {
  background: linear-gradient(
    90deg,
    var(--section-accent) 0%,
    color-mix(in srgb, var(--section-accent) 45%, transparent) 62%,
    transparent 100%
  );
  opacity: 0.85;
}
.py-4\.5 {
  padding-top: 1.125rem;
  padding-bottom: 1.125rem;
}
</style>
