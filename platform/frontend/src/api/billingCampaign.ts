import { apiClient } from './client'

export interface BillingCampaignGroup {
  id: number
  name: string
  day_discount: number
  week_discount: number
}

export interface BillingCampaign {
  id: string
  title: string
  pricing_basis: 'site_standard'
  starts_at: string
  day_ends_at: string
  ends_at: string
  groups: BillingCampaignGroup[]
}

export interface BillingCampaignResponse {
  campaign: BillingCampaign | null
  server_now: string
  status: 'disabled' | 'invalid' | 'scheduled' | 'championship-day' | 'celebration-week' | 'ended'
}

export async function getBillingCampaign(): Promise<BillingCampaignResponse> {
  const { data } = await apiClient.get<BillingCampaignResponse>('/billing-campaign')
  return data
}
