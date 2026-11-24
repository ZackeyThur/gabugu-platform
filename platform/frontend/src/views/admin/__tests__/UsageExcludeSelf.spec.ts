import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

import UsageView from '../UsageView.vue'

const {
  list,
  exportList,
  getStats,
  getModelStats,
  getSnapshotV2,
  listErrorLogs,
  saveAs,
  authState,
} = vi.hoisted(() => ({
  list: vi.fn(),
  exportList: vi.fn(),
  saveAs: vi.fn(),
  getStats: vi.fn(),
  getModelStats: vi.fn(),
  getSnapshotV2: vi.fn(),
  listErrorLogs: vi.fn(),
  authState: { user: { id: 7 } as { id: number } | null },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: { list, getStats },
    dashboard: { getSnapshotV2, getModelStats },
    users: { getById: vi.fn() },
  },
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: { list: exportList },
}))

vi.mock('file-saver', () => ({ saveAs }))

vi.mock('xlsx', () => ({
  utils: {
    aoa_to_sheet: vi.fn(() => ({})),
    sheet_add_aoa: vi.fn(),
    book_new: vi.fn(() => ({})),
    book_append_sheet: vi.fn(),
  },
  write: vi.fn(() => new Uint8Array([1])),
}))

vi.mock('@/api/admin/ops', () => ({ listErrorLogs }))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authState }))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showWarning: vi.fn(), showSuccess: vi.fn(), showInfo: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }) }))

const STORAGE_PREFIX = 'admin-usage-exclude-self'

const AppLayoutStub = { template: '<div><slot /></div>' }
const UsageFiltersStub = defineComponent({ template: '<div><slot name="after-reset" /></div>' })
const emptyStats = {
  total_requests: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_tokens: 0,
  total_tokens: 0,
  total_cost: 0,
  total_actual_cost: 0,
  average_duration_ms: 0,
}

function mountView() {
  return mount(UsageView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        UsageStatsCards: true,
        UsageFilters: UsageFiltersStub,
        UsageTable: true,
        UsageExportProgress: true,
        UsageCleanupDialog: true,
        UserBalanceHistoryModal: true,
        OpsErrorDetailModal: true,
        Pagination: true,
        Select: true,
        DateRangePicker: true,
        Icon: true,
        TokenUsageTrend: true,
        ModelDistributionChart: true,
        GroupDistributionChart: true,
        EndpointDistributionChart: true,
        UserTokenRanking: true,
        OpsErrorLogTable: true,
      },
    },
  })
}

async function settle(wrapper: ReturnType<typeof mountView>) {
  vi.advanceTimersByTime(120)
  await flushPromises()
  return wrapper
}

function clearCalls() {
  list.mockClear()
  getStats.mockClear()
  getModelStats.mockClear()
  getSnapshotV2.mockClear()
}

describe('admin usage 隐藏我的记录', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
    authState.user = { id: 7 }
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    getSnapshotV2.mockReset().mockResolvedValue({ trend: [], models: [], groups: [] })
    listErrorLogs.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('默认开启：列表、统计、模型与 snapshot 请求都带 exclude_self', async () => {
    const wrapper = await settle(mountView())
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }), expect.anything())
    expect(getStats).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }))
    expect(getModelStats).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }))
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }))
    wrapper.unmount()
  })

  it('关闭后写入账号专属键 false，请求不再带 exclude_self', async () => {
    const wrapper = await settle(mountView())
    clearCalls()
    await wrapper.get('[data-testid="usage-exclude-self-toggle"]').trigger('click')
    await flushPromises()
    expect(localStorage.getItem(`${STORAGE_PREFIX}:7`)).toBe('false')
    expect(list).toHaveBeenLastCalledWith(expect.not.objectContaining({ exclude_self: true }), expect.anything())
    expect(getStats).toHaveBeenLastCalledWith(expect.not.objectContaining({ exclude_self: true }))
    wrapper.unmount()
  })

  it('重新挂载恢复已存偏好；两个账号互不串偏好', async () => {
    localStorage.setItem(`${STORAGE_PREFIX}:7`, 'false')
    const first = await settle(mountView())
    expect(list).toHaveBeenLastCalledWith(expect.not.objectContaining({ exclude_self: true }), expect.anything())
    first.unmount()

    authState.user = { id: 8 }
    const second = await settle(mountView())
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }), expect.anything())
    second.unmount()
  })

  it('存储值非法或无法读取时安全回退为默认开启', async () => {
    localStorage.setItem(`${STORAGE_PREFIX}:7`, 'garbage')
    const wrapper = await settle(mountView())
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }), expect.anything())
    wrapper.unmount()

    const getSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied')
    })
    const fallback = await settle(mountView())
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }), expect.anything())
    fallback.unmount()
    getSpy.mockRestore()
  })

  it('指定 user_id 时暂停隐藏并显示暂停提示；清空后恢复', async () => {
    const wrapper = await settle(mountView())
    const vm = wrapper.vm as unknown as { filters: { user_id?: number }; applyFilters: () => void }
    clearCalls()
    vm.filters.user_id = 42
    vm.applyFilters()
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ user_id: 42 }), expect.anything())
    expect(list).toHaveBeenLastCalledWith(expect.not.objectContaining({ exclude_self: true }), expect.anything())
    expect(wrapper.find('[data-testid="usage-exclude-self-paused"]').exists()).toBe(true)

    vm.filters.user_id = undefined
    vm.applyFilters()
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }), expect.anything())
    expect(wrapper.find('[data-testid="usage-exclude-self-paused"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('重置普通筛选不改变偏好', async () => {
    const wrapper = await settle(mountView())
    await wrapper.get('[data-testid="usage-exclude-self-toggle"]').trigger('click')
    await flushPromises()
    ;(wrapper.vm as unknown as { resetFilters: () => void }).resetFilters()
    await flushPromises()
    expect(localStorage.getItem(`${STORAGE_PREFIX}:7`)).toBe('false')
    expect(list).toHaveBeenLastCalledWith(expect.not.objectContaining({ exclude_self: true }), expect.anything())
    wrapper.unmount()
  })
})

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => { resolve = res })
  return { promise, resolve }
}

