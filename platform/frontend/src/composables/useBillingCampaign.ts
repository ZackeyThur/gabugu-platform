import { computed, onMounted, onUnmounted, readonly, ref } from 'vue'
import { getBillingCampaign, type BillingCampaign, type BillingCampaignResponse } from '@/api/billingCampaign'
import { campaignDiscountForGroup, campaignPhaseAt } from '@/utils/billingCampaign'

const ACTIVE_POLL_INTERVAL_MS = 30_000
const IDLE_POLL_INTERVAL_MS = 10 * 60_000
const SHARED_FRESH_WINDOW_MS = 5_000

const campaign = ref<BillingCampaign | null>(null)
const now = ref(Number.NaN)
const error = ref(false)
let anchorServer = Number.NaN
let anchorMonotonic = 0
let lastVerifiedAt = 0
let lastAttemptAt = 0
let subscribers = 0
let ticker: ReturnType<typeof setInterval> | undefined
let inFlight: Promise<void> | undefined
let sharedFetchInFlight: Promise<BillingCampaignResponse> | undefined
let lastSharedResponse: { data: BillingCampaignResponse; fetchedAt: number } | undefined
let lastSharedError: { error: unknown; fetchedAt: number } | undefined

function isPageHidden(): boolean {
  return typeof document !== 'undefined' && document.hidden
}

function currentServerNow(): number {
  return Number.isFinite(anchorServer) ? anchorServer + performance.now() - anchorMonotonic : Number.NaN
}

function hasActiveCountdown(): boolean {
  if (!campaign.value) return false
  const currentPhase = campaignPhaseAt(campaign.value, currentServerNow())
  return currentPhase === 'scheduled' || currentPhase === 'championship-day' || currentPhase === 'celebration-week'
}

function syncClockIfNeeded(): void {
  if (!Number.isFinite(anchorServer)) return
  if (hasActiveCountdown()) {
    tick()
    return
  }
  if (campaign.value && campaignPhaseAt(campaign.value, now.value) !== campaignPhaseAt(campaign.value, currentServerNow())) {
    tick()
  }
}

function pollIntervalMs(): number {
  return (hasActiveCountdown() || error.value) ? ACTIVE_POLL_INTERVAL_MS : IDLE_POLL_INTERVAL_MS
}

function applyResponse(data: BillingCampaignResponse): void {
  const serverTime = Date.parse(data.server_now)
  if (!Number.isFinite(serverTime)) throw new Error('Invalid campaign clock')
  anchorServer = serverTime
  anchorMonotonic = performance.now()
  lastVerifiedAt = Date.now()
  campaign.value = data.campaign
  error.value = data.status === 'invalid'
  tick()
}

/**
 * Shared fetcher for /api/v1/billing-campaign so co-mounted banners
 * (e.g. BillingCampaignBanner + WeekendRateBanner) coalesce into one request.
 */
export async function fetchBillingCampaignShared(force = false): Promise<BillingCampaignResponse> {
  const currentNow = Date.now()
  if (!force && lastSharedResponse && currentNow - lastSharedResponse.fetchedAt < SHARED_FRESH_WINDOW_MS) {
    return lastSharedResponse.data
  }
  if (!force && lastSharedError && currentNow - lastSharedError.fetchedAt < SHARED_FRESH_WINDOW_MS) {
    throw lastSharedError.error
  }
  if (sharedFetchInFlight) return sharedFetchInFlight
  sharedFetchInFlight = getBillingCampaign()
    .then((data) => {
      lastSharedResponse = { data, fetchedAt: Date.now() }
      lastSharedError = undefined
      return data
    })
    .catch((err) => {
      lastSharedError = { error: err, fetchedAt: Date.now() }
      throw err
    })
    .finally(() => {
      sharedFetchInFlight = undefined
    })
  return sharedFetchInFlight
}

function tick() {
  if (Number.isFinite(anchorServer)) now.value = currentServerNow()
}

async function refresh(force = false): Promise<void> {
  if (inFlight) return inFlight
  lastAttemptAt = Date.now()
  inFlight = (async () => {
    try {
      const data = await fetchBillingCampaignShared(force)
      applyResponse(data)
    } catch {
      error.value = true
      // Retain a bounded server-clock estimate during a short network interruption.
      // Stop presenting an offer after five minutes without a verified response.
      if (!lastVerifiedAt || Date.now() - lastVerifiedAt > 300_000) campaign.value = null
    } finally {
      lastAttemptAt = Date.now()
    }
  })().finally(() => { inFlight = undefined })
  return inFlight
}

function onVisibilityChange() {
  if (isPageHidden()) return
  syncClockIfNeeded()
  if (!lastAttemptAt || Date.now() - lastAttemptAt >= pollIntervalMs()) {
    void refresh(false)
  }
}

export function useBillingCampaign() {
  const phase = computed(() => campaignPhaseAt(campaign.value, now.value))
  const discountForGroup = (groupId: number) => campaignDiscountForGroup(campaign.value, phase.value, groupId)
  const nextTransition = computed(() => {
    if (!campaign.value) return null
    if (phase.value === 'scheduled') return campaign.value.starts_at
    if (phase.value === 'championship-day') return campaign.value.day_ends_at
    if (phase.value === 'celebration-week') return campaign.value.ends_at
    return null
  })
  const remainingSeconds = computed(() => nextTransition.value ? Math.max(0, Math.ceil((Date.parse(nextTransition.value) - now.value) / 1000)) : 0)
  const countdown = computed(() => {
    const value = remainingSeconds.value
    const days = Math.floor(value / 86400)
    const hours = Math.floor(value % 86400 / 3600).toString().padStart(2, '0')
    const minutes = Math.floor(value % 3600 / 60).toString().padStart(2, '0')
    const seconds = (value % 60).toString().padStart(2, '0')
    return `${days ? `${days}天 ` : ''}${hours}:${minutes}:${seconds}`
  })
  onMounted(() => {
    subscribers++
    if (subscribers === 1 && typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', onVisibilityChange)
    }
    syncClockIfNeeded()
    if (!lastAttemptAt || Date.now() - lastAttemptAt >= pollIntervalMs()) {
      void refresh(false)
    }
    if (!ticker) ticker = setInterval(() => {
      if (isPageHidden()) return
      syncClockIfNeeded()
      if (!lastAttemptAt || Date.now() - lastAttemptAt >= pollIntervalMs()) void refresh(false)
    }, 1000)
  })
  onUnmounted(() => {
    subscribers--
    if (subscribers === 0) {
      if (ticker) { clearInterval(ticker); ticker = undefined }
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', onVisibilityChange)
      }
    }
  })
  return { campaign: readonly(campaign), now: readonly(now), error: readonly(error), phase, countdown, remainingSeconds, nextTransition, discountForGroup, refresh }
}
