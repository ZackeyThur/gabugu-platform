<template>
  <header class="arena-nav" :data-theme="theme" :class="{ 'is-scrolled': scrolled }">
    <div class="arena-nav-inner">
      <div class="arena-nav-left">
        <RouterLink to="/home" class="arena-nav-brand" :aria-label="siteName">
          <img :src="siteLogo || (theme === 'dark' ? '/brand/gabugu-logo-dark.png' : '/brand/gabugu-logo-light.png')" alt="" />
          <GabuguWordmark :name="siteName" :theme="theme" />
        </RouterLink>
        <nav class="arena-nav-links" aria-label="Public navigation">
          <RouterLink to="/arena" class="is-active">{{ t('arena.nav.rankings') }}</RouterLink>
          <RouterLink v-if="showModelPlaza" to="/model-plaza">{{ t('arena.nav.modelPlaza') }}</RouterLink>
        </nav>
      </div>
      <div class="arena-nav-right">
        <LocaleSwitcher />
        <button
          type="button"
          class="arena-nav-theme"
          data-testid="arena-theme-toggle"
          :aria-label="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
          :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
          @click="emit('toggle-theme')"
        >
          <Transition name="theme-icon" mode="out-in">
            <svg v-if="theme === 'dark'" key="sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true">
              <circle cx="12" cy="12" r="4" />
              <path stroke-linecap="round" d="M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M5.3 18.7l1.4-1.4M17.3 6.7l1.4-1.4" />
            </svg>
            <svg v-else key="moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20.5 14.5A8.5 8.5 0 019.5 3.5a8.5 8.5 0 1011 11z" />
            </svg>
          </Transition>
        </button>
        <RouterLink v-if="isAuthenticated" :to="backTarget" class="arena-nav-cta">{{ t('arena.nav.backToDashboard') }}</RouterLink>
        <RouterLink v-else :to="{ path: '/login', query: { redirect: '/arena' } }" class="arena-nav-cta">{{ t('arena.nav.login') }}</RouterLink>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

withDefaults(defineProps<{ theme?: 'dark' | 'light' }>(), { theme: 'dark' })
const emit = defineEmits<{ (e: 'toggle-theme'): void }>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const scrolled = ref(false)
function onScroll() {
  scrolled.value = typeof window !== 'undefined' && window.scrollY > 12
}

onMounted(() => {
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})

const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || 'GABUGU')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || '', { allowRelative: true, allowDataUrl: true }))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const backTarget = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const showModelPlaza = computed(
  () =>
    resolveFeatureFlag(settings.value, FeatureFlags.modelPlaza) &&
    (isAuthenticated.value || settings.value?.model_plaza_require_auth !== true)
)
</script>