describe('admin usage 隐藏我的记录：导出快照、API Key 暂停与错误请求', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
    authState.user = { id: 7 }
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    exportList.mockReset()
    saveAs.mockReset()
    getStats.mockReset().mockResolvedValue(emptyStats)
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    getSnapshotV2.mockReset().mockResolvedValue({ trend: [], models: [], groups: [] })
    listErrorLogs.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('选中具体 API Key 时暂停隐藏，清空后恢复', async () => {
    const wrapper = await settle(mountView())
    const vm = wrapper.vm as unknown as { filters: { api_key_id?: number }; applyFilters: () => void }
    clearCalls()
    vm.filters.api_key_id = 5
    vm.applyFilters()
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.not.objectContaining({ exclude_self: true }), expect.anything())
    expect(wrapper.find('[data-testid="usage-exclude-self-paused"]').exists()).toBe(true)

    vm.filters.api_key_id = undefined
    vm.applyFilters()
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }), expect.anything())
    wrapper.unmount()
  })

  it('多页导出：翻页中切换开关或改日期，后续页与文件名仍使用发起时的快照', async () => {
    const wrapper = await settle(mountView())
    const vm = wrapper.vm as unknown as Record<string, any>
    vm.filters.start_date = '2026-11-01'
    vm.filters.end_date = '2026-11-02'

    const page1 = deferred<any>()
    exportList
      .mockImplementationOnce(() => page1.promise)
      .mockResolvedValueOnce({ items: Array.from({ length: 50 }, (_, i) => ({ id: i })), total: 150, pages: 2 })

    const running = vm.exportToExcel()
    await flushPromises()

    // 第一页未返回时：关闭隐藏、改日期
    vm.excludeSelf = false
    vm.filters.start_date = '2026-12-01'
    vm.filters.end_date = '2026-12-02'
    await flushPromises()

    page1.resolve({ items: Array.from({ length: 100 }, (_, i) => ({ id: i })), total: 150, pages: 2 })
    await running
    await flushPromises()

    expect(exportList).toHaveBeenCalledTimes(2)
    for (const [params] of exportList.mock.calls) {
      expect(params).toEqual(expect.objectContaining({
        exclude_self: true,
        start_date: '2026-11-01',
        end_date: '2026-11-02',
      }))
    }
    expect(saveAs).toHaveBeenCalledWith(expect.anything(), 'usage_2026-11-01_to_2026-11-02.xlsx')
    wrapper.unmount()
  })

  it('切到用量 Tab 后，在途的错误请求响应不得回写错误列表', async () => {
    const wrapper = await settle(mountView())
    const vm = wrapper.vm as unknown as Record<string, any>
    const pending = deferred<any>()
    listErrorLogs.mockImplementationOnce(() => pending.promise)

    vm.switchTab('errors')
    await flushPromises()
    expect(listErrorLogs).toHaveBeenLastCalledWith(expect.objectContaining({ exclude_self: true }))

    vm.switchTab('usage')
    await flushPromises()

    pending.resolve({ items: [{ id: 99 }], total: 1, pages: 1 })
    await flushPromises()
    expect(vm.errRows).toEqual([])
    expect(vm.errLoading).toBe(false)
    wrapper.unmount()
  })
})
