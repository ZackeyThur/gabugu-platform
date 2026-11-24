import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import type { UsageDashboardSnapshotV2Response } from '@/api/usage'
import type { ModelStat, TrendDataPoint } from '@/types'

const mocks = vi.hoisted(() => ({
  stats: vi.fn(),
  snapshot: vi.fn(),
  quotas: vi.fn(),
  affiliate: vi.fn(),
  checkout: vi.fn(),
  keys: vi.fn(),
  groups: vi.fn(),
  user: vi.fn(),
  auth: { isSimpleMode: false },
}))

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardStats: mocks.stats,
    getDashboardSnapshotV2: mocks.snapshot,
  },
}))
vi.mock('@/api/user', () => ({
  getMyPlatformQuotas: mocks.quotas,
  userAPI: { getAffiliateDetail: mocks.affiliate },
}))
vi.mock('@/api/payment', () => ({
  paymentAPI: { getCheckoutInfo: mocks.checkout },
}))
vi.mock('@/api/keys', () => ({
  keysAPI: { list: mocks.keys },
}))
vi.mock('@/api/groups', () => ({
  userGroupsAPI: { getAvailable: mocks.groups },
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return mocks.auth.isSimpleMode
    },
    refreshUser: mocks.user,
  }),
}))

import { useDashboardOverview } from '../useDashboardOverview'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

const wrappers: ReturnType<typeof mount>[] = []
function start() {
  let overview!: ReturnType<typeof useDashboardOverview>
  const wrapper = mount(
    defineComponent({
      setup() {
        overview = useDashboardOverview()
        return () => null
      },
    }),
    { global: { plugins: [createPinia()] } }
  )
  wrappers.push(wrapper)
  return overview
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-11-08T15:59:30Z'))
  vi.resetAllMocks()
  mocks.auth.isSimpleMode = false
  mocks.stats.mockResolvedValue({ total_actual_cost: 1 })
  mocks.snapshot.mockResolvedValue({
    trend: [{ date: '2026-11-08', actual_cost: 2 } as TrendDataPoint],
    models: [],
    groups: [],
    api_keys: [],
  })
  mocks.quotas.mockResolvedValue({ platform_quotas: [] })
  mocks.affiliate.mockResolvedValue({ aff_quota: 0 })
  mocks.checkout.mockResolvedValue({ data: { methods: {}, plans: [] } })
  mocks.keys.mockResolvedValue({ items: [], total: 0 })
  mocks.groups.mockResolvedValue([])
  mocks.user.mockResolvedValue({ balance: 10 })
})

afterEach(() => {
  wrappers.splice(0).forEach((w) => w.unmount())
  vi.useRealTimers()
})

