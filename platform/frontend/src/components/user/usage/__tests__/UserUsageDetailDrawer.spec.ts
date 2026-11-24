import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { UsageLog } from '@/types'
vi.mock('vue-i18n', async importOriginal => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
import DetailDrawer from '../UserUsageDetailDrawer.vue'

enableAutoUnmount(afterEach)
afterEach(() => { document.body.innerHTML = ''; document.body.style.overflow = '' })
const props = { record: { id: 2, model: 'gpt-example', actual_cost: 0.3, total_cost: 0.1, request_id: 'req-copy-example', created_at: '2026-01-01T16:00:03Z', billing_type: 1, duration_ms: null, first_token_ms: null, billing_mode: 'token', input_tokens: 10, image_input_tokens: 0, output_tokens: 2, image_output_tokens: 0, api_key: { name: 'safe-key-name', key: 'api-secret-sentinel' }, upstream_endpoint: 'internal-upstream-sentinel', account: { name: 'account-private-sentinel' }, account_stats_cost: 991.23 } as unknown as UsageLog }
const global = { stubs: { ModelIcon: true, IpGeoCell: true, Icon: true } }

describe('usage record details', () => {
  it('whitelists fields and distinguishes subscription quota and above-standard billing', async () => {
    mount(DetailDrawer, { props, global, attachTo: document.body }); await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    const text = dialog.textContent!
    expect(text).toContain('usageRecordDetails.subscriptionNote')
    expect(text).toContain('usageRecordDetails.aboveStandard')
    expect(text).toContain('$0.2000')
    expect(text).toContain('UTC+8')
    expect(text).toContain('safe-key-name')
    for (const secret of ['api-secret-sentinel', 'internal-upstream-sentinel', 'account-private-sentinel', '991.23']) expect(text).not.toContain(secret)
    expect(text).not.toContain('0 ms')
    expect(text).toContain('—')
  })
  it('copies the request id and exposes a failed copy without claiming success', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    const w = mount(DetailDrawer, { props, global, attachTo: document.body }); await flushPromises()
    const copy = () => [...document.querySelectorAll('button')].find(b => b.textContent?.includes('usageRecordDetails.copy'))!
    copy().click(); await flushPromises()
    expect(writeText).toHaveBeenCalledWith('req-copy-example')
    expect(document.querySelector('[role="status"]')?.textContent).toBe('usageRecordDetails.copied')
    await w.setProps({ record: { ...props.record, request_id: 'req-other' } })
    writeText.mockRejectedValueOnce(new Error('clipboard denied'))
    copy().click(); await flushPromises()
    expect(document.querySelector('[role="alert"]')?.textContent).toContain('usageRecordDetails.copyFailed')
    expect(document.querySelector('[role="status"]')).toBeNull()
  })
  it('keeps video distinct from image units and avoids fabricating unknown timing', async () => {
    mount(DetailDrawer, { props: { record: { ...props.record, billing_mode: 'video', image_count: 3 } }, global, attachTo: document.body }); await flushPromises()
    const text = document.querySelector('[role="dialog"]')!.textContent!
    expect(text).toContain('usageRecordDetails.videoNote')
    expect(text).not.toContain('usageRecordDetails.imageStandardUnit')
    expect(text).not.toContain('tok/s')
  })
})

it('shows frozen campaign billing evidence without recomputing historical charges', async () => {
  mount(DetailDrawer, { props: { record: { ...props.record, actual_cost: 0.2, token_campaign: { id: 'blg', revision: 1, phase: 'championship-day', discount: 0.2, pricing_at: '2026-11-15T00:00:00Z', original_actual_cost: 1, discount_amount: 0.8 } } }, global, attachTo: document.body })
  await flushPromises()
  const evidence = document.querySelector('[data-token-campaign]')!
  expect(evidence).not.toBeNull()
  expect(evidence.textContent).toContain('usageRecordDetails.campaignDay')
  expect(evidence.textContent).toContain('$1.0000')
  expect(evidence.textContent).toContain('$0.8000')
  expect(evidence.textContent).toContain('$0.2000')
  expect(evidence.textContent).toContain('2026/11/15 08:00:00')
})
