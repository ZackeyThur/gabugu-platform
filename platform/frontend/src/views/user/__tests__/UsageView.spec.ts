import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import UserUsageDateRange from '@/components/user/usage/UserUsageDateRange.vue'
import UserUsageRecordsTable from '@/components/user/usage/UserUsageRecordsTable.vue'
import UserUsageSummary from '@/components/user/usage/UserUsageSummary.vue'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import UsageTable from '@/components/admin/usage/UsageTable.vue'

const {
  query,
  getStats,
  getDashboardModels,
  getDashboardSnapshotV2,
  listMyErrorRequests,
  list,
  getAvailable,
  showError,
  showWarning,
  showSuccess,
  showInfo,
} = vi.hoisted(() => ({
  query: vi.fn(),
  getStats: vi.fn(),
  getDashboardModels: vi.fn(),
  getDashboardSnapshotV2: vi.fn(),
  listMyErrorRequests: vi.fn(),
  list: vi.fn(),
  getAvailable: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn(),
}))

const messages: Record<string, string> = {
  'admin.dashboard.timeRange': 'Time range',
  'admin.dashboard.granularity': 'Granularity',
  'admin.dashboard.day': 'Day',
  'admin.dashboard.hour': 'Hour',
  'admin.users.columnSettings': 'Columns',
  'admin.usage.group': 'Group',
  'admin.usage.billingType': 'Billing type',
  'admin.usage.billingMode': 'Billing mode',
  'admin.usage.allTypes': 'All types',
  'admin.usage.allBillingTypes': 'All billing types',
  'admin.usage.billingTypeBalance': 'Balance',
  'admin.usage.billingTypeSubscription': 'Subscription',
  'admin.usage.allBillingModes': 'All billing modes',
  'admin.usage.billingModeToken': 'Token',
  'admin.usage.billingModePerRequest': 'Per request',
  'admin.usage.billingModeImage': 'Image',
  'admin.usage.allGroups': 'All groups',
  'admin.usage.allModels': 'All models',
  'usage.allApiKeys': 'All API Keys',
  'usage.errors.allKeys': 'All API Keys',
  'usage.tabs.usage': 'Usage records',
  'usage.tabs.errors': 'Error records',
  'usage.apiKeyFilter': 'API Key',
  'usage.model': 'Model',
  'usage.type': 'Type',
  'usage.ws': 'WS',
  'usage.stream': 'Stream',
  'usage.sync': 'Sync',
  'usage.compactionFilter': 'Request Kind',
  'usage.allCompactionTypes': 'All Requests',
  'usage.compactionOnly': 'Compaction Only',
  'usage.exporting': 'Exporting',
  'usage.exportCsv': 'Export CSV',
  'usage.failedToLoad': 'Failed to load',
  'usage.noDataToExport': 'No data',
  'usage.preparingExport': 'Preparing export',
  'usage.exportSuccess': 'Export success',
  'usage.exportFailed': 'Export failed',
  'usage.exportRateLimited': 'Rate limited',
  'common.refresh': 'Refresh',
  'common.reset': 'Reset',
}

vi.mock('@/api', () => ({
  usageAPI: {
    query,
    getStats,
    getDashboardModels,
    getDashboardSnapshotV2,
    listMyErrorRequests,
  },
  keysAPI: {
    list,
  },
  userGroupsAPI: {
    getAvailable,
  },
}))

const authState = vi.hoisted(() => ({ user: { id: 9001 } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authState }))

const routeState = vi.hoisted(() => ({ query: {} as Record<string, unknown> }))
vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
  }
})

