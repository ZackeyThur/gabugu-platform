<template>
  <AppLayout>
    <div class="dashboard-overview usage-overview">
      <header class="overview-header">
        <div><h1 class="g-blur-in">{{ t('usageOverview.title') }}</h1><p>{{ t('usageOverview.subtitle') }}</p></div>
        <div class="overview-header-actions">
          <button type="button" data-magnet data-spark class="overview-secondary-button" :disabled="busy" @click="refreshData"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button>
          <button v-if="activeTab === 'usage'" type="button" data-magnet class="overview-primary-button usage-export-btn" :disabled="exporting || loading || logsError || pagination.total === 0" :title="t('usageOverview.exportHint')" @click="exportToCSV"><Icon name="download" size="sm" />{{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}</button>
        </div>
      </header>

      <section data-no-spotlight class="overview-surface usage-filters" :aria-label="t('usageOverview.filterTitle')">
        <div class="usage-range-row">
          <UserUsageDateRange :start-date="startDate" :end-date="endDate" @change="onDateRangeChange" />
          <p class="usage-range-caption">{{ startDate }}<template v-if="endDate !== startDate"> — {{ endDate }}</template><span>UTC+8</span></p>
        </div>
        <div v-if="errorViewEnabled" class="usage-tabs" :aria-label="t('usageOverview.records')">
          <button type="button" :aria-pressed="activeTab === 'usage'" @click="activeTab = 'usage'">{{ t('usage.tabs.usage') }}</button>
          <button type="button" :aria-pressed="activeTab === 'errors'" @click="switchToErrors">{{ t('usage.tabs.errors') }}</button>
        </div>
        <template v-if="activeTab === 'usage'">
          <div class="usage-main-filters">
            <div><label for="usage-key" class="input-label">{{ t('usage.apiKeyFilter') }}</label><Select id="usage-key" v-model="filters.api_key_id" :aria-label="t('usage.apiKeyFilter')" :placeholder="t('usage.allApiKeys')" :options="apiKeyOptions" searchable @change="applyFilters" /></div>
            <div><label for="usage-model" class="input-label">{{ t('usage.model') }}</label><Select id="usage-model" v-model="filters.model" :aria-label="t('usage.model')" :placeholder="t('admin.usage.allModels')" :options="modelOptions" :loading="modelStatsLoading" searchable @change="applyFilters" /></div>
            <div class="usage-filter-actions"><button type="button" class="overview-secondary-button" :aria-expanded="moreFilters" aria-controls="usage-more-filters" @click="moreFilters = !moreFilters"><Icon name="filter" size="sm" />{{ t(moreFilters ? 'usageOverview.lessFilters' : 'usageOverview.moreFilters') }}<span v-if="activeFilterCount" class="usage-filter-count">{{ activeFilterCount }}</span></button><button type="button" class="overview-link" @click="resetFilters">{{ t('common.reset') }}</button></div>
          </div>
          <div v-show="moreFilters" id="usage-more-filters" class="usage-extra-filters">
            <div><label for="usage-platform" class="input-label">{{ t('usageOverview.platformFilter') }}</label><Select id="usage-platform" v-model="filters.platform" :aria-label="t('usageOverview.platformFilter')" :options="platformOptions" @change="applyFilters" /></div>
            <div><label for="usage-group" class="input-label">{{ t('admin.usage.group') }}</label><Select id="usage-group" v-model="filters.group_id" :aria-label="t('admin.usage.group')" :options="groupOptions" searchable @change="applyFilters" /></div>
            <div><label for="usage-type" class="input-label">{{ t('usage.type') }}</label><Select id="usage-type" v-model="filters.request_type" :aria-label="t('usage.type')" :options="requestTypeOptions" @change="applyFilters" /></div>
            <div><label for="usage-compaction" class="input-label">{{ t('usage.compactionFilter') }}</label><Select id="usage-compaction" v-model="filters.native_compaction_v2" :aria-label="t('usage.compactionFilter')" :options="compactionOptions" @change="applyFilters" /></div>
            <div v-if="subscriptionFeatureEnabled"><label for="usage-billing" class="input-label">{{ t('admin.usage.billingType') }}</label><Select id="usage-billing" v-model="filters.billing_type" :aria-label="t('admin.usage.billingType')" :options="billingTypeOptions" @change="applyFilters" /></div>
            <div><label for="usage-mode" class="input-label">{{ t('admin.usage.billingMode') }}</label><Select id="usage-mode" v-model="filters.billing_mode" :aria-label="t('admin.usage.billingMode')" :options="billingModeOptions" @change="applyFilters" /></div>
          </div>
        </template>
        <div v-else class="usage-error-filters">
          <div><label for="error-key" class="input-label">{{ t('usage.errors.keyName') }}</label><Select id="error-key" v-model="errorFilter.api_key_id" :aria-label="t('usage.errors.keyName')" :options="errorKeyOptions" searchable @change="applyErrorFilters" /></div>
          <div><label for="error-model" class="input-label">{{ t('usage.errors.model') }}</label><Select id="error-model" v-model="errorFilter.model" :aria-label="t('usage.errors.model')" :options="errorModelOptions" searchable creatable clearable :placeholder="t('usage.errors.modelPlaceholder')" @change="applyErrorFilters" /></div>
          <div><label for="error-category" class="input-label">{{ t('usage.errors.category') }}</label><Select id="error-category" v-model="errorFilter.category" :aria-label="t('usage.errors.category')" :options="errorCategoryOptions" @change="applyErrorFilters" /></div>
          <div><label for="error-status" class="input-label">{{ t('usage.errors.status') }}</label><Select id="error-status" v-model="errorFilter.status_code" :aria-label="t('usage.errors.status')" :options="errorStatusOptions" @change="applyErrorFilters" /></div>
          <button type="button" class="overview-link" @click="resetFilters">{{ t('common.reset') }}</button>
        </div>
        <p v-if="filterOptionsError || modelOptionsError" class="usage-options-error" role="alert">{{ t('usageOverview.filterError') }} <button type="button" class="overview-link" @click="retryOptions">{{ t('common.retry') }}</button></p>
      </section>

      <UserUsageSummary v-if="activeTab === 'usage'" :stats="usageStats" :loading="statsLoading" :error="statsError" @retry="loadStats" />

      <section class="usage-records-section">
        <header class="overview-section-heading">
          <div><h2>{{ t(activeTab === 'usage' ? 'usageOverview.records' : 'usage.tabs.errors') }}<span v-if="!busy && !(activeTab === 'usage' ? logsError : errorListError)" class="usage-count">{{ (activeTab === 'usage' ? pagination.total : errorTotal).toLocaleString() }}</span></h2><p>{{ t(activeTab === 'usage' ? 'usageOverview.listHint' : 'usageOverview.errorHint') }}</p></div>
          <div class="usage-table-actions">
            <div v-if="activeTab === 'usage'" class="overview-periods g-segmented" :style="{ '--seg-count': 2, '--seg-index': viewMode === 'technical' ? 1 : 0 }" :aria-label="t('usageOverview.viewLabel')"><span class="g-segmented-thumb" aria-hidden="true" /><button type="button" :aria-pressed="viewMode === 'simple'" @click="viewMode = 'simple'">{{ t('usageOverview.simpleView') }}</button><button type="button" :aria-pressed="viewMode === 'technical'" @click="viewMode = 'technical'">{{ t('usageOverview.technicalView') }}</button></div>
            <div v-if="activeTab === 'errors' || viewMode === 'technical'" ref="columnDropdownRef" class="relative">
              <button type="button" data-testid="usage-column-settings" class="overview-secondary-button" :aria-expanded="showColumnDropdown" @click="showColumnDropdown = !showColumnDropdown"><Icon name="grid" size="sm" />{{ t('admin.users.columnSettings') }}</button>
              <div v-if="showColumnDropdown" class="usage-column-menu" @keydown.esc.stop="showColumnDropdown = false">
                <button v-for="col in currentToggleableColumns" :key="col.key" type="button" :data-testid="`usage-column-toggle-${col.key}`" :aria-pressed="isCurrentColumnVisible(col.key)" @click="toggleCurrentColumn(col.key)"><span>{{ col.label }}</span><Icon v-if="isCurrentColumnVisible(col.key)" name="check" size="sm" /></button>
              </div>
            </div>
          </div>
        </header>
        <template v-if="activeTab === 'usage'">
          <UserUsageRecordsTable v-if="viewMode === 'simple'" :rows="usageLogs" :loading="loading" :error="logsError" :sort-by="sortState.sort_by" :sort-order="sortState.sort_order" @sort="handleSort" @retry="loadLogs" @reset="resetFilters" />
          <div v-else-if="logsError" class="overview-surface overview-empty" role="alert">{{ t('usageOverview.listError') }}<button class="overview-link" @click="loadLogs">{{ t('common.retry') }}</button></div>
          <UsageTable v-else :key="sortState.sort_by + sortState.sort_order" :data="loading ? [] : usageLogs" :loading="loading" :columns="visibleColumns" :record-layout="true" :server-side-sort="true" :show-account-billing="false" :show-upstream-endpoint="false" :default-sort-key="sortState.sort_by" :default-sort-order="sortState.sort_order" :timezone="DASHBOARD_TIMEZONE" @sort="handleSort" @details="technicalRecord = $event" @ipGeoBatchFailed="handleIpGeoBatchFailed" />
          <Pagination v-if="pagination.total > 0 && !loading && !logsError" :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" />
        </template>
        <template v-else-if="errorViewEnabled">
          <div v-if="errorListError" class="overview-surface overview-empty" role="alert">{{ t('usageOverview.errorsError') }}<button class="overview-link" @click="loadErrors">{{ t('common.retry') }}</button></div>
          <UserErrorRequestsTable v-else :rows="errorLoading ? [] : errorRows" :total="errorLoading ? 0 : errorTotal" :loading="errorLoading" :page="errorPage" :page-size="errorPageSize" :visible-column-keys="errVisibleColumnKeys" :timezone="DASHBOARD_TIMEZONE" @sort="onErrorSort" @update:page="onErrorPage" @update:pageSize="onErrorPageSize" @ipGeoBatchFailed="handleIpGeoBatchFailed" />
        </template>
      </section>
      <UserUsageDetailDrawer :record="technicalRecord" @close="technicalRecord = null" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { keysAPI, usageAPI, userGroupsAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'


import UsageTable from '@/components/admin/usage/UsageTable.vue'




import Icon from '@/components/icons/Icon.vue'
import UserUsageDateRange from '@/components/user/usage/UserUsageDateRange.vue'
import UserUsageSummary from '@/components/user/usage/UserUsageSummary.vue'
import UserUsageRecordsTable from '@/components/user/usage/UserUsageRecordsTable.vue'
import UserUsageDetailDrawer from '@/components/user/usage/UserUsageDetailDrawer.vue'
import { CONCRETE_PLATFORM_OPTIONS, DASHBOARD_TIMEZONE, dashboardToday } from '@/utils/dashboardOverview'
import '@/styles/dashboard-overview.css'
import '@/styles/usage-overview.css'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatReasoningEffort } from '@/utils/format'
import { getBillingModeLabel, getDisplayBillingMode as resolveDisplayBillingMode } from '@/utils/billingMode'
import { resolveUsageRequestType, requestTypeToLegacyStream } from '@/utils/usageRequestType'
import type {
  ApiKey,
  Group,
  ModelStat,
  UsageLog,
  UsageQueryParams,
  UsageStatsResponse,
  UserErrorRequest,
} from '@/types'
import type { Column } from '@/components/common/types'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const preferenceUserId = useAuthStore().user?.id
const preferenceKey = (key: string) => preferenceUserId == null ? null : `${key}:${preferenceUserId}`
const savePreference = (key: string, value: string) => {
  const scopedKey = preferenceKey(key)
  if (!scopedKey) return
  try { localStorage.setItem(scopedKey, value) } catch { /* Preferences are optional. */ }
}


const usageStats = ref<UsageStatsResponse | null>(null)
const usageLogs = ref<UsageLog[]>([])

const loading = ref(false)
const logsError = ref(false)
const statsLoading = ref(false)
const statsError = ref(false)
const modelOptionsError = ref(false)
const filterOptionsError = ref(false)
const errorListError = ref(false)
const moreFilters = ref(false)
const VIEW_MODE_KEY = 'user-usage-view-mode'
const viewMode = ref<'simple' | 'technical'>('simple')
try {
  const key = preferenceKey(VIEW_MODE_KEY)
  const saved = key ? localStorage.getItem(key) : null
  if (saved === 'simple' || saved === 'technical') viewMode.value = saved
} catch { /* Keep the first-visit default when storage is unavailable. */ }
watch(viewMode, value => savePreference(VIEW_MODE_KEY, value))
const technicalRecord = ref<UsageLog | null>(null)
const modelStatsLoading = ref(false)
const exporting = ref(false)
const errorRows = ref<UserErrorRequest[]>([])
const errorLoading = ref(false)
const errorPage = ref(1)
const errorPageSize = ref(20)
const errorSortBy = ref('created_at')
const errorSortOrder = ref<'asc' | 'desc'>('desc')
const errorTotal = ref(0)
const errorFilter = ref<{ model: string | null; category: string; api_key_id: number | null; status_code: number | null }>({
  model: '',
  category: '',
  api_key_id: null,
  status_code: null,
})

const errorKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allKeys') },
  ...apiKeys.value.map((k) => ({ value: k.id, label: k.name })),
])

