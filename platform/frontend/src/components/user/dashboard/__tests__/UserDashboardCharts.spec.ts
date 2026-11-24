import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { APIKeyStat, GroupStat, ModelStat, TrendDataPoint } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: object) => (params ? `${key}:${JSON.stringify(params)}` : key) }),
}))

const routerPush = vi.hoisted(() => vi.fn())
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

// Stub vue-chartjs canvas wrappers so jsdom never invokes HTMLCanvasElement.getContext
vi.mock('vue-chartjs', () => ({
  Bar: { name: 'Bar', props: ['data', 'options', 'plugins'], template: '<div data-testid="chart-bar" />' },
  Line: { name: 'Line', props: ['data', 'options', 'plugins'], template: '<div data-testid="chart-line" />' },
  Doughnut: { name: 'Doughnut', props: ['data', 'options'], template: '<div data-testid="chart-doughnut" />' },
}))

import Charts from '../UserDashboardCharts.vue'
import Select from '@/components/common/Select.vue'

enableAutoUnmount(afterEach)

const sampleModels: ModelStat[] = [1, 7, 3, 6, 2, 5, 4].map(
  (n) =>
    ({
      model: `m${n}`,
      requests: n * 10,
      input_tokens: 100 * n,
      output_tokens: 50 * n,
      cache_creation_tokens: 0,
      cache_read_tokens: 300 * n,
      total_tokens: 450 * n,
      actual_cost: n,
      cost: 100 - n,
    }) as ModelStat
)

const sampleGroups: GroupStat[] = [
  { group_id: 10, group_name: 'Claude Pro', requests: 20, total_tokens: 50000, cost: 12, actual_cost: 9 },
  { group_id: 0, group_name: '', requests: 2, total_tokens: 1000, cost: 1, actual_cost: 1 },
]

const sampleKeys: APIKeyStat[] = [
  { api_key_id: 1, key_name: 'prod-key', deleted: false, requests: 18, total_tokens: 40000, cost: 10, actual_cost: 8 },
  { api_key_id: 2, key_name: 'old-key', deleted: true, requests: 4, total_tokens: 10000, cost: 3, actual_cost: 2 },
]

