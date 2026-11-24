import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import Summary from '../UserUsageSummary.vue'
import type { UsageStatsResponse } from '@/types'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => `${key} ${Object.values(params || {}).join(' ')}`.trim() }) }))

enableAutoUnmount(afterEach)
const stats = { total_cost: 10, total_actual_cost: 4, total_requests: 109 } as UsageStatsResponse
function render(data: Partial<{ stats: UsageStatsResponse | null; loading: boolean; error: boolean }> = {}) {
  return mount(Summary, {
    props: { stats, loading: false, error: false, ...data },
    global: { stubs: { DashboardDrawer: { props: ['open'], template: '<aside v-if="open"><slot /></aside>' }, Icon: true } },
  })
}
describe('usage charges summary', () => {
  it('compares all filtered charges with the site standard and opens precise billing explanation', async () => {
    const wrapper = render()
    expect(wrapper.get('[data-testid="usage-total-actual"]').text()).toBe('$4.00')
    expect(wrapper.get('[data-testid="usage-total-discount"]').text()).toBe('$6.00')
    expect(wrapper.text()).toContain('usageOverview.recordCount 109')
    await wrapper.get('button').trigger('click')
    expect(wrapper.get('aside').text()).toContain('$10.0000')
    expect(wrapper.get('aside').text()).toContain('usageOverview.billingExplanation')
    expect(wrapper.get('aside').text()).toContain('dashboard.overview.subscriptionExplanation')
  })
  it('shows an above-standard charge honestly and preserves tiny charges', () => {
    const wrapper = render({ stats: { ...stats, total_cost: 0, total_actual_cost: 0.00003 } })
    expect(wrapper.text()).toContain('usageOverview.aboveStandard')
    expect(wrapper.get('[data-testid="usage-total-actual"]').text()).toBe('<$0.01')
    expect(wrapper.get('[data-testid="usage-total-discount"]').text()).toBe('<$0.01')
    expect(wrapper.text()).not.toContain('Infinity')
  })
  it('hides stale charges while loading and displays a retry instead of a false zero', async () => {
    const wrapper = render({ loading: true })
    expect(wrapper.find('[data-testid="usage-total-actual"]').exists()).toBe(false)
    await wrapper.setProps({ loading: false, error: true, stats: null })
    expect(wrapper.get('[role="alert"]').text()).toContain('usageOverview.statsError')
    expect(wrapper.text()).not.toContain('$0.00')
    await wrapper.get('[role="alert"] button').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })
})