// 模型候选取自当前已加载错误中出现过的模型；creatable 允许输入任意片段做后端模糊。
const errorModelOptions = computed<SelectOption[]>(() => {
  const seen = new Set<string>()
  const opts: SelectOption[] = []
  for (const r of errorRows.value) {
    if (r.model && !seen.has(r.model)) {
      seen.add(r.model)
      opts.push({ value: r.model, label: r.model })
    }
  }
  return opts
})

const errorCategoryCodes = ['auth', 'rate_limit', 'quota', 'invalid_request', 'service_unavailable', 'upstream', 'internal', 'cyber']

const errorCategoryOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('usage.errors.allCategories') },
  ...errorCategoryCodes.map((c) => ({ value: c, label: t('usage.errors.categories.' + c) })),
])

// 状态码候选用固定常用列表(与管理端 UsageFilters 共用常量),不受当前页数据限制:
// 后端 status_code 过滤对全量生效,若只列当前页出现过的码,用户就选不到仅在后续页的码。
const errorStatusOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allStatuses') },
  ...COMMON_ERROR_STATUS_CODES.map((c) => ({ value: c, label: String(c) })),
])

const applyErrorFilters = () => {
  errorPage.value = 1
  void loadErrors()
}

let abortController: AbortController | null = null
let errorReqSeq = 0
let filterReqSeq = 0
let disposed = false
let statsReqSeq = 0
let modelStatsReqSeq = 0

