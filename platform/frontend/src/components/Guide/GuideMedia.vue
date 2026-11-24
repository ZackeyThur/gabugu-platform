<template>
  <figure class="guide-media" :class="{ 'guide-media-compact': compact }">
    <button v-if="!failed" type="button" class="guide-media-open" :aria-label="`${t('guide.media.expand')}: ${title}`" @click="expanded = true">
      <img :src="src" :alt="alt" :width="width" :height="height" :style="{ objectFit: fit }" loading="lazy" decoding="async" @error="failed = true" />
      <span class="guide-media-zoom"><Icon name="search" size="sm" />{{ t('guide.media.expand') }}</span>
    </button>
    <div v-else class="guide-media-fallback" role="status"><Icon name="book" size="lg" /><span>{{ t('guide.media.unavailable') }}</span></div>
    <figcaption>
      <span>{{ title }}</span>
      <a v-if="source" :href="source" target="_blank" rel="noopener noreferrer">{{ t('guide.media.officialLabel') }} · {{ version }}</a>
      <span v-else class="guide-media-credit">{{ t('guide.media.demoLabel') }}</span>
    </figcaption>
    <BaseDialog :show="expanded" :title="title" width="full" :close-on-click-outside="true" @close="expanded = false">
      <img :src="fullSrc || src" :alt="alt" :width="fullSrc ? undefined : width" :height="fullSrc ? undefined : height" class="guide-media-expanded" />
      <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">{{ source ? t('guide.media.officialSource') : t('guide.media.demonstrationOnly') }}</p>
    </BaseDialog>
  </figure>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  src: string
  fullSrc?: string
  fit?: 'cover' | 'contain'
  compact?: boolean
  title: string
  alt: string
  width?: number
  height?: number
  source?: string
  version?: string
}>(), { width: 1400, height: 900, source: '', version: '', fullSrc: '', fit: 'cover', compact: false })
const { t } = useI18n()
const expanded = ref(false)
const failed = ref(false)
watch(() => props.src, () => { failed.value = false })
</script>

<style scoped>
.guide-media { min-width: 0; width: 100%; margin: 0; }

/* 预览框按原始比例分档，长图取顶部重点区域；点击后看完整原图 */
.guide-media-open {
  position: relative;
  display: grid;
  grid-template-rows: minmax(0,1fr) auto;
  gap: 9px;
  overflow: hidden;
  width: 100%;
  max-width: var(--guide-measure);
  height: 258px;
  padding: 10px;
  border: 1px solid var(--guide-border);
  border-radius: 12px;
  background: var(--guide-surface);
  cursor: zoom-in;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.guide-media-open img {
  display: block;
  width: 100%;
  height: 100%;
  min-height: 0;
  max-height: 100%;
  object-fit: cover;
  object-position: center top;
  border: 1px solid var(--guide-border);
  border-radius: 6px;
}
.guide-media-compact .guide-media-open { height: 144px; }
.guide-media-compact .guide-media-open img { object-position: center; }
.guide-media-open:hover {
  border-color: var(--guide-accent);
  box-shadow: 0 2px 12px rgb(16 24 40 / 0.07);
}
.guide-media-open:focus-visible { outline: 2px solid var(--guide-accent); outline-offset: 3px; }

.guide-media-zoom {
  position: static;
  justify-self: end;
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 3px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--guide-accent-text);
  font-size: var(--guide-fs-micro);
  line-height: 18px;
  font-weight: 500;
}
.dark .guide-media-zoom { color: var(--guide-accent); }
.guide-media-zoom svg { width: 13px; height: 13px; }

figcaption {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 3px 12px;
  max-width: var(--guide-measure);
  padding-top: 9px;
  color: var(--guide-muted);
  font-size: var(--guide-fs-caption);
  line-height: 21px;
}
figcaption a { color: var(--guide-accent-text); text-decoration: underline; text-underline-offset: 3px; }
.guide-media-credit { color: var(--guide-faint); }

.guide-media-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  max-width: var(--guide-measure);
  min-height: 140px;
  padding: 24px;
  border: 1px dashed var(--guide-border);
  border-radius: 12px;
  color: var(--guide-muted);
  font-size: var(--guide-fs-small);
}
.guide-media-expanded {
  display: block;
  width: auto;
  height: auto;
  max-width: 100%;
  margin-inline: auto;
  border-radius: 8px;
}
@media (max-width: 767px) {
  .guide-media-open { padding: 10px; height: 236px; }
  .guide-media-open img { max-height: 100%; }
}
</style>
