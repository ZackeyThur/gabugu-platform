import { describe, expect, it } from 'vitest'
import {
  WEEKEND_RATE_ENDS_AT,
  evaluateWeekendOffer,
  formatObservedRate,
  weekendBannerVisible,
  type WeekendGroupQuote
} from '../weekendRate'

const end = Date.parse(WEEKEND_RATE_ENDS_AT)
const during = Date.parse('2026-11-21T03:00:00+08:00')

function groups(gpt: number, claude: number, discount = 0.9, userRate?: number): WeekendGroupQuote[] {
  return [
    { id: 4, name: 'GPT', rate_multiplier: gpt, subscription_type: 'standard', token_campaign_discount: discount, user_rate_multiplier: userRate },
    { id: 5, name: 'Claude', rate_multiplier: claude, subscription_type: 'standard', token_campaign_discount: discount },
    { id: 23, name: 'DeepSeek', rate_multiplier: 0.4, subscription_type: 'standard', token_campaign_discount: discount }
  ]
}

describe('weekend group-rate offer', () => {
  it('is active only when both live defaults are the new rates inside the window', () => {
    const offer = evaluateWeekendOffer(during, groups(0.22, 1.2))
    expect(offer.status).toBe('active')
    expect(weekendBannerVisible(offer.status)).toBe(true)
    expect(offer.lines.map((line) => line.saveLabel)).toEqual(['约 40%', '约 10%'])
    expect(offer.lines.map((line) => [line.fromLabel, line.toLabel])).toEqual([['0.37', '0.22'], ['1.33', '1.20']])
    expect(offer.campaignDiscount).toBe(0.9)
    expect(offer.lines.map((line) => [line.effectiveFrom, line.effectiveTo])).toEqual([
      ['0.333', '0.198'],
      ['1.197', '1.08']
    ])
  })

  it('does not treat the previous rates or a partial change as live', () => {
    expect(evaluateWeekendOffer(during, groups(0.37, 1.33)).status).toBe('not-yet')
    expect(evaluateWeekendOffer(during, groups(0.22, 1.33)).status).toBe('unexpected')
    expect(evaluateWeekendOffer(during, groups(0.23, 1.2)).status).toBe('unexpected')
    expect(weekendBannerVisible('not-yet')).toBe(false)
    expect(weekendBannerVisible('unexpected')).toBe(false)
  })

  it('stops the promotion at 23:00-eve even if the new rates are still present', () => {
    expect(evaluateWeekendOffer(end, groups(0.22, 1.2)).status).toBe('ended')
    expect(evaluateWeekendOffer(end - 1, groups(0.22, 1.2)).status).toBe('active')
    expect(weekendBannerVisible('ended')).toBe(false)
  })

  it('stays unverified without a server clock or a rate list', () => {
    expect(evaluateWeekendOffer(Number.NaN, groups(0.22, 1.2)).status).toBe('unverified')
    expect(evaluateWeekendOffer(during, null).status).toBe('unverified')
    expect(evaluateWeekendOffer(during, null).campaignDiscount).toBeNull()
    expect(evaluateWeekendOffer(during, null).lines[0].effectiveFrom).toBeNull()
  })

  it('refuses a renamed, duplicate, or non-standard group with the same id', () => {
    const renamed = groups(0.22, 1.2)
    renamed[0] = { ...renamed[0], name: '低价' }
    expect(evaluateWeekendOffer(during, renamed).status).toBe('unexpected')
    const subscription = groups(0.22, 1.2)
    subscription[1] = { ...subscription[1], subscription_type: 'subscription' }
    expect(evaluateWeekendOffer(during, subscription).status).toBe('unexpected')
    expect(evaluateWeekendOffer(during, [...groups(0.22, 1.2), groups(0.22, 1.2)[0]]).status).toBe('unexpected')
  })

  it('matches the group default, not a personal rate', () => {
    const offer = evaluateWeekendOffer(during, groups(0.22, 1.2, 0.9, 0.5))
    expect(offer.status).toBe('active')
    expect(offer.lines[0].userRate).toBe(0.5)
    const personalOnly = groups(0.37, 1.33, 0.9, 0.22)
    expect(evaluateWeekendOffer(during, personalOnly).status).toBe('not-yet')
  })

  it('omits effective prices unless both groups report the same campaign factor', () => {
    const split = groups(0.22, 1.2)
    split[1] = { ...split[1], token_campaign_discount: 1 }
    const offer = evaluateWeekendOffer(during, split)
    expect(offer.status).toBe('active')
    expect(offer.campaignDiscount).toBeNull()
    expect(offer.lines.every((line) => line.effectiveFrom === null)).toBe(true)
  })

  it('keeps the stated savings below the next whole step', () => {
    expect(1 - 0.22 / 0.37).toBeGreaterThan(0.4)
    expect(1 - 0.22 / 0.37).toBeLessThan(0.41)
    expect(1 - 1.2 / 1.33).toBeGreaterThan(0.09)
    expect(1 - 1.2 / 1.33).toBeLessThan(0.1)
    expect(formatObservedRate(0.37 * 0.9)).toBe('0.333')
    expect(formatObservedRate(0.22 * 0.9)).toBe('0.198')
    expect(formatObservedRate(1.33 * 0.9)).toBe('1.197')
    expect(formatObservedRate(1.2 * 0.9)).toBe('1.08')
  })
})