<style scoped>
.arena-nav {
  --n-bg: rgba(7, 8, 13, 0.52);
  --n-bg-scrolled: rgba(7, 8, 13, 0.84);
  --n-line: rgba(255, 255, 255, 0.06);
  --n-line-scrolled: rgba(255, 255, 255, 0.11);
  --n-text: #f2f4f8;
  --n-muted: #8b92a1;
  --n-hover: rgba(255, 255, 255, 0.07);
  --n-cta-bg: #f2f4f8;
  --n-cta-text: #08090f;
  --n-shadow: 0 10px 30px rgba(0, 0, 0, 0.34);
  position: sticky;
  top: 0;
  z-index: 30;
  border-bottom: 1px solid var(--n-line);
  background: var(--n-bg);
  backdrop-filter: saturate(170%) blur(18px);
  -webkit-backdrop-filter: saturate(170%) blur(18px);
  transition: background-color 0.24s ease, border-color 0.24s ease, box-shadow 0.24s ease;
}
.arena-nav.is-scrolled {
  background: var(--n-bg-scrolled);
  border-bottom-color: var(--n-line-scrolled);
  box-shadow: var(--n-shadow);
}
.arena-nav[data-theme='light'] {
  --n-bg: rgba(250, 251, 253, 0.68);
  --n-bg-scrolled: rgba(255, 255, 255, 0.88);
  --n-line: rgba(15, 23, 42, 0.06);
  --n-line-scrolled: rgba(15, 23, 42, 0.1);
  --n-text: #0f172a;
  --n-muted: #64748b;
  --n-hover: rgba(15, 23, 42, 0.055);
  --n-cta-bg: #0f172a;
  --n-cta-text: #ffffff;
  --n-shadow: 0 8px 24px rgba(15, 23, 42, 0.06);
}
.arena-nav-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  max-width: 1184px;
  height: 64px;
  margin: 0 auto;
  padding: 0 32px;
}
.arena-nav-left {
  display: flex;
  align-items: center;
  gap: 40px;
  min-width: 0;
}
.arena-nav-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--n-text);
  text-decoration: none;
  transition: opacity 0.18s ease;
}
.arena-nav-brand:hover {
  opacity: 0.88;
}
.arena-nav-brand img {
  width: 24px;
  height: 27px;
  object-fit: contain;
}
.arena-nav-brand :deep(.gabugu-wordmark) {
  font-size: 18px;
}
.arena-nav-links {
  display: flex;
  align-items: center;
  gap: 6px;
}
.arena-nav-links a {
  position: relative;
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  color: var(--n-muted);
  font-size: 14px;
  font-weight: 500;
  text-decoration: none;
  transition: color 0.18s ease, background-color 0.18s ease;
}
.arena-nav-links a:hover {
  color: var(--n-text);
  background: var(--n-hover);
}
.arena-nav-links a.is-active {
  color: var(--n-text);
  background: var(--n-hover);
}

.arena-nav-right {
  display: flex;
  align-items: center;
  gap: 6px;
}
.arena-nav-right :deep(.text-base) {
  display: none;
}
.arena-nav-right :deep(.hidden.sm\:inline) {
  display: inline;
}
.arena-nav[data-theme='dark'] .arena-nav-right :deep(button) {
  color: #c5c9d3;
}
.arena-nav[data-theme='dark'] .arena-nav-right :deep(button:hover) {
  background: rgba(255, 255, 255, 0.07);
}
.arena-nav-theme {
  display: inline-flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: none;
  color: var(--n-muted) !important;
  cursor: pointer;
  transition: background-color 0.18s ease, color 0.18s ease, transform 0.14s ease;
}
.arena-nav-theme:hover {
  background: var(--n-hover) !important;
  color: var(--n-text) !important;
}
.arena-nav-theme:active {
  transform: scale(0.92);
}
.arena-nav-theme svg {
  width: 18px;
  height: 18px;
}

.theme-icon-enter-active,
.theme-icon-leave-active {
  transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.18s ease;
}
.theme-icon-enter-from {
  opacity: 0;
  transform: rotate(-45deg) scale(0.75);
}
.theme-icon-leave-to {
  opacity: 0;
  transform: rotate(45deg) scale(0.75);
}

.arena-nav-cta {
  display: inline-flex;
  height: 34px;
  align-items: center;
  margin-left: 4px;
  padding: 0 16px;
  border-radius: 999px;
  background: var(--n-cta-bg);
  color: var(--n-cta-text);
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: opacity 0.16s ease, transform 0.14s ease;
}
.arena-nav-cta:hover {
  opacity: 0.88;
}
.arena-nav-cta:active {
  transform: scale(0.97);
}

@media (max-width: 640px) {
  .arena-nav-inner {
    height: 56px;
    padding: 0 20px;
  }
  .arena-nav-links {
    display: none;
  }
  .arena-nav-brand :deep(.gabugu-wordmark) {
    font-size: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .arena-nav,
  .theme-icon-enter-active,
  .theme-icon-leave-active {
    transition: none;
  }
}
</style>
