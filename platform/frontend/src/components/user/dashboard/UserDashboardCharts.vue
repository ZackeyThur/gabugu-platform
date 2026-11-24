<template>
  <section aria-labelledby="analysis-heading">
    <div class="overview-section-heading overview-analysis-heading">
      <div>
        <h2 id="analysis-heading">{{ t('dashboard.overview.analysis') }}</h2>
        <p>{{ startDate }} — {{ endDate }} · UTC+8</p>
      </div>

      <div class="overview-analysis-controls">
        <!-- Custom Styled Key & Group Filters (non-simple mode) -->
        <div v-if="!isSimple && (filterApiKeys.length > 0 || filterGroups.length > 0)" class="overview-filter-selects">
          <div
            v-if="filterApiKeys.length > 0"
            class="overview-filter-select-wrap"
            :class="{ 'is-filtered': apiKeyId !== null }"
          >
            <Select
              :model-value="apiKeyId ?? ''"
              :options="apiKeyOptions"
              :aria-label="t('dashboard.overview.filterByApiKey')"
              :searchable="filterApiKeys.length > 6"
              @update:model-value="onApiKeyChange"
            />
          </div>

          <div
            v-if="filterGroups.length > 0"
            class="overview-filter-select-wrap"
            :class="{ 'is-filtered': groupId !== null }"
          >
            <Select
              :model-value="groupId ?? ''"
              :options="groupOptions"
              :aria-label="t('dashboard.overview.filterByGroup')"
              :searchable="filterGroups.length > 6"
              @update:model-value="onGroupChange"
            />
          </div>

          <button
            v-if="apiKeyId !== null || groupId !== null"
            type="button"
            class="overview-filter-reset"
            @click="resetFilters"
          >
            {{ t('dashboard.overview.resetFilters') }}
          </button>
        </div>

        <div
          class="overview-periods g-segmented"
          :aria-label="t('dashboard.timeRange')"
          :style="{ '--seg-count': 4, '--seg-index': presetIndex }"
        >
          <span class="g-segmented-thumb" aria-hidden="true" />
          <button type="button" :aria-pressed="preset === 1" @click="chooseToday">
            {{ t('dashboard.overview.todayHourly') }}
          </button>
          <button type="button" :aria-pressed="preset === 7" @click="choosePreset(7)">
            {{ t('dashboard.last7Days') }}
          </button>
          <button type="button" :aria-pressed="preset === 30" @click="choosePreset(30)">
            {{ t('dashboard.overview.last30Days') }}
          </button>
          <button
            type="button"
            :aria-pressed="preset === 'custom'"
            :aria-expanded="customOpen"
            aria-controls="overview-custom-range"
            @click="customOpen = !customOpen"
          >
            {{ t('dashboard.overview.custom') }}
          </button>
        </div>
      </div>
    </div>

    <Transition name="overview-range-drop">
      <form v-if="customOpen" id="overview-custom-range" class="overview-custom-range" @submit.prevent="applyCustom">
        <label>{{ t('dates.startDate') }}<input v-model="draftStart" type="date" :max="draftEnd || today" required /></label>
        <label>{{ t('dates.endDate') }}<input v-model="draftEnd" type="date" :min="draftStart" :max="today" required /></label>
        <div class="overview-custom-range-actions">
          <button class="overview-secondary-button overview-range-cancel" type="button" @click="customOpen = false">
            {{ t('common.cancel') }}
          </button>
          <button class="overview-primary-button overview-range-apply" type="submit">
            {{ t('dates.apply') }}
          </button>
        </div>
        <p v-if="rangeError" class="overview-custom-range-error" role="alert">{{ t('dashboard.overview.invalidRange') }}</p>
      </form>
    </Transition>

    <div class="overview-surface overview-analysis-grid overview-analysis-studio" data-no-spotlight>
      <DashboardTrendChart
        :start-date="startDate"
        :end-date="endDate"
        :today="today"
        :granularity="granularity"
        :trend="trend"
        :loading="trendLoading"
        :error="trendError"
        :api-key-id="apiKeyId"
        :group-id="groupId"
        @retry="$emit('retryTrend')"
      />
      <DashboardBreakdownChart
        :start-date="startDate"
        :end-date="endDate"
        :models="models"
        :groups="groups"
        :api-keys="apiKeys"
        :loading="modelsLoading"
        :error="modelsError"
        :api-key-id="apiKeyId"
        :group-id="groupId"
        @retry="$emit('retryModels')"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import DashboardTrendChart from './DashboardTrendChart.vue'