const appStoreState = vi.hoisted(() => ({
  cachedPublicSettings: { allow_user_view_error_requests: true } as Record<string, unknown>,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError, showWarning, showSuccess, showInfo,
    get cachedPublicSettings() {
      return appStoreState.cachedPublicSettings
    },
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const simpleStub = { template: '<div><slot /></div>' }
const chartStub = { template: '<div />' }

const usageLog = {
  id: 1,
  request_id: 'req-user-export',
  actual_cost: 0.092883,
  total_cost: 0.092883,
  rate_multiplier: 1,
  service_tier: 'priority',
  input_cost: 0.020285,
  output_cost: 0.00303,
  cache_creation_cost: 0.000001,
  cache_read_cost: 0.069568,
  input_tokens: 4057,
  output_tokens: 101,
  cache_creation_tokens: 4,
  cache_read_tokens: 278272,
  cache_creation_5m_tokens: 0,
  cache_creation_1h_tokens: 0,
  image_count: 0,
  image_size: null,
  first_token_ms: 12,
  duration_ms: 345,
  created_at: '2026-03-08T00:00:00Z',
  model: 'gpt-5.4',
  reasoning_effort: null,
  ip_address: '203.0.113.10',
  api_key: { name: 'demo-key' },
  billing_mode: 'token',
  request_type: 'sync',
  stream: false,
  native_compaction_v2: false,
}

function mountUsageView() {
  return mount(UsageView, {
    global: {
      stubs: {
        AppLayout: simpleStub,
        Pagination: true,
        Select: true,
        UserUsageDateRange: true,
        Icon: true,
        UserUsageSummary: { ...chartStub, props: ['stats', 'loading', 'error'] },
        UserUsageRecordsTable: { ...chartStub, props: ['rows', 'loading', 'error', 'sortBy', 'sortOrder'] },
        UsageTable: { ...chartStub, props: ['columns', 'data', 'loading'] },
        UserErrorRequestsTable: { ...chartStub, props: ['rows', 'total', 'loading', 'visibleColumnKeys'] },
        ModelDistributionChart: chartStub,
        GroupDistributionChart: chartStub,
        EndpointDistributionChart: chartStub,
        TokenUsageTrend: chartStub,
      },
    },
  })
}

describe('user UsageView', () => {
  beforeEach(() => {
    localStorage.clear()
    routeState.query = {}
    authState.user = { id: 9001 }
    appStoreState.cachedPublicSettings = { allow_user_view_error_requests: true }
    query.mockReset()
    getStats.mockReset()
    getDashboardModels.mockReset()
    getDashboardSnapshotV2.mockReset()
    listMyErrorRequests.mockReset()
    list.mockReset()
    getAvailable.mockReset()
    showError.mockReset()
    showWarning.mockReset()
    showSuccess.mockReset()
    showInfo.mockReset()

    query.mockResolvedValue({ items: [usageLog], total: 1, pages: 1 })
    getStats.mockResolvedValue({
      total_requests: 1,
      total_input_tokens: 10,
      total_output_tokens: 20,
      total_cache_tokens: 0,
      total_tokens: 30,
      total_cost: 0.1,
      total_actual_cost: 0.08,
      average_duration_ms: 12,
      endpoints: [],
      upstream_endpoints: [],
      endpoint_paths: [],
    })
    getDashboardModels.mockResolvedValue({
      models: [{ model: 'gpt-5.4', requests: 1, input_tokens: 10, output_tokens: 20, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 30, cost: 0.1, actual_cost: 0.08 }],
      start_date: '2026-03-08',
      end_date: '2026-03-08',
    })
    getDashboardSnapshotV2.mockResolvedValue({
      generated_at: '2026-03-08T00:00:00Z',
      start_date: '2026-03-08',
      end_date: '2026-03-08',
      granularity: 'hour',
      trend: [],
      groups: [],
    })
    listMyErrorRequests.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    list.mockResolvedValue({ items: [{ id: 1, name: 'demo-key' }], total: 1, page: 1, page_size: 100, pages: 1 })
    getAvailable.mockResolvedValue([{ id: 1, name: 'default' }])
  })

  it('starts on the Beijing calendar day and uses one explicit range for expenses, records, and model options', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-11-08T20:30:00Z'))
    const wrapper = mountUsageView()
    try {
      await flushPromises()
      const range = { start_date: '2026-11-09', end_date: '2026-11-09', timezone: 'Asia/Shanghai' }
      expect(query).toHaveBeenCalledWith(expect.objectContaining(range), expect.anything())
      expect(getStats).toHaveBeenCalledWith(expect.objectContaining(range))
      expect(getDashboardModels).toHaveBeenCalledWith({ ...range, model_source: 'requested' })
      expect(wrapper.findComponent(UserUsageDateRange).props()).toMatchObject({ startDate: range.start_date, endDate: range.end_date })
      expect(wrapper.find('#usage-more-filters').isVisible()).toBe(false)
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
    }
  })

  it('keeps all model options while filtering and changes only the records when paging or sorting', async () => {
    query.mockResolvedValue({ items: [usageLog], total: 101, pages: 6 })
    getDashboardModels.mockResolvedValue({ models: [{ model: 'gpt-5.4' }, { model: 'claude-sonnet-4' }] })
    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    const model = wrapper.findAllComponents(Select).find(select => select.attributes('id') === 'usage-model')!
    model.vm.$emit('update:modelValue', 'gpt-5.4')
    model.vm.$emit('change', 'gpt-5.4')
    await flushPromises()
    expect(getStats).toHaveBeenLastCalledWith(expect.objectContaining({ model: 'gpt-5.4', timezone: 'Asia/Shanghai' }))
    expect(model.props('options')).toContainEqual({ value: 'claude-sonnet-4', label: 'claude-sonnet-4' })
    expect(getDashboardModels).toHaveBeenCalledTimes(1)
    getStats.mockClear()
    wrapper.findComponent(Pagination).vm.$emit('update:page', 2)
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, model: 'gpt-5.4' }), expect.anything())
    wrapper.findComponent(UsageTable).vm.$emit('sort', 'model', 'asc')
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, sort_by: 'model', sort_order: 'asc' }), expect.anything())
    expect(getStats).not.toHaveBeenCalled()
    expect(getDashboardSnapshotV2).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('starts simple and restores the chosen technical view and columns on return', async () => {
    const wrapper = mountUsageView()
    await flushPromises()
    expect(wrapper.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').map((column: { key: string }) => column.key)).toEqual([
      'api_key', 'model', 'reasoning_effort', 'group', 'stream', 'billing_mode', 'tokens', 'cost', 'latency', 'created_at',
    ])
    await wrapper.get('[data-testid="usage-column-settings"]').trigger('click')
    await wrapper.get('[data-testid="usage-column-toggle-ip_address"]').trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').map((column: { key: string }) => column.key)).toContain('ip_address')
    wrapper.unmount()

    const returning = mountUsageView()
    await flushPromises()
    const keys = returning.findComponent(UsageTable).props('columns').map((column: { key: string }) => column.key)
    expect(keys).toContain('ip_address')
    expect(keys).not.toContain('endpoint')
    expect(keys).not.toContain('user_agent')
    returning.unmount()
  })

  it('isolates view and both column choices by user and restores them after switching accounts', async () => {
    const first = mountUsageView()
    await flushPromises()
    await first.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    await first.get('[data-testid="usage-column-settings"]').trigger('click')
    await first.get('[data-testid="usage-column-toggle-ip_address"]').trigger('click')
    await first.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    await first.get('[data-testid="usage-column-toggle-client_ip"]').trigger('click')
    first.unmount()

    authState.user = { id: 9002 }
    const second = mountUsageView()
    await flushPromises()
    expect(second.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    await second.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(second.findComponent(UsageTable).props('columns').map((col: { key: string }) => col.key)).not.toContain('ip_address')
    await second.get('[data-testid="usage-column-settings"]').trigger('click')
    await second.get('[data-testid="usage-column-toggle-latency"]').trigger('click')
    await second.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    expect(second.findComponent(UserErrorRequestsTable).props('visibleColumnKeys')).not.toContain('client_ip')
    second.unmount()

    authState.user = { id: 9001 }
    const returning = mountUsageView()
    await flushPromises()
    const columns = returning.findComponent(UsageTable).props('columns').map((col: { key: string }) => col.key)
    expect(columns).toContain('ip_address')
    expect(columns).toContain('latency')
    await returning.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    expect(returning.findComponent(UserErrorRequestsTable).props('visibleColumnKeys')).toContain('client_ip')
    await returning.findAll('button').find(button => button.text() === 'Usage records')!.trigger('click')
    await returning.findAll('button').find(button => button.text() === 'usageOverview.simpleView')!.trigger('click')
    returning.unmount()
    const simpleReturn = mountUsageView()
    await flushPromises()
    expect(simpleReturn.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    simpleReturn.unmount()
  })

  it('migrates legacy columns once and preserves an explicitly empty hidden list', async () => {
    localStorage.setItem('user-usage-hidden-columns', '[]')
    localStorage.setItem('user-usage-error-hidden-columns', '[]')
    const first = mountUsageView()
    await flushPromises()
    expect(localStorage.getItem('user-usage-hidden-columns:9001')).toBe('[]')
    expect(localStorage.getItem('user-usage-error-hidden-columns:9001')).toBe('[]')
    expect(localStorage.getItem('user-usage-hidden-columns')).toBeNull()
    expect(localStorage.getItem('user-usage-error-hidden-columns')).toBeNull()
    await first.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(first.findComponent(UsageTable).props('columns').map((col: { key: string }) => col.key)).toContain('user_agent')
    first.unmount()
    authState.user = { id: 9002 }
    const second = mountUsageView()
    await flushPromises()
    expect(second.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    await second.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(second.findComponent(UsageTable).props('columns').map((col: { key: string }) => col.key)).not.toContain('user_agent')
    await second.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    expect(second.findComponent(UserErrorRequestsTable).props('visibleColumnKeys')).not.toContain('client_ip')
    second.unmount()
  })

  it.each(['broken-json', '{}', '[1]'])('falls back safely for invalid saved preferences: %s', async saved => {
    localStorage.setItem('user-usage-view-mode:9001', 'invalid')
    localStorage.setItem('user-usage-hidden-columns:9001', saved)
    localStorage.setItem('user-usage-error-hidden-columns:9001', saved)
    const wrapper = mountUsageView()
    await flushPromises()
    expect(wrapper.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').map((col: { key: string }) => col.key)).not.toContain('ip_address')
    await wrapper.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    expect(wrapper.findComponent(UserErrorRequestsTable).props('visibleColumnKeys')).not.toContain('client_ip')
    wrapper.unmount()
  })

  it('keeps view and column controls usable when storage reads and writes fail', async () => {
    const readSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('unavailable') })
    const writeSpy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('unavailable') })
    const wrapper = mountUsageView()
    await flushPromises()
    expect(wrapper.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    await wrapper.get('[data-testid="usage-column-settings"]').trigger('click')
    await wrapper.get('[data-testid="usage-column-toggle-ip_address"]').trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').map((col: { key: string }) => col.key)).toContain('ip_address')
    await wrapper.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="usage-column-toggle-client_ip"]').trigger('click')
    expect(wrapper.findComponent(UserErrorRequestsTable).props('visibleColumnKeys')).toContain('client_ip')
    wrapper.unmount()
    readSpy.mockRestore()
    writeSpy.mockRestore()
  })

  it('keeps technical column preferences when switching back and forth', async () => {
    localStorage.setItem('user-usage-hidden-columns', JSON.stringify(['user_agent', 'latency']))
    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').map((column: { key: string }) => column.key)).not.toContain('latency')
    expect(wrapper.findComponent(UsageTable).props('columns').map((column: { key: string }) => column.key)).toEqual(expect.arrayContaining(['endpoint', 'ip_address']))
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.simpleView')!.trigger('click')
    expect(wrapper.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').map((column: { key: string }) => column.key)).not.toContain('latency')
    wrapper.unmount()
  })

  it('shows independent failures and retries without replacing failed expenses with zero', async () => {
    getStats.mockRejectedValueOnce(new Error('stats unavailable'))
    const wrapper = mountUsageView()
    await flushPromises()
    const summary = wrapper.findComponent(UserUsageSummary)
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.simpleView')!.trigger('click')
    expect(summary.props()).toMatchObject({ stats: null, error: true, loading: false })
    expect(wrapper.findComponent(UserUsageRecordsTable).props('rows')).toHaveLength(1)
    query.mockClear()
    summary.vm.$emit('retry')
    await flushPromises()
    expect(summary.props('error')).toBe(false)
    expect(query).not.toHaveBeenCalled()
    query.mockRejectedValueOnce(new Error('records unavailable'))
    wrapper.findComponent(UserUsageRecordsTable).vm.$emit('sort', 'model', 'asc')
    await flushPromises()
    expect(wrapper.findComponent(UserUsageRecordsTable).props()).toMatchObject({ rows: [], error: true, loading: false })
    expect(summary.props('stats')).toMatchObject({ total_actual_cost: 0.08 })
    const statsCalls = getStats.mock.calls.length
    wrapper.findComponent(UserUsageRecordsTable).vm.$emit('retry')
    await flushPromises()
    expect(wrapper.findComponent(UserUsageRecordsTable).props('error')).toBe(false)
    expect(getStats).toHaveBeenCalledTimes(statsCalls)
    wrapper.unmount()
  })

  it('ignores late record and expense responses after the user selects a newer range', async () => {
    let resolveOldRecords!: (value: unknown) => void
    let resolveOldStats!: (value: unknown) => void
    query.mockImplementationOnce(() => new Promise(resolve => { resolveOldRecords = resolve }))
    getStats.mockImplementationOnce(() => new Promise(resolve => { resolveOldStats = resolve }))
    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')
    wrapper.findComponent(UserUsageDateRange).vm.$emit('change', { startDate: '2026-11-01', endDate: '2026-11-07', preset: 'week' })
    await flushPromises()
    resolveOldRecords({ items: [{ ...usageLog, id: 999 }], total: 999 })
    resolveOldStats({ total_actual_cost: 999, total_requests: 999 })
    await flushPromises()
    expect(wrapper.findComponent(UsageTable).props('data')[0].id).toBe(1)
    expect(wrapper.findComponent(UserUsageSummary).props('stats').total_actual_cost).toBe(0.08)
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ start_date: '2026-11-01', timezone: 'Asia/Shanghai' }), expect.anything())
    wrapper.unmount()
  })

  it('keeps error records separate from expenses and ignores stale error responses', async () => {
    let resolveOld!: (value: unknown) => void
    listMyErrorRequests.mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    expect(wrapper.findComponent(UserUsageSummary).exists()).toBe(false)
    expect(wrapper.findAll('button').some(button => button.text() === 'Export CSV')).toBe(false)
    const key = wrapper.findAllComponents(Select).find(select => select.attributes('id') === 'error-key')!
    listMyErrorRequests.mockResolvedValueOnce({ items: [{ id: 2, model: 'new-error' }], total: 1 })
    key.vm.$emit('update:modelValue', 1)
    key.vm.$emit('change', 1)
    await flushPromises()
    resolveOld({ items: [{ id: 1, model: 'old-error' }], total: 999 })
    await flushPromises()
    expect(wrapper.findComponent(UserErrorRequestsTable).props('rows')).toEqual([{ id: 2, model: 'new-error' }])
    expect(listMyErrorRequests).toHaveBeenLastCalledWith(expect.objectContaining({ api_key_id: 1, timezone: 'Asia/Shanghai' }))
    listMyErrorRequests.mockRejectedValueOnce(new Error('errors unavailable'))
    await wrapper.findAll('button').find(button => button.text() === 'Refresh')!.trigger('click')
    await flushPromises()
    expect(wrapper.findComponent(UserErrorRequestsTable).exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').text()).toContain('usageOverview.errorsError')
    wrapper.unmount()
  })

  it('does not expose or request error records while the visibility setting is disabled', async () => {
    appStoreState.cachedPublicSettings = { allow_user_view_error_requests: false }
    const wrapper = mountUsageView()
    await flushPromises()
    expect(wrapper.findAll('button').some(button => button.text() === 'Error records')).toBe(false)
    expect(listMyErrorRequests).not.toHaveBeenCalled()
    expect(wrapper.findComponent(UserUsageSummary).exists()).toBe(true)
    wrapper.unmount()
  })

  it('invalidates pending hidden error records when the date changes before revisiting errors', async () => {
    let resolveOld!: (value: unknown) => void
    listMyErrorRequests.mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    expect(listMyErrorRequests).toHaveBeenCalledTimes(1)
    await wrapper.findAll('button').find(button => button.text() === 'Usage records')!.trigger('click')
    wrapper.findComponent(UserUsageDateRange).vm.$emit('change', {
      startDate: '2026-10-01', endDate: '2026-10-07', preset: null,
    })
    await flushPromises()
    resolveOld({ items: [{ id: 999, model: 'outdated-error' }], total: 999 })
    await flushPromises()
    listMyErrorRequests.mockResolvedValueOnce({ items: [{ id: 3, model: 'current-error' }], total: 1 })
    await wrapper.findAll('button').find(button => button.text() === 'Error records')!.trigger('click')
    await flushPromises()
    expect(listMyErrorRequests).toHaveBeenCalledTimes(2)
    expect(listMyErrorRequests).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: '2026-10-01', end_date: '2026-10-07', timezone: 'Asia/Shanghai', page: 1,
    }))
    expect(wrapper.findComponent(UserErrorRequestsTable).props('rows')).toEqual([{ id: 3, model: 'current-error' }])
    wrapper.unmount()
  })

  it('loads matching logs, expenses and model options without the removed analysis snapshot', async () => {
    const wrapper = mountUsageView()
    await flushPromises()

    expect(wrapper.findComponent(UserUsageRecordsTable).exists()).toBe(true)
    expect(query).toHaveBeenCalled()
    expect(getStats).toHaveBeenCalled()
    expect(getDashboardModels).toHaveBeenCalled()
    expect(getDashboardSnapshotV2).not.toHaveBeenCalled()
    expect(list).toHaveBeenCalledTimes(1)
    expect(list).toHaveBeenCalledWith(1, 100)
    expect(getAvailable).toHaveBeenCalled()
  })

  it('includes API keys after the first page in both record filters and queries by the selected key', async () => {
    const firstPageKeys = Array.from({ length: 100 }, (_, index) => ({
      id: index + 1,
      name: `key-${index + 1}`,
    }))
    const laterKey = { id: 101, name: 'key-from-second-page' }
    list
      .mockResolvedValueOnce({ items: firstPageKeys, total: 101, page: 1, page_size: 100, pages: 2 })
      .mockResolvedValueOnce({ items: [laterKey], total: 101, page: 2, page_size: 100, pages: 2 })

    const wrapper = mountUsageView()
    await flushPromises()

    expect(list.mock.calls).toEqual([[1, 100], [2, 100]])
    const usageKeySelect = wrapper.findAllComponents(Select).find((select) =>
      select.props('options').some((option: SelectOption) => option.label === 'All API Keys')
    )!
    expect(usageKeySelect.props('options')).toHaveLength(102)
    expect(usageKeySelect.props('options')).toContainEqual({ value: laterKey.id, label: laterKey.name })

    query.mockClear()
    usageKeySelect.vm.$emit('update:modelValue', laterKey.id)
    usageKeySelect.vm.$emit('change', laterKey.id)
    await flushPromises()

    expect(query).toHaveBeenCalledWith(
      expect.objectContaining({ api_key_id: laterKey.id, page: 1 }),
      expect.anything()
    )

    await wrapper.findAll('button').find((button) => button.text() === 'Error records')!.trigger('click')
    await flushPromises()

    const errorKeySelect = wrapper.findAllComponents(Select).find((select) =>
      select.props('options').some((option: SelectOption) => option.label === 'All API Keys')
    )!
    expect(errorKeySelect.props('options')).toHaveLength(102)
    expect(errorKeySelect.props('options')).toContainEqual({ value: laterKey.id, label: laterKey.name })

    listMyErrorRequests.mockClear()
    errorKeySelect.vm.$emit('update:modelValue', laterKey.id)
    errorKeySelect.vm.$emit('change', laterKey.id)
    await flushPromises()

    expect(listMyErrorRequests).toHaveBeenCalledWith(
      expect.objectContaining({ api_key_id: laterKey.id, page: 1 })
    )
    expect(list).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('does not request another API key page when the user has no keys', async () => {
    list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 0 })

    const wrapper = mountUsageView()
    await flushPromises()

    expect(list.mock.calls).toEqual([[1, 100]])
    const keySelect = wrapper.findAllComponents(Select).find((select) =>
      select.props('options').some((option: SelectOption) => option.label === 'All API Keys')
    )!
    expect(keySelect.props('options')).toEqual([{ value: null, label: 'All API Keys' }])
    expect(getAvailable).toHaveBeenCalled()
    wrapper.unmount()
  })

  it('stops loading API keys when a later page is empty despite an outdated page count', async () => {
    const firstPageKeys = Array.from({ length: 100 }, (_, index) => ({
      id: index + 1,
      name: `key-${index + 1}`,
    }))
    list
      .mockResolvedValueOnce({ items: firstPageKeys, total: 201, page: 1, page_size: 100, pages: 3 })
      .mockResolvedValueOnce({ items: [], total: 201, page: 2, page_size: 100, pages: 3 })

    const wrapper = mountUsageView()
    await flushPromises()

    expect(list.mock.calls).toEqual([[1, 100], [2, 100]])
    const keySelect = wrapper.findAllComponents(Select).find((select) =>
      select.props('options').some((option: SelectOption) => option.label === 'All API Keys')
    )!
    expect(keySelect.props('options')).toHaveLength(101)
    expect(keySelect.props('options')).toContainEqual({ value: 100, label: 'key-100' })
    wrapper.unmount()
  })

  it('propagates and resets the native compaction filter across page requests', async () => {
    const wrapper = mountUsageView()
    await flushPromises()

    expect((wrapper.vm as any).compactionOptions).toEqual([
      { value: null, label: 'All Requests' },
      { value: true, label: 'Compaction Only' },
    ])

    query.mockClear()
    getStats.mockClear()
    getDashboardModels.mockClear()
    getDashboardSnapshotV2.mockClear()

    ;(wrapper.vm as any).filters.native_compaction_v2 = true
    ;(wrapper.vm as any).applyFilters()
    await flushPromises()

    expect(query).toHaveBeenCalledWith(
      expect.objectContaining({ native_compaction_v2: true }),
      expect.anything()
    )
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({ native_compaction_v2: true }))
    expect(getDashboardModels).not.toHaveBeenCalled()
    expect(getDashboardSnapshotV2).not.toHaveBeenCalled()

    query.mockClear()
    getStats.mockClear()
    getDashboardModels.mockClear()
    getDashboardSnapshotV2.mockClear()

    ;(wrapper.vm as any).resetFilters()
    await flushPromises()

    expect((wrapper.vm as any).filters.native_compaction_v2).toBeNull()
    expect(query).toHaveBeenCalledWith(
      expect.objectContaining({ native_compaction_v2: null }),
      expect.anything()
    )
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({ native_compaction_v2: null }))
    expect(getDashboardModels).toHaveBeenCalledWith(expect.objectContaining({ timezone: 'Asia/Shanghai', model_source: 'requested' }))
    expect(getDashboardModels.mock.calls[0][0]).not.toHaveProperty('native_compaction_v2')
    expect(getDashboardSnapshotV2).not.toHaveBeenCalled()
  })

  it('exports csv with current filters and without admin-only fields', async () => {
    const wrapper = mountUsageView()
    await flushPromises()
    ;(wrapper.vm as any).filters.native_compaction_v2 = true

    let exportedBlob: Blob | null = null
    let csvContent = ''
    const OriginalBlob = globalThis.Blob
    vi.stubGlobal('Blob', vi.fn((parts: BlobPart[], options?: BlobPropertyBag) => {
      csvContent = parts.map((part) => String(part)).join('')
      return new OriginalBlob(parts, options)
    }))
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn((blob: Blob | MediaSource) => {
      exportedBlob = blob as Blob
      return 'blob:usage-export'
    }) as typeof window.URL.createObjectURL
    window.URL.revokeObjectURL = vi.fn(() => {}) as typeof window.URL.revokeObjectURL
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await (wrapper.vm as any).exportToCSV()

    expect(exportedBlob).not.toBeNull()
    expect(query).toHaveBeenCalledWith(expect.objectContaining({
      page_size: 1000,
      sort_by: 'created_at',
      sort_order: 'desc',
      native_compaction_v2: true,
      timezone: 'Asia/Shanghai',
    }))
    expect(clickSpy).toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalled()
    expect(csvContent.startsWith('\uFEFF')).toBe(true)
    expect(csvContent.slice(1)).toBe([
      'Time,API Key Name,Model,Reasoning Effort,Inbound Endpoint,IP Address,Type,Billing Mode,Input Tokens,Output Tokens,Cache Read Tokens,Cache Creation Tokens,Rate Multiplier,Billed Cost,Original Cost,First Token (ms),Duration (ms)',
      '2026-03-08T00:00:00Z,demo-key,gpt-5.4,-,,203.0.113.10,Sync,Token,4057,101,278272,4,1,0.09288300,0.09288300,12,345',
    ].join('\n'))
    expect(csvContent).toContain('IP Address')
    expect(csvContent).toContain('203.0.113.10')
    expect(csvContent).toContain('Billed Cost')
    expect(csvContent).toContain('Original Cost')
    expect(csvContent).not.toContain('Upstream Endpoint')
    expect(csvContent).not.toContain('account_cost')
    expect(csvContent).not.toContain('account_rate_multiplier')

    window.URL.createObjectURL = originalCreateObjectURL
    window.URL.revokeObjectURL = originalRevokeObjectURL
    vi.unstubAllGlobals()
    clickSpy.mockRestore()
  })

  it('keeps formula-injection protection for dangerous exported values', async () => {
    query.mockResolvedValue({
      items: [{ ...usageLog, api_key: { name: '-1+1' } }],
      total: 1,
      pages: 1,
    })
    const wrapper = mountUsageView()
    await flushPromises()

    let csvContent = ''
    const OriginalBlob = globalThis.Blob
    vi.stubGlobal('Blob', vi.fn((parts: BlobPart[], options?: BlobPropertyBag) => {
      csvContent = parts.map((part) => String(part)).join('')
      return new OriginalBlob(parts, options)
    }))
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn(() => 'blob:usage-export') as typeof window.URL.createObjectURL
    window.URL.revokeObjectURL = vi.fn(() => {}) as typeof window.URL.revokeObjectURL
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await (wrapper.vm as any).exportToCSV()

    expect(csvContent).toContain(',"\'-1+1",gpt-5.4,-,')
    expect(showSuccess).toHaveBeenCalled()

    window.URL.createObjectURL = originalCreateObjectURL
    window.URL.revokeObjectURL = originalRevokeObjectURL
    vi.unstubAllGlobals()
    clickSpy.mockRestore()
    wrapper.unmount()
  })

  it('keeps the initial filters, sort, and filename while exporting multiple pages', async () => {
    const pageResponse = { items: [usageLog], total: 1001, pages: 2 }
    query.mockResolvedValue(pageResponse)
    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'usageOverview.technicalView')!.trigger('click')

    const datePicker = wrapper.findComponent(UserUsageDateRange)
    datePicker.vm.$emit('change', { startDate: '2026-03-01', endDate: '2026-03-08', preset: null })
    await flushPromises()

    let resolveFirstPage!: (value: typeof pageResponse) => void
    const firstPage = new Promise<typeof pageResponse>((resolve) => { resolveFirstPage = resolve })
    query.mockClear()
    query.mockImplementation((params, options) =>
      !options && params.page === 1 ? firstPage : Promise.resolve(pageResponse)
    )
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn(() => 'blob:usage-export')
    window.URL.revokeObjectURL = vi.fn()
    let filename = ''
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function () {
      filename = this.download
    })

    try {
      await wrapper.findAll('button').find((button) => button.text() === 'Export CSV')!.trigger('click')
      const initialParams = { ...query.mock.calls[0][0] }
      expect(initialParams).toMatchObject({
        page: 1, page_size: 1000, start_date: '2026-03-01', end_date: '2026-03-08',
        sort_by: 'created_at', sort_order: 'desc',
      })

      const keySelect = wrapper.findAllComponents(Select).find((select) =>
        select.props('options').some((option: SelectOption) => option.label === 'All API Keys')
      )!
      keySelect.vm.$emit('update:modelValue', 1)
      keySelect.vm.$emit('change', 1)
      datePicker.vm.$emit('change', { startDate: '2026-04-01', endDate: '2026-04-08', preset: null })
      wrapper.findComponent(UsageTable).vm.$emit('sort', 'model', 'asc')
      await flushPromises()
      expect(query).toHaveBeenCalledWith(expect.objectContaining({
        api_key_id: 1, start_date: '2026-04-01', end_date: '2026-04-08',
        sort_by: 'model', sort_order: 'asc',
      }), expect.anything())

      resolveFirstPage(pageResponse)
      await flushPromises()

      const exportCalls = query.mock.calls.filter((call) => call.length === 1)
      expect.soft(exportCalls).toEqual([[initialParams], [{ ...initialParams, page: 2 }]])
      expect.soft(filename).toBe('usage_2026-03-01_to_2026-03-08.csv')
      expect(showSuccess).toHaveBeenCalledWith('Export success')
      expect(showError).not.toHaveBeenCalled()
    } finally {
      window.URL.createObjectURL = originalCreateObjectURL
      window.URL.revokeObjectURL = originalRevokeObjectURL
      clickSpy.mockRestore()
      wrapper.unmount()
    }
  })

  it('waits for Retry-After and retries the same page when export is rate limited', async () => {
    const wrapper = mountUsageView()
    await flushPromises()
    query.mockClear()
    query.mockRejectedValueOnce({ status: 429, code: 'RATE_LIMITED', retry_after_seconds: 2 })
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn(() => 'blob:usage-export') as typeof window.URL.createObjectURL
    window.URL.revokeObjectURL = vi.fn(() => {}) as typeof window.URL.revokeObjectURL
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    vi.useFakeTimers()

    try {
      const exporting = (wrapper.vm as any).exportToCSV()
      await vi.advanceTimersByTimeAsync(1999)
      expect(query).toHaveBeenCalledTimes(1)
      expect(showInfo).toHaveBeenCalledWith('Rate limited')

      await vi.advanceTimersByTimeAsync(1)
      await exporting
      expect(query).toHaveBeenCalledTimes(2)
      expect(query.mock.calls[1]).toEqual(query.mock.calls[0])
      expect(clickSpy).toHaveBeenCalled()
      expect(showSuccess).toHaveBeenCalledWith('Export success')
      expect(showError).not.toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
      window.URL.createObjectURL = originalCreateObjectURL
      window.URL.revokeObjectURL = originalRevokeObjectURL
      clickSpy.mockRestore()
      wrapper.unmount()
    }
  })

  it('fails the export after repeated rate limiting', async () => {
    const wrapper = mountUsageView()
    await flushPromises()
    query.mockClear()
    query.mockRejectedValue({ status: 429, code: 'RATE_LIMITED', retry_after_seconds: 1 })
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.useFakeTimers()

    try {
      const exporting = (wrapper.vm as any).exportToCSV()
      await vi.advanceTimersByTimeAsync(3000)
      await exporting
      expect(query).toHaveBeenCalledTimes(4)
      expect(clickSpy).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalledWith('Export failed')
      expect((wrapper.vm as any).exporting).toBe(false)
    } finally {
      vi.useRealTimers()
      vi.mocked(console.error).mockRestore()
      clickSpy.mockRestore()
      wrapper.unmount()
    }
  })

  it('exports historical image rows with image billing mode derived from image_count', async () => {
    query.mockResolvedValue({
      items: [
        {
          ...usageLog,
          request_id: 'req-user-export-legacy-image',
          actual_cost: 0.2,
          total_cost: 0.2,
          input_cost: 0,
          output_cost: 0,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 0,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          image_count: 1,
          model: 'gpt-image-2',
          billing_mode: null,
          ip_address: null,
        },
      ],
      total: 1,
      pages: 1,
    })

    const wrapper = mountUsageView()
    await flushPromises()

    let csvContent = ''
    const OriginalBlob = globalThis.Blob
    vi.stubGlobal('Blob', vi.fn((parts: BlobPart[], options?: BlobPropertyBag) => {
      csvContent = parts.map((part) => String(part)).join('')
      return new OriginalBlob(parts, options)
    }))
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn(() => 'blob:usage-export') as typeof window.URL.createObjectURL
    window.URL.revokeObjectURL = vi.fn(() => {}) as typeof window.URL.revokeObjectURL
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await (wrapper.vm as any).exportToCSV()

    expect(csvContent).toContain('Billing Mode')
    expect(csvContent).toContain('Image')
    expect(csvContent).not.toContain(',Token,0,0,0,0,')

    window.URL.createObjectURL = originalCreateObjectURL
    window.URL.revokeObjectURL = originalRevokeObjectURL
    vi.unstubAllGlobals()
    clickSpy.mockRestore()
  })
})

