import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ChannelStatusV1View from '../ChannelStatusV1View.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  status: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/channelMonitor', () => ({ list: mocks.list, status: mocks.status }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: { channel_monitor_enabled: true },
    showError: mocks.showError,
  }),
}))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) =>
    params ? `${key}:${JSON.stringify(params)}` : key }),
}))

enableAutoUnmount(afterEach)
afterEach(() => { vi.useRealTimers(); localStorage.clear() })

function makeItem(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    name: 'main-channel',
    provider: 'openai',
    group_name: 'default',
    primary_model: 'gpt-6.1-sol',
    primary_status: 'operational',
    primary_latency_ms: 1200,
    primary_ping_latency_ms: 40,
    availability_7d: 99.5,
    extra_models: [],
    timeline: [{ status: 'operational', latency_ms: 1200, ping_latency_ms: 40, checked_at: '2026-11-21T08:00:00Z' }],
    ...overrides,
  }
}

function mountView() {
  return shallowMount(ChannelStatusV1View, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        EmptyState: {
          props: ['title', 'description', 'actionText'],
          emits: ['action'],
          template: '<div class="empty" :data-title="title"><button class="empty-action" @click="$emit(\'action\')">act</button></div>',
        },
        MonitorHero: {
          props: ['overallStatus', 'counts', 'providers', 'activeProvider', 'searchQuery', 'autoRefresh', 'lastCheckAt'],
          emits: ['update:window', 'update:provider', 'update:search', 'refresh'],
          template: `<div class="hero" :data-overall="overallStatus" :data-provider="activeProvider" :data-search="searchQuery" :data-counts="JSON.stringify(counts)">
            <button class="to-15d" @click="$emit('update:window', '15d')">15d</button>
            <button class="filter-healthy" @click="$emit('update:provider', 'openai')">filter healthy only</button>
            <button class="search-none" @click="$emit('update:search', 'zzz-none')">search</button>
            <button class="refresh" @click="$emit('refresh')">refresh</button>
          </div>`,
        },
        MonitorCardGrid: {
          props: ['items', 'autoRefreshOn'],
          template: '<div class="grid" :data-count="items.length" :data-autorefresh="String(autoRefreshOn)"></div>',
        },
        MonitorDetailDialog: true,
      },
    },
  })
}

