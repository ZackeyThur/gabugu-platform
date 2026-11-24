<template>
  <div class="space-y-6">
    <!-- Editorial Masthead (independent mode) -->
    <div
      v-if="!embedded"
      class="g-stagger-in flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between"
      style="--stagger-i: 0"
    >
      <div>
        <div class="mb-2 inline-flex items-center gap-2">
          <span class="h-px w-5 bg-clay-600 dark:bg-gradient-to-r dark:from-[#A78BFA] dark:to-[#38D3B4]" aria-hidden="true" />
          <span class="text-xs font-semibold uppercase tracking-[0.16em] text-clay-700 dark:text-[#A5BFFF]">
            {{ t('modelPlaza.eyebrow') }}
          </span>
        </div>
        <h1 class="g-blur-in text-3xl font-bold tracking-[-0.025em] text-gray-900 dark:text-white sm:text-4xl">
          {{ t('modelPlaza.title') }}
        </h1>
        <p class="mt-2 max-w-2xl text-sm leading-relaxed text-gray-500 dark:text-dark-400">
          {{ t('modelPlaza.description') }}
        </p>
      </div>

      <!-- Live Catalog Summary Metrics Strip -->
      <div
        v-if="!loading && !error && catalogStats.groups > 0"
        class="flex flex-wrap items-center gap-2.5 sm:shrink-0"
      >
        <div
          class="ui-pill flex items-center gap-2.5 rounded-xl px-3.5 py-2"
        >
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.stats.groups') }}</span>
          <span class="font-mono text-base font-bold tabular-nums text-gray-900 dark:text-white">
            <CountUp :value="catalogStats.groups" />
          </span>
        </div>
        <div
          class="ui-pill flex items-center gap-2.5 rounded-xl px-3.5 py-2"
        >
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.stats.models') }}</span>
          <span class="font-mono text-base font-bold tabular-nums text-gray-900 dark:text-white">
            <CountUp :value="catalogStats.models" />
          </span>
        </div>
        <div
          v-if="catalogStats.lowestRate !== null"
          class="ui-pill flex items-center gap-2.5 rounded-xl px-3.5 py-2 dark:ring-1 dark:ring-inset dark:ring-[#F6D494]/[0.22]"
        >
          <span class="text-xs text-clay-700 dark:text-[#F6D494]">{{ t('modelPlaza.stats.lowestRate') }}</span>
          <span class="font-mono text-base font-bold tabular-nums text-clay-700 dark:text-[#F6D494]">
            {{ catalogStats.lowestRate }}x
          </span>
        </div>
      </div>
    </div>

    <!-- 全局价格说明 + 匿名登录条 (合并为紧凑高质感说明区) -->
    <div
      v-if="descriptionHtml || !isAuthenticated"
      class="g-stagger-in space-y-3"
      style="--stagger-i: 1"
    >
      <div
        v-if="descriptionHtml"
        data-no-spotlight
        class="plaza-description ui-flat-panel overflow-hidden px-5 py-3.5 pl-6 text-sm"
      >
        <span
          class="absolute inset-y-3 left-2.5 w-[3px] rounded-full bg-gradient-to-b from-clay-500 via-clay-600 to-primary-600 dark:from-[#38D3B4] dark:via-[#7CA2FF] dark:to-[#A78BFA]"
          aria-hidden="true"
        />
        <div v-html="descriptionHtml"></div>
      </div>

      <div
        v-if="!isAuthenticated"
        class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-gray-200/70 bg-pearl-raised/45 px-4 py-2.5 text-xs text-gray-600 dark:border-dark-700/60 dark:bg-dark-800/45 dark:text-dark-300"
      >
        <span class="flex items-center gap-2">
          <Icon name="infoCircle" size="xs" class="h-3.5 w-3.5 shrink-0 text-primary-600 dark:text-primary-400" />
          {{ t('modelPlaza.anonymousHint') }}
        </span>
        <RouterLink
          :to="{ path: '/login', query: { redirect: '/model-plaza' } }"
          class="g-hover-arrow inline-flex items-center gap-1 font-semibold text-primary-700 transition-colors hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('modelPlaza.nav.login') }}
          <Icon name="arrowRight" size="xs" />
        </RouterLink>
      </div>
    </div>

    <!-- 加载/错误/空 -->
    <div v-if="loading" class="space-y-5" aria-busy="true">
      <div class="card space-y-3.5 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="skeleton h-10 w-72 rounded-xl"></div>
          <div class="flex gap-2">
            <div v-for="chip in 4" :key="chip" class="skeleton h-9 w-20 rounded-xl"></div>
          </div>
        </div>
        <div class="flex gap-2 pt-1">
          <div v-for="chip in 5" :key="chip" class="skeleton h-7 w-16 rounded-lg"></div>
        </div>
      </div>
      <div v-for="card in 2" :key="card" class="card space-y-4 p-6">
        <div class="flex items-center justify-between">
          <div class="skeleton h-6 w-44 rounded-lg"></div>
          <div class="skeleton h-6 w-20 rounded-full"></div>
        </div>
        <div class="space-y-2.5">
          <div class="skeleton h-12 w-full rounded-xl"></div>
          <div class="skeleton h-12 w-full rounded-xl"></div>
        </div>
      </div>
    </div>
    <div
      v-else-if="error"
      class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
    >
      {{ t('modelPlaza.loadFailed') }}
    </div>
    <template v-else>
      <!-- 筛选区:双层精简指令栏 -->
      <div class="g-stagger-in" style="--stagger-i: 2">
        <PlazaFilterBar
          :platforms="platforms"
          :groups="groupOptions"
          :rates="rates"
          :platform="selectedPlatform"
          :group-id="selectedGroupId"
          :rate="selectedRate"
          :search="searchQuery"
          @update:platform="selectedPlatform = $event"
          @update:group-id="selectedGroupId = $event"
          @update:rate="selectedRate = $event"
          @update:search="searchQuery = $event"
        />
      </div>

      <!-- 分组分节的模型清单(默认按生效倍率升序) -->
      <div v-if="filteredGroups.length > 0" class="space-y-6">
        <PlazaGroupSection
          v-for="(g, idx) in filteredGroups"
          :key="g.id"
          :group="g"
          class="g-stagger-in"
          :style="{ '--stagger-i': Math.min(idx + 3, 8) }"
        />
      </div>
      <div
        v-else
        class="rounded-2xl border border-dashed border-gray-300/90 bg-pearl-surface/50 px-5 py-14 text-center text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-800/30 dark:text-dark-400"
      >
        {{ searchActive ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import CountUp from '@/components/common/CountUp.vue'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  /** 后台内嵌形态(AppLayout 内):隐藏页头。 */
  embedded?: boolean
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref('')

const searchActive = computed(() => searchQuery.value.trim() !== '')

const descriptionHtml = computed(() => {
  const md = props.response?.description?.trim()
  if (!md) return ''
  return DOMPurify.sanitize(marked.parse(md) as string)
})

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
function effectiveRate(g: ModelPlazaGroup): number {
  return g.user_rate_multiplier ?? g.rate_multiplier
}

const catalogStats = computed(() => {
  const groups = props.response?.groups ?? []
  const modelNames = new Set<string>()
  let minRate: number | null = null
  for (const g of groups) {
    const r = effectiveRate(g)
    if (Number.isFinite(r) && (minRate === null || r < minRate)) minRate = r
    for (const m of g.models ?? []) {
      if (m.name) modelNames.add(m.name)
    }
  }
  return {
    groups: groups.length,
    models: modelNames.size,
    lowestRate: minRate
  }
})

const platforms = computed(() =>
  [...new Set((props.response?.groups ?? []).map((g) => g.platform).filter(Boolean))].sort()
)

const groupOptions = computed(() =>
  (props.response?.groups ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    platform: g.platform,
    rate: effectiveRate(g)
  }))
)

