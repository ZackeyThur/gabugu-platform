<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <div class="flex items-center gap-3.5 rounded-2xl border border-gray-200/90 bg-white px-4 py-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-800">
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-500/15 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-400">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5.636 18.364a9 9 0 010-12.728m12.728 0a9 9 0 010 12.728m-9.9-2.829a5 5 0 010-7.07m7.072 0a5 5 0 010 7.07M13 12a1 1 0 11-2 0 1 1 0 012 0z" />
              </svg>
            </div>
            <div class="min-w-0">
              <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.status.active') }}
              </div>
              <div class="mt-0.5 flex items-baseline gap-1.5">
                <span class="font-mono text-xl font-bold tabular-nums text-gray-900 dark:text-white">
                  {{ overviewStats.activeCount }}
                </span>
                <span class="font-mono text-xs text-gray-400">/ {{ pagination.total }}</span>
              </div>
            </div>
          </div>

          <div class="flex items-center gap-3.5 rounded-2xl border border-gray-200/90 bg-white px-4 py-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-800">
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-500/15 text-primary-600 dark:bg-primary-500/15 dark:text-primary-400">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div class="min-w-0">
              <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.statsDialog.totalResponses') }}
              </div>
              <div class="mt-0.5 font-mono text-xl font-bold tabular-nums text-gray-900 dark:text-white">
                {{ overviewStats.totalResponses }}
              </div>
            </div>
          </div>

          <div class="flex items-center gap-3.5 rounded-2xl border border-gray-200/90 bg-white px-4 py-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-800">
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-amber-500/15 text-amber-600 dark:bg-amber-500/15 dark:text-amber-400">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
                <path stroke-linecap="round" stroke-linejoin="round" d="M20 12v8a2 2 0 01-2 2H6a2 2 0 01-2-2v-8m16 0H4m16 0a2 2 0 002-2V9a2 2 0 00-2-2h-3.17a3 3 0 00.17-1 3 3 0 00-5.83-1 3 3 0 00-5.83 1c0 .35.06.68.17 1H4a2 2 0 00-2 2v1a2 2 0 002 2m8-5v15" />
              </svg>
            </div>
            <div class="min-w-0">
              <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.statsDialog.rewardIssued') }}
              </div>
              <div class="mt-0.5 flex items-baseline gap-1.5">
                <span class="font-mono text-xl font-bold tabular-nums text-emerald-600 dark:text-emerald-400">
                  {{ overviewStats.rewardIssued }}
                </span>
                <span class="font-mono text-xs text-gray-400">
                  (${{ overviewStats.rewardValue.toFixed(2) }})
                </span>
              </div>
            </div>
          </div>

          <div class="flex items-center gap-3.5 rounded-2xl border border-gray-200/90 bg-white px-4 py-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-800">
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-sky-500/15 text-sky-600 dark:bg-sky-500/15 dark:text-sky-400">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm0 8a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zm10 0a1 1 0 011-1h4a1 1 0 011 1v6a1 1 0 01-1 1h-4a1 1 0 01-1-1v-6z" />
              </svg>
            </div>
            <div class="min-w-0">
              <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.columns.displayMode') }}
              </div>
              <div class="mt-1 flex flex-wrap items-center gap-1.5 text-xs font-semibold">
                <span
                  class="inline-flex items-center gap-1 rounded-md bg-primary-500/10 px-1.5 py-0.5 text-primary-700 dark:bg-primary-500/20 dark:text-primary-300"
                  :title="t('surveys.admin.displayMode.corner')"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="4" width="18" height="16" rx="2" />
                    <rect x="11" y="11" width="7" height="6" rx="1" fill="currentColor" stroke="none" />
                  </svg>
                  <span class="font-mono tabular-nums">{{ overviewStats.cornerCount }}</span>
                </span>
                <span
                  class="inline-flex items-center gap-1 rounded-md bg-indigo-500/10 px-1.5 py-0.5 text-indigo-700 dark:bg-indigo-500/20 dark:text-indigo-300"
                  :title="t('surveys.admin.displayMode.modal')"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <rect x="4" y="5" width="16" height="14" rx="2" />
                  </svg>
                  <span class="font-mono tabular-nums">{{ overviewStats.modalCount }}</span>
                </span>
                <span
                  class="inline-flex items-center gap-1 rounded-md bg-sky-500/10 px-1.5 py-0.5 text-sky-700 dark:bg-sky-500/20 dark:text-sky-300"
                  :title="t('surveys.admin.displayMode.micro')"
                >
                  <svg class="h-3 w-3" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M13 2L4.5 13.5h6L9.5 22 19.5 10.5h-6L13 2z" />
                  </svg>
                  <span class="font-mono tabular-nums">{{ overviewStats.microCount }}</span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-64">
            <input
              v-model="searchQuery"
              type="text"
              class="input"
              :placeholder="t('surveys.admin.searchPlaceholder')"
              @input="handleSearch"
            />
          </div>
          <Select
            v-model="filters.status"
            :options="statusFilterOptions"
            class="w-40"
            @change="handleFilterChange"
          />

          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadSurveys"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              type="button"
              data-testid="admin-create-survey-btn"
              class="btn btn-primary"
              @click="openCreate"
            >
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('surveys.admin.createSurvey') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="surveys"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-title="{ row }">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.title }}</div>
              <div class="mt-0.5 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span>#{{ row.id }}</span>
                <span>·</span>
                <span>{{ formatDateTime(row.created_at) }}</span>
              </div>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span
              class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold"
              :class="
                value === 'active'
                  ? 'bg-emerald-500/15 text-emerald-700 ring-1 ring-inset ring-emerald-500/20 dark:bg-emerald-500/15 dark:text-emerald-300'
                  : value === 'draft'
                    ? 'bg-gray-100 text-gray-600 ring-1 ring-inset ring-gray-300/60 dark:bg-dark-700 dark:text-gray-300'
                    : 'bg-amber-500/15 text-amber-700 ring-1 ring-inset ring-amber-500/20 dark:bg-amber-500/15 dark:text-amber-300'
              "
            >
              <span class="relative flex h-2 w-2">
                <span
                  v-if="value === 'active'"
                  class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"
                ></span>
                <span
                  class="relative inline-flex h-2 w-2 rounded-full"
                  :class="
                    value === 'active'
                      ? 'bg-emerald-500'
                      : value === 'draft'
                        ? 'bg-gray-400'
                        : 'bg-amber-500'
                  "
                ></span>
              </span>
              {{ t(`surveys.admin.status.${value}`) }}
            </span>
          </template>

          <template #cell-display_mode="{ value }">
            <span
              class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold"
              :class="
                value === 'modal'
                  ? 'bg-indigo-500/15 text-indigo-700 ring-1 ring-inset ring-indigo-500/20 dark:bg-indigo-500/15 dark:text-indigo-300'
                  : value === 'micro'
                    ? 'bg-sky-500/15 text-sky-700 ring-1 ring-inset ring-sky-500/20 dark:bg-sky-500/15 dark:text-sky-300'
                    : 'bg-primary-500/15 text-primary-700 ring-1 ring-inset ring-primary-500/20 dark:bg-primary-500/15 dark:text-primary-300'
              "
            >
              <svg v-if="value === 'micro'" class="h-3 w-3" viewBox="0 0 24 24" fill="currentColor">
                <path d="M13 2L4.5 13.5h6L9.5 22 19.5 10.5h-6L13 2z" />
              </svg>
              <svg v-else-if="value === 'modal'" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <rect x="4" y="5" width="16" height="14" rx="2" />
              </svg>
              <svg v-else class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <rect x="3" y="4" width="18" height="16" rx="2" />
                <rect x="11" y="11" width="7" height="6" rx="1" fill="currentColor" stroke="none" />
              </svg>
              {{ t(`surveys.admin.displayMode.${value}`) }}
            </span>
          </template>

          <template #cell-reward="{ row }">
            <span v-if="row.reward_type === 'none' || !row.reward_value" class="text-xs text-gray-400">
              {{ t('surveys.admin.rewardSummaryNone') }}
            </span>
            <div v-else class="min-w-[100px] space-y-1">
              <div class="flex items-center justify-between gap-2 text-xs">
                <span class="font-mono font-semibold text-emerald-700 dark:text-emerald-300">
                  ${{ row.reward_value.toFixed(2) }}
                </span>
                <span class="font-mono text-xs tabular-nums text-gray-500 dark:text-dark-400">
                  {{
                    row.reward_budget_limit
                      ? `${row.reward_issued_count}/${row.reward_budget_limit}`
                      : t('surveys.admin.rewardSummaryUnlimited', {
                          value: row.reward_value.toFixed(2),
                          issued: row.reward_issued_count
                        }).split('·')[1]?.trim() || row.reward_issued_count
                  }}
                </span>
              </div>
              <div
                v-if="row.reward_budget_limit"
                class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200/80 dark:bg-dark-700"
              >
                <div
                  class="h-full rounded-full transition-all"
                  :class="
                    row.reward_issued_count / row.reward_budget_limit >= 0.85
                      ? 'bg-gradient-to-r from-amber-500 to-orange-500'
                      : 'bg-gradient-to-r from-emerald-500 to-teal-500'
                  "
                  :style="{
                    width: `${Math.min(100, Math.round((row.reward_issued_count / Math.max(1, row.reward_budget_limit)) * 100))}%`
                  }"
                ></div>
              </div>
            </div>
          </template>

          <template #cell-response_count="{ value }">
            <span class="font-mono text-sm font-semibold tabular-nums text-gray-900 dark:text-white">
              {{ value }}
            </span>
          </template>

          <template #cell-schedule="{ row }">
            <div class="text-xs text-gray-600 dark:text-gray-300">
              <div>{{ row.starts_at ? formatDateTime(row.starts_at) : t('admin.announcements.timeImmediate') }}</div>
              <div class="text-gray-400 dark:text-dark-400">
                → {{ row.ends_at ? formatDateTime(row.ends_at) : t('admin.announcements.timeNever') }}
              </div>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
                @click="openPreview(row)"
              >
                {{ t('surveys.admin.preview') }}
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-xs font-medium text-emerald-600 transition-colors hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20"
                @click="openStats(row)"
              >
                {{ t('surveys.admin.stats') }}
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                @click="openEdit(row)"
              >
                {{ t('common.edit') }}
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                @click="openDelete(row)"
              >
                {{ t('common.delete') }}
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('surveys.admin.emptyTitle')"
              :description="t('surveys.admin.emptyDescription')"
              :action-text="t('surveys.admin.createSurvey')"
              @action="openCreate"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :total="pagination.total"
          :page="pagination.page"
          :page-size="pagination.page_size"
          @update:page="(p) => { pagination.page = p; loadSurveys() }"
          @update:page-size="(s) => { pagination.page_size = s; pagination.page = 1; loadSurveys() }"
        />
      </template>
    </TablePageLayout>

    <!-- 编辑 / 新建问卷弹窗 -->
    <SurveyEditorDialog
      :show="showEditor"
      :survey="editingSurvey"
      :saving="saving"
      :subscription-groups="subscriptionGroups"
      @close="showEditor = false"
      @save="handleSave"
      @preview="(s) => { previewSurvey = s }"
    />

    <!-- 统计与答卷弹窗 -->
    <SurveyStatsDialog
      :show="showStats"
      :survey="statsSurvey"
      @close="showStats = false"
    />

    <!-- 删除确认 -->
    <ConfirmDialog
      :show="showDelete"
      :title="t('surveys.admin.deleteTitle')"
      :message="t('surveys.admin.deleteConfirm', { title: deletingSurvey?.title || '' })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDelete = false"
    />

    <!-- 实时交互预览弹窗（复用同一个用户端组件） -->
    <SurveyPopup
      :survey="previewSurvey"
      preview
      @close="previewSurvey = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import SurveyPopup from '@/components/common/SurveyPopup.vue'