function render(over: Partial<InstanceType<typeof Charts>['$props']> = {}) {
  routerPush.mockReset()
  return mount(Charts, {
    props: {
      startDate: '2026-11-03',
      endDate: '2026-11-09',
      today: '2026-11-09',
      granularity: 'day',
      trend: [
        {
          date: '2026-11-05',
          requests: 12,
          input_tokens: 200,
          output_tokens: 100,
          cache_creation_tokens: 0,
          cache_read_tokens: 600,
          total_tokens: 900,
          actual_cost: 0.000123,
          cost: 1.5,
        } as TrendDataPoint,
      ],
      models: sampleModels,
      groups: sampleGroups,
      apiKeys: sampleKeys,
      trendLoading: false,
      modelsLoading: false,
      trendError: false,
      modelsError: false,
      filterApiKeys: [{ id: 1, name: 'prod-key' } as any],
      filterGroups: [{ id: 10, name: 'Claude Pro' } as any],
      ...over,
    },
    global: {
      stubs: {
        Icon: true,
        ModelIcon: true,
        'router-link': { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
      },
    },
  })
}

describe('精工分析台趋势与分布组件', () => {
  it('默认7天消费柱图支持4指标切换、柱/面积切换以及键盘焦点切换', async () => {
    const w = render()
    expect(w.find('[data-testid="chart-bar"]').exists()).toBe(true)
    expect(w.get('.overview-chart-summary strong').text()).toBe('<$0.01')

    // Switch to area style -> Line chart
    const styleBtns = w.findAll('.overview-style-btn')
    await styleBtns[1].trigger('click')
    expect(w.find('[data-testid="chart-line"]').exists()).toBe(true)

    // Switch to Tokens -> stacked Bar chart
    const metricTabs = w.findAll('.overview-trend-head [role="tab"]')
    await metricTabs[1].trigger('click')
    expect(w.find('[data-testid="chart-bar"]').exists()).toBe(true)

    // Switch to Cache hit rate -> Line chart
    await metricTabs[3].trigger('click')
    expect(w.find('[data-testid="chart-line"]').exists()).toBe(true)
    expect(w.get('.overview-chart-summary strong').text()).toBe('75.0%')

    // Keyboard navigation on canvas region updates focused bucket and Enter drills into /usage
    const region = w.get('.overview-trend-canvas-wrap')
    await region.trigger('keydown', { key: 'Home' })
    expect(w.get('.overview-selected-day').text()).toContain('2026-11-03')
    await region.trigger('keydown', { key: 'Enter' })
    expect(routerPush).toHaveBeenCalledWith({
      path: '/usage',
      query: { start_date: '2026-11-03', end_date: '2026-11-03' },
    })
  })

  it('今日按小时/7天/30天/自定义与密钥分组筛选发出正确事件并拒绝非法自定义范围', async () => {
    const w = render()
    const periodBtns = w.findAll('.overview-periods button')
    // 0: Today (hour), 1: 7d, 2: 30d, 3: Custom
    await periodBtns[0].trigger('click')
    expect(w.emitted('rangeChange')?.at(-1)).toEqual(['2026-11-09', '2026-11-09', 'hour'])

    await periodBtns[2].trigger('click')
    expect(w.emitted('rangeChange')?.at(-1)).toEqual(['2026-10-11', '2026-11-09', 'day'])

    await periodBtns[1].trigger('click')
    expect(w.emitted('rangeChange')?.at(-1)).toEqual(['2026-11-03', '2026-11-09', 'day'])

    await periodBtns[3].trigger('click')
    const inputs = w.findAll('input')
    await inputs[0].setValue('2026-11-08')
    await inputs[1].setValue('2026-11-07')
    await w.get('form').trigger('submit')
    expect(w.get('[role="alert"]').text()).toContain('invalidRange')
    expect(w.emitted('rangeChange')).toHaveLength(3)

    await inputs[0].setValue('2026-10-01')
    await inputs[1].setValue('2026-10-03')
    await w.get('form').trigger('submit')
    expect(w.emitted('rangeChange')?.at(-1)).toEqual(['2026-10-01', '2026-10-03', 'day'])

    // API key & group custom Select components (zero raw <select> elements)
    expect(w.findAll('select')).toHaveLength(0)
    const selects = w.findAllComponents(Select)
    expect(selects).toHaveLength(2)
    selects[0].vm.$emit('update:modelValue', 1)
    await w.vm.$nextTick()
    expect(w.emitted('filterChange')?.at(-1)).toEqual([{ apiKeyId: 1 }])
    selects[1].vm.$emit('update:modelValue', 10)
    await w.vm.$nextTick()
    expect(w.emitted('filterChange')?.at(-1)).toEqual([{ groupId: 10 }])
  })

  it('分布薄环图支持模型/分组/密钥三视角、前5+其他展开折叠，且已删除密钥禁止下钻', async () => {
    const w = render()
    expect(w.find('[data-testid="chart-doughnut"]').exists()).toBe(true)
    expect(w.findAll('.overview-model-name').map((x) => x.text())).toEqual([
      'm7',
      'm6',
      'm5',
      'm4',
      'm3',
      'dashboard.overview.otherModels',
    ])
    expect(w.findAll('.overview-model-row')[0].text()).toContain('25.0%')
    expect(w.findAll('.overview-model-row').at(-1)?.text()).toContain('$3.00')

    // Click first model -> drills to /usage?model=m7
    await w.findAll('.overview-model-row')[0].trigger('click')
    expect(routerPush).toHaveBeenLastCalledWith({
      path: '/usage',
      query: { start_date: '2026-11-03', end_date: '2026-11-09', model: 'm7' },
    })

    // Expand / collapse
    await w.get('.overview-expand-models').trigger('click')
    expect(w.findAll('.overview-model-row')).toHaveLength(7)
    expect(w.get('.overview-expand-models').attributes('aria-expanded')).toBe('true')

    // Switch to Group tab -> collapses & drills valid group only
    const dimTabs = w.findAll('.overview-breakdown-tabs [role="tab"]')
    await dimTabs[1].trigger('click')
    expect(w.findAll('.overview-model-name').map((x) => x.text())).toEqual([
      'Claude Pro',
      'dashboard.overview.unassignedGroup',
    ])

    // Switch to API Key tab -> deleted key renders badge & does NOT drill down
    await dimTabs[2].trigger('click')
    const keyRows = w.findAll('.overview-model-row')
    expect(keyRows).toHaveLength(2)
    expect(keyRows[1].text()).toContain('deletedKeyTag')
    expect(keyRows[1].classes()).toContain('is-disabled-drill')

    routerPush.mockClear()
    expect(keyRows[1].attributes('tabindex')).toBeUndefined()
    await keyRows[1].trigger('click')
    await keyRows[1].trigger('keydown.enter')
    expect(routerPush).not.toHaveBeenCalled()

    expect(keyRows[0].attributes('tabindex')).toBe('0')
    expect(keyRows[0].attributes('role')).toBe('button')
    await keyRows[0].trigger('keydown.enter')
    expect(routerPush).toHaveBeenCalledWith({
      path: '/usage',
      query: { start_date: '2026-11-03', end_date: '2026-11-09', api_key_id: '1' },
    })
  })

  it('趋势与分布错误独立隔离并可独立重试', async () => {
    const w = render()
    await w.setProps({ trendError: true })
    expect(w.findAll('.overview-model-row')).toHaveLength(6)
    await w.get('.overview-chart-panel [role="alert"] button').trigger('click')
    expect(w.emitted('retryTrend')).toHaveLength(1)

    await w.setProps({ trendError: false, modelsError: true })
    expect(w.find('[data-testid="chart-bar"]').exists()).toBe(true)
    await w.get('.overview-ranking-panel [role="alert"] button').trigger('click')
    expect(w.emitted('retryModels')).toHaveLength(1)
  })

  it('分布切到 Token 指标时保留 K/M 紧凑单位后缀不截断', async () => {
    const w = render()
    const metricBtns = w.findAll('.overview-breakdown-metrics button')
    // 0: Cost, 1: Token, 2: Requests
    await metricBtns[1].trigger('click')
    expect(w.get('.overview-ring-center-value').text()).toMatch(/[KM]$/)
    expect(w.findAll('.overview-model-top strong')[0].text()).toMatch(/3\.2K|3\.1K/)
  })

  it('启用筛选后高亮当前筛选器并展示一键重置按钮', async () => {
    const w = render({ apiKeyId: 1 })
    expect(w.findAll('.overview-filter-select-wrap')[0].classes()).toContain('is-filtered')
    const resetBtn = w.get('.overview-filter-reset')
    await resetBtn.trigger('click')
    expect(w.emitted('filterChange')?.at(-1)).toEqual([{ apiKeyId: null, groupId: null }])
  })
})