describe('UsageView subscription feature flag', () => {
  afterEach(() => {
    appStoreState.cachedPublicSettings = { allow_user_view_error_requests: true }
  })

  function billingTypeSelect(wrapper: ReturnType<typeof mountUsageView>) {
    return wrapper.findAllComponents(Select).find((select) =>
      select.props('options').some((option: SelectOption) => option.label === 'Subscription')
    )
  }

  it('offers the balance / subscription billing-type filter by default', async () => {
    const wrapper = mountUsageView()
    await flushPromises()

    expect(billingTypeSelect(wrapper)).toBeDefined()
    expect(wrapper.text()).toContain('Billing type')
    wrapper.unmount()
  })

  it('hides the billing-type filter entirely when subscriptions are disabled', async () => {
    appStoreState.cachedPublicSettings = { allow_user_view_error_requests: true, subscription_enabled: false }

    const wrapper = mountUsageView()
    await flushPromises()

    expect(billingTypeSelect(wrapper)).toBeUndefined()
    expect(wrapper.text()).not.toContain('Billing type')
    wrapper.unmount()
  })
})

describe('UsageView dashboard drilldown URL hydration and platform filter', () => {
  beforeEach(() => {
    routeState.query = {}
    query.mockClear()
    getStats.mockClear()
  })

  afterEach(() => {
    routeState.query = {}
  })

  it('hydrates start_date, end_date, platform, model, and group_id from URL query and auto-expands More filters', async () => {
    routeState.query = {
      start_date: '2026-11-01',
      end_date: '2026-11-07',
      platform: 'openai',
      model: 'gpt-5.4',
      group_id: '5',
    }
    const wrapper = mountUsageView()
    await flushPromises()

    expect(query).toHaveBeenCalledWith(
      expect.objectContaining({
        start_date: '2026-11-01',
        end_date: '2026-11-07',
        platform: 'openai',
        model: 'gpt-5.4',
        group_id: 5,
      }),
      expect.any(Object)
    )
    expect(getStats).toHaveBeenCalledWith(
      expect.objectContaining({
        start_date: '2026-11-01',
        end_date: '2026-11-07',
        platform: 'openai',
        model: 'gpt-5.4',
        group_id: 5,
      })
    )
    // Extra filters auto-expanded because platform & group_id were present
    const moreBtn = wrapper.find('button[aria-controls="usage-more-filters"]')
    expect(moreBtn.attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('.usage-filter-count').text()).toBe('2')

    // Fallback group option #5 is present even when groups API only returns id=1
    const groupSelect = wrapper.findAllComponents(Select).find((s) => s.attributes('id') === 'usage-group')
    expect(groupSelect?.props('options').some((o: SelectOption) => o.value === 5)).toBe(true)
    wrapper.unmount()
  })

  it('ignores impossible calendar dates in URL query without sending invalid dates to API', async () => {
    routeState.query = { start_date: '2026-02-31', platform: 'invalid_platform' }
    const wrapper = mountUsageView()
    await flushPromises()

    expect(query).toHaveBeenCalledTimes(1)
    const sent = query.mock.calls[0][0]
    expect(sent.start_date).not.toBe('2026-02-31')
    expect(sent.platform).toBeUndefined()
    wrapper.unmount()
  })
})
