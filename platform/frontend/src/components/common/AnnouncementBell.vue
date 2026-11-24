<template>
  <div>
    <!-- 铃铛按钮 -->
    <button
      type="button"
      @click="openModal"
      class="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-600 transition-colors hover:bg-pearl-raised hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
      :class="{ 'text-primary-700 dark:text-primary-300': unreadCount > 0 }"
      :aria-label="t('announcements.title')"
    >
      <Icon name="bell" size="md" />
      <!-- 未读角标 -->
      <span
        v-if="unreadCount > 0"
        class="absolute -right-0.5 -top-0.5 flex h-[17px] min-w-[17px] items-center justify-center rounded-full bg-[#A5532F] px-1 text-xs font-bold leading-none tabular-nums text-white ring-2 ring-pearl-surface dark:bg-primary-500 dark:ring-dark-800"
      >
        {{ unreadCount > 9 ? '9+' : unreadCount }}
      </span>
    </button>

    <!-- 公告中心 Modal（单层遮罩：列表 ↔ 详情顺滑切换） -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="isModalOpen || detailModalOpen"
          class="fixed inset-0 z-[100] flex items-start justify-center overflow-y-auto bg-black/55 p-4 pt-[7vh] backdrop-blur-sm sm:items-center sm:pt-4"
          role="dialog"
          aria-modal="true"
          :aria-label="detailModalOpen && selectedAnnouncement ? selectedAnnouncement.title : t('announcements.title')"
          @click="closeAll"
        >
          <div
            :class="[
              'w-full overflow-hidden rounded-2xl border border-gray-200/90 bg-pearl-surface shadow-[0_24px_64px_-16px_rgb(var(--shadow)/0.28)] transition-[max-width] duration-200 dark:border-dark-700 dark:bg-dark-800 dark:shadow-[0_28px_72px_-16px_rgba(0,0,0,0.65)]',
              detailModalOpen && selectedAnnouncement ? 'max-w-[700px]' : 'max-w-[580px]'
            ]"
            @click.stop
          >
            <!-- 详情视图 -->
            <template v-if="detailModalOpen && selectedAnnouncement">
              <!-- Top accent hairline -->
              <div class="h-[3px] w-full bg-gradient-to-r from-[#2D5890] via-[#4C78B3] to-[#A5532F] dark:from-primary-500 dark:via-primary-400 dark:to-amber-400"></div>

              <!-- Detail Header -->
              <div class="border-b border-gray-200/80 bg-pearl-bg/45 px-6 pb-5 pt-4 sm:px-7 dark:border-dark-700/70 dark:bg-dark-900/50">
                <div class="flex items-center justify-between gap-3">
                  <div class="flex items-center gap-2.5">
                    <button
                      type="button"
                      class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs font-medium text-gray-600 transition-colors hover:bg-pearl-raised hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-white"
                      @click="closeDetail"
                    >
                      <Icon name="arrowLeft" size="xs" />
                      <span>{{ t('announcements.backToList') }}</span>
                    </button>
                    <span class="h-3.5 w-px bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
                    <span
                      v-if="!selectedAnnouncement.read_at"
                      class="inline-flex items-center gap-1.5 rounded-full border border-[#E2C3AE] bg-[#F6E7DC] px-2.5 py-0.5 text-xs font-semibold text-[#A5532F] dark:border-amber-500/30 dark:bg-amber-500/15 dark:text-amber-300"
                    >
                      <span class="h-1.5 w-1.5 rounded-full bg-[#A5532F] dark:bg-amber-400" aria-hidden="true"></span>
                      {{ t('announcements.unread') }}
                    </span>
                    <span
                      v-else
                      class="inline-flex items-center gap-1 rounded-full bg-emerald-100/80 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                    >
                      {{ t('announcements.read') }}
                    </span>
                  </div>

                  <button
                    type="button"
                    class="btn-ghost btn-icon shrink-0 text-gray-400 hover:text-gray-700 dark:text-dark-400 dark:hover:text-gray-200"
                    :aria-label="t('common.close')"
                    @click="closeAll"
                  >
                    <Icon name="x" size="sm" />
                  </button>
                </div>

                <!-- Title -->
                <h2 class="mt-3.5 text-xl font-bold leading-snug tracking-tight text-gray-900 sm:text-[22px] dark:text-white">
                  {{ selectedAnnouncement.title }}
                </h2>

                <!-- Meta -->
                <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
                  <span class="inline-flex items-center gap-1.5 tabular-nums">
                    <svg class="h-3.5 w-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <time>{{ formatRelativeWithDateTime(selectedAnnouncement.created_at) }}</time>
                  </span>
                </div>
              </div>

              <!-- Detail Body -->
              <div class="announcement-scroll max-h-[58vh] overflow-y-auto bg-pearl-surface px-6 py-6 sm:px-7 dark:bg-dark-800">
                <AnnouncementContent :content="selectedAnnouncement.content" />
              </div>

              <!-- Detail Footer -->
              <div class="border-t border-gray-200/80 bg-pearl-bg/40 px-6 py-4 sm:px-7 dark:border-dark-700/70 dark:bg-dark-900/40">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                    <Icon name="infoCircle" size="xs" class="shrink-0" />
                    <span>{{ selectedAnnouncement.read_at ? t('announcements.readStatus') : t('announcements.markReadHint') }}</span>
                  </div>
                  <div class="flex items-center gap-2.5">
                    <button
                      type="button"
                      class="btn btn-secondary"
                      @click="closeDetail"
                    >
                      {{ t('common.close') }}
                    </button>
                    <button
                      v-if="!selectedAnnouncement.read_at"
                      type="button"
                      class="btn btn-primary"
                      @click="markAsReadAndClose(selectedAnnouncement.id)"
                    >
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                      <span>{{ t('announcements.markRead') }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </template>

            <!-- 列表视图 -->
            <template v-else>
              <!-- Top accent hairline -->
              <div class="h-[3px] w-full bg-primary-600 dark:bg-primary-500"></div>

              <!-- List Header -->
              <div class="border-b border-gray-200/80 bg-pearl-bg/45 px-5 py-4 sm:px-6 dark:border-dark-700/70 dark:bg-dark-900/50">
                <div class="flex items-center justify-between gap-3">
                  <div class="flex min-w-0 items-center gap-3">
                    <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary-600/[0.09] text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">
                      <Icon name="bell" size="sm" />
                    </div>
                    <div class="flex min-w-0 items-center gap-2">
                      <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                        {{ t('announcements.title') }}
                      </h2>
                      <span
                        v-if="unreadCount > 0"
                        class="inline-flex shrink-0 items-center rounded-full border border-[#E2C3AE] bg-[#F6E7DC] px-2 py-0.5 text-xs font-semibold tabular-nums text-[#A5532F] dark:border-primary-500/30 dark:bg-primary-500/15 dark:text-primary-300"
                      >
                        {{ unreadCount }} {{ t('announcements.unread') }}
                      </span>
                    </div>
                  </div>

                  <div class="flex shrink-0 items-center gap-2">
                    <button
                      v-if="unreadCount > 0"
                      type="button"
                      :disabled="loading"
                      class="btn btn-secondary btn-sm"
                      @click="markAllAsRead"
                    >
                      {{ t('announcements.markAllRead') }}
                    </button>
                    <button
                      type="button"
                      class="btn-ghost btn-icon text-gray-400 hover:text-gray-700 dark:text-dark-400 dark:hover:text-gray-200"
                      :aria-label="t('common.close')"
                      @click="closeModal"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>
                </div>

                <!-- Segmented filter tabs when announcements exist -->
                <div v-if="!loading && announcements.length > 0" class="mt-3 flex items-center gap-1.5">
                  <button
                    type="button"
                    class="rounded-lg px-2.5 py-1 text-xs font-medium transition-colors"
                    :class="filterMode === 'all'
                      ? 'bg-pearl-surface text-gray-900 shadow-sm ring-1 ring-gray-200/90 dark:bg-dark-700 dark:text-white dark:ring-dark-600'
                      : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-gray-200'"
                    @click="filterMode = 'all'"
                  >
                    {{ t('announcements.all') }}
                    <span class="ml-1 tabular-nums opacity-65">{{ announcements.length }}</span>
                  </button>
                  <button
                    type="button"
                    class="rounded-lg px-2.5 py-1 text-xs font-medium transition-colors"
                    :class="filterMode === 'unread'
                      ? 'bg-pearl-surface text-gray-900 shadow-sm ring-1 ring-gray-200/90 dark:bg-dark-700 dark:text-white dark:ring-dark-600'
                      : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-gray-200'"
                    @click="filterMode = 'unread'"
                  >
                    {{ t('announcements.unread') }}
                    <span class="ml-1 tabular-nums opacity-65">{{ unreadCount }}</span>
                  </button>
                </div>
              </div>

              <!-- List Body -->
              <div class="announcement-scroll max-h-[60vh] overflow-y-auto p-3">
                <!-- Skeleton Loading -->
                <div v-if="loading" class="space-y-2 p-1" aria-busy="true">
                  <div v-for="n in 3" :key="n" class="rounded-xl border border-gray-200/60 p-4 dark:border-dark-700/60">
                    <div class="flex items-center justify-between gap-3">
                      <div class="skeleton h-4 w-48"></div>
                      <div class="skeleton h-3 w-16"></div>
                    </div>
                    <div class="skeleton mt-2.5 h-3 w-3/4"></div>
                  </div>
                </div>

                <!-- Announcements List -->
                <div v-else-if="filteredAnnouncements.length > 0" class="space-y-1.5">
                  <div
                    v-for="item in filteredAnnouncements"
                    :key="item.id"
                    role="button"
                    tabindex="0"
                    class="group relative flex cursor-pointer items-start gap-3.5 rounded-xl border px-4 py-3.5 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50"
                    :class="!item.read_at
                      ? 'border-primary-200/80 bg-primary-600/[0.04] hover:border-primary-300 hover:bg-primary-600/[0.08] dark:border-primary-500/25 dark:bg-primary-500/[0.08] dark:hover:border-primary-500/40 dark:hover:bg-primary-500/[0.13]'
                      : 'border-transparent bg-transparent hover:border-gray-200/80 hover:bg-pearl-bg/65 dark:hover:border-dark-700 dark:hover:bg-dark-700/45'"
                    @click="openDetail(item)"
                    @keydown.enter.prevent="openDetail(item)"
                    @keydown.space.prevent="openDetail(item)"
                  >
                    <!-- Unread / Read Status Dot -->
                    <div class="pt-1.5 shrink-0">
                      <span
                        v-if="!item.read_at"
                        class="block h-2.5 w-2.5 rounded-full bg-[#A5532F] ring-4 ring-[#A5532F]/15 dark:bg-primary-400 dark:ring-primary-400/20"
                        aria-hidden="true"
                      ></span>
                      <span
                        v-else
                        class="block h-2.5 w-2.5 rounded-full border border-gray-300 dark:border-dark-500"
                        aria-hidden="true"
                      ></span>
                    </div>

                    <!-- Title + Preview Snippet -->
                    <div class="min-w-0 flex-1">
                      <div class="flex items-baseline justify-between gap-3">
                        <h3
                          class="truncate text-sm transition-colors"
                          :class="!item.read_at
                            ? 'font-semibold text-gray-900 dark:text-white'
                            : 'font-medium text-gray-700 group-hover:text-gray-900 dark:text-gray-300 dark:group-hover:text-white'"
                        >
                          {{ item.title }}
                        </h3>
                        <time class="shrink-0 text-xs tabular-nums text-gray-400 dark:text-dark-400">
                          {{ formatRelativeTime(item.created_at) }}
                        </time>
                      </div>
                      <p
                        v-if="previewText(item.content)"
                        class="mt-1 line-clamp-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400"
                      >
                        {{ previewText(item.content) }}
                      </p>
                    </div>

                    <!-- Chevron -->
                    <Icon
                      name="chevronRight"
                      size="sm"
                      class="mt-1 shrink-0 text-gray-300 transition-transform duration-150 group-hover:translate-x-0.5 group-hover:text-primary-600 dark:text-dark-500 dark:group-hover:text-primary-400"
                    />
                  </div>
                </div>

                <!-- Empty State -->
                <div v-else class="flex flex-col items-center justify-center px-6 py-14 text-center">
                  <div class="mb-3.5 flex h-12 w-12 items-center justify-center rounded-xl bg-pearl-raised text-gray-500 dark:bg-dark-700 dark:text-dark-300">
                    <Icon name="inbox" size="lg" />
                  </div>
                  <p class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ filterMode === 'unread' ? t('announcements.emptyUnread') : t('announcements.empty') }}
                  </p>
                  <p class="mt-1 max-w-xs text-xs text-gray-500 dark:text-dark-400">
                    {{ t('announcements.emptyDescription') }}
                  </p>
                </div>
              </div>
            </template>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import AnnouncementContent from './AnnouncementContent.vue'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import '@/styles/announcement-markdown.css'

