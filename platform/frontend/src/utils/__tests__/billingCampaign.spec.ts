import { describe, expect, it } from 'vitest'
import type { BillingCampaign } from '@/api/billingCampaign'
import { campaignDiscountForGroup, campaignPhaseAt } from '../billingCampaign'

const campaign: BillingCampaign = {
  id: 'blg-20261115', title: 'BLG夺冠，GABUGU兑现', pricing_basis: 'site_standard',
  starts_at: '2026-11-15T14:00:00Z',
  day_ends_at: '2026-11-16T14:00:00Z',
  ends_at: '2026-11-23T14:00:00Z',
  groups: [
    { id: 4, name: 'GPT', day_discount: 0.2, week_discount: 0.9 },
    { id: 5, name: 'Claude', day_discount: 0.5, week_discount: 0.9 }
  ]
}

describe('campaign display follows server time and explicit group membership', () => {
  it('switches at the inclusive start and exclusive phase/end boundaries', () => {
    const start = Date.parse(campaign.starts_at)
    const dayEnd = Date.parse(campaign.day_ends_at)
    const end = Date.parse(campaign.ends_at)
    expect(campaignPhaseAt(campaign, start - 1)).toBe('scheduled')
    expect(campaignPhaseAt(campaign, start)).toBe('championship-day')
    expect(campaignPhaseAt(campaign, dayEnd - 1)).toBe('championship-day')
    expect(campaignPhaseAt(campaign, dayEnd)).toBe('celebration-week')
    expect(campaignPhaseAt(campaign, end - 1)).toBe('celebration-week')
    expect(campaignPhaseAt(campaign, end)).toBe('ended')
  })

  it('never discounts a different group carrying the same provider or model', () => {
    expect(campaignDiscountForGroup(campaign, 'championship-day', 4)).toBe(0.2)
    expect(campaignDiscountForGroup(campaign, 'championship-day', 25)).toBe(1)
    expect(campaignDiscountForGroup(campaign, 'championship-day', 35)).toBe(1)
  })

  it('uses the week discount alone and restores regular prices outside the windows', () => {
    expect(campaignDiscountForGroup(campaign, 'celebration-week', 4)).toBe(0.9)
    expect(campaignDiscountForGroup(campaign, 'celebration-week', 5)).toBe(0.9)
    expect(campaignDiscountForGroup(campaign, 'scheduled', 4)).toBe(1)
    expect(campaignDiscountForGroup(campaign, 'ended', 4)).toBe(1)
    expect(campaignDiscountForGroup(null, 'unavailable', 4)).toBe(1)
  })

  it('does not show an active offer for invalid dates, overlaps or invalid discounts', () => {
    expect(campaignPhaseAt({ ...campaign, day_ends_at: campaign.starts_at }, Date.parse(campaign.starts_at))).toBe('unavailable')
    expect(campaignPhaseAt(campaign, Number.NaN)).toBe('unavailable')
    expect(campaignDiscountForGroup({ ...campaign, groups: [{ ...campaign.groups[0]!, day_discount: -0.2 }] }, 'championship-day', 4)).toBe(1)
  })
})
