import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { CheckoutInfoResponse } from '@/types/payment'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: object) => (params ? `${key}:${JSON.stringify(params)}` : key) }),
}))

import DashboardOnboardingCard from '../DashboardOnboardingCard.vue'

enableAutoUnmount(afterEach)

describe('DashboardOnboardingCard 零用量新用户3步起飞卡', () => {
  it('根据余额/订阅与密钥数量自动检测完成状态并展示首档满赠', () => {
    const checkout = {
      methods: { alipay: { single_min: 10, single_max: 500, daily_limit: 0, available: true, fee_rate: 0, currency: 'CNY' } },
      global_min: 10,
      global_max: 500,
      plans: [],
      balance_disabled: false,
      balance_recharge_multiplier: 1,
      recharge_fee_rate: 0,
      help_text: '',
      help_image_url: '',
      stripe_publishable_key: '',
      recharge_bonus_mode: 'bonus',
      recharge_bonus_tiers: [{ min_amount: 100, bonus_percent: 10 }],
    } as CheckoutInfoResponse

    const w = mount(DashboardOnboardingCard, {
      props: {
        balance: 50,
        hasActiveSubscription: false,
        totalApiKeys: 1,
        paymentEnabled: true,
        isSimple: false,
        checkoutInfo: checkout,
      },
      global: {
        stubs: {
          Icon: true,
          'router-link': { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })

    const steps = w.findAll('.overview-onboarding-step')
    expect(steps).toHaveLength(3)
    expect(steps[0].classes()).toContain('is-done')
    expect(steps[1].classes()).toContain('is-done')
    expect(w.get('.overview-onboarding-progress').text()).toContain('"done":2,"total":3')
    expect(w.get('.overview-onboarding-bonus').text()).toContain('firstBonusHint')
  })

  it('简易模式精简为2步并隐藏充值与画室入口', () => {
    const w = mount(DashboardOnboardingCard, {
      props: {
        balance: null,
        totalApiKeys: 0,
        isSimple: true,
      },
      global: {
        stubs: {
          Icon: true,
          'router-link': { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })
    expect(w.findAll('.overview-onboarding-step')).toHaveLength(2)
    expect(w.get('.overview-onboarding-progress').text()).toContain('"done":0,"total":2')
    expect(w.find('a[href="/purchase"]').exists()).toBe(false)
    expect(w.find('a[href="/studio"]').exists()).toBe(false)
  })
})
