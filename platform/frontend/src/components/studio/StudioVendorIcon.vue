<template>
  <svg
    v-if="meta"
    :class="['st-vendor-svg', `vendor-${meta.id}`]"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="currentColor"
    fill-rule="evenodd"
    aria-hidden="true"
  >
    <path :d="meta.path" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { VENDOR_META, type StudioVendorId } from '@/studio/models'

const props = withDefaults(
  defineProps<{
    vendor: StudioVendorId
    size?: number
  }>(),
  {
    size: 16,
  }
)

const meta = computed(() => VENDOR_META[props.vendor])
</script>
