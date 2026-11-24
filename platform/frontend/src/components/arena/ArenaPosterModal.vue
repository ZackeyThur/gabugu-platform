<template>
  <Teleport to="body">
    <Transition name="poster-modal">
      <div
        v-if="open"
        class="poster-backdrop"
        :data-theme="theme"
        role="dialog"
        aria-modal="true"
        :aria-label="t('arena.poster.modalTitle')"
        @click.self="emit('close')"
      >
        <div class="poster-dialog">
          <header class="poster-head">
            <div>
              <h2>{{ t('arena.poster.modalTitle') }}</h2>
              <p>{{ t('arena.poster.modalSubtitle') }}</p>
            </div>
            <button type="button" class="poster-close" :aria-label="t('arena.poster.close')" @click="emit('close')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M6 6l12 12M18 6L6 18" /></svg>
            </button>
          </header>

          <div class="poster-body">
            <div class="poster-preview">
              <canvas ref="canvasRef" :class="{ 'is-rendering': rendering }" data-testid="arena-poster-canvas"></canvas>
              <div v-if="rendering" class="poster-rendering">{{ t('arena.poster.rendering') }}</div>
            </div>

            <div class="poster-side">
              <div class="poster-field">
                <span class="poster-label">{{ t('arena.poster.modeLabel') }}</span>
                <div class="poster-segment" :data-index="mode === 'top10' ? 0 : 1">
                  <span class="poster-segment-thumb" aria-hidden="true"></span>
                  <button type="button" data-testid="poster-mode-top10" :class="{ 'is-active': mode === 'top10' }" @click="mode = 'top10'">
                    {{ t('arena.poster.modeTop10') }}
                  </button>
                  <button type="button" data-testid="poster-mode-overview" :class="{ 'is-active': mode === 'overview' }" @click="mode = 'overview'">
                    {{ t('arena.poster.modeOverview') }}
                  </button>
                </div>
              </div>

              <div class="poster-field">
                <span class="poster-label">{{ t('arena.poster.themeLabel') }}</span>
                <div class="poster-segment" :data-index="posterTheme === 'dark' ? 0 : 1">
                  <span class="poster-segment-thumb" aria-hidden="true"></span>
                  <button type="button" data-testid="poster-theme-dark" :class="{ 'is-active': posterTheme === 'dark' }" @click="posterTheme = 'dark'">
                    {{ t('arena.poster.themeDark') }}
                  </button>
                  <button type="button" data-testid="poster-theme-light" :class="{ 'is-active': posterTheme === 'light' }" @click="posterTheme = 'light'">
                    {{ t('arena.poster.themeLight') }}
                  </button>
                </div>
              </div>

              <p class="poster-spec">{{ t('arena.poster.specHint') }}</p>

              <Transition name="poster-toast">
                <p v-if="feedback" class="poster-feedback" :class="`is-${feedback.type}`">{{ feedback.message }}</p>
              </Transition>

              <div class="poster-actions">
                <button type="button" class="poster-btn poster-btn--primary" data-testid="poster-download-btn" :disabled="rendering" @click="handleDownload">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" aria-hidden="true">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v11m0 0l-4-4m4 4l4-4M4 17v2a2 2 0 002 2h12a2 2 0 002-2v-2" />
                  </svg>
                  <span>{{ t('arena.poster.downloadPng') }}</span>
                </button>
                <div class="poster-actions-row">
                  <button type="button" class="poster-btn" data-testid="poster-copy-image-btn" :disabled="rendering" @click="handleCopyImage">
                    {{ t('arena.poster.copyImage') }}
                  </button>
                  <button type="button" class="poster-btn" data-testid="poster-copy-link-btn" @click="handleCopyLink">
                    {{ t('arena.poster.copyLink') }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ArenaCategoryBoard } from '@/api/arena'
