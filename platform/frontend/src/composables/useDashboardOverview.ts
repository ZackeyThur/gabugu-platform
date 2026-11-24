import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { getActivePinia } from 'pinia'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { getMyPlatformQuotas, userAPI } from '@/api/user'
import { paymentAPI } from '@/api/payment'
import { keysAPI } from '@/api/keys'
import { userGroupsAPI } from '@/api/groups'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type {
  ApiKey,
  APIKeyStat,
  Group,
  GroupStat,
  ModelStat,
  PlatformQuotaItem,
  TrendDataPoint,
  User,
  UserAffiliateDetail,
} from '@/types'
import type { CheckoutInfoResponse } from '@/types/payment'
import { dashboardToday, DASHBOARD_TIMEZONE, shiftDate } from '@/utils/dashboardOverview'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

// Each section owns its error/loading state. A superseded response must never
// paint an older date range under the current range label.
function resource<T>(initial: T) {
  const data = shallowRef<T>(initial)
  const loading = ref(false)
  const error = ref(false)
  let sequence = 0
  async function run(fetcher: () => Promise<T>) {
    const current = ++sequence
    loading.value = true
    error.value = false
    try {
      const result = await fetcher()
      if (current === sequence) data.value = result
    } catch {
      if (current === sequence) error.value = true
    } finally {
      if (current === sequence) loading.value = false
    }
  }
  onBeforeUnmount(() => { sequence++ })
  return { data, loading, error, run }
}

export interface DashboardAnalysisFilters {
  apiKeyId: number | null
  groupId: number | null
}

