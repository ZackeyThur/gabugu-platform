<template>
  <BaseDialog
    :show="show"
    :title="survey ? `${t('surveys.admin.statsDialog.title')} · ${survey.title}` : t('surveys.admin.statsDialog.title')"
    width="extra-wide"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <!-- 顶部 4 项关键指标 -->
      <div v-if="stats" class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <div class="flex items-center gap-3 rounded-2xl border border-gray-200/90 bg-white p-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-900/50">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-500/15 text-primary-600 dark:bg-primary-500/20 dark:text-primary-400">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
              <path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </div>
          <div class="min-w-0">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('surveys.admin.statsDialog.totalResponses') }}
            </div>
            <div class="mt-0.5 font-mono text-2xl font-bold tabular-nums text-gray-900 dark:text-white">
              {{ stats.total_responses }}
            </div>
          </div>
        </div>

        <div class="flex items-center gap-3 rounded-2xl border border-gray-200/90 bg-white p-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-900/50">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-indigo-500/15 text-indigo-600 dark:bg-indigo-500/20 dark:text-indigo-400">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div class="min-w-0">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('surveys.admin.statsDialog.avgDuration') }}
            </div>
            <div class="mt-0.5 font-mono text-2xl font-bold tabular-nums text-gray-900 dark:text-white">
              {{ t('surveys.admin.statsDialog.secondsUnit', { sec: stats.average_duration_seconds }) }}
            </div>
          </div>
        </div>

        <div class="flex items-center gap-3 rounded-2xl border border-gray-200/90 bg-white p-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-900/50">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-500/15 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20 12v8a2 2 0 01-2 2H6a2 2 0 01-2-2v-8m16 0H4m16 0a2 2 0 002-2V9a2 2 0 00-2-2h-3.17a3 3 0 00.17-1 3 3 0 00-5.83-1 3 3 0 00-5.83 1c0 .35.06.68.17 1H4a2 2 0 00-2 2v1a2 2 0 002 2m8-5v15" />
            </svg>
          </div>
          <div class="min-w-0">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('surveys.admin.statsDialog.rewardIssued') }}
            </div>
            <div class="mt-0.5 flex items-baseline gap-1 font-mono text-2xl font-bold tabular-nums text-emerald-600 dark:text-emerald-400">
              <span>{{ stats.reward_issued_count }}</span>
              <span class="text-xs font-medium text-gray-500 dark:text-dark-400">
                (${{ stats.reward_total_value.toFixed(2) }})
              </span>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-3 rounded-2xl border border-gray-200/90 bg-white p-3.5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.12)] dark:border-dark-700 dark:bg-dark-900/50">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-amber-500/15 text-amber-600 dark:bg-amber-500/20 dark:text-amber-400">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.9">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
          </div>
          <div class="min-w-0">
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('surveys.admin.statsDialog.snoozedDismissed') }}
            </div>
            <div class="mt-0.5 font-mono text-2xl font-bold tabular-nums text-gray-900 dark:text-white">
              {{ stats.snoozed_count }} <span class="text-base text-gray-400">/</span> {{ stats.dismissed_count }}
            </div>
          </div>
        </div>
      </div>

      <!-- 标签页切换 -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="tabs">
          <button
            type="button"
            class="tab"
            :class="{ 'tab-active': activeTab === 'summary' }"
            @click="activeTab = 'summary'"
          >
            {{ t('surveys.admin.statsDialog.tabSummary') }}
          </button>
          <button
            type="button"
            class="tab"
            :class="{ 'tab-active': activeTab === 'responses' }"
            @click="switchToResponses"
          >
            {{ t('surveys.admin.statsDialog.tabResponses') }}
          </button>
        </div>

        <button
          v-if="activeTab === 'responses' && responses.length > 0"
          type="button"
          class="btn btn-secondary btn-sm"
          @click="exportResponsesCsv"
        >
          {{ t('surveys.admin.statsDialog.exportCsv') }}
        </button>
      </div>

      <!-- 加载中 -->
      <div v-if="loadingStats && activeTab === 'summary'" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ t('common.loading') }}
      </div>

      <!-- Tab 1: 可视化统计 -->
      <div v-else-if="activeTab === 'summary'" class="space-y-4">
        <div
          v-if="!stats || stats.total_responses === 0"
          class="rounded-2xl border border-dashed border-gray-300 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('surveys.admin.statsDialog.noResponses') }}
        </div>

        <div
          v-for="(q, idx) in stats?.questions || []"
          v-else
          :key="q.question_id"
          class="space-y-3.5 rounded-2xl border border-gray-200/90 bg-white p-5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.1)] dark:border-dark-700 dark:bg-dark-900/40"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <span class="inline-flex h-6 min-w-[28px] items-center justify-center rounded-lg bg-primary-500/15 px-1.5 font-mono text-xs font-bold text-primary-700 dark:bg-primary-500/20 dark:text-primary-300">
                Q{{ idx + 1 }}
              </span>
              <h4 class="text-sm font-bold text-gray-900 dark:text-white">
                {{ q.title }}
              </h4>
            </div>
            <span class="rounded-full bg-gray-100 px-2.5 py-0.5 font-mono text-xs font-semibold tabular-nums text-gray-600 dark:bg-dark-700 dark:text-gray-300">
              {{ q.answer_count }}
            </span>
          </div>

          <!-- 单选 / 多选梯度进度条（最高票自动高亮） -->
          <div v-if="q.options?.length" class="space-y-3">
            <div v-for="opt in q.options" :key="opt.option_id" class="space-y-1.5">
              <div class="flex items-center justify-between gap-2 text-xs">
                <div class="flex min-w-0 items-center gap-2">
                  <span
                    class="truncate"
                    :class="
                      isTopOption(q.options, opt.percent)
                        ? 'font-bold text-slate-900 dark:text-white'
                        : 'font-medium text-slate-700 dark:text-slate-300'
                    "
                  >
                    {{ opt.label }}
                  </span>
                  <span
                    v-if="isTopOption(q.options, opt.percent)"
                    class="inline-flex flex-shrink-0 items-center rounded-full bg-primary-500/15 px-1.5 py-0.5 text-xs font-bold uppercase tracking-wider text-primary-700 dark:text-primary-300"
                  >
                    TOP
                  </span>
                </div>
                <span class="flex-shrink-0 font-mono font-semibold tabular-nums text-slate-600 dark:text-slate-300">
                  {{ opt.count }}
                  <span
                    :class="
                      isTopOption(q.options, opt.percent)
                        ? 'text-primary-600 dark:text-primary-400'
                        : 'text-slate-400 dark:text-slate-400'
                    "
                  >
                    ({{ opt.percent }}%)
                  </span>
                </span>
              </div>
              <div class="h-2.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-dark-700/80">
                <div
                  class="h-full rounded-full transition-all duration-500"
                  :class="
                    isTopOption(q.options, opt.percent)
                      ? 'bg-gradient-to-r from-primary-600 via-primary-500 to-sky-400 shadow-[0_0_12px_rgba(37,99,235,0.35)]'
                      : 'bg-slate-300 dark:bg-slate-600'
                  "
                  :style="{ width: `${Math.min(100, Math.max(0, opt.percent))}%` }"
                ></div>
              </div>
            </div>
          </div>

          <!-- 评分分布：App Store / Stripe 级双栏评分分析卡 -->
          <div
            v-else-if="q.type === 'rating'"
            class="flex flex-col gap-5 rounded-2xl border border-slate-100 bg-slate-50/70 p-4 sm:flex-row sm:items-center dark:border-white/[0.06] dark:bg-white/[0.02]"
          >
            <!-- 左栏：超大展示均分 + 5 颗比例星标 -->
            <div class="flex flex-col items-center justify-center sm:w-44 sm:border-r sm:border-slate-200/80 sm:pr-5 dark:sm:border-white/[0.08]">
              <div class="font-mono text-[40px] font-bold leading-none tracking-[-0.03em] tabular-nums text-slate-950 dark:text-white">
                {{ (q.average_rating || 0).toFixed(2) }}
              </div>
              <div class="mt-2 flex items-center gap-1" aria-hidden="true">
                <svg
                  v-for="s in 5"
                  :key="s"
                  class="h-4 w-4"
                  :class="
                    s <= Math.round(q.average_rating || 0)
                      ? 'text-amber-400 drop-shadow-[0_1px_3px_rgba(245,158,11,0.35)]'
                      : 'text-slate-200 dark:text-slate-700'
                  "
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                </svg>
              </div>
              <div class="mt-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
                {{ t('surveys.admin.statsDialog.avgRating', { score: q.average_rating || 0 }) }}
              </div>
            </div>

            <!-- 右栏：5★ 至 1★ 分布横向直方图 -->
            <div class="flex-1 space-y-2">
              <div
                v-for="bucket in [...(q.rating_buckets || [])].reverse()"
                :key="bucket.score"
                class="flex items-center gap-3"
              >
                <span class="flex w-9 flex-shrink-0 items-center justify-end gap-0.5 font-mono text-xs font-semibold tabular-nums text-slate-600 dark:text-slate-300">
                  <span>{{ bucket.score }}</span>
                  <span class="text-amber-400">★</span>
                </span>
                <div class="h-2.5 flex-1 overflow-hidden rounded-full bg-slate-200/75 dark:bg-dark-700/90">
                  <div
                    class="h-full rounded-full transition-all duration-500"
                    :class="
                      bucket.score >= 4
                        ? 'bg-gradient-to-r from-primary-600 to-sky-400'
                        : bucket.score === 3
                          ? 'bg-slate-400 dark:bg-slate-500'
                          : 'bg-gradient-to-r from-amber-500 to-rose-400'
                    "
                    :style="{ width: `${Math.min(100, Math.max(0, bucket.percent))}%` }"
                  ></div>
                </div>
                <span class="w-20 flex-shrink-0 text-right font-mono text-xs font-semibold tabular-nums text-slate-500 dark:text-slate-400">
                  {{ bucket.count }} ({{ bucket.percent }}%)
                </span>
              </div>
            </div>
          </div>

          <!-- 标签与开放文本 -->
          <div v-else-if="q.type === 'tags_text'" class="space-y-3.5">
            <div v-if="q.top_tags?.length">
              <div class="mb-2 text-xs font-semibold text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.statsDialog.topTags') }}
              </div>
              <div class="flex flex-wrap gap-2">
                <span
                  v-for="(tag, tIdx) in q.top_tags"
                  :key="tag.tag"
                  class="inline-flex items-center gap-1.5 rounded-full py-1 pl-3 pr-1.5 text-xs font-semibold ring-1 ring-inset"
                  :class="
                    tIdx === 0
                      ? 'bg-primary-600 text-white ring-primary-600 shadow-sm shadow-primary-600/25'
                      : 'bg-primary-500/10 text-primary-700 ring-primary-500/20 dark:bg-primary-500/15 dark:text-primary-300'
                  "
                >
                  <span>{{ tag.tag }}</span>
                  <span
                    class="rounded-full px-1.5 py-0.5 font-mono text-xs font-bold leading-none"
                    :class="
                      tIdx === 0
                        ? 'bg-white/20 text-white'
                        : 'bg-primary-600 text-white'
                    "
                  >
                    {{ tag.count }}
                  </span>
                </span>
              </div>
            </div>

            <div v-if="q.recent_texts?.length" class="space-y-2">
              <div class="text-xs font-semibold text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.statsDialog.recentTexts') }}
              </div>
              <div
                v-for="(item, tIdx) in q.recent_texts"
                :key="tIdx"
                class="flex items-start gap-3 rounded-2xl border border-slate-200/80 bg-slate-50/75 px-3.5 py-3 text-xs dark:border-white/[0.08] dark:bg-white/[0.025]"
              >
                <span class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full bg-primary-500/15 font-mono text-xs font-bold uppercase text-primary-700 dark:text-primary-300">
                  {{ (item.user_email || 'U').slice(0, 1) }}
                </span>
                <div class="min-w-0 flex-1">
                  <p class="leading-relaxed text-slate-900 dark:text-slate-100">{{ item.text }}</p>
                  <div class="mt-1 text-xs text-slate-400 dark:text-slate-500">
                    {{ item.user_email }} · {{ formatDateTime(item.submitted_at) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab 2: 答卷明细 -->
      <div v-else class="space-y-3">
        <div class="flex items-center gap-2">
          <input
            v-model="responseSearch"
            type="text"
            class="input max-w-sm"
            :placeholder="t('surveys.admin.statsDialog.searchResponses')"
            @input="onResponseSearchInput"
          />
        </div>

        <div class="table-container">
          <table class="table">
            <thead>
              <tr>
                <th>{{ t('surveys.admin.statsDialog.colUser') }}</th>
                <th>{{ t('surveys.admin.statsDialog.colAnswers') }}</th>
                <th>{{ t('surveys.admin.statsDialog.colDuration') }}</th>
                <th>{{ t('surveys.admin.statsDialog.colReward') }}</th>
                <th>{{ t('surveys.admin.statsDialog.colSubmittedAt') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loadingResponses">
                <td colspan="5" class="py-8 text-center text-gray-500">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else-if="responses.length === 0">
                <td colspan="5" class="py-8 text-center text-gray-500">
                  {{ t('surveys.admin.statsDialog.noResponses') }}
                </td>
              </tr>
              <tr v-for="row in responses" v-else :key="row.id">
                <td class="whitespace-nowrap">
                  <div class="font-medium text-gray-900 dark:text-white">{{ row.user_email || `#${row.user_id}` }}</div>
                  <div v-if="row.username" class="text-xs text-gray-500 dark:text-dark-400">{{ row.username }}</div>
                </td>
                <td class="max-w-md">
                  <div class="space-y-1 text-xs">
                    <div v-for="line in formatAnswersForDisplay(row)" :key="line" class="truncate" :title="line">
                      {{ line }}
                    </div>
                  </div>
                </td>
                <td class="whitespace-nowrap font-mono text-xs tabular-nums">
                  {{ t('surveys.admin.statsDialog.secondsUnit', { sec: row.duration_seconds }) }}
                </td>
                <td class="whitespace-nowrap text-xs">
                  <span
                    class="badge"
                    :class="
                      row.reward_status === 'granted' || row.reward_status === 'code_issued'
                        ? 'badge-success'
                        : row.reward_status === 'budget_exhausted'
                          ? 'badge-warning'
                          : 'badge-gray'
                    "
                  >
                    {{ t(`surveys.admin.statsDialog.rewardStatus.${row.reward_status}`) }}
                  </span>
                  <div v-if="row.reward_redeem_code" class="mt-0.5 font-mono text-xs text-gray-500 dark:text-dark-400">
                    {{ row.reward_redeem_code }}
                  </div>
                </td>
                <td class="whitespace-nowrap text-xs text-gray-500 dark:text-dark-400">
                  {{ formatDateTime(row.submitted_at) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="responseTotal > responsePageSize"
          :total="responseTotal"
          :page="responsePage"
          :page-size="responsePageSize"
          @update:page="(p) => { responsePage = p; loadResponses() }"
        />
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">
        {{ t('common.close') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminAPI } from '@/api/admin'
import { formatDateTime } from '@/utils/format'
import type { Survey, SurveyOptionStat, SurveyResponseItem, SurveyStats } from '@/types/survey'

function isTopOption(options: SurveyOptionStat[], percent: number): boolean {
  if (!options.length || percent <= 0) return false
  const max = options.reduce((acc, item) => Math.max(acc, item.percent), 0)
  return max > 0 && percent === max
}

const props = defineProps<{
  show: boolean
  survey: Survey | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const activeTab = ref<'summary' | 'responses'>('summary')
const loadingStats = ref(false)
const stats = ref<SurveyStats | null>(null)

const loadingResponses = ref(false)
const responses = ref<SurveyResponseItem[]>([])
const responsePage = ref(1)
const responsePageSize = 20
const responseTotal = ref(0)
const responseSearch = ref('')

let searchTimer: ReturnType<typeof setTimeout> | null = null

async function loadStats() {
  if (!props.survey?.id) return
  try {
    loadingStats.value = true
    stats.value = await adminAPI.surveys.getStats(props.survey.id)
  } finally {
    loadingStats.value = false
  }
}

async function loadResponses() {
  if (!props.survey?.id) return
  try {
    loadingResponses.value = true
    const res = await adminAPI.surveys.listResponses(
      props.survey.id,
      responsePage.value,
      responsePageSize,
      { search: responseSearch.value.trim() || undefined }
    )
    responses.value = res.items || []
    responseTotal.value = res.total || 0
  } finally {
    loadingResponses.value = false
  }
}

function switchToResponses() {
  activeTab.value = 'responses'
  loadResponses()
}

function onResponseSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    responsePage.value = 1
    loadResponses()
  }, 300)
}

function formatAnswersForDisplay(row: SurveyResponseItem): string[] {
  const qs = props.survey?.questions || []
  const lines: string[] = []
  qs.forEach((q, idx) => {
    const val = row.answers?.[q.id]
    if (!val) return
    if (q.type === 'single' && val.choice_id) {
      const label = q.options?.find((o) => o.id === val.choice_id)?.label || val.choice_id
      lines.push(`Q${idx + 1}: ${label}`)
    } else if (q.type === 'multi' && val.choice_ids?.length) {
      const labels = val.choice_ids.map((id) => q.options?.find((o) => o.id === id)?.label || id)
      lines.push(`Q${idx + 1}: ${labels.join(' / ')}`)
    } else if (q.type === 'rating' && typeof val.rating === 'number') {
      lines.push(`Q${idx + 1}: ${val.rating} 分`)
    } else if (q.type === 'tags_text') {
      const parts = [...(val.tags || [])]
      if (val.text) parts.push(`“${val.text}”`)
      if (parts.length) lines.push(`Q${idx + 1}: ${parts.join('；')}`)
    }
  })
  return lines
}

function sanitizeCsvCell(val: string): string {
  const s = String(val ?? '')
  const safe = /^[=+\-@\t\r]/.test(s) ? `'${s}` : s
  return `"${safe.replace(/"/g, '""')}"`
}

function exportResponsesCsv() {
  if (!props.survey) return
  const headers = ['UserID', 'Email', 'Username', 'DurationSec', 'RewardStatus', 'RewardValue', 'RedeemCode', 'SubmittedAt', 'Answers']
  const rows = responses.value.map((r) => [
    String(r.user_id),
    r.user_email || '',
    r.username || '',
    String(r.duration_seconds),
    r.reward_status,
    String(r.reward_value),
    r.reward_redeem_code || '',
    r.submitted_at,
    formatAnswersForDisplay(r).join(' | ')
  ])
  const csv = '\uFEFF' + [headers, ...rows].map((cols) => cols.map(sanitizeCsvCell).join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `survey-${props.survey.id}-responses.csv`
  link.click()
  URL.revokeObjectURL(url)
}

watch(
  () => [props.show, props.survey?.id],
  () => {
    if (!props.show || !props.survey?.id) return
    activeTab.value = 'summary'
    responsePage.value = 1
    responseSearch.value = ''
    loadStats()
  },
  { immediate: true }
)
</script>