import {
  buildPosterFilename,
  renderArenaPosterToCanvas,
  type ArenaPosterMode,
  type ArenaPosterTheme
} from '@/utils/arenaPosterRenderer'
import { useAppStore } from '@/stores/app'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(
  defineProps<{
    open: boolean
    activeBoard: ArenaCategoryBoard
    allBoards: ArenaCategoryBoard[]
    lastCheckedText: string
    theme?: 'dark' | 'light'
  }>(),
  { theme: 'dark' }
)

const emit = defineEmits<{ (e: 'close'): void }>()

const { t, locale } = useI18n()
const appStore = useAppStore()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const mode = ref<ArenaPosterMode>('top10')
const posterTheme = ref<ArenaPosterTheme>(props.theme)
watch(
  () => props.open,
  (open) => {
    if (open) posterTheme.value = props.theme
  }
)
const rendering = ref(false)
const feedback = ref<{ type: 'success' | 'error'; message: string } | null>(null)
let feedbackTimer: number | undefined

function showFeedback(type: 'success' | 'error', message: string) {
  feedback.value = { type, message }
  if (feedbackTimer) window.clearTimeout(feedbackTimer)
  feedbackTimer = window.setTimeout(() => (feedback.value = null), 3600)
}

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'GABUGU')
const customSiteLogo = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.site_logo || '', { allowRelative: true, allowDataUrl: true })
)

const shareUrl = computed(() => {
  const origin = typeof window !== 'undefined' ? window.location.origin : 'https://gabugu.com'
  return mode.value === 'overview' ? `${origin}/arena` : `${origin}/arena?tab=${props.activeBoard.id}`
})

const displayUrl = computed(() => {
  try {
    return `${new URL(shareUrl.value).host}/arena`
  } catch {
    return 'gabugu.com/arena'
  }
})

async function redraw() {
  if (!props.open) return
  await nextTick()
  const canvas = canvasRef.value
  if (!canvas) return
  rendering.value = true
  try {
    await renderArenaPosterToCanvas(canvas, {
      mode: mode.value,
      theme: posterTheme.value,
      siteName: siteName.value,
      siteLogoUrl: customSiteLogo.value || (posterTheme.value === 'dark' ? '/brand/gabugu-logo-dark.png' : '/brand/gabugu-logo-light.png'),
      shareUrl: shareUrl.value,
      displayUrl: displayUrl.value,
      lastCheckedText: props.lastCheckedText,
      activeBoard: props.activeBoard,
      allBoards: props.allBoards,
      labels: {
        categoryLabels: {
          text: t('arena.categories.text.label'),
          code: t('arena.categories.code.label'),
          vision: t('arena.categories.vision.label'),
          image: t('arena.categories.image.label')
        },
        top10Suffix: t('arena.poster.top10Suffix'),
        overviewTitle: t('arena.poster.overviewTitle'),
        overviewSubtitle: t('arena.poster.overviewSubtitle'),
        publishDateLabel: t('arena.poster.publishDateLabel'),
        baselineDateLabel: t('arena.poster.baselineDateLabel'),
        modelCountText: (count: number) => t('arena.poster.modelCount', { count }),
        openBadge: t('arena.table.openBadge'),
        newBadge: t('arena.table.newBadge'),
        footerScanHint: t('arena.poster.footerScanHint'),
        footerSource: t('arena.poster.footerSource'),
        lastCheckedLabel: t('arena.poster.lastChecked')
      }
    })
  } finally {
    rendering.value = false
  }
}

function canvasToPngBlob(): Promise<Blob | null> {
  return new Promise((resolve) => {
    const canvas = canvasRef.value
    if (!canvas) return resolve(null)
    canvas.toBlob((blob) => resolve(blob), 'image/png')
  })
}

async function handleDownload() {
  const blob = await canvasToPngBlob()
  if (!blob) return
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = buildPosterFilename(mode.value, props.activeBoard.id, props.activeBoard.publish_date)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  showFeedback('success', t('arena.poster.downloaded'))
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

async function handleCopyImage() {
  const blob = await canvasToPngBlob()
  if (!blob) return
  if (typeof navigator === 'undefined' || !navigator.clipboard?.write || typeof ClipboardItem === 'undefined') {
    showFeedback('error', t('arena.poster.copyImageUnsupported'))
    return
  }
  try {
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])
    showFeedback('success', t('arena.poster.copiedImage'))
  } catch {
    showFeedback('error', t('arena.poster.copyImageUnsupported'))
  }
}

