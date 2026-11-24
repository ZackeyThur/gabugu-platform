<template>
  <AppLayout>
    <MonitorHero
      :overall-status="overallStatus"
      :interval-seconds="DEFAULT_INTERVAL_SECONDS"
      :window="currentWindow"
      :loading="loading"
      :counts="counts"
      :last-check-at="lastCheckAt"
      :providers="providers"
      :active-provider="activeProvider"
      :search-query="searchQuery"
      :auto-refresh="autoRefresh"
      @update:window="handleWindowChange"
      @update:provider="activeProvider = $event"
      @update:search="searchQuery = $event"
      @refresh="refresh()"
    />

    <!-- 刷新失败但有旧数据：保留卡片，明确标注不是最新 -->
    <div
      v-if="staleMessage"
      class="mt-3 flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200"
      role="status"
    >
      <Icon name="exclamationTriangle" size="sm" class="mt-0.5 shrink-0" />
      <span class="min-w-0">{{ staleMessage }}</span>
    </div>

    <div class="mt-4">
      <MonitorCardGrid
        v-if="showGrid"
        :items="visibleItems"
        :window="currentWindow"
        :countdown-seconds="countdown"
        :loading="loading"
        :detail-cache="detailCache"
        :auto-refresh-on="autoRefresh.enabled.value"
        @card-click="openDetail"
      />

      <!-- 请求失败且没有任何可用数据 -->
      <EmptyState
        v-else-if="errorMessage"
        :title="t('channelStatus.error.title')"
        :description="errorMessage"
        :action-text="t('channelStatus.retry')"
        :action-icon="false"
        @action="refresh()"
      />

      <!-- 请求成功，但管理员没有配置可监控的渠道 -->
      <EmptyState
        v-else-if="items.length === 0"
        :title="t('channelStatus.empty.title')"
        :description="t('channelStatus.empty.description')"
      />

      <!-- 有渠道，但当前平台筛选/搜索没有命中 -->
      <EmptyState
        v-else
        :title="t('channelStatus.filter.emptyTitle')"
        :description="t('channelStatus.filter.emptyDescription')"
        :action-text="t('channelStatus.filter.clear')"
        :action-icon="false"
        @action="clearFilters"
      />
    </div>

    <MonitorDetailDialog
      :show="showDetail"
      :monitor-id="detailTarget?.id ?? null"
      :title="detailTitle"
      @close="closeDetail"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  list as listChannelMonitorViews,
  status as fetchChannelMonitorDetail,
  type UserMonitorView,
  type UserMonitorDetail,
} from '@/api/channelMonitor'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import MonitorHero, {
  type MonitorWindow,
  type OverallStatus,
} from '@/components/user/monitor/MonitorHero.vue'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'
import MonitorDetailDialog from '@/components/user/MonitorDetailDialog.vue'
import {
  DEFAULT_INTERVAL_SECONDS,
  STATUS_OPERATIONAL,
  STATUS_DEGRADED,
  STATUS_FAILED,
  STATUS_ERROR,
  PROVIDERS,
} from '@/constants/channelMonitor'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t } = useI18n()
const appStore = useAppStore()

// ── State ──
const items = ref<UserMonitorView[]>([])
const loading = ref(false)
const currentWindow = ref<MonitorWindow>('7d')
const detailCache = reactive<Record<number, UserMonitorDetail>>({})
const showDetail = ref(false)
const detailTarget = ref<UserMonitorView | null>(null)
const activeProvider = ref('')
const searchQuery = ref('')
/** 最近一次请求失败的原因；为空表示最近一次请求成功。 */
const lastErrorRaw = ref('')
/** 组件是否已卸载：卸载后不再回写缓存或弹提示。 */
const unmounted = ref(false)

let abortController: AbortController | null = null

