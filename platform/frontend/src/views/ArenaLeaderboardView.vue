<template>
  <!-- 后台内嵌形态：?embedded=1 且已登录，套用后台布局并跟随后台明暗主题 -->
  <AppLayout v-if="isEmbedded">
    <ArenaLeaderboardContent
      :response="data"
      :loading="loading"
      :initializing="initializing"
      :error="loadFailed"
      :initial-category="activeTab"
      :theme="appTheme"
      embedded
      @refresh="loadData"
      @update:category="handleCategoryChange"
    />
  </AppLayout>

  <!-- 公开页：跟随全站明暗设置（已保存的偏好，否则跟随系统），顶栏可切换 -->
  <div v-else class="arena-page" :data-theme="appTheme">
    <div class="arena-atmosphere" aria-hidden="true">
      <div class="arena-atmosphere-glow"></div>
      <div class="arena-atmosphere-grid"></div>
    </div>
    <ArenaNavBar :theme="appTheme" @toggle-theme="toggleTheme" />
    <main class="arena-page-main">
      <ArenaLeaderboardContent
        :response="data"
        :loading="loading"
        :initializing="initializing"
        :error="loadFailed"
        :initial-category="activeTab"
        :theme="appTheme"
        @refresh="loadData"
        @update:category="handleCategoryChange"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import ArenaNavBar from '@/components/arena/ArenaNavBar.vue'
import ArenaLeaderboardContent from '@/components/arena/ArenaLeaderboardContent.vue'
import { getArenaLeaderboards, type ArenaCategoryId, type ArenaLeaderboardsResponse } from '@/api/arena'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import '@/components/arena/arena-font.css'

const VALID_TABS: ReadonlySet<string> = new Set(['text', 'code', 'vision', 'image'])

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

// embedded=1 但未登录（如转发的链接）时退回公开页。
const isEmbedded = computed(() => route.query.embedded === '1' && authStore.isAuthenticated)

const activeTab = computed<ArenaCategoryId>(() => {
  const raw = typeof route.query.tab === 'string' ? route.query.tab.toLowerCase() : ''
  return VALID_TABS.has(raw) ? (raw as ArenaCategoryId) : 'text'
})

// 全站主题记在 <html> 的 dark class 和 localStorage('theme')（main.ts 启动时设置，后台侧栏也这样切换），这里跟着变。
const appTheme = ref<'dark' | 'light'>(document.documentElement.classList.contains('dark') ? 'dark' : 'light')
let themeObserver: MutationObserver | null = null
function toggleTheme() {
  const dark = appTheme.value !== 'dark'
  document.documentElement.classList.toggle('dark', dark)
  localStorage.setItem('theme', dark ? 'dark' : 'light')
}
function readTheme() {
  appTheme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const data = ref<ArenaLeaderboardsResponse | null>(null)
const loading = ref(true)
const initializing = ref(false)
const loadFailed = ref(false)

async function loadData() {
  loading.value = true
  initializing.value = false
  loadFailed.value = false
  try {
    data.value = await getArenaLeaderboards()
  } catch (err: unknown) {
    const e = err as { status?: number; response?: { status?: number } }
    if ((e?.status ?? e?.response?.status) === 503) initializing.value = true
    else loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function handleCategoryChange(next: ArenaCategoryId) {
  void router.replace({ query: { ...route.query, tab: next } })
}

onMounted(() => {
  readTheme()
  themeObserver = new MutationObserver(readTheme)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  void appStore.fetchPublicSettings()
  void loadData()
})

onBeforeUnmount(() => themeObserver?.disconnect())
</script>

<style scoped>
.arena-page {
  position: relative;
  min-height: 100vh;
  background: #07080d;
  color: #f2f4f8;
  overflow-x: clip;
  transition: background-color 0.28s cubic-bezier(0.16, 1, 0.3, 1), color 0.28s cubic-bezier(0.16, 1, 0.3, 1);
}
.arena-page[data-theme='light'] {
  background: #fafbfd;
  color: #0f172a;
}

/* Restrained architectural zenith bloom + precision hairline grid fading into pure ground */
.arena-atmosphere {
  position: absolute;
  inset: 0 0 auto 0;
  height: 680px;
  pointer-events: none;
  overflow: hidden;
  z-index: 0;
}
.arena-atmosphere-glow {
  position: absolute;
  inset: -140px -10% auto -10%;
  height: 620px;
  background:
    radial-gradient(48% 58% at 24% 12%, rgba(91, 140, 255, 0.14), rgba(91, 140, 255, 0) 70%),
    radial-gradient(42% 52% at 78% 6%, rgba(227, 193, 138, 0.075), rgba(227, 193, 138, 0) 68%),
    radial-gradient(60% 48% at 50% 0%, rgba(148, 178, 255, 0.06), rgba(7, 8, 13, 0) 72%);
  transition: opacity 0.35s ease;
}
.arena-page[data-theme='light'] .arena-atmosphere-glow {
  background:
    radial-gradient(50% 60% at 22% 8%, rgba(59, 130, 246, 0.085), rgba(59, 130, 246, 0) 72%),
    radial-gradient(40% 50% at 78% 4%, rgba(168, 118, 42, 0.055), rgba(168, 118, 42, 0) 70%);
}
.arena-atmosphere-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(to right, rgba(255, 255, 255, 0.028) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(255, 255, 255, 0.028) 1px, transparent 1px);
  background-size: 64px 64px;
  mask-image: radial-gradient(78% 72% at 50% 0%, #000 0%, rgba(0, 0, 0, 0.45) 54%, transparent 100%);
  -webkit-mask-image: radial-gradient(78% 72% at 50% 0%, #000 0%, rgba(0, 0, 0, 0.45) 54%, transparent 100%);
}
.arena-page[data-theme='light'] .arena-atmosphere-grid {
  background-image:
    linear-gradient(to right, rgba(15, 23, 42, 0.032) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(15, 23, 42, 0.032) 1px, transparent 1px);
}

.arena-page-main {
  position: relative;
  z-index: 1;
  max-width: 1184px;
  margin: 0 auto;
  padding: 0 32px;
}

@media (max-width: 640px) {
  .arena-page-main {
    padding: 0 20px;
  }
  .arena-atmosphere {
    height: 460px;
  }
}
</style>
