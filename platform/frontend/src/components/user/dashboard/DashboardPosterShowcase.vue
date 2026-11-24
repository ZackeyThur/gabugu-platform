<template>
  <section
    v-if="posters.length > 0"
    class="overview-surface overview-poster-card"
    data-no-spotlight
    aria-roledescription="carousel"
    :aria-label="t('dashboard.overview.posterShowcase')"
    @mouseenter="paused = true"
    @mouseleave="paused = false"
    @focusin="paused = true"
    @focusout="onFocusOut"
    @keydown="onCarouselKeydown"
    @touchstart.passive="onTouchStart"
    @touchend.passive="onTouchEnd"
  >
    <!-- Live Ambient Color-Bleed Aura (Apple-TV / Gallery exhibition lighting mirroring active poster) -->
    <img
      v-if="!isCurrentImageBroken"
      :src="currentPoster.image.src"
      alt=""
      aria-hidden="true"
      class="overview-poster-ambient"
    />

    <div class="overview-poster-frame">
      <button
        type="button"
        class="overview-poster-stage"
        :aria-label="currentPoster.announcement.title"
        @click="handlePosterClick(currentPoster)"
      >
        <Transition :name="slideTransitionName" mode="out-in">
          <div :key="currentPoster.announcement.id" class="overview-poster-slide">
            <img
              v-if="!isCurrentImageBroken"
              :src="currentPoster.image.src"
              :alt="currentPoster.image.alt || currentPoster.announcement.title"
              class="overview-poster-img"
              loading="lazy"
              @error="markImageBroken(currentImageKey)"
            />
            <div
              v-else
              class="overview-poster-fallback"
              data-testid="poster-image-fallback"
              aria-hidden="true"
            >
              <div class="overview-poster-fallback-mark">
                <Icon name="photo" size="lg" :stroke-width="1.4" />
              </div>
            </div>
          </div>
        </Transition>
      </button>

      <template v-if="posters.length > 1">
        <button
          type="button"
          class="overview-poster-edge is-prev"
          :aria-label="t('dashboard.overview.posterPrev')"
          @click.stop="step(-1)"
        >
          <span class="overview-poster-edge-icon" aria-hidden="true">
            <Icon name="chevronLeft" size="md" :stroke-width="2.2" />
          </span>
        </button>
        <button
          type="button"
          class="overview-poster-edge is-next"
          :aria-label="t('dashboard.overview.posterNext')"
          @click.stop="step(1)"
        >
          <span class="overview-poster-edge-icon" aria-hidden="true">
            <Icon name="chevronRight" size="md" :stroke-width="2.2" />
          </span>
        </button>
        <div
          v-if="!reducedMotion"
          :key="`prog-${activeIndex}`"
          class="overview-poster-progress"
          :class="{ 'is-paused': paused }"
          aria-hidden="true"
        />
      </template>
    </div>

    <!-- Separated Editorial Caption Dock (Zero pill clutter, inline micro-dots, pure 16:9 artwork above) -->
    <div class="overview-poster-dock-bar">
      <button
        type="button"
        class="overview-poster-dock"
        :aria-label="currentPoster.announcement.title"
        @click="handlePosterClick(currentPoster)"
      >
        <div class="overview-poster-dock-left">
          <span class="overview-poster-kicker">
            {{ posters.length > 1 ? `${String(activeIndex + 1).padStart(2, '0')} / ${String(posters.length).padStart(2, '0')}` : t('dashboard.overview.featuredCampaign') }}
          </span>
          <span class="overview-poster-divider" aria-hidden="true" />
          <h3 class="overview-poster-title">{{ currentPoster.announcement.title }}</h3>
        </div>
        <span class="overview-poster-cta" aria-hidden="true">
          <span>{{ currentPoster.link ? t('dashboard.overview.posterExplore') : t('dashboard.overview.posterViewDetail') }}</span>
          <Icon name="chevronRight" size="sm" />
        </span>
      </button>
      <div
        v-if="posters.length > 1"
        class="overview-poster-dots"
        role="tablist"
        :aria-label="t('dashboard.overview.posterSlides')"
      >
        <button
          v-for="(item, idx) in posters"
          :key="item.announcement.id"
          type="button"
          role="tab"
          class="overview-poster-dot"
          :class="{ 'is-active': idx === activeIndex }"
          :aria-selected="idx === activeIndex"
          :aria-label="item.announcement.title"
          @click.stop="goToSlide(idx)"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { DashboardPosterItem } from '@/stores/announcements'
