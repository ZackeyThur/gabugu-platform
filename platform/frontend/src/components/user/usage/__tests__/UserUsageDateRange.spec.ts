import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import Range from '../UserUsageDateRange.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

enableAutoUnmount(afterEach)
afterEach(() => { vi.useRealTimers() })
function render() {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2026-12-31T18:00:00Z'))
  return mount(Range, { props: { startDate: '2027-01-01', endDate: '2027-01-01' } })
}
describe('usage date range', () => {
  it('uses Beijing today across a year boundary and includes exactly seven days', async () => {
    const wrapper = render()
    expect(wrapper.findAll('button')[0]!.attributes('aria-pressed')).toBe('true')
    await wrapper.findAll('button')[1]!.trigger('click')
    expect(wrapper.emitted('change')).toEqual([[{ startDate: '2026-12-26', endDate: '2027-01-01', preset: '7days' }]])
  })
  it('keeps custom dates as a draft until valid apply', async () => {
    const wrapper = render()
    await wrapper.findAll('button')[3]!.trigger('click')
    const fields = wrapper.findAll('input')
    await fields[0]!.setValue('2027-01-02')
    expect(wrapper.get('[type="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('change')).toBeUndefined()
    await fields[0]!.setValue('2026-12-01')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('change')).toEqual([[{ startDate: '2026-12-01', endDate: '2027-01-01', preset: null }]])
  })
})