const startDate = ref(dashboardToday())
const endDate = ref(dashboardToday())
const activeTab = ref<'usage' | 'errors'>('usage')
const errorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false)

const filters = ref<UsageQueryParams>({
  start_date: startDate.value,
  end_date: endDate.value,
  platform: undefined,
  request_type: undefined,
  native_compaction_v2: null,
  billing_type: null,
  billing_mode: null,
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc',
})

const requestTypeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allTypes') },
  { value: 'ws_v2', label: t('usage.ws') },
  { value: 'live', label: t('usage.live') },
  { value: 'stream', label: t('usage.stream') },
  { value: 'sync', label: t('usage.sync') },
])
const compactionOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.allCompactionTypes') },
  { value: true, label: t('usage.compactionOnly') },
])
// 订阅功能关闭后只剩余额计费，「计费类型」筛选（余额/订阅）失去意义，整块隐藏。
const subscriptionFeatureEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.subscription))
const billingTypeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allBillingTypes') },
  { value: 0, label: t('admin.usage.billingTypeBalance') },
  { value: 1, label: t('admin.usage.billingTypeSubscription') },
])
const billingModeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allBillingModes') },
  { value: 'token', label: t('admin.usage.billingModeToken') },
  { value: 'per_request', label: t('admin.usage.billingModePerRequest') },
  { value: 'image', label: t('admin.usage.billingModeImage') },
  { value: 'video', label: t('admin.usage.billingModeVideo') },
])

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const modelOptionValues = ref<string[]>([])

const platformOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usageOverview.allPlatforms') },
  ...CONCRETE_PLATFORM_OPTIONS.map((p) => ({ value: p.value, label: p.label })),
])

const apiKeyOptions = computed<SelectOption[]>(() => {
  const opts: SelectOption[] = [
    { value: null, label: t('usage.allApiKeys') },
    ...apiKeys.value.map((key) => ({ value: key.id, label: key.name })),
  ]
  const selectedId = filters.value.api_key_id
  if (typeof selectedId === 'number' && selectedId > 0 && !opts.some((opt) => opt.value === selectedId)) {
    opts.push({ value: selectedId, label: `#${selectedId}` })
  }
  return opts
})
const groupOptions = computed<SelectOption[]>(() => {
  const opts: SelectOption[] = [
    { value: null, label: t('admin.usage.allGroups') },
    ...groups.value.map((group) => ({ value: group.id, label: group.name })),
  ]
  const selectedId = filters.value.group_id
  if (typeof selectedId === 'number' && selectedId > 0 && !opts.some((opt) => opt.value === selectedId)) {
    opts.push({ value: selectedId, label: `#${selectedId}` })
  }
  return opts
})
const modelOptions = computed<SelectOption[]>(() => {
  const set = new Set<string>(modelOptionValues.value)
  if (filters.value.model) set.add(filters.value.model)
  return [
    { value: null, label: t('admin.usage.allModels') },
    ...Array.from(set).sort().map((model) => ({ value: model, label: model })),
  ]
})

const normalizedFilters = computed<UsageQueryParams>(() => {
  const requestType = filters.value.request_type
  const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
  const platform = typeof filters.value.platform === 'string' && filters.value.platform.trim()
    ? filters.value.platform.trim()
    : undefined
  return {
    ...filters.value,
    platform,
    timezone: DASHBOARD_TIMEZONE,
    start_date: startDate.value,
    end_date: endDate.value,
    stream: legacyStream === null ? undefined : legacyStream,
  }
})

const buildUsageListParams = (page: number, pageSize: number): UsageQueryParams => ({
  page,
  page_size: pageSize,
  ...normalizedFilters.value,
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order,
})

const loadLogs = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  logsError.value = false
  try {
    const res = await usageAPI.query(buildUsageListParams(pagination.page, pagination.page_size), {
      signal: controller.signal,
    })
    if (!controller.signal.aborted) {
      usageLogs.value = res.items
      pagination.total = res.total
    }
  } catch (error: any) {
    if (abortController === controller && !controller.signal.aborted && error?.name !== 'AbortError' && error?.code !== 'ERR_CANCELED') {
      logsError.value = true
      usageLogs.value = []
      pagination.total = 0
    }
  } finally {
    if (abortController === controller) loading.value = false
  }
}