import type { UserAnnouncement } from '@/types'
import { prefersReducedMotion } from '@/utils/chartPlugins'

const props = defineProps<{
  posters: DashboardPosterItem[]
}>()

const emit = defineEmits<{
  'open-announcement': [announcement: UserAnnouncement]
}>()

const { t } = useI18n()
const router = useRouter()
const activeIndex = ref(0)
const slideDirection = ref<'next' | 'prev'>('next')
const paused = ref(false)
const reducedMotion = prefersReducedMotion()
const brokenImageKeys = ref<Record<string, boolean>>({})
let rotateTimer: ReturnType<typeof setInterval> | null = null

const slideTransitionName = computed(() =>
  slideDirection.value === 'prev' ? 'poster-slide-prev' : 'poster-slide-next'
)

const currentPoster = computed(() => props.posters[activeIndex.value] ?? props.posters[0])
const currentImageKey = computed(() => `${currentPoster.value.announcement.id}:${currentPoster.value.image.src}`)
const isCurrentImageBroken = computed(() => Boolean(brokenImageKeys.value[currentImageKey.value]))

function markImageBroken(key: string) {
  brokenImageKeys.value = { ...brokenImageKeys.value, [key]: true }
}

function onFocusOut(event: FocusEvent) {
  const container = event.currentTarget as HTMLElement | null
  const next = event.relatedTarget as Node | null
  if (!container || !next || !container.contains(next)) {
    paused.value = false
  }
}

watch(
  () => props.posters.length,
  (len) => {
    if (activeIndex.value >= len) {
      activeIndex.value = 0
    }
    restartAutoplay()
  }
)

function step(delta: number) {
  const len = props.posters.length
  if (len <= 1) return
  slideDirection.value = delta < 0 ? 'prev' : 'next'
  activeIndex.value = (activeIndex.value + delta + len) % len
}

function goToSlide(idx: number) {
  if (idx === activeIndex.value) return
  slideDirection.value = idx < activeIndex.value ? 'prev' : 'next'
  activeIndex.value = idx
}

let touchStartX = 0
let touchStartY = 0

function onTouchStart(event: TouchEvent) {
  const touch = event.touches?.[0]
  if (!touch) return
  touchStartX = touch.clientX
  touchStartY = touch.clientY
}

function onTouchEnd(event: TouchEvent) {
  if (props.posters.length <= 1) return
  const touch = event.changedTouches?.[0]
  if (!touch) return
  const dx = touch.clientX - touchStartX
  const dy = touch.clientY - touchStartY
  if (Math.abs(dx) >= 36 && Math.abs(dx) > Math.abs(dy) * 1.25) {
    step(dx < 0 ? 1 : -1)
  }
}

function onCarouselKeydown(event: KeyboardEvent) {
  if (props.posters.length <= 1) return
  if (event.key === 'ArrowRight') {
    event.preventDefault()
    step(1)
  } else if (event.key === 'ArrowLeft') {
    event.preventDefault()
    step(-1)
  }
}

function handlePosterClick(item: DashboardPosterItem) {
  if (item.link?.kind === 'internal') {
    void router?.push(item.link.href)
    return
  }
  if (item.link?.kind === 'external') {
    window.open(item.link.href, '_blank', 'noopener,noreferrer')
    return
  }
  emit('open-announcement', item.announcement)
}

function stopAutoplay() {
  if (rotateTimer !== null) {
    clearInterval(rotateTimer)
    rotateTimer = null
  }
}

function restartAutoplay() {
  stopAutoplay()
  if (props.posters.length <= 1 || prefersReducedMotion()) return
  rotateTimer = setInterval(() => {
    if (!paused.value) {
      step(1)
    }
  }, 6500)
}

onMounted(() => {
  restartAutoplay()
})

onBeforeUnmount(() => {
  stopAutoplay()
})
</script>
