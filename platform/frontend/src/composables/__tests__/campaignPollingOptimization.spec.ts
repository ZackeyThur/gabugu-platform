import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'

const getBillingCampaignMock = vi.fn()
const getModelPlazaMock = vi.fn()

vi.mock('@/api/billingCampaign', () => ({
  getBillingCampaign: () => getBillingCampaignMock()
}))

vi.mock('@/api/modelPlaza', () => ({
  getModelPlaza: () => getModelPlazaMock()
}))

function setDocumentHidden(hidden: boolean) {
  Object.defineProperty(document, 'hidden', {
    configurable: true,
    get: () => hidden
  })
}

describe('campaign & weekend rate polling optimization', () => {
  beforeEach(() => {
    vi.useFakeTimers({
      toFake: ['Date', 'performance', 'setTimeout', 'clearTimeout', 'setInterval', 'clearInterval']
    })
    vi.resetModules()
    getBillingCampaignMock.mockReset()
    getModelPlazaMock.mockReset()
    setDocumentHidden(false)
  })

  afterEach(() => {
    setDocumentHidden(false)
    vi.useRealTimers()
  })

  it('coalesces simultaneous /billing-campaign calls and stops polling after weekend end', async () => {
    getBillingCampaignMock.mockResolvedValue({
      campaign: null,
      server_now: '2026-11-23T02:00:00+08:00',
      status: 'disabled'
    })

    const { useBillingCampaign } = await import('../useBillingCampaign')
    const { useWeekendRate } = await import('../useWeekendRate')

    const Host = defineComponent({
      setup() {
        const campaign = useBillingCampaign()
        const weekend = useWeekendRate()
        return { campaign, weekend }
      },
      render: () => h('div')
    })

    const wrapper = mount(Host)
    await vi.advanceTimersByTimeAsync(10)

    // Co-mounted banners must coalesce into a single API request
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(1)
    expect(getModelPlazaMock).not.toHaveBeenCalled()

    // Advancing 90s while both campaigns are inactive/ended must not trigger 30s polls
    await vi.advanceTimersByTimeAsync(90_000)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('pauses active campaign polling while tab is hidden and refreshes when visible again', async () => {
    let serverEpoch = Date.parse('2026-11-15T13:00:00+08:00')
    getBillingCampaignMock.mockImplementation(() =>
      Promise.resolve({
        campaign: {
          id: 'blg-20261115',
          title: 'BLG夺冠，GABUGU兑现。',
          pricing_basis: 'site_standard',
          starts_at: '2026-11-15T12:00:00+08:00',
          day_ends_at: '2026-11-16T12:00:00+08:00',
          ends_at: '2026-11-23T12:00:00+08:00',
          groups: []
        },
        server_now: new Date(serverEpoch).toISOString(),
        status: 'championship-day'
      })
    )

    const { useBillingCampaign } = await import('../useBillingCampaign')
    const Host = defineComponent({
      setup() {
        return useBillingCampaign()
      },
      render: () => h('div')
    })

    const wrapper = mount(Host)
    await vi.advanceTimersByTimeAsync(10)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(1)

    // Switch tab to background: no API calls during 2 minutes hidden
    setDocumentHidden(true)
    serverEpoch += 120_000
    await vi.advanceTimersByTimeAsync(120_000)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(1)

    // Return to foreground: immediately re-verifies once
    setDocumentHidden(false)
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(10)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(2)

    wrapper.unmount()
  })

  it('transitions phase and discount to ended immediately when crossing ends_at in foreground or after hidden', async () => {
    getBillingCampaignMock.mockResolvedValue({
      campaign: {
        id: 'blg-20261115',
        title: 'BLG夺冠，GABUGU兑现。',
        pricing_basis: 'site_standard',
        starts_at: '2026-11-15T12:00:00+08:00',
        day_ends_at: '2026-11-16T12:00:00+08:00',
        ends_at: '2026-11-23T12:00:00+08:00',
        groups: [{ id: 4, day_discount: 0.7, week_discount: 0.9 }]
      },
      server_now: '2026-11-23T11:59:58+08:00',
      status: 'celebration-week'
    })

    const { useBillingCampaign } = await import('../useBillingCampaign')
    let state!: ReturnType<typeof useBillingCampaign>
    const Host = defineComponent({
      setup() {
        state = useBillingCampaign()
        return {}
      },
      render: () => h('div')
    })

    const wrapper = mount(Host)
    await vi.advanceTimersByTimeAsync(10)
    expect(state.phase.value).toBe('celebration-week')
    expect(state.discountForGroup(4)).toBe(0.9)

    // Advance 3s in foreground across ends_at -> must transition immediately to 'ended'
    await vi.advanceTimersByTimeAsync(3_000)
    expect(state.phase.value).toBe('ended')
    expect(state.discountForGroup(4)).toBe(1)
    expect(state.remainingSeconds.value).toBe(0)
    wrapper.unmount()

    // Also test crossing ends_at while hidden and returning within < 10 minutes
    vi.resetModules()
    const mod2 = await import('../useBillingCampaign')
    let state2!: ReturnType<typeof mod2.useBillingCampaign>
    const Host2 = defineComponent({
      setup() {
        state2 = mod2.useBillingCampaign()
        return {}
      },
      render: () => h('div')
    })
    const wrapper2 = mount(Host2)
    await vi.advanceTimersByTimeAsync(10)
    expect(state2.phase.value).toBe('celebration-week')

    setDocumentHidden(true)
    await vi.advanceTimersByTimeAsync(10_000)
    setDocumentHidden(false)
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(10)

    expect(state2.phase.value).toBe('ended')
    expect(state2.discountForGroup(4)).toBe(1)
    wrapper2.unmount()
  })

  it('backs off to 30s and coalesces retries when /billing-campaign fails', async () => {
    getBillingCampaignMock.mockRejectedValue(new Error('503 Service Unavailable'))

    const { useBillingCampaign } = await import('../useBillingCampaign')
    const { useWeekendRate } = await import('../useWeekendRate')

    const Host = defineComponent({
      setup() {
        const campaign = useBillingCampaign()
        const weekend = useWeekendRate()
        return { campaign, weekend }
      },
      render: () => h('div')
    })

    const wrapper = mount(Host)
    await vi.advanceTimersByTimeAsync(10)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(1)

    // Must NOT hammer every 1s during the 29s backoff window
    await vi.advanceTimersByTimeAsync(25_000)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(1)

    // After 30s, retries once (coalesced across both composables)
    await vi.advanceTimersByTimeAsync(6_000)
    expect(getBillingCampaignMock).toHaveBeenCalledTimes(2)

    wrapper.unmount()
  })
})
