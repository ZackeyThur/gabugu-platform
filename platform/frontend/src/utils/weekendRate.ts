/** Weekend default-rate cut. Active only when the server clock and live group rates both match. */

export const WEEKEND_RATE_ENDS_AT = '2026-11-23T00:00:00+08:00'

export type WeekendOfferStatus = 'active' | 'not-yet' | 'ended' | 'unverified' | 'unexpected'

export interface WeekendGroupQuote {
  id: number
  name: string
  /** Group default multiplier. Personal overrides must not satisfy this check. */
  rate_multiplier: number
  subscription_type?: string
  token_campaign_discount?: number
  user_rate_multiplier?: number
}

export interface WeekendOfferLine {
  id: number
  label: string
  fromLabel: string
  toLabel: string
  saveLabel: '约 40%' | '约 10%'
  liveRate: number | null
  userRate: number | null
  effectiveFrom: string | null
  effectiveTo: string | null
}

export interface WeekendOffer {
  status: WeekendOfferStatus
  endsAt: string
  lines: WeekendOfferLine[]
  /** Shared live campaign factor, only when both target groups report the same one. */
  campaignDiscount: number | null
}

const TARGETS = [
  { id: 4, token: 'gpt', label: 'GPT 分组', from: 0.37, to: 0.22, fromLabel: '0.37', toLabel: '0.22', saveLabel: '约 40%' as const },
  { id: 5, token: 'claude', label: 'Claude 正价分组', from: 1.33, to: 1.2, fromLabel: '1.33', toLabel: '1.20', saveLabel: '约 10%' as const }
]

const RATE_EPSILON = 1e-6

export function weekendBannerVisible(status: WeekendOfferStatus): boolean {
  return status === 'active'
}

export function sameGroupRate(actual: number, expected: number): boolean {
  return Number.isFinite(actual) && Math.abs(actual - expected) <= RATE_EPSILON
}

export function formatObservedRate(value: number): string {
  if (!Number.isFinite(value)) return ''
  const rounded = Math.round((value + Number.EPSILON) * 1000) / 1000
  return rounded.toFixed(3).replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
}

function nameMatches(name: string, token: string): boolean {
  return name.trim().toLowerCase().includes(token)
}

function sharedDiscount(groups: WeekendGroupQuote[]): number | null {
  const factors = TARGETS.map((target) => groups.find((group) => group.id === target.id)?.token_campaign_discount)
  const first = factors[0]
  if (typeof first !== 'number' || !Number.isFinite(first) || first <= 0 || first > 1) return null
  if (!factors.every((factor) => factor === first)) return null
  return first
}

export function evaluateWeekendOffer(now: number, groups: WeekendGroupQuote[] | null): WeekendOffer {
  const discount = groups ? sharedDiscount(groups) : null
  const lines: WeekendOfferLine[] = TARGETS.map((target) => {
    const live = groups?.find((group) => group.id === target.id)
    const userRate = typeof live?.user_rate_multiplier === 'number' && Number.isFinite(live.user_rate_multiplier)
      ? live.user_rate_multiplier
      : null
    return {
      id: target.id,
      label: target.label,
      fromLabel: target.fromLabel,
      toLabel: target.toLabel,
      saveLabel: target.saveLabel,
      liveRate: live && Number.isFinite(live.rate_multiplier) ? live.rate_multiplier : null,
      userRate,
      effectiveFrom: discount === null ? null : formatObservedRate(target.from * discount),
      effectiveTo: discount === null ? null : formatObservedRate(target.to * discount)
    }
  })
  const endsAt = WEEKEND_RATE_ENDS_AT
  if (!Number.isFinite(now)) return { status: 'unverified', endsAt, lines, campaignDiscount: discount }
  if (now >= Date.parse(endsAt)) return { status: 'ended', endsAt, lines, campaignDiscount: discount }
  if (!groups) return { status: 'unverified', endsAt, lines, campaignDiscount: null }

  const matched = TARGETS.every((target) => {
    const found = groups.filter((group) => group.id === target.id)
    if (found.length !== 1) return false
    const group = found[0]
    if (!nameMatches(group.name, target.token)) return false
    if (group.subscription_type !== undefined && group.subscription_type !== 'standard') return false
    return sameGroupRate(group.rate_multiplier, target.to)
  })
  if (matched) return { status: 'active', endsAt, lines, campaignDiscount: discount }

  const stillPrevious = TARGETS.every((target) => {
    const found = groups.filter((group) => group.id === target.id)
    return found.length === 1 && sameGroupRate(found[0].rate_multiplier, target.from)
  })
  return { status: stillPrevious ? 'not-yet' : 'unexpected', endsAt, lines, campaignDiscount: discount }
}