/** 全量生效倍率;当前组合下不可用的项由 FilterBar 置灰而非隐藏。 */
const rates = computed(() =>
  [...new Set((props.response?.groups ?? []).map(effectiveRate))].sort((a, b) => a - b)
)

/** 数据刷新后选中的倍率可能不复存在,重置为全部。 */
watch(rates, (list) => {
  if (selectedRate.value !== 'all' && !list.includes(selectedRate.value)) {
    selectedRate.value = 'all'
  }
})

const filteredGroups = computed(() => {
  let groups = props.response?.groups ?? []
  if (selectedPlatform.value !== 'all') {
    groups = groups.filter((g) => g.platform === selectedPlatform.value)
  }
  if (selectedGroupId.value !== 'all') {
    groups = groups.filter((g) => g.id === selectedGroupId.value)
  }
  if (selectedRate.value !== 'all') {
    groups = groups.filter((g) => effectiveRate(g) === selectedRate.value)
  }
  // 模型名搜索:分组内只留命中的模型,整组无命中则隐藏该分组。
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    groups = groups
      .map((g) => ({ ...g, models: g.models.filter((m) => m.name.toLowerCase().includes(q)) }))
      .filter((g) => g.models.length > 0)
  }
  // 专属倍率会改变生效值,不能只依赖后端按默认倍率的排序。
  return [...groups].sort(
    (a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name)
  )
})
</script>

<style scoped>
.plaza-description {
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.plaza-description :deep(h1),
.plaza-description :deep(h2),
.plaza-description :deep(h3) {
  @apply mb-2 mt-3 font-semibold text-gray-900 first:mt-0 dark:text-white;
}

.plaza-description :deep(p) {
  @apply mb-2 text-gray-700 last:mb-0 dark:text-dark-200;
}

.plaza-description :deep(strong) {
  @apply font-semibold text-gray-900 dark:text-white;
}

.plaza-description :deep(a) {
  @apply text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300;
}

.plaza-description :deep(ul) {
  @apply mb-2 list-disc pl-5;
}

.plaza-description :deep(ol) {
  @apply mb-2 list-decimal pl-5;
}

.plaza-description :deep(li) {
  @apply mb-0.5 text-gray-700 dark:text-dark-200;
}

.plaza-description :deep(code) {
  @apply rounded bg-clay-50 px-1.5 py-0.5 font-mono text-xs text-clay-700 dark:bg-dark-800 dark:text-primary-300;
}

.plaza-description :deep(blockquote) {
  @apply my-2 border-l-4 border-clay-300 pl-3 text-gray-600 dark:border-dark-600 dark:text-dark-300;
}
</style>
