<template>
  <header
    class="glass sticky top-0 z-30 border-b border-gray-200/70 dark:border-dark-700/60"
  >
    <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-3 sm:px-6">
      <!-- 左:站点 logo + 名称（可返回首页） -->
      <RouterLink
        to="/home"
        class="flex min-w-0 items-center gap-3 transition-opacity hover:opacity-85"
      >
        <template v-if="settings">
          <span
            class="flex h-9 w-9 flex-shrink-0 items-center justify-center"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <GabuguWordmark :name="siteName" class="text-lg" />
        </template>
        <template v-else>
          <span class="h-9 w-9 flex-shrink-0 animate-pulse rounded-lg bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
          <span class="h-5 w-28 animate-pulse rounded bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
        </template>
      </RouterLink>

      <!-- 右:模型榜单 + 语言 + 主题 + 登录 / 回到后台 -->
      <div class="flex shrink-0 items-center gap-1.5 sm:gap-2.5">
        <RouterLink
          to="/arena"
          class="btn btn-ghost btn-sm flex-shrink-0"
        >
          {{ t('arena.nav.rankings') }}
        </RouterLink>
        <LocaleSwitcher />
        <button
          type="button"
          class="btn-ghost btn-icon"
          :title="isDark ? t('nav.lightMode') : t('nav.darkMode')"
          :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
          @click="toggleTheme"
        >
          <Icon v-if="isDark" name="sun" size="sm" class="text-amber-400" />
          <Icon v-else name="moon" size="sm" />
        </button>

        <RouterLink
          v-if="isAuthenticated"
          :to="backTarget"
          data-magnet
          class="btn btn-primary btn-md flex-shrink-0"
        >
          {{ t('modelPlaza.nav.backToDashboard') }}
        </RouterLink>
        <RouterLink
          v-else
          :to="{ path: '/login', query: { redirect: '/model-plaza' } }"
          data-magnet
          class="btn btn-primary btn-md flex-shrink-0"
        >
          {{ t('modelPlaza.nav.login') }}
        </RouterLink>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { sanitizeUrl } from '@/utils/url'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const isDark = ref(typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  try {
    localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
  } catch {
    // ignore storage errors
  }
}

const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || 'Gabugu')
const siteLogo = computed(() =>
  sanitizeUrl(settings.value?.site_logo || '', { allowRelative: true, allowDataUrl: true })
)
const isAuthenticated = computed(() => authStore.isAuthenticated)
const backTarget = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
</script>
