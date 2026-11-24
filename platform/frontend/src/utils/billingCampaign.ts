import type { BillingCampaign } from '@/api/billingCampaign'

export type CampaignPhase = 'unavailable' | 'scheduled' | 'championship-day' | 'celebration-week' | 'ended'

export function campaignPhaseAt(campaign: BillingCampaign | null, now: number): CampaignPhase {
  if (!campaign || !Number.isFinite(now)) return 'unavailable'
  const start = Date.parse(campaign.starts_at)
  const dayEnd = Date.parse(campaign.day_ends_at)
  const end = Date.parse(campaign.ends_at)
  if (![start, dayEnd, end].every(Number.isFinite) || !(start < dayEnd && dayEnd < end)) return 'unavailable'
  if (now < start) return 'scheduled'
  if (now < dayEnd) return 'championship-day'
  if (now < end) return 'celebration-week'
  return 'ended'
}

export function campaignDiscountForGroup(campaign: BillingCampaign | null, phase: CampaignPhase, groupId: number): number {
  const group = campaign?.groups.find((item) => item.id === groupId)
  if (!group) return 1
  const discount = phase === 'championship-day' ? group.day_discount : phase === 'celebration-week' ? group.week_discount : 1
  return Number.isFinite(discount) && discount > 0 && discount <= 1 ? discount : 1
}

export function formatCampaignTime(value: string | undefined): string {
  if (!value || !Number.isFinite(Date.parse(value))) return '时间待确认'
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', hourCycle: 'h23'
  }).format(new Date(value))
}
