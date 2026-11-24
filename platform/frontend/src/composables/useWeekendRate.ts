import { computed, onMounted, onUnmounted, readonly, ref } from 'vue'
import { fetchBillingCampaignShared } from '@/composables/useBillingCampaign'
import { getModelPlaza, type ModelPlazaGroup } from '@/api/modelPlaza'
import {
  WEEKEND_RATE_ENDS_AT,
  evaluateWeekendOffer,
  weekendBannerVisible,
  type WeekendGroupQuote,
  type WeekendOffer
} from '@/utils/weekendRate'

const WEEKEND_POLL_INTERVAL_MS = 30_000
const WEEKEND_ENDS_AT_MS = Date.parse(WEEKEND_RATE_ENDS_AT)

const offer = ref<WeekendOffer>(evaluateWeekendOffer(Number.NaN, null))
let anchorServer = Number.NaN
let anchorMonotonic = 0
let lastAttemptAt = 0
let groups: WeekendGroupQuote[] | null = null
let subscribers = 0
let ticker: ReturnType<typeof setInterval> | undefined
let inFlight: Promise<void> | undefined

function isPageHidden(): boolean {
  return typeof document !== 'undefined' && document.hidden
}

function currentServerNow(): number {
  return Number.isFinite(anchorServer) ? anchorServer + performance.now() - anchorMonotonic : Number.NaN
}

function isConfirmedEnded(): boolean {
  const current = currentServerNow()
  return Number.isFinite(current) && current >= WEEKEND_ENDS_AT_MS && offer.value.status === 'ended'
}

function stopTicker(): void {
  if (ticker) {
    clearInterval(ticker)
    ticker = undefined
  }
}

function hasSession(): boolean {
  try {
    return !!localStorage.getItem('auth_token')
  } catch {
    return false
  }
}

function quote(group: ModelPlazaGroup): WeekendGroupQuote {
  return {
    id: group.id,
    name: group.name,
    rate_multiplier: group.rate_multiplier,
    subscription_type: group.subscription_type,
    token_campaign_discount: group.token_campaign_discount,
    user_rate_multiplier: group.user_rate_multiplier
  }
}

function publish() {
  const now = currentServerNow()
  offer.value = evaluateWeekendOffer(now, groups)
  if (isConfirmedEnded()) {
    stopTicker()
  }
}

async function refresh(force = false): Promise<void> {
  if (!force && isConfirmedEnded()) return
  if (inFlight) return inFlight
  lastAttemptAt = Date.now()
  inFlight = (async () => {
    try {
      const campaign = await fetchBillingCampaignShared(force)
      const serverTime = Date.parse(campaign.server_now)
      if (!Number.isFinite(serverTime)) throw new Error('Invalid campaign clock')
      anchorServer = serverTime
      anchorMonotonic = performance.now()
    } catch {
      anchorServer = Number.NaN
      groups = null
      publish()
      return
    } finally {
      lastAttemptAt = Date.now()
    }
    const now = anchorServer
    if (hasSession() && now < WEEKEND_ENDS_AT_MS) {
      try {
        const plaza = await getModelPlaza()
        groups = plaza.groups.map(quote)
      } catch {
        groups = null
      }
    } else {
      groups = null
    }
    publish()
  })().finally(() => { inFlight = undefined })
  return inFlight
}

const tick = ref(0)

function onVisibilityChange() {
  if (isPageHidden() || isConfirmedEnded()) return
  tick.value++
  publish()
  if (!lastAttemptAt || Date.now() - lastAttemptAt >= WEEKEND_POLL_INTERVAL_MS) {
    void refresh(false)
  }
}

export function useWeekendRate() {
  const countdown = computed(() => {
    void tick.value
    const current = currentServerNow()
    if (!Number.isFinite(current) || offer.value.status === 'ended') return ''
    const seconds = Math.max(0, Math.ceil((WEEKEND_ENDS_AT_MS - current) / 1000))
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor(seconds % 86400 / 3600).toString().padStart(2, '0')
    const minutes = Math.floor(seconds % 3600 / 60).toString().padStart(2, '0')
    const rest = (seconds % 60).toString().padStart(2, '0')
    return `${days ? `${days}天 ` : ''}${hours}:${minutes}:${rest}`
  })
  onMounted(() => {
    subscribers++
    if (subscribers === 1 && typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', onVisibilityChange)
    }
    if (!isConfirmedEnded()) {
      if (!lastAttemptAt || Date.now() - lastAttemptAt >= WEEKEND_POLL_INTERVAL_MS) {
        void refresh(false)
      }
      if (!ticker) ticker = setInterval(() => {
        if (isPageHidden()) return
        if (isConfirmedEnded()) {
          stopTicker()
          return
        }
        tick.value++
        publish()
        if (!lastAttemptAt || Date.now() - lastAttemptAt >= WEEKEND_POLL_INTERVAL_MS) {
          void refresh(false)
        }
      }, 1000)
    }
  })
  onUnmounted(() => {
    subscribers--
    if (subscribers === 0) {
      stopTicker()
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', onVisibilityChange)
      }
    }
  })
  return {
    offer: readonly(offer),
    countdown,
    bannerVisible: computed(() => weekendBannerVisible(offer.value.status)),
    refresh
  }
}
