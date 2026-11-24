<template>
  <div class="markdown-body prose prose-sm max-w-none dark:prose-invert" @click="openImage" @keydown="openImage" v-html="html"></div>
  <BaseDialog :show="!!selectedImage" :title="selectedImage?.alt || t('announcements.viewImage')" width="full" :z-index="150" @close="selectedImage = null">
    <img v-if="selectedImage" :src="selectedImage.src" :alt="selectedImage.alt" class="mx-auto h-auto max-w-full" />
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import BaseDialog from './BaseDialog.vue'
import '@/styles/announcement-markdown.css'

const props = defineProps<{ content: string }>()
const { t } = useI18n()
const selectedImage = ref<{ src: string; alt: string } | null>(null)
watch(() => props.content, () => { selectedImage.value = null })
function closeTopImage(event: KeyboardEvent) {
  if (selectedImage.value && event.key === 'Escape') {
    event.preventDefault()
    event.stopImmediatePropagation()
    selectedImage.value = null
  }
}
onMounted(() => document.addEventListener('keydown', closeTopImage, true))
onBeforeUnmount(() => document.removeEventListener('keydown', closeTopImage, true))
const html = computed(() => {
  const clean = DOMPurify.sanitize(marked.parse(props.content || '', { breaks: true, gfm: true }) as string)
  const document = new DOMParser().parseFromString(clean, 'text/html')
  document.querySelectorAll('img').forEach(image => {
    image.tabIndex = 0
    image.setAttribute('role', 'button')
    image.setAttribute('aria-label', t('announcements.viewImage') + (image.alt ? `: ${image.alt}` : ''))
  })
  return document.body.innerHTML
})
function openImage(event: MouseEvent | KeyboardEvent) {
  if (event instanceof KeyboardEvent && event.key !== 'Enter' && event.key !== ' ') return
  if (!(event.target instanceof HTMLImageElement)) return
  event.preventDefault()
  event.stopPropagation()
  selectedImage.value = { src: event.target.src, alt: event.target.alt }
}
</script>
