<template>
  <div class="usage-records">
    <div v-if="loading" class="records-state" role="status" :aria-label="t('usageRecordDetails.loading')">
      <span class="sr-only">{{ t('usageRecordDetails.loading') }}</span>
      <div v-for="n in 5" :key="n" class="records-skeleton" aria-hidden="true" />
    </div>
    <div v-else-if="error" class="records-state records-message" role="alert">
      <strong>{{ t('usageRecordDetails.loadError') }}</strong>
      <p>{{ t('usageRecordDetails.loadErrorHelp') }}</p>
      <button type="button" class="overview-secondary-button" @click="emit('retry')">{{ t('usageRecordDetails.retry') }}</button>
    </div>
    <div v-else-if="rows.length === 0" class="records-state records-message">
      <strong>{{ t('usageRecordDetails.empty') }}</strong>
      <p>{{ t('usageRecordDetails.emptyHelp') }}</p>
      <button type="button" class="overview-secondary-button" @click="emit('reset')">{{ t('usageRecordDetails.reset') }}</button>
    </div>
    <template v-else>
    <div class="mobile-sorts" :aria-label="t('usageRecordDetails.sort')">
      <button type="button" class="overview-link" @click="sort('created_at')">{{ t('usageRecordDetails.time') }} <span aria-hidden="true">{{ sortArrow('created_at') }}</span><span class="sr-only">{{ sortDirection('created_at') }}</span></button>
      <button type="button" class="overview-link" @click="sort('model')">{{ t('usageRecordDetails.model') }} <span aria-hidden="true">{{ sortArrow('model') }}</span><span class="sr-only">{{ sortDirection('model') }}</span></button>
    </div>
    <table class="records-table">
      <caption class="sr-only">{{ t('usageRecordDetails.recordsCaption') }}</caption>
      <colgroup><col class="time-col"><col class="model-col"><col class="key-col"><col class="cost-col"><col class="detail-col"></colgroup>
      <thead><tr>
        <th scope="col" :aria-sort="ariaSort('created_at')"><button type="button" @click="sort('created_at')">{{ t('usageRecordDetails.time') }} <span aria-hidden="true">{{ sortArrow('created_at') }}</span></button></th>
        <th scope="col" :aria-sort="ariaSort('model')"><button type="button" @click="sort('model')">{{ t('usageRecordDetails.model') }} <span aria-hidden="true">{{ sortArrow('model') }}</span></button></th>
        <th scope="col">{{ t('usageRecordDetails.apiKey') }}</th>
        <th scope="col" class="records-cost">{{ t('usageRecordDetails.actual') }}</th>
        <th scope="col"><span class="sr-only">{{ t('usageRecordDetails.details') }}</span></th>
      </tr></thead>
      <tbody><tr v-for="row in rows" :key="row.id">
        <td class="records-time"><span class="mobile-label">{{ t('usageRecordDetails.time') }}</span><time :datetime="row.created_at">{{ recordTime(row.created_at) }}</time></td>
        <td class="records-model"><div class="model-identity"><span class="model-logo" aria-hidden="true"><ModelIcon :model="row.model || ''" size="32px" /></span><div><strong>{{ row.model || '—' }}</strong><p>{{ recordBillingMode(row, t) }} · {{ recordRequestType(row, t) }}</p></div></div></td>
        <td class="records-key"><span class="mobile-label">{{ t('usageRecordDetails.apiKey') }}</span>{{ row.api_key?.name || '—' }}</td>
        <td class="records-cost"><span class="mobile-label">{{ t('usageRecordDetails.actual') }}</span><strong>{{ actualCost(row.actual_cost) }}</strong><small v-if="row.token_campaign" data-campaign-badge>{{ t('usageRecordDetails.campaignBadge') }}</small><small v-if="row.billing_type === 1">{{ t('usageRecordDetails.subscriptionShort') }}</small></td>
        <td class="records-detail"><button type="button" class="overview-link" :aria-label="t('usageRecordDetails.openDetails', { model: row.model || '—', time: recordTime(row.created_at) })" @click="selected = row">{{ t('usageRecordDetails.details') }} <span aria-hidden="true">↗</span></button></td>
      </tr></tbody>
    </table>
    </template>
    <UserUsageDetailDrawer :record="selected" @close="selected = null" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageLog } from '@/types'
import ModelIcon from '@/components/common/ModelIcon.vue'
import UserUsageDetailDrawer from './UserUsageDetailDrawer.vue'
import { isRecordedNumber, recordBillingMode, recordRequestType, recordTime } from '@/utils/usageRecords'

