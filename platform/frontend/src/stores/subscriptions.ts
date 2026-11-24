/**
 * Subscription Store
 * Global state management for user subscriptions with caching and deduplication
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'

// Cache TTL: 60 seconds
const CACHE_TTL_MS = 60_000

// Request generation counter to invalidate stale in-flight responses
let requestGeneration = 0
let activeSubscriptionVisibilityListener: (() => void) | null = null

export const useSubscriptionStore = defineStore('subscriptions', () => {
  // State
  const activeSubscriptions = ref<UserSubscription[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const lastFetchedAt = ref<number | null>(null)

  // In-flight request deduplication
  let activePromise: Promise<UserSubscription[]> | null = null

  // Auto-refresh interval
  let pollerInterval: ReturnType<typeof setInterval> | null = null

  // Computed
  const hasActiveSubscriptions = computed(() => activeSubscriptions.value.length > 0)

  /**
   * Fetch active subscriptions with caching and deduplication
   * @param force - Force refresh even if cache is valid
   */
  async function fetchActiveSubscriptions(force = false): Promise<UserSubscription[]> {
    const now = Date.now()

    // Return cached data if valid
    if (
      !force &&
      loaded.value &&
      lastFetchedAt.value &&
      now - lastFetchedAt.value < CACHE_TTL_MS
    ) {
      return activeSubscriptions.value
    }

    // Return in-flight request if exists (deduplication)
    if (activePromise && !force) {
      return activePromise
    }

    const currentGeneration = ++requestGeneration

    // Start new request
    loading.value = true
    const requestPromise = subscriptionsAPI
      .getActiveSubscriptions()
      .then((data) => {
        if (currentGeneration === requestGeneration) {
          activeSubscriptions.value = data
          loaded.value = true
          lastFetchedAt.value = Date.now()
        }
        return data
      })
      .catch((error) => {
        console.error('Failed to fetch active subscriptions:', error)
        throw error
      })
      .finally(() => {
        if (activePromise === requestPromise) {
          loading.value = false
          activePromise = null
        }
      })

    activePromise = requestPromise

    return activePromise
  }

  const POLL_INTERVAL_MS = 5 * 60 * 1000
  let visibilityListenerBound = false

  function onVisibilityChange() {
    if (typeof document === 'undefined' || document.hidden) return
    if (lastFetchedAt.value === null || Date.now() - lastFetchedAt.value >= POLL_INTERVAL_MS) {
      fetchActiveSubscriptions(true).catch((error) => {
        console.error('Subscription visibility-resume refresh failed:', error)
      })
    }
  }

  /**
   * Start auto-refresh polling 
   */
  function startPolling() {
    if (pollerInterval) return

    if (typeof document !== 'undefined') {
      if (activeSubscriptionVisibilityListener) {
        document.removeEventListener('visibilitychange', activeSubscriptionVisibilityListener)
      }
      document.addEventListener('visibilitychange', onVisibilityChange)
      activeSubscriptionVisibilityListener = onVisibilityChange
      visibilityListenerBound = true
    }

    pollerInterval = setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) return
      fetchActiveSubscriptions(true).catch((error) => {
        console.error('Subscription polling failed:', error)
      })
    }, POLL_INTERVAL_MS)
  }

  /**
   * Stop auto-refresh polling
   */
  function stopPolling() {
    if (pollerInterval) {
      clearInterval(pollerInterval)
      pollerInterval = null
    }
    if (visibilityListenerBound && typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisibilityChange)
      if (activeSubscriptionVisibilityListener === onVisibilityChange) {
        activeSubscriptionVisibilityListener = null
      }
      visibilityListenerBound = false
    }
  }

  /**
   * Clear all subscription data and stop polling
   */
  function clear() {
    requestGeneration++
    activePromise = null
    loading.value = false
    activeSubscriptions.value = []
    loaded.value = false
    lastFetchedAt.value = null
    stopPolling()
  }

  /**
   * Invalidate cache (force next fetch to reload)
   */
  function invalidateCache() {
    lastFetchedAt.value = null
  }

  return {
    // State
    activeSubscriptions,
    loading,
    hasActiveSubscriptions,

    // Actions
    fetchActiveSubscriptions,
    startPolling,
    stopPolling,
    clear,
    invalidateCache
  }
})
