import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import zh from '@/i18n/locales/zh'
import type { UserDashboardStats } from '@/api/usage'
import Savings from '../UserDashboardSavings.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => {
  const message = key.split('.').reduce<unknown>((node, part) => node && typeof node === 'object' ? (node as Record<string, unknown>)[part] : undefined, zh)
  return typeof message === 'string' ? message.replace(/\{(\w+)\}/g, (_, name: string) => String(params?.[name] ?? '')) : key
} }) }))
enableAutoUnmount(afterEach)
function render(standard = 10, actual = 6) { return mount(Savings, { props: { stats: { total_cost: standard, total_actual_cost: actual, today_cost: .001, today_actual_cost: .002 } as UserDashboardStats, loading: false, error: false, subscriptionEnabled: false }, global: { stubs: { Icon: true, teleport: true } } }) }
describe('计费优惠真实文案', () => {
  it('展示本站标准计费优惠并解释官方API不可比边界', async () => {
    const w = render()
    expect(w.text()).toContain('本站标准计费')
    expect(w.get('.overview-savings-value').text()).toContain('$4.00')
    expect(w.get('.overview-savings-value').text()).toContain('40.0')
    await w.get('.overview-savings-explain').trigger('click')
    expect(w.get('[role="dialog"]').text()).toContain('并非相较官方 API 的节省金额')
  })
  it('零差额不隐藏，负差额显示超出标准与带符号精确值', async () => {
    const zero = render(1, 1)
    expect(zero.get('.overview-savings-value').text()).toBe('$0.00')
    const negative = render(.001, .002)
    expect(negative.get('.overview-savings-value').text()).toBe('<$0.01')
    expect(negative.get('.overview-savings-label').text()).toContain('高于')
    await negative.get('.overview-savings-explain').trigger('click')
    expect(negative.get('[role="dialog"]').text()).toContain('-$0.0010')
  })
  it('加载/错误不把未知优惠显示成零且可重试', async () => {
    const w = render()
    await w.setProps({ error: true })
    expect(w.find('.overview-savings-value').exists()).toBe(false)
    await w.get('[role="alert"] button').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
    await w.setProps({ error: false, loading: true })
    expect(w.find('.overview-savings-comparison').exists()).toBe(false)
  })
  it('展示提示词缓存命中率环形证明并支持紧凑模式', () => {
    const w = mount(Savings, {
      props: {
        stats: {
          total_cost: 20,
          total_actual_cost: 14,
          today_cost: 2,
          today_actual_cost: 1.4,
          total_input_tokens: 2000,
          total_cache_creation_tokens: 500,
          total_cache_read_tokens: 7500,
        } as UserDashboardStats,
        loading: false,
        error: false,
        subscriptionEnabled: false,
        compact: true,
      },
      global: { stubs: { Icon: true, teleport: true } },
    })
    expect(w.get('.overview-savings').classes()).toContain('is-compact')
    expect(w.get('.overview-cache-arc-pct').text()).toBe('75%')
    expect(w.get('.overview-cache-proof').text()).toContain('提示词缓存命中率')
    expect(w.findAll('.overview-chrono-ticks line')).toHaveLength(24)
    expect(w.findAll('.overview-chrono-ticks line.is-cardinal')).toHaveLength(4)
  })

  it('统一差额对比条支持悬停高亮实付占比与已省区间', async () => {
    const w = render(100, 60)
    expect(w.get('.overview-diff-actual').attributes('style')).toContain('60%')
    expect(w.get('.overview-diff-saved').attributes('style')).toContain('40%')

    await w.get('.overview-diff-saved').trigger('mouseenter')
    expect(w.get('.overview-diff-labels').text()).toContain('$40.00')
    expect(w.get('.overview-diff-labels').text()).toContain('40.0%')

    await w.get('.overview-diff-actual').trigger('mouseenter')
    expect(w.get('.overview-diff-labels').text()).toContain('(60.0%)')
  })
})
