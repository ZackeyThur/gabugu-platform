<template>
  <div class="space-y-4">
    <div>
      <label :for="bodyId" class="input-label">{{ t('admin.announcements.form.content') }}</label>
      <textarea :id="bodyId" v-model="body" rows="6" class="input" required></textarea>
    </div>
    <fieldset class="min-w-0 space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
      <legend class="px-1 text-sm font-medium">{{ t('admin.announcements.images.title') }}</legend>
      <p class="input-hint !mt-0">{{ t('admin.announcements.images.hint') }}</p>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="busy || images.length >= MAX_ANNOUNCEMENT_IMAGES" @click="chooseFiles()">
          <Icon name="upload" size="sm" class="mr-1.5" />
          {{ busy ? t('common.loading') : t('admin.announcements.images.add') }}
        </button>
        <input ref="fileInput" type="file" class="hidden" :accept="ANNOUNCEMENT_IMAGE_TYPES.join(',')" :multiple="replaceIndex === null" @change="readFiles" />
      </div>
      <div class="flex flex-wrap items-end gap-2">
        <div class="min-w-0 flex-1 basis-56">
          <label :for="urlId" class="sr-only">{{ t('admin.announcements.images.url') }}</label>
          <input :id="urlId" v-model="imageUrl" type="text" class="input" :placeholder="t('admin.announcements.images.url')" @keydown.enter.prevent="addUrl" />
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="busy || !imageUrl.trim() || images.length >= MAX_ANNOUNCEMENT_IMAGES" @click="addUrl">
          {{ t('admin.announcements.images.addUrl') }}
        </button>
      </div>
      <p v-if="error" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <ol v-if="images.length" class="grid gap-3 sm:grid-cols-2">
        <li v-for="(image, index) in images" :key="index" class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <img :src="image.src" :alt="image.alt" class="h-36 w-full rounded object-contain" />
          <label :for="`${bodyId}-alt-${index}`" class="input-label mt-2">{{ t('admin.announcements.images.alt') }}</label>
          <input :id="`${bodyId}-alt-${index}`" :value="image.alt" type="text" maxlength="200" class="input" @input="changeAlt(index, ($event.target as HTMLInputElement).value)" />
          <div class="mt-2 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="busy || index === 0" :aria-label="t('admin.announcements.images.moveUp', { index: index + 1 })" @click="moveImage(index, -1)">↑</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="busy || index === images.length - 1" :aria-label="t('admin.announcements.images.moveDown', { index: index + 1 })" @click="moveImage(index, 1)">↓</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="chooseFiles(index)">{{ t('admin.announcements.images.replace') }}</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="removeImage(index)">{{ t('common.remove') }}</button>
          </div>
        </li>
      </ol>
    </fieldset>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  ANNOUNCEMENT_IMAGE_TYPES, MAX_ANNOUNCEMENT_IMAGES, MAX_ANNOUNCEMENT_IMAGE_BYTES,
  MAX_ANNOUNCEMENT_IMAGE_TOTAL_BYTES, announcementImageBytes, isAnnouncementImageSource,
  joinAnnouncementImages, splitAnnouncementImages, type AnnouncementImage
} from '@/utils/announcementImages'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string]; busy: [value: boolean] }>()
const { t } = useI18n()
const bodyId = useId()
const urlId = useId()
const fileInput = ref<HTMLInputElement | null>(null)
const imageUrl = ref('')
const error = ref('')
const busy = ref(false)
const replaceIndex = ref<number | null>(null)
let disposed = false
onBeforeUnmount(() => { disposed = true })

const parsed = computed(() => splitAnnouncementImages(props.modelValue))
const images = computed(() => parsed.value.images)
const body = computed({
  get: () => parsed.value.body,
  set: value => emit('update:modelValue', joinAnnouncementImages(value, images.value))
})
function updateImages(value: AnnouncementImage[]) {
  emit('update:modelValue', joinAnnouncementImages(body.value, value))
}
async function chooseFiles(index: number | null = null) {
  replaceIndex.value = index
  await nextTick()
  fileInput.value?.click()
}
function removeImage(index: number) { updateImages(images.value.filter((_, i) => i !== index)) }
function changeAlt(index: number, alt: string) { updateImages(images.value.map((image, i) => i === index ? { ...image, alt } : image)) }
function moveImage(index: number, direction: number) {
  const next = [...images.value]
  const target = index + direction
  if (target < 0 || target >= next.length) return
  ;[next[index], next[target]] = [next[target]!, next[index]!]
  updateImages(next)
}
function addUrl() {
  const src = imageUrl.value.trim()
  if (busy.value || images.value.length >= MAX_ANNOUNCEMENT_IMAGES) return
  if (!isAnnouncementImageSource(src) || src.startsWith('data:')) {
    error.value = t('admin.announcements.images.invalidUrl')
    return
  }
  updateImages([...images.value, { src, alt: t('admin.announcements.images.defaultAlt') }])
  imageUrl.value = ''
  error.value = ''
}
function readImage(file: File): Promise<AnnouncementImage> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(new Error(t('common.fileReadFailed')))
    reader.onload = () => {
      const src = String(reader.result || '')
      if (!isAnnouncementImageSource(src)) { reject(new Error(t('admin.announcements.images.invalidType'))); return }
      const image = new Image()
      image.onerror = () => reject(new Error(t('admin.announcements.images.invalidImage')))
      image.onload = () => resolve({ src, alt: file.name.replace(/\.[^.]+$/, '').slice(0, 200) })
      image.src = src
    }
    reader.readAsDataURL(file)
  })
}
async function readFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length || busy.value) return
  error.value = ''
  const replacing = replaceIndex.value
  if ((replacing === null ? images.value.length + files.length : images.value.length) > MAX_ANNOUNCEMENT_IMAGES || (replacing !== null && files.length !== 1)) {
    error.value = t('admin.announcements.images.tooMany'); return
  }
  if (files.some(file => !ANNOUNCEMENT_IMAGE_TYPES.includes(file.type))) { error.value = t('admin.announcements.images.invalidType'); return }
  if (files.some(file => file.size > MAX_ANNOUNCEMENT_IMAGE_BYTES)) { error.value = t('admin.announcements.images.tooLarge'); return }
  const remaining = images.value.filter((_, index) => index !== replacing)
  if (announcementImageBytes(remaining) + files.reduce((sum, file) => sum + file.size, 0) > MAX_ANNOUNCEMENT_IMAGE_TOTAL_BYTES) {
    error.value = t('admin.announcements.images.totalTooLarge'); return
  }
  busy.value = true
  emit('busy', true)
  try {
    const added = await Promise.all(files.map(readImage))
    if (disposed) return
    const next = [...images.value]
    if (replacing === null) next.push(...added)
    else next.splice(replacing, 1, added[0]!)
    updateImages(next)
  } catch (cause) {
    if (!disposed) error.value = cause instanceof Error ? cause.message : t('common.fileReadFailed')
  } finally {
    busy.value = false
    emit('busy', false)
  }
}
</script>