async function handleCopyLink() {
  try {
    await navigator.clipboard.writeText(shareUrl.value)
    showFeedback('success', t('arena.poster.copiedLink'))
  } catch {
    showFeedback('error', shareUrl.value)
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (props.open && e.key === 'Escape') emit('close')
}

watch(
  () => [props.open, mode.value, posterTheme.value, props.activeBoard.id, locale.value],
  () => {
    if (props.open) void redraw()
  }
)

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  if (props.open) void redraw()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  if (feedbackTimer) window.clearTimeout(feedbackTimer)
})
</script>

<style scoped>
.poster-backdrop {
  --p-ease: cubic-bezier(0.16, 1, 0.3, 1);
  --p-bg: #10121a;
  --p-text: #f3f5f9;
  --p-muted: #8b92a1;
  --p-faint: #5d6473;
  --p-line: rgba(255, 255, 255, 0.09);
  --p-field: rgba(255, 255, 255, 0.05);
  --p-active: rgba(255, 255, 255, 0.12);
  --p-stage: #07080d;
  --p-btn-bg: #f3f5f9;
  --p-btn-text: #08090f;
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(3, 4, 8, 0.74);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  font-family: Inter, 'Gabugu Arena SC', 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  color: var(--p-text);
}
.poster-backdrop[data-theme='light'] {
  --p-bg: #ffffff;
  --p-text: #0b1220;
  --p-muted: #5f6c80;
  --p-faint: #8e9aaf;
  --p-line: rgba(15, 23, 42, 0.09);
  --p-field: rgba(15, 23, 42, 0.045);
  --p-active: #ffffff;
  --p-stage: #f1f5f9;
  --p-btn-bg: #0f172a;
  --p-btn-text: #ffffff;
  background: rgba(15, 23, 42, 0.42);
}

.poster-dialog {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 896px;
  max-height: calc(100vh - 48px);
  overflow: hidden;
  border: 1px solid var(--p-line);
  border-radius: 20px;
  background: var(--p-bg);
  box-shadow: 0 32px 84px rgba(0, 0, 0, 0.48);
}

.poster-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 26px 18px;
  border-bottom: 1px solid var(--p-line);
}
.poster-head h2 {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
}
.poster-head p {
  margin: 4px 0 0;
  color: var(--p-muted);
  font-size: 13px;
  text-wrap: balance;
}
.poster-close {
  display: inline-flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: none;
  color: var(--p-muted);
  cursor: pointer;
  transition: background-color 0.16s ease, color 0.16s ease;
}
.poster-close:hover {
  background: var(--p-field);
  color: var(--p-text);
}
.poster-close svg {
  width: 18px;
  height: 18px;
}

.poster-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 288px;
  gap: 24px;
  padding: 24px 26px;
  overflow-y: auto;
}
.poster-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 22px;
  border: 1px solid var(--p-line);
  border-radius: 14px;
  background: var(--p-stage);
}
.poster-preview canvas {
  display: block;
  width: auto;
  max-width: 100%;
  height: auto;
  max-height: 62vh;
  border-radius: 10px;
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.36);
  transition: opacity 0.2s ease, transform 0.24s var(--p-ease);
}
.poster-preview canvas.is-rendering {
  opacity: 0.55;
  transform: scale(0.99);
}
.poster-rendering {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: rgba(0, 0, 0, 0.28);
  color: #fff;
  font-size: 13px;
}