const autoRefresh = useAutoRefresh({
  storageKey: 'channel-status-auto-refresh',
  intervals: [30, 60, 120] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  defaultEnabled: true,
  onRefresh: () => refresh(true),
  shouldPause: () => document.hidden || loading.value,
})
const countdown = autoRefresh.countdown

// ── Computed ──
const ABNORMAL_STATUSES: readonly string[] = [STATUS_DEGRADED, STATUS_FAILED, STATUS_ERROR]

/** 全站统计：基于全部渠道，不随平台筛选/搜索变化（否则筛选会藏住故障并变绿）。 */
const counts = computed(() => {
  let operational = 0
  let degraded = 0
  for (const it of items.value) {
    if (it.primary_status === STATUS_OPERATIONAL) operational += 1
    else if (ABNORMAL_STATUSES.includes(it.primary_status)) degraded += 1
  }
  return { total: items.value.length, operational, degraded }
})

/**
 * 只有当「非空 + 每条状态都是已知的正常」才给 operational。
 * 空列表不判成正常；状态值无法识别算 partial。
 * 注意：timeline 与 latest 是后端两条独立查询，历史为空只表示没有可展示的历史，
 * 不推翻已有的 latest 状态，所以这里不拿 timeline 参与总体结论。
 */
const overallStatus = computed<OverallStatus>(() => {
  const { total, operational, degraded } = counts.value
  if (total === 0) return 'empty'
  if (degraded > 0) return 'degraded'
  if (operational < total) return 'partial'
  return 'operational'
})

/**
 * 全部渠道里最近一次真实检测时间（取 timeline 最新的 checked_at）。
 * 这是后台探测时间，不是页面请求时间；没有样本就不显示。
 */
const lastCheckAt = computed<string | null>(() => {
  let newest = 0
  for (const it of items.value) {
    const ts = Date.parse(it.timeline?.[0]?.checked_at ?? '')
    if (!Number.isNaN(ts) && ts > newest) newest = ts
  }
  return newest > 0 ? new Date(newest).toISOString() : null
})

/** 渠道里实际出现的平台，按统一顺序排列，避免筛选按钮顺序抖动。 */
const providers = computed<string[]>(() => {
  const present = new Set<string>(items.value.map(it => it.provider))
  const known = (PROVIDERS as readonly string[]).filter(p => present.has(p))
  const extra = [...present].filter(p => !(PROVIDERS as readonly string[]).includes(p)).sort()
  return [...known, ...extra]
})

const visibleItems = computed<UserMonitorView[]>(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return items.value.filter(it => {
    if (activeProvider.value && it.provider !== activeProvider.value) return false
    if (!keyword) return true
    return (
      it.name.toLowerCase().includes(keyword) ||
      it.primary_model.toLowerCase().includes(keyword) ||
      (it.group_name ?? '').toLowerCase().includes(keyword)
    )
  })
})

/** 首屏加载时给骨架；有渠道但筛选无命中时进入筛选空态，而不是骨架。 */
const showGrid = computed(
  () => visibleItems.value.length > 0 || (loading.value && items.value.length === 0)
)

/**
 * 首次请求就失败、且手上没有可展示的渠道 → 错误态 + 重试。
 * 已有卡片时失败 → 保留卡片并标"不是最新"，不能把它说成"管理员没配渠道"。
 */
const errorMessage = computed(() =>
  items.value.length === 0 && lastErrorRaw.value
    ? lastErrorRaw.value || t('channelStatus.error.description')
    : ''
)

const staleMessage = computed(() =>
  items.value.length > 0 && lastErrorRaw.value ? t('channelStatus.stale') : ''
)

const detailTitle = computed(() => {
  return detailTarget.value?.name || t('channelStatus.detailTitle')
})

// ── Loaders ──
/**
 * 拉取渠道列表。返回「本次是否真的拿到有效列表」：
 * 失败、被取消、被更新请求取代、接口没给数组，都要返回 false，
 * 否则刷新失败后会继续去拉详情，把新详情和旧列表混在一起。
 */
