<template>
  <div data-no-spotlight class="ui-flat-panel overflow-hidden p-4 sm:p-5">
    <!-- Row 1: Command Search + Platform Segmented Rail + Reset -->
    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="relative w-full lg:max-w-sm">
        <Icon
          name="search"
          size="sm"
          class="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-400"
        />
        <input
          ref="searchInputRef"
          :value="search"
          type="text"
          :placeholder="t('modelPlaza.filters.searchPlaceholder')"
          class="input rounded-xl py-2 pl-10 pr-14 text-sm"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="search"
          type="button"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-md p-1 text-gray-400 transition-colors hover:bg-pearl-raised hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-gray-200"
          @click="$emit('update:search', '')"
        >
          <Icon name="x" size="xs" class="h-3.5 w-3.5" />
        </button>
        <kbd
          v-else
          class="pointer-events-none absolute right-3 top-1/2 hidden -translate-y-1/2 rounded-md border border-gray-200/90 bg-pearl-bg/80 px-1.5 py-0.5 font-mono text-xs leading-none text-gray-400 dark:border-dark-600 dark:bg-dark-900/70 dark:text-dark-400 sm:inline-block"
          aria-hidden="true"
        >/</kbd>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <span class="mr-0.5 text-xs font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-dark-400">
          {{ t('modelPlaza.filters.platformLabel') }}
        </span>
        <div
          class="flex flex-wrap items-center gap-1.5 rounded-xl bg-pearl-raised/60 p-1 dark:bg-white/[0.025]"
        >
          <button
            v-for="p in ['all', ...platforms]"
            :key="`platform-${p}`"
            type="button"
            class="plaza-chip inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all duration-200 active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-35 disabled:grayscale"
            :class="p === 'all' ? chipClass(platform === 'all') : platform === p ? 'chip-tinted-active' : 'chip-tinted'"
            :style="p === 'all' ? undefined : { '--chip-accent': platformAccentColor(p) }"
            :disabled="p !== 'all' && !platformEnabled(p)"
            @click="$emit('update:platform', p)"
          >
            <PlatformIcon v-if="p !== 'all'" :platform="p as GroupPlatform" size="xs" />
            <span>{{ p === 'all' ? t('modelPlaza.filters.all') : p }}</span>
          </button>
        </div>

        <button
          v-if="hasActiveFilter"
          type="button"
          class="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium text-clay-700 transition-colors hover:bg-clay-50 dark:text-primary-300 dark:hover:bg-primary-500/15"
          @click="resetAll"
        >
          <Icon name="x" size="xs" class="h-3 w-3" />
          {{ t('modelPlaza.filters.reset') }}
        </button>
      </div>
    </div>

    <!-- Row 2: Compact Inline Facet Rail (Groups + Rates) -->
    <div
      class="mt-3.5 flex flex-col gap-3 border-t border-gray-200/70 pt-3.5 dark:border-dark-700/70 lg:flex-row lg:items-center lg:gap-5"
    >
      <!-- 分组 -->
      <div class="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
        <span class="mr-1 shrink-0 text-xs font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-dark-400">
          {{ t('modelPlaza.filters.groupLabel') }}
        </span>
        <button
          type="button"
          class="plaza-chip rounded-lg px-2.5 py-1 text-xs font-medium transition-all duration-150 active:scale-[0.97]"
          :class="chipClass(groupId === 'all')"
          @click="$emit('update:groupId', 'all')"
        >
          {{ t('modelPlaza.filters.all') }}
        </button>
        <button
          v-for="g in groups"
          :key="`group-${g.id}`"
          type="button"
          class="plaza-chip rounded-lg px-2.5 py-1 text-xs font-medium transition-all duration-150 active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-35 disabled:grayscale"
          :class="groupId === g.id ? 'chip-tinted-active' : 'chip-tinted'"
          :style="{ '--chip-accent': platformAccentColor(g.platform) }"
          :disabled="!groupEnabled(g)"
          @click="$emit('update:groupId', g.id)"
        >
          {{ g.name }}
        </button>
      </div>

      <!-- Divider on desktop -->
      <div class="hidden h-5 w-px shrink-0 bg-gray-200/80 dark:bg-dark-700 lg:block" aria-hidden="true" />

      <!-- 倍率 -->
      <div class="flex shrink-0 flex-wrap items-center gap-1.5">
        <span class="mr-1 shrink-0 text-xs font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-dark-400">
          {{ t('modelPlaza.filters.rateLabel') }}
        </span>
        <button
          type="button"
          class="plaza-chip rounded-lg px-2.5 py-1 text-xs font-medium transition-all duration-150 active:scale-[0.97]"
          :class="chipClass(rate === 'all')"
          @click="$emit('update:rate', 'all')"
        >
          {{ t('modelPlaza.filters.all') }}
        </button>
        <button
          v-for="r in rates"
          :key="`rate-${r}`"
          type="button"
          class="plaza-chip rounded-lg px-2.5 py-1 font-mono text-xs font-semibold tabular-nums transition-all duration-150 active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-35 disabled:grayscale"
          :class="chipClass(rate === r)"
          :disabled="!rateEnabled(r)"
          @click="$emit('update:rate', r)"
        >
          {{ r }}x
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformAccentColor } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  /** 数据中出现的平台(去重排序后)。 */
  platforms: string[]
  /** 全量分组(含平台与生效倍率),三个维度的置灰联动由此推导。 */
  groups: Array<{ id: number; name: string; platform: string; rate: number }>
  /** 全量生效倍率去重升序。 */
  rates: number[]
  platform: string
  groupId: number | 'all'
  rate: number | 'all'
  /** 模型名搜索词(纯前端过滤)。 */
  search: string
}>()