import SurveyEditorDialog from '@/components/admin/surveys/SurveyEditorDialog.vue'
import SurveyStatsDialog from '@/components/admin/surveys/SurveyStatsDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import type { AdminGroup } from '@/types'
import type { CreateSurveyRequest, Survey, UpdateSurveyRequest, UserSurvey } from '@/types/survey'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const surveys = ref<Survey[]>([])
const subscriptionGroups = ref<AdminGroup[]>([])

const searchQuery = ref('')
const filters = reactive({
  status: '',
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const showEditor = ref(false)
const editingSurvey = ref<Survey | null>(null)
const showStats = ref(false)
const statsSurvey = ref<Survey | null>(null)
const showDelete = ref(false)
const deletingSurvey = ref<Survey | null>(null)
const previewSurvey = ref<UserSurvey | null>(null)

let searchTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed(() => [
  { key: 'title', label: t('surveys.admin.columns.title'), sortable: true },
  { key: 'status', label: t('surveys.admin.columns.status'), sortable: true },
  { key: 'display_mode', label: t('surveys.admin.columns.displayMode'), sortable: true },
  { key: 'reward', label: t('surveys.admin.columns.reward') },
  { key: 'response_count', label: t('surveys.admin.columns.responses') },
  { key: 'schedule', label: t('surveys.admin.columns.schedule') },
  { key: 'actions', label: t('surveys.admin.columns.actions') }
])

const overviewStats = computed(() => {
  let activeCount = 0
  let totalResponses = 0
  let rewardIssued = 0
  let rewardValue = 0
  let cornerCount = 0
  let modalCount = 0
  let microCount = 0
  for (const s of surveys.value) {
    if (s.status === 'active') activeCount++
    totalResponses += s.response_count || 0
    rewardIssued += s.reward_issued_count || 0
    if (s.reward_type !== 'none' && s.reward_value > 0) {
      rewardValue += (s.reward_issued_count || 0) * s.reward_value
    }
    if (s.display_mode === 'modal') modalCount++
    else if (s.display_mode === 'micro') microCount++
    else cornerCount++
  }
  return {
    activeCount,
    totalResponses,
    rewardIssued,
    rewardValue,
    cornerCount,
    modalCount,
    microCount
  }
})

const statusFilterOptions = computed(() => [
  { value: '', label: t('surveys.admin.allStatuses') },
  { value: 'draft', label: t('surveys.admin.status.draft') },
  { value: 'active', label: t('surveys.admin.status.active') },
  { value: 'archived', label: t('surveys.admin.status.archived') }
])

async function loadSurveys() {
  try {
    loading.value = true
    const res = await adminAPI.surveys.list(pagination.page, pagination.page_size, {
      status: filters.status || undefined,
      search: searchQuery.value.trim() || undefined,
      sort_by: filters.sort_by,
      sort_order: filters.sort_order
    })
    surveys.value = res.items || []
    pagination.total = res.total || 0
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || t('surveys.admin.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadSubscriptionGroups() {
  try {
    const groups = await adminAPI.groups.getAll()
    subscriptionGroups.value = (groups || []).filter((g) => g.subscription_type === 'subscription')
  } catch {
    subscriptionGroups.value = []
  }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1
    loadSurveys()
  }, 300)
}

function handleFilterChange() {
  pagination.page = 1
  loadSurveys()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  filters.sort_by = key
  filters.sort_order = order
  pagination.page = 1
  loadSurveys()
}

function openCreate() {
  editingSurvey.value = null
  showEditor.value = true
}

function openEdit(row: Survey) {
  editingSurvey.value = row
  showEditor.value = true
}

function openStats(row: Survey) {
  statsSurvey.value = row
  showStats.value = true
}

function openDelete(row: Survey) {
  deletingSurvey.value = row
  showDelete.value = true
}

function openPreview(row: Survey) {
  const hasReward =
    row.reward_type !== 'none' &&
    row.reward_value > 0 &&
    (!row.reward_budget_limit || row.reward_issued_count < row.reward_budget_limit)
  previewSurvey.value = {
    id: row.id,
    title: row.title,
    description: row.description,
    display_mode: row.display_mode,
    questions: row.questions,
    has_reward: hasReward,
    reward_type: hasReward ? row.reward_type : 'none',
    reward_value: hasReward ? row.reward_value : 0,
    snooze_days: row.delivery_policy?.snooze_days || 3,
    skip_if_announcement_unread: row.delivery_policy?.skip_if_announcement_unread ?? true,
    delay_seconds: 0,
    show_micro_results: row.delivery_policy?.show_micro_results ?? true,
    created_at: row.created_at
  }
}

async function handleSave(payload: { id?: number; data: CreateSurveyRequest | UpdateSurveyRequest }) {
  try {
    saving.value = true
    if (payload.id) {
      await adminAPI.surveys.update(payload.id, payload.data as UpdateSurveyRequest)
      appStore.showSuccess(t('surveys.admin.updateSuccess'))
    } else {
      await adminAPI.surveys.create(payload.data as CreateSurveyRequest)
      appStore.showSuccess(t('surveys.admin.createSuccess'))
    }
    showEditor.value = false
    await loadSurveys()
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || t('surveys.admin.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingSurvey.value) return
  try {
    await adminAPI.surveys.delete(deletingSurvey.value.id)
    appStore.showSuccess(t('surveys.admin.deleteSuccess'))
    showDelete.value = false
    deletingSurvey.value = null
    await loadSurveys()
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || t('surveys.admin.saveFailed'))
  }
}

onMounted(() => {
  loadSurveys()
  loadSubscriptionGroups()
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>