const loadStats = async () => {
  const seq = ++statsReqSeq
  statsLoading.value = true
  statsError.value = false
  try {
    const stats = await usageAPI.getStats(normalizedFilters.value)
    if (seq !== statsReqSeq || disposed) return
    usageStats.value = stats
  } catch {
    if (seq !== statsReqSeq || disposed) return
    statsError.value = true
    usageStats.value = null
  } finally {
    if (seq === statsReqSeq) statsLoading.value = false
  }
}

// Model options span the date range; selecting a model must not remove its siblings.
const loadModelStats = async () => {
  const seq = ++modelStatsReqSeq
  modelStatsLoading.value = true
  modelOptionsError.value = false
  try {
    const response = await usageAPI.getDashboardModels({
      start_date: startDate.value, end_date: endDate.value,
      timezone: DASHBOARD_TIMEZONE, model_source: 'requested',
    })
    if (seq !== modelStatsReqSeq || disposed) return
    refreshModelOptions(response.models || [])
  } catch {
    if (seq !== modelStatsReqSeq || disposed) return
    modelOptionsError.value = true
  } finally {
    if (seq === modelStatsReqSeq) modelStatsLoading.value = false
  }
}

const refreshModelOptions = (models: ModelStat[]) => {
  const current = filters.value.model
  const set = new Set<string>()
  models.forEach((item) => {
    if (item.model) set.add(item.model)
  })
  if (current) set.add(current)
  modelOptionValues.value = Array.from(set).sort()
}

const applyFilters = () => {
  pagination.page = 1
  void loadLogs()
  void loadStats()
}

const refreshData = () => {
  if (activeTab.value === 'errors') { void loadErrors(); return }
  void loadLogs()
  void loadStats()
  void loadModelStats()
}

const resetFilters = () => {
  moreFilters.value = false
  const range = { start: dashboardToday(), end: dashboardToday() }
  startDate.value = range.start
  endDate.value = range.end
  filters.value = {
    start_date: range.start,
    end_date: range.end,
    platform: undefined,
    request_type: undefined,
    native_compaction_v2: null,
    billing_type: null,
    billing_mode: null,
  }
  applyFilters()
  void loadModelStats()
  if (activeTab.value === 'errors') {
    errorFilter.value = { model: '', category: '', api_key_id: null, status_code: null }
  }
  resetErrorRows()
}

const onDateRangeChange = (range: { startDate: string; endDate: string; preset: string | null }) => {
  startDate.value = range.startDate
  endDate.value = range.endDate
  filters.value.start_date = range.startDate
  filters.value.end_date = range.endDate
  applyFilters()
  void loadModelStats()
  resetErrorRows()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  void loadLogs()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadLogs()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadLogs()
}

const handleIpGeoBatchFailed = () => {
  appStore.showError(t('usage.ipGeo.batchFailed'))
}

const getRequestTypeExportText = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'Cyber'
  if (requestType === 'live') return 'Live'
  if (requestType === 'ws_v2') return 'WS'
  if (requestType === 'stream') return 'Stream'
  if (requestType === 'sync') return 'Sync'
  return 'Unknown'
}

const getDisplayBillingMode = (
  row: Pick<UsageLog, 'billing_mode' | 'image_count'> | null | undefined
): string | null | undefined => resolveDisplayBillingMode(row)

