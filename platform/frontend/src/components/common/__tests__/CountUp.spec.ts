import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CountUp from '../CountUp.vue'
import { overviewMoney } from '@/utils/dashboardOverview'

describe('CountUp', () => {
  it('renders exact target value immediately in test environment', () => {
    const wrapper = mount(CountUp, {
      props: { value: 428.5, formatter: overviewMoney },
    })
    expect(wrapper.text()).toBe('$428.50')
  })

  it('preserves tiny values with custom formatter and updates reactively', async () => {
    const wrapper = mount(CountUp, {
      props: { value: 0.00012, formatter: overviewMoney },
    })
    expect(wrapper.text()).toBe('<$0.01')

    await wrapper.setProps({ value: 1200 })
    expect(wrapper.text()).toBe('$1,200.00')
  })

  it('supports fixed decimals without locale commas when locale is false', () => {
    const wrapper = mount(CountUp, {
      props: { value: 1500.5, decimals: 2, locale: false },
    })
    expect(wrapper.text()).toBe('1500.50')
  })

  it('renders fallback dash for nullish or non-finite values', () => {
    const wrapper = mount(CountUp, {
      props: { value: null },
    })
    expect(wrapper.text()).toBe('—')
  })

  it('splits optical decimals cleanly when splitDecimals is enabled and guards <$0.01 sentinels', () => {
    const standard = mount(CountUp, {
      props: { value: 86.5, formatter: overviewMoney, splitDecimals: true },
    })
    expect(standard.text()).toBe('$86.50')
    expect(standard.get('.g-countup-major').text()).toBe('$86')
    expect(standard.get('.g-countup-cents').text()).toBe('.50')

    const tiny = mount(CountUp, {
      props: { value: 0.0004, formatter: overviewMoney, splitDecimals: true },
    })
    expect(tiny.text()).toBe('<$0.01')
    expect(tiny.find('.g-countup-cents').exists()).toBe(false)
  })
})