import DashboardBreakdownChart from './DashboardBreakdownChart.vue'
import type { ApiKey, APIKeyStat, Group, GroupStat, ModelStat, TrendDataPoint } from '@/types'
import { shiftDate } from '@/utils/dashboardOverview'

const props = withDefaults(defineProps<{
  startDate: string
  endDate: string
  today: string
  granularity?: 'day' | 'hour'
  trend: TrendDataPoint[]
  models: ModelStat[]
  groups?: GroupStat[]
  apiKeys?: APIKeyStat[]
  trendLoading: boolean
  modelsLoading: boolean
  trendError: boolean
  modelsError: boolean
  isSimple?: boolean
  apiKeyId?: number | null
  groupId?: number | null
  filterApiKeys?: ApiKey[]
  filterGroups?: Group[]
}>(), {
  granularity: 'day',
  groups: () => [],
  apiKeys: () => [],
  isSimple: false,
  apiKeyId: null,
  groupId: null,
  filterApiKeys: () => [],
  filterGroups: () => [],
})

const emit = defineEmits<{
  rangeChange: [start: string, end: string, granularity?: 'day' | 'hour']
  filterChange: [filters: { apiKeyId?: number | null; groupId?: number | null }]
  retryTrend: []
  retryModels: []
}>()

const { t } = useI18n()
const customOpen = ref(false)
const rangeError = ref(false)
const draftStart = ref(props.startDate)
const draftEnd = ref(props.endDate)

const apiKeyOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('dashboard.overview.allApiKeys') },
  ...props.filterApiKeys.map((key) => ({ value: key.id, label: key.name })),
])

const groupOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('dashboard.overview.allGroups') },
  ...props.filterGroups.map((group) => ({ value: group.id, label: group.name })),
])

const preset = computed(() => {
  if (props.endDate === props.today && props.startDate === props.today) return 1
  if (props.endDate === props.today && props.startDate === shiftDate(props.today, -6)) return 7
  if (props.endDate === props.today && props.startDate === shiftDate(props.today, -29)) return 30
  return 'custom'
})

const presetIndex = computed(() => {
  if (preset.value === 1) return 0
  if (preset.value === 7) return 1
  if (preset.value === 30) return 2
  return 3
})

watch(() => [props.startDate, props.endDate], () => {
  draftStart.value = props.startDate
  draftEnd.value = props.endDate
})

function chooseToday() {
  customOpen.value = false
  rangeError.value = false
  emit('rangeChange', props.today, props.today, 'hour')
}

function choosePreset(days: number) {
  customOpen.value = false
  rangeError.value = false
  emit('rangeChange', shiftDate(props.today, -(days - 1)), props.today, 'day')
}

function applyCustom() {
  const length = (Date.parse(draftEnd.value) - Date.parse(draftStart.value)) / 86_400_000
  rangeError.value = !Number.isFinite(length) || length < 0 || length >= 366 || draftEnd.value > props.today
  if (!rangeError.value) {
    emit('rangeChange', draftStart.value, draftEnd.value, draftStart.value === draftEnd.value ? 'hour' : 'day')
    customOpen.value = false
  }
}

function onApiKeyChange(val: string | number | boolean | null) {
  const num = val === '' || val === null || val === undefined ? null : Number(val)
  emit('filterChange', { apiKeyId: Number.isFinite(num) && (num ?? 0) > 0 ? num : null })
}

function onGroupChange(val: string | number | boolean | null) {
  const num = val === '' || val === null || val === undefined ? null : Number(val)
  emit('filterChange', { groupId: Number.isFinite(num) && (num ?? 0) > 0 ? num : null })
}

function resetFilters() {
  emit('filterChange', { apiKeyId: null, groupId: null })
}
</script>