async function reload(silent = false): Promise<boolean> {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  if (!silent) loading.value = true
  let ok = false
  try {
    const res = await listChannelMonitorViews({ signal: ctrl.signal })
    // 旧请求可能在卸载或新请求之后返回：只让当前请求写状态。
    if (ctrl.signal.aborted || abortController !== ctrl) return false
    // 缺少 items 数组属于接口异常，不能当成「管理员没有配置渠道」的空列表。
    if (!Array.isArray(res?.items)) {
      lastErrorRaw.value = t('channelStatus.error.description')
      return false
    }
    items.value = res.items
    lastErrorRaw.value = ''
    ok = true
    // 渠道表里已经没有当前筛选的平台时退出该筛选，避免用户停在空结果里。
    if (activeProvider.value && !items.value.some(it => it.provider === activeProvider.value)) {
      activeProvider.value = ''
    }
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return false
    if (abortController !== ctrl) return false
    const message = extractApiErrorMessage(err, t('channelStatus.loadError'))
    lastErrorRaw.value = message
    // 有旧数据时保留卡片（页面会标"不是最新"），只在没有可展示内容时弹全局错误。
    if (items.value.length === 0) appStore.showError(message)
  } finally {
    if (abortController === ctrl) {
      // 手动请求可能被后续静默刷新中止；由当前这次请求负责收尾，否则 loading 会卡住。
      loading.value = false
      autoRefresh.resetCountdown()
      abortController = null
    }
  }
  return ok
}

async function refresh(silent = false) {
  const ok = await reload(silent)
  if (!ok || currentWindow.value === '7d') return
  // 强制刷新「已有详情缓存」的渠道；没有缓存的交给 watch(items) 首次加载。
  // 这里不去重正在进行的请求，只保证同一渠道的旧响应不会覆盖新响应。
  await Promise.all(
    items.value
      .filter(it => detailCache[it.id])
      .map(it => loadDetail(it.id, true))
  )
}

/** 同一渠道的旧响应不能覆盖新响应。 */
const detailRequests = new Map<number, number>()

async function loadDetail(id: number, force = false) {
  if (!force && detailCache[id]) return
  const token = (detailRequests.get(id) ?? 0) + 1
  detailRequests.set(id, token)
  try {
    const detail = await fetchChannelMonitorDetail(id)
    // 组件已卸载时不再回写缓存，避免卸载后仍弹提示
    if (unmounted.value || detailRequests.get(id) !== token) return
    detailCache[id] = detail
  } catch (err: unknown) {
    if (unmounted.value || detailRequests.get(id) !== token) return
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.detailLoadError')))
  }
}

async function ensureDetailsForWindow() {
  if (currentWindow.value === '7d') return
  await Promise.all(items.value.map(it => loadDetail(it.id)))
}

// ── Handlers ──
async function handleWindowChange(value: MonitorWindow) {
  currentWindow.value = value
  await ensureDetailsForWindow()
}

/** 只本地清筛选，不重新请求：表格数据本来就是全量。 */
function clearFilters() {
  activeProvider.value = ''
  searchQuery.value = ''
}

function openDetail(row: UserMonitorView) {
  detailTarget.value = row
  showDetail.value = true
}

function closeDetail() {
  showDetail.value = false
  detailTarget.value = null
}

watch(items, () => {
  void ensureDetailsForWindow()
})

watch(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled,
  (enabled) => {
    if (enabled === false) autoRefresh.stop()
    else if (autoRefresh.enabled.value) autoRefresh.start()
  },
)

onMounted(() => {
  void reload(false)
  if (appStore.cachedPublicSettings?.channel_monitor_enabled !== false) {
    autoRefresh.setEnabled(autoRefresh.enabled.value)
  }
})

onBeforeUnmount(() => {
  unmounted.value = true
  if (abortController) abortController.abort()
})
</script>
