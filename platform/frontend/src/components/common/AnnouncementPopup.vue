<template>
  <Teleport to="body">
    <Transition name="popup-fade">
      <div
        v-if="displayedAnnouncement"
        class="fixed inset-0 z-[120] flex items-start justify-center overflow-y-auto bg-black/55 p-4 pt-[7vh] backdrop-blur-sm sm:items-center sm:pt-4"
        role="dialog"
        aria-modal="true"
        :aria-label="displayedAnnouncement.title"
        @click="preview && handleDismiss()"
      >
        <div
          class="ui-popover w-full max-w-[640px] overflow-hidden rounded-[22px]"
          @click.stop
        >
          <!-- Top editorial accent hairline -->
          <div class="h-[2px] w-full bg-gradient-to-r from-[#A5532F] via-[#C6754E] to-[#2D5890] dark:from-[#5682F5] dark:via-[#A78BFA] dark:to-[#38D3B4]"></div>

          <!-- Header -->
          <div class="border-b border-gray-200/60 bg-pearl-bg/35 px-6 pb-5 pt-5 sm:px-7 sm:pt-6 dark:border-white/[0.08] dark:bg-white/[0.025]">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex-1">
                <!-- Eyebrow & Timestamp -->
                <div class="flex flex-wrap items-center gap-2.5">
                  <span class="inline-flex items-center gap-1.5 rounded-full border border-[#E2C3AE] bg-[#F6E7DC] px-2.5 py-0.5 text-xs font-semibold text-[#A5532F] dark:border-amber-500/30 dark:bg-amber-500/15 dark:text-amber-300">
                    <span class="h-1.5 w-1.5 rounded-full bg-[#A5532F] dark:bg-amber-400" aria-hidden="true"></span>
                    {{ t('announcements.importantNotice') }}
                  </span>
                  <span class="inline-flex items-center gap-1 text-xs tabular-nums text-gray-500 dark:text-dark-400">
                    <svg class="h-3.5 w-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <time>{{ formatRelativeWithDateTime(displayedAnnouncement.created_at) }}</time>
                  </span>
                </div>

                <!-- Title -->
                <h2 class="mt-3 text-xl font-bold leading-snug tracking-tight text-gray-900 sm:text-[22px] dark:text-white">
                  {{ displayedAnnouncement.title }}
                </h2>
              </div>

              <!-- Close icon (preview mode) -->
              <button
                v-if="preview"
                type="button"
                class="btn-ghost btn-icon shrink-0 text-gray-400 hover:text-gray-700 dark:text-dark-400 dark:hover:text-gray-200"
                :aria-label="t('common.close')"
                @click="handleDismiss"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>

          <!-- Body -->
          <div class="announcement-scroll max-h-[56vh] overflow-y-auto bg-pearl-surface px-6 py-6 sm:px-7 dark:bg-dark-800">
            <AnnouncementContent :content="displayedAnnouncement.content" />
          </div>

          <!-- Footer -->
          <div class="border-t border-gray-200/80 bg-pearl-bg/40 px-6 py-4 sm:px-7 dark:border-dark-700/70 dark:bg-dark-900/40">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="text-xs text-gray-500 dark:text-dark-400">
                <span
                  v-if="!preview && remainingPopupCount > 0"
                  class="inline-flex items-center gap-1.5 rounded-md bg-pearl-raised px-2 py-1 font-medium tabular-nums text-gray-700 dark:bg-dark-700 dark:text-gray-300"
                >
                  {{ t('announcements.remainingPopups', { count: remainingPopupCount }) }}
                </span>
                <span v-else-if="!preview">
                  {{ t('announcements.markReadHint') }}
                </span>
              </div>

              <button
                type="button"
                data-testid="announcement-popup-dismiss"
                class="btn btn-primary px-5 py-2.5"
                @click="handleDismiss"
              >
                <svg v-if="preview" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
                <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                <span>{{ preview ? t('common.close') : t('announcements.markRead') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AnnouncementContent from './AnnouncementContent.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeWithDateTime } from '@/utils/format'
import type { Announcement, UserAnnouncement } from '@/types'
import '@/styles/announcement-markdown.css'

type PreviewAnnouncement = Pick<Announcement | UserAnnouncement, 'title' | 'content' | 'created_at'>

const props = withDefaults(defineProps<{
  announcement?: PreviewAnnouncement | null
  preview?: boolean
}>(), {
  announcement: null,
  preview: false,
})

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const displayedAnnouncement = computed(() => (
  props.preview ? props.announcement : announcementStore.currentPopup
))
const remainingPopupCount = computed(() => announcementStore.remainingPopupCount ?? 0)

function handleDismiss() {
  if (props.preview) {
    emit('close')
    return
  }
  announcementStore.dismissPopup()
}

function handleEscape(event: KeyboardEvent) {
  if (props.preview && displayedAnnouncement.value && event.key === 'Escape') {
    handleDismiss()
  }
}

// Manage body overflow — only set, never unset (bell component handles restore)
watch(
  displayedAnnouncement,
  (popup) => {
    if (popup) {
      document.body.style.overflow = 'hidden'
    } else if (props.preview) {
      document.body.style.overflow = ''
    }
  },
  { immediate: true },
)

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
  if (props.preview) {
    document.body.style.overflow = ''
  }
})
</script>

<style scoped>
.popup-fade-enter-active {
  transition: opacity 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}

.popup-fade-leave-active {
  transition: opacity 0.16s cubic-bezier(0.4, 0, 1, 1);
}

.popup-fade-enter-from,
.popup-fade-leave-to {
  opacity: 0;
}

.popup-fade-enter-active > div {
  transition: transform 0.24s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.22s ease;
}

.popup-fade-leave-active > div {
  transition: transform 0.16s ease, opacity 0.16s ease;
}

.popup-fade-enter-from > div {
  transform: scale(0.97) translateY(8px);
  opacity: 0;
}

.popup-fade-leave-to > div {
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