const { t } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()

// Use store state (storeToRefs for reactivity)
const { announcements, loading } = storeToRefs(announcementStore)
const unreadCount = computed(() => announcementStore.unreadCount)

// Local modal state
const isModalOpen = ref(false)
const detailModalOpen = ref(false)
const selectedAnnouncement = ref<UserAnnouncement | null>(null)
const filterMode = ref<'all' | 'unread'>('all')

const filteredAnnouncements = computed(() => (
  filterMode.value === 'unread'
    ? announcements.value.filter((item) => !item.read_at)
    : announcements.value
))

function previewText(content: string): string {
  if (!content) return ''
  return content
    .replace(/<[^>]+>/g, ' ')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/[#>*_~|]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

// Methods
function openModal() {
  filterMode.value = 'all'
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
}

function openDetail(announcement: UserAnnouncement) {
  selectedAnnouncement.value = announcement
  detailModalOpen.value = true
  if (!announcement.read_at) {
    markAsRead(announcement.id)
  }
}

function closeDetail() {
  detailModalOpen.value = false
  selectedAnnouncement.value = null
}

function closeAll() {
  detailModalOpen.value = false
  selectedAnnouncement.value = null
  isModalOpen.value = false
}

async function markAsRead(id: number) {
  try {
    const marked = await announcementStore.markAsRead(id)
    if (!marked) appStore.showError(t('common.unknownError'))
    return marked
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
    return false
  }
}

async function markAsReadAndClose(id: number) {
  if (!await markAsRead(id)) return
  appStore.showSuccess(t('announcements.markedAsRead'))
  closeDetail()
}

async function markAllAsRead() {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (detailModalOpen.value) {
      closeDetail()
    } else if (isModalOpen.value) {
      closeModal()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
  document.body.style.overflow = ''
})

watch(
  [isModalOpen, detailModalOpen, () => announcementStore.currentPopup],
  ([modal, detail, popup]) => {
    document.body.style.overflow = (modal || detail || popup) ? 'hidden' : ''
  }
)
</script>

<style scoped>
.modal-fade-enter-active {
  transition: opacity 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-fade-leave-active {
  transition: opacity 0.16s cubic-bezier(0.4, 0, 1, 1);
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-active > div {
  transition: transform 0.24s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.22s ease;
}

.modal-fade-leave-active > div {
  transition: transform 0.16s ease, opacity 0.16s ease;
}

.modal-fade-enter-from > div {
  transform: scale(0.97) translateY(8px);
  opacity: 0;
}

.modal-fade-leave-to > div {
  transform: scale(0.98) translateY(4px);
  opacity: 0;
}

.announcement-scroll::-webkit-scrollbar {
  width: 6px;
}

.announcement-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.announcement-scroll::-webkit-scrollbar-thumb {
  background: rgb(var(--gray-300) / 0.8);
  border-radius: 999px;
}

.dark .announcement-scroll::-webkit-scrollbar-thumb {
  background: rgba(100, 116, 139, 0.45);
}

.announcement-scroll::-webkit-scrollbar-thumb:hover {
  background: rgb(var(--gray-400) / 0.9);
}

.dark .announcement-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.6);
}
</style>