describe('费用总览数据生命周期', () => {
  it('首次通过单次 snapshot-v2 拉取分析数据，切换日期/筛选不重复请求顶部统计且冻结7日基线', async () => {
    const s = start()
    await flushPromises()

    expect(s.startDate.value).toBe('2026-11-02')
    expect(s.endDate.value).toBe('2026-11-08')
    expect(s.granularity.value).toBe('day')
    expect(mocks.stats).toHaveBeenCalledWith({ timezone: 'Asia/Shanghai' })
    expect(mocks.snapshot).toHaveBeenCalledTimes(1)
    expect(mocks.snapshot).toHaveBeenLastCalledWith({
      start_date: '2026-11-02',
      end_date: '2026-11-08',
      granularity: 'day',
      timezone: 'Asia/Shanghai',
      include_trend: true,
      include_model_stats: true,
      include_group_stats: true,
      include_api_key_stats: true,
    })
    expect(s.baseline7d.value).toHaveLength(1)

    // Switch to 30d and custom range with filter: baseline7d remains frozen
    mocks.snapshot.mockResolvedValueOnce({
      trend: [{ date: '2026-10-10', actual_cost: 9 } as TrendDataPoint],
      models: [],
      groups: [],
      api_keys: [],
    })
    s.setRange('2026-10-10', '2026-11-08', 'day')
    await flushPromises()

    s.setFilters({ apiKeyId: 42, groupId: 7 })
    await flushPromises()

    expect(mocks.stats).toHaveBeenCalledTimes(1)
    expect(mocks.user).toHaveBeenCalledTimes(1)
    expect(mocks.quotas).toHaveBeenCalledTimes(1)
    expect(mocks.snapshot).toHaveBeenLastCalledWith({
      start_date: '2026-10-10',
      end_date: '2026-11-08',
      granularity: 'day',
      timezone: 'Asia/Shanghai',
      api_key_id: 42,
      group_id: 7,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: true,
      include_api_key_stats: true,
    })
    // baseline7d stays on the unfiltered 7-day baseline
    expect(s.baseline7d.value[0].date).toBe('2026-11-08')
  })

  it('各错误互不污染、余额失败保持null且支持独立重试', async () => {
    mocks.user.mockRejectedValueOnce(new Error('balance unavailable'))
    mocks.snapshot.mockRejectedValueOnce(new Error('snapshot unavailable'))
    const s = start()
    await flushPromises()

    expect(s.balance.error.value).toBe(true)
    expect(s.balance.data.value).toBeNull()
    expect(s.trend.error.value).toBe(true)
    expect(s.models.error.value).toBe(true)
    expect(s.stats.error.value).toBe(false)

    await s.loadBalance()
    expect(s.balance.error.value).toBe(false)
    expect(s.balance.data.value?.balance).toBe(10)

    mocks.snapshot.mockResolvedValueOnce({
      trend: [{ date: '2026-11-08', actual_cost: 3 } as TrendDataPoint],
    })
    await s.loadTrend()
    expect(s.trend.error.value).toBe(false)
    expect(s.trend.data.value[0].actual_cost).toBe(3)
    expect(s.models.error.value).toBe(true) // independent retry keeps other error state isolated
  })

  it('忽略旧日期范围成功和失败，保留最新loading与结果', async () => {
    const s = start()
    await flushPromises()

    const oldSnap = deferred<UsageDashboardSnapshotV2Response>()
    const newSnap = deferred<UsageDashboardSnapshotV2Response>()
    mocks.snapshot.mockReturnValueOnce(oldSnap.promise).mockReturnValueOnce(newSnap.promise)

    s.setRange('2026-10-01', '2026-10-03')
    s.setRange('2026-10-10', '2026-10-12')

    oldSnap.resolve({
      trend: [{ date: '2026-10-01' } as TrendDataPoint],
      models: [{ model: 'stale' } as ModelStat],
    })
    await flushPromises()

    expect(s.trend.loading.value).toBe(true)
    expect(s.models.error.value).toBe(false)

    newSnap.resolve({
      trend: [{ date: '2026-10-12' } as TrendDataPoint],
      models: [{ model: 'current' } as ModelStat],
      groups: [],
      api_keys: [],
    })
    await flushPromises()

    expect(s.trend.data.value[0].date).toBe('2026-10-12')
    expect(s.models.data.value[0].model).toBe('current')
    expect(s.trend.loading.value).toBe(false)
  })

  it('独立重试 loadTrend/loadModels 若晚于新范围 loadAnalysis 返回不会覆盖新数据', async () => {
    const s = start()
    await flushPromises()

    const slowTrend = deferred<UsageDashboardSnapshotV2Response>()
    const slowModels = deferred<UsageDashboardSnapshotV2Response>()
    const fastAnalysis = deferred<UsageDashboardSnapshotV2Response>()
    mocks.snapshot
      .mockReturnValueOnce(slowTrend.promise)
      .mockReturnValueOnce(slowModels.promise)
      .mockReturnValueOnce(fastAnalysis.promise)

    const p1 = s.loadTrend()
    const p2 = s.loadModels()
    s.setRange('2026-10-01', '2026-10-30', 'day')

    fastAnalysis.resolve({
      trend: [{ date: '2026-10-30', actual_cost: 99 } as TrendDataPoint],
      models: [{ model: 'fresh-model' } as ModelStat],
      groups: [{ group_id: 9, group_name: 'fresh-group', requests: 1, total_tokens: 1, cost: 1, actual_cost: 1 }],
      api_keys: [{ api_key_id: 9, key_name: 'fresh-key', deleted: false, requests: 1, total_tokens: 1, cost: 1, actual_cost: 1 }],
    })
    await flushPromises()

    slowTrend.resolve({
      trend: [{ date: '2026-11-08', actual_cost: 1 } as TrendDataPoint],
    })
    slowModels.resolve({
      models: [{ model: 'stale-model' } as ModelStat],
      groups: [{ group_id: 1, group_name: 'stale-group', requests: 1, total_tokens: 1, cost: 1, actual_cost: 1 }],
      api_keys: [{ api_key_id: 1, key_name: 'stale-key', deleted: false, requests: 1, total_tokens: 1, cost: 1, actual_cost: 1 }],
    })
    await Promise.all([p1, p2])
    await flushPromises()

    expect(s.trend.data.value[0].date).toBe('2026-10-30')
    expect(s.models.data.value[0].model).toBe('fresh-model')
    expect(s.groups.data.value[0].group_name).toBe('fresh-group')
    expect(s.apiKeys.data.value[0].key_name).toBe('fresh-key')
  })

  it.each([1, 7, 30])('跨北京时间午夜自动更新%d日预设和顶部统计', async (days) => {
    const s = start()
    await flushPromises()

    if (days === 1) {
      s.setRange('2026-11-08', '2026-11-08', 'hour')
      await flushPromises()
    } else if (days === 30) {
      s.setRange('2026-10-10', '2026-11-08', 'day')
      await flushPromises()
    }

    vi.setSystemTime(new Date('2026-11-08T16:00:30Z'))
    await vi.advanceTimersByTimeAsync(60000)
    await flushPromises()

    expect(s.today.value).toBe('2026-11-09')
    expect(s.endDate.value).toBe('2026-11-09')
    expect(s.startDate.value).toBe(
      days === 1 ? '2026-11-09' : days === 7 ? '2026-11-03' : '2026-10-11'
    )
    expect(s.granularity.value).toBe(days === 1 ? 'hour' : 'day')
    expect(mocks.stats).toHaveBeenCalledTimes(2)
  })

  it('窗口focus跨午夜保留自定义日期，简易模式不取额度与筛选列表，卸载取消刷新', async () => {
    mocks.auth.isSimpleMode = true
    const s = start()
    await flushPromises()

    s.setRange('2026-10-01', '2026-10-03')
    await flushPromises()

    vi.setSystemTime(new Date('2026-11-08T16:01:00Z'))
    window.dispatchEvent(new Event('focus'))
    await flushPromises()

    expect(s.today.value).toBe('2026-11-09')
    expect(s.startDate.value).toBe('2026-10-01')
    expect(s.endDate.value).toBe('2026-10-03')
    expect(mocks.quotas).not.toHaveBeenCalled()
    expect(mocks.checkout).not.toHaveBeenCalled()
    expect(mocks.keys).not.toHaveBeenCalled()

    wrappers.splice(0).forEach((w) => w.unmount())
    const calls = mocks.stats.mock.calls.length
    vi.setSystemTime(new Date('2026-11-09T16:01:00Z'))
    await vi.advanceTimersByTimeAsync(60000)
    window.dispatchEvent(new Event('focus'))
    await flushPromises()
    expect(mocks.stats).toHaveBeenCalledTimes(calls)
  })
})
