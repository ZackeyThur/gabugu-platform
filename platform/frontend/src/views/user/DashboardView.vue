<template>
  <AppLayout>
    <div ref="overviewRootRef" class="dashboard-overview">
      <header class="overview-header">
        <div>
          <h1 class="g-blur-in">{{ t('dashboard.overview.title') }}</h1>
          <p class="overview-header-meta">
            <span>{{ t('dashboard.overview.subtitle') }}</span>
            <span class="overview-meta-time">
              <span class="overview-meta-dot" aria-hidden="true">·</span>
              <span>UTC+8</span>
              <template v-if="lastUpdatedLabel">
                <span class="overview-meta-dot" aria-hidden="true">·</span>
                <span class="overview-sync-readout">
                  <span class="overview-sync-dot" aria-hidden="true" />
                  {{ t('dashboard.overview.lastUpdatedAt', { time: lastUpdatedLabel }) }}
                </span>
              </template>
            </span>
          </p>
        </div>
        <div class="overview-header-actions">
          <button
            type="button"
            data-magnet
            data-spark
            class="overview-secondary-button"
            :class="{ 'is-refreshing': busy }"
            :disabled="busy"
            @click="handleRefresh"
          >
            <Icon name="refresh" size="sm" class="overview-refresh-icon" />
            {{ t('common.refresh') }}
          </button>
          <router-link to="/keys" data-magnet class="overview-secondary-button">
            <Icon name="key" size="sm" />
            <span>{{ t('dashboard.overview.manageKeys') }}</span>
            <span v-if="activeKeysCount !== null" class="overview-keys-badge">
              {{ activeKeysCount }}
            </span>
          </router-link>
        </div>
      </header>

      <UserDashboardStats
        :stats="stats.data.value"
        :loading="stats.loading.value"
        :error="stats.error.value"
        :balance="balance.data.value?.balance ?? null"
        :frozen-balance="balance.data.value?.frozen_balance ?? 0"
        :balance-loading="balance.loading.value"
        :balance-error="balance.error.value"
        :platform-quotas="quotas.data.value"
        :quotas-loading="quotas.loading.value"
        :quotas-error="quotas.error.value"
        :is-simple="auth.isSimpleMode"
        :payment-enabled="paymentEnabled"
        :subscription-enabled="subscriptionEnabled"
        :billing-mode="billingMode"
        :checkout-info="checkoutInfo"
        :affiliate-detail="affiliateDetail"
        :active-subscriptions="subscriptionStore.activeSubscriptions"
        :baseline-7d="baseline7d"
        :today="today"
        :balance-low-notify-enabled="balanceLowNotifyEnabled"
        :user-balance-notify-enabled="Boolean(balance.data.value?.balance_notify_enabled)"
        :hide-platforms="isZeroUsageNewUser"
        :affiliate-transferring="affiliateTransferring"
        @retry-stats="loadStats"
        @retry-balance="loadBalance"
        @retry-quotas="loadQuotas"
        @transfer-affiliate="handleTransferAffiliate"
        @drill-day="handleDrillDay"
      >
        <template #savings>
          <div class="overview-value-showcase" :class="{ 'has-poster': hasDashboardPosters }">
            <UserDashboardSavings
              :stats="stats.data.value"
              :loading="stats.loading.value"
              :error="stats.error.value"
              :subscription-enabled="subscriptionEnabled"
              :compact="hasDashboardPosters"
              @retry="loadStats"
            />
            <DashboardPosterShowcase
              v-if="hasDashboardPosters"
              :posters="announcementStore.dashboardPosters"
              @open-announcement="openPosterAnnouncement"
            />
          </div>
        </template>
      </UserDashboardStats>

      <!-- Zero-Usage New User 3-Step Onboarding Card -->
      <DashboardOnboardingCard
        v-if="isZeroUsageNewUser"
        :balance="balance.data.value?.balance ?? null"
        :has-active-subscription="subscriptionStore.hasActiveSubscriptions"
        :total-api-keys="stats.data.value?.total_api_keys ?? 0"
        :payment-enabled="paymentEnabled"
        :is-simple="auth.isSimpleMode"
        :checkout-info="checkoutInfo"
      />

      <!-- Full Analytics Workbench for Active Users -->
      <template v-else>
        <UserDashboardCharts
          :start-date="startDate"
          :end-date="endDate"
          :today="today"
          :granularity="granularity"
          :trend="trend.data.value"
          :models="models.data.value"
          :groups="groups.data.value"
          :api-keys="apiKeys.data.value"
          :trend-loading="trend.loading.value"
          :models-loading="models.loading.value"
          :trend-error="trend.error.value"
          :models-error="models.error.value"
          :is-simple="auth.isSimpleMode"
          :api-key-id="apiKeyId"
          :group-id="groupId"
          :filter-api-keys="filterApiKeys"
          :filter-groups="filterGroups"
          @range-change="setRange"
          @filter-change="setFilters"
          @retry-trend="loadTrend"
          @retry-models="loadModels"
        />

        <!-- Restrained Cross-Product Light Exploration Strip -->
        <nav
          v-if="exploreItems.length > 0"
          class="overview-explore-strip"
          :aria-label="t('dashboard.overview.exploreTitle')"
        >
          <span class="overview-explore-label">{{ t('dashboard.overview.exploreTitle') }}</span>
          <div class="overview-explore-pills">
            <router-link
              v-for="item in exploreItems"
              :key="item.to"
              :to="item.to"
              class="overview-explore-pill"
            >
              <Icon :name="item.icon" size="sm" />
              <span>{{ item.label }}</span>
              <Icon name="chevronRight" size="xs" class="overview-explore-arrow" />
            </router-link>
          </div>
        </nav>
      </template>

      <AnnouncementPopup
        :announcement="previewAnnouncement"
        preview
        @close="previewAnnouncement = null"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import gsap from 'gsap'