const escapeCSVValue = (value: unknown): string => {
  if (value == null) return ''
  const str = String(value)
  const escaped = str.replace(/"/g, '""')
  if (str === '-') return str
  if (/^[=+\-@\t\r]/.test(str)) return `"\'${escaped}"`
  if (/[,"\n\r]/.test(str)) return `"${escaped}"`
  return str
}

const EXPORT_PAGE_SIZE = 1000
const EXPORT_RATE_LIMIT_MAX_RETRIES = 3
const EXPORT_RATE_LIMIT_MAX_WAIT_SECONDS = 60

// 导出逐页请求计入按用户的重查询限流；429 时按 Retry-After 等待后重试同一页。
const queryExportPage = async (params: UsageQueryParams) => {
  for (let attempt = 0; ; attempt++) {
    try {
      return await usageAPI.query(params)
    } catch (error: any) {
      if (error?.status !== 429 || attempt >= EXPORT_RATE_LIMIT_MAX_RETRIES) throw error
      const waitSeconds = Math.min(
        Math.max(error.retry_after_seconds ?? EXPORT_RATE_LIMIT_MAX_WAIT_SECONDS, 1),
        EXPORT_RATE_LIMIT_MAX_WAIT_SECONDS
      )
      appStore.showInfo(t('usage.exportRateLimited', { seconds: waitSeconds }))
      await new Promise((resolve) => setTimeout(resolve, waitSeconds * 1000))
    }
  }
}

const exportToCSV = async () => {
  if (exporting.value || loading.value || logsError.value) return
  if (pagination.total === 0) {
    appStore.showWarning(t('usage.noDataToExport'))
    return
  }
  exporting.value = true
  appStore.showInfo(t('usage.preparingExport'))
  try {
    const allLogs: UsageLog[] = []
    const exportParams = buildUsageListParams(1, EXPORT_PAGE_SIZE)
    const totalPages = Math.ceil(pagination.total / EXPORT_PAGE_SIZE)
    for (let page = 1; page <= totalPages; page++) {
      const response = await queryExportPage({ ...exportParams, page })
      allLogs.push(...response.items)
    }
    if (allLogs.length === 0) {
      appStore.showWarning(t('usage.noDataToExport'))
      return
    }
    const headers = [
      'Time',
      'API Key Name',
      'Model',
      'Reasoning Effort',
      'Inbound Endpoint',
      'IP Address',
      'Type',
      'Billing Mode',
      'Input Tokens',
      'Output Tokens',
      'Cache Read Tokens',
      'Cache Creation Tokens',
      'Rate Multiplier',
      'Billed Cost',
      'Original Cost',
      'First Token (ms)',
      'Duration (ms)',
    ]
    const rows = allLogs.map((log) => [
      log.created_at,
      log.api_key?.name || '',
      log.model,
      formatReasoningEffort(log.reasoning_effort),
      log.inbound_endpoint || '',
      log.ip_address || '',
      getRequestTypeExportText(log),
      getBillingModeLabel(getDisplayBillingMode(log), t),
      log.input_tokens,
      log.output_tokens,
      log.cache_read_tokens,
      log.cache_creation_tokens,
      log.rate_multiplier,
      log.actual_cost.toFixed(8),
      log.total_cost.toFixed(8),
      log.first_token_ms ?? '',
      log.duration_ms ?? '',
    ].map(escapeCSVValue))
    const csvContent = [
      headers.map(escapeCSVValue).join(','),
      ...rows.map((row) => row.join(',')),
    ].join('\n')
    const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage_${exportParams.start_date}_to_${exportParams.end_date}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('usage.exportSuccess'))
  } catch (error) {
    console.error('CSV Export failed:', error)
    appStore.showError(t('usage.exportFailed'))
  } finally {
    exporting.value = false
  }
}

const ALWAYS_VISIBLE = ['created_at']
const DEFAULT_HIDDEN_COLUMNS = ['endpoint', 'ip_address', 'user_agent']
const HIDDEN_COLUMNS_KEY = 'user-usage-hidden-columns'

const allColumns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'model', label: t('usage.model'), sortable: true },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), sortable: false },
  { key: 'endpoint', label: t('usage.endpoint'), sortable: false },
  { key: 'ip_address', label: 'IP', sortable: false },
  { key: 'group', label: t('admin.usage.group'), sortable: false },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), sortable: false },
  { key: 'tokens', label: t('usage.tokens'), sortable: false },
  { key: 'cost', label: t('usage.cost'), sortable: false },
  { key: 'latency', label: t('usage.latency'), sortable: false },
  { key: 'created_at', label: t('usage.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false },
])

const hiddenColumns = reactive<Set<string>>(new Set())
const toggleableColumns = computed(() => allColumns.value.filter((col) => !ALWAYS_VISIBLE.includes(col.key)))
const visibleColumns = computed(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE.includes(col.key) || !hiddenColumns.has(col.key))
)
const isColumnVisible = (key: string) => !hiddenColumns.has(key)
const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) hiddenColumns.delete(key)
  else hiddenColumns.add(key)
  savePreference(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
}
const loadColumnPreference = (key: string, defaults: string[]): string[] => {
  const scopedKey = preferenceKey(key)
  if (!scopedKey) return defaults
  try {
    const scoped = localStorage.getItem(scopedKey)
    const saved = scoped ?? localStorage.getItem(key)
    if (saved === null) return defaults
    const values: unknown = JSON.parse(saved)
    if (!Array.isArray(values) || !values.every(value => typeof value === 'string')) return defaults
    if (scoped === null) {
      // Transfer the old browser-wide choice to this account only.
      localStorage.setItem(scopedKey, JSON.stringify(values))
      localStorage.removeItem(key)
    }
    return values
  } catch {
    return defaults
  }
}
const loadSavedColumns = () => {
  loadColumnPreference(HIDDEN_COLUMNS_KEY, DEFAULT_HIDDEN_COLUMNS).forEach(key => hiddenColumns.add(key))
}

// 错误请求 tab 独立列设置(机制同用量列设置,存储互不影响)
const ERR_ALWAYS_VISIBLE = ['status', 'created_at']
const ERR_DEFAULT_HIDDEN_COLUMNS = ['endpoint', 'client_ip', 'group', 'type', 'platform', 'user_agent']
const ERR_HIDDEN_COLUMNS_KEY = 'user-usage-error-hidden-columns'

