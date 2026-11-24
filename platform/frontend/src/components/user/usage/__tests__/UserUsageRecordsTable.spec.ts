import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { UsageLog } from '@/types'
vi.mock('vue-i18n', async importOriginal => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
import RecordsTable from '../UserUsageRecordsTable.vue'

enableAutoUnmount(afterEach)
afterEach(() => { document.body.innerHTML = ''; document.body.style.overflow = '' })
const row = { id: 1, model: 'unknown-custom-model-complete-name', actual_cost: 0.0000000012, total_cost: 0.001, created_at: '2026-01-01T16:00:03Z', api_key: { name: 'test key', key: 'must-never-render' }, request_id: 'req-example', billing_mode: 'token', request_type: 'stream', billing_type: 1 } as UsageLog
const baseProps = { rows: [row], loading: false, error: false, sortBy: 'created_at', sortOrder: 'desc' as const }
const global = { stubs: { ModelIcon: true, IpGeoCell: true, Icon: true, transition: true } }

describe('compact usage records', () => {
  it('shows full model name, Beijing seconds and tiny cost, and only supported sort fields', async () => {
    const w = mount(RecordsTable, { props: baseProps, global })
    expect(w.text()).toContain(row.model)
    expect(w.text()).toContain('2026/01/02 00:00:03')
    expect(w.text()).toContain('<$0.0001')
    expect(w.text()).toContain('test key')
    expect(w.text()).not.toContain('must-never-render')
    const sorts = w.findAll('thead button')
    expect(sorts).toHaveLength(2)
    await sorts[0]!.trigger('click')
    await sorts[1]!.trigger('click')
    expect(w.emitted('sort')).toEqual([['created_at', 'asc'], ['model', 'desc']])
    expect(w.find('th').attributes('aria-sort')).toBe('descending')
  })
  it.each([
    [0.066326, '$0.0663'],
    [0.123456, '$0.1235'],
    [1, '$1.0000'],
    [0, '$0.0000'],
    [null, '—'],
  ])('formats the actual charge %s with four decimal places', (actual_cost, expected) => {
    const w = mount(RecordsTable, { props: { ...baseProps, rows: [{ ...row, actual_cost } as UsageLog] }, global })
    expect(w.get('tbody .records-cost strong').text()).toBe(expected)
  })
  it('hides stale rows while loading, then renders an independent retry or empty reset', async () => {
    const w = mount(RecordsTable, { props: { ...baseProps, loading: true }, global })
    expect(w.find('table').exists()).toBe(false)
    expect(w.find('[role="status"]').exists()).toBe(true)
    await w.setProps({ loading: false, error: true })
    expect(w.find('table').exists()).toBe(false)
    expect(w.find('[role="alert"]').exists()).toBe(true)
    await w.find('button').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
    await w.setProps({ error: false, rows: [] })
    await w.find('button').trigger('click')
    expect(w.emitted('reset')).toHaveLength(1)
    expect(w.text()).not.toContain('$0.00')
  })
  it('opens the same accessible drawer and restores keyboard focus on Escape', async () => {
    const app = document.createElement('div'); app.id = 'app'; document.body.append(app)
    const w = mount(RecordsTable, { props: baseProps, global, attachTo: app })
    const opener = w.find('tbody button')
    ;(opener.element as HTMLButtonElement).focus()
    await opener.trigger('click'); await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog).not.toBeNull()
    expect(dialog.textContent).toContain('$0.0000000012')
    expect(dialog.textContent).toContain('usageRecordDetails.subscriptionNote')
    expect(app.inert).toBe(true)
    expect(document.activeElement).toBe(dialog.querySelector('button'))
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    await flushPromises()
    expect(app.inert).toBe(false)
    expect(document.activeElement).toBe(opener.element)
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })
})

it('marks only recorded campaign charges while preserving the actual amount', () => {
  const w = mount(RecordsTable, { props: { ...baseProps, rows: [{ ...row, actual_cost: 0.2, token_campaign: { id: 'blg', revision: 1, phase: 'championship-day', discount: 0.2, pricing_at: row.created_at, original_actual_cost: 1, discount_amount: 0.8 } }, { ...row, id: 2 }] }, global })
  expect(w.findAll('[data-campaign-badge]')).toHaveLength(1)
  expect(w.find('tbody .records-cost strong').text()).toBe('$0.2000')
})