import { prefersReducedMotion } from '@/utils/chartPlugins'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { userAPI } from '@/api/user'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardSavings from '@/components/user/dashboard/UserDashboardSavings.vue'
import DashboardPosterShowcase from '@/components/user/dashboard/DashboardPosterShowcase.vue'
import DashboardOnboardingCard from '@/components/user/dashboard/DashboardOnboardingCard.vue'
import { useDashboardOverview } from '@/composables/useDashboardOverview'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { DASHBOARD_TIMEZONE } from '@/utils/dashboardOverview'
import type { UserAnnouncement } from '@/types'
import '@/styles/dashboard-overview.css'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const announcementStore = useAnnouncementStore()
const subscriptionStore = useSubscriptionStore()

const billingMode = computed(() => resolveSiteBillingMode(app.cachedPublicSettings))
const paymentEnabled = computed(
  () =>
    resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment) &&
    app.cachedPublicSettings?.payment_balance_disabled !== true
)
const subscriptionEnabled = computed(() =>
  resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription)
)
const balanceLowNotifyEnabled = computed(
  () => app.cachedPublicSettings?.balance_low_notify_enabled === true
)

const {
  today,
  startDate,
  endDate,
  granularity,
  apiKeyId,
  groupId,
  lastUpdatedAt,
  balance,
  stats,
  quotas,
  trend,
  models,
  groups,
  apiKeys,
  baseline7d,
  checkoutInfo,
  affiliateDetail,
  filterApiKeys,
  filterGroups,
  busy,
  refresh,
  setRange,
  setFilters,
  loadBalance,
  loadStats,
  loadQuotas,
  loadTrend,
  loadModels,
  reloadAffiliate,
} = useDashboardOverview()

const previewAnnouncement = ref<UserAnnouncement | null>(null)

const hasDashboardPosters = computed(
  () => !auth.isSimpleMode && announcementStore.dashboardPosters.length > 0
)

const isZeroUsageNewUser = computed(
  () =>
    !stats.loading.value &&
    !stats.error.value &&
    stats.data.value !== null &&
    stats.data.value.total_requests === 0
)

const activeKeysCount = computed(() =>
  !stats.loading.value && !stats.error.value && stats.data.value
    ? stats.data.value.active_api_keys
    : null
)

const lastUpdatedLabel = computed(() => {
  if (!lastUpdatedAt.value) return ''
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: DASHBOARD_TIMEZONE,
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(lastUpdatedAt.value)
})

const exploreItems = computed<Array<{ to: string; icon: 'grid' | 'photo' | 'signal'; label: string }>>(() => {
  const items: Array<{ to: string; icon: 'grid' | 'photo' | 'signal'; label: string }> = []
  if (!auth.isSimpleMode && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza)) {
    items.push({ to: '/model-plaza', icon: 'grid', label: t('dashboard.overview.exploreModelPlaza') })
  }
  if (!auth.isSimpleMode) {
    items.push({ to: '/studio', icon: 'photo', label: t('dashboard.overview.exploreStudio') })
  }
  if (resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.channelMonitor)) {
    items.push({ to: '/monitor', icon: 'signal', label: t('dashboard.overview.exploreMonitor') })
  }
  return items.slice(0, 3)
})

function openPosterAnnouncement(announcement: UserAnnouncement) {
  previewAnnouncement.value = announcement
  if (!announcement.read_at) {
    void announcementStore.markAsRead(announcement.id)
  }
}

function handleRefresh() {
  refresh()
  if (!auth.isSimpleMode && subscriptionEnabled.value) {
    void subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function handleDrillDay(date: string) {
  if (!date) return
  void router.push({ path: '/usage', query: { start_date: date, end_date: date } })
}

const affiliateTransferring = ref(false)

async function handleTransferAffiliate() {
  if (affiliateTransferring.value) return
  affiliateTransferring.value = true
  try {
    await userAPI.transferAffiliateQuota()
    app.showSuccess(t('common.success'))
    await Promise.allSettled([loadBalance(), reloadAffiliate()])
  } catch {
    app.showError(t('common.error'))
  } finally {
    affiliateTransferring.value = false
  }
}

const overviewRootRef = ref<HTMLElement | null>(null)
let motionCtx: gsap.Context | null = null

onMounted(() => {
  if (!auth.isSimpleMode) {
    void announcementStore.fetchAnnouncements()
    if (subscriptionEnabled.value) {
      void subscriptionStore.fetchActiveSubscriptions().catch(() => {})
    }
  }
  if (import.meta.env.MODE !== 'test' && !prefersReducedMotion() && overviewRootRef.value) {
    void nextTick(() => {
      if (!overviewRootRef.value) return
      motionCtx = gsap.context(() => {
        const targets = overviewRootRef.value?.querySelectorAll(
          '.overview-command-monolith, .overview-savings, .overview-poster-card, .overview-sub-strip, .overview-plan-nudge, .overview-platform-ribbon, .overview-analysis-studio'
        )
        if (!targets || targets.length === 0) return
        gsap.from(targets, {
          y: 14,
          opacity: 0,
          duration: 0.46,
          stagger: 0.04,
          ease: 'power3.out',
          clearProps: 'transform,opacity',
        })
      }, overviewRootRef.value)
    })
  }
})

onBeforeUnmount(() => {
  motionCtx?.revert()
  motionCtx = null
})
</script>