const props = defineProps<{ rows: UsageLog[]; loading: boolean; error: boolean; sortBy: string; sortOrder: 'asc' | 'desc' }>()
const emit = defineEmits<{ sort: [key: string, order: 'asc' | 'desc']; retry: []; reset: [] }>()
const { t } = useI18n()
const selected = ref<UsageLog | null>(null)
const actualCost = (value: unknown): string => !isRecordedNumber(value) ? '—' : value > 0 && value < 0.0001 ? '<$0.0001' : `$${value.toFixed(4)}`
function sort(key: string) { emit('sort', key, props.sortBy === key && props.sortOrder === 'desc' ? 'asc' : 'desc') }
const ariaSort = (key: string) => props.sortBy !== key ? 'none' : props.sortOrder === 'asc' ? 'ascending' : 'descending'
const sortArrow = (key: string) => props.sortBy !== key ? '↕' : props.sortOrder === 'asc' ? '↑' : '↓'
const sortDirection = (key: string) => props.sortBy === key ? t(props.sortOrder === 'asc' ? 'usageRecordDetails.ascending' : 'usageRecordDetails.descending') : ''
watch(() => props.rows, rows => { if (selected.value) selected.value = rows.find(row => row.id === selected.value?.id) ?? null })
</script>

<style scoped>
.usage-records {
  position: relative;
  min-width: 0;
  border-radius: 20px;
  background-image: var(--ui-card-grad);
  box-shadow: var(--ui-shadow-card);
  overflow: hidden;
}
.usage-records::before {
  content: '';
  position: absolute;
  top: 0;
  left: 20px;
  right: 20px;
  height: 1px;
  background: var(--ui-rim-light);
  pointer-events: none;
  z-index: 1;
}
.mobile-sorts { display: none; }
.records-table { width: 100%; table-layout: fixed; border-collapse: collapse; }
.time-col { width: 21%; }.model-col { width: 34%; }.key-col { width: 20%; }.cost-col { width: 16%; }.detail-col { width: 9%; }
.records-table th { color: var(--ov-muted); font-size: 13px; font-weight: 500; text-align: left; background: var(--ov-thead-bg, rgba(246, 238, 227, 0.48)); }
.records-table th, .records-table td { padding: 18px 20px; border-bottom: 1px solid var(--ui-hairline); overflow-wrap: anywhere; }
.records-table th button { display: inline-flex; align-items: center; gap: 10px; min-height: 32px; text-align: left; transition: color 150ms ease; }
.records-table th button:hover { color: var(--ov-blue); }.records-table tr:last-child td { border-bottom: 0; }
.records-table tbody tr { transition: background-color 160ms ease; }
.records-table tbody tr:hover { background: color-mix(in srgb, var(--ov-blue) 4.5%, transparent); }
.records-time { font-size: 13px; color: var(--ov-muted); font-variant-numeric: tabular-nums; line-height: 1.8; }
.model-identity { display: flex; gap: 12px; align-items: center; min-width: 0; }.model-identity > div { min-width: 0; }
.model-logo { display: inline-flex; flex-shrink: 0; }
.model-identity strong { font-size: 14px; font-weight: 550; line-height: 1.6; }.model-identity p { color: var(--ov-muted); font-size: 13px; margin-top: 4px; line-height: 1.7; }
.records-key { font-size: 13px; color: var(--ov-muted); }.records-table .records-cost { text-align: right; font-variant-numeric: tabular-nums; }
.records-cost strong { font-size: 16px; font-weight: 600; }.records-cost small { display: block; color: var(--ov-muted); margin-top: 4px; font-size: 13px; }
.records-detail { text-align: right; }.records-detail button { white-space: nowrap; }.mobile-label { display: none; }
.records-state { padding: 28px; }.records-message { display: flex; align-items: center; flex-direction: column; gap: 12px; padding: 54px 20px; text-align: center; }
.records-message strong { font-size: 16px; }.records-message p { color: var(--ov-muted); line-height: 1.7; font-size: 13px; }
.records-skeleton { height: 50px; border-radius: 7px; margin: 12px 0; background: var(--ov-track); animation: record-pulse 1.5s ease-in-out infinite alternate; }
@keyframes record-pulse { to { opacity: .45; } }
@media (max-width: 1000px) and (min-width: 701px) { .records-table th, .records-table td { padding: 14px 10px; }.detail-col { width: 12%; }.model-col { width: 31%; } }
@media (max-width: 700px) {
  .mobile-sorts { display: flex; gap: 24px; padding: 8px 20px; border-bottom: 1px solid var(--ov-border); }
  .records-table, .records-table tbody { display: block; }.records-table colgroup, .records-table thead { display: none; }
  .records-table tr { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 14px 16px; padding: 20px; border-bottom: 1px solid var(--ov-border); }
  .records-table tr:last-child { border-bottom: 0; }.records-table td { display: block; border: 0; padding: 0; min-width: 0; }
  .records-model { grid-column: 1 / -1; grid-row: 1; }.records-time { grid-column: 1; grid-row: 3; }.records-key { grid-column: 1; grid-row: 2; }
  .records-table .records-cost { grid-column: 2; grid-row: 2; }.records-detail { grid-column: 2; grid-row: 3; align-self: end; }
  .mobile-label { display: block; font-size: 13px; color: var(--ov-muted); margin-bottom: 4px; }.records-cost strong { font-size: 18px; }
}
@media (prefers-reduced-motion: reduce) { .records-skeleton { animation: none; } }
</style>
