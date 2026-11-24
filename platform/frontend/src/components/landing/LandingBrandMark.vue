<template>
  <img
    :src="imageSource"
    :alt="alt"
    :width="width"
    :height="height"
    class="landing-brand-mark"
    @error="useDefaultMark"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{
  source?: string
  alt?: string
  width?: string | number
  height?: string | number
}>(), { source: '', alt: '' })

const customFailed = ref(false)
watch(() => props.source, () => { customFailed.value = false })
const imageSource = computed(() => {
  const source = sanitizeUrl(props.source, { allowRelative: true, allowDataUrl: true })
  if (customFailed.value || !source || source === '/logo.svg') return '/brand/gabugu-logo-dark.png'
  return source
})

function useDefaultMark() {
  if (imageSource.value !== '/brand/gabugu-logo-dark.png') customFailed.value = true
}
</script>

<style scoped>
.landing-brand-mark { display: block; flex-shrink: 0; object-fit: contain; }
</style>
