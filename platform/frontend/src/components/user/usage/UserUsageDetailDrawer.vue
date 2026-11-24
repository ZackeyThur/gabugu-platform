<template>
  <DashboardDrawer :open="!!record" :title="t('usageRecordDetails.title')" @close="emit('close')">
    <div v-if="record" class="usage-detail">
      <div class="detail-model"><span class="detail-logo" aria-hidden="true"><ModelIcon :model="record.model || ''" size="32px" /></span><div><h3>{{ record.model || '—' }}</h3><p>{{ recordTime(record.created_at) }} · UTC+8</p></div></div>
      <section class="detail-section detail-charge" :aria-label="t('usageRecordDetails.charges')">
        <p class="detail-muted">{{ t('usageRecordDetails.actual') }}</p>
        <strong class="detail-amount">{{ recordMoney(record.actual_cost, true) }}</strong>
        <p v-if="record.billing_type === 1" class="detail-note">{{ t('usageRecordDetails.subscriptionNote') }}</p>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.standard') }}</dt><dd>{{ recordMoney(record.total_cost, true) }}</dd></div>
          <div><dt>{{ t(discount != null && discount < 0 ? 'usageRecordDetails.aboveStandard' : 'usageRecordDetails.discount') }}</dt><dd>{{ recordMoney(discount == null ? null : Math.abs(discount), true) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.billingType') }}</dt><dd>{{ billingType }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.billingMode') }}</dt><dd>{{ recordBillingMode(record, t) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.multiplier') }}</dt><dd>{{ isRecordedNumber(record.rate_multiplier) ? `${formatMultiplier(record.rate_multiplier)}×` : '—' }}</dd></div>
        </dl>
        <p class="detail-note">{{ t('usageRecordDetails.comparisonNote') }}</p>
      </section>

      <section v-if="record.token_campaign" class="detail-section" data-token-campaign :aria-labelledby="`${sectionId}-campaign`">
        <h3 :id="`${sectionId}-campaign`">{{ t('usageRecordDetails.campaignTitle') }}</h3>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.campaignPhase') }}</dt><dd>{{ campaignPhase }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.campaignOriginal') }}</dt><dd>{{ recordMoney(record.token_campaign.original_actual_cost, true) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.campaignSavings') }}</dt><dd>{{ recordMoney(record.token_campaign.discount_amount, true) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.actual') }}</dt><dd>{{ recordMoney(record.actual_cost, true) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.campaignFactor') }}</dt><dd>{{ formatMultiplier(record.token_campaign.discount) }}×</dd></div>
          <div><dt>{{ t('usageRecordDetails.campaignPricingAt') }}</dt><dd>{{ recordTime(record.token_campaign.pricing_at) }} · UTC+8</dd></div>
          <div><dt>{{ t('usageRecordDetails.campaignVersion') }}</dt><dd>{{ record.token_campaign.id }} · {{ record.token_campaign.revision }}</dd></div>
        </dl>
        <p class="detail-note">{{ t('usageRecordDetails.campaignEvidenceNote') }}</p>
      </section>

      <section class="detail-section" :aria-labelledby="`${sectionId}-tokens`">
        <h3 :id="`${sectionId}-tokens`">{{ t('usageRecordDetails.tokensAndCosts') }}</h3>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.inputTotal') }}</dt><dd>{{ recordNumber(record.input_tokens) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.outputTotal') }}</dt><dd>{{ recordNumber(record.output_tokens) }}</dd></div>
        </dl>
        <table class="detail-token-table">
          <caption class="sr-only">{{ t('usageRecordDetails.tokenCaption') }}</caption>
          <thead><tr><th scope="col">{{ t('usageRecordDetails.part') }}</th><th scope="col">Token</th><th scope="col">{{ t('usageRecordDetails.cost') }}</th></tr></thead>
          <tbody><tr v-for="part in tokenParts" :key="part.key"><th scope="row">{{ t(`usageRecordDetails.${part.key}`) }}</th><td>{{ recordNumber(part.tokens) }}</td><td><span class="detail-part-cost">{{ recordMoney(part.cost, true) }}</span><small v-if="tokenBilling" class="detail-part-price">{{ t('usageRecordDetails.perMillion') }}<span>{{ recordMoney(part.price, true) }}</span></small></td></tr></tbody>
        </table>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.cache5m') }}</dt><dd>{{ recordNumber(record.cache_creation_5m_tokens) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.cache1h') }}</dt><dd>{{ recordNumber(record.cache_creation_1h_tokens) }}</dd></div>
        </dl>
        <p class="detail-note">{{ t('usageRecordDetails.tokenNote') }}</p>
        <p v-if="tokenBilling" class="detail-note">{{ t('usageRecordDetails.priceNote') }}</p>
      </section>

      <section v-if="hasImageDetails" class="detail-section" :aria-labelledby="`${sectionId}-image`">
        <h3 :id="`${sectionId}-image`">{{ t('usageRecordDetails.imageDetails') }}</h3>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.imageCount') }}</dt><dd>{{ recordNumber(record.image_count) }}</dd></div>
          <div><dt>{{ t('usage.imageBillingSize') }}</dt><dd>{{ formatImageBillingSize(record, t) }}</dd></div>
          <div><dt>{{ t('usage.imageSizeSource') }}</dt><dd>{{ formatImageSizeSource(record, t) }}</dd></div>
          <div><dt>{{ t('usage.imageInputSize') }}</dt><dd>{{ formatImageInputSize(record, t) }}</dd></div>
          <div><dt>{{ t('usage.imageOutputSize') }}</dt><dd>{{ formatImageOutputSize(record, t) }}</dd></div>
          <div v-if="formatImageSizeBreakdown(record)"><dt>{{ t('usage.imageSizeBreakdown') }}</dt><dd>{{ formatImageSizeBreakdown(record) }}</dd></div>
          <div v-if="isImageUsage(record)"><dt>{{ t('usageRecordDetails.imageStandardUnit') }}</dt><dd>{{ isRecordedNumber(record.total_cost) ? recordMoney(imageUnitPrice(record), true) : '—' }}</dd></div>
        </dl>
      </section>
      <p v-if="displayMode === 'video'" class="detail-note">{{ t('usageRecordDetails.videoNote') }}</p>

      <section class="detail-section" :aria-labelledby="`${sectionId}-request`">
        <h3 :id="`${sectionId}-request`">{{ t('usageRecordDetails.requestInfo') }}</h3>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.requestType') }}</dt><dd>{{ recordRequestType(record, t) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.serviceTier') }}</dt><dd>{{ record.service_tier ? getUsageServiceTierLabel(record.service_tier, t) : '—' }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.reasoningEffort') }}</dt><dd>{{ record.reasoning_effort ? formatReasoningEffort(record.reasoning_effort) : '—' }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.longContext') }}</dt><dd>{{ flag(record.long_context_billing_applied) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.cacheOverride') }}</dt><dd>{{ flag(record.cache_ttl_overridden) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.nativeCompaction') }}</dt><dd>{{ flag(record.native_compaction_v2) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.duration') }}</dt><dd>{{ recordDuration(record.duration_ms) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.firstToken') }}</dt><dd>{{ recordDuration(record.first_token_ms) }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.outputRate') }}</dt><dd>{{ outputRate }}</dd></div>
        </dl>
        <p class="detail-note">{{ t('usageRecordDetails.outputRateNote') }}</p>
        <p class="detail-note">{{ t('usageRecordDetails.firstTokenNote') }}</p>
      </section>

      <section class="detail-section" :aria-labelledby="`${sectionId}-source`">
        <h3 :id="`${sectionId}-source`">{{ t('usageRecordDetails.source') }}</h3>
        <dl class="detail-list">
          <div><dt>{{ t('usageRecordDetails.apiKey') }}</dt><dd>{{ record.api_key?.name || '—' }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.group') }}</dt><dd>{{ record.group?.name || '—' }}</dd></div>
          <div><dt>{{ t('usageRecordDetails.endpoint') }}</dt><dd>{{ record.inbound_endpoint || '—' }}</dd></div>
          <div><dt>IP</dt><dd>{{ record.ip_address || '—' }}<IpGeoCell v-if="record.ip_address" :ip="record.ip_address" /></dd></div>
          <div><dt>User-Agent</dt><dd>{{ record.user_agent || '—' }}</dd></div>
          <div class="request-id-row"><dt>Request ID</dt><dd><code>{{ record.request_id || '—' }}</code><button v-if="record.request_id" type="button" class="overview-link" @click="copyId">{{ t(copyState === 'done' ? 'usageRecordDetails.copied' : 'usageRecordDetails.copy') }}</button><span v-if="copyState === 'error'" role="alert" class="detail-note">{{ t('usageRecordDetails.copyFailed') }}</span><span v-else-if="copyState === 'done'" role="status" class="sr-only">{{ t('usageRecordDetails.copied') }}</span></dd></div>
        </dl>
      </section>
    </div>
  </DashboardDrawer>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageLog } from '@/types'
import DashboardDrawer from '@/components/user/dashboard/DashboardDrawer.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import IpGeoCell from '@/components/common/IpGeoCell.vue'
import { formatMultiplier } from '@/utils/formatters'
import { formatReasoningEffort } from '@/utils/format'
import { getDisplayBillingMode, imageUnitPrice, isImageUsage } from '@/utils/billingMode'
import { formatImageBillingSize, formatImageInputSize, formatImageOutputSize, formatImageSizeBreakdown, formatImageSizeSource } from '@/utils/imageUsage'
import { getUsageServiceTierLabel } from '@/utils/usageServiceTier'
import { formatUsageOutputRate } from '@/utils/latencyHealth'
import { isRecordedNumber, recordBillingMode, recordDiscount, recordDuration, recordMoney, recordNumber, recordRequestType, recordTime, recordTokenParts } from '@/utils/usageRecords'

const props = defineProps<{ record: UsageLog | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const sectionId = useId()
const copyState = ref<'idle' | 'done' | 'error'>('idle')
watch(() => props.record, () => { copyState.value = 'idle' })
const discount = computed(() => props.record ? recordDiscount(props.record) : null)
const campaignPhase = computed(() => {
  const phase = props.record?.token_campaign?.phase
  return phase === 'championship-day' ? t('usageRecordDetails.campaignDay')
    : phase === 'celebration-week' ? t('usageRecordDetails.campaignWeek') : phase || '—'
})
const displayMode = computed(() => getDisplayBillingMode(props.record))
const tokenBilling = computed(() => displayMode.value === 'token')
const tokenParts = computed(() => props.record ? recordTokenParts(props.record) : [])
const hasImageDetails = computed(() => !!props.record && displayMode.value !== 'video' && (props.record.image_count > 0 || props.record.image_input_tokens > 0 || props.record.image_output_tokens > 0 || displayMode.value === 'image'))
const outputRate = computed(() => props.record && displayMode.value !== 'video' ? formatUsageOutputRate(props.record) : '—')
const billingType = computed(() => props.record?.billing_type === 1 ? t('usageRecordDetails.subscriptionShort') : props.record?.billing_type === 0 ? t('usageRecordDetails.balance') : '—')
const flag = (value: unknown) => typeof value === 'boolean' ? t(value ? 'usageRecordDetails.yes' : 'usageRecordDetails.no') : '—'
async function copyId() {
  const requestId = props.record?.request_id
  if (!requestId) return
  try {
    await navigator.clipboard.writeText(requestId)
    if (props.record?.request_id === requestId) copyState.value = 'done'
  } catch {
    if (props.record?.request_id === requestId) copyState.value = 'error'
  }
}
</script>

<style scoped>
.usage-detail { color: inherit; min-width: 0; }
.detail-model { display: flex; align-items: center; gap: 14px; margin-bottom: 28px; }.detail-model > div { min-width: 0; }
.detail-model h3 { font-size: 16px; line-height: 1.7; font-weight: 600; overflow-wrap: anywhere; }.detail-model p { font-size: 13px; color: var(--ov-muted); margin-top: 5px; line-height: 1.8; }
.detail-logo { display: inline-flex; background: var(--ov-track, transparent); border: 1px solid var(--ov-border); border-radius: 9px; padding: 8px; flex-shrink: 0; }
.detail-logo :deep(.model-icon-fallback) { background: #384a67; }
.detail-section { border-top: 1px solid var(--ov-border); padding-top: 22px; margin-top: 24px; }.detail-section > h3 { font-size: 14px; font-weight: 600; margin-bottom: 18px; }
.detail-charge { border-top: 0; padding-top: 0; }.detail-muted { color: var(--ov-muted); font-size: 13px; }
.detail-amount { font-size: clamp(24px, 6vw, 34px); font-weight: 600; letter-spacing: -.8px; display: block; margin: 8px 0 18px; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
.detail-list { display: flex; flex-direction: column; gap: 15px; margin: 16px 0; }.detail-list > div { display: grid; grid-template-columns: minmax(90px, .85fr) minmax(0, 1.3fr); gap: 18px; font-size: 13px; line-height: 1.7; }
.detail-list dt { color: var(--ov-muted); }.detail-list dd { text-align: right; min-width: 0; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
.detail-note { color: var(--ov-muted); font-size: 13px; line-height: 1.9; margin-top: 12px; }
.detail-token-table { width: 100%; table-layout: fixed; font-size: 13px; border-collapse: collapse; margin-top: 20px; font-variant-numeric: tabular-nums; }
.detail-token-table th, .detail-token-table td { padding: 10px 4px; text-align: right; overflow-wrap: anywhere; border-bottom: 1px solid var(--ov-border); vertical-align: top; font-weight: 400; }
.detail-token-table thead { color: var(--ov-muted); }.detail-token-table th:first-child { text-align: left; width: 27%; }.detail-token-table th:nth-child(2) { width: 24%; }.detail-token-table tbody th { color: var(--ov-muted); }
.detail-part-cost { display: block; white-space: nowrap; }.detail-part-price { display: block; color: var(--ov-muted); font-size: 13px; margin-top: 8px; }.detail-part-price span { display: block; white-space: nowrap; margin-top: 3px; }
.request-id-row code { font-size: 13px; }.request-id-row button { display: flex; margin-left: auto; }
</style>
