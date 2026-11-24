<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { computed, defineAsyncComponent, onMounted, onBeforeUnmount, watch } from 'vue'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import { resolveRouteDocumentTitle } from '@/router/title'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore, useSurveyStore, useAdminComplianceStore, useAdminSettingsStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'
import { updateFavicon } from '@/utils/branding'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { useMotionEffects } from '@/composables/useMotionEffects'

useMotionEffects()

const AdminComplianceDialog = defineAsyncComponent(() => import('@/components/admin/AdminComplianceDialog.vue'))
const AnnouncementPopup = defineAsyncComponent(() => import('@/components/common/AnnouncementPopup.vue'))
const SurveyPopup = defineAsyncComponent(() => import('@/components/common/SurveyPopup.vue'))

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const announcementStore = useAnnouncementStore()
const surveyStore = useSurveyStore()
const adminComplianceStore = useAdminComplianceStore()
const adminSettingsStore = useAdminSettingsStore()

function updateDocumentTitle() {
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems, {
    billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings),
  })
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

watch(
  [
    () => route.fullPath,
    () => route.meta.title,
    () => route.meta.titleKey,
    () => appStore.siteName,
    () => appStore.cachedPublicSettings?.custom_menu_items,
    () => appStore.cachedPublicSettings?.subscription_enabled,
    () => appStore.cachedPublicSettings?.payment_balance_disabled,
    () => authStore.isAdmin,
    () => adminSettingsStore.customMenuItems,
  ],
  updateDocumentTitle,
  { deep: true }
)

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
    surveyStore.fetchActiveSurveys()
  }
}

function onAdminComplianceRequired(event: Event) {
  const detail = (event as CustomEvent<Record<string, string>>).detail || {}
  adminComplianceStore.requireAcknowledgement(detail)
}

// 订阅功能开关（opt-out）。关闭后不再预加载/轮询订阅接口；开关在登录后才到达时补启动，反向则清空。
const subscriptionFeatureEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription))

function startSubscriptionSync() {
  subscriptionStore.fetchActiveSubscriptions().catch((error) => {
    console.error('Failed to preload subscriptions:', error)
  })
  subscriptionStore.startPolling()
}

watch(subscriptionFeatureEnabled, (enabled) => {
  if (!authStore.isAuthenticated) return
  if (enabled) {
    startSubscriptionSync()
  } else {
    subscriptionStore.clear()
  }
})

let loginPromptTimer: ReturnType<typeof setTimeout> | null = null

function clearLoginPromptTimer() {
  if (loginPromptTimer !== null) {
    clearTimeout(loginPromptTimer)
    loginPromptTimer = null
  }
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    clearLoginPromptTimer()
    if (isAuthenticated) {
      if (authStore.isAdmin) {
        adminComplianceStore.fetchStatus().catch((error) => {
          console.error('Failed to fetch admin compliance status:', error)
        })
      }

      // User logged in: preload subscriptions and start polling (skipped when the
      // subscription feature is switched off; see the flag watcher below)
      if (subscriptionFeatureEnabled.value) {
        startSubscriptionSync()
      }

      // Announcements & Surveys: new login vs page refresh restore
      if (oldValue === false) {
        // New login: delay 3s then force fetch
        loginPromptTimer = setTimeout(() => {
          loginPromptTimer = null
          if (!authStore.isAuthenticated) return
          announcementStore.fetchAnnouncements(true)
          surveyStore.fetchActiveSurveys(true)
        }, 3000)
      } else {
        // Page refresh restore (oldValue was undefined)
        announcementStore.fetchAnnouncements()
        surveyStore.fetchActiveSurveys()
      }

      // Register visibility change listener
      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      // User logged out: clear data and stop polling
      subscriptionStore.clear()
      announcementStore.reset()
      surveyStore.reset()
      adminComplianceStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  },
  { immediate: true }
)

// Route change trigger (throttled by store)
router.afterEach(() => {
  if (authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  clearLoginPromptTimer()
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('admin-compliance-required', onAdminComplianceRequired)
})

onMounted(async () => {
  window.addEventListener('admin-compliance-required', onAdminComplianceRequired)

  // Skip the setup probe when backend SSR config is already injected on normal routes;
  // otherwise run setup check and public settings fetch in parallel instead of serially.
  const shouldCheckSetup = route.path === '/setup' || typeof window === 'undefined' || !window.__APP_CONFIG__
  const setupPromise = shouldCheckSetup
    ? getSetupStatus().catch(() => null)
    : Promise.resolve(null)

  const [status] = await Promise.all([
    setupPromise,
    appStore.fetchPublicSettings(),
  ])

  if (status?.needs_setup && route.path !== '/setup') {
    router.replace('/setup')
    return
  }

  // Re-resolve document title now that site settings are available
  updateDocumentTitle()
})
</script>

<template>
  <NavigationProgress />
  <RouterView />
  <Toast />
  <AnnouncementPopup v-if="authStore.isAuthenticated" />
  <SurveyPopup v-if="authStore.isAuthenticated" />
  <AdminComplianceDialog v-if="authStore.isAuthenticated && authStore.isAdmin" />
</template>