.poster-side {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.poster-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.poster-label {
  color: var(--p-muted);
  font-size: 12px;
  font-weight: 500;
}

/* Sliding 2-way segmented control */
.poster-segment {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 3px;
  border: 1px solid var(--p-line);
  border-radius: 11px;
  background: var(--p-field);
}
.poster-segment-thumb {
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 3px;
  width: calc(50% - 3px);
  border-radius: 8px;
  background: var(--p-active);
  transition: transform 0.28s var(--p-ease);
  pointer-events: none;
}
.poster-segment[data-index='1'] .poster-segment-thumb {
  transform: translateX(100%);
}
.poster-backdrop[data-theme='light'] .poster-segment-thumb {
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}
.poster-segment button {
  position: relative;
  z-index: 1;
  height: 36px;
  border: 0;
  border-radius: 8px;
  background: none;
  color: var(--p-muted);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.16s ease;
}
.poster-segment button:hover {
  color: var(--p-text);
}
.poster-segment button.is-active {
  color: var(--p-text);
}

.poster-spec {
  margin: 0;
  color: var(--p-faint);
  font-size: 12px;
  line-height: 1.6;
}
.poster-feedback {
  margin: 0;
  padding: 8px 12px;
  border-radius: 9px;
  font-size: 12.5px;
  line-height: 1.5;
}
.poster-feedback.is-success {
  background: rgba(61, 214, 140, 0.11);
  color: #3dd68c;
}
.poster-backdrop[data-theme='light'] .poster-feedback.is-success {
  color: #059669;
}
.poster-feedback.is-error {
  background: rgba(245, 177, 74, 0.12);
  color: #f5b14a;
}

.poster-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: auto;
}
.poster-actions-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.poster-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  height: 40px;
  border: 1px solid var(--p-line);
  border-radius: 999px;
  background: none;
  color: var(--p-text);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.16s ease, opacity 0.16s ease, transform 0.12s ease;
}
.poster-btn svg {
  width: 16px;
  height: 16px;
}
.poster-btn:hover:not(:disabled) {
  background: var(--p-field);
}
.poster-btn:active:not(:disabled) {
  transform: scale(0.975);
}
.poster-btn:disabled {
  opacity: 0.5;
  cursor: default;
}
.poster-btn--primary {
  height: 44px;
  border-color: transparent;
  background: var(--p-btn-bg);
  color: var(--p-btn-text);
  font-size: 14px;
}
.poster-btn--primary:hover:not(:disabled) {
  background: var(--p-btn-bg);
  opacity: 0.9;
}

/* Modal & Toast transitions */
.poster-modal-enter-active,
.poster-modal-leave-active {
  transition: opacity 0.24s ease;
}
.poster-modal-enter-active .poster-dialog,
.poster-modal-leave-active .poster-dialog {
  transition: transform 0.28s var(--p-ease), opacity 0.24s ease;
}
.poster-modal-enter-from,
.poster-modal-leave-to {
  opacity: 0;
}
.poster-modal-enter-from .poster-dialog,
.poster-modal-leave-to .poster-dialog {
  opacity: 0;
  transform: translateY(14px) scale(0.965);
}

.poster-toast-enter-active,
.poster-toast-leave-active {
  transition: opacity 0.18s ease, transform 0.2s var(--p-ease);
}
.poster-toast-enter-from,
.poster-toast-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

@media (max-width: 720px) {
  .poster-backdrop {
    align-items: flex-end;
    padding: 0;
  }
  .poster-dialog {
    max-height: 92vh;
    border-radius: 20px 20px 0 0;
  }
  .poster-modal-enter-from .poster-dialog,
  .poster-modal-leave-to .poster-dialog {
    transform: translateY(32px);
  }
  .poster-body {
    grid-template-columns: 1fr;
    gap: 16px;
    padding: 16px;
  }
  .poster-preview {
    padding: 14px;
  }
  .poster-preview canvas {
    max-height: 36vh;
  }
}

@media (prefers-reduced-motion: reduce) {
  .poster-modal-enter-active,
  .poster-modal-leave-active,
  .poster-modal-enter-active .poster-dialog,
  .poster-modal-leave-active .poster-dialog,
  .poster-segment-thumb,
  .poster-preview canvas {
    transition: none;
  }
}
</style>
