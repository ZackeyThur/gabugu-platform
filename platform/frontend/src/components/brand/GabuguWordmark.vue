<template>
  <span class="gabugu-wordmark" :class="`gabugu-wordmark--${theme}`">
    <svg
      v-if="isGabugu"
      viewBox="0 0 932 154"
      width="932"
      height="154"
      role="img"
      :aria-label="name"
      focusable="false"
      fill-rule="evenodd"
    >
      <path :d="wordmarkPaths.G" />
      <path :d="wordmarkPaths.A" transform="translate(162 0)" />
      <path :d="wordmarkPaths.B" transform="translate(340 0)" />
      <path :d="wordmarkPaths.U" transform="translate(490 0)" />
      <path :d="wordmarkPaths.G" transform="translate(630 0)" />
      <path :d="wordmarkPaths.U" transform="translate(796 0)" />
    </svg>
    <span v-else class="gabugu-wordmark-custom">{{ name }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { wordmarkPaths } from './wordmarkPaths'

const props = withDefaults(defineProps<{
  name?: string
  theme?: 'auto' | 'light' | 'dark'
}>(), { name: 'GABUGU', theme: 'auto' })
const isGabugu = computed(() => /^gabugu(?: ai)?$/i.test(props.name.trim()))
</script>

<style scoped>
.gabugu-wordmark { display: inline-flex; align-items: center; min-width: 0; max-width: 100%; vertical-align: middle; color: #101723; }
.gabugu-wordmark svg { display: block; width: 6.052em; height: 1em; max-width: 100%; flex-shrink: 1; fill: currentColor; }
.gabugu-wordmark-custom { overflow-wrap: anywhere; }
html.dark .gabugu-wordmark--auto, .gabugu-wordmark--dark { color: #fff; }
.gabugu-wordmark--light { color: #101723; }
</style>
