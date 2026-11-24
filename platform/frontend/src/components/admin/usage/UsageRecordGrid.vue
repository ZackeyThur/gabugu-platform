<template>
  <div class="usage-record-grid" :aria-busy="loading">
    <div v-if="sortableColumns.length" class="record-grid-sort" :aria-label="t('usageRecordDetails.sort')">
      <span>{{ t('usageRecordDetails.sort') }}</span>
      <button v-for="column in sortableColumns" :key="column.key" type="button" :aria-pressed="sortKey === column.key" @click="sort(column.key)">
        {{ column.label }} <span aria-hidden="true">{{ sortKey === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span>
        <span v-if="sortKey === column.key" class="sr-only">{{ t(sortOrder === 'asc' ? 'usageRecordDetails.ascending' : 'usageRecordDetails.descending') }}</span>
      </button>
    </div>
    <div v-if="loading" class="record-grid-loading" role="status">
      <span class="sr-only">{{ t('common.loading') }}</span>
      <div v-for="n in 3" :key="n" class="overview-skeleton" />
    </div>
    <div v-else-if="!data.length" class="overview-empty"><slot name="empty" /></div>
    <article v-for="row in loading ? [] : data" v-else :key="row.id" class="technical-record" :class="{ 'technical-record--no-source': !sourceColumns.length }" :aria-label="row.model || t('usageOverview.records')">
      <header class="technical-record-header">
        <div v-if="modelColumn || billingModeColumn" class="technical-record-model">
          <span v-if="modelColumn" class="overview-model-logo" aria-hidden="true"><ModelIcon :model="row.model || ''" size="28px" /></span>
          <div class="technical-record-name-and-mode">
            <div v-if="modelColumn" class="technical-field-value" data-field="model"><slot name="cell-model" :row="row" :value="row.model" :expanded="true">{{ row.model }}</slot></div>
            <div v-if="billingModeColumn" class="technical-field-value technical-record-billing" data-field="billing_mode" :aria-label="billingModeColumn.label"><slot name="cell-billing_mode" :row="row" :value="row.billing_mode" :expanded="true">{{ row.billing_mode }}</slot></div>
          </div>
        </div>
        <div class="technical-record-meta">
          <div v-if="timeColumn" data-field="created_at"><span class="sr-only">{{ timeColumn.label }} </span><slot name="cell-created_at" :row="row" :value="row.created_at" :expanded="true">{{ row.created_at }}</slot></div>
          <button type="button" class="overview-link" :aria-label="t('usageRecordDetails.openDetails', { model: row.model || '—', time: recordTime(row.created_at) })" @click="$emit('details', row)">{{ t('usageRecordDetails.details') }} <span class="record-details-arrow" aria-hidden="true"><Icon name="arrowRight" size="xs" :stroke-width="2" /></span></button>
        </div>
      </header>
      <div v-if="groups.length" class="technical-record-fields">
        <section v-for="group in groups" :key="group.key" class="technical-record-group" :data-group="group.key" :aria-label="t(group.label)">
          <dl><div v-for="column in group.columns" :key="column.key" class="technical-field" :data-field="column.key">
            <dt>{{ column.key === 'api_key' ? t('usageOverview.keyLabel') : column.key === 'reasoning_effort' ? t('usageOverview.effortLabel') : column.label }}</dt>
            <dd class="technical-field-value"><slot :name="`cell-${column.key}`" :row="row" :value="value(row, column.key)" :expanded="true">{{ column.formatter ? column.formatter(value(row, column.key), row) : value(row, column.key) }}</slot></dd>
          </div></dl>
        </section>
      </div>
      <dl v-if="sourceColumns.length" class="technical-record-source">
        <div v-for="column in sourceColumns" :key="column.key" :data-field="column.key"><dt>{{ column.label }}</dt><dd class="technical-field-value"><slot :name="`cell-${column.key}`" :row="row" :value="value(row, column.key)" :expanded="true">{{ value(row, column.key) }}</slot></dd></div>
      </dl>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminUsageLog } from '@/types'
import type { Column } from '@/components/common/types'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { recordTime } from '@/utils/usageRecords'

const props = withDefaults(defineProps<{
  data: AdminUsageLog[]
  columns: Column[]
  loading?: boolean
  serverSideSort?: boolean
  defaultSortKey?: string
  defaultSortOrder?: 'asc' | 'desc'
}>(), { loading: false, serverSideSort: true, defaultSortKey: 'created_at', defaultSortOrder: 'desc' })
const emit = defineEmits<{ sort: [key: string, order: 'asc' | 'desc']; details: [row: AdminUsageLog] }>()
const { t } = useI18n()
const sortKey = ref(props.defaultSortKey)
const sortOrder = ref(props.defaultSortOrder)
watch(() => [props.defaultSortKey, props.defaultSortOrder] as const, ([key, order]) => { sortKey.value = key; sortOrder.value = order })
const sortableColumns = computed(() => props.columns.filter(column => column.sortable))
const modelColumn = computed(() => props.columns.find(column => column.key === 'model'))
const billingModeColumn = computed(() => props.columns.find(column => column.key === 'billing_mode'))
const timeColumn = computed(() => props.columns.find(column => column.key === 'created_at'))
const sourceKeys = ['endpoint', 'ip_address', 'user_agent']
const sourceColumns = computed(() => props.columns.filter(column => sourceKeys.includes(column.key)))
const definitions = [
  { key: 'source', label: 'usageOverview.recordSource', keys: ['api_key', 'group'] },
  { key: 'request', label: 'usageOverview.requestSettings', keys: ['reasoning_effort', 'stream'] },
  { key: 'usage', label: 'usageOverview.usageAndCharges', keys: ['cost', 'tokens'] },
  { key: 'performance', label: 'usageOverview.responsePerformance', keys: ['latency'] },
]
const groups = computed(() => definitions.map(group => ({ ...group, columns: group.keys.flatMap(key => props.columns.filter(column => column.key === key)) })).filter(group => group.columns.length))
const value = (row: AdminUsageLog, key: string) => row[key as keyof AdminUsageLog]
function sort(key: string) {
  sortOrder.value = sortKey.value === key && sortOrder.value === 'desc' ? 'asc' : 'desc'
  sortKey.value = key
  emit('sort', key, sortOrder.value)
}
</script>

<style scoped>
.usage-record-grid { min-width: 0; container-type: inline-size; color: var(--g-text); }
.record-grid-sort { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 18px; padding: 10px 20px; border-bottom: 1px solid var(--ov-border); color: var(--ov-muted); font-size: 13px; }
.record-grid-sort button { min-height: 36px; padding: 6px 10px; border-radius: 5px; }
.record-grid-sort button[aria-pressed="true"] { color: var(--ov-blue); background: var(--ov-track); }
.technical-record { --record-columns: repeat(3, minmax(0, 1fr)) minmax(0, 160px); --record-column-gap: 16px; --record-row-gap: 12px; display: flex; flex-direction: column; row-gap: var(--record-row-gap); min-width: 0; min-height: 199px; padding: 10px 24px; }
/* Balance the outer space above the model header and below the final data row. */
.technical-record.technical-record--no-source { padding-block: 24px; }
.technical-record--no-source .technical-record-fields { margin-block: auto 0; }
.technical-record + .technical-record { border-top: 1px solid var(--ov-border); }
.technical-record-header { display: flex; align-items: center; justify-content: space-between; gap: 12px 24px; }
.technical-record-model { min-width: 0; display: flex; align-items: center; gap: 12px; flex: 1; }
.technical-record-model .overview-model-logo { flex-shrink: 0; }
.technical-record-name-and-mode { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; min-width: 0; }
.technical-record-billing { display: inline-flex; flex-shrink: 0; }
.technical-record-meta { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 8px 20px; min-width: 0; font-size: 13px; }
.technical-record-meta > div { overflow-wrap: anywhere; }
.record-details-arrow { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid var(--ov-border); border-radius: 7px; background: var(--ov-track); }
/* Share row tracks across groups and label widths within each group. */
.technical-record-fields { display: grid; grid-template-columns: var(--record-columns); grid-template-rows: minmax(20px, auto) auto; gap: 18px var(--record-column-gap); margin-block: auto; min-height: 82px; }
.technical-record-group { width: 100%; max-width: 266px; min-width: 0; display: grid; grid-row: span 2; grid-template-rows: subgrid; }
.technical-record-group[data-group="source"] { grid-column: 1; }
.technical-record-group[data-group="request"] { grid-column: 2; }
.technical-record-group[data-group="usage"] { grid-column: 3; }
.technical-record-group[data-group="performance"] { grid-column: 4; }
.technical-record-group dl { display: grid; grid-row: span 2; grid-template-rows: subgrid; grid-template-columns: fit-content(45%) minmax(0, 1fr); column-gap: 8px; }
.technical-field { display: grid; grid-column: span 2; grid-template-columns: subgrid; align-items: start; align-content: start; gap: 8px; min-width: 0; font-size: 13px; line-height: 20px; }
.technical-field dt { color: var(--ov-muted); font-size: 13px; line-height: 20px; overflow-wrap: anywhere; }
.technical-field dd { min-width: 0; max-width: 100%; }
.technical-field[data-field="group"], .technical-field[data-field="stream"], .technical-field[data-field="tokens"] { align-self: end; }
.technical-field[data-field="tokens"] { align-items: center; }
.technical-field-value { min-width: 0; max-width: 100%; overflow-wrap: anywhere; white-space: normal; font-variant-numeric: tabular-nums; }
.technical-field-value :deep(.truncate) { overflow: visible; text-overflow: clip; white-space: normal; }
.technical-field-value :deep(.flex) { flex-wrap: wrap; min-width: 0; max-width: 100%; }
.technical-field-value :deep(.text-xs) { font-size: 13px; }
.technical-field-value :deep(.whitespace-nowrap) { white-space: normal; }
.technical-field-value :deep(.inline-flex.items-center.rounded:not(.rounded-full)) { padding-block: 1px; line-height: 16px; }
.technical-field[data-field="tokens"] :deep(.usage-token-values) { display: grid; grid-template-columns: minmax(0, max-content) minmax(0, max-content); gap: 1px 8px; }
.technical-field[data-field="tokens"] :deep(.usage-token-row) { display: grid; grid-column: span 2; grid-template-columns: subgrid; column-gap: 8px; margin-top: 0; }
.technical-field[data-field="tokens"] :deep(.usage-token-row > .inline-flex) { flex-wrap: wrap; min-width: 0; }
.technical-field[data-field="tokens"] :deep(.usage-token-write) { grid-column: 2; }
.technical-field[data-field="tokens"] :deep(.usage-token-row svg) { width: 14px; height: 14px; flex-shrink: 0; }
.technical-field[data-field="latency"] .technical-field-value :deep(.grid) { line-height: 20px; row-gap: 11px; }
.technical-record-model :deep(.font-medium) { font-size: 14px; }
.technical-field[data-field="latency"] { display: block; grid-row: span 2; }
.technical-field[data-field="latency"] dt { display: none; }
.technical-field[data-field="latency"] dd, .technical-field[data-field="latency"] dd > :deep(div) { height: 100%; }
.technical-field[data-field="latency"] :deep(.grid) { grid-template-columns: minmax(0, auto) minmax(0, 1fr); align-content: space-between; }
.technical-record-source { display: grid; grid-template-columns: var(--record-columns); align-items: start; gap: 6px var(--record-column-gap); padding-top: 8px; border-top: 1px solid var(--ov-border); }
.technical-record-source > [data-field="endpoint"] { grid-column: 1; }
.technical-record-source > [data-field="ip_address"] { grid-column: 2; }
.technical-record-source > [data-field="user_agent"] { grid-column: 3 / span 2; }
.technical-record-source > div { display: flex; gap: 6px; align-items: baseline; min-width: 0; line-height: 16px; }
.technical-record-source dt { color: var(--ov-muted); font-size: 13px; line-height: 16px; flex-shrink: 0; }
.technical-record-source dd { min-width: 0; }
.technical-record-source dd :deep(.text-xs) { line-height: 16px; }
.technical-record-source [data-field="ip_address"] dd > :deep(div) { display: flex; flex-wrap: wrap; align-items: baseline; gap: 2px 10px; }
.technical-record-source [data-field="ip_address"] dd :deep(.mt-0\.5) { margin-top: 0; }
.record-grid-loading { padding: 24px; display: grid; gap: 20px; }.record-grid-loading > div { height: 180px; }
@container (max-width: 900px) {
  .technical-record { --record-columns: minmax(0, 1fr) minmax(0, 1.1fr) minmax(0, 1.55fr) minmax(0, 1.15fr); --record-column-gap: 12px; --record-row-gap: 8px; }
  /* The row details control remains available for the full token cost breakdown. */
  .technical-field[data-field="tokens"] dd :deep(.group.relative) { display: none; }
}
@container (max-width: 500px) {
  .technical-record { padding: 10px 18px; --record-row-gap: 12px; }
  .technical-record-header { flex-direction: column; align-items: flex-start; }
  .technical-record-meta { justify-content: flex-start; }
  .technical-record-fields { grid-template-columns: minmax(0, 1fr); grid-template-rows: none; gap: 12px; min-height: 0; }
  .technical-record-group[data-group] { max-width: none; grid-column: auto; grid-row: auto; grid-template-rows: none; }
  .technical-record-group dl { grid-row: auto; grid-template-rows: none; grid-template-columns: 72px minmax(0, 1fr); row-gap: 12px; }
  .technical-field[data-field="latency"] { grid-row: auto; }
  .technical-field[data-field="latency"] :deep(.flex.items-stretch) { position: relative; }
  .technical-field[data-field="latency"] :deep(.w-1) { position: absolute; inset-block: 0; inset-inline-start: -8px; }
  .technical-field[data-field="latency"] :deep(.grid) { grid-template-columns: 72px minmax(0, 1fr); }
  .technical-record-source { grid-template-columns: minmax(0, 1fr); }
  .technical-record-source > div[data-field] { display: grid; grid-column: auto; grid-template-columns: 72px minmax(0, 1fr); gap: 8px; width: 100%; }
  .record-grid-sort { padding: 10px 18px; gap: 8px; }
  .record-grid-sort button, .technical-record-meta button { min-height: 44px; }
}
</style>