// key 须与 UserErrorRequestsTable 的 allColumns 一致
const errAllColumns = computed<Column[]>(() => [
  { key: 'key_name', label: t('usage.errors.keyName') },
  { key: 'model', label: t('usage.errors.model') },
  { key: 'endpoint', label: t('usage.errors.endpoint') },
  { key: 'client_ip', label: 'IP' },
  { key: 'group', label: t('admin.usage.group') },
  { key: 'type', label: t('usage.type') },
  { key: 'platform', label: t('usage.errors.platform') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('usage.errors.status') },
  { key: 'message', label: t('usage.errors.message') },
  { key: 'created_at', label: t('usage.errors.time') },
  { key: 'user_agent', label: t('usage.userAgent') },
])

const errHiddenColumns = reactive<Set<string>>(new Set())
const errToggleableColumns = computed(() =>
  errAllColumns.value.filter((col) => !ERR_ALWAYS_VISIBLE.includes(col.key))
)
const errVisibleColumnKeys = computed(() =>
  errAllColumns.value
    .filter((col) => ERR_ALWAYS_VISIBLE.includes(col.key) || !errHiddenColumns.has(col.key))
    .map((col) => col.key)
)
const isErrColumnVisible = (key: string) => !errHiddenColumns.has(key)
const toggleErrColumn = (key: string) => {
  if (errHiddenColumns.has(key)) errHiddenColumns.delete(key)
  else errHiddenColumns.add(key)
  savePreference(ERR_HIDDEN_COLUMNS_KEY, JSON.stringify([...errHiddenColumns]))
}
const loadSavedErrColumns = () => {
  loadColumnPreference(ERR_HIDDEN_COLUMNS_KEY, ERR_DEFAULT_HIDDEN_COLUMNS).forEach(key => errHiddenColumns.add(key))
}

// 列设置下拉按当前 tab 分发
const currentToggleableColumns = computed(() =>
  activeTab.value === 'errors' ? errToggleableColumns.value : toggleableColumns.value
)
const isCurrentColumnVisible = (key: string) =>
  activeTab.value === 'errors' ? isErrColumnVisible(key) : isColumnVisible(key)
const toggleCurrentColumn = (key: string) => {
  if (activeTab.value === 'errors') toggleErrColumn(key)
  else toggleColumn(key)
}

const showColumnDropdown = ref(false)
const columnDropdownRef = ref<HTMLElement | null>(null)
const handleColumnClickOutside = (event: MouseEvent) => {
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(event.target as HTMLElement)) {
    showColumnDropdown.value = false
  }
}

const loadApiKeys = async () => {
  const firstPage = await keysAPI.list(1, 100)
  const keys = [...firstPage.items]
  for (let page = 2; page <= firstPage.pages && keys.length > 0; page++) {
    const response = await keysAPI.list(page, 100)
    if (response.items.length === 0) break
    keys.push(...response.items)
  }
  return keys
}

const loadFilterOptions = async () => {
  const seq = ++filterReqSeq
  filterOptionsError.value = false
  const [keys, availableGroups] = await Promise.allSettled([loadApiKeys(), userGroupsAPI.getAvailable()])
  if (seq !== filterReqSeq || disposed) return
  if (keys.status === 'fulfilled') apiKeys.value = keys.value
  if (availableGroups.status === 'fulfilled') groups.value = availableGroups.value
  filterOptionsError.value = keys.status === 'rejected' || availableGroups.status === 'rejected'
}

const retryOptions = () => { void loadFilterOptions(); void loadModelStats() }
const activeFilterCount = computed(() => [filters.value.platform, filters.value.group_id, filters.value.request_type,
  filters.value.native_compaction_v2, filters.value.billing_type, filters.value.billing_mode]
  .filter(value => value !== null && value !== undefined && value !== '').length)
const busy = computed(() => activeTab.value === 'errors' ? errorLoading.value : loading.value || statsLoading.value)

const resetErrorRows = () => {
  // A date change also invalidates requests started before leaving the error tab.
  errorReqSeq++
  errorPage.value = 1
  errorRows.value = []
  errorTotal.value = 0
  errorListError.value = false
  if (activeTab.value === 'errors') {
    void loadErrors()
  } else {
    errorLoading.value = false
  }
}

const loadErrors = async () => {
  const seq = ++errorReqSeq
  errorLoading.value = true
  errorListError.value = false
  try {
    const resp = await usageAPI.listMyErrorRequests({
      page: errorPage.value,
      page_size: errorPageSize.value,
      timezone: DASHBOARD_TIMEZONE,
      start_date: startDate.value,
      end_date: endDate.value,
      model: (errorFilter.value.model ?? '').trim() || undefined,
      category: errorFilter.value.category || undefined,
      api_key_id: errorFilter.value.api_key_id ?? undefined,
      status_code: errorFilter.value.status_code ?? undefined,
      sort_by: errorSortBy.value,
      sort_order: errorSortOrder.value,
    })
    if (seq !== errorReqSeq || disposed) return
    errorRows.value = resp.items
    errorTotal.value = resp.total
  } catch (error) {
    if (seq !== errorReqSeq || disposed) return
    errorRows.value = []
    errorTotal.value = 0
    errorListError.value = true
  } finally {
    if (seq === errorReqSeq) errorLoading.value = false
  }
}

