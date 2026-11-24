import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { AdminUsageLog } from '@/types'
import UsageRecordGrid from '../UsageRecordGrid.vue'

vi.mock('vue-i18n', async original => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
const keys = ['api_key', 'model', 'reasoning_effort', 'endpoint', 'ip_address', 'group', 'stream', 'billing_mode', 'tokens', 'cost', 'latency', 'created_at', 'user_agent']
const columns = keys.map(key => ({ key, label: key, sortable: key === 'created_at' || key === 'model' }))
const row = { id: 12, model: 'long-model-name-remains-complete', created_at: '2026-11-09T06:00:00Z' } as AdminUsageLog
function render() {
  return mount(UsageRecordGrid, { props: { columns, data: [row], defaultSortKey: 'created_at', defaultSortOrder: 'desc' }, slots: Object.fromEntries(keys.map(key => [`cell-${key}`, `<span>rendered-${key}</span>`])), global: { stubs: { ModelIcon: true, Icon: true } } })
}
describe('usage records that fit their container', () => {
  it('renders every selected field exactly once, including optional fields, and honors hidden columns', async () => {
    const wrapper = render()
    for (const key of keys) {
      expect(wrapper.findAll(`[data-field="${key}"]`)).toHaveLength(1)
      expect(wrapper.get(`[data-field="${key}"]`).text()).toContain(`rendered-${key}`)
    }
    await wrapper.setProps({ columns: columns.filter(column => column.key !== 'model' && column.key !== 'user_agent') })
    expect(wrapper.find('[data-field="model"]').exists()).toBe(false)
    expect(wrapper.find('[data-field="user_agent"]').exists()).toBe(false)
    await wrapper.get('.technical-record-meta button').trigger('click')
    expect(wrapper.emitted('details')).toEqual([[row]])
  })
  it('keeps server sorting and hides obsolete records during loading', async () => {
    const wrapper = render()
    await wrapper.get('.record-grid-sort button').trigger('click')
    expect(wrapper.emitted('sort')).toEqual([['model', 'desc']])
    await wrapper.setProps({ loading: true })
    expect(wrapper.find('.technical-record').exists()).toBe(false)
    expect(wrapper.get('[role="status"]').exists()).toBe(true)
  })
})
