<template>
  <svg
    :width="dimensions[size]"
    :height="dimensions[size]"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    focusable="false"
  >
    <component
      :is="element"
      v-for="([element, attributes], index) in icons[name]"
      :key="index"
      v-bind="attributes"
    />
  </svg>
</template>

<script setup lang="ts">
import ArrowDown02Icon from '@hugeicons/core-free-icons/ArrowDown02Icon'
import ArrowRight02Icon from '@hugeicons/core-free-icons/ArrowRight02Icon'
import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon'
import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon'
import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon'

// Render the library's original SVG data directly in Vue.
const icons = {
  arrowRight: ArrowRight02Icon,
  arrowDown: ArrowDown02Icon,
  copy: Copy01Icon,
  check: Tick02Icon,
  infoCircle: InformationCircleIcon
} as const
const dimensions = { sm: 18, md: 20 } as const

withDefaults(defineProps<{
  name: keyof typeof icons
  size?: keyof typeof dimensions
}>(), { size: 'sm' })
</script>