export function useDashboardOverview() {
  const auth = useAuthStore()
  const today = ref(dashboardToday())
  const startDate = ref(shiftDate(today.value, -6))
  const endDate = ref(today.value)
  const granularity = ref<'day' | 'hour'>('day')
  const apiKeyId = ref<number | null>(null)
  const groupId = ref<number | null>(null)
  const lastUpdatedAt = ref<Date | null>(null)

  const balance = resource<User | null>(null)
  const stats = resource<UserDashboardStats | null>(null)
  const quotas = resource<PlatformQuotaItem[]>([])
  const trend = resource<TrendDataPoint[]>([])
  const models = resource<ModelStat[]>([])
  const groups = resource<GroupStat[]>([])
  const apiKeys = resource<APIKeyStat[]>([])

  // Frozen unfiltered 7-day daily trend baseline for the top "Today Pulse" sparkline
  const baseline7d = shallowRef<TrendDataPoint[]>([])

  // Silent enrichment data (failures gracefully degrade without affecting main UI)
  const checkoutInfo = shallowRef<CheckoutInfoResponse | null>(null)
  const affiliateDetail = shallowRef<UserAffiliateDetail | null>(null)
  const filterApiKeys = shallowRef<ApiKey[]>([])
  const filterGroups = shallowRef<Group[]>([])

  let trendSequence = 0
  let breakdownSequence = 0
  let unmounted = false

  function getPublicSettings() {
    const pinia = getActivePinia()
    if (!pinia) return null
    try {
      return useAppStore(pinia).cachedPublicSettings
    } catch {
      return null
    }
  }

  function isDefault7dBaselineQuery() {
    return (
      startDate.value === shiftDate(today.value, -6) &&
      endDate.value === today.value &&
      granularity.value === 'day' &&
      apiKeyId.value === null &&
      groupId.value === null
    )
  }

  const analysisParams = () => ({
    start_date: startDate.value,
    end_date: endDate.value,
    granularity: granularity.value,
    timezone: DASHBOARD_TIMEZONE,
    ...(apiKeyId.value ? { api_key_id: apiKeyId.value } : {}),
    ...(groupId.value ? { group_id: groupId.value } : {}),
  })

  const loadBalance = () => balance.run(() => auth.refreshUser())
  const loadStats = () => stats.run(() => usageAPI.getDashboardStats({ timezone: DASHBOARD_TIMEZONE }))
  const loadQuotas = () => quotas.run(async () => (await getMyPlatformQuotas()).platform_quotas ?? [])

  async function loadAnalysis() {
    const currentTrend = ++trendSequence
    const currentBreakdown = ++breakdownSequence
    const captureBaseline = isDefault7dBaselineQuery()
    trend.loading.value = true
    trend.error.value = false
    models.loading.value = true
    models.error.value = false
    groups.loading.value = true
    groups.error.value = false
    apiKeys.loading.value = true
    apiKeys.error.value = false
    try {
      const snapshot = await usageAPI.getDashboardSnapshotV2({
        ...analysisParams(),
        include_trend: true,
        include_model_stats: true,
        include_group_stats: true,
        include_api_key_stats: true,
      })
      if (unmounted) return
      if (currentTrend === trendSequence) {
        const nextTrend = snapshot.trend ?? []
        trend.data.value = nextTrend
        if (captureBaseline) {
          baseline7d.value = nextTrend
        }
      }
      if (currentBreakdown === breakdownSequence) {
        models.data.value = snapshot.models ?? []
        groups.data.value = snapshot.groups ?? []
        apiKeys.data.value = snapshot.api_keys ?? []
      }
    } catch {
      if (unmounted) return
      if (currentTrend === trendSequence) {
        trend.error.value = true
      }
      if (currentBreakdown === breakdownSequence) {
        models.error.value = true
        groups.error.value = true
        apiKeys.error.value = true
      }
    } finally {
      if (!unmounted) {
        if (currentTrend === trendSequence) {
          trend.loading.value = false
        }
        if (currentBreakdown === breakdownSequence) {
          models.loading.value = false
          groups.loading.value = false
          apiKeys.loading.value = false
        }
      }
    }
  }

  async function loadTrend() {
    const current = ++trendSequence
    const captureBaseline = isDefault7dBaselineQuery()
    trend.loading.value = true
    trend.error.value = false
    try {
      const res = await usageAPI.getDashboardSnapshotV2({
        ...analysisParams(),
        include_trend: true,
        include_model_stats: false,
        include_group_stats: false,
        include_api_key_stats: false,
      })
      if (unmounted || current !== trendSequence) return
      const nextTrend = res.trend ?? []
      trend.data.value = nextTrend
      if (captureBaseline) {
        baseline7d.value = nextTrend
      }
    } catch {
      if (unmounted || current !== trendSequence) return
      trend.error.value = true
    } finally {
      if (!unmounted && current === trendSequence) {
        trend.loading.value = false
      }
    }
  }

  async function loadModels() {
    const current = ++breakdownSequence
    models.loading.value = true
    models.error.value = false
    groups.loading.value = true
    groups.error.value = false
    apiKeys.loading.value = true
    apiKeys.error.value = false
    try {
      const res = await usageAPI.getDashboardSnapshotV2({
        ...analysisParams(),
        include_trend: false,
        include_model_stats: true,
        include_group_stats: true,
        include_api_key_stats: true,
      })
      if (unmounted || current !== breakdownSequence) return
      models.data.value = res.models ?? []
      groups.data.value = res.groups ?? []
      apiKeys.data.value = res.api_keys ?? []
    } catch {
      if (unmounted || current !== breakdownSequence) return
      models.error.value = true
      groups.error.value = true
      apiKeys.error.value = true
    } finally {
      if (!unmounted && current === breakdownSequence) {
        models.loading.value = false
        groups.loading.value = false
        apiKeys.loading.value = false
      }
    }
  }

  async function loadCheckoutInfoSilently() {
    const settings = getPublicSettings()
    if (!resolveFeatureFlag(settings, FeatureFlags.payment)) return
    try {
      const res = await paymentAPI.getCheckoutInfo()
      if (!unmounted && res?.data) {
        checkoutInfo.value = res.data
      }
    } catch {
      // Silent fallback: quick recharge pills degrade gracefully
    }
  }

  async function reloadAffiliate() {
    const settings = getPublicSettings()
    if (auth.isSimpleMode || !resolveFeatureFlag(settings, FeatureFlags.affiliate)) return
    try {
      const detail = await userAPI.getAffiliateDetail()
      if (!unmounted && detail) {
        affiliateDetail.value = detail
      }
    } catch {
      // Silent fallback
    }
  }

  async function loadFilterOptionsSilently() {
    if (auth.isSimpleMode) return
    await Promise.allSettled([
      keysAPI.list(1, 100).then((res) => {
        if (!unmounted && res?.items) {
          filterApiKeys.value = res.items
        }
      }),
      userGroupsAPI.getAvailable().then((list) => {
        if (!unmounted && Array.isArray(list)) {
          filterGroups.value = list
        }
      }),
    ])
  }

  const refresh = async () => {
    const tasks: Promise<unknown>[] = [loadBalance(), loadStats(), loadAnalysis()]
    if (!auth.isSimpleMode) {
      tasks.push(loadQuotas(), loadCheckoutInfoSilently(), reloadAffiliate(), loadFilterOptionsSilently())
    }
    await Promise.allSettled(tasks)
    if (!unmounted) {
      lastUpdatedAt.value = new Date()
    }
  }

  const busy = computed(() =>
    [balance, stats, quotas, trend, models].some(section => section.loading.value)
  )

  function setRange(start: string, end: string, nextGranularity?: 'day' | 'hour') {
    startDate.value = start
    endDate.value = end
    if (nextGranularity) {
      granularity.value = nextGranularity
    } else {
      granularity.value = start === end ? 'hour' : 'day'
    }
    void loadAnalysis()
  }

  function setFilters(next: Partial<DashboardAnalysisFilters>) {
    if ('apiKeyId' in next) apiKeyId.value = next.apiKeyId ?? null
    if ('groupId' in next) groupId.value = next.groupId ?? null
    void loadAnalysis()
  }

  function checkDay() {
    const next = dashboardToday()
    if (next === today.value) return
    const presetDays = endDate.value === today.value
      ? startDate.value === today.value
        ? 1
        : startDate.value === shiftDate(today.value, -6)
          ? 7
          : startDate.value === shiftDate(today.value, -29)
            ? 30
            : null
      : null
    today.value = next
    if (presetDays === 1) {
      startDate.value = next
      endDate.value = next
      granularity.value = 'hour'
    } else if (presetDays) {
      startDate.value = shiftDate(next, -(presetDays - 1))
      endDate.value = next
      granularity.value = 'day'
    }
    void refresh()
  }

  watch(
    () => getPublicSettings()?.affiliate_enabled,
    (enabled) => {
      if (enabled === true && !auth.isSimpleMode && affiliateDetail.value === null) {
        void reloadAffiliate()
      }
    }
  )

  watch(
    () => getPublicSettings()?.payment_enabled,
    (enabled) => {
      if (enabled !== false && !auth.isSimpleMode && checkoutInfo.value === null) {
        void loadCheckoutInfoSilently()
      }
    }
  )

  let timer: ReturnType<typeof setInterval> | undefined
  onMounted(() => {
    void refresh()
    timer = setInterval(checkDay, 60_000)
    window.addEventListener('focus', checkDay)
  })
  onBeforeUnmount(() => {
    unmounted = true
    trendSequence++
    breakdownSequence++
    clearInterval(timer)
    window.removeEventListener('focus', checkDay)
  })

  return {
    today,
    startDate,
    endDate,
    granularity,
    apiKeyId,
    groupId,
    lastUpdatedAt,
    balance,
    stats,
    quotas,
    trend,
    models,
    groups,
    apiKeys,
    baseline7d,
    checkoutInfo,
    affiliateDetail,
    filterApiKeys,
    filterGroups,
    busy,
    refresh,
    setRange,
    setFilters,
    loadBalance,
    loadStats,
    loadQuotas,
    loadTrend,
    loadModels,
    loadAnalysis,
    reloadAffiliate,
  }
}