const emit = defineEmits<{
  'update:platform': [value: string]
  'update:groupId': [value: number | 'all']
  'update:rate': [value: number | 'all']
  'update:search': [value: string]
}>()

const { t } = useI18n()
const searchInputRef = ref<HTMLInputElement | null>(null)

const hasActiveFilter = computed(
  () =>
    props.platform !== 'all' ||
    props.groupId !== 'all' ||
    props.rate !== 'all' ||
    props.search.trim() !== ''
)

function resetAll() {
  emit('update:platform', 'all')
  emit('update:groupId', 'all')
  emit('update:rate', 'all')
  emit('update:search', '')
}

function handleSlashFocus(e: KeyboardEvent) {
  if (e.key !== '/' || e.ctrlKey || e.metaKey || e.altKey) return
  const target = e.target as HTMLElement | null
  if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) {
    return
  }
  e.preventDefault()
  searchInputRef.value?.focus()
}

onMounted(() => document.addEventListener('keydown', handleSlashFocus))
onBeforeUnmount(() => document.removeEventListener('keydown', handleSlashFocus))

/**
 * 三个维度互为约束(faceted):某选项可点 ⟺ 在「其他两维」当前选择下仍有分组命中。
 * 「全部」永远可点,作为解除本维约束的出口;可点项组合恒有结果,无需选择修正。
 */
function platformEnabled(p: string): boolean {
  return props.groups.some(
    (g) =>
      g.platform === p &&
      (props.groupId === 'all' || g.id === props.groupId) &&
      (props.rate === 'all' || g.rate === props.rate)
  )
}

function groupEnabled(g: { platform: string; rate: number }): boolean {
  return (
    (props.platform === 'all' || g.platform === props.platform) &&
    (props.rate === 'all' || g.rate === props.rate)
  )
}

function rateEnabled(r: number): boolean {
  return props.groups.some(
    (g) =>
      g.rate === r &&
      (props.platform === 'all' || g.platform === props.platform) &&
      (props.groupId === 'all' || g.id === props.groupId)
  )
}

function chipClass(active: boolean): string {
  return active
    ? 'bg-primary-600 text-white shadow-[0_1px_3px_rgba(24,47,85,0.22),inset_0_1px_0_rgba(255,255,255,0.22)] dark:bg-primary-500/85 dark:shadow-none'
    : 'bg-pearl-surface/90 text-gray-600 ring-1 ring-inset ring-gray-200/90 enabled:hover:bg-pearl-raised/80 enabled:hover:text-gray-900 enabled:hover:ring-gray-300 dark:bg-white/[0.03] dark:text-dark-200 dark:ring-white/[0.07] dark:enabled:hover:bg-white/[0.07] dark:enabled:hover:text-white'
}
</script>

<style scoped>
.chip-tinted {
  color: var(--chip-accent);
  color: color-mix(in srgb, var(--chip-accent) 78%, black);
  background-color: color-mix(in srgb, var(--chip-accent) 7%, rgb(var(--c-white)));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chip-accent) 22%, transparent);
}

.chip-tinted:not(:disabled):hover {
  background-color: color-mix(in srgb, var(--chip-accent) 14%, rgb(var(--c-white)));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chip-accent) 34%, transparent);
}

.dark .chip-tinted {
  color: color-mix(in srgb, var(--chip-accent) 72%, #dce5f7);
  background-color: color-mix(in srgb, var(--chip-accent) 8%, transparent);
  box-shadow: inset 0 0 0 0.5px color-mix(in srgb, var(--chip-accent) 24%, transparent);
}

.dark .chip-tinted:not(:disabled):hover {
  background-color: color-mix(in srgb, var(--chip-accent) 16%, transparent);
  box-shadow: inset 0 0 0 0.5px color-mix(in srgb, var(--chip-accent) 38%, transparent);
}

.chip-tinted-active {
  color: #fff;
  background-color: var(--chip-accent);
  background-color: color-mix(in srgb, var(--chip-accent) 84%, black);
  box-shadow:
    0 2px 6px -1px color-mix(in srgb, var(--chip-accent) 38%, transparent),
    inset 0 1px 0 rgba(255, 255, 255, 0.22);
}

.chip-tinted-active:not(:disabled):hover {
  background-color: color-mix(in srgb, var(--chip-accent) 76%, black);
}

.dark .chip-tinted-active {
  background-color: color-mix(in srgb, var(--chip-accent) 82%, transparent);
}

.dark .chip-tinted-active:not(:disabled):hover {
  background-color: var(--chip-accent);
}
</style>
