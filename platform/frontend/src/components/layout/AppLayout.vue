<template>
  <div class="g-shell min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-[204px]']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="p-4 md:p-6 lg:p-8">
        <BillingCampaignBanner />
        <WeekendRateBanner />
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import BillingCampaignBanner from '@/components/common/BillingCampaignBanner.vue'
import WeekendRateBanner from '@/components/common/WeekendRateBanner.vue'

const appStore = useAppStore()
const props = withDefaults(defineProps<{ autoStartTour?: boolean }>(), { autoStartTour: true })
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: props.autoStartTour
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