describe('channel status page state semantics', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.resetAllMocks()
    localStorage.clear()
    localStorage.setItem('channel-status-auto-refresh', JSON.stringify({ enabled: false, interval_seconds: 60 }))
  })

  it('shows cards and no stale/error/notification after a successful load', async () => {
    mocks.list.mockResolvedValue({ items: [makeItem()] })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.grid').attributes('data-count')).toBe('1')
    expect(wrapper.find('.empty').exists()).toBe(false)
    // 成功路径里绝不能出现「刷新失败」或全局错误
    expect(wrapper.text()).not.toContain('channelStatus.stale')
    expect(mocks.showError).not.toHaveBeenCalled()
  })

  it('does not report an empty list as operational', async () => {
    mocks.list.mockResolvedValue({ items: [] })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.hero').attributes('data-overall')).toBe('empty')
    expect(wrapper.find('.empty').attributes('data-title')).toBe('channelStatus.empty.title')
  })

  it('shows a retry error state when the first request fails', async () => {
    mocks.list.mockRejectedValue(new Error('boom'))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.empty').attributes('data-title')).toBe('channelStatus.error.title')
  })

  it('keeps cards and marks them stale when a later refresh fails', async () => {
    mocks.list.mockResolvedValueOnce({ items: [makeItem()] })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('.grid').attributes('data-count')).toBe('1')

    mocks.list.mockRejectedValueOnce(new Error('boom'))
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()

    // 旧数据保留，但明确标注不是最新，而不是变成「管理员没配渠道」
    expect(wrapper.find('.grid').attributes('data-count')).toBe('1')
    expect(wrapper.text()).toContain('channelStatus.stale')
    expect(wrapper.find('.empty').exists()).toBe(false)
  })

  it('treats a malformed response as an error instead of an empty channel list', async () => {
    mocks.list.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.empty').attributes('data-title')).toBe('channelStatus.error.title')
  })

  it('keeps the global status degraded when filtering hides the broken channel', async () => {
    mocks.list.mockResolvedValue({
      items: [
        makeItem({ id: 1, provider: 'openai', primary_status: 'operational' }),
        makeItem({ id: 2, provider: 'anthropic', primary_status: 'degraded' }),
      ],
    })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('.hero').attributes('data-overall')).toBe('degraded')

    // 只筛出正常的那条，异常渠道被隐藏
    await wrapper.get('.filter-healthy').trigger('click')
    await flushPromises()

    expect(wrapper.find('.hero').attributes('data-provider')).toBe('openai')
    expect(wrapper.find('.grid').attributes('data-count')).toBe('1')
    // 列表只剩正常渠道，但全站结论不能跟着变绿
    expect(wrapper.find('.hero').attributes('data-overall')).toBe('degraded')
    expect(wrapper.find('.hero').attributes('data-counts')).toContain('"degraded":1')
  })

  it('shows a filter empty state and clears filters without a network call', async () => {
    mocks.list.mockResolvedValue({ items: [makeItem()] })
    const wrapper = mountView()
    await flushPromises()
    expect(mocks.list).toHaveBeenCalledTimes(1)

    await wrapper.get('.search-none').trigger('click')
    await flushPromises()

    expect(wrapper.find('.grid').exists()).toBe(false)
    expect(wrapper.find('.empty').attributes('data-title')).toBe('channelStatus.filter.emptyTitle')

    await wrapper.get('.empty-action').trigger('click')
    await flushPromises()

    // 清空筛选是纯本地操作
    expect(mocks.list).toHaveBeenCalledTimes(1)
    expect(wrapper.find('.grid').attributes('data-count')).toBe('1')
  })

  it('passes the real auto-refresh state down to the cards', async () => {
    mocks.list.mockResolvedValue({ items: [makeItem()] })
    const wrapper = mountView()
    await flushPromises()

    // 持久化的开关是关闭，页面必须把这个真实值传给卡片，而不是默认 true
    expect(wrapper.find('.grid').attributes('data-autorefresh')).toBe('false')
  })

  it('does not request details after a failed list refresh', async () => {
    mocks.list.mockResolvedValue({ items: [makeItem()] })
    mocks.status.mockResolvedValue({
      id: 1,
      models: [{ model: 'gpt-6.1-sol', availability_15d: 90, availability_30d: 88 }],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.to-15d').trigger('click')
    await flushPromises()
    expect(mocks.status).toHaveBeenCalledTimes(1)

    // 列表刷新失败，不应把新详情拼到旧列表上
    mocks.list.mockRejectedValueOnce(new Error('boom'))
    await wrapper.get('.refresh').trigger('click')
    await flushPromises()
    expect(mocks.status).toHaveBeenCalledTimes(1)
  })

  // latest 与 timeline 是后端两条独立查询：历史查询失败会降级为空数组，
  // 所以「没有历史」不能推翻已知的最新状态，也不能被算成状态未知。
  it('keeps an operational channel operational when it has no timeline history', async () => {
    mocks.list.mockResolvedValue({
      items: [makeItem({ primary_status: 'operational', timeline: [] })],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.hero').attributes('data-overall')).toBe('operational')
    expect(wrapper.find('.hero').attributes('data-counts')).toContain('"operational":1')
    expect(wrapper.find('.grid').attributes('data-count')).toBe('1')
  })

  it('reports partial when a status value is not recognised', async () => {
    mocks.list.mockResolvedValue({ items: [makeItem({ primary_status: 'weird-state' })] })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.hero').attributes('data-overall')).toBe('partial')
  })

  it('does not write cache or toast after unmount while a detail request is pending', async () => {
    mocks.list.mockResolvedValue({ items: [makeItem()] })

    let rejectDetail: (err: unknown) => void = () => {}
    mocks.status.mockImplementation(() => new Promise((_, reject) => { rejectDetail = reject }))

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.to-15d').trigger('click')
    await flushPromises()
    expect(mocks.status).toHaveBeenCalledTimes(1)

    const errorsBefore = mocks.showError.mock.calls.length
    wrapper.unmount()
    rejectDetail(new Error('late failure'))
    await flushPromises()

    // 卸载后返回的失败不能变成用户可见的错误提示
    expect(mocks.showError.mock.calls.length).toBe(errorsBefore)
  })
})