const onErrorSort = (sortBy: string, sortOrder: 'asc' | 'desc') => {
  errorSortBy.value = sortBy
  errorSortOrder.value = sortOrder
  errorPage.value = 1
  void loadErrors()
}

const onErrorPage = (page: number) => {
  errorPage.value = page
  void loadErrors()
}

const onErrorPageSize = (pageSize: number) => {
  errorPageSize.value = pageSize
  errorPage.value = 1
  void loadErrors()
}

const switchToErrors = () => {
  activeTab.value = 'errors'
  if (errorRows.value.length === 0) void loadErrors()
}

const ISO_DATE_RE = /^\d{4}-\d{2}-\d{2}$/
const ALLOWED_PLATFORMS = new Set<string>(CONCRETE_PLATFORM_OPTIONS.map((p) => p.value))

const queryStringParam = (value: unknown): string | undefined => {
  if (typeof value === 'string') return value.trim() || undefined
  if (Array.isArray(value) && typeof value[0] === 'string') return value[0].trim() || undefined
  return undefined
}

const queryValidIsoDate = (value: unknown): string | undefined => {
  const raw = queryStringParam(value)
  if (!raw || !ISO_DATE_RE.test(raw)) return undefined
  const [y, m, d] = raw.split('-').map(Number)
  const dt = new Date(Date.UTC(y, m - 1, d))
  if (
    Number.isNaN(dt.getTime()) ||
    dt.getUTCFullYear() !== y ||
    dt.getUTCMonth() !== m - 1 ||
    dt.getUTCDate() !== d
  ) {
    return undefined
  }
  return raw
}

const queryPositiveIntParam = (value: unknown): number | undefined => {
  const raw = queryStringParam(value)
  if (!raw || !/^\d+$/.test(raw)) return undefined
  const num = Number(raw)
  return Number.isSafeInteger(num) && num > 0 ? num : undefined
}

const applyRouteQueryFilters = (query: Record<string, unknown> | undefined): boolean => {
  if (!query) return false
  const qStart = queryValidIsoDate(query.start_date)
  const qEnd = queryValidIsoDate(query.end_date)
  const qModel = queryStringParam(query.model)
  const qGroupId = queryPositiveIntParam(query.group_id)
  const qApiKeyId = queryPositiveIntParam(query.api_key_id)
  const qPlatformRaw = queryStringParam(query.platform)?.toLowerCase()
  const qPlatform = qPlatformRaw && ALLOWED_PLATFORMS.has(qPlatformRaw) ? qPlatformRaw : undefined
  const qBillingTypeRaw = queryStringParam(query.billing_type)
  const qBillingType = subscriptionFeatureEnabled.value && (qBillingTypeRaw === '0' || qBillingTypeRaw === '1')
    ? Number(qBillingTypeRaw)
    : undefined

  const hasAny = Boolean(qStart || qEnd || qModel || qGroupId || qApiKeyId || qPlatform || qBillingType !== undefined)
  if (!hasAny) return false

  let nextStart = startDate.value
  let nextEnd = endDate.value
  if (qStart) {
    nextStart = qStart
    if (!qEnd) nextEnd = qStart
  }
  if (qEnd) {
    nextEnd = qEnd
    if (!qStart) nextStart = qEnd
  }
  if (nextStart > nextEnd) {
    nextStart = nextEnd
  }
  startDate.value = nextStart
  endDate.value = nextEnd
  filters.value = {
    start_date: nextStart,
    end_date: nextEnd,
    api_key_id: qApiKeyId,
    group_id: qGroupId,
    platform: qPlatform,
    model: qModel,
    request_type: undefined,
    native_compaction_v2: null,
    billing_type: qBillingType ?? null,
    billing_mode: null,
  }
  if (qPlatform || qGroupId || qBillingType !== undefined) {
    moreFilters.value = true
  }
  pagination.page = 1
  return true
}

onMounted(() => {
  loadSavedColumns()
  loadSavedErrColumns()
  document.addEventListener('click', handleColumnClickOutside)
  applyRouteQueryFilters(route?.query as Record<string, unknown> | undefined)
  void loadFilterOptions()
  refreshData()
})

watch(
  () => route?.query,
  (nextQuery, prevQuery) => {
    if (!prevQuery || nextQuery === prevQuery) return
    if (applyRouteQueryFilters(nextQuery as Record<string, unknown> | undefined)) {
      refreshData()
    }
  }
)

onUnmounted(() => {
  disposed = true
  statsReqSeq++
  modelStatsReqSeq++
  errorReqSeq++
  abortController?.abort()
  document.removeEventListener('click', handleColumnClickOutside)
})

watch(errorViewEnabled, enabled => { if (!enabled) { activeTab.value = 'usage'; errorReqSeq++; errorRows.value = []; errorLoading.value = false } })
</script>
